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

type AuthMiddleware struct {
	authService   ProfileService
	sessionStore  *sessions.CookieStore
	sessionConfig config.SessionConfig
}

func NewAuthMiddleware(authService *service.AuthService, sessionStore *sessions.CookieStore, sessionConfig config.SessionConfig) *AuthMiddleware {
	return &AuthMiddleware{
		authService:   authService,
		sessionStore:  sessionStore,
		sessionConfig: sessionConfig,
	}
}

func (middleware *AuthMiddleware) RequireAdmin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		session, err := middleware.sessionStore.Get(ctx.Request, middleware.sessionConfig.CookieName)
		if err != nil {
			abortUnauthorized(ctx)
			return
		}

		adminID, ok := sessionAdminID(session.Values[currentAdminSessionKey])
		if !ok || adminID == 0 {
			abortUnauthorized(ctx)
			return
		}

		now := time.Now()
		expiresAt, ok := sessionExpiresAt(session.Values[sessionExpiresAtKey])
		if !ok || !expiresAt.After(now) {
			abortUnauthorized(ctx)
			return
		}

		if _, err := middleware.authService.GetProfile(ctx.Request.Context(), adminID); err != nil {
			if errors.Is(err, service.ErrAdminNotFound) {
				abortUnauthorized(ctx)
				return
			}

			ctx.JSON(http.StatusInternalServerError, dto.Response{
				Code:    http.StatusInternalServerError,
				Data:    nil,
				Message: "internal server error",
			})
			return
		}

		if shouldRenewSession(now, expiresAt, middleware.sessionConfig.MaxAge) {
			// expires_at 存放在签名 Cookie 中，用于服务端判断滑动过期。
			session.Values[sessionExpiresAtKey] = now.Add(middleware.sessionConfig.MaxAge).Unix()
			if err := session.Save(ctx.Request, ctx.Writer); err != nil {
				ctx.JSON(http.StatusInternalServerError, dto.Response{
					Code:    http.StatusInternalServerError,
					Data:    nil,
					Message: "internal server error",
				})
				return
			}
		}

		ctx.Set(currentAdminIDKey, adminID)
		ctx.Next()
	}
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
