package handler

import (
	"encoding/json"
	"errors"
	"fmt"
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
	authenticatedGroup.POST("/notification-channels/webhook/test", handler.TestWebhookChannel)
	authenticatedGroup.PUT("/notification-channels/telegram", handler.UpsertTelegramChannel)
	authenticatedGroup.POST("/notification-channels/telegram/test", handler.TestTelegramChannel)
	authenticatedGroup.GET("/notification-proxies", handler.ListNotificationProxies)
	authenticatedGroup.POST("/notification-proxies", handler.CreateNotificationProxy)
	authenticatedGroup.PUT("/notification-proxies/:id", handler.UpdateNotificationProxy)
	authenticatedGroup.DELETE("/notification-proxies/:id", handler.DeleteNotificationProxy)
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

func (handler *AlertHandler) ListNotificationProxies(ctx *gin.Context) {
	notificationProxies, err := handler.alertService.ListNotificationProxies(ctx.Request.Context())
	if err != nil {
		internalServerError(ctx)
		return
	}

	ctx.JSON(http.StatusOK, dto.Response{Code: http.StatusOK, Data: notificationProxies, Message: "success"})
}

func (handler *AlertHandler) CreateNotificationProxy(ctx *gin.Context) {
	var req dto.UpsertNotificationProxyReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		badRequest(ctx, "invalid request")
		return
	}

	notificationProxy, err := handler.alertService.CreateNotificationProxy(ctx.Request.Context(), req)
	if err != nil {
		handleNotificationProxyError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, dto.Response{Code: http.StatusCreated, Data: notificationProxy, Message: "success"})
}

func (handler *AlertHandler) UpdateNotificationProxy(ctx *gin.Context) {
	proxyID, err := parseUintParam(ctx.Param("id"))
	if err != nil {
		badRequest(ctx, "invalid notification proxy id")
		return
	}

	var req dto.UpsertNotificationProxyReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		badRequest(ctx, "invalid request")
		return
	}

	notificationProxy, err := handler.alertService.UpdateNotificationProxy(ctx.Request.Context(), proxyID, req)
	if err != nil {
		handleNotificationProxyError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.Response{Code: http.StatusOK, Data: notificationProxy, Message: "success"})
}

func (handler *AlertHandler) DeleteNotificationProxy(ctx *gin.Context) {
	proxyID, err := parseUintParam(ctx.Param("id"))
	if err != nil {
		badRequest(ctx, "invalid notification proxy id")
		return
	}

	if err := handler.alertService.DeleteNotificationProxy(ctx.Request.Context(), proxyID); err != nil {
		handleNotificationProxyError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.Response{Code: http.StatusOK, Data: nil, Message: "success"})
}

func (handler *AlertHandler) UpsertWebhookChannel(ctx *gin.Context) {
	var req dto.UpsertWebhookChannelReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		badRequest(ctx, "invalid request")
		return
	}

	if err := handler.alertService.UpsertWebhookChannel(ctx.Request.Context(), req); err != nil {
		handleAlertServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.Response{Code: http.StatusOK, Data: nil, Message: "success"})
}

func (handler *AlertHandler) TestWebhookChannel(ctx *gin.Context) {
	var req dto.UpsertWebhookChannelReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		badRequest(ctx, "invalid request")
		return
	}

	responseExcerpt, err := handler.alertService.TestWebhookChannel(ctx.Request.Context(), req)
	if err != nil {
		handleWebhookTestError(ctx, err)
		return
	}

	var response dto.TestWebhookChannelResp
	if err := json.Unmarshal([]byte(responseExcerpt), &response); err != nil {
		ctx.JSON(http.StatusOK, dto.Response{
			Code: http.StatusOK,
			Data: dto.TestWebhookChannelResp{
				StatusCode: http.StatusOK,
				Body:       responseExcerpt,
			},
			Message: "success",
		})
		return
	}

	response.StatusCode = http.StatusOK
	ctx.JSON(http.StatusOK, dto.Response{
		Code:    http.StatusOK,
		Data:    response,
		Message: "success",
	})
}

func (handler *AlertHandler) UpsertTelegramChannel(ctx *gin.Context) {
	var req dto.UpsertTelegramChannelReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		badRequest(ctx, "invalid request")
		return
	}

	if err := handler.alertService.UpsertTelegramChannel(ctx.Request.Context(), req); err != nil {
		handleAlertServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.Response{Code: http.StatusOK, Data: nil, Message: "success"})
}

func (handler *AlertHandler) TestTelegramChannel(ctx *gin.Context) {
	var req dto.UpsertTelegramChannelReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		badRequest(ctx, "invalid request")
		return
	}

	response, err := handler.alertService.TestTelegramChannel(ctx.Request.Context(), req)
	if err != nil {
		handleTelegramTestError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.Response{Code: http.StatusOK, Data: response, Message: "success"})
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

func handleAlertServiceError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidNotificationChannel):
		ctx.JSON(http.StatusBadRequest, dto.Response{
			Code:    http.StatusBadRequest,
			Data:    nil,
			Message: "invalid notification channel config",
		})
	case errors.Is(err, service.ErrInvalidNotificationProxy):
		ctx.JSON(http.StatusBadRequest, dto.Response{
			Code:    http.StatusBadRequest,
			Data:    nil,
			Message: "invalid notification proxy config",
		})
	default:
		internalServerError(ctx)
	}
}

func handleNotificationProxyError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidNotificationProxy):
		ctx.JSON(http.StatusBadRequest, dto.Response{
			Code:    http.StatusBadRequest,
			Data:    nil,
			Message: "invalid notification proxy config",
		})
	case errors.Is(err, service.ErrNotificationProxyNotFound):
		ctx.JSON(http.StatusNotFound, dto.Response{
			Code:    http.StatusNotFound,
			Data:    nil,
			Message: "notification proxy not found",
		})
	default:
		internalServerError(ctx)
	}
}

func handleWebhookTestError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidNotificationChannel):
		ctx.JSON(http.StatusBadRequest, dto.Response{
			Code:    http.StatusBadRequest,
			Data:    nil,
			Message: "invalid notification channel config",
		})
	case errors.Is(err, service.ErrInvalidNotificationProxy):
		ctx.JSON(http.StatusBadRequest, dto.Response{
			Code:    http.StatusBadRequest,
			Data:    nil,
			Message: "invalid notification proxy config",
		})
	default:
		ctx.JSON(http.StatusBadRequest, dto.Response{
			Code:    http.StatusBadRequest,
			Data:    nil,
			Message: fmt.Sprintf("webhook test failed: %v", err),
		})
	}
}

func handleTelegramTestError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidNotificationChannel):
		ctx.JSON(http.StatusBadRequest, dto.Response{
			Code:    http.StatusBadRequest,
			Data:    nil,
			Message: "invalid notification channel config",
		})
	case errors.Is(err, service.ErrInvalidNotificationProxy):
		ctx.JSON(http.StatusBadRequest, dto.Response{
			Code:    http.StatusBadRequest,
			Data:    nil,
			Message: "invalid notification proxy config",
		})
	default:
		ctx.JSON(http.StatusBadRequest, dto.Response{
			Code:    http.StatusBadRequest,
			Data:    nil,
			Message: fmt.Sprintf("telegram test failed: %v", err),
		})
	}
}
