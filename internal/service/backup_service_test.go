package service

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

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

type stubBackupNotificationChannelStore struct {
	items []*model.NotificationChannel
}

func (store *stubBackupNotificationChannelStore) CreateIfAbsent(_ context.Context, channel *model.NotificationChannel) (bool, error) {
	for _, item := range store.items {
		if item.ChannelType == channel.ChannelType {
			return false, nil
		}
	}

	channel.ID = uint(len(store.items) + 1)
	store.items = append(store.items, channel)
	return true, nil
}

func (store *stubBackupNotificationChannelStore) List(_ context.Context) ([]model.NotificationChannel, error) {
	result := make([]model.NotificationChannel, 0, len(store.items))
	for _, item := range store.items {
		result = append(result, *item)
	}

	return result, nil
}

type stubBackupNotificationProxyStore struct {
	items []*model.NotificationProxy
}

func (store *stubBackupNotificationProxyStore) Create(_ context.Context, notificationProxy *model.NotificationProxy) error {
	notificationProxy.ID = uint(len(store.items) + 1)
	store.items = append(store.items, notificationProxy)
	return nil
}

func (store *stubBackupNotificationProxyStore) List(_ context.Context) ([]model.NotificationProxy, error) {
	result := make([]model.NotificationProxy, 0, len(store.items))
	for _, item := range store.items {
		result = append(result, *item)
	}

	return result, nil
}

type stubBackupThresholdRuleStore struct {
	globalRules    []model.GlobalThresholdRule
	machineRules   []model.MachineThresholdRule
	globalBatches  [][]model.GlobalThresholdRule
	machineBatches [][]model.MachineThresholdRule
}

func (store *stubBackupThresholdRuleStore) ListGlobalRules(_ context.Context) ([]model.GlobalThresholdRule, error) {
	return append([]model.GlobalThresholdRule(nil), store.globalRules...), nil
}

func (store *stubBackupThresholdRuleStore) CreateGlobalRulesIfAbsent(_ context.Context, rules []model.GlobalThresholdRule) (int64, error) {
	store.globalBatches = append(store.globalBatches, rules)

	var created int64
	for _, rule := range rules {
		exists := false
		for _, existing := range store.globalRules {
			if existing.PeriodType == rule.PeriodType && existing.MetricType == rule.MetricType {
				exists = true
				break
			}
		}

		if exists {
			continue
		}

		store.globalRules = append(store.globalRules, rule)
		created++
	}

	return created, nil
}

func (store *stubBackupThresholdRuleStore) ListMachineRulesByMachineIDs(_ context.Context, machineIDs []uint) ([]model.MachineThresholdRule, error) {
	machineIDSet := make(map[uint]struct{}, len(machineIDs))
	for _, machineID := range machineIDs {
		machineIDSet[machineID] = struct{}{}
	}

	result := make([]model.MachineThresholdRule, 0, len(store.machineRules))
	for _, rule := range store.machineRules {
		if _, ok := machineIDSet[rule.MachineID]; ok {
			result = append(result, rule)
		}
	}

	return result, nil
}

func (store *stubBackupThresholdRuleStore) CreateMachineRulesIfAbsent(_ context.Context, rules []model.MachineThresholdRule) (int64, error) {
	store.machineBatches = append(store.machineBatches, rules)

	var created int64
	for _, rule := range rules {
		exists := false
		for _, existing := range store.machineRules {
			if existing.MachineID == rule.MachineID && existing.PeriodType == rule.PeriodType && existing.MetricType == rule.MetricType {
				exists = true
				break
			}
		}

		if exists {
			continue
		}

		store.machineRules = append(store.machineRules, rule)
		created++
	}

	return created, nil
}

