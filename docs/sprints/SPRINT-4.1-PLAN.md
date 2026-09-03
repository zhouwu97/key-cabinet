# Sprint 4.1: Backend Foundation

**Sprint**: 4.1  
**Goal**: 建立 Go 后端服务的完整基础设施  
**Duration**: 预计 5-7 天  
**Status**: Planning  
**Date**: 2026-09-03

---

## Sprint Goal

完成 Go 后端服务的**基础设施层**，包括：

- ✅ 项目结构与依赖管理
- ✅ 配置加载系统
- ✅ PostgreSQL 连接与 Migration
- ✅ 统一错误处理
- ✅ JWT Middleware 骨架
- ✅ Repository Interface 定义
- ✅ DeviceGateway Interface 定义
- ✅ MockDeviceGateway 骨架
- ✅ Health API
- ✅ 基础测试通过

**验收标准**：能够启动服务，访问 Health API，执行 Migration，所有测试通过。

---

## Tasks Breakdown

### Task 4.1.1: Project Initialization

**目标**：创建 Go Module，定义目录结构，安装核心依赖。

#### Subtasks

1. **初始化 Go Module**
   ```bash
   cd server
   go mod init github.com/yourusername/key-cabinet/server
   ```

2. **创建目录结构**
   ```bash
   mkdir -p cmd/api
   mkdir -p cmd/migrate
   mkdir -p internal/{config,domain,service,repository,transport/http/handler,transport/http/middleware,transport/http/dto,transport/http/mapper,infrastructure/{postgres,wechat,device},platform/{jwt,clock,idgen,errors}}
   mkdir -p migrations
   mkdir -p tests/{integration,contract}
   ```

3. **安装核心依赖**
   ```bash
   # HTTP Framework
   go get github.com/gin-gonic/gin@v1.10.0

   # ORM
   go get gorm.io/gorm@v1.25.12
   go get gorm.io/driver/postgres@v1.5.9

   # JWT
   go get github.com/golang-jwt/jwt/v5@v5.2.1

   # Config
   go get github.com/spf13/viper@v1.19.0

   # Logging
   go get go.uber.org/zap@v1.27.0

   # UUID
   go get github.com/google/uuid@v1.6.0

   # Testing
   go get github.com/stretchr/testify@v1.9.0
   ```

4. **创建 Makefile**
   ```makefile
   .PHONY: run test migrate-up migrate-down build clean

   run:
       go run cmd/api/main.go

   test:
       go test ./...

   test-integration:
       go test ./tests/integration/...

   migrate-up:
       migrate -path migrations -database "postgresql://postgres:postgres@localhost:5432/keycabinet?sslmode=disable" up

   migrate-down:
       migrate -path migrations -database "postgresql://postgres:postgres@localhost:5432/keycabinet?sslmode=disable" down 1

   build:
       go build -o bin/api cmd/api/main.go

   clean:
       rm -rf bin/
   ```

**验收**：
```bash
go mod tidy
tree server/  # 目录结构正确
```

---

### Task 4.1.2: Configuration System

**目标**：实现配置加载系统，支持 YAML + 环境变量。

#### Files to Create

1. **`internal/config/config.go`**
   ```go
   package config

   import (
       "fmt"
       "github.com/spf13/viper"
   )

   type Config struct {
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
       AppID     string `mapstructure:"app_id"`
       AppSecret string `mapstructure:"app_secret"`
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
       viper.SetConfigFile(configPath)
       viper.AutomaticEnv()
       viper.SetEnvPrefix("KC") // Key Cabinet prefix

       if err := viper.ReadInConfig(); err != nil {
           return nil, fmt.Errorf("failed to read config: %w", err)
       }

       var config Config
       if err := viper.Unmarshal(&config); err != nil {
           return nil, fmt.Errorf("failed to unmarshal config: %w", err)
       }

       return &config, nil
   }
   ```

2. **`internal/config/config.yaml`**
   ```yaml
   server:
     port: 8080
     mode: debug # debug / release

   database:
     host: localhost
     port: 5432
     user: postgres
     password: postgres
     dbname: keycabinet
     sslmode: disable
     max_open_conns: 10
     max_idle_conns: 5

   jwt:
     secret: dev-secret-key-change-in-production
     expiration: 86400 # 24 hours

   wechat:
     app_id: your-wechat-app-id
     app_secret: your-wechat-app-secret

   device:
     gateway_type: mock # mock / mqtt
     mqtt_broker: tcp://localhost:1883
     mqtt_username: ""
     mqtt_password: ""

   log:
     level: debug # debug / info / warn / error
     format: console # json / console
   ```

3. **`internal/config/config_test.go`**
   ```go
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
   }
   ```

**验收**：
```bash
cd server/internal/config
go test -v
# PASS
```

---

### Task 4.1.3: Error Handling System

**目标**：实现统一的错误类型与错误码。

#### Files to Create

1. **`internal/platform/errors/codes.go`**
   ```go
   package errors

   type Code string

   const (
       // Client errors (4xx)
       CodeInvalidInput  Code = "INVALID_INPUT"
       CodeUnauthorized  Code = "UNAUTHORIZED"
       CodeForbidden     Code = "FORBIDDEN"
       CodeNotFound      Code = "RESOURCE_NOT_FOUND"
       CodeConflict      Code = "CONFLICT"
       CodeInvalidState  Code = "INVALID_STATE"
       CodeTooEarly      Code = "TOO_EARLY"
       CodeExpired       Code = "EXPIRED"
       CodeDuplicate     Code = "DUPLICATE"

       // Server errors (5xx)
       CodeInternalError Code = "INTERNAL_ERROR"
       CodeServiceUnavailable Code = "SERVICE_UNAVAILABLE"
   )
   ```

