package postgres

import (
	"github.com/stretchr/testify/assert"
	"github.com/zhouwu97/key-cabinet/server/internal/config"
	"testing"
)

func TestNewDatabase(t *testing.T) {
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

	db, err := NewDatabase(cfg)
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
		return
	}

	assert.NotNil(t, db)

	err = HealthCheck(db)
	assert.NoError(t, err)

	sqlDB, _ := db.DB()
	sqlDB.Close()
}
