package middleware

import (
	"context"
	"errors"
	"net/http"
	"time"

	"traffic-monitor/internal/config"
	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

const (
	currentAdminIDKey      = "current_admin_id"
	currentAdminSessionKey = "current_admin_id"
	sessionExpiresAtKey    = "expires_at"
)

type ProfileService interface {
	GetProfile(ctx context.Context, adminID uint) (dto.AdminProfileResp, error)
}

type GuestModeService interface {
	IsGuestModeEnabled(ctx context.Context) (bool, error)
}

type AuthMiddleware struct {
	authService   ProfileService
	sessionStore  *sessions.CookieStore
	sessionConfig config.SessionConfig
}

type adminAuthStatus int

const (
	adminAuthOK adminAuthStatus = iota
	adminAuthUnauthorized
	adminAuthInternalError
)

func NewAuthMiddleware(authService *service.AuthService, sessionStore *sessions.CookieStore, sessionConfig config.SessionConfig) *AuthMiddleware {
	return &AuthMiddleware{
		authService:   authService,
		sessionStore:  sessionStore,
		sessionConfig: sessionConfig,
	}
}

func (middleware *AuthMiddleware) RequireAdmin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		adminID, status := middleware.authenticateAdmin(ctx)
		switch status {
		case adminAuthOK:
			ctx.Set(currentAdminIDKey, adminID)
			ctx.Next()
		case adminAuthUnauthorized:
			abortUnauthorized(ctx)
		case adminAuthInternalError:
			internalServerError(ctx)
		}
	}
}

func (middleware *AuthMiddleware) RequireAdminOrGuest(guestModeService GuestModeService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		adminID, status := middleware.authenticateAdmin(ctx)
		switch status {
		case adminAuthOK:
			ctx.Set(currentAdminIDKey, adminID)
			ctx.Next()
			return
		case adminAuthInternalError:
			internalServerError(ctx)
			return
		}

		enabled, err := guestModeService.IsGuestModeEnabled(ctx.Request.Context())
		if err != nil {
			internalServerError(ctx)
			return
		}
		if !enabled {
			abortUnauthorized(ctx)
			return
		}

		ctx.Next()
	}
}

func (middleware *AuthMiddleware) authenticateAdmin(ctx *gin.Context) (uint, adminAuthStatus) {
	session, err := middleware.sessionStore.Get(ctx.Request, middleware.sessionConfig.CookieName)
	if err != nil {
		return 0, adminAuthUnauthorized
	}

	adminID, ok := sessionAdminID(session.Values[currentAdminSessionKey])
	if !ok || adminID == 0 {
		return 0, adminAuthUnauthorized
	}

	now := time.Now()
	expiresAt, ok := sessionExpiresAt(session.Values[sessionExpiresAtKey])
	if !ok || !expiresAt.After(now) {
		return 0, adminAuthUnauthorized
	}

	if _, err := middleware.authService.GetProfile(ctx.Request.Context(), adminID); err != nil {
		if errors.Is(err, service.ErrAdminNotFound) {
			return 0, adminAuthUnauthorized
		}

		return 0, adminAuthInternalError
	}

	if shouldRenewSession(now, expiresAt, middleware.sessionConfig.MaxAge) {
		// expires_at 存放在签名 Cookie 中，用于服务端判断滑动过期。
		session.Values[sessionExpiresAtKey] = now.Add(middleware.sessionConfig.MaxAge).Unix()
		if err := session.Save(ctx.Request, ctx.Writer); err != nil {
			return 0, adminAuthInternalError
		}
	}

	return adminID, adminAuthOK
}

func CurrentAdminID(ctx *gin.Context) (uint, bool) {
	value, exists := ctx.Get(currentAdminIDKey)
	if !exists {
		return 0, false
	}

	adminID, ok := value.(uint)
	return adminID, ok
}

func abortUnauthorized(ctx *gin.Context) {
	ctx.AbortWithStatusJSON(http.StatusUnauthorized, dto.Response{
		Code:    http.StatusUnauthorized,
		Data:    nil,
		Message: "unauthorized",
	})
}

func internalServerError(ctx *gin.Context) {
	ctx.AbortWithStatusJSON(http.StatusInternalServerError, dto.Response{
		Code:    http.StatusInternalServerError,
		Data:    nil,
		Message: "internal server error",
	})
}

func SessionAdminKey() string {
	return currentAdminSessionKey
}

func SessionExpiresAtKey() string {
	return sessionExpiresAtKey
}

func sessionAdminID(value interface{}) (uint, bool) {
	switch typedValue := value.(type) {
	case uint:
		return typedValue, true
	case uint64:
		return uint(typedValue), true
	case int:
		if typedValue <= 0 {
			return 0, false
		}

		return uint(typedValue), true
	default:
		return 0, false
	}
}

func sessionExpiresAt(value interface{}) (time.Time, bool) {
	expiresAtUnix, ok := value.(int64)
	if !ok || expiresAtUnix <= 0 {
		return time.Time{}, false
	}

	return time.Unix(expiresAtUnix, 0), true
}

func shouldRenewSession(now time.Time, expiresAt time.Time, sessionTTL time.Duration) bool {
	if sessionTTL <= 0 {
		return false
	}

	return expiresAt.Sub(now) < sessionTTL/2
}