2. **`internal/platform/errors/error.go`**
   ```go
   package errors

   import "fmt"

   type AppError struct {
       Code    Code
       Message string
       Cause   error
   }

   func (e *AppError) Error() string {
       if e.Cause != nil {
           return fmt.Sprintf("[%s] %s (caused by: %v)", e.Code, e.Message, e.Cause)
       }
       return fmt.Sprintf("[%s] %s", e.Code, e.Message)
   }

   func (e *AppError) Unwrap() error {
       return e.Cause
   }

   // New creates a new AppError
   func New(code Code, message string) *AppError {
       return &AppError{
           Code:    code,
           Message: message,
       }
   }

   // Newf creates a new AppError with formatted message
   func Newf(code Code, format string, args ...interface{}) *AppError {
       return &AppError{
           Code:    code,
           Message: fmt.Sprintf(format, args...),
       }
   }

   // Wrap wraps an existing error with a message
   func Wrap(err error, message string) *AppError {
       return &AppError{
           Code:    CodeInternalError,
           Message: message,
           Cause:   err,
       }
   }

   // Wrapf wraps an existing error with a formatted message
   func Wrapf(err error, format string, args ...interface{}) *AppError {
       return &AppError{
           Code:    CodeInternalError,
           Message: fmt.Sprintf(format, args...),
           Cause:   err,
       }
   }

   // WrapWithCode wraps an existing error with a specific code
   func WrapWithCode(err error, code Code, message string) *AppError {
       return &AppError{
           Code:    code,
           Message: message,
           Cause:   err,
       }
   }

   // IsAppError checks if an error is an AppError
   func IsAppError(err error) (*AppError, bool) {
       if err == nil {
           return nil, false
       }
       if appErr, ok := err.(*AppError); ok {
           return appErr, true
       }
       return nil, false
   }
   ```

3. **`internal/platform/errors/error_test.go`**
   ```go
   package errors

   import (
       "errors"
       "testing"
       "github.com/stretchr/testify/assert"
   )

   func TestNew(t *testing.T) {
       err := New(CodeInvalidInput, "invalid email")
       assert.Equal(t, CodeInvalidInput, err.Code)
       assert.Equal(t, "invalid email", err.Message)
       assert.Contains(t, err.Error(), "INVALID_INPUT")
   }

   func TestWrap(t *testing.T) {
       cause := errors.New("db connection failed")
       err := Wrap(cause, "failed to query user")
       assert.Equal(t, CodeInternalError, err.Code)
       assert.NotNil(t, err.Cause)
       assert.Equal(t, cause, err.Unwrap())
   }

   func TestIsAppError(t *testing.T) {
       appErr := New(CodeNotFound, "user not found")
       result, ok := IsAppError(appErr)
       assert.True(t, ok)
       assert.Equal(t, CodeNotFound, result.Code)

       stdErr := errors.New("standard error")
       _, ok = IsAppError(stdErr)
       assert.False(t, ok)
   }
   ```

**验收**：
```bash
cd server/internal/platform/errors
go test -v
# PASS
```

---

### Task 4.1.4: PostgreSQL Connection & Migration

**目标**：实现数据库连接池与 Migration 工具。

#### Files to Create

1. **`internal/infrastructure/postgres/db.go`**
   ```go
   package postgres

   import (
       "fmt"
       "time"
       "gorm.io/driver/postgres"
       "gorm.io/gorm"
       "gorm.io/gorm/logger"
       "github.com/yourusername/key-cabinet/server/internal/config"
   )

   func NewDatabase(cfg config.DatabaseConfig) (*gorm.DB, error) {
       dsn := fmt.Sprintf(
           "host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
           cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
       )

       db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
           Logger: logger.Default.LogMode(logger.Info),
       })
       if err != nil {
           return nil, fmt.Errorf("failed to connect database: %w", err)
       }

       sqlDB, err := db.DB()
       if err != nil {
           return nil, fmt.Errorf("failed to get sql.DB: %w", err)
       }

       sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
       sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
       sqlDB.SetConnMaxLifetime(time.Hour)

       return db, nil
   }

   func HealthCheck(db *gorm.DB) error {
       sqlDB, err := db.DB()
       if err != nil {
           return err
       }
       return sqlDB.Ping()
   }
   ```

2. **`internal/infrastructure/postgres/transaction.go`**
   ```go
   package postgres

   import (
       "context"
       "gorm.io/gorm"
   )

   // TransactionFunc is a function that will be executed within a transaction
   type TransactionFunc func(tx *gorm.DB) error

   // WithTransaction executes a function within a database transaction
   func WithTransaction(db *gorm.DB, fn TransactionFunc) error {
       return db.Transaction(fn)
   }

   // WithTransactionContext executes a function within a database transaction with context
   func WithTransactionContext(ctx context.Context, db *gorm.DB, fn TransactionFunc) error {
       return db.WithContext(ctx).Transaction(fn)
   }
   ```

