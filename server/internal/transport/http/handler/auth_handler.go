package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhouwu97/key-cabinet/server/internal/platform/errors"
	"github.com/zhouwu97/key-cabinet/server/internal/service"
	"github.com/zhouwu97/key-cabinet/server/internal/transport/http/dto"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) WechatLogin(c *gin.Context) {
	var req dto.WechatLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.New(errors.CodeInvalidInput, "invalid request body: code is required"))
		return
	}

	result, err := h.authService.WechatLogin(c.Request.Context(), req.Code)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(result))
}

func (h *AuthHandler) GetMe(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.Error(errors.New(errors.CodeUnauthorized, "missing user identity in context"))
		return
	}

	user, err := h.authService.GetMe(c.Request.Context(), userID.(string))
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(user))
}

func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.Error(errors.New(errors.CodeUnauthorized, "missing user identity in context"))
		return
	}

	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.New(errors.CodeInvalidInput, "invalid request body"))
		return
	}

	user, err := h.authService.UpdateProfile(c.Request.Context(), userID.(string), service.UpdateProfileRequest{
		Name:       req.Name,
		StudentNo:  req.StudentNo,
		Department: req.Department,
		Phone:      req.Phone,
	})
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(user))
}
