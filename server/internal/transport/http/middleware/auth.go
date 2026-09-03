package middleware

import (
	"net/http"
	"strings"
	"github.com/gin-gonic/gin"
	"github.com/zhouwu97/key-cabinet/server/internal/platform/jwt"
	"github.com/zhouwu97/key-cabinet/server/internal/transport/http/dto"
)

func AuthMiddleware(tokenService *jwt.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, dto.NewErrorResponse(
				"UNAUTHORIZED",
				"Missing authorization token",
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
				err.Error(),
			))
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, dto.NewErrorResponse(
				"FORBIDDEN",
				"Role not found in context",
			))
			c.Abort()
			return
		}

		if role != "ADMIN" && role != "SUPER_ADMIN" {
			c.JSON(http.StatusForbidden, dto.NewErrorResponse(
				"FORBIDDEN",
				"Insufficient permissions",
			))
			c.Abort()
			return
		}

		c.Next()
	}
}
