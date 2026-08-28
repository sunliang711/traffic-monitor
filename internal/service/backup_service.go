package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/model"
	"traffic-monitor/internal/repo"

	"golang.org/x/crypto/pbkdf2"
)

const (
	backupVersion             = 2
	backupMinSupportedVersion = 1
	backupChannelProxyIDField = "proxy_id"
	emptyBackupChannelConfig  = "{}"
	// 与 model 中对应列的 size 保持一致，避免超长字段在写入一半时才被数据库拒绝。
	backupMaxNameLength        = 255
	backupMaxProxyNameLength   = 100
	backupEncryptionAlgorithm  = "AES-256-GCM"
	backupEncryptionKDF        = "PBKDF2-SHA256"
	backupEncryptionIterations = 210000
	backupSaltSize             = 16
	backupKeySize              = 32
)

var (
	ErrInvalidBackupRequest    = errors.New("invalid backup request")
	ErrInvalidBackupPayload    = errors.New("invalid backup payload")
	ErrBackupDecryptFailed     = errors.New("backup decrypt failed")
	ErrBackupSSHKeyDecryptFail = errors.New("backup ssh key decrypt failed")
)

type BackupService struct {
	machineStore             BackupMachineStore
	sshKeyStore              BackupSSHKeyStore
	notificationChannelStore BackupNotificationChannelStore
	notificationProxyStore   BackupNotificationProxyStore
	thresholdRuleStore       BackupThresholdRuleStore
	dataProtector            SSHKeyProtector
}

type BackupMachineStore interface {
	Create(ctx context.Context, machine *model.Machine) error
	List(ctx context.Context) ([]model.Machine, error)
}

type BackupSSHKeyStore interface {
	Create(ctx context.Context, sshKey *model.SSHKey) error
	List(ctx context.Context) ([]model.SSHKey, error)
}

type BackupNotificationChannelStore interface {
	CreateIfAbsent(ctx context.Context, channel *model.NotificationChannel) (bool, error)
	List(ctx context.Context) ([]model.NotificationChannel, error)
}

type BackupNotificationProxyStore interface {
	Create(ctx context.Context, notificationProxy *model.NotificationProxy) error
	List(ctx context.Context) ([]model.NotificationProxy, error)
}

type BackupThresholdRuleStore interface {
	ListGlobalRules(ctx context.Context) ([]model.GlobalThresholdRule, error)
	CreateGlobalRulesIfAbsent(ctx context.Context, rules []model.GlobalThresholdRule) (int64, error)
	ListMachineRulesByMachineIDs(ctx context.Context, machineIDs []uint) ([]model.MachineThresholdRule, error)
	CreateMachineRulesIfAbsent(ctx context.Context, rules []model.MachineThresholdRule) (int64, error)
}

type backupPayload struct {
	ExportedAt            time.Time                          `json:"exported_at"`
	SSHKeys               []backupSSHKeyPayload              `json:"ssh_keys"`
	Machines              []backupMachinePayload             `json:"machines"`
	NotificationProxies   []backupNotificationProxyPayload   `json:"notification_proxies,omitempty"`
	NotificationChannels  []backupNotificationChannelPayload `json:"notification_channels,omitempty"`
	GlobalThresholdRules  []backupGlobalThresholdPayload     `json:"global_threshold_rules,omitempty"`
	MachineThresholdRules []backupMachineThresholdPayload    `json:"machine_threshold_rules,omitempty"`
}

type backupNotificationProxyPayload struct {
	Name      string `json:"name"`
	ProxyType string `json:"proxy_type"`
	URL       string `json:"url"`
}

type backupNotificationChannelPayload struct {
	ChannelType string `json:"channel_type"`
	Enabled     bool   `json:"enabled"`
	ConfigJSON  string `json:"config_json"`
	ProxyName   string `json:"proxy_name,omitempty"`
}

type backupGlobalThresholdPayload struct {
	PeriodType  string  `json:"period_type"`
	MetricType  string  `json:"metric_type"`
	ThresholdMB float64 `json:"threshold_mb"`
	Enabled     bool    `json:"enabled"`
}

type backupMachineThresholdPayload struct {
	MachineName string  `json:"machine_name"`
	PeriodType  string  `json:"period_type"`
	MetricType  string  `json:"metric_type"`
	ThresholdMB float64 `json:"threshold_mb"`
	Enabled     bool    `json:"enabled"`
}

type backupSSHKeyPayload struct {
	Name       string `json:"name"`
	SourceType string `json:"source_type"`
	PrivateKey string `json:"private_key"`
}

