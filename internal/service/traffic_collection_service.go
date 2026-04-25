package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"traffic-monitor/internal/config"
	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/model"
	"traffic-monitor/internal/repo"
)

const (
	trafficPeriodHourly = "hourly"
	trafficPeriodDaily  = "daily"
)

var (
	ErrVNStatParseFailed      = errors.New("vnstat parse failed")
	ErrVNStatInterfaceMissing = errors.New("vnstat interface missing")
)

type TrafficMachineStore interface {
	List(ctx context.Context) ([]model.Machine, error)
	GetByID(ctx context.Context, machineID uint) (*model.Machine, error)
}

type TrafficSSHKeyStore interface {
	GetByID(ctx context.Context, sshKeyID uint) (*model.SSHKey, error)
}

type TrafficSampleStore interface {
	UpsertSamples(ctx context.Context, samples []model.TrafficSample) error
	List(ctx context.Context, filter repo.TrafficSampleFilter) ([]model.TrafficSample, int64, error)
}

type TrafficCollectionService struct {
	machineStore       TrafficMachineStore
	sshKeyStore        TrafficSSHKeyStore
	dataProtector      SSHKeyProtector
	sshRunner          SSHCommandRunner
	trafficSampleStore TrafficSampleStore
	sshConfig          config.SSHConfig
}

func NewTrafficCollectionService(machineStore *repo.MachineRepo, sshKeyStore *repo.SSHKeyRepo, dataProtector SSHKeyProtector, sshRunner SSHCommandRunner, trafficSampleStore *repo.TrafficSampleRepo, sshConfig config.SSHConfig) *TrafficCollectionService {
	return &TrafficCollectionService{
		machineStore:       machineStore,
		sshKeyStore:        sshKeyStore,
		dataProtector:      dataProtector,
		sshRunner:          sshRunner,
		trafficSampleStore: trafficSampleStore,
		sshConfig:          sshConfig,
	}
}

func (service *TrafficCollectionService) CollectNow(ctx context.Context, machineID *uint) (dto.CollectNowResp, error) {
	machines, err := service.resolveTargetMachines(ctx, machineID)
	if err != nil {
		return dto.CollectNowResp{}, err
	}

	results := make([]dto.CollectNowMachineResp, 0, len(machines))
	for _, machine := range machines {
		result := dto.CollectNowMachineResp{
			MachineID: machine.ID,
			Status:    "success",
		}

		samples, collectErr := service.collectMachineSamples(ctx, &machine)
		if collectErr != nil {
			result.Status = "failed"
			result.Error = collectErr.Error()
			results = append(results, result)
			continue
		}

		result.SampleCount = len(samples)
		results = append(results, result)
	}

	return dto.CollectNowResp{Results: results}, nil
}

func (service *TrafficCollectionService) ListEnabledMachines(ctx context.Context) ([]model.Machine, error) {
	machines, err := service.machineStore.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list machines for scheduler: %w", err)
	}

	result := make([]model.Machine, 0, len(machines))
	for _, machine := range machines {
		if machine.CollectEnabled {
			result = append(result, machine)
		}
	}

	return result, nil
}

func (service *TrafficCollectionService) CollectMachine(ctx context.Context, machine *model.Machine) ([]model.TrafficSample, error) {
	return service.collectMachineSamples(ctx, machine)
}

func (service *TrafficCollectionService) ListSamples(ctx context.Context, query dto.ListTrafficSamplesQuery) (dto.TrafficSampleListResp, error) {
	samples, total, err := service.trafficSampleStore.List(ctx, repo.TrafficSampleFilter{
		MachineID:  query.MachineID,
		PeriodType: query.PeriodType,
		Page:       query.Page,
		PageSize:   query.PageSize,
	})
	if err != nil {
		return dto.TrafficSampleListResp{}, fmt.Errorf("list traffic samples: %w", err)
	}

	items := make([]dto.TrafficSampleResp, 0, len(samples))
	for _, sample := range samples {
		items = append(items, dto.TrafficSampleResp{
			ID:          sample.ID,
			MachineID:   sample.MachineID,
			PeriodType:  sample.PeriodType,
			BucketTime:  sample.BucketTime,
			UploadMB:    sample.UploadMB,
			DownloadMB:  sample.DownloadMB,
			TotalMB:     sample.TotalMB,
			CollectedAt: sample.CollectedAt,
		})
	}

	return dto.TrafficSampleListResp{
		Items: items,
		Total: total,
	}, nil
}

func (service *TrafficCollectionService) resolveTargetMachines(ctx context.Context, machineID *uint) ([]model.Machine, error) {
	if machineID != nil {
		machine, err := service.machineStore.GetByID(ctx, *machineID)
		if err != nil {
			if repo.IsRecordNotFound(err) {
				return nil, ErrMachineNotFound
			}

			return nil, fmt.Errorf("get machine for collect now: %w", err)
		}

		return []model.Machine{*machine}, nil
	}

	machines, err := service.machineStore.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list machines for collect now: %w", err)
	}

	result := make([]model.Machine, 0, len(machines))
	for _, machine := range machines {
		if machine.CollectEnabled {
			result = append(result, machine)
		}
	}

	return result, nil
}

