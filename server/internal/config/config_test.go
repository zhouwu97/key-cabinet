package config

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	cfg, err := Load("config.yaml")
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "postgres", cfg.Database.User)
	assert.Equal(t, "mock", cfg.Device.GatewayType)
	assert.Equal(t, 86400, cfg.JWT.Expiration)
}
