## Sprint 4.1 — Backend Foundation

### Completed Date
2026-09-03

### Sprint Goal
Establish backend foundation: project structure, configuration, database connection, unified error handling, JWT middleware, core interfaces, and health check API.

### Completed Tasks

#### 1. Project Structure
- ✅ Created Go module with proper directory layout
- ✅ Established `cmd/api` for main entry point
- ✅ Established `cmd/migrate` for database migrations
- ✅ Created `internal/` with domain-driven structure:
  - `config/` - Configuration management
  - `domain/` - Domain models (prepared for Sprint 4.2+)
  - `service/` - Application services (prepared for Sprint 4.2+)
  - `repository/` - Repository interfaces
  - `transport/http/` - HTTP handlers, middleware, router
  - `infrastructure/` - PostgreSQL, device gateway
  - `platform/` - Cross-cutting concerns (errors, JWT, clock, idgen)
- ✅ Created `migrations/` for SQL migrations
- ✅ Created `tests/integration/` for integration tests

#### 2. Configuration Management
- ✅ Implemented YAML-based configuration (`internal/config/`)
- ✅ Support for server, database, and JWT settings
- ✅ Unit tests for configuration loading

#### 3. Database Infrastructure
- ✅ PostgreSQL connection with GORM (`internal/infrastructure/postgres/`)
- ✅ Connection pooling configuration
- ✅ Health check function
- ✅ Initial migration files:
  - `000001_init.up.sql` - Core tables (users, user_identities, keys, devices, slots, reservations, borrow_records, device_operations)
  - `000002_reservation_constraint.up.sql` - PostgreSQL exclusion constraint for reservation time conflicts
- ✅ Migration tool (`cmd/migrate/main.go`)

#### 4. Platform Layer
- ✅ Unified error handling (`internal/platform/errors/`)
  - Error codes: INVALID_INPUT, UNAUTHORIZED, FORBIDDEN, NOT_FOUND, CONFLICT, etc.
  - `AppError` with code, message, and cause
  - Wrap and unwrap functionality
- ✅ JWT token service (`internal/platform/jwt/`)
  - Token generation with user ID and role
  - Token validation
  - Custom claims structure
- ✅ Clock abstraction (`internal/platform/clock/`)
- ✅ ID generator (`internal/platform/idgen/`)

#### 5. Transport Layer
- ✅ Gin router setup (`internal/transport/http/router.go`)
  - Route groups: `/health`, `/api/v1` (public), `/api/v1` (protected), `/admin`
- ✅ Middleware:
  - `AuthMiddleware` - JWT authentication
  - `AdminMiddleware` - Role-based authorization
  - `ErrorMiddleware` - Unified error response
- ✅ DTO structures (`internal/transport/http/dto/`)
  - Unified error response format
- ✅ Health check handler (`internal/transport/http/handler/health_handler.go`)

#### 6. Repository Interfaces
- ✅ `UserRepository` - User and identity management
- ✅ `KeyRepository` - Key operations
- ✅ `ReservationRepository` - Reservation CRUD and conflict detection
- ✅ `BorrowRepository` - Borrow record management
- ✅ `OperationRepository` - Device operation tracking

#### 7. Device Gateway
- ✅ `DeviceGateway` interface (`internal/infrastructure/device/gateway.go`)
  - Commands: `StartPickup`, `StartReturn`, `GetDeviceStatus`
  - Event handler: `DeviceEventHandler` for async callbacks
- ✅ `MockDeviceGateway` implementation
  - Simulates device operations with 100ms delay
  - Fires success events via registered handler

#### 8. Testing
- ✅ Unit tests:
  - Config loading
  - Error handling (5 test cases)
  - JWT generation and validation (3 test cases)
  - Auth middleware (6 test cases: no token, valid token, invalid token, invalid format, student/admin access)
- ✅ Integration test setup:
  - Health check tests (skipped in short mode when DB unavailable)
  - Test database setup/teardown utilities
- ✅ All tests pass with `go test ./... -short`

### Dependencies
```
- github.com/gin-gonic/gin v1.10.0
- github.com/golang-jwt/jwt/v5 v5.2.1
- github.com/google/uuid v1.6.0
- github.com/stretchr/testify v1.10.0
- gopkg.in/yaml.v3 v3.0.1
- gorm.io/driver/postgres v1.5.11
- gorm.io/gorm v1.25.12
- github.com/golang-migrate/migrate/v4 v4.19.1
```

### Verification Results

#### 1. Project Builds Successfully
```bash
✅ go mod tidy
✅ go build ./cmd/api
✅ go build ./cmd/migrate
```

#### 2. All Tests Pass
```bash
✅ go test ./... -short
```

Test Results:
- `internal/config` - PASS
- `internal/infrastructure/postgres` - PASS (1 skip: requires PostgreSQL)
- `internal/platform/errors` - PASS (5/5 tests)
- `internal/platform/jwt` - PASS (3/3 tests)
- `tests/integration` - PASS (6/6 auth middleware tests, 2 health check tests skipped in short mode)

#### 3. Code Quality
```bash
✅ go fmt ./...
✅ go vet ./...
```

### Deliverables

#### Architecture
- ✅ `docs/BACKEND-ARCHITECTURE.md` - Complete backend architecture documentation
- ✅ `docs/sprints/SPRINT-4.1-PLAN.md` - Sprint 4.1 implementation plan

#### Code
- ✅ 50+ files created across the backend structure
- ✅ Core interfaces defined for all major components
- ✅ MockDeviceGateway implements event-driven device interaction pattern

#### Database
- ✅ 2 migration files with up/down scripts
- ✅ PostgreSQL exclusion constraint for reservation time conflict prevention
- ✅ Migration tool ready for production use

#### Testing
- ✅ 15+ unit tests
- ✅ Integration test framework
- ✅ All tests passing in CI-friendly mode (`-short`)

### Next Steps (Sprint 4.2)

The foundation is complete. Sprint 4.2 will implement:
1. Wechat authentication flow
2. User identity management
3. Auth endpoints: `POST /api/v1/auth/wechat-login`, `GET /api/v1/me`
4. GORM repository implementations for User and UserIdentity
5. Integration tests for complete auth flow

### Notes

- Database tests are skipped in short mode when PostgreSQL is unavailable
- The event-driven device gateway pattern is ready for MQTT integration in v0.5
- All repository interfaces use context for cancellation support
- Error handling follows the contract: HTTP status codes map to error codes
- JWT tokens include user_id and role claims for authorization