type backupMachinePayload struct {
	Name             string `json:"name"`
	Host             string `json:"host"`
	Port             int    `json:"port"`
	SSHUser          string `json:"ssh_user"`
	NetworkInterface string `json:"network_interface"`
	SSHKeyName       string `json:"ssh_key_name"`
	CollectEnabled   bool   `json:"collect_enabled"`
	Remark           string `json:"remark"`
}

func NewBackupService(
	machineStore *repo.MachineRepo,
	sshKeyStore *repo.SSHKeyRepo,
	notificationChannelStore *repo.NotificationChannelRepo,
	notificationProxyStore *repo.NotificationProxyRepo,
	thresholdRuleStore *repo.ThresholdRuleRepo,
	dataProtector SSHKeyProtector,
) *BackupService {
	return &BackupService{
		machineStore:             machineStore,
		sshKeyStore:              sshKeyStore,
		notificationChannelStore: notificationChannelStore,
		notificationProxyStore:   notificationProxyStore,
		thresholdRuleStore:       thresholdRuleStore,
		dataProtector:            dataProtector,
	}
}

func (service *BackupService) Export(ctx context.Context, req dto.BackupExportReq) (dto.EncryptedBackup, error) {
	if strings.TrimSpace(req.Password) == "" {
		return dto.EncryptedBackup{}, ErrInvalidBackupRequest
	}

	machines, err := service.machineStore.List(ctx)
	if err != nil {
		return dto.EncryptedBackup{}, fmt.Errorf("list machines for backup export: %w", err)
	}

	sshKeys, err := service.sshKeyStore.List(ctx)
	if err != nil {
		return dto.EncryptedBackup{}, fmt.Errorf("list ssh keys for backup export: %w", err)
	}

	selectedMachines := selectBackupMachines(machines, req)
	selectedSSHKeys := selectBackupSSHKeys(sshKeys, selectedMachines, req)
	if len(selectedMachines) == 0 && len(selectedSSHKeys) == 0 {
		return dto.EncryptedBackup{}, ErrInvalidBackupRequest
	}

	payload, err := service.buildBackupPayload(ctx, selectedMachines, selectedSSHKeys, sshKeys)
	if err != nil {
		return dto.EncryptedBackup{}, err
	}

	plaintext, err := json.Marshal(payload)
	if err != nil {
		return dto.EncryptedBackup{}, fmt.Errorf("marshal backup payload: %w", err)
	}

	backup, err := encryptBackupPayload([]byte(req.Password), plaintext)
	clearBytes(plaintext)
	if err != nil {
		return dto.EncryptedBackup{}, err
	}

	return backup, nil
}

func (service *BackupService) Import(ctx context.Context, req dto.BackupImportReq) (dto.BackupImportResp, error) {
	if strings.TrimSpace(req.Password) == "" {
		return dto.BackupImportResp{}, ErrInvalidBackupRequest
	}

	plaintext, err := decryptBackupPayload([]byte(req.Password), req.Backup)
	if err != nil {
		return dto.BackupImportResp{}, err
	}
	defer clearBytes(plaintext)

	var payload backupPayload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return dto.BackupImportResp{}, fmt.Errorf("%w: decode backup payload", ErrInvalidBackupPayload)
	}

	return service.importPayload(ctx, payload)
}

