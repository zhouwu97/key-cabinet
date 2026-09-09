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
	deviceStatusSink, ok := deviceRepo.(device.DeviceStatusSink)
	if !ok {
		log.Fatal("Device repository does not support runtime status updates")
	}
	deviceGateway, err := newDeviceGateway(cfg.Device, deviceStatusSink)
	if err != nil {
		log.Fatalf("Failed to initialize device gateway: %v", err)
	}
	if mqttGateway, ok := deviceGateway.(*device.MQTTDeviceGateway); ok {
		defer mqttGateway.Close()
	}

	// Initialize domain services
	authService := service.NewAuthService(userRepo, wechatClient, tokenService, cfg.JWT.Expiration)
	keyService := service.NewKeyService(keyRepo, slotRepo)
	deviceService := service.NewDeviceService(deviceRepo, slotRepo)
	reservationService := service.NewReservationService(reservationRepo, keyRepo, deviceRepo, borrowRepo, userRepo)
	borrowService := service.NewBorrowService(borrowRepo)
	operationService := service.NewOperationService(operationRepo, reservationService, borrowService, keyRepo, deviceRepo, slotRepo, deviceGateway, userRepo)
	startOverdueScheduler(borrowService)
	startOperationTimeoutScheduler(operationService)
	startReservationExpiryScheduler(reservationService)

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

func newDeviceGateway(cfg config.DeviceConfig, statusSink device.DeviceStatusSink) (device.DeviceGateway, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.GatewayType)) {
	case "", "mock":
		return device.NewMockDeviceGateway(), nil
	case "mqtt":
		return device.NewMQTTDeviceGateway(device.MQTTGatewayConfig{
			Broker:           cfg.MQTTBroker,
			ClientID:         cfg.MQTTClientID,
			Username:         cfg.MQTTUsername,
			Password:         cfg.MQTTPassword,
			TopicPrefix:      cfg.MQTTTopicPrefix,
			QoS:              byte(cfg.MQTTQoS),
			ConnectTimeout:   time.Duration(cfg.MQTTConnectTimeoutSec) * time.Second,
			CommandTimeout:   time.Duration(cfg.MQTTCommandTimeoutSec) * time.Second,
			HeartbeatTimeout: time.Duration(cfg.MQTTHeartbeatTimeoutSec) * time.Second,
		}, statusSink)
	default:
		return nil, fmt.Errorf("unsupported device gateway %q", cfg.GatewayType)
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

// 超时任务将长期无响应的操作收敛到终态，避免设备或钥匙一直被活动操作占用。
func startOperationTimeoutScheduler(operationService service.OperationService) {
	const operationTimeout = 2 * time.Minute
	go func() {
		check := func() {
			expired, err := operationService.ExpireTimedOutOperations(context.Background(), time.Now().UTC(), operationTimeout)
			if err != nil {
				log.Printf("failed to expire timed out device operations: %v", err)
				return
			}
			if expired > 0 {
				log.Printf("expired %d timed out device operations", expired)
			}
		}
		check()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			check()
		}
	}()
}

func startReservationExpiryScheduler(reservationService service.ReservationService) {
	go func() {
		check := func() {
			expired, err := reservationService.ExpireReservations(context.Background(), time.Now().UTC())
			if err != nil {
				log.Printf("failed to expire reservations: %v", err)
				return
			}
			if expired > 0 {
				log.Printf("expired %d reservations", expired)
			}
		}
		check()
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			check()
		}
	}()
}
