package service

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"

	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/model"
	"traffic-monitor/internal/repo"

	"golang.org/x/crypto/ssh"
)

const (
	sshKeySourceImported  = "imported"
	sshKeySourceGenerated = "generated"
	sshKeyTypeED25519     = "ed25519"
)

var ErrDuplicateSSHKeyFingerprint = errors.New("duplicate ssh key fingerprint")

type SSHKeyStore interface {
	Create(ctx context.Context, sshKey *model.SSHKey) error
	List(ctx context.Context) ([]model.SSHKey, error)
	GetByID(ctx context.Context, sshKeyID uint) (*model.SSHKey, error)
	DeleteByID(ctx context.Context, sshKeyID uint) error
}

type SSHKeyService struct {
	sshKeyStore   SSHKeyStore
	dataProtector SSHKeyProtector
}

type SSHKeyProtector interface {
	Encrypt(plaintext []byte) (string, error)
}

func NewSSHKeyService(sshKeyStore *repo.SSHKeyRepo, dataProtector SSHKeyProtector) *SSHKeyService {
	return &SSHKeyService{
		sshKeyStore:   sshKeyStore,
		dataProtector: dataProtector,
	}
}

func (service *SSHKeyService) Import(ctx context.Context, req dto.ImportSSHKeyReq) (dto.SSHKeyResp, error) {
	privateKeyBytes := []byte(req.PrivateKey)
	keyType, publicKey, fingerprint, err := parsePrivateKey(privateKeyBytes)
	if err != nil {
		return dto.SSHKeyResp{}, err
	}

	ciphertext, err := service.dataProtector.Encrypt(privateKeyBytes)
	if err != nil {
		return dto.SSHKeyResp{}, fmt.Errorf("encrypt private key: %w", err)
	}

	sshKey := &model.SSHKey{
		Name:                 req.Name,
		SourceType:           sshKeySourceImported,
		KeyType:              keyType,
		PublicKey:            publicKey,
		PrivateKeyCiphertext: ciphertext,
		Fingerprint:          fingerprint,
	}

	if err := service.sshKeyStore.Create(ctx, sshKey); err != nil {
		return dto.SSHKeyResp{}, mapSSHKeyRepoError(err)
	}

	return toSSHKeyResp(sshKey), nil
}

func (service *SSHKeyService) Generate(ctx context.Context, req dto.GenerateSSHKeyReq) (dto.SSHKeyResp, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return dto.SSHKeyResp{}, fmt.Errorf("generate ed25519 keypair: %w", err)
	}

	privateKeyBlock, err := ssh.MarshalPrivateKey(privateKey, "")
	if err != nil {
		return dto.SSHKeyResp{}, fmt.Errorf("marshal private key: %w", err)
	}

	privateKeyBytes := pem.EncodeToMemory(privateKeyBlock)
	authorizedKey, fingerprint, err := marshalAuthorizedKey(publicKey)
	if err != nil {
		return dto.SSHKeyResp{}, err
	}

	ciphertext, err := service.dataProtector.Encrypt(privateKeyBytes)
	if err != nil {
		return dto.SSHKeyResp{}, fmt.Errorf("encrypt private key: %w", err)
	}

	sshKey := &model.SSHKey{
		Name:                 req.Name,
		SourceType:           sshKeySourceGenerated,
		KeyType:              sshKeyTypeED25519,
		PublicKey:            authorizedKey,
		PrivateKeyCiphertext: ciphertext,
		Fingerprint:          fingerprint,
	}

	if err := service.sshKeyStore.Create(ctx, sshKey); err != nil {
		return dto.SSHKeyResp{}, mapSSHKeyRepoError(err)
	}

	return toSSHKeyResp(sshKey), nil
}

func (service *SSHKeyService) List(ctx context.Context) ([]dto.SSHKeyResp, error) {
	sshKeys, err := service.sshKeyStore.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list ssh keys: %w", err)
	}

	result := make([]dto.SSHKeyResp, 0, len(sshKeys))
	for _, sshKey := range sshKeys {
		result = append(result, toSSHKeyResp(&sshKey))
	}

	return result, nil
}

func (service *SSHKeyService) GetPublicKey(ctx context.Context, sshKeyID uint) (dto.SSHKeyResp, error) {
	sshKey, err := service.sshKeyStore.GetByID(ctx, sshKeyID)
	if err != nil {
		return dto.SSHKeyResp{}, fmt.Errorf("get ssh key public key: %w", err)
	}

	return toSSHKeyResp(sshKey), nil
}

func (service *SSHKeyService) Delete(ctx context.Context, sshKeyID uint) error {
	if err := service.sshKeyStore.DeleteByID(ctx, sshKeyID); err != nil {
		return fmt.Errorf("delete ssh key: %w", err)
	}

	return nil
}

func parsePrivateKey(privateKeyBytes []byte) (string, string, string, error) {
	signer, err := ssh.ParsePrivateKey(privateKeyBytes)
	if err != nil {
		return "", "", "", fmt.Errorf("parse private key: %w", err)
	}

	publicKey := signer.PublicKey()
	return sshKeyType(publicKey), string(ssh.MarshalAuthorizedKey(publicKey)), ssh.FingerprintSHA256(publicKey), nil
}

func marshalAuthorizedKey(publicKey ed25519.PublicKey) (string, string, error) {
	sshPublicKey, err := ssh.NewPublicKey(publicKey)
	if err != nil {
		return "", "", fmt.Errorf("new ssh public key: %w", err)
	}

	authorizedKey := string(ssh.MarshalAuthorizedKey(sshPublicKey))
	return authorizedKey, ssh.FingerprintSHA256(sshPublicKey), nil
}

func sshKeyType(publicKey ssh.PublicKey) string {
	switch publicKey.Type() {
	case ssh.KeyAlgoED25519:
		return sshKeyTypeED25519
	case ssh.KeyAlgoRSA:
		return "rsa"
	case ssh.KeyAlgoECDSA256, ssh.KeyAlgoECDSA384, ssh.KeyAlgoECDSA521:
		return "ecdsa"
	default:
		return publicKey.Type()
	}
}

func toSSHKeyResp(sshKey *model.SSHKey) dto.SSHKeyResp {
	return dto.SSHKeyResp{
		ID:          sshKey.ID,
		Name:        sshKey.Name,
		SourceType:  sshKey.SourceType,
		KeyType:     sshKey.KeyType,
		PublicKey:   sshKey.PublicKey,
		Fingerprint: sshKey.Fingerprint,
		CreatedAt:   sshKey.CreatedAt,
		UpdatedAt:   sshKey.UpdatedAt,
	}
}

func mapSSHKeyRepoError(err error) error {
	if repo.IsDuplicateKeyError(err) {
		return ErrDuplicateSSHKeyFingerprint
	}

	return fmt.Errorf("persist ssh key: %w", err)
}
