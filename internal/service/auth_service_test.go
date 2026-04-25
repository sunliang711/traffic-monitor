package service

import (
	"context"
	"testing"

	"traffic-monitor/internal/config"
	"traffic-monitor/internal/model"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

type stubAdminStore struct {
	adminByID       map[uint]*model.Admin
	adminByUsername map[string]*model.Admin
	createdAdmins   []*model.Admin
}

func (store *stubAdminStore) GetByID(_ context.Context, adminID uint) (*model.Admin, error) {
	admin, ok := store.adminByID[adminID]
	if !ok {
		return nil, ErrAdminNotFound
	}

	return admin, nil
}

func (store *stubAdminStore) GetByUsername(_ context.Context, username string) (*model.Admin, error) {
	admin, ok := store.adminByUsername[username]
	if !ok {
		return nil, ErrAdminNotFound
	}

	return admin, nil
}

func (store *stubAdminStore) Create(_ context.Context, admin *model.Admin) error {
	store.createdAdmins = append(store.createdAdmins, admin)
	return nil
}

func (store *stubAdminStore) ExistsByUsername(_ context.Context, username string) (bool, error) {
	_, ok := store.adminByUsername[username]
	return ok, nil
}

func TestAuthServiceAuthenticate(t *testing.T) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	require.NoError(t, err)

	service := &AuthService{
		adminStore: &stubAdminStore{
			adminByUsername: map[string]*model.Admin{
				"admin": {
					Base:         model.Base{ID: 1},
					Username:     "admin",
					PasswordHash: string(passwordHash),
				},
			},
		},
	}

	profile, err := service.Authenticate(context.Background(), "admin", "secret")
	require.NoError(t, err)
	require.Equal(t, uint(1), profile.ID)
	require.Equal(t, "admin", profile.Username)
}

func TestAuthServiceAuthenticate_InvalidPassword(t *testing.T) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	require.NoError(t, err)

	service := &AuthService{
		adminStore: &stubAdminStore{
			adminByUsername: map[string]*model.Admin{
				"admin": {
					Base:         model.Base{ID: 1},
					Username:     "admin",
					PasswordHash: string(passwordHash),
				},
			},
		},
	}

	_, err = service.Authenticate(context.Background(), "admin", "other")
	require.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestAuthServiceEnsureBootstrapAdmin(t *testing.T) {
	store := &stubAdminStore{
		adminByUsername: map[string]*model.Admin{},
	}

	service := &AuthService{
		adminStore: store,
		bootstrapConfig: config.BootstrapConfig{
			InitAdminUsername: "admin",
			InitAdminPassword: "secret",
		},
	}

	created, err := service.EnsureBootstrapAdmin(context.Background())
	require.NoError(t, err)
	require.True(t, created)
	require.Len(t, store.createdAdmins, 1)
	require.Equal(t, "admin", store.createdAdmins[0].Username)
	require.NotEmpty(t, store.createdAdmins[0].PasswordHash)
}
