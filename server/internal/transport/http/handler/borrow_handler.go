package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zhouwu97/key-cabinet/server/internal/platform/errors"
	"github.com/zhouwu97/key-cabinet/server/internal/service"
	"github.com/zhouwu97/key-cabinet/server/internal/transport/http/dto"
)

type createBorrowRequest struct {
	KeyID            string    `json:"keyId" binding:"required"`
	DeviceID         string    `json:"deviceId" binding:"required"`
	SlotID           string    `json:"slotId" binding:"required"`
	ReservationID    string    `json:"reservationId"`
	Purpose          string    `json:"purpose"`
	ExpectedReturnAt time.Time `json:"expectedReturnAt" binding:"required"`
}
type updateBorrowStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

func (h *BorrowHandler) Create(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req createBorrowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.New(errors.CodeInvalidInput, "invalid borrow request"))
		return
	}
	record, err := h.borrowService.CreateBorrowing(c.Request.Context(), userID, req.KeyID, req.DeviceID, req.SlotID, req.ReservationID, req.Purpose, req.ExpectedReturnAt)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, dto.NewSuccessResponse(record))
}

func (h *BorrowHandler) UpdateStatus(c *gin.Context) {
	var req updateBorrowStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.New(errors.CodeInvalidInput, "status is required"))
		return
	}
	switch req.Status {
	case "RETURNING":
		userID, ok := currentUserID(c)
		if !ok {
			return
		}
		record, err := h.borrowService.BeginReturn(c.Request.Context(), userID, c.Param("id"))
		if err != nil {
			c.Error(err)
			return
		}
		c.JSON(http.StatusOK, dto.NewSuccessResponse(record))
	case "BORROWED":
		if err := h.borrowService.MarkBorrowed(c.Request.Context(), c.Param("id"), time.Now().UTC()); err != nil {
			c.Error(err)
			return
		}
		c.JSON(http.StatusOK, dto.NewSuccessResponse(map[string]bool{"updated": true}))
	default:
		c.Error(errors.New(errors.CodeInvalidInput, "unsupported borrow status"))
	}
}

func (h *BorrowHandler) Complete(c *gin.Context) {
	if err := h.borrowService.CompleteReturn(c.Request.Context(), c.Param("id"), time.Now().UTC()); err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.NewSuccessResponse(map[string]bool{"completed": true}))
}

func (h *BorrowHandler) CheckOverdue(c *gin.Context) {
	if err := h.borrowService.CheckOverdue(c.Request.Context(), time.Now().UTC()); err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.NewSuccessResponse(map[string]bool{"updated": true}))
}

type BorrowHandler struct {
	borrowService service.BorrowService
}

func NewBorrowHandler(borrowService service.BorrowService) *BorrowHandler {
	return &BorrowHandler{borrowService: borrowService}
}

func (h *BorrowHandler) ListMine(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	records, err := h.borrowService.ListUserBorrowRecords(c.Request.Context(), userID, c.Query("keyId"), c.Query("status"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.NewSuccessResponse(records))
}

func (h *BorrowHandler) GetMine(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	record, err := h.borrowService.GetUserBorrowRecord(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.NewSuccessResponse(record))
}