func newTestBackupService(
	machineStore *stubBackupMachineStore,
	sshKeyStore *stubSSHKeyStore,
	channelStore *stubBackupNotificationChannelStore,
	proxyStore *stubBackupNotificationProxyStore,
	thresholdStore *stubBackupThresholdRuleStore,
) *BackupService {
	return &BackupService{
		machineStore:             machineStore,
		sshKeyStore:              sshKeyStore,
		notificationChannelStore: channelStore,
		notificationProxyStore:   proxyStore,
		thresholdRuleStore:       thresholdStore,
		dataProtector:            stubProtector{},
	}
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
	exportChannelStore := &stubBackupNotificationChannelStore{
		items: []*model.NotificationChannel{
			{
				Base:        model.Base{ID: 1},
				ChannelType: channelTypeWebhook,
				Enabled:     true,
				ConfigJSON:  `{"url":"https://hook.example.com","method":"POST","proxy_id":7}`,
			},
		},
	}
	exportProxyStore := &stubBackupNotificationProxyStore{
		items: []*model.NotificationProxy{
			{
				Base:      model.Base{ID: 7},
				Name:      "proxy-a",
				ProxyType: "http",
				URL:       "http://127.0.0.1:8080",
			},
		},
	}
	exportThresholdStore := &stubBackupThresholdRuleStore{
		globalRules: []model.GlobalThresholdRule{
			{PeriodType: thresholdPeriodDaily, MetricType: thresholdMetricTotal, ThresholdMB: 1024, Enabled: true},
		},
		machineRules: []model.MachineThresholdRule{
			{MachineID: 1, PeriodType: thresholdPeriodHourly, MetricType: thresholdMetricUpload, ThresholdMB: 512, Enabled: true},
			{MachineID: 99, PeriodType: thresholdPeriodHourly, MetricType: thresholdMetricDownload, ThresholdMB: 256, Enabled: true},
		},
	}
	exportService := newTestBackupService(exportMachineStore, exportSSHKeyStore, exportChannelStore, exportProxyStore, exportThresholdStore)

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
	require.Equal(t, backupVersion, backup.Version)

	require.Len(t, payload.NotificationProxies, 1)
	require.Equal(t, "proxy-a", payload.NotificationProxies[0].Name)

	require.Len(t, payload.NotificationChannels, 1)
	require.Equal(t, channelTypeWebhook, payload.NotificationChannels[0].ChannelType)
	require.Equal(t, "proxy-a", payload.NotificationChannels[0].ProxyName)
	require.NotContains(t, payload.NotificationChannels[0].ConfigJSON, "proxy_id")

	require.Len(t, payload.GlobalThresholdRules, 1)
	require.Len(t, payload.MachineThresholdRules, 1)
	require.Equal(t, "server-a", payload.MachineThresholdRules[0].MachineName)

	importMachineStore := &stubBackupMachineStore{}
	importSSHKeyStore := &stubSSHKeyStore{}
	importChannelStore := &stubBackupNotificationChannelStore{}
	importProxyStore := &stubBackupNotificationProxyStore{}
	importThresholdStore := &stubBackupThresholdRuleStore{}
	importService := newTestBackupService(importMachineStore, importSSHKeyStore, importChannelStore, importProxyStore, importThresholdStore)

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

	require.Equal(t, 1, response.ImportedNotificationProxies)
	require.Equal(t, 1, response.ImportedNotificationChannels)
	require.Equal(t, 2, response.ImportedThresholdRules)
	require.Zero(t, response.SkippedThresholdRules)

	require.Len(t, importProxyStore.items, 1)
	require.Len(t, importChannelStore.items, 1)
	require.Contains(t, importChannelStore.items[0].ConfigJSON, `"proxy_id":`+strconv.FormatUint(uint64(importProxyStore.items[0].ID), 10))
	require.True(t, importChannelStore.items[0].Enabled)

	require.Len(t, importThresholdStore.globalRules, 1)
	require.Len(t, importThresholdStore.machineRules, 1)
	require.Equal(t, importMachineStore.items[0].ID, importThresholdStore.machineRules[0].MachineID)
	require.Equal(t, float64(512), importThresholdStore.machineRules[0].ThresholdMB)
}

