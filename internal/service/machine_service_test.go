package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"traffic-monitor/internal/config"
	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/model"

	"github.com/stretchr/testify/require"
)

type stubMachineStore struct {
	items  map[uint]*model.Machine
	nextID uint
}

func (store *stubMachineStore) Create(_ context.Context, machine *model.Machine) error {
	store.nextID++
	machine.ID = store.nextID
	machine.CreatedAt = time.Now()
	machine.UpdatedAt = machine.CreatedAt
	store.items[machine.ID] = machine
	return nil
}

func (store *stubMachineStore) List(_ context.Context) ([]model.Machine, error) {
	result := make([]model.Machine, 0, len(store.items))
	for _, item := range store.items {
		result = append(result, *item)
	}

	return result, nil
}

func (store *stubMachineStore) GetByID(_ context.Context, machineID uint) (*model.Machine, error) {
	item, ok := store.items[machineID]
	if !ok {
		return nil, errors.New("get machine by id: record not found")
	}

	return item, nil
}

func (store *stubMachineStore) Update(_ context.Context, machine *model.Machine) error {
	machine.UpdatedAt = time.Now()
	store.items[machine.ID] = machine
	return nil
}

func (store *stubMachineStore) DeleteByID(_ context.Context, machineID uint) error {
	delete(store.items, machineID)
	return nil
}

type stubSSHKeyLookup struct {
	items map[uint]*model.SSHKey
}

func (store *stubSSHKeyLookup) GetByID(_ context.Context, sshKeyID uint) (*model.SSHKey, error) {
	item, ok := store.items[sshKeyID]
	if !ok {
		return nil, errors.New("get ssh key by id: record not found")
	}

	return item, nil
}

type stubDecryptor struct {
	plaintext []byte
}

func (stub stubDecryptor) Encrypt(plaintext []byte) (string, error) {
	return string(plaintext), nil
}

func (stub stubDecryptor) Decrypt(_ string) ([]byte, error) {
	return stub.plaintext, nil
}

type stubSSHRunner struct {
	result  SSHExecutionResult
	err     error
	command *string
}

func (runner stubSSHRunner) Run(_ context.Context, _ string, _ int, _ string, _ []byte, command string) (SSHExecutionResult, error) {
	if runner.command != nil {
		*runner.command = command
	}

	return runner.result, runner.err
}

func TestMachineServiceCreate(t *testing.T) {
	store := &stubMachineStore{items: map[uint]*model.Machine{}}
	service := &MachineService{
		machineStore:  store,
		sshKeyStore:   &stubSSHKeyLookup{items: map[uint]*model.SSHKey{1: {Base: model.Base{ID: 1}}}},
		dataProtector: stubDecryptor{},
		sshRunner:     stubSSHRunner{},
		sshConfig:     config.SSHConfig{DialTimeout: 5 * time.Second, CommandTimeout: 5 * time.Second},
	}

	resp, err := service.Create(context.Background(), dto.CreateMachineReq{
		Name:             "host-a",
		Host:             "10.0.0.1",
		Port:             22,
		SSHUser:          "root",
		NetworkInterface: "eth0",
		SSHKeyID:         1,
		CollectEnabled:   true,
	})
	require.NoError(t, err)
	require.Equal(t, "host-a", resp.Name)
	require.Equal(t, uint(1), resp.SSHKeyID)
}

func TestMachineServiceTestConnection(t *testing.T) {
	store := &stubMachineStore{items: map[uint]*model.Machine{
		1: {
			Base:             model.Base{ID: 1},
			Name:             "host-a",
			Host:             "10.0.0.1",
			Port:             22,
			SSHUser:          "root",
			NetworkInterface: "eth0",
			SSHKeyID:         1,
		},
	}}

	var command string
	service := &MachineService{
		machineStore: store,
		sshKeyStore: &stubSSHKeyLookup{items: map[uint]*model.SSHKey{
			1: {Base: model.Base{ID: 1}, PrivateKeyCiphertext: "cipher"},
		}},
		dataProtector: stubDecryptor{plaintext: []byte("private-key")},
		sshRunner: stubSSHRunner{result: SSHExecutionResult{
			Stdout: "vnStat 2.12 by Teemu Toivola",
		}, command: &command},
		sshConfig: config.SSHConfig{DialTimeout: 5 * time.Second, CommandTimeout: 5 * time.Second},
	}

	resp, err := service.TestConnection(context.Background(), 1)
	require.NoError(t, err)
	require.True(t, resp.SSHReachable)
	require.True(t, resp.VNStatReady)
	require.True(t, resp.NetworkInterfaceReady)
	require.Equal(t, "vnStat 2.12 by Teemu Toivola", resp.VNStatVersion)
	require.Equal(t, "ok", resp.Status)
	require.True(t, strings.Contains(command, "vnstat --json -i 'eth0'"))
}