3. **`migrations/000001_init.up.sql`**
   ```sql
   -- Enable extensions
   CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
   CREATE EXTENSION IF NOT EXISTS "btree_gist";

   -- Users table
   CREATE TABLE users (
       id VARCHAR(64) PRIMARY KEY,
       name VARCHAR(100) NOT NULL,
       student_id VARCHAR(50),
       phone VARCHAR(20),
       role VARCHAR(20) NOT NULL DEFAULT 'STUDENT',
       status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
       created_at TIMESTAMP NOT NULL DEFAULT NOW(),
       updated_at TIMESTAMP NOT NULL DEFAULT NOW()
   );

   CREATE INDEX idx_users_student_id ON users(student_id);
   CREATE INDEX idx_users_status ON users(status);

   -- User identities table
   CREATE TABLE user_identities (
       id VARCHAR(64) PRIMARY KEY,
       user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
       identity_type VARCHAR(20) NOT NULL,
       provider_user_id VARCHAR(255) NOT NULL,
       metadata JSONB,
       created_at TIMESTAMP NOT NULL DEFAULT NOW(),
       UNIQUE(identity_type, provider_user_id)
   );

   CREATE INDEX idx_user_identities_user_id ON user_identities(user_id);

   -- Devices table
   CREATE TABLE devices (
       id VARCHAR(64) PRIMARY KEY,
       name VARCHAR(100) NOT NULL,
       location VARCHAR(200),
       capacity INT NOT NULL,
       status VARCHAR(20) NOT NULL DEFAULT 'ONLINE',
       created_at TIMESTAMP NOT NULL DEFAULT NOW(),
       updated_at TIMESTAMP NOT NULL DEFAULT NOW()
   );

   -- Slots table
   CREATE TABLE slots (
       id VARCHAR(64) PRIMARY KEY,
       device_id VARCHAR(64) NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
       slot_number INT NOT NULL,
       status VARCHAR(20) NOT NULL DEFAULT 'EMPTY',
       created_at TIMESTAMP NOT NULL DEFAULT NOW(),
       updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
       UNIQUE(device_id, slot_number)
   );

   CREATE INDEX idx_slots_device_id ON slots(device_id);
   CREATE INDEX idx_slots_status ON slots(status);

   -- Keys table
   CREATE TABLE keys (
       id VARCHAR(64) PRIMARY KEY,
       name VARCHAR(100) NOT NULL,
       key_number VARCHAR(50) NOT NULL UNIQUE,
       rfid_tag VARCHAR(100),
       device_id VARCHAR(64) NOT NULL REFERENCES devices(id),
       slot_id VARCHAR(64) REFERENCES slots(id),
       category VARCHAR(50),
       status VARCHAR(20) NOT NULL DEFAULT 'AVAILABLE',
       created_at TIMESTAMP NOT NULL DEFAULT NOW(),
       updated_at TIMESTAMP NOT NULL DEFAULT NOW()
   );

   CREATE INDEX idx_keys_device_id ON keys(device_id);
   CREATE INDEX idx_keys_status ON keys(status);
   CREATE INDEX idx_keys_rfid_tag ON keys(rfid_tag);

   -- Reservations table
   CREATE TABLE reservations (
       id VARCHAR(64) PRIMARY KEY,
       user_id VARCHAR(64) NOT NULL REFERENCES users(id),
       key_id VARCHAR(64) NOT NULL REFERENCES keys(id),
       status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
       start_time TIMESTAMP NOT NULL,
       end_time TIMESTAMP NOT NULL,
       purpose TEXT,
       created_at TIMESTAMP NOT NULL DEFAULT NOW(),
       updated_at TIMESTAMP NOT NULL DEFAULT NOW()
   );

   CREATE INDEX idx_reservations_user_id ON reservations(user_id);
   CREATE INDEX idx_reservations_key_id ON reservations(key_id);
   CREATE INDEX idx_reservations_status ON reservations(status);
   CREATE INDEX idx_reservations_time ON reservations(start_time, end_time);

   -- Borrow records table
   CREATE TABLE borrow_records (
       id VARCHAR(64) PRIMARY KEY,
       reservation_id VARCHAR(64) NOT NULL REFERENCES reservations(id),
       user_id VARCHAR(64) NOT NULL REFERENCES users(id),
       key_id VARCHAR(64) NOT NULL REFERENCES keys(id),
       status VARCHAR(20) NOT NULL DEFAULT 'BORROWED',
       borrowed_at TIMESTAMP NOT NULL,
       expected_return_at TIMESTAMP NOT NULL,
       returned_at TIMESTAMP,
       rfid_verified BOOLEAN DEFAULT FALSE,
       notes TEXT,
       created_at TIMESTAMP NOT NULL DEFAULT NOW(),
       updated_at TIMESTAMP NOT NULL DEFAULT NOW()
   );

   CREATE INDEX idx_borrow_records_reservation_id ON borrow_records(reservation_id);
   CREATE INDEX idx_borrow_records_user_id ON borrow_records(user_id);
   CREATE INDEX idx_borrow_records_key_id ON borrow_records(key_id);
   CREATE INDEX idx_borrow_records_status ON borrow_records(status);

   -- Device operations table
   CREATE TABLE device_operations (
       id VARCHAR(64) PRIMARY KEY,
       reservation_id VARCHAR(64) REFERENCES reservations(id),
       borrow_record_id VARCHAR(64) REFERENCES borrow_records(id),
       device_id VARCHAR(64) NOT NULL REFERENCES devices(id),
       slot_id VARCHAR(64) REFERENCES slots(id),
       key_id VARCHAR(64) REFERENCES keys(id),
       operation_type VARCHAR(20) NOT NULL,
       status VARCHAR(20) NOT NULL DEFAULT 'CREATED',
       initiated_at TIMESTAMP NOT NULL DEFAULT NOW(),
       completed_at TIMESTAMP,
       error_message TEXT,
       metadata JSONB,
       created_at TIMESTAMP NOT NULL DEFAULT NOW(),
       updated_at TIMESTAMP NOT NULL DEFAULT NOW()
   );

   CREATE INDEX idx_device_operations_reservation_id ON device_operations(reservation_id);
   CREATE INDEX idx_device_operations_device_id ON device_operations(device_id);
   CREATE INDEX idx_device_operations_status ON device_operations(status);
   CREATE INDEX idx_device_operations_type ON device_operations(operation_type);
   ```

4. **`migrations/000001_init.down.sql`**
   ```sql
   DROP TABLE IF EXISTS device_operations;
   DROP TABLE IF EXISTS borrow_records;
   DROP TABLE IF EXISTS reservations;
   DROP TABLE IF EXISTS keys;
   DROP TABLE IF EXISTS slots;
   DROP TABLE IF EXISTS devices;
   DROP TABLE IF EXISTS user_identities;
   DROP TABLE IF EXISTS users;
   DROP EXTENSION IF EXISTS btree_gist;
   DROP EXTENSION IF EXISTS "uuid-ossp";
   ```