func TestBackupServiceImportSkipsExistingNotificationAndThresholdSettings(t *testing.T) {
	exportService := newTestBackupService(
		&stubBackupMachineStore{items: []*model.Machine{{
			Base:             model.Base{ID: 1},
			Name:             "server-a",
			Host:             "10.0.0.1",
			Port:             22,
			SSHUser:          "root",
			NetworkInterface: "eth0",
			SSHKeyID:         1,
			CollectEnabled:   true,
		}}},
		&stubSSHKeyStore{items: []*model.SSHKey{{
			Base:                 model.Base{ID: 1},
			Name:                 "key-a",
			SourceType:           sshKeySourceImported,
			PrivateKeyCiphertext: testBackupPrivateKeyPEM(t),
		}}},
		&stubBackupNotificationChannelStore{items: []*model.NotificationChannel{{
			Base:        model.Base{ID: 1},
			ChannelType: channelTypeTelegram,
			Enabled:     true,
			ConfigJSON:  `{"bot_token":"token","chat_id":"1"}`,
		}}},
		&stubBackupNotificationProxyStore{items: []*model.NotificationProxy{{
			Base:      model.Base{ID: 1},
			Name:      "proxy-a",
			ProxyType: "socks",
			URL:       "socks5://127.0.0.1:1080",
		}}},
		&stubBackupThresholdRuleStore{
			globalRules: []model.GlobalThresholdRule{
				{PeriodType: thresholdPeriodDaily, MetricType: thresholdMetricTotal, ThresholdMB: 1024, Enabled: true},
			},
			machineRules: []model.MachineThresholdRule{
				{MachineID: 1, PeriodType: thresholdPeriodHourly, MetricType: thresholdMetricUpload, ThresholdMB: 512, Enabled: true},
			},
		},
	)

	backup, err := exportService.Export(context.Background(), dto.BackupExportReq{
		Password:           "secret",
		IncludeAllMachines: true,
		IncludeAllSSHKeys:  true,
	})
	require.NoError(t, err)

	importChannelStore := &stubBackupNotificationChannelStore{items: []*model.NotificationChannel{{
		Base:        model.Base{ID: 1},
		ChannelType: channelTypeTelegram,
		Enabled:     false,
		ConfigJSON:  `{"bot_token":"existing","chat_id":"9"}`,
	}}}
	importProxyStore := &stubBackupNotificationProxyStore{items: []*model.NotificationProxy{{
		Base:      model.Base{ID: 3},
		Name:      "proxy-a",
		ProxyType: "socks",
		URL:       "socks5://10.0.0.9:1080",
	}}}
	importThresholdStore := &stubBackupThresholdRuleStore{
		globalRules: []model.GlobalThresholdRule{
			{PeriodType: thresholdPeriodDaily, MetricType: thresholdMetricTotal, ThresholdMB: 2048, Enabled: false},
		},
	}
	importService := newTestBackupService(
		&stubBackupMachineStore{},
		&stubSSHKeyStore{},
		importChannelStore,
		importProxyStore,
		importThresholdStore,
	)

	response, err := importService.Import(context.Background(), dto.BackupImportReq{Password: "secret", Backup: backup})
	require.NoError(t, err)
	require.Equal(t, 1, response.SkippedNotificationChannels)
	require.Zero(t, response.ImportedNotificationChannels)
	require.Equal(t, 1, response.SkippedNotificationProxies)
	require.Zero(t, response.ImportedNotificationProxies)
	require.Equal(t, 1, response.SkippedThresholdRules)
	require.Equal(t, 1, response.ImportedThresholdRules)

	require.Len(t, importProxyStore.items, 1)
	require.Equal(t, "socks5://10.0.0.9:1080", importProxyStore.items[0].URL)
	require.Len(t, importChannelStore.items, 1)
	require.Equal(t, `{"bot_token":"existing","chat_id":"9"}`, importChannelStore.items[0].ConfigJSON)
	require.Len(t, importThresholdStore.globalRules, 1)
	require.Equal(t, float64(2048), importThresholdStore.globalRules[0].ThresholdMB)
	require.Len(t, importThresholdStore.machineRules, 1)
	require.Equal(t, importThresholdStore.machineRules[0].MachineID, uint(1))
}

