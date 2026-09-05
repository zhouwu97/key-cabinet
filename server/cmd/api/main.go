package main

import (
	"fmt"
	"log"

	"github.com/zhouwu97/key-cabinet/server/internal/config"
	"github.com/zhouwu97/key-cabinet/server/internal/infrastructure/postgres"
	"github.com/zhouwu97/key-cabinet/server/internal/infrastructure/wechat"
	"github.com/zhouwu97/key-cabinet/server/internal/platform/jwt"
	"github.com/zhouwu97/key-cabinet/server/internal/service"
	"github.com/zhouwu97/key-cabinet/server/internal/transport/http"
	"github.com/zhouwu97/key-cabinet/server/internal/transport/http/handler"
)

func main() {
	// Load config
	cfg, err := config.Load("internal/config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	db, err := postgres.NewDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Database connected successfully")

	// Initialize JWT service
	tokenService := jwt.NewTokenService(cfg.JWT.Secret, cfg.JWT.Expiration)

	// Initialize repositories
	userRepo := postgres.NewUserRepository(db)

	// Initialize infrastructure clients
	wechatClient := wechat.NewClient(cfg.Wechat.AppID, cfg.Wechat.AppSecret, cfg.Wechat.MockEnabled)

	// Initialize domain services
	authService := service.NewAuthService(userRepo, wechatClient, tokenService, cfg.JWT.Expiration)

	// Initialize handlers
	healthHandler := handler.NewHealthHandler(db)
	authHandler := handler.NewAuthHandler(authService)

	// Setup router
	router := http.SetupRouter(http.RouterConfig{
		HealthHandler: healthHandler,
		AuthHandler:   authHandler,
		TokenService:  tokenService,
	})

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Server starting on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