5. **`cmd/migrate/main.go`**
   ```go
   package main

   import (
       "flag"
       "fmt"
       "log"
       "os"
       "github.com/golang-migrate/migrate/v4"
       _ "github.com/golang-migrate/migrate/v4/database/postgres"
       _ "github.com/golang-migrate/migrate/v4/source/file"
   )

   func main() {
       var migrationsPath string
       var databaseURL string
       var command string

       flag.StringVar(&migrationsPath, "path", "migrations", "Path to migrations directory")
       flag.StringVar(&databaseURL, "database", os.Getenv("DATABASE_URL"), "Database URL")
       flag.StringVar(&command, "command", "up", "Migration command: up, down, force, version")
       flag.Parse()

       if databaseURL == "" {
           databaseURL = "postgresql://postgres:postgres@localhost:5432/keycabinet?sslmode=disable"
       }

       m, err := migrate.New(
           fmt.Sprintf("file://%s", migrationsPath),
           databaseURL,
       )
       if err != nil {
           log.Fatalf("Failed to create migrate instance: %v", err)
       }

       switch command {
       case "up":
           if err := m.Up(); err != nil && err != migrate.ErrNoChange {
               log.Fatalf("Failed to run migrations: %v", err)
           }
           log.Println("Migrations applied successfully")
       case "down":
           if err := m.Down(); err != nil && err != migrate.ErrNoChange {
               log.Fatalf("Failed to rollback migrations: %v", err)
           }
           log.Println("Migrations rolled back successfully")
       case "version":
           version, dirty, err := m.Version()
           if err != nil {
               log.Fatalf("Failed to get version: %v", err)
           }
           log.Printf("Current version: %d, Dirty: %t\n", version, dirty)
       default:
           log.Fatalf("Unknown command: %s", command)
       }
   }
   ```

**验收**：
```bash
# 启动 PostgreSQL
docker run --name postgres-keycabinet -e POSTGRES_PASSWORD=postgres -p 5432:5432 -d postgres:15

# 创建数据库
docker exec -it postgres-keycabinet psql -U postgres -c "CREATE DATABASE keycabinet;"

# 执行 migration
cd server
go run cmd/migrate/main.go -command up

# 验证表创建成功
docker exec -it postgres-keycabinet psql -U postgres -d keycabinet -c "\dt"
# 应该看到所有表
```

---

### Task 4.1.5: JWT Middleware Skeleton

**目标**：实现 JWT 生成、验证与中间件（暂时不接微信登录）。

#### Files to Create

1. **`internal/platform/jwt/claims.go`**
   ```go
   package jwt

   import "github.com/golang-jwt/jwt/v5"

   type Claims struct {
       UserID string `json:"user_id"`
       Role   string `json:"role"`
       jwt.RegisteredClaims
   }
   ```

2. **`internal/platform/jwt/token.go`**
   ```go
   package jwt

   import (
       "fmt"
       "time"
       "github.com/golang-jwt/jwt/v5"
   )

   type TokenService struct {
       secret     string
       expiration time.Duration
   }

   func NewTokenService(secret string, expirationSeconds int) *TokenService {
       return &TokenService{
           secret:     secret,
           expiration: time.Duration(expirationSeconds) * time.Second,
       }
   }

   func (s *TokenService) Generate(userID, role string) (string, error) {
       claims := &Claims{
           UserID: userID,
           Role:   role,
           RegisteredClaims: jwt.RegisteredClaims{
               ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.expiration)),
               IssuedAt:  jwt.NewNumericDate(time.Now()),
           },
       }

       token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
       return token.SignedString([]byte(s.secret))
   }

   func (s *TokenService) Validate(tokenString string) (*Claims, error) {
       token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
           if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
               return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
           }
           return []byte(s.secret), nil
       })

       if err != nil {
           return nil, err
       }

       if claims, ok := token.Claims.(*Claims); ok && token.Valid {
           return claims, nil
       }

       return nil, fmt.Errorf("invalid token")
   }
   ```

3. **`internal/platform/jwt/token_test.go`**
   ```go
   package jwt

   import (
       "testing"
       "github.com/stretchr/testify/assert"
   )

   func TestGenerateAndValidate(t *testing.T) {
       service := NewTokenService("test-secret", 3600)

       token, err := service.Generate("U001", "STUDENT")
       assert.NoError(t, err)
       assert.NotEmpty(t, token)

       claims, err := service.Validate(token)
       assert.NoError(t, err)
       assert.Equal(t, "U001", claims.UserID)
       assert.Equal(t, "STUDENT", claims.Role)
   }

   func TestValidateInvalidToken(t *testing.T) {
       service := NewTokenService("test-secret", 3600)

       _, err := service.Validate("invalid.token.here")
       assert.Error(t, err)
   }
   ```

4. **`internal/transport/http/middleware/auth.go`**
   ```go
   package middleware

   import (
       "net/http"
       "strings"
       "github.com/gin-gonic/gin"
       "github.com/yourusername/key-cabinet/server/internal/platform/jwt"
       "github.com/yourusername/key-cabinet/server/internal/transport/http/dto"
   )

   func AuthMiddleware(tokenService *jwt.TokenService) gin.HandlerFunc {
       return func(c *gin.Context) {
           authHeader := c.GetHeader("Authorization")
           if authHeader == "" {
               c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
                   Error: dto.ErrorDetail{
                       Code:    "UNAUTHORIZED",
                       Message: "Missing authorization token",
                   },
               })
               c.Abort()
               return
           }

           tokenString := strings.TrimPrefix(authHeader, "Bearer ")
           if tokenString == authHeader {
               c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
                   Error: dto.ErrorDetail{
                       Code:    "UNAUTHORIZED",
                       Message: "Invalid authorization header format",
                   },
               })
               c.Abort()
               return
           }

           claims, err := tokenService.Validate(tokenString)
           if err != nil {
               c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
                   Error: dto.ErrorDetail{
                       Code:    "INVALID_TOKEN",
                       Message: err.Error(),
                   },
               })
               c.Abort()
               return
           }

           c.Set("user_id", claims.UserID)
           c.Set("role", claims.Role)
           c.Next()
       }
   }

   func AdminMiddleware() gin.HandlerFunc {
       return func(c *gin.Context) {
           role, exists := c.Get("role")
           if !exists {
               c.JSON(http.StatusForbidden, dto.ErrorResponse{
                   Error: dto.ErrorDetail{
                       Code:    "FORBIDDEN",
                       Message: "Role not found in context",
                   },
               })
               c.Abort()
               return
           }

           if role != "ADMIN" && role != "SUPER_ADMIN" {
               c.JSON(http.StatusForbidden, dto.ErrorResponse{
                   Error: dto.ErrorDetail{
                       Code:    "FORBIDDEN",
                       Message: "Insufficient permissions",
                   },
               })
               c.Abort()
               return
           }

           c.Next()
       }
   }
   ```

