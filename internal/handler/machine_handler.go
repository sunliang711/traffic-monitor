package handler

import (
	"errors"
	"net/http"
	"strconv"

	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type MachineHandler struct {
	machineService *service.MachineService
	log            zerolog.Logger
}

func NewMachineHandler(machineService *service.MachineService, log zerolog.Logger) *MachineHandler {
	return &MachineHandler{
		machineService: machineService,
		log:            log,
	}
}

func (handler *MachineHandler) RegisterRoutes(readGroup *gin.RouterGroup, authenticatedGroup *gin.RouterGroup) {
	readGroup.GET("/machines", handler.List)
	readGroup.GET("/machines/:id", handler.Get)
	authenticatedGroup.POST("/machines", handler.Create)
	authenticatedGroup.PATCH("/machines/:id", handler.Update)
	authenticatedGroup.DELETE("/machines/:id", handler.Delete)
	authenticatedGroup.POST("/machines/:id/test-connection", handler.TestConnection)
}

func (handler *MachineHandler) List(ctx *gin.Context) {
	machines, err := handler.machineService.List(ctx.Request.Context())
	if err != nil {
		internalServerError(ctx)
		return
	}

	ctx.JSON(http.StatusOK, dto.Response{Code: http.StatusOK, Data: machines, Message: "success"})
}

func (handler *MachineHandler) Create(ctx *gin.Context) {
	var req dto.CreateMachineReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		badRequest(ctx, "invalid request")
		return
	}

	machine, err := handler.machineService.Create(ctx.Request.Context(), req)
	if err != nil {
		handleMachineServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, dto.Response{Code: http.StatusCreated, Data: machine, Message: "success"})
}

func (handler *MachineHandler) Get(ctx *gin.Context) {
	machineID, err := parseMachineID(ctx.Param("id"))
	if err != nil {
		badRequest(ctx, "invalid machine id")
		return
	}

	machine, err := handler.machineService.Get(ctx.Request.Context(), machineID)
	if err != nil {
		handleMachineServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.Response{Code: http.StatusOK, Data: machine, Message: "success"})
}

func (handler *MachineHandler) Update(ctx *gin.Context) {
	machineID, err := parseMachineID(ctx.Param("id"))
	if err != nil {
		badRequest(ctx, "invalid machine id")
		return
	}

	var req dto.UpdateMachineReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		badRequest(ctx, "invalid request")
		return
	}

	machine, err := handler.machineService.Update(ctx.Request.Context(), machineID, req)
	if err != nil {
		handleMachineServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.Response{Code: http.StatusOK, Data: machine, Message: "success"})
}

func (handler *MachineHandler) Delete(ctx *gin.Context) {
	machineID, err := parseMachineID(ctx.Param("id"))
	if err != nil {
		badRequest(ctx, "invalid machine id")
		return
	}

	if err := handler.machineService.Delete(ctx.Request.Context(), machineID); err != nil {
		handleMachineServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.Response{Code: http.StatusOK, Data: nil, Message: "success"})
}

func (handler *MachineHandler) TestConnection(ctx *gin.Context) {
	machineID, err := parseMachineID(ctx.Param("id"))
	if err != nil {
		badRequest(ctx, "invalid machine id")
		return
	}

	result, err := handler.machineService.TestConnection(ctx.Request.Context(), machineID)
	if err != nil {
		if errors.Is(err, service.ErrSSHKeyDecryptFailed) {
			handler.log.Error().
				Err(err).
				Uint("machine_id", machineID).
				Msg("machine connection test failed: stored ssh key cannot be decrypted with current APP_MASTER_KEY")
			ctx.JSON(http.StatusConflict, dto.Response{
				Code:    http.StatusConflict,
				Data:    result,
				Message: "SSH 私钥无法解密，当前 APP_MASTER_KEY 可能与导入该密钥时使用的不一致，请恢复旧密钥或重新导入 SSH Key",
			})
			return
		}

		handler.log.Error().Err(err).Uint("machine_id", machineID).Msg("machine connection test failed")
		if errors.Is(err, service.ErrVNStatUnavailable) {
			message := "vnstat unavailable"
			switch result.Status {
			case "vnstat_not_installed":
				message = "vnstat not installed"
			case "vnstat_interface_missing":
				message = "vnstat interface not found"
			}

			ctx.JSON(http.StatusBadRequest, dto.Response{
				Code:    http.StatusBadRequest,
				Data:    result,
				Message: message,
			})
			return
		}

		handleMachineServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.Response{Code: http.StatusOK, Data: result, Message: "success"})
}

func parseMachineID(raw string) (uint, error) {
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, err
	}

	return uint(value), nil
}

func handleMachineServiceError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrMachineNotFound):
		ctx.JSON(http.StatusNotFound, dto.Response{Code: http.StatusNotFound, Data: nil, Message: "machine not found"})
	case errors.Is(err, service.ErrSSHKeyNotFound):
		ctx.JSON(http.StatusBadRequest, dto.Response{Code: http.StatusBadRequest, Data: nil, Message: "ssh key not found"})
	case errors.Is(err, service.ErrInvalidMachineConfig):
		ctx.JSON(http.StatusBadRequest, dto.Response{Code: http.StatusBadRequest, Data: nil, Message: "invalid machine config"})
	case errors.Is(err, service.ErrSSHKeyDecryptFailed):
		ctx.JSON(http.StatusConflict, dto.Response{
			Code:    http.StatusConflict,
			Data:    nil,
			Message: "SSH 私钥无法解密，当前 APP_MASTER_KEY 可能与导入该密钥时使用的不一致，请恢复旧密钥或重新导入 SSH Key",
		})
	default:
		internalServerError(ctx)
	}
}

func badRequest(ctx *gin.Context, message string) {
	ctx.JSON(http.StatusBadRequest, dto.Response{Code: http.StatusBadRequest, Data: nil, Message: message})
}

func internalServerError(ctx *gin.Context) {
	ctx.JSON(http.StatusInternalServerError, dto.Response{Code: http.StatusInternalServerError, Data: nil, Message: "internal server error"})
}