func TestBackupServiceImportAcceptsLegacyVersionOnePayload(t *testing.T) {
	payload := backupPayload{
		ExportedAt: time.Unix(0, 0).UTC(),
		SSHKeys: []backupSSHKeyPayload{{
			Name:       "key-a",
			SourceType: sshKeySourceImported,
			PrivateKey: testBackupPrivateKeyPEM(t),
		}},
		Machines: []backupMachinePayload{{
			Name:             "server-a",
			Host:             "10.0.0.1",
			Port:             22,
			SSHUser:          "root",
			NetworkInterface: "eth0",
			SSHKeyName:       "key-a",
			CollectEnabled:   true,
		}},
	}

	plaintext, err := json.Marshal(payload)
	require.NoError(t, err)

	backup, err := encryptBackupPayload([]byte("secret"), plaintext)
	require.NoError(t, err)
	backup.Version = 1

	importService := newTestBackupService(
		&stubBackupMachineStore{},
		&stubSSHKeyStore{},
		&stubBackupNotificationChannelStore{},
		&stubBackupNotificationProxyStore{},
		&stubBackupThresholdRuleStore{},
	)

	response, err := importService.Import(context.Background(), dto.BackupImportReq{Password: "secret", Backup: backup})
	require.NoError(t, err)
	require.Equal(t, 1, response.ImportedSSHKeys)
	require.Equal(t, 1, response.ImportedMachines)
	require.Zero(t, response.ImportedNotificationChannels)
	require.Zero(t, response.ImportedThresholdRules)
}

func TestBackupServiceImportRejectsInvalidNotificationAndThresholdPayload(t *testing.T) {
	testCases := map[string]backupPayload{
		"unknown channel type": {
			NotificationChannels: []backupNotificationChannelPayload{{ChannelType: "sms", ConfigJSON: "{}"}},
		},
		"broken channel config": {
			NotificationChannels: []backupNotificationChannelPayload{{ChannelType: channelTypeWebhook, ConfigJSON: "not-json"}},
		},
		"invalid proxy type": {
			NotificationProxies: []backupNotificationProxyPayload{{Name: "proxy-a", ProxyType: "ftp", URL: "127.0.0.1:1080"}},
		},
		"invalid threshold dimension": {
			GlobalThresholdRules: []backupGlobalThresholdPayload{{PeriodType: "weekly", MetricType: thresholdMetricTotal, ThresholdMB: 1}},
		},
		"non positive threshold": {
			GlobalThresholdRules: []backupGlobalThresholdPayload{{PeriodType: thresholdPeriodDaily, MetricType: thresholdMetricTotal, ThresholdMB: 0}},
		},
	}

	for name, payload := range testCases {
		t.Run(name, func(t *testing.T) {
			plaintext, err := json.Marshal(payload)
			require.NoError(t, err)

			backup, err := encryptBackupPayload([]byte("secret"), plaintext)
			require.NoError(t, err)

			importService := newTestBackupService(
				&stubBackupMachineStore{},
				&stubSSHKeyStore{},
				&stubBackupNotificationChannelStore{},
				&stubBackupNotificationProxyStore{},
				&stubBackupThresholdRuleStore{},
			)

			_, err = importService.Import(context.Background(), dto.BackupImportReq{Password: "secret", Backup: backup})
			require.ErrorIs(t, err, ErrInvalidBackupPayload)
		})
	}
}

func testBackupPrivateKeyPEM(t *testing.T) string {
	t.Helper()

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	privateKeyBlock, err := ssh.MarshalPrivateKey(privateKey, "")
	require.NoError(t, err)

	return string(pemEncode(privateKeyBlock))
}

func encryptTestBackup(t *testing.T, payload backupPayload) dto.EncryptedBackup {
	t.Helper()

	plaintext, err := json.Marshal(payload)
	require.NoError(t, err)

	backup, err := encryptBackupPayload([]byte("secret"), plaintext)
	require.NoError(t, err)

	return backup
}

func newEmptyTestBackupService() (*BackupService, *stubBackupMachineStore, *stubSSHKeyStore, *stubBackupNotificationChannelStore, *stubBackupNotificationProxyStore, *stubBackupThresholdRuleStore) {
	machineStore := &stubBackupMachineStore{}
	sshKeyStore := &stubSSHKeyStore{}
	channelStore := &stubBackupNotificationChannelStore{}
	proxyStore := &stubBackupNotificationProxyStore{}
	thresholdStore := &stubBackupThresholdRuleStore{}

	return newTestBackupService(machineStore, sshKeyStore, channelStore, proxyStore, thresholdStore),
		machineStore, sshKeyStore, channelStore, proxyStore, thresholdStore
}

