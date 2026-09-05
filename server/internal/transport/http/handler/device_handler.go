package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhouwu97/key-cabinet/server/internal/service"
	"github.com/zhouwu97/key-cabinet/server/internal/transport/http/dto"
)

type DeviceHandler struct {
	deviceService service.DeviceService
}

func NewDeviceHandler(deviceService service.DeviceService) *DeviceHandler {
	return &DeviceHandler{deviceService: deviceService}
}

func (h *DeviceHandler) List(c *gin.Context) {
	devices, err := h.deviceService.ListDevices(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.NewSuccessResponse(devices))
}

func (h *DeviceHandler) GetByID(c *gin.Context) {
	device, err := h.deviceService.GetDevice(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.NewSuccessResponse(device))
}

func (h *DeviceHandler) GetStatus(c *gin.Context) {
	device, err := h.deviceService.GetDeviceStatus(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.NewSuccessResponse(device))
}

func (h *DeviceHandler) GetSlots(c *gin.Context) {
	slots, err := h.deviceService.GetDeviceSlots(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.NewSuccessResponse(slots))
}