func (service *BackupService) buildBackupPayload(ctx context.Context, machines []model.Machine, sshKeys []model.SSHKey, allSSHKeys []model.SSHKey) (backupPayload, error) {
	sshKeyByID := make(map[uint]model.SSHKey, len(allSSHKeys))
	for _, sshKey := range allSSHKeys {
		sshKeyByID[sshKey.ID] = sshKey
	}

	sshKeyPayloads := make([]backupSSHKeyPayload, 0, len(sshKeys))
	for _, sshKey := range sshKeys {
		privateKey, err := service.dataProtector.Decrypt(sshKey.PrivateKeyCiphertext)
		if err != nil {
			return backupPayload{}, fmt.Errorf("%w: decrypt ssh key %d", ErrBackupSSHKeyDecryptFail, sshKey.ID)
		}

		sshKeyPayloads = append(sshKeyPayloads, backupSSHKeyPayload{
			Name:       sshKey.Name,
			SourceType: sshKey.SourceType,
			PrivateKey: string(privateKey),
		})
		clearBytes(privateKey)
	}

	machinePayloads := make([]backupMachinePayload, 0, len(machines))
	for _, machine := range machines {
		sshKey, ok := sshKeyByID[machine.SSHKeyID]
		if !ok {
			return backupPayload{}, fmt.Errorf("%w: machine ssh key missing", ErrInvalidBackupPayload)
		}

		machinePayloads = append(machinePayloads, backupMachinePayload{
			Name:             machine.Name,
			Host:             machine.Host,
			Port:             machine.Port,
			SSHUser:          machine.SSHUser,
			NetworkInterface: machine.NetworkInterface,
			SSHKeyName:       sshKey.Name,
			CollectEnabled:   machine.CollectEnabled,
			Remark:           machine.Remark,
		})
	}

	notificationProxies, err := service.notificationProxyStore.List(ctx)
	if err != nil {
		return backupPayload{}, fmt.Errorf("list notification proxies for backup export: %w", err)
	}

	notificationChannels, err := service.notificationChannelStore.List(ctx)
	if err != nil {
		return backupPayload{}, fmt.Errorf("list notification channels for backup export: %w", err)
	}

	globalRules, err := service.thresholdRuleStore.ListGlobalRules(ctx)
	if err != nil {
		return backupPayload{}, fmt.Errorf("list global threshold rules for backup export: %w", err)
	}

	machineIDs := make([]uint, 0, len(machines))
	machineNameByID := make(map[uint]string, len(machines))
	for _, machine := range machines {
		machineIDs = append(machineIDs, machine.ID)
		machineNameByID[machine.ID] = machine.Name
	}

	machineRules, err := service.thresholdRuleStore.ListMachineRulesByMachineIDs(ctx, machineIDs)
	if err != nil {
		return backupPayload{}, fmt.Errorf("list machine threshold rules for backup export: %w", err)
	}

	proxyNameByID := make(map[uint]string, len(notificationProxies))
	proxyPayloads := make([]backupNotificationProxyPayload, 0, len(notificationProxies))
	for _, notificationProxy := range notificationProxies {
		proxyNameByID[notificationProxy.ID] = notificationProxy.Name
		proxyPayloads = append(proxyPayloads, backupNotificationProxyPayload{
			Name:      notificationProxy.Name,
			ProxyType: notificationProxy.ProxyType,
			URL:       notificationProxy.URL,
		})
	}

	channelPayloads := make([]backupNotificationChannelPayload, 0, len(notificationChannels))
	for _, channel := range notificationChannels {
		channelPayloads = append(channelPayloads, buildBackupChannelPayload(channel, proxyNameByID))
	}

	globalRulePayloads := make([]backupGlobalThresholdPayload, 0, len(globalRules))
	for _, rule := range globalRules {
		globalRulePayloads = append(globalRulePayloads, backupGlobalThresholdPayload{
			PeriodType:  rule.PeriodType,
			MetricType:  rule.MetricType,
			ThresholdMB: rule.ThresholdMB,
			Enabled:     rule.Enabled,
		})
	}

	machineRulePayloads := make([]backupMachineThresholdPayload, 0, len(machineRules))
	for _, rule := range machineRules {
		machineName, ok := machineNameByID[rule.MachineID]
		if !ok {
			continue
		}

		machineRulePayloads = append(machineRulePayloads, backupMachineThresholdPayload{
			MachineName: machineName,
			PeriodType:  rule.PeriodType,
			MetricType:  rule.MetricType,
			ThresholdMB: rule.ThresholdMB,
			Enabled:     rule.Enabled,
		})
	}

	return backupPayload{
		ExportedAt:            time.Now().UTC(),
		SSHKeys:               sshKeyPayloads,
		Machines:              machinePayloads,
		NotificationProxies:   proxyPayloads,
		NotificationChannels:  channelPayloads,
		GlobalThresholdRules:  globalRulePayloads,
		MachineThresholdRules: machineRulePayloads,
	}, nil
}

