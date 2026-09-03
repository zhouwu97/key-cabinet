package main

import (
	"fmt"
	"log"
	"github.com/yourusername/key-cabinet/server/internal/config"
	"github.com/yourusername/key-cabinet/server/internal/infrastructure/postgres"
	"github.com/yourusername/key-cabinet/server/internal/platform/jwt"
	"github.com/yourusername/key-cabinet/server/internal/transport/http"
	"github.com/yourusername/key-cabinet/server/internal/transport/http/handler"
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

	// Initialize handlers
	healthHandler := handler.NewHealthHandler(db)

	// Setup router
	router := http.SetupRouter(http.RouterConfig{
		HealthHandler: healthHandler,
		TokenService:  tokenService,
	})

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Server starting on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
