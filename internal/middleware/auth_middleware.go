package middleware

import (
	"errors"
	"net/http"

	"traffic-monitor/internal/config"
	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

const (
	currentAdminIDKey      = "current_admin_id"
	currentAdminSessionKey = "current_admin_id"
)

type AuthMiddleware struct {
	authService   *service.AuthService
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
