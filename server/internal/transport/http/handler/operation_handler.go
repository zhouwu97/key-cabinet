package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhouwu97/key-cabinet/server/internal/platform/errors"
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
	operation, err := h.operationService.StartPickup(c.Request.Context(), userID, req.ReservationID, req.ClientRequestID)
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
	c.JSON(http.StatusOK, dto.NewSuccessResponse(operation))
}

func (h *OperationHandler) Cancel(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	if err := h.operationService.CancelOperation(c.Request.Context(), userID, c.Param("id")); err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.NewSuccessResponse(gin.H{"id": c.Param("id"), "status": "CANCELLED"}))
}
