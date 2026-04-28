package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"traffic-monitor/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

type stubRestorePasswordService struct {
	calls        int
	lastUsername string
	lastPassword string
	err          error
}

func (service *stubRestorePasswordService) ResetPasswordByUsername(_ context.Context, username string, newPassword string) error {
	service.calls++
	service.lastUsername = username
	service.lastPassword = newPassword
	return service.err
}

func TestRestoreHandlerStatus(t *testing.T) {
	engine := newRestoreTestEngine(&RestoreHandler{
		authService: &stubRestorePasswordService{},
		restoreConfig: config.RestoreConfig{
			Mode:  config.RestoreModeAdminPassword,
			Token: "0123456789abcdef0123456789abcdef",
		},
		log:            zerolog.Nop(),
		tokenExpiresAt: time.Now().Add(restoreTokenTTL),
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/restore/status", nil)
	engine.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Body.String(), `"enabled":true`)
	require.Contains(t, response.Body.String(), `"mode":"admin-password"`)
}

func TestRestoreHandlerResetAdminPassword(t *testing.T) {
	passwordService := &stubRestorePasswordService{}
	engine := newRestoreTestEngine(&RestoreHandler{
		authService: passwordService,
		restoreConfig: config.RestoreConfig{
			Mode:  config.RestoreModeAdminPassword,
			Token: "0123456789abcdef0123456789abcdef",
		},
		log:            zerolog.Nop(),
		tokenExpiresAt: time.Now().Add(restoreTokenTTL),
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/restore/admin-password", strings.NewReader(`{
		"username": "admin",
		"restore_token": "0123456789abcdef0123456789abcdef",
		"new_password": "new-secret"
	}`))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, 1, passwordService.calls)
	require.Equal(t, "admin", passwordService.lastUsername)
	require.Equal(t, "new-secret", passwordService.lastPassword)
}

func TestRestoreHandlerResetAdminPassword_RejectsInvalidToken(t *testing.T) {
	passwordService := &stubRestorePasswordService{}
	engine := newRestoreTestEngine(&RestoreHandler{
		authService: passwordService,
		restoreConfig: config.RestoreConfig{
			Mode:  config.RestoreModeAdminPassword,
			Token: "0123456789abcdef0123456789abcdef",
		},
		log:            zerolog.Nop(),
		tokenExpiresAt: time.Now().Add(restoreTokenTTL),
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/restore/admin-password", strings.NewReader(`{
		"username": "admin",
		"restore_token": "wrong-token",
		"new_password": "new-secret"
	}`))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(response, request)

	require.Equal(t, http.StatusForbidden, response.Code)
	require.Equal(t, 0, passwordService.calls)
}

func TestRestoreHandlerResetAdminPassword_RejectsExpiredToken(t *testing.T) {
	passwordService := &stubRestorePasswordService{}
	engine := newRestoreTestEngine(&RestoreHandler{
		authService: passwordService,
		restoreConfig: config.RestoreConfig{
			Mode:  config.RestoreModeAdminPassword,
			Token: "0123456789abcdef0123456789abcdef",
		},
		log:            zerolog.Nop(),
		tokenExpiresAt: time.Now().Add(-time.Second),
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/restore/admin-password", strings.NewReader(`{
		"username": "admin",
		"restore_token": "0123456789abcdef0123456789abcdef",
		"new_password": "new-secret"
	}`))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(response, request)

	require.Equal(t, http.StatusForbidden, response.Code)
	require.Equal(t, 0, passwordService.calls)
}

func TestRestoreHandlerResetAdminPassword_ConsumesTokenOnce(t *testing.T) {
	passwordService := &stubRestorePasswordService{}
	engine := newRestoreTestEngine(&RestoreHandler{
		authService: passwordService,
		restoreConfig: config.RestoreConfig{
			Mode:  config.RestoreModeAdminPassword,
			Token: "0123456789abcdef0123456789abcdef",
		},
		log:            zerolog.Nop(),
		tokenExpiresAt: time.Now().Add(restoreTokenTTL),
	})

	firstResponse := httptest.NewRecorder()
	firstRequest := httptest.NewRequest(http.MethodPost, "/api/v1/restore/admin-password", strings.NewReader(`{
		"username": "admin",
		"restore_token": "0123456789abcdef0123456789abcdef",
		"new_password": "new-secret"
	}`))
	firstRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(firstResponse, firstRequest)

	secondResponse := httptest.NewRecorder()
	secondRequest := httptest.NewRequest(http.MethodPost, "/api/v1/restore/admin-password", strings.NewReader(`{
		"username": "admin",
		"restore_token": "0123456789abcdef0123456789abcdef",
		"new_password": "other-secret"
	}`))
	secondRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(secondResponse, secondRequest)

	require.Equal(t, http.StatusOK, firstResponse.Code)
	require.Equal(t, http.StatusForbidden, secondResponse.Code)
	require.Equal(t, 1, passwordService.calls)
}

func newRestoreTestEngine(handler *RestoreHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	apiGroup := engine.Group("/api/v1")
	handler.RegisterRoutes(apiGroup)
	return engine
}