func TestBackupServiceImportWritesNothingWhenLaterSectionIsInvalid(t *testing.T) {
	backup := encryptTestBackup(t, backupPayload{
		SSHKeys: []backupSSHKeyPayload{{
			Name:       "key-a",
			SourceType: sshKeySourceImported,
			PrivateKey: testBackupPrivateKeyPEM(t),
		}},
		Machines: []backupMachinePayload{{
			Name:             "server-a",
			Host:             "10.0.0.1",
			Port:             22,
			SSHUser:          "root",
			NetworkInterface: "eth0",
			SSHKeyName:       "key-a",
			CollectEnabled:   true,
		}},
		NotificationProxies: []backupNotificationProxyPayload{{
			Name:      "proxy-a",
			ProxyType: "http",
			URL:       "http://127.0.0.1:8080",
		}},
		NotificationChannels: []backupNotificationChannelPayload{{
			ChannelType: "sms",
			ConfigJSON:  "{}",
		}},
	})

	importService, machineStore, sshKeyStore, channelStore, proxyStore, thresholdStore := newEmptyTestBackupService()

	_, err := importService.Import(context.Background(), dto.BackupImportReq{Password: "secret", Backup: backup})
	require.ErrorIs(t, err, ErrInvalidBackupPayload)
	require.Empty(t, sshKeyStore.items)
	require.Empty(t, machineStore.items)
	require.Empty(t, proxyStore.items)
	require.Empty(t, channelStore.items)
	require.Empty(t, thresholdStore.globalRules)
	require.Empty(t, thresholdStore.machineRules)
}

func TestBackupServiceImportCollapsesDuplicateThresholdRulesBeforeWriting(t *testing.T) {
	backup := encryptTestBackup(t, backupPayload{
		SSHKeys: []backupSSHKeyPayload{{
			Name:       "key-a",
			SourceType: sshKeySourceImported,
			PrivateKey: testBackupPrivateKeyPEM(t),
		}},
		Machines: []backupMachinePayload{{
			Name:             "server-a",
			Host:             "10.0.0.1",
			Port:             22,
			SSHUser:          "root",
			NetworkInterface: "eth0",
			SSHKeyName:       "key-a",
			CollectEnabled:   true,
		}},
		GlobalThresholdRules: []backupGlobalThresholdPayload{
			{PeriodType: thresholdPeriodDaily, MetricType: thresholdMetricTotal, ThresholdMB: 1024, Enabled: true},
			{PeriodType: thresholdPeriodDaily, MetricType: thresholdMetricTotal, ThresholdMB: 2048, Enabled: false},
		},
		MachineThresholdRules: []backupMachineThresholdPayload{
			{MachineName: "server-a", PeriodType: thresholdPeriodHourly, MetricType: thresholdMetricUpload, ThresholdMB: 512, Enabled: true},
			{MachineName: "server-a", PeriodType: thresholdPeriodHourly, MetricType: thresholdMetricUpload, ThresholdMB: 256, Enabled: false},
		},
	})

	importService, _, _, _, _, thresholdStore := newEmptyTestBackupService()

	response, err := importService.Import(context.Background(), dto.BackupImportReq{Password: "secret", Backup: backup})
	require.NoError(t, err)
	require.Equal(t, 2, response.ImportedThresholdRules)
	require.Equal(t, 2, response.SkippedThresholdRules)

	require.Len(t, thresholdStore.globalBatches, 1)
	require.Len(t, thresholdStore.globalBatches[0], 1)
	require.Equal(t, float64(1024), thresholdStore.globalBatches[0][0].ThresholdMB)
	require.Len(t, thresholdStore.machineBatches, 1)
	require.Len(t, thresholdStore.machineBatches[0], 1)
	require.Equal(t, float64(512), thresholdStore.machineBatches[0][0].ThresholdMB)
}

