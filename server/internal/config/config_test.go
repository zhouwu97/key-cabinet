package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	cfg, err := Load("config.example.yaml")
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "development", cfg.AppEnv)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "postgres", cfg.Database.User)
	assert.Equal(t, "mock", cfg.Device.GatewayType)
	assert.True(t, cfg.Wechat.MockEnabled)
	assert.Equal(t, 86400, cfg.JWT.Expiration)
}

func TestLoadConfigEnvironmentOverrides(t *testing.T) {
	t.Setenv("KC_DATABASE_HOST", "postgres")
	t.Setenv("KC_WECHAT_MOCK_ENABLED", "false")
	t.Setenv("KC_WECHAT_APP_ID", "wx-test")
	t.Setenv("KC_WECHAT_APP_SECRET", "secret-test")

	cfg, err := Load("config.example.yaml")
	require.NoError(t, err)
	assert.Equal(t, "postgres", cfg.Database.Host)
	assert.False(t, cfg.Wechat.MockEnabled)
	assert.Equal(t, "wx-test", cfg.Wechat.AppID)
}

func TestValidateProductionConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	configContent := []byte(`
app_env: production
jwt:
  secret: production-secret
  expiration: 3600
wechat:
  app_id: wx-test
  app_secret: secret-test
  mock_enabled: true
`)
	require.NoError(t, os.WriteFile(configPath, configContent, 0600))

	_, err := Load(configPath)
	assert.ErrorContains(t, err, "mock_enabled must be false in production")
}

func TestValidateProductionRequiresWechatCredentials(t *testing.T) {
	cfg := Config{
		AppEnv: "production",
		JWT: JWTConfig{
			Secret:     "production-secret",
			Expiration: 3600,
		},
		Wechat: WechatConfig{MockEnabled: false},
	}

	assert.ErrorContains(t, cfg.Validate(), "wechat.app_id and wechat.app_secret are required")
}
