package handler

import (
	"net/http"

	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/service"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	healthService *service.HealthService
}

func NewHealthHandler(healthService *service.HealthService) *HealthHandler {
	return &HealthHandler{
		healthService: healthService,
	}
}

func RegisterRoutes(engine *gin.Engine, healthHandler *HealthHandler) {
	engine.GET("/healthz", healthHandler.GetHealth)

	apiGroup := engine.Group("/api/v1")
	apiGroup.GET("/health", healthHandler.GetHealth)
}

func (handler *HealthHandler) GetHealth(ctx *gin.Context) {
	response, err := handler.healthService.Check(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusServiceUnavailable, dto.Response{
			Code:    http.StatusServiceUnavailable,
			Data:    response,
			Message: "database unavailable",
		})
		return
	}

	ctx.JSON(http.StatusOK, dto.Response{
		Code:    http.StatusOK,
		Data:    response,
		Message: "success",
	})
}