func (service *TrafficCollectionService) collectMachineSamples(ctx context.Context, machine *model.Machine) ([]model.TrafficSample, error) {
	sshKey, err := service.sshKeyStore.GetByID(ctx, machine.SSHKeyID)
	if err != nil {
		if repo.IsRecordNotFound(err) {
			return nil, ErrSSHKeyNotFound
		}

		return nil, fmt.Errorf("get ssh key for collection: %w", err)
	}

	privateKeyPEM, err := service.dataProtector.Decrypt(sshKey.PrivateKeyCiphertext)
	if err != nil {
		return nil, fmt.Errorf("decrypt ssh private key: %w", err)
	}

	commandContext, cancel := context.WithTimeout(ctx, service.sshConfig.CommandTimeout)
	defer cancel()

	command := fmt.Sprintf("vnstat --json -i %s", shellEscapeArg(machine.NetworkInterface))
	result, err := service.sshRunner.Run(commandContext, machine.Host, machine.Port, machine.SSHUser, privateKeyPEM, command)
	if err != nil {
		return nil, fmt.Errorf("run vnstat json command: %w", err)
	}

	samples, err := parseVNStatSamples(machine.ID, machine.NetworkInterface, result.Stdout)
	if err != nil {
		return nil, err
	}

	if err := service.trafficSampleStore.UpsertSamples(ctx, samples); err != nil {
		return nil, fmt.Errorf("persist traffic samples: %w", err)
	}

	return samples, nil
}

type vnstatResponse struct {
	Interfaces []vnstatInterface `json:"interfaces"`
}

type vnstatInterface struct {
	Name    string        `json:"name"`
	Traffic vnstatTraffic `json:"traffic"`
}

type vnstatTraffic struct {
	Hours []vnstatEntry `json:"hours"`
	Hour  []vnstatEntry `json:"hour"`
	Days  []vnstatEntry `json:"days"`
	Day   []vnstatEntry `json:"day"`
}

type vnstatEntry struct {
	Date      vnstatDate `json:"date"`
	Time      vnstatTime `json:"time"`
	Timestamp int64      `json:"timestamp"`
	RX        float64    `json:"rx"`
	TX        float64    `json:"tx"`
}

type vnstatDate struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Day   int `json:"day"`
}

type vnstatTime struct {
	Hour   int `json:"hour"`
	Minute int `json:"minute"`
}

func parseVNStatSamples(machineID uint, networkInterface string, rawPayload string) ([]model.TrafficSample, error) {
	var payload vnstatResponse
	if err := json.Unmarshal([]byte(rawPayload), &payload); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrVNStatParseFailed, err)
	}

	targetInterface, ok := findVNStatInterface(payload.Interfaces, networkInterface)
	if !ok {
		return nil, ErrVNStatInterfaceMissing
	}

	now := time.Now().UTC()
	samples := make([]model.TrafficSample, 0, 2)

	hourEntries := targetInterface.Traffic.Hours
	if len(hourEntries) == 0 {
		hourEntries = targetInterface.Traffic.Hour
	}

	if hourlyEntry, ok := latestVNStatEntry(hourEntries); ok {
		samples = append(samples, trafficSampleFromEntry(machineID, trafficPeriodHourly, hourlyEntry, rawPayload, now))
	}

	dayEntries := targetInterface.Traffic.Days
	if len(dayEntries) == 0 {
		dayEntries = targetInterface.Traffic.Day
	}

	if dailyEntry, ok := latestVNStatEntry(dayEntries); ok {
		samples = append(samples, trafficSampleFromEntry(machineID, trafficPeriodDaily, dailyEntry, rawPayload, now))
	}

	if len(samples) == 0 {
		return nil, ErrVNStatParseFailed
	}

	return samples, nil
}

func findVNStatInterface(interfaces []vnstatInterface, networkInterface string) (vnstatInterface, bool) {
	for _, iface := range interfaces {
		if iface.Name == networkInterface {
			return iface, true
		}
	}

	if len(interfaces) == 1 {
		return interfaces[0], true
	}

	return vnstatInterface{}, false
}

func latestVNStatEntry(entries []vnstatEntry) (vnstatEntry, bool) {
	if len(entries) == 0 {
		return vnstatEntry{}, false
	}

	latest := entries[0]
	latestTime := vnstatEntryTime(latest)
	for _, entry := range entries[1:] {
		entryTime := vnstatEntryTime(entry)
		if entryTime.After(latestTime) {
			latest = entry
			latestTime = entryTime
		}
	}

	return latest, true
}

func vnstatEntryTime(entry vnstatEntry) time.Time {
	if entry.Timestamp > 0 {
		return time.Unix(entry.Timestamp, 0).UTC()
	}

	month := entry.Date.Month
	if month <= 0 {
		month = 1
	}

	day := entry.Date.Day
	if day <= 0 {
		day = 1
	}

	return time.Date(entry.Date.Year, time.Month(month), day, entry.Time.Hour, entry.Time.Minute, 0, 0, time.UTC)
}

func trafficSampleFromEntry(machineID uint, periodType string, entry vnstatEntry, rawPayload string, collectedAt time.Time) model.TrafficSample {
	uploadMB := bytesToMB(entry.TX)
	downloadMB := bytesToMB(entry.RX)
	totalMB := bytesToMB(entry.TX + entry.RX)

	return model.TrafficSample{
		MachineID:   machineID,
		PeriodType:  periodType,
		BucketTime:  vnstatEntryTime(entry),
		UploadMB:    uploadMB,
		DownloadMB:  downloadMB,
		TotalMB:     totalMB,
		RawPayload:  rawPayload,
		CollectedAt: collectedAt,
	}
}

func bytesToMB(bytes float64) float64 {
	return math.Round((bytes/(1024*1024))*1000) / 1000
}

func shellEscapeArg(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
