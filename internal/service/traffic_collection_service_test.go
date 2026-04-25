package service

import (
	"context"
	"testing"
	"time"

	"traffic-monitor/internal/config"
	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/model"
	"traffic-monitor/internal/repo"

	"github.com/stretchr/testify/require"
)

type stubTrafficSampleStore struct {
	samples []model.TrafficSample
}

func (store *stubTrafficSampleStore) UpsertSamples(_ context.Context, samples []model.TrafficSample) error {
	store.samples = append(store.samples, samples...)
	return nil
}

func (store *stubTrafficSampleStore) List(_ context.Context, _ repo.TrafficSampleFilter) ([]model.TrafficSample, int64, error) {
	return store.samples, int64(len(store.samples)), nil
}

type stubTrafficMachineStore struct {
	machines []model.Machine
	byID     map[uint]*model.Machine
}

func (store *stubTrafficMachineStore) List(_ context.Context) ([]model.Machine, error) {
	return store.machines, nil
}

func (store *stubTrafficMachineStore) GetByID(_ context.Context, machineID uint) (*model.Machine, error) {
	machine, ok := store.byID[machineID]
	if !ok {
		return nil, ErrMachineNotFound
	}

	return machine, nil
}

type stubTrafficSSHKeyStore struct {
	items map[uint]*model.SSHKey
}

func (store *stubTrafficSSHKeyStore) GetByID(_ context.Context, sshKeyID uint) (*model.SSHKey, error) {
	item, ok := store.items[sshKeyID]
	if !ok {
		return nil, ErrSSHKeyNotFound
	}

	return item, nil
}

func TestParseVNStatSamples(t *testing.T) {
	rawPayload := `{
  "vnstatversion": "2.10",
  "jsonversion": "2",
  "interfaces": [
    {
      "name": "eth0",
      "traffic": {
        "hour": [
          {
            "date": {"year": 2026, "month": 4, "day": 25},
            "time": {"hour": 10, "minute": 0},
            "timestamp": 1777082400,
            "rx": 104857600,
            "tx": 52428800
          }
        ],
        "day": [
          {
            "date": {"year": 2026, "month": 4, "day": 25},
            "timestamp": 1777046400,
            "rx": 1073741824,
            "tx": 536870912
          }
        ]
      }
    }
  ]
}`

	samples, err := parseVNStatSamples(1, "eth0", rawPayload)
	require.NoError(t, err)
	require.Len(t, samples, 2)
	require.Equal(t, trafficPeriodHourly, samples[0].PeriodType)
	require.Equal(t, 50.0, samples[0].UploadMB)
	require.Equal(t, 100.0, samples[0].DownloadMB)
	require.Equal(t, 150.0, samples[0].TotalMB)
	require.Equal(t, trafficPeriodDaily, samples[1].PeriodType)
	require.Equal(t, 512.0, samples[1].UploadMB)
	require.Equal(t, 1024.0, samples[1].DownloadMB)
}

func TestCollectNowForSpecificMachine(t *testing.T) {
	machine := &model.Machine{
		Base:             model.Base{ID: 1},
		Name:             "host-a",
		Host:             "10.0.0.1",
		Port:             22,
		SSHUser:          "root",
		NetworkInterface: "eth0",
		SSHKeyID:         1,
		CollectEnabled:   true,
	}

	service := &TrafficCollectionService{
		machineStore: &stubTrafficMachineStore{
			machines: []model.Machine{*machine},
			byID:     map[uint]*model.Machine{1: machine},
		},
		sshKeyStore: &stubTrafficSSHKeyStore{
			items: map[uint]*model.SSHKey{
				1: {Base: model.Base{ID: 1}, PrivateKeyCiphertext: "cipher"},
			},
		},
		dataProtector: stubDecryptor{plaintext: []byte("private-key")},
		sshRunner: stubSSHRunner{
			result: SSHExecutionResult{
				Stdout: `{"vnstatversion":"2.10","jsonversion":"2","interfaces":[{"name":"eth0","traffic":{"hour":[{"date":{"year":2026,"month":4,"day":25},"time":{"hour":10,"minute":0},"timestamp":1777082400,"rx":1048576,"tx":2097152}],"day":[{"date":{"year":2026,"month":4,"day":25},"timestamp":1777046400,"rx":3145728,"tx":4194304}]}}]}`,
			},
		},
		trafficSampleStore: &stubTrafficSampleStore{},
		sshConfig:          config.SSHConfig{DialTimeout: 5 * time.Second, CommandTimeout: 5 * time.Second},
	}

	resp, err := service.CollectNow(context.Background(), ptrUint(1))
	require.NoError(t, err)
	require.Len(t, resp.Results, 1)
	require.Equal(t, "success", resp.Results[0].Status)
	require.Equal(t, 2, resp.Results[0].SampleCount)
}

func TestListSamples(t *testing.T) {
	service := &TrafficCollectionService{
		trafficSampleStore: &stubTrafficSampleStore{
			samples: []model.TrafficSample{
				{
					Base:        model.Base{ID: 1},
					MachineID:   2,
					PeriodType:  trafficPeriodHourly,
					BucketTime:  time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC),
					UploadMB:    10,
					DownloadMB:  20,
					TotalMB:     30,
					CollectedAt: time.Date(2026, 4, 25, 10, 5, 0, 0, time.UTC),
				},
			},
		},
	}

	resp, err := service.ListSamples(context.Background(), dto.ListTrafficSamplesQuery{})
	require.NoError(t, err)
	require.Equal(t, int64(1), resp.Total)
	require.Len(t, resp.Items, 1)
	require.Equal(t, uint(2), resp.Items[0].MachineID)
}

func ptrUint(value uint) *uint {
	return &value
}
