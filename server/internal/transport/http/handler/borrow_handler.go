package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhouwu97/key-cabinet/server/internal/service"
	"github.com/zhouwu97/key-cabinet/server/internal/transport/http/dto"
)

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