5. **`internal/transport/http/dto/common.go`**
   ```go
   package dto

   import "time"

   type ErrorResponse struct {
       Error ErrorDetail `json:"error"`
   }

   type ErrorDetail struct {
       Code      string `json:"code"`
       Message   string `json:"message"`
       Timestamp string `json:"timestamp"`
   }

   func NewErrorResponse(code, message string) ErrorResponse {
       return ErrorResponse{
           Error: ErrorDetail{
               Code:      code,
               Message:   message,
               Timestamp: time.Now().Format(time.RFC3339),
           },
       }
   }
   ```

**验收**：
```bash
cd server/internal/platform/jwt
go test -v
# PASS
```

---

### Task 4.1.6: Repository Interfaces

**目标**：定义核心 Repository 接口（实现留到后续 Sprint）。

#### Files to Create

1. **`internal/repository/user_repository.go`**
   ```go
   package repository

   import (
       "context"
   )

   type User struct {
       ID        string
       Name      string
       StudentID string
       Phone     string
       Role      string
       Status    string
   }

   type UserIdentity struct {
       ID             string
       UserID         string
       IdentityType   string
       ProviderUserID string
   }

   type UserRepository interface {
       Create(ctx context.Context, user *User) error
       FindByID(ctx context.Context, id string) (*User, error)
       FindByStudentID(ctx context.Context, studentID string) (*User, error)
       Update(ctx context.Context, user *User) error
       FindIdentity(ctx context.Context, identityType, providerUserID string) (*UserIdentity, error)
       CreateIdentity(ctx context.Context, identity *UserIdentity) error
   }
   ```

2. **`internal/repository/key_repository.go`**
   ```go
   package repository

   import "context"

   type Key struct {
       ID        string
       Name      string
       KeyNumber string
       RFIDTag   string
       DeviceID  string
       SlotID    string
       Category  string
       Status    string
   }

   type KeyRepository interface {
       FindByID(ctx context.Context, id string) (*Key, error)
       FindAll(ctx context.Context) ([]*Key, error)
       FindByStatus(ctx context.Context, status string) ([]*Key, error)
       Update(ctx context.Context, key *Key) error
   }
   ```

3. **`internal/repository/reservation_repository.go`**
   ```go
   package repository

   import (
       "context"
       "time"
   )

   type Reservation struct {
       ID        string
       UserID    string
       KeyID     string
       Status    string
       StartTime time.Time
       EndTime   time.Time
       Purpose   string
       CreatedAt time.Time
       UpdatedAt time.Time
   }

   type ReservationRepository interface {
       Create(ctx context.Context, r *Reservation) error
       FindByID(ctx context.Context, id string) (*Reservation, error)
       FindByUserID(ctx context.Context, userID string) ([]*Reservation, error)
       FindConflicts(ctx context.Context, keyID string, startTime, endTime time.Time) ([]*Reservation, error)
       Update(ctx context.Context, r *Reservation) error
   }
   ```

4. **`internal/repository/borrow_repository.go`**
   ```go
   package repository

   import (
       "context"
       "time"
   )

   type BorrowRecord struct {
       ID               string
       ReservationID    string
       UserID           string
       KeyID            string
       Status           string
       BorrowedAt       time.Time
       ExpectedReturnAt time.Time
       ReturnedAt       *time.Time
       RFIDVerified     bool
       Notes            string
   }

   type BorrowRepository interface {
       Create(ctx context.Context, record *BorrowRecord) error
       FindByID(ctx context.Context, id string) (*BorrowRecord, error)
       FindByReservationID(ctx context.Context, reservationID string) (*BorrowRecord, error)
       FindByUserID(ctx context.Context, userID string) ([]*BorrowRecord, error)
       Update(ctx context.Context, record *BorrowRecord) error
   }
   ```

5. **`internal/repository/operation_repository.go`**
   ```go
   package repository

   import (
       "context"
       "time"
   )

   type DeviceOperation struct {
       ID              string
       ReservationID   string
       BorrowRecordID  string
       DeviceID        string
       SlotID          string
       KeyID           string
       OperationType   string
       Status          string
       InitiatedAt     time.Time
       CompletedAt     *time.Time
       ErrorMessage    string
   }

   type OperationRepository interface {
       Create(ctx context.Context, op *DeviceOperation) error
       FindByID(ctx context.Context, id string) (*DeviceOperation, error)
       FindByReservationID(ctx context.Context, reservationID string) ([]*DeviceOperation, error)
       FindActiveByReservation(ctx context.Context, reservationID string) (*DeviceOperation, error)
       Update(ctx context.Context, op *DeviceOperation) error
   }
   ```

**验收**：接口定义清晰，编译通过。

---

### Task 4.1.7: DeviceGateway Interface & MockDeviceGateway

**目标**：定义设备网关接口，实现 Mock 版本。

#### Files to Create

1. **`internal/infrastructure/device/gateway.go`**
   ```go
   package device

   import (
       "context"
       "time"
   )

   type DeviceCommand struct {
       OperationID string
       DeviceID    string
       SlotID      string
       Type        string // PICKUP / RETURN
   }

   type DeviceEvent struct {
       OperationID string
       DeviceID    string
       EventType   string // PICKUP_SUCCESS / PICKUP_FAILED / RETURN_SUCCESS / RETURN_FAILED
       Timestamp   time.Time
       ErrorCode   string
       ErrorMessage string
   }

   type DeviceStatus struct {
       DeviceID string
       Online   bool
       LastSeen time.Time
   }

   type DeviceGateway interface {
       StartPickup(ctx context.Context, cmd DeviceCommand) error
       StartReturn(ctx context.Context, cmd DeviceCommand) error
       GetDeviceStatus(ctx context.Context, deviceID string) (*DeviceStatus, error)
       RegisterEventHandler(handler DeviceEventHandler)
   }

   type DeviceEventHandler interface {
       OnPickupSuccess(ctx context.Context, event DeviceEvent) error
       OnPickupFailed(ctx context.Context, event DeviceEvent) error
       OnReturnSuccess(ctx context.Context, event DeviceEvent) error
       OnReturnFailed(ctx context.Context, event DeviceEvent) error
   }
   ```

