package handler

import (
	"errors"
	"net/http"
	"strconv"

	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/service"

	"github.com/gin-gonic/gin"
)

type TrafficSampleHandler struct {
	trafficCollectionService *service.TrafficCollectionService
}

func NewTrafficSampleHandler(trafficCollectionService *service.TrafficCollectionService) *TrafficSampleHandler {
	return &TrafficSampleHandler{
		trafficCollectionService: trafficCollectionService,
	}
}

func (handler *TrafficSampleHandler) RegisterRoutes(authenticatedGroup *gin.RouterGroup) {
	authenticatedGroup.GET("/traffic-samples", handler.ListSamples)
	authenticatedGroup.POST("/system/collect-now", handler.CollectNow)
}

func (handler *TrafficSampleHandler) ListSamples(ctx *gin.Context) {
	query, err := parseListTrafficSamplesQuery(ctx)
	if err != nil {
		badRequest(ctx, "invalid query")
		return
	}

	response, err := handler.trafficCollectionService.ListSamples(ctx.Request.Context(), query)
	if err != nil {
		internalServerError(ctx)
		return
	}

	ctx.JSON(http.StatusOK, dto.Response{
		Code:    http.StatusOK,
		Data:    response,
		Message: "success",
	})
}

func (handler *TrafficSampleHandler) CollectNow(ctx *gin.Context) {
	var req dto.CollectNowReq
	if err := ctx.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		badRequest(ctx, "invalid request")
		return
	}

	response, err := handler.trafficCollectionService.CollectNow(ctx.Request.Context(), req.MachineID)
	if err != nil {
		handleTrafficCollectionError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.Response{
		Code:    http.StatusOK,
		Data:    response,
		Message: "success",
	})
}

func parseListTrafficSamplesQuery(ctx *gin.Context) (dto.ListTrafficSamplesQuery, error) {
	var query dto.ListTrafficSamplesQuery
	if rawMachineID := ctx.Query("machine_id"); rawMachineID != "" {
		value, err := strconv.ParseUint(rawMachineID, 10, 64)
		if err != nil {
			return dto.ListTrafficSamplesQuery{}, err
		}

		machineID := uint(value)
		query.MachineID = &machineID
	}

	if rawPage := ctx.Query("page"); rawPage != "" {
		value, err := strconv.Atoi(rawPage)
		if err != nil {
			return dto.ListTrafficSamplesQuery{}, err
		}

		query.Page = value
	}

	if rawPageSize := ctx.Query("page_size"); rawPageSize != "" {
		value, err := strconv.Atoi(rawPageSize)
		if err != nil {
			return dto.ListTrafficSamplesQuery{}, err
		}

		query.PageSize = value
	}

	query.PeriodType = ctx.Query("period_type")
	return query, nil
}

func handleTrafficCollectionError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrMachineNotFound):
		ctx.JSON(http.StatusNotFound, dto.Response{Code: http.StatusNotFound, Data: nil, Message: "machine not found"})
	case errors.Is(err, service.ErrSSHKeyNotFound):
		ctx.JSON(http.StatusBadRequest, dto.Response{Code: http.StatusBadRequest, Data: nil, Message: "ssh key not found"})
	default:
		internalServerError(ctx)
	}
}
