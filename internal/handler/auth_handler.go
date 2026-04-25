package handler

import (
	"errors"
	"net/http"

	"traffic-monitor/internal/config"
	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/middleware"
	"traffic-monitor/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

type AuthHandler struct {
	authService   *service.AuthService
	sessionStore  *sessions.CookieStore
	sessionConfig config.SessionConfig
}

func NewAuthHandler(authService *service.AuthService, sessionStore *sessions.CookieStore, sessionConfig config.SessionConfig) *AuthHandler {
	return &AuthHandler{
		authService:   authService,
		sessionStore:  sessionStore,
		sessionConfig: sessionConfig,
	}
}

func RegisterRoutes(engine *gin.Engine, healthHandler *HealthHandler, authHandler *AuthHandler, authMiddleware *middleware.AuthMiddleware, sshKeyHandler *SSHKeyHandler, machineHandler *MachineHandler, backupHandler *BackupHandler, thresholdHandler *ThresholdHandler, trafficSampleHandler *TrafficSampleHandler, alertHandler *AlertHandler) {
	engine.GET("/healthz", healthHandler.GetHealth)

	apiGroup := engine.Group("/api/v1")
	apiGroup.GET("/health", healthHandler.GetHealth)

	authGroup := apiGroup.Group("/auth")
	authGroup.POST("/login", authHandler.Login)
	authGroup.POST("/logout", authHandler.Logout)
	authGroup.GET("/profile", authMiddleware.RequireAdmin(), authHandler.Profile)

	authenticatedGroup := apiGroup.Group("")
	authenticatedGroup.Use(authMiddleware.RequireAdmin())
	sshKeyHandler.RegisterRoutes(authenticatedGroup)
	machineHandler.RegisterRoutes(authenticatedGroup)
	backupHandler.RegisterRoutes(authenticatedGroup)
	thresholdHandler.RegisterRoutes(apiGroup, authenticatedGroup)
	trafficSampleHandler.RegisterRoutes(authenticatedGroup)
	alertHandler.RegisterRoutes(authenticatedGroup)
}

func (handler *AuthHandler) Login(ctx *gin.Context) {
	var req dto.LoginReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.Response{
			Code:    http.StatusBadRequest,
			Data:    nil,
			Message: "invalid request",
		})
		return
	}

	profile, err := handler.authService.Authenticate(ctx.Request.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			ctx.JSON(http.StatusUnauthorized, dto.Response{
				Code:    http.StatusUnauthorized,
				Data:    nil,
				Message: "authentication failed",
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

	session, err := handler.sessionStore.Get(ctx.Request, handler.sessionConfig.CookieName)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.Response{
			Code:    http.StatusInternalServerError,
			Data:    nil,
			Message: "internal server error",
		})
		return
	}

	session.Values[middleware.SessionAdminKey()] = profile.ID
	if err := session.Save(ctx.Request, ctx.Writer); err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.Response{
			Code:    http.StatusInternalServerError,
			Data:    nil,
			Message: "internal server error",
		})
		return
	}

	ctx.JSON(http.StatusOK, dto.Response{
		Code:    http.StatusOK,
		Data:    profile,
		Message: "success",
	})
}

func (handler *AuthHandler) Logout(ctx *gin.Context) {
	session, err := handler.sessionStore.Get(ctx.Request, handler.sessionConfig.CookieName)
	if err == nil {
		session.Options.MaxAge = -1
		_ = session.Save(ctx.Request, ctx.Writer)
	}

	ctx.JSON(http.StatusOK, dto.Response{
		Code:    http.StatusOK,
		Data:    nil,
		Message: "success",
	})
}

func (handler *AuthHandler) Profile(ctx *gin.Context) {
	adminID, ok := middleware.CurrentAdminID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, dto.Response{
			Code:    http.StatusUnauthorized,
			Data:    nil,
			Message: "unauthorized",
		})
		return
	}

	profile, err := handler.authService.GetProfile(ctx.Request.Context(), adminID)
	if err != nil {
		if errors.Is(err, service.ErrAdminNotFound) {
			ctx.JSON(http.StatusUnauthorized, dto.Response{
				Code:    http.StatusUnauthorized,
				Data:    nil,
				Message: "unauthorized",
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

	ctx.JSON(http.StatusOK, dto.Response{
		Code:    http.StatusOK,
		Data:    profile,
		Message: "success",
	})
}