2. **`internal/infrastructure/device/mock_gateway.go`**
   ```go
   package device

   import (
       "context"
       "log"
       "time"
   )

   type MockDeviceGateway struct {
       handler DeviceEventHandler
   }

   func NewMockDeviceGateway() *MockDeviceGateway {
       return &MockDeviceGateway{}
   }

   func (g *MockDeviceGateway) RegisterEventHandler(handler DeviceEventHandler) {
       g.handler = handler
   }

   func (g *MockDeviceGateway) StartPickup(ctx context.Context, cmd DeviceCommand) error {
       log.Printf("[MockDeviceGateway] StartPickup: OperationID=%s, DeviceID=%s, SlotID=%s",
           cmd.OperationID, cmd.DeviceID, cmd.SlotID)

       // 模拟设备处理延迟
       go func() {
           time.Sleep(100 * time.Millisecond)
           event := DeviceEvent{
               OperationID: cmd.OperationID,
               DeviceID:    cmd.DeviceID,
               EventType:   "PICKUP_SUCCESS",
               Timestamp:   time.Now(),
           }
           if g.handler != nil {
               if err := g.handler.OnPickupSuccess(context.Background(), event); err != nil {
                   log.Printf("[MockDeviceGateway] OnPickupSuccess failed: %v", err)
               }
           }
       }()

       return nil
   }

   func (g *MockDeviceGateway) StartReturn(ctx context.Context, cmd DeviceCommand) error {
       log.Printf("[MockDeviceGateway] StartReturn: OperationID=%s, DeviceID=%s, SlotID=%s",
           cmd.OperationID, cmd.DeviceID, cmd.SlotID)

       // 模拟设备处理延迟
       go func() {
           time.Sleep(100 * time.Millisecond)
           event := DeviceEvent{
               OperationID: cmd.OperationID,
               DeviceID:    cmd.DeviceID,
               EventType:   "RETURN_SUCCESS",
               Timestamp:   time.Now(),
           }
           if g.handler != nil {
               if err := g.handler.OnReturnSuccess(context.Background(), event); err != nil {
                   log.Printf("[MockDeviceGateway] OnReturnSuccess failed: %v", err)
               }
           }
       }()

       return nil
   }

   func (g *MockDeviceGateway) GetDeviceStatus(ctx context.Context, deviceID string) (*DeviceStatus, error) {
       return &DeviceStatus{
           DeviceID: deviceID,
           Online:   true,
           LastSeen: time.Now(),
       }, nil
   }
   ```

**验收**：
```go
gateway := NewMockDeviceGateway()
cmd := DeviceCommand{
    OperationID: "OP001",
    DeviceID:    "DEVICE001",
    SlotID:      "SLOT001",
    Type:        "PICKUP",
}
err := gateway.StartPickup(context.Background(), cmd)
// 应该返回 nil，并在 100ms 后触发 OnPickupSuccess
```

---

### Task 4.1.8: Error Middleware

**目标**：实现统一错误处理中间件。

#### Files to Create

1. **`internal/transport/http/middleware/error.go`**
   ```go
   package middleware

   import (
       "net/http"
       "github.com/gin-gonic/gin"
       "github.com/yourusername/key-cabinet/server/internal/platform/errors"
       "github.com/yourusername/key-cabinet/server/internal/transport/http/dto"
   )

   func ErrorMiddleware() gin.HandlerFunc {
       return func(c *gin.Context) {
           c.Next()

           if len(c.Errors) > 0 {
               err := c.Errors.Last().Err
               appErr, ok := errors.IsAppError(err)
               if !ok {
                   appErr = errors.New(errors.CodeInternalError, err.Error())
               }

               statusCode := mapErrorToHTTPStatus(appErr.Code)
               c.JSON(statusCode, dto.NewErrorResponse(string(appErr.Code), appErr.Message))
           }
       }
   }

   func mapErrorToHTTPStatus(code errors.Code) int {
       switch code {
       case errors.CodeInvalidInput:
           return http.StatusBadRequest
       case errors.CodeUnauthorized:
           return http.StatusUnauthorized
       case errors.CodeForbidden:
           return http.StatusForbidden
       case errors.CodeNotFound:
           return http.StatusNotFound
       case errors.CodeConflict, errors.CodeInvalidState, errors.CodeDuplicate:
           return http.StatusConflict
       case errors.CodeTooEarly:
           return http.StatusBadRequest
       case errors.CodeExpired:
           return http.StatusGone
       case errors.CodeServiceUnavailable:
           return http.StatusServiceUnavailable
       default:
           return http.StatusInternalServerError
       }
   }
   ```

**验收**：
```go
// Handler 中调用：
c.Error(errors.New(errors.CodeNotFound, "Key not found"))
// 应该返回：
// HTTP 404
// {"error": {"code": "RESOURCE_NOT_FOUND", "message": "Key not found", "timestamp": "..."}}
```

---

### Task 4.1.9: Health API & Router

**目标**：实现 Health API 与基础路由。

#### Files to Create

