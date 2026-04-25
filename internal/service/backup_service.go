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
	"strings"
	"time"

	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/model"
	"traffic-monitor/internal/repo"

	"golang.org/x/crypto/pbkdf2"
)

const (
	backupVersion              = 1
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
	machineStore  BackupMachineStore
	sshKeyStore   BackupSSHKeyStore
	dataProtector SSHKeyProtector
}

type BackupMachineStore interface {
	Create(ctx context.Context, machine *model.Machine) error
	List(ctx context.Context) ([]model.Machine, error)
}

type BackupSSHKeyStore interface {
	Create(ctx context.Context, sshKey *model.SSHKey) error
	List(ctx context.Context) ([]model.SSHKey, error)
}

type backupPayload struct {
	ExportedAt time.Time              `json:"exported_at"`
	SSHKeys    []backupSSHKeyPayload  `json:"ssh_keys"`
	Machines   []backupMachinePayload `json:"machines"`
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

func NewBackupService(machineStore *repo.MachineRepo, sshKeyStore *repo.SSHKeyRepo, dataProtector SSHKeyProtector) *BackupService {
	return &BackupService{
		machineStore:  machineStore,
		sshKeyStore:   sshKeyStore,
		dataProtector: dataProtector,
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

	payload, err := service.buildBackupPayload(selectedMachines, selectedSSHKeys, sshKeys)
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

func (service *BackupService) buildBackupPayload(machines []model.Machine, sshKeys []model.SSHKey, allSSHKeys []model.SSHKey) (backupPayload, error) {
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

	return backupPayload{
		ExportedAt: time.Now().UTC(),
		SSHKeys:    sshKeyPayloads,
		Machines:   machinePayloads,
	}, nil
}

func (service *BackupService) importPayload(ctx context.Context, payload backupPayload) (dto.BackupImportResp, error) {
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

	machineNameExists := make(map[string]struct{}, len(existingMachines)+len(payload.Machines))
	for _, machine := range existingMachines {
		machineNameExists[machine.Name] = struct{}{}
	}

	for _, item := range payload.Machines {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			return dto.BackupImportResp{}, ErrInvalidBackupPayload
		}

		if _, ok := machineNameExists[name]; ok {
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

		machineNameExists[name] = struct{}{}
		resp.ImportedMachines++
	}

	return resp, nil
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
	if backup.Version != backupVersion || !backup.Encrypted || backup.Payload == "" {
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
