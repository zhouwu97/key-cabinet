package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zhouwu97/key-cabinet/server/internal/platform/errors"
	"github.com/zhouwu97/key-cabinet/server/internal/service"
	"github.com/zhouwu97/key-cabinet/server/internal/transport/http/dto"
)

type KeyHandler struct {
	keyService service.KeyService
}

func NewKeyHandler(keyService service.KeyService) *KeyHandler {
	return &KeyHandler{keyService: keyService}
}

func (h *KeyHandler) List(c *gin.Context) {
	enabled, err := optionalBool(c.Query("enabled"))
	if err != nil {
		c.Error(err)
		return
	}

	keys, err := h.keyService.ListKeys(c.Request.Context(), service.KeyListFilter{
		Keyword:  c.Query("keyword"),
		DeviceID: c.Query("deviceId"),
		Status:   strings.ToUpper(strings.TrimSpace(c.Query("status"))),
		Enabled:  enabled,
	})
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(keys))
}

func (h *KeyHandler) GetByID(c *gin.Context) {
	key, err := h.keyService.GetKey(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.NewSuccessResponse(key))
}

func (h *KeyHandler) GetSlot(c *gin.Context) {
	slot, err := h.keyService.GetKeySlot(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.NewSuccessResponse(slot))
}

func optionalBool(value string) (*bool, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return nil, nil
	}
	if value != "true" && value != "false" {
		return nil, errors.New(errors.CodeInvalidInput, "enabled must be true or false")
	}
	parsed := value == "true"
	return &parsed, nil
}