1. **`internal/transport/http/handler/health_handler.go`**
   ```go
   package handler

   import (
       "net/http"
       "github.com/gin-gonic/gin"
       "gorm.io/gorm"
       "github.com/yourusername/key-cabinet/server/internal/infrastructure/postgres"
   )

   type HealthHandler struct {
       db *gorm.DB
   }

   func NewHealthHandler(db *gorm.DB) *HealthHandler {
       return &HealthHandler{db: db}
   }

   type HealthResponse struct {
       Status    string `json:"status"`
       Database  string `json:"database"`
       Timestamp string `json:"timestamp"`
   }

   func (h *HealthHandler) Check(c *gin.Context) {
       dbStatus := "ok"
       if err := postgres.HealthCheck(h.db); err != nil {
           dbStatus = "error"
       }

       status := "ok"
       if dbStatus != "ok" {
           status = "degraded"
       }

       c.JSON(http.StatusOK, HealthResponse{
           Status:    status,
           Database:  dbStatus,
           Timestamp: time.Now().Format(time.RFC3339),
       })
   }
   ```

2. **`internal/transport/http/router.go`**
   ```go
   package http

   import (
       "github.com/gin-gonic/gin"
       "github.com/yourusername/key-cabinet/server/internal/transport/http/handler"
       "github.com/yourusername/key-cabinet/server/internal/transport/http/middleware"
       "github.com/yourusername/key-cabinet/server/internal/platform/jwt"
   )

   type RouterConfig struct {
       HealthHandler *handler.HealthHandler
       TokenService  *jwt.TokenService
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
           // Auth endpoints will be added in Sprint 4.2
       }

       // API v1 - Protected
       v1Protected := r.Group("/api/v1")
       v1Protected.Use(middleware.AuthMiddleware(cfg.TokenService))
       {
           // Protected endpoints will be added in Sprint 4.3+
       }

       // Admin routes
       admin := r.Group("/admin")
       admin.Use(middleware.AuthMiddleware(cfg.TokenService))
       admin.Use(middleware.AdminMiddleware())
       {
           // Admin endpoints will be added in Sprint 4.5+
       }

       return r
   }
   ```

3. **`cmd/api/main.go`**
   ```go
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
   ```

**验收**：
```bash
# 启动服务
go run cmd/api/main.go

# 测试 Health API
curl http://localhost:8080/health

# 应该返回：
# {
#   "status": "ok",
#   "database": "ok",
#   "timestamp": "2026-09-03T10:30:00Z"
# }
```

---

### Task 4.1.10: Basic Tests

**目标**：编写基础测试，确保所有组件可以正常工作。

#### Files to Create

1. **`tests/integration/setup_test.go`**
   ```go
   package integration

   import (
       "testing"
       "gorm.io/gorm"
       "github.com/yourusername/key-cabinet/server/internal/config"
       "github.com/yourusername/key-cabinet/server/internal/infrastructure/postgres"
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
   ```

2. **`tests/integration/health_test.go`**
   ```go
   package integration

   import (
       "net/http"
       "net/http/httptest"
       "testing"
       "github.com/stretchr/testify/assert"
       transportHttp "github.com/yourusername/key-cabinet/server/internal/transport/http"
       "github.com/yourusername/key-cabinet/server/internal/transport/http/handler"
       "github.com/yourusername/key-cabinet/server/internal/platform/jwt"
   )

   func TestHealthCheck(t *testing.T) {
       db := setupTestDB(t)
       defer teardownTestDB(t, db)

       healthHandler := handler.NewHealthHandler(db)
       tokenService := jwt.NewTokenService("test-secret", 3600)

       router := transportHttp.SetupRouter(transportHttp.RouterConfig{
           HealthHandler: healthHandler,
           TokenService:  tokenService,
       })

       w := httptest.NewRecorder()
       req, _ := http.NewRequest("GET", "/health", nil)
       router.ServeHTTP(w, req)

       assert.Equal(t, http.StatusOK, w.Code)
       assert.Contains(t, w.Body.String(), "ok")
       assert.Contains(t, w.Body.String(), "database")
   }
   ```

3. **`tests/integration/auth_middleware_test.go`**
   ```go
   package integration

   import (
       "net/http"
       "net/http/httptest"
       "testing"
       "github.com/gin-gonic/gin"
       "github.com/stretchr/testify/assert"
       "github.com/yourusername/key-cabinet/server/internal/platform/jwt"
       "github.com/yourusername/key-cabinet/server/internal/transport/http/middleware"
   )

   func TestAuthMiddleware_NoToken(t *testing.T) {
       gin.SetMode(gin.TestMode)
       r := gin.New()
       tokenService := jwt.NewTokenService("test-secret", 3600)
       r.Use(middleware.AuthMiddleware(tokenService))
       r.GET("/test", func(c *gin.Context) {
           c.JSON(http.StatusOK, gin.H{"message": "ok"})
       })

       w := httptest.NewRecorder()
       req, _ := http.NewRequest("GET", "/test", nil)
       r.ServeHTTP(w, req)

       assert.Equal(t, http.StatusUnauthorized, w.Code)
       assert.Contains(t, w.Body.String(), "UNAUTHORIZED")
   }

   func TestAuthMiddleware_ValidToken(t *testing.T) {
       gin.SetMode(gin.TestMode)
       r := gin.New()
       tokenService := jwt.NewTokenService("test-secret", 3600)

       token, _ := tokenService.Generate("U001", "STUDENT")

       r.Use(middleware.AuthMiddleware(tokenService))
       r.GET("/test", func(c *gin.Context) {
           userID, _ := c.Get("user_id")
           c.JSON(http.StatusOK, gin.H{"user_id": userID})
       })

       w := httptest.NewRecorder()
       req, _ := http.NewRequest("GET", "/test", nil)
       req.Header.Set("Authorization", "Bearer "+token)
       r.ServeHTTP(w, req)

       assert.Equal(t, http.StatusOK, w.Code)
       assert.Contains(t, w.Body.String(), "U001")
   }
   ```

**验收**：
```bash
# 创建测试数据库
docker exec -it postgres-keycabinet psql -U postgres -c "CREATE DATABASE keycabinet_test;"

# 执行测试 migration
cd server
export DATABASE_URL="postgresql://postgres:postgres@localhost:5432/keycabinet_test?sslmode=disable"
go run cmd/migrate/main.go -database $DATABASE_URL -command up

# 运行测试
go test ./tests/integration/... -v
# 应该全部 PASS
```

---

## Acceptance Criteria

