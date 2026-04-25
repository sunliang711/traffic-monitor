package service

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"testing"

	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/model"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

type stubProtector struct{}

func (stubProtector) Encrypt(plaintext []byte) (string, error) {
	return "encrypted:" + string(plaintext), nil
}

type stubSSHKeyStore struct {
	created []*model.SSHKey
	items   []*model.SSHKey
}

func (store *stubSSHKeyStore) Create(_ context.Context, sshKey *model.SSHKey) error {
	sshKey.ID = uint(len(store.created) + 1)
	store.created = append(store.created, sshKey)
	store.items = append(store.items, sshKey)
	return nil
}

func (store *stubSSHKeyStore) List(_ context.Context) ([]model.SSHKey, error) {
	result := make([]model.SSHKey, 0, len(store.items))
	for _, item := range store.items {
		result = append(result, *item)
	}

	return result, nil
}

func (store *stubSSHKeyStore) GetByID(_ context.Context, sshKeyID uint) (*model.SSHKey, error) {
	for _, item := range store.items {
		if item.ID == sshKeyID {
			return item, nil
		}
	}

	return nil, ErrAdminNotFound
}

func (store *stubSSHKeyStore) DeleteByID(_ context.Context, sshKeyID uint) error {
	filtered := make([]*model.SSHKey, 0, len(store.items))
	for _, item := range store.items {
		if item.ID != sshKeyID {
			filtered = append(filtered, item)
		}
	}

	store.items = filtered
	return nil
}

func TestSSHKeyServiceGenerate(t *testing.T) {
	store := &stubSSHKeyStore{}
	service := &SSHKeyService{
		sshKeyStore:   store,
		dataProtector: stubProtector{},
	}

	resp, err := service.Generate(context.Background(), dto.GenerateSSHKeyReq{Name: "generated-key"})
	require.NoError(t, err)
	require.Equal(t, sshKeyTypeED25519, resp.KeyType)
	require.NotEmpty(t, resp.PublicKey)
	require.NotEmpty(t, resp.Fingerprint)
	require.Len(t, store.created, 1)
	require.NotEmpty(t, store.created[0].PrivateKeyCiphertext)
}

func TestSSHKeyServiceImport(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	signer, err := ssh.NewSignerFromKey(privateKey)
	require.NoError(t, err)

	store := &stubSSHKeyStore{}
	service := &SSHKeyService{
		sshKeyStore:   store,
		dataProtector: stubProtector{},
	}

	privateKeyBlockForImport, err := ssh.MarshalPrivateKey(privateKey, "")
	require.NoError(t, err)

	resp, err := service.Import(context.Background(), dto.ImportSSHKeyReq{
		Name:       "imported-key",
		PrivateKey: string(pemEncode(privateKeyBlockForImport)),
	})
	require.NoError(t, err)
	require.Equal(t, sshKeyType(signer.PublicKey()), resp.KeyType)
	require.NotEmpty(t, resp.PublicKey)
	require.NotEmpty(t, resp.Fingerprint)
	require.Len(t, store.created, 1)
}

func pemEncode(block *pem.Block) []byte {
	return pem.EncodeToMemory(block)
}