func (service *BackupService) importPayload(ctx context.Context, payload backupPayload) (dto.BackupImportResp, error) {
	// 导入不是单个事务，因此所有校验必须先于第一次写入，否则前面的段落会被写入而整体返回失败。
	if err := validateBackupPayload(payload); err != nil {
		return dto.BackupImportResp{}, err
	}

	resp := dto.BackupImportResp{}

	existingSSHKeys, err := service.sshKeyStore.List(ctx)
	if err != nil {
		return dto.BackupImportResp{}, fmt.Errorf("list ssh keys for backup import: %w", err)
	}

	sshKeyIDByName := make(map[string]uint, len(existingSSHKeys)+len(payload.SSHKeys))
	sshKeyIDByFingerprint := make(map[string]uint, len(existingSSHKeys))
	for _, sshKey := range existingSSHKeys {
		sshKeyIDByName[sshKey.Name] = sshKey.ID
		sshKeyIDByFingerprint[sshKey.Fingerprint] = sshKey.ID
	}

	for _, item := range payload.SSHKeys {
		name := strings.TrimSpace(item.Name)
		privateKey := strings.TrimSpace(item.PrivateKey)
		if name == "" || privateKey == "" {
			return dto.BackupImportResp{}, ErrInvalidBackupPayload
		}

		if _, ok := sshKeyIDByName[name]; ok {
			resp.SkippedSSHKeys++
			continue
		}

		keyType, publicKey, fingerprint, err := parsePrivateKey([]byte(privateKey))
		if err != nil {
			return dto.BackupImportResp{}, fmt.Errorf("%w: parse ssh key", ErrInvalidBackupPayload)
		}

		ciphertext, err := service.dataProtector.Encrypt([]byte(privateKey))
		if err != nil {
			return dto.BackupImportResp{}, fmt.Errorf("encrypt imported backup ssh key: %w", err)
		}

		sshKey := &model.SSHKey{
			Name:                 name,
			SourceType:           normalizeBackupSSHKeySource(item.SourceType),
			KeyType:              keyType,
			PublicKey:            publicKey,
			PrivateKeyCiphertext: ciphertext,
			Fingerprint:          fingerprint,
		}

		if err := service.sshKeyStore.Create(ctx, sshKey); err != nil {
			if repo.IsDuplicateKeyError(err) {
				if existingSSHKeyID, ok := sshKeyIDByFingerprint[fingerprint]; ok {
					sshKeyIDByName[name] = existingSSHKeyID
				}
				resp.SkippedSSHKeys++
				continue
			}

			return dto.BackupImportResp{}, fmt.Errorf("create backup ssh key: %w", err)
		}

		sshKeyIDByName[name] = sshKey.ID
		sshKeyIDByFingerprint[fingerprint] = sshKey.ID
		resp.ImportedSSHKeys++
	}

	existingMachines, err := service.machineStore.List(ctx)
	if err != nil {
		return dto.BackupImportResp{}, fmt.Errorf("list machines for backup import: %w", err)
	}

	machineIDByName := make(map[string]uint, len(existingMachines)+len(payload.Machines))
	for _, machine := range existingMachines {
		machineIDByName[machine.Name] = machine.ID
	}

	for _, item := range payload.Machines {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			return dto.BackupImportResp{}, ErrInvalidBackupPayload
		}

		if _, ok := machineIDByName[name]; ok {
			resp.SkippedMachines++
			continue
		}

		sshKeyID, ok := sshKeyIDByName[strings.TrimSpace(item.SSHKeyName)]
		if !ok {
			resp.SkippedMachines++
			continue
		}

		machine := &model.Machine{
			Name:             name,
			Host:             strings.TrimSpace(item.Host),
			Port:             item.Port,
			SSHUser:          strings.TrimSpace(item.SSHUser),
			NetworkInterface: strings.TrimSpace(item.NetworkInterface),
			SSHKeyID:         sshKeyID,
			CollectEnabled:   item.CollectEnabled,
			Remark:           strings.TrimSpace(item.Remark),
		}

		if err := validateBackupMachine(machine); err != nil {
			return dto.BackupImportResp{}, err
		}

		if err := service.machineStore.Create(ctx, machine); err != nil {
			return dto.BackupImportResp{}, fmt.Errorf("create backup machine: %w", err)
		}

		machineIDByName[name] = machine.ID
		resp.ImportedMachines++
	}

	proxyIDByName, err := service.importNotificationProxies(ctx, payload.NotificationProxies, &resp)
	if err != nil {
		return dto.BackupImportResp{}, err
	}

	if err := service.importNotificationChannels(ctx, payload.NotificationChannels, proxyIDByName, &resp); err != nil {
		return dto.BackupImportResp{}, err
	}

	if err := service.importThresholdRules(ctx, payload, machineIDByName, &resp); err != nil {
		return dto.BackupImportResp{}, err
	}

	return resp, nil
}