### Sprint 4.1 完成标准

所有以下验收步骤必须成功：

#### 1. 项目结构检查
```bash
cd server
tree -L 3
# 应该看到完整的目录结构
```

#### 2. 依赖安装成功
```bash
go mod tidy
go mod verify
# 无错误
```

#### 3. 配置加载成功
```bash
cd internal/config
go test -v
# PASS
```

#### 4. 错误处理测试通过
```bash
cd internal/platform/errors
go test -v
# PASS
```

#### 5. JWT 测试通过
```bash
cd internal/platform/jwt
go test -v
# PASS
```

#### 6. 数据库连接成功
```bash
# 启动 PostgreSQL
docker run --name postgres-keycabinet -e POSTGRES_PASSWORD=postgres -p 5432:5432 -d postgres:15

# 创建数据库
docker exec -it postgres-keycabinet psql -U postgres -c "CREATE DATABASE keycabinet;"

# 测试连接
psql -h localhost -U postgres -d keycabinet -c "SELECT 1;"
# 应该返回 1
```

#### 7. Migration 执行成功
```bash
cd server
go run cmd/migrate/main.go -command up

# 验证表创建
docker exec -it postgres-keycabinet psql -U postgres -d keycabinet -c "\dt"
# 应该看到所有表：users, user_identities, devices, slots, keys, reservations, borrow_records, device_operations
```

#### 8. 服务启动成功
```bash
go run cmd/api/main.go
# 应该看到：
# Database connected successfully
# Server starting on :8080
```

#### 9. Health API 验证
```bash
curl http://localhost:8080/health
# 应该返回：
# {
#   "status": "ok",
#   "database": "ok",
#   "timestamp": "2026-09-03T..."
# }
```

#### 10. Auth Middleware 验证
```bash
# 无 Token
curl http://localhost:8080/api/v1/test
# 应该返回 401: {"error": {"code": "UNAUTHORIZED", ...}}

# 非法 Token
curl -H "Authorization: Bearer invalid.token.here" http://localhost:8080/api/v1/test
# 应该返回 401: {"error": {"code": "INVALID_TOKEN", ...}}
```

#### 11. 集成测试通过
```bash
# 创建测试数据库
docker exec -it postgres-keycabinet psql -U postgres -c "CREATE DATABASE keycabinet_test;"

# 执行测试 migration
export DATABASE_URL="postgresql://postgres:postgres@localhost:5432/keycabinet_test?sslmode=disable"
go run cmd/migrate/main.go -database $DATABASE_URL -command up

# 运行所有测试
go test ./... -v
# 应该全部 PASS
```

#### 12. 代码规范检查
```bash
go fmt ./...
go vet ./...
# 无错误
```

---

## Technical Notes

### 1. Database Connection Pool

```go
// internal/infrastructure/postgres/db.go
sqlDB.SetMaxOpenConns(10)      // 最大打开连接数
sqlDB.SetMaxIdleConns(5)       // 最大空闲连接数
sqlDB.SetConnMaxLifetime(time.Hour) // 连接最大生命周期
```

### 2. JWT Secret

⚠️ **重要**：`config.yaml` 中的 `jwt.secret` 仅用于开发环境。

生产环境必须通过环境变量设置：
```bash
export KC_JWT_SECRET="production-secret-key-use-32-bytes-or-more"
```

### 3. Migration Best Practices

- 每个 migration 文件只做一件事
- `up` 和 `down` 必须成对
- 不要修改已执行的 migration，创建新的 migration
- 生产环境只执行 `up`，本地开发可以 `down` 回滚

### 4. Error Handling Pattern

```go
// 业务错误
if user == nil {
    return errors.New(errors.CodeNotFound, "用户不存在")
}

// 包装系统错误
if err := db.Create(user).Error; err != nil {
    return errors.Wrap(err, "创建用户失败")
}

// Handler 中传递错误
if err := service.CreateUser(ctx, req); err != nil {
    c.Error(err) // 由 ErrorMiddleware 统一处理
    return
}
```

### 5. Context Propagation

所有 Repository 和 Service 方法都应该接受 `context.Context`：

```go
func (s *UserService) CreateUser(ctx context.Context, req CreateUserRequest) error {
    // 传递 context 到 repository
    return s.userRepo.Create(ctx, user)
}
```

这样可以支持：
- Request-scoped tracing
- Timeout control
- Cancellation propagation

---

## Dependencies

```go
// server/go.mod
require (
    github.com/gin-gonic/gin v1.10.0
    gorm.io/gorm v1.25.12
    gorm.io/driver/postgres v1.5.9
    github.com/golang-jwt/jwt/v5 v5.2.1
    github.com/golang-migrate/migrate/v4 v4.18.1
    github.com/google/uuid v1.6.0
    github.com/spf13/viper v1.19.0
    go.uber.org/zap v1.27.0
    github.com/stretchr/testify v1.9.0
)
```

安装 Migration CLI：
```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

---

## Risks & Mitigations

### Risk 1: PostgreSQL 不可用
**Mitigation**: 提供 Docker Compose 配置，快速启动本地 PostgreSQL

### Risk 2: Migration 执行失败
**Mitigation**: 每次 migration 前备份数据库，确保可回滚

### Risk 3: JWT Secret 泄露
**Mitigation**: 明确文档说明生产环境使用环境变量，不提交 secret 到 Git

---

## Next Sprint Preview

**Sprint 4.2**: Auth + Identity + User

- 实现微信 code2Session 调用
- 实现 UserRepository
- 实现 UserIdentity 机制
- 实现 `/api/v1/auth/wechat-login` 接口
- 实现 `/api/v1/me` 接口
- 编写 Auth 集成测试

---

## References

- [Go Project Layout](https://go.dev/doc/modules/layout)
- [Gin Documentation](https://gin-gonic.com/docs/)
- [GORM Documentation](https://gorm.io/docs/)
- [golang-migrate](https://github.com/golang-migrate/migrate)
- [Viper Configuration](https://github.com/spf13/viper)

---

**End of Sprint 4.1 Plan**
