package http

import (
	"github.com/gin-gonic/gin"
	"github.com/zhouwu97/key-cabinet/server/internal/platform/jwt"
	"github.com/zhouwu97/key-cabinet/server/internal/transport/http/handler"
	"github.com/zhouwu97/key-cabinet/server/internal/transport/http/middleware"
)

type RouterConfig struct {
	HealthHandler      *handler.HealthHandler
	AuthHandler        *handler.AuthHandler
	KeyHandler         *handler.KeyHandler
	DeviceHandler      *handler.DeviceHandler
	ReservationHandler *handler.ReservationHandler
	BorrowHandler      *handler.BorrowHandler
	OperationHandler   *handler.OperationHandler
	TokenService       *jwt.TokenService
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
		if cfg.ReservationHandler != nil {
			v1Protected.GET("/keys/:id/availability", cfg.ReservationHandler.Availability)
		}
		if cfg.KeyHandler != nil {
			v1Protected.GET("/keys", cfg.KeyHandler.List)
			v1Protected.GET("/keys/:id/slot", cfg.KeyHandler.GetSlot)
			v1Protected.GET("/keys/:id", cfg.KeyHandler.GetByID)
		}
		if cfg.DeviceHandler != nil {
			v1Protected.GET("/devices", cfg.DeviceHandler.List)
			v1Protected.GET("/devices/:id/slots", cfg.DeviceHandler.GetSlots)
			v1Protected.GET("/devices/:id/status", cfg.DeviceHandler.GetStatus)
			v1Protected.GET("/devices/:id", cfg.DeviceHandler.GetByID)
		}
		if cfg.ReservationHandler != nil {
			v1Protected.POST("/reservations", cfg.ReservationHandler.Create)
			v1Protected.GET("/me/reservations", cfg.ReservationHandler.ListMine)
			v1Protected.GET("/reservations/:id", cfg.ReservationHandler.GetMine)
			v1Protected.POST("/reservations/:id/cancel", cfg.ReservationHandler.Cancel)
		}
		if cfg.BorrowHandler != nil {
			v1Protected.GET("/me/borrow-records", cfg.BorrowHandler.ListMine)
			v1Protected.GET("/borrow-records/:id", cfg.BorrowHandler.GetMine)
		}
		if cfg.OperationHandler != nil {
			v1Protected.POST("/device-operations/pickup", cfg.OperationHandler.StartPickup)
			v1Protected.POST("/device-operations/return", cfg.OperationHandler.StartReturn)
			v1Protected.GET("/device-operations/active", cfg.OperationHandler.GetActive)
			v1Protected.GET("/device-operations/:id", cfg.OperationHandler.GetByID)
			v1Protected.POST("/device-operations/:id/cancel", cfg.OperationHandler.Cancel)
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
