package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"traffic-monitor/internal/bootstrap"
	"traffic-monitor/internal/config"
	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/middleware"
	"traffic-monitor/internal/model"
	"traffic-monitor/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type stubAdminAuthService struct {
	profile dto.AdminProfileResp
	err     error
}

type stubSettingsStore struct {
	guestModeEnabled bool
}

func (store *stubSettingsStore) GetByKey(_ context.Context, _ string) (*model.AppSetting, error) {
	if store.guestModeEnabled {
		return &model.AppSetting{SettingValue: "true"}, nil
	}

	return &model.AppSetting{SettingValue: "false"}, nil
}

func (store *stubSettingsStore) Upsert(_ context.Context, setting *model.AppSetting) error {
	store.guestModeEnabled = setting.SettingValue == "true"
	return nil
}

func (service *stubAdminAuthService) Authenticate(_ context.Context, _ string, _ string) (dto.AdminProfileResp, error) {
	if service.err != nil {
		return dto.AdminProfileResp{}, service.err
	}

	return service.profile, nil
}

func (service *stubAdminAuthService) GetProfile(_ context.Context, adminID uint) (dto.AdminProfileResp, error) {
	return dto.AdminProfileResp{
		ID:       adminID,
		Username: "admin",
	}, nil
}

func (service *stubAdminAuthService) ChangePassword(_ context.Context, _ uint, _ string, _ string) error {
	return nil
}

func TestAuthHandlerLogin_SetsSessionExpiresAt(t *testing.T) {
	sessionConfig := authHandlerTestSessionConfig()
	sessionStore := bootstrap.NewSessionStore(sessionConfig)
	engine := newAuthHandlerTestEngine(&AuthHandler{
		authService: &stubAdminAuthService{
			profile: dto.AdminProfileResp{
				ID:       7,
				Username: "admin",
			},
		},
		sessionStore:  sessionStore,
		sessionConfig: sessionConfig,
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{
		"username": "admin",
		"password": "secret"
	}`))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	sessionCookie := requireAuthHandlerResponseCookie(t, response, sessionConfig.CookieName)

	decodeRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	decodeRequest.AddCookie(sessionCookie)
	session, err := sessionStore.Get(decodeRequest, sessionConfig.CookieName)
	require.NoError(t, err)

	adminID, ok := session.Values[middleware.SessionAdminKey()].(uint)
	require.True(t, ok)
	require.Equal(t, uint(7), adminID)

	expiresAtUnix, ok := session.Values[middleware.SessionExpiresAtKey()].(int64)
	require.True(t, ok)
	expiresAt := time.Unix(expiresAtUnix, 0)
	require.True(t, expiresAt.After(time.Now().Add(sessionConfig.MaxAge-time.Minute)))
}

func TestRegisterRoutes_RequiresAdminForReadRoutesWhenGuestDisabled(t *testing.T) {
	engine := newRegisterRoutesTestEngine(false)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/traffic-samples", nil)
	engine.ServeHTTP(response, request)

	require.Equal(t, http.StatusUnauthorized, response.Code)
}

func TestRegisterRoutes_KeepsWriteRoutesProtectedWhenGuestEnabled(t *testing.T) {
	engine := newRegisterRoutesTestEngine(true)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/machines", nil)
	engine.ServeHTTP(response, request)

	require.Equal(t, http.StatusUnauthorized, response.Code)
}

func TestRegisterRoutes_KeepsMachineReadRoutesProtectedWhenGuestEnabled(t *testing.T) {
	engine := newRegisterRoutesTestEngine(true)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/machines", nil)
	engine.ServeHTTP(response, request)

	require.Equal(t, http.StatusUnauthorized, response.Code)
}

func newAuthHandlerTestEngine(handler *AuthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	authGroup := engine.Group("/api/v1/auth")
	authGroup.POST("/login", handler.Login)
	return engine
}

func newRegisterRoutesTestEngine(guestModeEnabled bool) *gin.Engine {
	gin.SetMode(gin.TestMode)

	sessionConfig := authHandlerTestSessionConfig()
	sessionStore := bootstrap.NewSessionStore(sessionConfig)
	authMiddleware := middleware.NewAuthMiddleware(&service.AuthService{}, sessionStore, sessionConfig)
	settingsService := service.NewSettingsService(&stubSettingsStore{guestModeEnabled: guestModeEnabled})
	engine := gin.New()
	RegisterRoutes(
		engine,
		&HealthHandler{},
		&AuthHandler{},
		&RestoreHandler{},
		&SettingsHandler{},
		authMiddleware,
		settingsService,
		&SSHKeyHandler{},
		&MachineHandler{},
		&BackupHandler{},
		&ThresholdHandler{},
		&TrafficSampleHandler{},
		&AlertHandler{},
		config.RestoreConfig{},
	)
	return engine
}

func authHandlerTestSessionConfig() config.SessionConfig {
	return config.SessionConfig{
		Secret:     "test-session-secret",
		CookieName: "traffic_monitor_session",
		MaxAge:     2 * time.Hour,
		Secure:     false,
	}
}

func requireAuthHandlerResponseCookie(t *testing.T, response *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()

	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}

	require.Failf(t, "cookie not found", "cookie %q not found", name)
	return nil
}
