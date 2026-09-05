# Backend README

## Project Structure

```
server/
├── cmd/
│   ├── api/              # API server entry point
│   └── migrate/          # Database migration tool
├── internal/
│   ├── config/           # Configuration management
│   ├── domain/           # Domain models and business logic
│   ├── service/          # Application services
│   ├── repository/       # Data access interfaces
│   ├── transport/
│   │   └── http/         # HTTP handlers, middleware, router
│   ├── infrastructure/
│   │   ├── postgres/     # PostgreSQL implementation
│   │   ├── wechat/       # WeChat API integration
│   │   └── device/       # Device gateway (Mock/MQTT)
│   └── platform/
│       ├── errors/       # Unified error handling
│       ├── jwt/          # JWT token service
│       ├── clock/        # Time abstraction
│       └── idgen/        # ID generation
├── migrations/           # SQL migration files
└── tests/
    ├── integration/      # Integration tests
    └── contract/         # Contract tests
```

## Quick Start

### Prerequisites

- Go 1.23+
- PostgreSQL 14+
- Make (optional)

### Setup

1. Install dependencies:
```bash
go mod download
```

2. Create database:
```bash
createdb keycabinet
```

3. Run migrations:
```bash
go run cmd/migrate/main.go -command up
```

4. Copy config:
```bash
cp internal/config/config.example.yaml internal/config/config.yaml
# Edit config.yaml with your settings
```

5. Start server:
```bash
go run cmd/api/main.go
```

The server will start on `http://localhost:8080`

### Verify Installation

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "ok",
  "database": "ok",
  "timestamp": "2026-09-03T10:30:00Z"
}
```

## Development

### Running Tests

```bash
# All tests (requires PostgreSQL)
go test ./...

# Unit tests only (no database)
go test ./... -short

# Specific package
go test ./internal/platform/jwt/... -v

# With coverage
go test ./... -cover
```

### Database Migrations

```bash
# Apply migrations
go run cmd/migrate/main.go -command up

# Rollback migrations
go run cmd/migrate/main.go -command down

# Check version
go run cmd/migrate/main.go -command version
```

### Code Quality

```bash
# Format code
go fmt ./...

# Run linter
go vet ./...

# Run static analysis (if installed)
staticcheck ./...
```

## API Documentation

### Authentication

All protected endpoints require JWT token:

```bash
Authorization: Bearer <token>
```

### Endpoints

#### Health Check
```http
GET /health
```

#### Auth (Sprint 4.2+)
```http
POST /api/v1/auth/wechat-login
GET /api/v1/me
```

#### Keys (Sprint 4.3+)
```http
GET /api/v1/keys
GET /api/v1/keys/:id
GET /api/v1/keys/:id/slot
GET /api/v1/keys/:id/availability
```

#### Reservations (Sprint 4.4+)
```http
POST /api/v1/reservations
GET /api/v1/me/reservations
GET /api/v1/reservations/:id
POST /api/v1/reservations/:id/cancel
```

#### Borrow Records
```http
GET /api/v1/me/borrow-records
GET /api/v1/borrow-records/:id
```

#### Device Operations (Sprint 4.6+)
```http
POST /api/v1/device-operations/pickup
POST /api/v1/device-operations/return
GET /api/v1/device-operations/active
GET /api/v1/device-operations/:id
POST /api/v1/device-operations/:id/cancel
```

取钥匙和还钥匙接口返回 `202 Accepted`，客户端应使用返回的操作 ID 轮询操作状态；`clientRequestId` 用于重复提交幂等。

## Architecture Decisions

### Repository Pattern
- Interfaces in `internal/repository/`
- GORM implementations in `internal/infrastructure/postgres/`
- Enables testing with mocks

### Device Gateway
- Interface: `internal/infrastructure/device/gateway.go`
- Mock implementation for v0.4
- MQTT implementation for v0.5
- Event-driven callbacks for async operations

### Error Handling
- Unified `AppError` with error codes
- HTTP status code mapping in middleware
- Consistent error response format

### JWT Authentication
- HS256 signing algorithm
- Claims: user_id, role
- Configurable expiration

### Database Constraints
- PostgreSQL exclusion constraint for reservation conflicts
- Prevents overlapping time ranges at database level
- Better than application-level checks

## Environment Variables

```bash
# Runtime environment. Mock login is only valid outside production and must be explicit.
KC_APP_ENV=development
KC_WECHAT_MOCK_ENABLED=true
KC_DEVICE_GATEWAY_TYPE=mock

# Server
KC_SERVER_PORT=8080

# Database
KC_DATABASE_HOST=localhost
KC_DATABASE_PORT=5432
KC_DATABASE_USER=postgres
KC_DATABASE_PASSWORD=postgres
KC_DATABASE_DBNAME=keycabinet
KC_DATABASE_SSLMODE=disable

# JWT
KC_JWT_SECRET=your-secret-key
KC_JWT_EXPIRATION=86400

# WeChat (Sprint 4.2+)
KC_WECHAT_APP_ID=your-app-id
KC_WECHAT_APP_SECRET=your-app-secret
```

开发环境当前提供 Mock 微信登录和 Mock 设备网关；接入真实设备前需配置网关实现并关闭 Mock。借还逾期状态由 API 进程内定时任务每 5 分钟检查一次。

`APP_ENV`、`JWT_SECRET` 等未加 `KC_` 前缀的变量仍兼容读取，但部署配置统一使用 `KC_` 前缀。生产环境必须关闭 `KC_WECHAT_MOCK_ENABLED`，并提供真实微信凭据和非占位 JWT 密钥，否则服务会拒绝启动。

## Testing Strategy

### Unit Tests
- Test business logic in isolation
- Mock external dependencies
- Fast, no database required

### Integration Tests
- Test with real database
- Use test database: `keycabinet_test`
- Skipped in short mode

### Contract Tests
- Verify API contracts
- Ensure frontend compatibility
- Run before releases

## Contributing

### Code Style
- Follow Go conventions
- Use `gofmt` for formatting
- Keep functions small and focused
- Write tests for new features

### Commit Messages
```
type(scope): subject

body

footer
```

Types: feat, fix, docs, refactor, test, chore

### Branch Strategy
- `main` - production-ready code
- `develop` - integration branch
- `feature/*` - feature branches
- `bugfix/*` - bug fix branches

## Troubleshooting

### Database Connection Failed
```bash
# Check PostgreSQL is running
pg_isready

# Verify connection string
psql postgresql://user:pass@localhost:5432/keycabinet
```

### Migration Failed
```bash
# Check current version
go run cmd/migrate/main.go -command version

# Force version (use carefully)
go run cmd/migrate/main.go -command force -version 1
```

### Tests Failing
```bash
# Run verbose to see details
go test ./... -v

# Check database is available for integration tests
go test ./tests/integration/... -v
```

## Resources

- [Go Documentation](https://go.dev/doc/)
- [Gin Framework](https://gin-gonic.com/)
- [GORM](https://gorm.io/)
- [PostgreSQL](https://www.postgresql.org/docs/)
- [golang-migrate](https://github.com/golang-migrate/migrate)

## License

MIT
