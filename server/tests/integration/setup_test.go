package integration

import (
	"github.com/zhouwu97/key-cabinet/server/internal/config"
	"github.com/zhouwu97/key-cabinet/server/internal/infrastructure/postgres"
	"gorm.io/gorm"
	"testing"
)

func setupTestDB(t *testing.T) *gorm.DB {
	cfg := config.DatabaseConfig{
		Host:         "localhost",
		Port:         5432,
		User:         "postgres",
		Password:     "postgres",
		DBName:       "keycabinet_test",
		SSLMode:      "disable",
		MaxOpenConns: 5,
		MaxIdleConns: 2,
	}

	db, err := postgres.NewDatabase(cfg)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	return db
}

func teardownTestDB(t *testing.T, db *gorm.DB) {
	sqlDB, _ := db.DB()
	sqlDB.Close()
}
