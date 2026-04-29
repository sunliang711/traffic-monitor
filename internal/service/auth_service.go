package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"unicode/utf8"

	"traffic-monitor/internal/config"
	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/model"
	"traffic-monitor/internal/repo"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAdminNotFound      = errors.New("admin not found")
	ErrPasswordTooShort   = errors.New("password too short")
)

const (
	minAdminPasswordLength          = 6
	defaultBootstrapAdminUsername   = "admin"
	generatedBootstrapPasswordBytes = 24
)

type BootstrapAdminResult struct {
	Created           bool
	Username          string
	Password          string
	GeneratedPassword bool
}

type AdminStore interface {
	GetByID(ctx context.Context, adminID uint) (*model.Admin, error)
	GetByUsername(ctx context.Context, username string) (*model.Admin, error)
	Create(ctx context.Context, admin *model.Admin) error
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	UpdatePasswordHash(ctx context.Context, adminID uint, passwordHash string) error
}

type AuthService struct {
	adminStore      AdminStore
	bootstrapConfig config.BootstrapConfig
}

func NewAuthService(adminStore *repo.AdminRepo, bootstrapConfig config.BootstrapConfig) *AuthService {
	return &AuthService{
		adminStore:      adminStore,
		bootstrapConfig: bootstrapConfig,
	}
}

func (service *AuthService) Authenticate(ctx context.Context, username string, password string) (dto.AdminProfileResp, error) {
	admin, err := service.adminStore.GetByUsername(ctx, username)
	if err != nil {
		if repo.IsRecordNotFound(err) {
			return dto.AdminProfileResp{}, ErrInvalidCredentials
		}

		return dto.AdminProfileResp{}, fmt.Errorf("get admin for authentication: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)); err != nil {
		return dto.AdminProfileResp{}, ErrInvalidCredentials
	}

	return dto.AdminProfileResp{
		ID:       admin.ID,
		Username: admin.Username,
	}, nil
}

func (service *AuthService) GetProfile(ctx context.Context, adminID uint) (dto.AdminProfileResp, error) {
	admin, err := service.adminStore.GetByID(ctx, adminID)
	if err != nil {
		if repo.IsRecordNotFound(err) {
			return dto.AdminProfileResp{}, ErrAdminNotFound
		}

		return dto.AdminProfileResp{}, fmt.Errorf("get admin profile: %w", err)
	}

	return dto.AdminProfileResp{
		ID:       admin.ID,
		Username: admin.Username,
	}, nil
}

func (service *AuthService) ChangePassword(ctx context.Context, adminID uint, currentPassword string, newPassword string) error {
	if utf8.RuneCountInString(newPassword) < minAdminPasswordLength {
		return ErrPasswordTooShort
	}

	admin, err := service.adminStore.GetByID(ctx, adminID)
	if err != nil {
		if repo.IsRecordNotFound(err) {
			return ErrAdminNotFound
		}

		return fmt.Errorf("get admin for password change: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(currentPassword)); err != nil {
		return ErrInvalidCredentials
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash new admin password: %w", err)
	}

	if err := service.adminStore.UpdatePasswordHash(ctx, adminID, string(passwordHash)); err != nil {
		if repo.IsRecordNotFound(err) {
			return ErrAdminNotFound
		}

		return fmt.Errorf("update admin password hash: %w", err)
	}

	return nil
}

func (service *AuthService) ResetPasswordByUsername(ctx context.Context, username string, newPassword string) error {
	if utf8.RuneCountInString(newPassword) < minAdminPasswordLength {
		return ErrPasswordTooShort
	}

	admin, err := service.adminStore.GetByUsername(ctx, username)
	if err != nil {
		if repo.IsRecordNotFound(err) {
			return ErrAdminNotFound
		}

		return fmt.Errorf("get admin for password reset: %w", err)
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash reset admin password: %w", err)
	}

	if err := service.adminStore.UpdatePasswordHash(ctx, admin.ID, string(passwordHash)); err != nil {
		if repo.IsRecordNotFound(err) {
			return ErrAdminNotFound
		}

		return fmt.Errorf("update reset admin password hash: %w", err)
	}

	return nil
}

func (service *AuthService) EnsureBootstrapAdmin(ctx context.Context) (BootstrapAdminResult, error) {
	username := service.bootstrapConfig.InitAdminUsername
	password := service.bootstrapConfig.InitAdminPassword
	generatePassword := false
	if username == "" {
		if password != "" {
			return BootstrapAdminResult{}, nil
		}
		username = defaultBootstrapAdminUsername
		generatePassword = true
	}

	exists, err := service.adminStore.ExistsByUsername(ctx, username)
	if err != nil {
		return BootstrapAdminResult{}, fmt.Errorf("check bootstrap admin: %w", err)
	}

	if exists {
		return BootstrapAdminResult{Username: username}, nil
	}

	if generatePassword {
		generatedPassword, err := generateBootstrapAdminPassword()
		if err != nil {
			return BootstrapAdminResult{}, err
		}
		password = generatedPassword
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return BootstrapAdminResult{}, fmt.Errorf("hash bootstrap admin password: %w", err)
	}

	admin := &model.Admin{
		Username:     username,
		PasswordHash: string(passwordHash),
	}

	if err := service.adminStore.Create(ctx, admin); err != nil {
		return BootstrapAdminResult{}, fmt.Errorf("create bootstrap admin: %w", err)
	}

	result := BootstrapAdminResult{
		Created:           true,
		Username:          username,
		GeneratedPassword: generatePassword,
	}
	if generatePassword {
		result.Password = password
	}

	return result, nil
}

func generateBootstrapAdminPassword() (string, error) {
	passwordBytes := make([]byte, generatedBootstrapPasswordBytes)
	if _, err := rand.Read(passwordBytes); err != nil {
		return "", fmt.Errorf("generate bootstrap admin password: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(passwordBytes), nil
}
