package http

import (
	"github.com/gin-gonic/gin"
	"github.com/zhouwu97/key-cabinet/server/internal/transport/http/handler"
	"github.com/zhouwu97/key-cabinet/server/internal/transport/http/middleware"
	"github.com/zhouwu97/key-cabinet/server/internal/platform/jwt"
)

type RouterConfig struct {
	HealthHandler *handler.HealthHandler
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
		// Auth endpoints will be added in Sprint 4.2
		_ = v1Public
	}

	// API v1 - Protected
	v1Protected := r.Group("/api/v1")
	v1Protected.Use(middleware.AuthMiddleware(cfg.TokenService))
	{
		// Protected endpoints will be added in Sprint 4.3+
		_ = v1Protected
	}

	// Admin routes
	admin := r.Group("/admin")
	admin.Use(middleware.AuthMiddleware(cfg.TokenService))
	admin.Use(middleware.AdminMiddleware())
	{
		// Admin endpoints will be added in Sprint 4.5+
		_ = admin
	}

	return r
}