func TestBackupServiceImportLeavesChannelWithoutProxyWhenProxyNameUnknown(t *testing.T) {
	backup := encryptTestBackup(t, backupPayload{
		NotificationChannels: []backupNotificationChannelPayload{{
			ChannelType: channelTypeWebhook,
			Enabled:     true,
			ConfigJSON:  `{"url":"https://hook.example.com","method":"POST"}`,
			ProxyName:   "missing-proxy",
		}},
	})

	importService, _, _, channelStore, _, _ := newEmptyTestBackupService()

	response, err := importService.Import(context.Background(), dto.BackupImportReq{Password: "secret", Backup: backup})
	require.NoError(t, err)
	require.Equal(t, 1, response.ImportedNotificationChannels)
	require.Len(t, channelStore.items, 1)
	require.NotContains(t, channelStore.items[0].ConfigJSON, backupChannelProxyIDField)
}

func TestBackupServiceExportDropsDanglingChannelProxyID(t *testing.T) {
	exportService := newTestBackupService(
		&stubBackupMachineStore{},
		&stubSSHKeyStore{items: []*model.SSHKey{{
			Base:                 model.Base{ID: 1},
			Name:                 "key-a",
			SourceType:           sshKeySourceImported,
			PrivateKeyCiphertext: testBackupPrivateKeyPEM(t),
		}}},
		&stubBackupNotificationChannelStore{items: []*model.NotificationChannel{{
			Base:        model.Base{ID: 1},
			ChannelType: channelTypeWebhook,
			Enabled:     true,
			ConfigJSON:  `{"url":"https://hook.example.com","proxy_id":999}`,
		}}},
		&stubBackupNotificationProxyStore{},
		&stubBackupThresholdRuleStore{},
	)

	backup, err := exportService.Export(context.Background(), dto.BackupExportReq{Password: "secret", IncludeAllSSHKeys: true})
	require.NoError(t, err)

	plaintext, err := decryptBackupPayload([]byte("secret"), backup)
	require.NoError(t, err)

	var payload backupPayload
	require.NoError(t, json.Unmarshal(plaintext, &payload))
	require.Len(t, payload.NotificationChannels, 1)
	require.Empty(t, payload.NotificationChannels[0].ProxyName)
	require.NotContains(t, payload.NotificationChannels[0].ConfigJSON, backupChannelProxyIDField)
}

func TestBackupServiceExportReplacesUndecodableChannelConfig(t *testing.T) {
	exportService := newTestBackupService(
		&stubBackupMachineStore{},
		&stubSSHKeyStore{items: []*model.SSHKey{{
			Base:                 model.Base{ID: 1},
			Name:                 "key-a",
			SourceType:           sshKeySourceImported,
			PrivateKeyCiphertext: testBackupPrivateKeyPEM(t),
		}}},
		&stubBackupNotificationChannelStore{items: []*model.NotificationChannel{{
			Base:        model.Base{ID: 1},
			ChannelType: channelTypeWebhook,
			Enabled:     true,
			ConfigJSON:  "not-json",
		}}},
		&stubBackupNotificationProxyStore{},
		&stubBackupThresholdRuleStore{},
	)

	backup, err := exportService.Export(context.Background(), dto.BackupExportReq{Password: "secret", IncludeAllSSHKeys: true})
	require.NoError(t, err)

	plaintext, err := decryptBackupPayload([]byte("secret"), backup)
	require.NoError(t, err)

	var payload backupPayload
	require.NoError(t, json.Unmarshal(plaintext, &payload))
	require.Len(t, payload.NotificationChannels, 1)
	require.Equal(t, emptyBackupChannelConfig, payload.NotificationChannels[0].ConfigJSON)

	importService, _, _, channelStore, _, _ := newEmptyTestBackupService()
	_, err = importService.Import(context.Background(), dto.BackupImportReq{Password: "secret", Backup: backup})
	require.NoError(t, err)
	require.Len(t, channelStore.items, 1)
}

