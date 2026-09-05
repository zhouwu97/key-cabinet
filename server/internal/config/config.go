package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	AppEnv   string         `mapstructure:"app_env"`
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Wechat   WechatConfig   `mapstructure:"wechat"`
	Device   DeviceConfig   `mapstructure:"device"`
	Log      LogConfig      `mapstructure:"log"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DBName       string `mapstructure:"dbname"`
	SSLMode      string `mapstructure:"sslmode"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

type JWTConfig struct {
	Secret     string `mapstructure:"secret"`
	Expiration int    `mapstructure:"expiration"` // seconds
}

type WechatConfig struct {
	AppID       string `mapstructure:"app_id"`
	AppSecret   string `mapstructure:"app_secret"`
	MockEnabled bool   `mapstructure:"mock_enabled"`
}

type DeviceConfig struct {
	GatewayType  string `mapstructure:"gateway_type"` // mock / mqtt
	MQTTBroker   string `mapstructure:"mqtt_broker"`
	MQTTUsername string `mapstructure:"mqtt_username"`
	MQTTPassword string `mapstructure:"mqtt_password"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`  // debug / info / warn / error
	Format string `mapstructure:"format"` // json / console
}

func Load(configPath string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetEnvPrefix("KC") // Key Cabinet prefix
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	v.SetDefault("app_env", "development")

	// 显式绑定嵌套配置，确保 AutomaticEnv 能覆盖 YAML 中的同名字段。
	for _, key := range []string{
		"app_env",
		"server.port", "server.mode",
		"database.host", "database.port", "database.user", "database.password",
		"database.dbname", "database.sslmode", "database.max_open_conns", "database.max_idle_conns",
		"jwt.secret", "jwt.expiration",
		"wechat.app_id", "wechat.app_secret", "wechat.mock_enabled",
		"device.gateway_type", "device.mqtt_broker", "device.mqtt_username", "device.mqtt_password",
		"log.level", "log.format",
	} {
		envName := "KC_" + strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
		legacyEnvName := strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
		if err := v.BindEnv(key, envName, legacyEnvName); err != nil {
			return nil, fmt.Errorf("failed to bind environment variable for %s: %w", key, err)
		}
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// Validate 检查启动所需的安全配置，避免生产环境静默回退到 Mock 或弱密钥。
func (c Config) Validate() error {
	env := strings.ToLower(strings.TrimSpace(c.AppEnv))
	if env == "" {
		env = "development"
	}

	if c.JWT.Secret == "" {
		return fmt.Errorf("jwt.secret is required")
	}
	if c.JWT.Expiration <= 0 {
		return fmt.Errorf("jwt.expiration must be greater than zero")
	}

	if env == "production" {
		if c.Wechat.MockEnabled {
			return fmt.Errorf("wechat.mock_enabled must be false in production")
		}
		if c.Wechat.AppID == "" || c.Wechat.AppSecret == "" {
			return fmt.Errorf("wechat.app_id and wechat.app_secret are required in production")
		}
		if isPlaceholder(c.JWT.Secret) || isPlaceholder(c.Wechat.AppID) || isPlaceholder(c.Wechat.AppSecret) {
			return fmt.Errorf("production configuration contains placeholder credentials")
		}
		return nil
	}

	if !c.Wechat.MockEnabled &&
		(c.Wechat.AppID == "" || c.Wechat.AppSecret == "" ||
			isPlaceholder(c.Wechat.AppID) || isPlaceholder(c.Wechat.AppSecret)) {
		return fmt.Errorf("wechat credentials are required when mock login is disabled")
	}
	return nil
}

func isPlaceholder(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(normalized, "change-in-production") ||
		strings.HasPrefix(normalized, "your-wechat-")
}
