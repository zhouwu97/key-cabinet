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
```

#### Reservations (Sprint 4.4+)
```http
POST /api/v1/reservations
GET /api/v1/reservations
GET /api/v1/reservations/:id
DELETE /api/v1/reservations/:id
```

#### Operations (Sprint 4.6+)
```http
POST /api/v1/operations/pickup
POST /api/v1/operations/return
GET /api/v1/operations/:id
```

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
# Server
PORT=8080

# Database
DATABASE_URL=postgresql://user:pass@localhost:5432/keycabinet?sslmode=disable

# JWT
JWT_SECRET=your-secret-key
JWT_EXPIRATION=86400

# WeChat (Sprint 4.2+)
WECHAT_APP_ID=your-app-id
WECHAT_APP_SECRET=your-app-secret
```

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
