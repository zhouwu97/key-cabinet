package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zhouwu97/key-cabinet/server/internal/platform/jwt"
	"github.com/zhouwu97/key-cabinet/server/internal/repository"
	"github.com/zhouwu97/key-cabinet/server/internal/transport/http/dto"
)

// CabinetAuthMiddleware authenticates cabinet edge hardware via X-Cabinet-Device-ID.
// It verifies that the cabinet exists in the database and is currently ONLINE.
func CabinetAuthMiddleware(deviceRepo repository.DeviceRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := strings.TrimSpace(c.GetHeader("X-Cabinet-Device-ID"))
		if deviceID == "" {
			// Also support deviceId query param for fallback diagnostics if needed
			deviceID = strings.TrimSpace(c.Query("deviceId"))
		}

		if deviceID == "" {
			c.JSON(http.StatusUnauthorized, dto.NewErrorResponse(
				"UNAUTHORIZED_CABINET",
				"Missing X-Cabinet-Device-ID header",
			))
			c.Abort()
			return
		}

		device, err := deviceRepo.FindByID(c.Request.Context(), deviceID)
		if err != nil || device == nil {
			c.JSON(http.StatusUnauthorized, dto.NewErrorResponse(
				"DEVICE_NOT_FOUND",
				"Cabinet hardware is not registered in the system",
			))
			c.Abort()
			return
		}

		if device.Status != "ONLINE" {
			c.JSON(http.StatusForbidden, dto.NewErrorResponse(
				"DEVICE_OFFLINE",
				"Cabinet device is offline or unavailable",
			))
			c.Abort()
			return
		}

		c.Set("cabinet_device_id", deviceID)
		c.Next()
	}
}

// FaceSessionMiddleware validates that the request carries a short-lived face session token
// generated after successful edge biometric verification on the specific cabinet.
func FaceSessionMiddleware(tokenService *jwt.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, dto.NewErrorResponse(
				"UNAUTHORIZED",
				"Missing face session token",
			))
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, dto.NewErrorResponse(
				"UNAUTHORIZED",
				"Invalid authorization header format",
			))
			c.Abort()
			return
		}

		claims, err := tokenService.Validate(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, dto.NewErrorResponse(
				"INVALID_TOKEN",
				"Face session has expired or is invalid: "+err.Error(),
			))
			c.Abort()
			return
		}

		if claims.TokenType != "FACE_SESSION" {
			c.JSON(http.StatusForbidden, dto.NewErrorResponse(
				"INVALID_TOKEN_TYPE",
				"Operation requires a valid FACE_SESSION token",
			))
			c.Abort()
			return
		}

		if claims.DeviceID == "" || claims.UserID == "" {
			c.JSON(http.StatusForbidden, dto.NewErrorResponse(
				"INVALID_TOKEN",
				"Face session token missing user or device binding",
			))
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Set("cabinet_device_id", claims.DeviceID)
		c.Next()
	}
}
