package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhouwu97/key-cabinet/server/internal/platform/errors"
	"github.com/zhouwu97/key-cabinet/server/internal/repository"
	"github.com/zhouwu97/key-cabinet/server/internal/service"
	"github.com/zhouwu97/key-cabinet/server/internal/transport/http/dto"
)

type OperationHandler struct {
	operationService service.OperationService
}

func NewOperationHandler(operationService service.OperationService) *OperationHandler {
	return &OperationHandler{operationService: operationService}
}

type startPickupRequest struct {
	ReservationID   string `json:"reservationId" binding:"required"`
	ClientRequestID string `json:"clientRequestId" binding:"required"`
}

type startReturnRequest struct {
	BorrowRecordID  string `json:"borrowRecordId" binding:"required"`
	DeviceID        string `json:"deviceId"`
	ClientRequestID string `json:"clientRequestId" binding:"required"`
}

func (h *OperationHandler) StartPickup(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req startPickupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.New(errors.CodeInvalidInput, "reservationId and clientRequestId are required"))
		return
	}
	var operation *repository.DeviceOperation
	var err error
	if deviceID := c.GetString("cabinet_device_id"); deviceID != "" {
		operation, err = h.operationService.StartCabinetPickup(c.Request.Context(), userID, req.ReservationID, deviceID, req.ClientRequestID)
	} else {
		operation, err = h.operationService.StartPickup(c.Request.Context(), userID, req.ReservationID, req.ClientRequestID)
	}
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusAccepted, dto.NewSuccessResponse(operation))
}

func (h *OperationHandler) StartReturn(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req startReturnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.New(errors.CodeInvalidInput, "borrowRecordId and clientRequestId are required"))
		return
	}
	if deviceID := c.GetString("cabinet_device_id"); deviceID != "" {
		if req.DeviceID != "" && req.DeviceID != deviceID {
			c.Error(errors.New(errors.CodeForbidden, "return cabinet does not match face session"))
			return
		}
		req.DeviceID = deviceID
	}
	operation, err := h.operationService.StartReturn(c.Request.Context(), userID, req.BorrowRecordID, req.DeviceID, req.ClientRequestID)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusAccepted, dto.NewSuccessResponse(operation))
}

func (h *OperationHandler) GetActive(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	operation, err := h.operationService.GetActiveOperation(c.Request.Context(), userID)
	if err != nil {
		c.Error(err)
		return
	}
	if deviceID := c.GetString("cabinet_device_id"); deviceID != "" && operation != nil && operation.DeviceID != deviceID {
		operation = nil
	}
	c.JSON(http.StatusOK, dto.NewSuccessResponse(operation))
}

func (h *OperationHandler) GetByID(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	operation, err := h.operationService.GetOperation(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	if deviceID := c.GetString("cabinet_device_id"); deviceID != "" && operation != nil && operation.DeviceID != deviceID {
		c.Error(errors.New(errors.CodeForbidden, "operation belongs to a different cabinet"))
		return
	}
	c.JSON(http.StatusOK, dto.NewSuccessResponse(operation))
}

func (h *OperationHandler) Cancel(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	if deviceID := c.GetString("cabinet_device_id"); deviceID != "" {
		operation, err := h.operationService.GetOperation(c.Request.Context(), userID, c.Param("id"))
		if err != nil {
			c.Error(err)
			return
		}
		if operation == nil || operation.DeviceID != deviceID {
			c.Error(errors.New(errors.CodeForbidden, "operation belongs to a different cabinet"))
			return
		}
	}
	if err := h.operationService.CancelOperation(c.Request.Context(), userID, c.Param("id")); err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.NewSuccessResponse(gin.H{"id": c.Param("id"), "status": "CANCELLED"}))
}