func (service *BackupService) importNotificationProxies(ctx context.Context, payloads []backupNotificationProxyPayload, resp *dto.BackupImportResp) (map[string]uint, error) {
	existingProxies, err := service.notificationProxyStore.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list notification proxies for backup import: %w", err)
	}

	// 代理名没有唯一约束，导入判重只看目标实例原有的名字：备份内部的同名代理会被全部还原，不静默丢弃。
	existingProxyNames := make(map[string]struct{}, len(existingProxies))
	proxyIDByName := make(map[string]uint, len(existingProxies)+len(payloads))
	for _, existingProxy := range existingProxies {
		existingProxyNames[existingProxy.Name] = struct{}{}
		if _, ok := proxyIDByName[existingProxy.Name]; !ok {
			proxyIDByName[existingProxy.Name] = existingProxy.ID
		}
	}

	for _, item := range payloads {
		name := strings.TrimSpace(item.Name)
		if _, ok := existingProxyNames[name]; ok {
			resp.SkippedNotificationProxies++
			continue
		}

		proxyType := normalizeNotificationProxyType(item.ProxyType)
		parsedProxyURL, err := parseNotificationProxyURL(proxyType, strings.TrimSpace(item.URL))
		if err != nil {
			return nil, fmt.Errorf("%w: parse notification proxy url", ErrInvalidBackupPayload)
		}

		notificationProxy := &model.NotificationProxy{
			Name:      name,
			ProxyType: proxyType,
			URL:       parsedProxyURL.String(),
		}

		if err := service.notificationProxyStore.Create(ctx, notificationProxy); err != nil {
			return nil, fmt.Errorf("create backup notification proxy: %w", err)
		}

		if _, ok := proxyIDByName[name]; !ok {
			proxyIDByName[name] = notificationProxy.ID
		}
		resp.ImportedNotificationProxies++
	}

	return proxyIDByName, nil
}

func (service *BackupService) importNotificationChannels(ctx context.Context, payloads []backupNotificationChannelPayload, proxyIDByName map[string]uint, resp *dto.BackupImportResp) error {
	existingChannels, err := service.notificationChannelStore.List(ctx)
	if err != nil {
		return fmt.Errorf("list notification channels for backup import: %w", err)
	}

	channelTypeExists := make(map[string]struct{}, len(existingChannels)+len(payloads))
	for _, existingChannel := range existingChannels {
		channelTypeExists[existingChannel.ChannelType] = struct{}{}
	}

	for _, item := range payloads {
		channelType := strings.TrimSpace(item.ChannelType)
		if _, ok := channelTypeExists[channelType]; ok {
			resp.SkippedNotificationChannels++
			continue
		}

		configJSON, err := applyBackupChannelProxy(item.ConfigJSON, item.ProxyName, proxyIDByName)
		if err != nil {
			return err
		}

		channel := &model.NotificationChannel{
			ChannelType: channelType,
			Enabled:     item.Enabled,
			ConfigJSON:  configJSON,
		}

		created, err := service.notificationChannelStore.CreateIfAbsent(ctx, channel)
		if err != nil {
			return fmt.Errorf("create backup notification channel: %w", err)
		}

		channelTypeExists[channelType] = struct{}{}
		if !created {
			resp.SkippedNotificationChannels++
			continue
		}

		resp.ImportedNotificationChannels++
	}

	return nil
}

func (service *BackupService) importThresholdRules(ctx context.Context, payload backupPayload, machineIDByName map[string]uint, resp *dto.BackupImportResp) error {
	if err := service.importGlobalThresholdRules(ctx, payload.GlobalThresholdRules, resp); err != nil {
		return err
	}

	return service.importMachineThresholdRules(ctx, payload.MachineThresholdRules, machineIDByName, resp)
}

func (service *BackupService) importGlobalThresholdRules(ctx context.Context, payloads []backupGlobalThresholdPayload, resp *dto.BackupImportResp) error {
	if len(payloads) == 0 {
		return nil
	}

	existingRules, err := service.thresholdRuleStore.ListGlobalRules(ctx)
	if err != nil {
		return fmt.Errorf("list global threshold rules for backup import: %w", err)
	}

	ruleKeyExists := make(map[string]struct{}, len(existingRules)+len(payloads))
	for _, existingRule := range existingRules {
		ruleKeyExists[thresholdRuleKey(existingRule.PeriodType, existingRule.MetricType)] = struct{}{}
	}

	rules := make([]model.GlobalThresholdRule, 0, len(payloads))
	for _, item := range payloads {
		key := thresholdRuleKey(item.PeriodType, item.MetricType)
		if _, ok := ruleKeyExists[key]; ok {
			resp.SkippedThresholdRules++
			continue
		}

		ruleKeyExists[key] = struct{}{}
		rules = append(rules, model.GlobalThresholdRule{
			PeriodType:  item.PeriodType,
			MetricType:  item.MetricType,
			ThresholdMB: item.ThresholdMB,
			Enabled:     item.Enabled,
		})
	}

	created, err := service.thresholdRuleStore.CreateGlobalRulesIfAbsent(ctx, rules)
	if err != nil {
		return fmt.Errorf("create backup global threshold rules: %w", err)
	}

	resp.ImportedThresholdRules += int(created)
	resp.SkippedThresholdRules += len(rules) - int(created)
	return nil
}

