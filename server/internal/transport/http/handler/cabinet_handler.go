package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	apperrors "github.com/zhouwu97/key-cabinet/server/internal/platform/errors"
	"github.com/zhouwu97/key-cabinet/server/internal/service"
	"github.com/zhouwu97/key-cabinet/server/internal/transport/http/dto"
)

type CabinetHandler struct {
	cabinetService service.CabinetService
}

func NewCabinetHandler(cabinetService service.CabinetService) *CabinetHandler {
	return &CabinetHandler{cabinetService: cabinetService}
}

func (h *CabinetHandler) MatchRoom(c *gin.Context) {
	var query dto.MatchRoomQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.Error(apperrors.New(apperrors.CodeInvalidInput, "roomNo is required"))
		return
	}

	results, err := h.cabinetService.MatchRoom(c.Request.Context(), query.DeviceID, query.RoomNo)
	if err != nil {
		c.Error(err)
		return
	}

	dtos := make([]dto.MatchedKeyDTO, 0, len(results))
	for _, item := range results {
		slotNo := 0
		presence := "UNKNOWN"
		if item.Slot != nil {
			slotNo = item.Slot.SlotNo
			presence = item.Slot.Presence
		}
		dtos = append(dtos, dto.MatchedKeyDTO{
			KeyID:        item.Key.ID,
			KeyName:      item.Key.Name,
			RoomNo:       item.Key.RoomNo,
			Building:     item.Key.Building,
			DeviceID:     item.Key.DeviceID,
			SlotID:       item.Key.SlotID,
			SlotNo:       slotNo,
			Status:       item.Key.Status,
			SlotPresence: presence,
			IsBorrowable: item.IsBorrowable,
		})
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(dtos))
}

func (h *CabinetHandler) DirectDispense(c *gin.Context) {
	var req dto.CabinetDirectDispenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.New(apperrors.CodeInvalidInput, "invalid direct dispense payload: "+err.Error()))
		return
	}

	op, borrow, slot, key, err := h.cabinetService.DirectDispense(c.Request.Context(), service.CabinetDirectDispenseParams{
		RequestID: req.RequestID,
		DeviceID:  req.DeviceID,
		RoomNo:    req.RoomNo,
		KeyID:     req.KeyID,
		StudentNo: req.StudentNo,
		Purpose:   req.Purpose,
	})
	if err != nil {
		c.Error(err)
		return
	}

	slotNo := 0
	if slot != nil {
		slotNo = slot.SlotNo
	}
	borrowID := ""
	if borrow != nil {
		borrowID = borrow.ID
	}
	keyID := ""
	keyName := ""
	roomNo := req.RoomNo
	if key != nil {
		keyID = key.ID
		keyName = key.Name
		if key.RoomNo != "" {
			roomNo = key.RoomNo
		}
	}

	resp := dto.CabinetDirectDispenseResponse{
		OperationID:    op.ID,
		BorrowRecordID: borrowID,
		KeyID:          keyID,
		KeyName:        keyName,
		RoomNo:         roomNo,
		DeviceID:       op.DeviceID,
		SlotID:         op.SlotID,
		SlotNo:         slotNo,
		Status:         op.Status,
	}

	c.JSON(http.StatusAccepted, dto.NewSuccessResponse(resp))
}

func (h *CabinetHandler) FaceAuth(c *gin.Context) {
	var req dto.FaceAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.New(apperrors.CodeInvalidInput, "invalid face auth payload: "+err.Error()))
		return
	}

	res, err := h.cabinetService.FaceAuth(c.Request.Context(), service.FaceAuthParams{
		DeviceID:       req.DeviceID,
		StudentNo:      req.StudentNo,
		Confidence:     req.Confidence,
		LivenessPassed: req.LivenessPassed,
	})
	if err != nil {
		c.Error(err)
		return
	}

	resp := dto.FaceAuthResponse{
		User:               res.User,
		ActiveReservations: res.ActiveReservations,
		ActiveBorrows:      res.ActiveBorrows,
		CabinetToken:       res.CabinetToken,
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(resp))
}
