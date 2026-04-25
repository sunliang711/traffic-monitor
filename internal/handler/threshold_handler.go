package handler

import (
	"errors"
	"net/http"

	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/service"

	"github.com/gin-gonic/gin"
)

type ThresholdHandler struct {
	thresholdService *service.ThresholdService
}

func NewThresholdHandler(thresholdService *service.ThresholdService) *ThresholdHandler {
	return &ThresholdHandler{
		thresholdService: thresholdService,
	}
}

func (handler *ThresholdHandler) RegisterRoutes(apiGroup *gin.RouterGroup, authenticatedGroup *gin.RouterGroup) {
	apiGroup.GET("/thresholds/global", handler.ListGlobalRules)
	authenticatedGroup.PUT("/thresholds/global", handler.UpsertGlobalRules)
	authenticatedGroup.GET("/machines/:id/thresholds", handler.ListMachineRules)
	authenticatedGroup.PUT("/machines/:id/thresholds", handler.UpsertMachineRules)
}

func (handler *ThresholdHandler) ListGlobalRules(ctx *gin.Context) {
	rules, err := handler.thresholdService.ListGlobalRules(ctx.Request.Context())
	if err != nil {
		internalServerError(ctx)
		return
	}

	ctx.JSON(http.StatusOK, dto.Response{Code: http.StatusOK, Data: rules, Message: "success"})
}

func (handler *ThresholdHandler) UpsertGlobalRules(ctx *gin.Context) {
	var req dto.UpsertThresholdRulesReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		badRequest(ctx, "invalid request")
		return
	}

	if err := handler.thresholdService.UpsertGlobalRules(ctx.Request.Context(), req); err != nil {
		handleThresholdError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.Response{Code: http.StatusOK, Data: nil, Message: "success"})
}

func (handler *ThresholdHandler) ListMachineRules(ctx *gin.Context) {
	machineID, err := parseMachineID(ctx.Param("id"))
	if err != nil {
		badRequest(ctx, "invalid machine id")
		return
	}

	rules, err := handler.thresholdService.ListEffectiveMachineRules(ctx.Request.Context(), machineID)
	if err != nil {
		handleThresholdError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.Response{Code: http.StatusOK, Data: rules, Message: "success"})
}

func (handler *ThresholdHandler) UpsertMachineRules(ctx *gin.Context) {
	machineID, err := parseMachineID(ctx.Param("id"))
	if err != nil {
		badRequest(ctx, "invalid machine id")
		return
	}

	var req dto.UpsertThresholdRulesReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		badRequest(ctx, "invalid request")
		return
	}

	if err := handler.thresholdService.UpsertMachineRules(ctx.Request.Context(), machineID, req); err != nil {
		handleThresholdError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.Response{Code: http.StatusOK, Data: nil, Message: "success"})
}

func handleThresholdError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidThresholdRule):
		badRequest(ctx, "invalid threshold rule")
	case errors.Is(err, service.ErrMachineNotFound):
		ctx.JSON(http.StatusNotFound, dto.Response{Code: http.StatusNotFound, Data: nil, Message: "machine not found"})
	default:
		internalServerError(ctx)
	}
}
