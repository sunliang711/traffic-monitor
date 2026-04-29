package handler

import (
	"net/http"

	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/service"

	"github.com/gin-gonic/gin"
)

type SettingsHandler struct {
	settingsService *service.SettingsService
}

func NewSettingsHandler(settingsService *service.SettingsService) *SettingsHandler {
	return &SettingsHandler{
		settingsService: settingsService,
	}
}

func (handler *SettingsHandler) RegisterRoutes(authenticatedGroup *gin.RouterGroup) {
	settingsGroup := authenticatedGroup.Group("/settings")
	settingsGroup.GET("/guest-mode", handler.GetGuestMode)
	settingsGroup.PUT("/guest-mode", handler.UpdateGuestMode)
}

func (handler *SettingsHandler) GetGuestMode(ctx *gin.Context) {
	response, err := handler.settingsService.GetGuestMode(ctx.Request.Context())
	if err != nil {
		internalServerError(ctx)
		return
	}

	ctx.JSON(http.StatusOK, dto.Response{Code: http.StatusOK, Data: response, Message: "success"})
}

func (handler *SettingsHandler) UpdateGuestMode(ctx *gin.Context) {
	var req dto.UpdateGuestModeReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		badRequest(ctx, "invalid request")
		return
	}

	response, err := handler.settingsService.UpdateGuestMode(ctx.Request.Context(), req)
	if err != nil {
		internalServerError(ctx)
		return
	}

	ctx.JSON(http.StatusOK, dto.Response{Code: http.StatusOK, Data: response, Message: "success"})
}
