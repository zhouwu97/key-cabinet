package http

import (
	"github.com/gin-gonic/gin"
	"github.com/zhouwu97/key-cabinet/server/internal/platform/jwt"
	"github.com/zhouwu97/key-cabinet/server/internal/transport/http/handler"
	"github.com/zhouwu97/key-cabinet/server/internal/transport/http/middleware"
)

type RouterConfig struct {
	HealthHandler *handler.HealthHandler
	AuthHandler   *handler.AuthHandler
	TokenService  *jwt.TokenService
}

func SetupRouter(cfg RouterConfig) *gin.Engine {
	r := gin.Default()

	// Global middleware
	r.Use(middleware.ErrorMiddleware())

	// Public routes
	r.GET("/health", cfg.HealthHandler.Check)

	// API v1 - Public
	v1Public := r.Group("/api/v1")
	{
		if cfg.AuthHandler != nil {
			v1Public.POST("/auth/wechat-login", cfg.AuthHandler.WechatLogin)
		}
	}

	// API v1 - Protected
	v1Protected := r.Group("/api/v1")
	v1Protected.Use(middleware.AuthMiddleware(cfg.TokenService))
	{
		if cfg.AuthHandler != nil {
			v1Protected.GET("/me", cfg.AuthHandler.GetMe)
			v1Protected.PATCH("/me", cfg.AuthHandler.UpdateProfile)
			v1Protected.PUT("/me", cfg.AuthHandler.UpdateProfile)
		}
	}

	// Admin routes
	admin := r.Group("/admin")
	admin.Use(middleware.AuthMiddleware(cfg.TokenService))
	admin.Use(middleware.AdminMiddleware())
	{
		admin.GET("/health", cfg.HealthHandler.Check)
	}

	return r
}
