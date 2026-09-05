package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/zhouwu97/key-cabinet/server/internal/config"
	"github.com/zhouwu97/key-cabinet/server/internal/infrastructure/device"
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
	keyRepo := postgres.NewKeyRepository(db)
	deviceRepo := postgres.NewDeviceRepository(db)
	slotRepo := postgres.NewSlotRepository(db)
	reservationRepo := postgres.NewReservationRepository(db)
	borrowRepo := postgres.NewBorrowRepository(db)
	operationRepo := postgres.NewOperationRepository(db)

	// Initialize infrastructure clients
	wechatClient := wechat.NewClient(cfg.Wechat.AppID, cfg.Wechat.AppSecret, cfg.Wechat.MockEnabled)
	deviceGateway, err := newDeviceGateway(cfg.Device.GatewayType)
	if err != nil {
		log.Fatalf("Failed to initialize device gateway: %v", err)
	}

	// Initialize domain services
	authService := service.NewAuthService(userRepo, wechatClient, tokenService, cfg.JWT.Expiration)
	keyService := service.NewKeyService(keyRepo, slotRepo)
	deviceService := service.NewDeviceService(deviceRepo, slotRepo)
	reservationService := service.NewReservationService(reservationRepo, keyRepo, deviceRepo, borrowRepo)
	borrowService := service.NewBorrowService(borrowRepo)
	operationService := service.NewOperationService(operationRepo, reservationService, borrowService, keyRepo, deviceRepo, slotRepo, deviceGateway)
	startOverdueScheduler(borrowService)

	// Initialize handlers
	healthHandler := handler.NewHealthHandler(db)
	authHandler := handler.NewAuthHandler(authService)
	keyHandler := handler.NewKeyHandler(keyService)
	deviceHandler := handler.NewDeviceHandler(deviceService)
	reservationHandler := handler.NewReservationHandler(reservationService)
	borrowHandler := handler.NewBorrowHandler(borrowService)
	operationHandler := handler.NewOperationHandler(operationService)

	// Setup router
	router := http.SetupRouter(http.RouterConfig{
		HealthHandler:      healthHandler,
		AuthHandler:        authHandler,
		KeyHandler:         keyHandler,
		DeviceHandler:      deviceHandler,
		ReservationHandler: reservationHandler,
		BorrowHandler:      borrowHandler,
		OperationHandler:   operationHandler,
		TokenService:       tokenService,
	})

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Server starting on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func newDeviceGateway(gatewayType string) (device.DeviceGateway, error) {
	switch strings.ToLower(strings.TrimSpace(gatewayType)) {
	case "", "mock":
		return device.NewMockDeviceGateway(), nil
	default:
		return nil, fmt.Errorf("unsupported device gateway %q; MQTT gateway is not implemented yet", gatewayType)
	}
}

// 后台统一标记逾期借用，避免把借用状态判定分散到小程序端。
func startOverdueScheduler(borrowService service.BorrowService) {
	go func() {
		check := func() {
			if err := borrowService.CheckOverdue(context.Background(), time.Now().UTC()); err != nil {
				log.Printf("failed to mark overdue borrow records: %v", err)
			}
		}
		check()
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			check()
		}
	}()
}