func TestBackupServiceImportBindsThresholdRulesToExistingMachine(t *testing.T) {
	backup := encryptTestBackup(t, backupPayload{
		MachineThresholdRules: []backupMachineThresholdPayload{
			{MachineName: "server-a", PeriodType: thresholdPeriodHourly, MetricType: thresholdMetricUpload, ThresholdMB: 512, Enabled: true},
			{MachineName: "server-gone", PeriodType: thresholdPeriodDaily, MetricType: thresholdMetricTotal, ThresholdMB: 1024, Enabled: true},
		},
	})

	machineStore := &stubBackupMachineStore{items: []*model.Machine{{
		Base:             model.Base{ID: 42},
		Name:             "server-a",
		Host:             "10.0.0.1",
		Port:             22,
		SSHUser:          "root",
		NetworkInterface: "eth0",
		SSHKeyID:         1,
	}}}
	thresholdStore := &stubBackupThresholdRuleStore{}
	importService := newTestBackupService(
		machineStore,
		&stubSSHKeyStore{},
		&stubBackupNotificationChannelStore{},
		&stubBackupNotificationProxyStore{},
		thresholdStore,
	)

	response, err := importService.Import(context.Background(), dto.BackupImportReq{Password: "secret", Backup: backup})
	require.NoError(t, err)
	require.Equal(t, 1, response.ImportedThresholdRules)
	require.Equal(t, 1, response.SkippedThresholdRules)
	require.Len(t, thresholdStore.machineRules, 1)
	require.Equal(t, uint(42), thresholdStore.machineRules[0].MachineID)
}

func TestBackupServiceImportRestoresDuplicateProxyNames(t *testing.T) {
	backup := encryptTestBackup(t, backupPayload{
		NotificationProxies: []backupNotificationProxyPayload{
			{Name: "proxy-a", ProxyType: "http", URL: "http://10.0.0.1:8080"},
			{Name: "proxy-a", ProxyType: "socks", URL: "socks5://10.0.0.2:1080"},
		},
		NotificationChannels: []backupNotificationChannelPayload{{
			ChannelType: channelTypeWebhook,
			Enabled:     true,
			ConfigJSON:  `{"url":"https://hook.example.com"}`,
			ProxyName:   "proxy-a",
		}},
	})

	importService, _, _, channelStore, proxyStore, _ := newEmptyTestBackupService()

	response, err := importService.Import(context.Background(), dto.BackupImportReq{Password: "secret", Backup: backup})
	require.NoError(t, err)
	require.Equal(t, 2, response.ImportedNotificationProxies)
	require.Zero(t, response.SkippedNotificationProxies)
	require.Len(t, proxyStore.items, 2)
	require.Contains(t, channelStore.items[0].ConfigJSON, `"proxy_id":`+strconv.FormatUint(uint64(proxyStore.items[0].ID), 10))
}

func TestBackupServiceExportImportPreservesChannelSecrets(t *testing.T) {
	webhookConfig := `{"url":"https://hook.example.com","method":"POST","headers":{"Authorization":"Bearer token-value"},"body":"{\"text\":\"hi\"}"}`
	telegramConfig := `{"bot_token":"123456:AAAA","chat_id":"-100200","message":"alert"}`

	exportService := newTestBackupService(
		&stubBackupMachineStore{},
		&stubSSHKeyStore{items: []*model.SSHKey{{
			Base:                 model.Base{ID: 1},
			Name:                 "key-a",
			SourceType:           sshKeySourceImported,
			PrivateKeyCiphertext: testBackupPrivateKeyPEM(t),
		}}},
		&stubBackupNotificationChannelStore{items: []*model.NotificationChannel{
			{Base: model.Base{ID: 1}, ChannelType: channelTypeWebhook, Enabled: true, ConfigJSON: webhookConfig},
			{Base: model.Base{ID: 2}, ChannelType: channelTypeTelegram, Enabled: false, ConfigJSON: telegramConfig},
		}},
		&stubBackupNotificationProxyStore{},
		&stubBackupThresholdRuleStore{},
	)

	backup, err := exportService.Export(context.Background(), dto.BackupExportReq{Password: "secret", IncludeAllSSHKeys: true})
	require.NoError(t, err)

	importService, _, _, channelStore, _, _ := newEmptyTestBackupService()
	_, err = importService.Import(context.Background(), dto.BackupImportReq{Password: "secret", Backup: backup})
	require.NoError(t, err)
	require.Len(t, channelStore.items, 2)

	restored := make(map[string]*model.NotificationChannel, len(channelStore.items))
	for _, item := range channelStore.items {
		restored[item.ChannelType] = item
	}

	require.JSONEq(t, webhookConfig, restored[channelTypeWebhook].ConfigJSON)
	require.True(t, restored[channelTypeWebhook].Enabled)
	require.JSONEq(t, telegramConfig, restored[channelTypeTelegram].ConfigJSON)
	require.False(t, restored[channelTypeTelegram].Enabled)
}

