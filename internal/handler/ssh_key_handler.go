package handler

import (
	"errors"
	"net/http"
	"strconv"

	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/service"

	"github.com/gin-gonic/gin"
)

type SSHKeyHandler struct {
	sshKeyService *service.SSHKeyService
}

func NewSSHKeyHandler(sshKeyService *service.SSHKeyService) *SSHKeyHandler {
	return &SSHKeyHandler{
		sshKeyService: sshKeyService,
	}
}

func (handler *SSHKeyHandler) RegisterRoutes(authenticatedGroup *gin.RouterGroup) {
	authenticatedGroup.GET("/ssh-keys", handler.List)
	authenticatedGroup.POST("/ssh-keys/import", handler.Import)
	authenticatedGroup.POST("/ssh-keys/generate", handler.Generate)
	authenticatedGroup.GET("/ssh-keys/:id/public-key", handler.GetPublicKey)
	authenticatedGroup.DELETE("/ssh-keys/:id", handler.Delete)
}

func (handler *SSHKeyHandler) List(ctx *gin.Context) {
	sshKeys, err := handler.sshKeyService.List(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.Response{
			Code:    http.StatusInternalServerError,
			Data:    nil,
			Message: "internal server error",
		})
		return
	}

	ctx.JSON(http.StatusOK, dto.Response{
		Code:    http.StatusOK,
		Data:    sshKeys,
		Message: "success",
	})
}

func (handler *SSHKeyHandler) Import(ctx *gin.Context) {
	var req dto.ImportSSHKeyReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.Response{
			Code:    http.StatusBadRequest,
			Data:    nil,
			Message: "invalid request",
		})
		return
	}

	sshKey, err := handler.sshKeyService.Import(ctx.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrDuplicateSSHKeyFingerprint) {
			ctx.JSON(http.StatusConflict, dto.Response{
				Code:    http.StatusConflict,
				Data:    nil,
				Message: "ssh key fingerprint already exists",
			})
			return
		}

		ctx.JSON(http.StatusBadRequest, dto.Response{
			Code:    http.StatusBadRequest,
			Data:    nil,
			Message: "invalid private key",
		})
		return
	}

	ctx.JSON(http.StatusCreated, dto.Response{
		Code:    http.StatusCreated,
		Data:    sshKey,
		Message: "success",
	})
}

func (handler *SSHKeyHandler) Generate(ctx *gin.Context) {
	var req dto.GenerateSSHKeyReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.Response{
			Code:    http.StatusBadRequest,
			Data:    nil,
			Message: "invalid request",
		})
		return
	}

	sshKey, err := handler.sshKeyService.Generate(ctx.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrDuplicateSSHKeyFingerprint) {
			ctx.JSON(http.StatusConflict, dto.Response{
				Code:    http.StatusConflict,
				Data:    nil,
				Message: "ssh key fingerprint already exists",
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

	ctx.JSON(http.StatusCreated, dto.Response{
		Code:    http.StatusCreated,
		Data:    sshKey,
		Message: "success",
	})
}

func (handler *SSHKeyHandler) GetPublicKey(ctx *gin.Context) {
	sshKeyID, err := parseUintParam(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, dto.Response{
			Code:    http.StatusBadRequest,
			Data:    nil,
			Message: "invalid ssh key id",
		})
		return
	}

	sshKey, err := handler.sshKeyService.GetPublicKey(ctx.Request.Context(), sshKeyID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, dto.Response{
			Code:    http.StatusNotFound,
			Data:    nil,
			Message: "ssh key not found",
		})
		return
	}

	ctx.JSON(http.StatusOK, dto.Response{
		Code:    http.StatusOK,
		Data:    sshKey,
		Message: "success",
	})
}

func (handler *SSHKeyHandler) Delete(ctx *gin.Context) {
	sshKeyID, err := parseUintParam(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, dto.Response{
			Code:    http.StatusBadRequest,
			Data:    nil,
			Message: "invalid ssh key id",
		})
		return
	}

	if err := handler.sshKeyService.Delete(ctx.Request.Context(), sshKeyID); err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.Response{
			Code:    http.StatusInternalServerError,
			Data:    nil,
			Message: "internal server error",
		})
		return
	}

	ctx.JSON(http.StatusOK, dto.Response{
		Code:    http.StatusOK,
		Data:    nil,
		Message: "success",
	})
}

func parseUintParam(raw string) (uint, error) {
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, err
	}

	return uint(value), nil
}
