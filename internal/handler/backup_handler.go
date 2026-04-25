package handler

import (
	"errors"
	"net/http"

	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/service"

	"github.com/gin-gonic/gin"
)

type BackupHandler struct {
	backupService *service.BackupService
}

func NewBackupHandler(backupService *service.BackupService) *BackupHandler {
	return &BackupHandler{backupService: backupService}
}

func (handler *BackupHandler) RegisterRoutes(authenticatedGroup *gin.RouterGroup) {
	authenticatedGroup.POST("/backups/export", handler.Export)
	authenticatedGroup.POST("/backups/import", handler.Import)
}

func (handler *BackupHandler) Export(ctx *gin.Context) {
	var req dto.BackupExportReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		badRequest(ctx, "invalid request")
		return
	}

	backup, err := handler.backupService.Export(ctx.Request.Context(), req)
	if err != nil {
		handleBackupServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.Response{
		Code:    http.StatusOK,
		Data:    backup,
		Message: "success",
	})
}

func (handler *BackupHandler) Import(ctx *gin.Context) {
	var req dto.BackupImportReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		badRequest(ctx, "invalid request")
		return
	}

	response, err := handler.backupService.Import(ctx.Request.Context(), req)
	if err != nil {
		handleBackupServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.Response{
		Code:    http.StatusOK,
		Data:    response,
		Message: "success",
	})
}

func handleBackupServiceError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidBackupRequest):
		ctx.JSON(http.StatusBadRequest, dto.Response{Code: http.StatusBadRequest, Data: nil, Message: "invalid backup request"})
	case errors.Is(err, service.ErrInvalidBackupPayload):
		ctx.JSON(http.StatusBadRequest, dto.Response{Code: http.StatusBadRequest, Data: nil, Message: "invalid backup payload"})
	case errors.Is(err, service.ErrBackupDecryptFailed):
		ctx.JSON(http.StatusBadRequest, dto.Response{Code: http.StatusBadRequest, Data: nil, Message: "backup decrypt failed"})
	case errors.Is(err, service.ErrBackupSSHKeyDecryptFail):
		ctx.JSON(http.StatusConflict, dto.Response{
			Code:    http.StatusConflict,
			Data:    nil,
			Message: "SSH 私钥无法解密，当前 APP_MASTER_KEY 可能与导入该密钥时使用的不一致，请恢复旧密钥或重新导入 SSH Key",
		})
	default:
		internalServerError(ctx)
	}
}