func (service *BackupService) importMachineThresholdRules(ctx context.Context, payloads []backupMachineThresholdPayload, machineIDByName map[string]uint, resp *dto.BackupImportResp) error {
	if len(payloads) == 0 {
		return nil
	}

	machineIDSet := make(map[uint]struct{}, len(payloads))
	for _, item := range payloads {
		if machineID, ok := machineIDByName[strings.TrimSpace(item.MachineName)]; ok {
			machineIDSet[machineID] = struct{}{}
		}
	}

	machineIDs := make([]uint, 0, len(machineIDSet))
	for machineID := range machineIDSet {
		machineIDs = append(machineIDs, machineID)
	}

	existingRules, err := service.thresholdRuleStore.ListMachineRulesByMachineIDs(ctx, machineIDs)
	if err != nil {
		return fmt.Errorf("list machine threshold rules for backup import: %w", err)
	}

	ruleKeyExists := make(map[string]struct{}, len(existingRules)+len(payloads))
	for _, existingRule := range existingRules {
		ruleKeyExists[machineThresholdRuleKey(existingRule.MachineID, existingRule.PeriodType, existingRule.MetricType)] = struct{}{}
	}

	rules := make([]model.MachineThresholdRule, 0, len(payloads))
	for _, item := range payloads {
		machineID, ok := machineIDByName[strings.TrimSpace(item.MachineName)]
		if !ok {
			resp.SkippedThresholdRules++
			continue
		}

		key := machineThresholdRuleKey(machineID, item.PeriodType, item.MetricType)
		if _, exists := ruleKeyExists[key]; exists {
			resp.SkippedThresholdRules++
			continue
		}

		ruleKeyExists[key] = struct{}{}
		rules = append(rules, model.MachineThresholdRule{
			MachineID:   machineID,
			PeriodType:  item.PeriodType,
			MetricType:  item.MetricType,
			ThresholdMB: item.ThresholdMB,
			Enabled:     item.Enabled,
		})
	}

	created, err := service.thresholdRuleStore.CreateMachineRulesIfAbsent(ctx, rules)
	if err != nil {
		return fmt.Errorf("create backup machine threshold rules: %w", err)
	}

	resp.ImportedThresholdRules += int(created)
	resp.SkippedThresholdRules += len(rules) - int(created)
	return nil
}

func selectBackupMachines(machines []model.Machine, req dto.BackupExportReq) []model.Machine {
	if req.IncludeAllMachines {
		return machines
	}

	machineIDSet := uintSet(req.MachineIDs)
	result := make([]model.Machine, 0, len(machineIDSet))
	for _, machine := range machines {
		if _, ok := machineIDSet[machine.ID]; ok {
			result = append(result, machine)
		}
	}

	return result
}

func selectBackupSSHKeys(sshKeys []model.SSHKey, machines []model.Machine, req dto.BackupExportReq) []model.SSHKey {
	sshKeyIDSet := make(map[uint]struct{})
	if req.IncludeAllSSHKeys {
		for _, sshKey := range sshKeys {
			sshKeyIDSet[sshKey.ID] = struct{}{}
		}
	} else {
		sshKeyIDSet = uintSet(req.SSHKeyIDs)
	}

	for _, machine := range machines {
		sshKeyIDSet[machine.SSHKeyID] = struct{}{}
	}

	result := make([]model.SSHKey, 0, len(sshKeyIDSet))
	for _, sshKey := range sshKeys {
		if _, ok := sshKeyIDSet[sshKey.ID]; ok {
			result = append(result, sshKey)
		}
	}

	return result
}

