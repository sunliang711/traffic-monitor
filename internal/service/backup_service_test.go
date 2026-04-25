package service

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"testing"

	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/model"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

type stubBackupMachineStore struct {
	items []*model.Machine
}

func (store *stubBackupMachineStore) Create(_ context.Context, machine *model.Machine) error {
	machine.ID = uint(len(store.items) + 1)
	store.items = append(store.items, machine)
	return nil
}

func (store *stubBackupMachineStore) List(_ context.Context) ([]model.Machine, error) {
	result := make([]model.Machine, 0, len(store.items))
	for _, item := range store.items {
		result = append(result, *item)
	}

	return result, nil
}

func TestBackupServiceExportImport(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	privateKeyBlock, err := ssh.MarshalPrivateKey(privateKey, "")
	require.NoError(t, err)

	privateKeyPEM := string(pemEncode(privateKeyBlock))
	exportMachineStore := &stubBackupMachineStore{
		items: []*model.Machine{
			{
				Base:             model.Base{ID: 1},
				Name:             "server-a",
				Host:             "10.0.0.1",
				Port:             22,
				SSHUser:          "root",
				NetworkInterface: "eth0",
				SSHKeyID:         1,
				CollectEnabled:   true,
			},
		},
	}
	exportSSHKeyStore := &stubSSHKeyStore{
		items: []*model.SSHKey{
			{
				Base:                 model.Base{ID: 1},
				Name:                 "key-a",
				SourceType:           sshKeySourceImported,
				PrivateKeyCiphertext: privateKeyPEM,
			},
		},
	}
	exportService := &BackupService{
		machineStore:  exportMachineStore,
		sshKeyStore:   exportSSHKeyStore,
		dataProtector: stubProtector{},
	}

	backup, err := exportService.Export(context.Background(), dto.BackupExportReq{
		Password:           "secret",
		IncludeAllMachines: true,
		IncludeAllSSHKeys:  false,
	})
	require.NoError(t, err)
	require.True(t, backup.Encrypted)

	plaintext, err := decryptBackupPayload([]byte("secret"), backup)
	require.NoError(t, err)

	var payload backupPayload
	require.NoError(t, json.Unmarshal(plaintext, &payload))
	require.Len(t, payload.Machines, 1)
	require.Len(t, payload.SSHKeys, 1)
	require.Equal(t, "key-a", payload.Machines[0].SSHKeyName)

	importMachineStore := &stubBackupMachineStore{}
	importSSHKeyStore := &stubSSHKeyStore{}
	importService := &BackupService{
		machineStore:  importMachineStore,
		sshKeyStore:   importSSHKeyStore,
		dataProtector: stubProtector{},
	}

	response, err := importService.Import(context.Background(), dto.BackupImportReq{
		Password: "secret",
		Backup:   backup,
	})
	require.NoError(t, err)
	require.Equal(t, 1, response.ImportedSSHKeys)
	require.Equal(t, 1, response.ImportedMachines)
	require.Len(t, importSSHKeyStore.items, 1)
	require.Len(t, importMachineStore.items, 1)
	require.Equal(t, importSSHKeyStore.items[0].ID, importMachineStore.items[0].SSHKeyID)
}
