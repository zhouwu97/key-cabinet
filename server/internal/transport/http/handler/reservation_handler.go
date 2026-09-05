package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zhouwu97/key-cabinet/server/internal/platform/errors"
	"github.com/zhouwu97/key-cabinet/server/internal/service"
	"github.com/zhouwu97/key-cabinet/server/internal/transport/http/dto"
)

type ReservationHandler struct {
	reservationService service.ReservationService
}

func NewReservationHandler(reservationService service.ReservationService) *ReservationHandler {
	return &ReservationHandler{reservationService: reservationService}
}

type createReservationRequest struct {
	KeyID             string    `json:"keyId" binding:"required"`
	PickupWindowStart time.Time `json:"pickupWindowStart"`
	PickupWindowEnd   time.Time `json:"pickupWindowEnd"`
	ExpectedReturnAt  time.Time `json:"expectedReturnAt"`
	ExpectedDuration  int64     `json:"expectedDuration"` // 小程序传毫秒，避免直接映射 time.Duration 的纳秒语义。
	Purpose           string    `json:"purpose"`
}

func (h *ReservationHandler) Create(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req createReservationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.New(errors.CodeInvalidInput, "invalid reservation request"))
		return
	}

	reservation, err := h.reservationService.CreateReservation(c.Request.Context(), userID, service.CreateReservationRequest{
		KeyID:             req.KeyID,
		PickupWindowStart: req.PickupWindowStart,
		PickupWindowEnd:   req.PickupWindowEnd,
		ExpectedReturnAt:  req.ExpectedReturnAt,
		ExpectedDuration:  time.Duration(req.ExpectedDuration) * time.Millisecond,
		Purpose:           req.Purpose,
	})
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, dto.NewSuccessResponse(reservation))
}

func (h *ReservationHandler) ListMine(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	reservations, err := h.reservationService.ListUserReservations(c.Request.Context(), userID, c.Query("status"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.NewSuccessResponse(reservations))
}

func (h *ReservationHandler) GetMine(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	reservation, err := h.reservationService.GetUserReservation(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.NewSuccessResponse(reservation))
}

func (h *ReservationHandler) Cancel(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	reservation, err := h.reservationService.CancelReservation(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.NewSuccessResponse(reservation))
}

func (h *ReservationHandler) Availability(c *gin.Context) {
	start, err := parseTimeQuery(c.Query("pickupWindowStart"))
	if err != nil {
		c.Error(err)
		return
	}
	end, err := parseTimeQuery(c.Query("pickupWindowEnd"))
	if err != nil {
		c.Error(err)
		return
	}
	available, err := h.reservationService.CanReserveKey(c.Request.Context(), c.Param("id"), start, end)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.NewSuccessResponse(gin.H{"available": available}))
}

func parseTimeQuery(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, errors.New(errors.CodeInvalidInput, "time query must be RFC3339")
	}
	return parsed, nil
}