func encryptBackupPayload(password []byte, plaintext []byte) (dto.EncryptedBackup, error) {
	salt, err := randomBytes(backupSaltSize)
	if err != nil {
		return dto.EncryptedBackup{}, err
	}

	key := pbkdf2.Key(password, salt, backupEncryptionIterations, backupKeySize, sha256.New)
	defer clearBytes(key)

	block, err := aes.NewCipher(key)
	if err != nil {
		return dto.EncryptedBackup{}, fmt.Errorf("create backup cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return dto.EncryptedBackup{}, fmt.Errorf("create backup gcm: %w", err)
	}

	nonce, err := randomBytes(gcm.NonceSize())
	if err != nil {
		return dto.EncryptedBackup{}, err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	return dto.EncryptedBackup{
		Version:   backupVersion,
		Encrypted: true,
		Encryption: dto.BackupEncryption{
			Algorithm:  backupEncryptionAlgorithm,
			KDF:        backupEncryptionKDF,
			Iterations: backupEncryptionIterations,
			Salt:       base64.StdEncoding.EncodeToString(salt),
			Nonce:      base64.StdEncoding.EncodeToString(nonce),
		},
		Payload: base64.StdEncoding.EncodeToString(ciphertext),
	}, nil
}

func decryptBackupPayload(password []byte, backup dto.EncryptedBackup) ([]byte, error) {
	if backup.Version < backupMinSupportedVersion || backup.Version > backupVersion || !backup.Encrypted || backup.Payload == "" {
		return nil, ErrInvalidBackupPayload
	}

	if backup.Encryption.Algorithm != backupEncryptionAlgorithm || backup.Encryption.KDF != backupEncryptionKDF || backup.Encryption.Iterations != backupEncryptionIterations {
		return nil, ErrInvalidBackupPayload
	}

	salt, err := base64.StdEncoding.DecodeString(backup.Encryption.Salt)
	if err != nil || len(salt) == 0 {
		return nil, ErrInvalidBackupPayload
	}

	nonce, err := base64.StdEncoding.DecodeString(backup.Encryption.Nonce)
	if err != nil || len(nonce) == 0 {
		return nil, ErrInvalidBackupPayload
	}

	ciphertext, err := base64.StdEncoding.DecodeString(backup.Payload)
	if err != nil || len(ciphertext) == 0 {
		return nil, ErrInvalidBackupPayload
	}

	key := pbkdf2.Key(password, salt, backup.Encryption.Iterations, backupKeySize, sha256.New)
	defer clearBytes(key)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create backup cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create backup gcm: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrBackupDecryptFailed
	}

	return plaintext, nil
}

func randomBytes(size int) ([]byte, error) {
	buffer := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, buffer); err != nil {
		return nil, fmt.Errorf("generate random bytes: %w", err)
	}

	return buffer, nil
}

func uintSet(values []uint) map[uint]struct{} {
	result := make(map[uint]struct{}, len(values))
	for _, value := range values {
		if value > 0 {
			result[value] = struct{}{}
		}
	}

	return result
}

func normalizeBackupSSHKeySource(sourceType string) string {
	switch strings.TrimSpace(sourceType) {
	case sshKeySourceGenerated:
		return sshKeySourceGenerated
	default:
		return sshKeySourceImported
	}
}

// validateBackupPayload 在写入任何数据前完整校验备份内容，保证非法备份不会留下写了一半的记录。
func validateBackupPayload(payload backupPayload) error {
	for _, item := range payload.SSHKeys {
		name := strings.TrimSpace(item.Name)
		privateKey := strings.TrimSpace(item.PrivateKey)
		if name == "" || privateKey == "" {
			return fmt.Errorf("%w: ssh key", ErrInvalidBackupPayload)
		}

		if err := validateBackupFieldLength("ssh key name", name, backupMaxNameLength); err != nil {
			return err
		}

		if _, _, _, err := parsePrivateKey([]byte(privateKey)); err != nil {
			return fmt.Errorf("%w: parse ssh key", ErrInvalidBackupPayload)
		}
	}

	for _, item := range payload.Machines {
		if err := validateBackupMachinePayload(item); err != nil {
			return err
		}
	}

	for _, item := range payload.NotificationProxies {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			return fmt.Errorf("%w: notification proxy name", ErrInvalidBackupPayload)
		}

		if err := validateBackupFieldLength("notification proxy name", name, backupMaxProxyNameLength); err != nil {
			return err
		}

		if _, err := parseNotificationProxyURL(item.ProxyType, item.URL); err != nil {
			return fmt.Errorf("%w: parse notification proxy url", ErrInvalidBackupPayload)
		}
	}

	for _, item := range payload.NotificationChannels {
		if !isBackupNotificationChannelType(strings.TrimSpace(item.ChannelType)) {
			return fmt.Errorf("%w: notification channel type", ErrInvalidBackupPayload)
		}

		if _, err := decodeBackupChannelConfig(item.ConfigJSON); err != nil {
			return err
		}
	}

	for _, item := range payload.GlobalThresholdRules {
		if err := validateBackupThresholdRule(item.PeriodType, item.MetricType, item.ThresholdMB); err != nil {
			return err
		}
	}

	for _, item := range payload.MachineThresholdRules {
		if err := validateBackupThresholdRule(item.PeriodType, item.MetricType, item.ThresholdMB); err != nil {
			return err
		}
	}

	return nil
}