func TestMachineServiceTestConnectionReturnsVNStatUnavailableForMissingInterface(t *testing.T) {
	store := &stubMachineStore{items: map[uint]*model.Machine{
		1: {
			Base:             model.Base{ID: 1},
			Name:             "host-a",
			Host:             "10.0.0.1",
			Port:             22,
			SSHUser:          "root",
			NetworkInterface: "missing0",
			SSHKeyID:         1,
		},
	}}

	service := &MachineService{
		machineStore: store,
		sshKeyStore: &stubSSHKeyLookup{items: map[uint]*model.SSHKey{
			1: {Base: model.Base{ID: 1}, PrivateKeyCiphertext: "cipher"},
		}},
		dataProtector: stubDecryptor{plaintext: []byte("private-key")},
		sshRunner: stubSSHRunner{
			result: SSHExecutionResult{Stdout: vnstatInterfaceMissingMarker + "\nvnStat 2.12 by Teemu Toivola\n"},
			err:    errors.New("run ssh command: exit status 1"),
		},
		sshConfig: config.SSHConfig{DialTimeout: 5 * time.Second, CommandTimeout: 5 * time.Second},
	}

	resp, err := service.TestConnection(context.Background(), 1)
	require.NoError(t, err)
	require.True(t, resp.SSHReachable)
	require.True(t, resp.VNStatReady)
	require.False(t, resp.NetworkInterfaceReady)
	require.Equal(t, "vnstat_interface_missing", resp.Status)
	require.Equal(t, "vnStat 2.12 by Teemu Toivola", resp.VNStatVersion)
}

func TestMachineServiceTestConnectionReturnsVNStatUnavailableWhenVNStatIsMissing(t *testing.T) {
	store := &stubMachineStore{items: map[uint]*model.Machine{
		1: {
			Base:             model.Base{ID: 1},
			Name:             "host-a",
			Host:             "10.0.0.1",
			Port:             22,
			SSHUser:          "root",
			NetworkInterface: "eth0",
			SSHKeyID:         1,
		},
	}}

	service := &MachineService{
		machineStore: store,
		sshKeyStore: &stubSSHKeyLookup{items: map[uint]*model.SSHKey{
			1: {Base: model.Base{ID: 1}, PrivateKeyCiphertext: "cipher"},
		}},
		dataProtector: stubDecryptor{plaintext: []byte("private-key")},
		sshRunner: stubSSHRunner{
			result: SSHExecutionResult{Stdout: vnstatNotInstalledMarker + "\n"},
			err:    errors.New("run ssh command: exit status 127"),
		},
		sshConfig: config.SSHConfig{DialTimeout: 5 * time.Second, CommandTimeout: 5 * time.Second},
	}

	resp, err := service.TestConnection(context.Background(), 1)
	require.NoError(t, err)
	require.True(t, resp.SSHReachable)
	require.False(t, resp.VNStatReady)
	require.False(t, resp.NetworkInterfaceReady)
	require.Equal(t, "vnstat_not_installed", resp.Status)
	require.Empty(t, resp.VNStatVersion)
}

func TestMachineServiceTestConnectionReturnsSSHFailedWhenSSHIsUnreachable(t *testing.T) {
	store := &stubMachineStore{items: map[uint]*model.Machine{
		1: {
			Base:             model.Base{ID: 1},
			Name:             "host-a",
			Host:             "10.0.0.1",
			Port:             22,
			SSHUser:          "root",
			NetworkInterface: "eth0",
			SSHKeyID:         1,
		},
	}}

	service := &MachineService{
		machineStore: store,
		sshKeyStore: &stubSSHKeyLookup{items: map[uint]*model.SSHKey{
			1: {Base: model.Base{ID: 1}, PrivateKeyCiphertext: "cipher"},
		}},
		dataProtector: stubDecryptor{plaintext: []byte("private-key")},
		sshRunner:     stubSSHRunner{err: errors.New("dial ssh: i/o timeout")},
		sshConfig:     config.SSHConfig{DialTimeout: 5 * time.Second, CommandTimeout: 5 * time.Second},
	}

	resp, err := service.TestConnection(context.Background(), 1)
	require.NoError(t, err)
	require.False(t, resp.SSHReachable)
	require.False(t, resp.VNStatReady)
	require.False(t, resp.NetworkInterfaceReady)
	require.Equal(t, "ssh_failed", resp.Status)
}
