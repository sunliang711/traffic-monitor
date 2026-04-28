package handler

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"traffic-monitor/internal/config"
	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

const restoreTokenTTL = 5 * time.Minute

type RestorePasswordService interface {
	ResetPasswordByUsername(ctx context.Context, username string, newPassword string) error
}

type RestoreHandler struct {
	authService    RestorePasswordService
	restoreConfig  config.RestoreConfig
	log            zerolog.Logger
	tokenMu        sync.Mutex
	tokenUsed      bool
	tokenExpiresAt time.Time
}

func NewRestoreHandler(authService *service.AuthService, restoreConfig config.RestoreConfig, log zerolog.Logger) *RestoreHandler {
	return &RestoreHandler{
		authService:    authService,
		restoreConfig:  restoreConfig,
		log:            log.With().Str("component", "restore").Logger(),
		tokenExpiresAt: time.Now().Add(restoreTokenTTL),
	}
}

func (handler *RestoreHandler) RegisterRoutes(apiGroup *gin.RouterGroup) {
	restoreGroup := apiGroup.Group("/restore")
	restoreGroup.GET("/status", handler.Status)
	restoreGroup.POST("/admin-password", handler.ResetAdminPassword)
}

func (handler *RestoreHandler) Status(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, dto.Response{
		Code: http.StatusOK,
		Data: dto.RestoreStatusResp{
			Enabled: handler.restoreConfig.Enabled(),
			Mode:    handler.restoreConfig.Mode,
		},
		Message: "success",
	})
}

func (handler *RestoreHandler) ResetAdminPassword(ctx *gin.Context) {
	var req dto.RestoreAdminPasswordReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.Response{
			Code:    http.StatusBadRequest,
			Data:    nil,
			Message: "invalid request",
		})
		return
	}

	if !handler.consumeRestoreToken(req.RestoreToken) {
		handler.log.Warn().
			Str("remote_ip", ctx.ClientIP()).
			Str("username", req.Username).
			Msg("restore token rejected")
		ctx.JSON(http.StatusForbidden, dto.Response{
			Code:    http.StatusForbidden,
			Data:    nil,
			Message: "restore token is invalid",
		})
		return
	}

	if err := handler.authService.ResetPasswordByUsername(ctx.Request.Context(), req.Username, req.NewPassword); err != nil {
		handler.releaseRestoreToken()

		if errors.Is(err, service.ErrPasswordTooShort) {
			ctx.JSON(http.StatusBadRequest, dto.Response{
				Code:    http.StatusBadRequest,
				Data:    nil,
				Message: "invalid request",
			})
			return
		}

		if errors.Is(err, service.ErrAdminNotFound) {
			ctx.JSON(http.StatusNotFound, dto.Response{
				Code:    http.StatusNotFound,
				Data:    nil,
				Message: "admin not found",
			})
			return
		}

		ctx.JSON(http.StatusInternalServerError, dto.Response{
			Code:    http.StatusInternalServerError,
			Data:    nil,
			Message: "internal server error",
		})
		return
	}

	handler.log.Info().
		Str("remote_ip", ctx.ClientIP()).
		Str("username", req.Username).
		Msg("admin password restored")
	ctx.JSON(http.StatusOK, dto.Response{
		Code:    http.StatusOK,
		Data:    nil,
		Message: "success",
	})
}

func (handler *RestoreHandler) consumeRestoreToken(token string) bool {
	handler.tokenMu.Lock()
	defer handler.tokenMu.Unlock()

	if handler.tokenUsed {
		return false
	}

	if time.Now().After(handler.tokenExpiresAt) {
		return false
	}

	expectedToken := strings.TrimSpace(handler.restoreConfig.Token)
	actualToken := strings.TrimSpace(token)
	if expectedToken == "" || len(expectedToken) != len(actualToken) {
		return false
	}

	if subtle.ConstantTimeCompare([]byte(expectedToken), []byte(actualToken)) != 1 {
		return false
	}

	handler.tokenUsed = true
	return true
}

func (handler *RestoreHandler) releaseRestoreToken() {
	handler.tokenMu.Lock()
	defer handler.tokenMu.Unlock()

	handler.tokenUsed = false
}