func validateBackupMachinePayload(item backupMachinePayload) error {
	fields := map[string]string{
		"machine name":              strings.TrimSpace(item.Name),
		"machine host":              strings.TrimSpace(item.Host),
		"machine ssh user":          strings.TrimSpace(item.SSHUser),
		"machine network interface": strings.TrimSpace(item.NetworkInterface),
	}

	for field, value := range fields {
		if value == "" {
			return fmt.Errorf("%w: %s is empty", ErrInvalidBackupPayload, field)
		}

		if err := validateBackupFieldLength(field, value, backupMaxNameLength); err != nil {
			return err
		}
	}

	if item.Port <= 0 || item.Port > 65535 {
		return fmt.Errorf("%w: machine port", ErrInvalidBackupPayload)
	}

	return nil
}

func validateBackupFieldLength(field string, value string, maxLength int) error {
	if utf8.RuneCountInString(value) > maxLength {
		return fmt.Errorf("%w: %s exceeds %d characters", ErrInvalidBackupPayload, field, maxLength)
	}

	return nil
}

func buildBackupChannelPayload(channel model.NotificationChannel, proxyNameByID map[uint]string) backupNotificationChannelPayload {
	payload := backupNotificationChannelPayload{
		ChannelType: channel.ChannelType,
		Enabled:     channel.Enabled,
		ConfigJSON:  emptyBackupChannelConfig,
	}

	config, err := decodeBackupChannelConfig(channel.ConfigJSON)
	if err != nil {
		// 配置已损坏时只保留渠道本身，避免把源实例的 proxy_id 带进备份，也避免导出无法再导入的文件。
		return payload
	}

	if proxyID, ok := backupChannelProxyID(config); ok {
		payload.ProxyName = proxyNameByID[proxyID]
	}

	delete(config, backupChannelProxyIDField)
	encodedConfig, err := json.Marshal(config)
	if err != nil {
		return payload
	}

	payload.ConfigJSON = string(encodedConfig)
	return payload
}

// applyBackupChannelProxy 用代理名重新解析出本实例的 proxy_id，避免导入源实例的 ID 造成悬空引用。
func applyBackupChannelProxy(configJSON string, proxyName string, proxyIDByName map[string]uint) (string, error) {
	config, err := decodeBackupChannelConfig(configJSON)
	if err != nil {
		return "", err
	}

	delete(config, backupChannelProxyIDField)

	if name := strings.TrimSpace(proxyName); name != "" {
		if proxyID, ok := proxyIDByName[name]; ok {
			config[backupChannelProxyIDField] = proxyID
		}
	}

	encodedConfig, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("encode backup notification channel config: %w", err)
	}

	return string(encodedConfig), nil
}

func decodeBackupChannelConfig(configJSON string) (map[string]any, error) {
	trimmedConfig := strings.TrimSpace(configJSON)
	if trimmedConfig == "" {
		return map[string]any{}, nil
	}

	decoder := json.NewDecoder(strings.NewReader(trimmedConfig))
	decoder.UseNumber()

	var config map[string]any
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("%w: decode notification channel config", ErrInvalidBackupPayload)
	}

	if config == nil {
		return map[string]any{}, nil
	}

	return config, nil
}

func backupChannelProxyID(config map[string]any) (uint, bool) {
	rawProxyID, ok := config[backupChannelProxyIDField]
	if !ok || rawProxyID == nil {
		return 0, false
	}

	number, ok := rawProxyID.(json.Number)
	if !ok {
		return 0, false
	}

	proxyID, err := number.Int64()
	if err != nil || proxyID <= 0 {
		return 0, false
	}

	return uint(proxyID), true
}

func isBackupNotificationChannelType(channelType string) bool {
	switch channelType {
	case channelTypeWebhook, channelTypeTelegram:
		return true
	default:
		return false
	}
}

func validateBackupThresholdRule(periodType string, metricType string, thresholdMB float64) error {
	if err := validateThresholdDimension(periodType, metricType); err != nil {
		return fmt.Errorf("%w: threshold rule dimension", ErrInvalidBackupPayload)
	}

	if thresholdMB <= 0 {
		return fmt.Errorf("%w: threshold rule value", ErrInvalidBackupPayload)
	}

	return nil
}

func machineThresholdRuleKey(machineID uint, periodType string, metricType string) string {
	return strconv.FormatUint(uint64(machineID), 10) + ":" + thresholdRuleKey(periodType, metricType)
}

func validateBackupMachine(machine *model.Machine) error {
	if machine.Name == "" || machine.Host == "" || machine.SSHUser == "" || machine.NetworkInterface == "" || machine.SSHKeyID == 0 {
		return ErrInvalidBackupPayload
	}

	if machine.Port <= 0 || machine.Port > 65535 {
		return ErrInvalidBackupPayload
	}

	return nil
}

func clearBytes(value []byte) {
	for index := range value {
		value[index] = 0
	}
}
