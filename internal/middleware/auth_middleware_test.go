package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"traffic-monitor/internal/bootstrap"
	"traffic-monitor/internal/config"
	"traffic-monitor/internal/dto"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/stretchr/testify/require"
)

type stubProfileService struct {
	calls int
}

func (service *stubProfileService) GetProfile(_ context.Context, adminID uint) (dto.AdminProfileResp, error) {
	service.calls++
	return dto.AdminProfileResp{
		ID:       adminID,
		Username: "admin",
	}, nil
}

func TestAuthMiddlewareRequireAdmin_RenewsSessionWhenRemainingTTLBelowHalf(t *testing.T) {
	sessionConfig := authMiddlewareTestSessionConfig()
	sessionStore := bootstrap.NewSessionStore(sessionConfig)
	profileService := &stubProfileService{}
	engine := newAuthMiddlewareTestEngine(&AuthMiddleware{
		authService:   profileService,
		sessionStore:  sessionStore,
		sessionConfig: sessionConfig,
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/machines", nil)
	request.AddCookie(newAuthMiddlewareSessionCookie(t, sessionStore, sessionConfig, time.Now().Add(30*time.Minute), true))
	engine.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, 1, profileService.calls)

	renewedCookie := requireResponseCookie(t, response, sessionConfig.CookieName)
	require.Equal(t, int(sessionConfig.MaxAge/time.Second), renewedCookie.MaxAge)
	renewedExpiresAt := decodeSessionExpiresAt(t, sessionStore, sessionConfig, renewedCookie)
	require.True(t, renewedExpiresAt.After(time.Now().Add(sessionConfig.MaxAge-time.Minute)))
}

func TestAuthMiddlewareRequireAdmin_DoesNotRenewSessionWhenRemainingTTLAboveHalf(t *testing.T) {
	sessionConfig := authMiddlewareTestSessionConfig()
	sessionStore := bootstrap.NewSessionStore(sessionConfig)
	profileService := &stubProfileService{}
	engine := newAuthMiddlewareTestEngine(&AuthMiddleware{
		authService:   profileService,
		sessionStore:  sessionStore,
		sessionConfig: sessionConfig,
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/machines", nil)
	request.AddCookie(newAuthMiddlewareSessionCookie(t, sessionStore, sessionConfig, time.Now().Add(90*time.Minute), true))
	engine.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, 1, profileService.calls)
	require.False(t, responseHasCookie(response, sessionConfig.CookieName))
}

func TestAuthMiddlewareRequireAdmin_RejectsExpiredSession(t *testing.T) {
	sessionConfig := authMiddlewareTestSessionConfig()
	sessionStore := bootstrap.NewSessionStore(sessionConfig)
	profileService := &stubProfileService{}
	engine := newAuthMiddlewareTestEngine(&AuthMiddleware{
		authService:   profileService,
		sessionStore:  sessionStore,
		sessionConfig: sessionConfig,
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/machines", nil)
	request.AddCookie(newAuthMiddlewareSessionCookie(t, sessionStore, sessionConfig, time.Now().Add(-time.Minute), true))
	engine.ServeHTTP(response, request)

	require.Equal(t, http.StatusUnauthorized, response.Code)
	require.Equal(t, 0, profileService.calls)
}

func TestAuthMiddlewareRequireAdmin_RejectsSessionWithoutExpiresAt(t *testing.T) {
	sessionConfig := authMiddlewareTestSessionConfig()
	sessionStore := bootstrap.NewSessionStore(sessionConfig)
	profileService := &stubProfileService{}
	engine := newAuthMiddlewareTestEngine(&AuthMiddleware{
		authService:   profileService,
		sessionStore:  sessionStore,
		sessionConfig: sessionConfig,
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/machines", nil)
	request.AddCookie(newAuthMiddlewareSessionCookie(t, sessionStore, sessionConfig, time.Now().Add(sessionConfig.MaxAge), false))
	engine.ServeHTTP(response, request)

	require.Equal(t, http.StatusUnauthorized, response.Code)
	require.Equal(t, 0, profileService.calls)
}

func newAuthMiddlewareTestEngine(authMiddleware *AuthMiddleware) *gin.Engine {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	apiGroup := engine.Group("/api/v1")
	apiGroup.Use(authMiddleware.RequireAdmin())
	apiGroup.GET("/machines", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, dto.Response{
			Code:    http.StatusOK,
			Data:    nil,
			Message: "success",
		})
	})
	return engine
}

func authMiddlewareTestSessionConfig() config.SessionConfig {
	return config.SessionConfig{
		Secret:     "test-session-secret",
		CookieName: "traffic_monitor_session",
		MaxAge:     2 * time.Hour,
		Secure:     false,
	}
}

func newAuthMiddlewareSessionCookie(t *testing.T, sessionStore *sessions.CookieStore, sessionConfig config.SessionConfig, expiresAt time.Time, withExpiresAt bool) *http.Cookie {
	t.Helper()

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	session, err := sessionStore.Get(request, sessionConfig.CookieName)
	require.NoError(t, err)

	session.Values[SessionAdminKey()] = uint(1)
	if withExpiresAt {
		session.Values[SessionExpiresAtKey()] = expiresAt.Unix()
	}
	require.NoError(t, session.Save(request, response))

	return requireResponseCookie(t, response, sessionConfig.CookieName)
}

func decodeSessionExpiresAt(t *testing.T, sessionStore *sessions.CookieStore, sessionConfig config.SessionConfig, cookie *http.Cookie) time.Time {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(cookie)
	session, err := sessionStore.Get(request, sessionConfig.CookieName)
	require.NoError(t, err)

	expiresAt, ok := sessionExpiresAt(session.Values[SessionExpiresAtKey()])
	require.True(t, ok)
	return expiresAt
}

func requireResponseCookie(t *testing.T, response *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()

	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}

	require.Failf(t, "cookie not found", "cookie %q not found", name)
	return nil
}

func responseHasCookie(response *httptest.ResponseRecorder, name string) bool {
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == name {
			return true
		}
	}

	return false
}
