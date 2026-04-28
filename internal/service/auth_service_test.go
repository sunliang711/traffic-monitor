package service

import (
	"context"
	"testing"

	"traffic-monitor/internal/config"
	"traffic-monitor/internal/model"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type stubAdminStore struct {
	adminByID       map[uint]*model.Admin
	adminByUsername map[string]*model.Admin
	createdAdmins   []*model.Admin
	updatedAdminID  uint
	updatedPassword string
}

func (store *stubAdminStore) GetByID(_ context.Context, adminID uint) (*model.Admin, error) {
	admin, ok := store.adminByID[adminID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}

	return admin, nil
}

func (store *stubAdminStore) GetByUsername(_ context.Context, username string) (*model.Admin, error) {
	admin, ok := store.adminByUsername[username]
	if !ok {
		return nil, gorm.ErrRecordNotFound
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

func (store *stubAdminStore) UpdatePasswordHash(_ context.Context, adminID uint, passwordHash string) error {
	store.updatedAdminID = adminID
	store.updatedPassword = passwordHash

	if admin, ok := store.adminByID[adminID]; ok {
		admin.PasswordHash = passwordHash
	}

	return nil
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

func TestAuthServiceChangePassword(t *testing.T) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("old-secret"), bcrypt.DefaultCost)
	require.NoError(t, err)

	store := &stubAdminStore{
		adminByID: map[uint]*model.Admin{
			1: {
				Base:         model.Base{ID: 1},
				Username:     "admin",
				PasswordHash: string(passwordHash),
			},
		},
	}

	service := &AuthService{
		adminStore: store,
	}

	err = service.ChangePassword(context.Background(), 1, "old-secret", "new-secret")
	require.NoError(t, err)
	require.Equal(t, uint(1), store.updatedAdminID)
	require.NotEmpty(t, store.updatedPassword)
	require.NotEqual(t, string(passwordHash), store.updatedPassword)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(store.updatedPassword), []byte("new-secret")))
}

func TestAuthServiceChangePassword_InvalidCurrentPassword(t *testing.T) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("old-secret"), bcrypt.DefaultCost)
	require.NoError(t, err)

	store := &stubAdminStore{
		adminByID: map[uint]*model.Admin{
			1: {
				Base:         model.Base{ID: 1},
				Username:     "admin",
				PasswordHash: string(passwordHash),
			},
		},
	}

	service := &AuthService{
		adminStore: store,
	}

	err = service.ChangePassword(context.Background(), 1, "wrong-secret", "new-secret")
	require.ErrorIs(t, err, ErrInvalidCredentials)
	require.Empty(t, store.updatedPassword)
}

func TestAuthServiceChangePassword_NewPasswordTooShort(t *testing.T) {
	service := &AuthService{
		adminStore: &stubAdminStore{},
	}

	err := service.ChangePassword(context.Background(), 1, "old-secret", "short")
	require.ErrorIs(t, err, ErrPasswordTooShort)
}

func TestAuthServiceResetPasswordByUsername(t *testing.T) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("old-secret"), bcrypt.DefaultCost)
	require.NoError(t, err)

	store := &stubAdminStore{
		adminByID: map[uint]*model.Admin{
			1: {
				Base:         model.Base{ID: 1},
				Username:     "admin",
				PasswordHash: string(passwordHash),
			},
		},
		adminByUsername: map[string]*model.Admin{
			"admin": {
				Base:         model.Base{ID: 1},
				Username:     "admin",
				PasswordHash: string(passwordHash),
			},
		},
	}

	service := &AuthService{
		adminStore: store,
	}

	err = service.ResetPasswordByUsername(context.Background(), "admin", "new-secret")
	require.NoError(t, err)
	require.Equal(t, uint(1), store.updatedAdminID)
	require.NotEmpty(t, store.updatedPassword)
	require.NotEqual(t, string(passwordHash), store.updatedPassword)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(store.updatedPassword), []byte("new-secret")))
}

func TestAuthServiceResetPasswordByUsername_AdminNotFound(t *testing.T) {
	service := &AuthService{
		adminStore: &stubAdminStore{
			adminByUsername: map[string]*model.Admin{},
		},
	}

	err := service.ResetPasswordByUsername(context.Background(), "admin", "new-secret")
	require.ErrorIs(t, err, ErrAdminNotFound)
}

func TestAuthServiceResetPasswordByUsername_NewPasswordTooShort(t *testing.T) {
	service := &AuthService{
		adminStore: &stubAdminStore{},
	}

	err := service.ResetPasswordByUsername(context.Background(), "admin", "short")
	require.ErrorIs(t, err, ErrPasswordTooShort)
}
