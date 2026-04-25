package handler

import (
	"net/http"
	"strconv"

	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/service"

	"github.com/gin-gonic/gin"
)

type AlertHandler struct {
	alertService *service.AlertService
}

func NewAlertHandler(alertService *service.AlertService) *AlertHandler {
	return &AlertHandler{
		alertService: alertService,
	}
}

func (handler *AlertHandler) RegisterRoutes(authenticatedGroup *gin.RouterGroup) {
	authenticatedGroup.GET("/alerts", handler.ListAlerts)
	authenticatedGroup.GET("/notification-channels", handler.ListNotificationChannels)
	authenticatedGroup.PUT("/notification-channels/webhook", handler.UpsertWebhookChannel)
	authenticatedGroup.PUT("/notification-channels/telegram", handler.UpsertTelegramChannel)
}

func (handler *AlertHandler) ListAlerts(ctx *gin.Context) {
	query, err := parseListAlertsQuery(ctx)
	if err != nil {
		badRequest(ctx, "invalid query")
		return
	}

	response, err := handler.alertService.ListAlerts(ctx.Request.Context(), query)
	if err != nil {
		internalServerError(ctx)
		return
	}

	ctx.JSON(http.StatusOK, dto.Response{Code: http.StatusOK, Data: response, Message: "success"})
}

func (handler *AlertHandler) ListNotificationChannels(ctx *gin.Context) {
	channels, err := handler.alertService.ListNotificationChannels(ctx.Request.Context())
	if err != nil {
		internalServerError(ctx)
		return
	}

	ctx.JSON(http.StatusOK, dto.Response{Code: http.StatusOK, Data: channels, Message: "success"})
}

func (handler *AlertHandler) UpsertWebhookChannel(ctx *gin.Context) {
	var req dto.UpsertWebhookChannelReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		badRequest(ctx, "invalid request")
		return
	}

	if err := handler.alertService.UpsertWebhookChannel(ctx.Request.Context(), req); err != nil {
		internalServerError(ctx)
		return
	}

	ctx.JSON(http.StatusOK, dto.Response{Code: http.StatusOK, Data: nil, Message: "success"})
}

func (handler *AlertHandler) UpsertTelegramChannel(ctx *gin.Context) {
	var req dto.UpsertTelegramChannelReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		badRequest(ctx, "invalid request")
		return
	}

	if err := handler.alertService.UpsertTelegramChannel(ctx.Request.Context(), req); err != nil {
		internalServerError(ctx)
		return
	}

	ctx.JSON(http.StatusOK, dto.Response{Code: http.StatusOK, Data: nil, Message: "success"})
}

func parseListAlertsQuery(ctx *gin.Context) (dto.ListAlertsQuery, error) {
	var query dto.ListAlertsQuery
	if rawMachineID := ctx.Query("machine_id"); rawMachineID != "" {
		value, err := strconv.ParseUint(rawMachineID, 10, 64)
		if err != nil {
			return dto.ListAlertsQuery{}, err
		}

		machineID := uint(value)
		query.MachineID = &machineID
	}

	if rawPage := ctx.Query("page"); rawPage != "" {
		value, err := strconv.Atoi(rawPage)
		if err != nil {
			return dto.ListAlertsQuery{}, err
		}

		query.Page = value
	}

	if rawPageSize := ctx.Query("page_size"); rawPageSize != "" {
		value, err := strconv.Atoi(rawPageSize)
		if err != nil {
			return dto.ListAlertsQuery{}, err
		}

		query.PageSize = value
	}

	query.PeriodType = ctx.Query("period_type")
	return query, nil
}