func TestBackupServiceImportRejectsUnsupportedVersion(t *testing.T) {
	for _, version := range []int{0, backupVersion + 1} {
		backup := encryptTestBackup(t, backupPayload{})
		backup.Version = version

		importService, _, _, _, _, _ := newEmptyTestBackupService()
		_, err := importService.Import(context.Background(), dto.BackupImportReq{Password: "secret", Backup: backup})
		require.ErrorIs(t, err, ErrInvalidBackupPayload)
	}
}

func TestBackupServiceImportRejectsOverlongFieldsBeforeWriting(t *testing.T) {
	validSSHKey := backupSSHKeyPayload{
		Name:       "key-a",
		SourceType: sshKeySourceImported,
		PrivateKey: testBackupPrivateKeyPEM(t),
	}
	validMachine := backupMachinePayload{
		Name:             "server-a",
		Host:             "10.0.0.1",
		Port:             22,
		SSHUser:          "root",
		NetworkInterface: "eth0",
		SSHKeyName:       "key-a",
		CollectEnabled:   true,
	}

	overlongName := strings.Repeat("n", backupMaxNameLength+1)
	overlongProxyName := strings.Repeat("p", backupMaxProxyNameLength+1)

	testCases := map[string]backupPayload{
		"ssh key name": {
			SSHKeys: []backupSSHKeyPayload{{
				Name:       overlongName,
				SourceType: sshKeySourceImported,
				PrivateKey: validSSHKey.PrivateKey,
			}},
		},
		"machine name": func() backupPayload {
			machine := validMachine
			machine.Name = overlongName
			return backupPayload{SSHKeys: []backupSSHKeyPayload{validSSHKey}, Machines: []backupMachinePayload{machine}}
		}(),
		"machine host": func() backupPayload {
			machine := validMachine
			machine.Host = overlongName
			return backupPayload{SSHKeys: []backupSSHKeyPayload{validSSHKey}, Machines: []backupMachinePayload{machine}}
		}(),
		"proxy name": {
			SSHKeys:  []backupSSHKeyPayload{validSSHKey},
			Machines: []backupMachinePayload{validMachine},
			NotificationProxies: []backupNotificationProxyPayload{{
				Name:      overlongProxyName,
				ProxyType: "http",
				URL:       "http://127.0.0.1:8080",
			}},
		},
	}

	for name, payload := range testCases {
		t.Run(name, func(t *testing.T) {
			backup := encryptTestBackup(t, payload)
			importService, machineStore, sshKeyStore, _, proxyStore, _ := newEmptyTestBackupService()

			_, err := importService.Import(context.Background(), dto.BackupImportReq{Password: "secret", Backup: backup})
			require.ErrorIs(t, err, ErrInvalidBackupPayload)
			require.Empty(t, sshKeyStore.items)
			require.Empty(t, machineStore.items)
			require.Empty(t, proxyStore.items)
		})
	}
}

func TestBackupServiceImportAcceptsMaximumLengthFields(t *testing.T) {
	maxName := strings.Repeat("n", backupMaxNameLength)
	backup := encryptTestBackup(t, backupPayload{
		SSHKeys: []backupSSHKeyPayload{{
			Name:       maxName,
			SourceType: sshKeySourceImported,
			PrivateKey: testBackupPrivateKeyPEM(t),
		}},
		NotificationProxies: []backupNotificationProxyPayload{{
			Name:      strings.Repeat("p", backupMaxProxyNameLength),
			ProxyType: "http",
			URL:       "http://127.0.0.1:8080",
		}},
	})

	importService, _, sshKeyStore, _, proxyStore, _ := newEmptyTestBackupService()

	response, err := importService.Import(context.Background(), dto.BackupImportReq{Password: "secret", Backup: backup})
	require.NoError(t, err)
	require.Equal(t, 1, response.ImportedSSHKeys)
	require.Equal(t, 1, response.ImportedNotificationProxies)
	require.Len(t, sshKeyStore.items, 1)
	require.Len(t, proxyStore.items, 1)
}
