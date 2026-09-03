# Key Cabinet Backend - v0.4 Sprint Plan

## Overview

The backend will be built in 8 sprints, progressively implementing the full reservation and borrowing system. Each sprint delivers working, tested functionality.

## Technology Stack

- **Language**: Go 1.23+
- **Web Framework**: Gin
- **ORM**: GORM
- **Database**: PostgreSQL 14+
- **Auth**: JWT (self-issued after WeChat code2Session)
- **Device Gateway**: MockDeviceGateway (v0.4) → MQTTDeviceGateway (v0.5)
- **Migration**: golang-migrate

## Sprint Breakdown

### Sprint 4.1 — Backend Foundation ✅ COMPLETED
**Duration**: 2-3 days  
**Status**: Completed 2026-09-03

**Goals**:
- Project structure and configuration
- Database connection and migrations
- Unified error handling
- JWT middleware
- Core interfaces
- Health check API

**Deliverables**:
- ✅ Go module with proper structure
- ✅ Config management (YAML)
- ✅ PostgreSQL connection with GORM
- ✅ Migration files (init + reservation constraint)
- ✅ Platform layer (errors, JWT, clock, idgen)
- ✅ HTTP router with middleware
- ✅ Repository interfaces
- ✅ DeviceGateway interface + MockDeviceGateway
- ✅ Health check endpoint
- ✅ 15+ unit tests passing

**Verification**:
```bash
✅ go test ./... -short
✅ GET /health → 200 OK
```

---

### Sprint 4.2 — Auth + User + Identity
**Duration**: 2-3 days  
**Status**: Planned

**Goals**:
- WeChat login integration
- User and identity management
- JWT issuance
- Protected routes working

**Tasks**:
1. Implement `internal/infrastructure/wechat/client.go`
   - `Code2Session(code) → (openid, session_key)`
2. Implement GORM repositories:
   - `postgres.UserRepository`
   - `postgres.UserIdentityRepository`
3. Implement `service.AuthService`:
   - `WechatLogin(code) → (User, JWT)`
   - `GetUserByID(userID) → User`
4. Implement handlers:
   - `POST /api/v1/auth/wechat-login`
   - `GET /api/v1/me`
5. Domain models:
   - `domain/user/user.go`
   - `domain/user/identity.go`

**Tests**:
- B001: User login with valid WeChat code
- B002: Access `/me` without JWT → 401
- B003: Access `/me` with valid JWT → user info
- B004: Invalid WeChat code → 401
- B005: Create user on first login
- B006: Return existing user on subsequent login

**Deliverables**:
- WeChat integration working
- User can login and get JWT
- Protected routes verified with JWT

---

### Sprint 4.3 — Key + Slot + Device
**Duration**: 2 days  
**Status**: Planned

**Goals**:
- Key management
- Device and slot tracking
- Read-only key API

**Tasks**:
1. Domain models:
   - `domain/key/key.go` (with status machine: AVAILABLE, BORROWED, MAINTENANCE)
   - `domain/device/device.go`
   - `domain/device/slot.go`
2. Implement GORM repositories:
   - `postgres.KeyRepository`
   - `postgres.DeviceRepository`
   - `postgres.SlotRepository`
3. Implement `service.KeyService`:
   - `GetAllKeys() → []Key`
   - `GetKeyByID(id) → Key`
   - `GetAvailableKeys() → []Key`
4. Implement handlers:
   - `GET /api/v1/keys`
   - `GET /api/v1/keys/:id`
   - `GET /api/v1/devices`

**Tests**:
- K001: Get all keys
- K002: Get key by ID
- K003: Get available keys only
- K004: Key not found → 404

**Deliverables**:
- Key domain with status machine
- Key listing API
- Device and slot models ready for operations

---

### Sprint 4.4 — Reservation + DB Constraint
**Duration**: 3 days  
**Status**: Planned

**Goals**:
- Reservation CRUD
- Time conflict detection at database level
- Concurrent reservation handling

**Tasks**:
1. Domain model:
   - `domain/reservation/reservation.go` (status: PENDING, ACTIVE, USED, CANCELLED, EXPIRED)
   - `CanPickup(now) → error`
   - `CanCancel() → error`
2. Implement `postgres.ReservationRepository`:
   - `Create()` - will fail if time conflict due to DB constraint
   - `FindByID()`
   - `FindByUserID()`
   - `FindConflicts()` - for business layer check
   - `Update()`
3. Implement `service.ReservationService`:
   - `CreateReservation(userID, keyID, startTime, endTime) → Reservation`
   - `GetUserReservations(userID) → []Reservation`
   - `CancelReservation(reservationID) → error`
4. Handlers:
   - `POST /api/v1/reservations`
   - `GET /api/v1/reservations` (user's own)
   - `GET /api/v1/reservations/:id`
   - `DELETE /api/v1/reservations/:id` (cancel)

**Tests**:
- R001: Normal reservation creation
- R002: Time overlap → 409 Conflict
- R003: Two goroutines create overlapping reservations → only one succeeds
- R004: Cancel ACTIVE reservation → success
- R005: Cancel USED reservation → 422 Invalid State
- R006: Reserve unavailable key → 422
- R007: Reserve with invalid time range → 400

**Deliverables**:
- Reservation domain with state machine
- DB constraint prevents time conflicts
- Concurrent creation tested

---

### Sprint 4.5 — BorrowRecord
**Duration**: 2 days  
**Status**: Planned

**Goals**:
- Borrow record tracking
- RFID verification support
- Return time tracking

**Tasks**:
1. Domain model:
   - `domain/borrow/borrow_record.go` (status: BORROWED, RETURNED, OVERDUE)
   - `IsOverdue(now) → bool`
   - `CanReturn() → error`
2. Implement `postgres.BorrowRepository`
3. Implement `service.BorrowService`:
   - `CreateBorrowRecord(reservation) → BorrowRecord`
   - `CompleteBorrowRecord(recordID, rfidVerified) → error`
   - `GetUserBorrowHistory(userID) → []BorrowRecord`
4. Handlers:
   - `GET /api/v1/borrows` (user's history)
   - `GET /api/v1/borrows/:id`

**Tests**:
- B101: Create borrow record
- B102: Complete borrow with RFID verified
- B103: Complete borrow without RFID verification
- B104: Check overdue status

**Deliverables**:
- Borrow record domain
- History tracking
- Overdue detection

---

### Sprint 4.6 — DeviceOperation + MockGateway Integration
**Duration**: 3-4 days  
**Status**: Planned

**Goals**:
- Device operation state machine
- MockDeviceGateway event handling
- Full pickup/return transaction

**Tasks**:
1. Domain model:
   - `domain/operation/device_operation.go`
   - Status: CREATED, AUTHORIZED, SENT, EXECUTING, SUCCESS, FAILED
2. Implement `postgres.OperationRepository`
3. Implement `service.OperationService` (implements `DeviceEventHandler`):
   - `InitiatePickup(reservationID) → DeviceOperation`
   - `InitiateReturn(borrowRecordID) → DeviceOperation`
   - `OnPickupSuccess(event) → error`
   - `OnPickupFailed(event) → error`
   - `OnReturnSuccess(event) → error`
   - `OnReturnFailed(event) → error`
4. Transaction implementation:
   - Pickup transaction: Reservation→USED, BorrowRecord→BORROWED, Key→BORROWED, Slot→ABSENT
   - Return transaction: BorrowRecord→RETURNED, Key→AVAILABLE, Slot→PRESENT
5. Handlers:
   - `POST /api/v1/operations/pickup`
   - `POST /api/v1/operations/return`
   - `GET /api/v1/operations/:id`
6. Wire up MockDeviceGateway with OperationService

**Tests**:
- O001: ACTIVE reservation → pickup → success
- O002: No reservation → pickup → 422
- O003: Same reservation → duplicate pickup → idempotent
- O004: Concurrent operations on same device → 409
- O005: Operation failure → no database changes
- O006: Pickup success → all entities updated in one transaction
- O007: Return success → all entities updated in one transaction
- O008: RFID mismatch → return fails, state stays BORROWED

**Deliverables**:
- Full device operation state machine
- Event-driven callbacks working
- Transactional integrity verified

---

### Sprint 4.7 — Frontend Integration
**Duration**: 2-3 days  
**Status**: Planned

**Goals**:
- Remove all mock services from frontend
- Connect frontend to real backend
- DTO mapping on frontend
- Polling for operation status

**Tasks**:
1. Frontend:
   - Implement `services/http/AuthHttpService.ts`
   - Implement `services/http/KeyHttpService.ts`
   - Implement `services/http/ReservationHttpService.ts`
   - Implement `services/http/OperationHttpService.ts`
   - Implement DTO → Domain mappers
   - Remove all Mock services from business code
2. Backend:
   - Add CORS middleware
   - Verify all DTO formats match frontend contract

**Tests**:
- Frontend E2E:
  - Login → Get keys → Create reservation → Pickup → Return
- Contract tests:
  - Verify all API responses match v0.3.1 contract

**Deliverables**:
- Frontend fully connected to backend
- Mock services removed
- Polling working

---

### Sprint 4.8 — Full System Verification
**Duration**: 2 days  
**Status**: Planned

**Goals**:
- End-to-end system testing
- Performance testing
- Documentation
- Deployment preparation

**Tasks**:
1. Integration tests:
   - Full user journey: login → reserve → pickup → return
   - Concurrent user scenarios
   - Edge cases and error paths
2. Performance tests:
   - 10 concurrent users creating reservations
   - Verify no duplicate reservations created
3. Documentation:
   - API documentation (OpenAPI/Swagger)
   - Deployment guide
   - Troubleshooting guide
4. Deployment:
   - Docker compose setup
   - Environment variable documentation
   - Health check verification

**Tests**:
- E2E001: Complete user journey
- E2E002: 10 concurrent reservations on same key → only 1 succeeds per time slot
- E2E003: Operation status polling
- E2E004: Error handling across all endpoints

**Deliverables**:
- Fully tested system
- Complete documentation
- Ready for deployment

---

## Success Criteria

### Technical
- ✅ All tests passing (`go test ./...`)
- ✅ No mock services in frontend business code
- ✅ Database constraints prevent conflicts
- ✅ Transactions ensure data consistency
- ✅ JWT authentication working
- ✅ Event-driven device gateway

### Functional
- User can login with WeChat
- User can view available keys
- User can create/cancel reservations
- User can pickup keys (MockDeviceGateway simulates success)
- User can return keys
- System prevents conflicting reservations
- System tracks borrow history

### Quality
- Test coverage > 70%
- All endpoints documented
- Error messages clear and actionable
- Response times < 200ms (excluding device operations)
- Database migrations versioned and reversible

## Risk Management

### Risk 1: WeChat API Integration
**Mitigation**: Test with WeChat sandbox environment first

### Risk 2: Database Constraint Complexity
**Mitigation**: Thoroughly test PostgreSQL exclusion constraint; have fallback to application-level locking

### Risk 3: Event-Driven Complexity
**Mitigation**: MockDeviceGateway is synchronous for v0.4; MQTT complexity deferred to v0.5

### Risk 4: Frontend-Backend Contract
**Mitigation**: Contract tests in Sprint 4.7; freeze API contract after v0.3.1

## Timeline

Total: 16-21 days (~3-4 weeks)

- Sprint 4.1: ✅ 2 days (Completed)
- Sprint 4.2: 2-3 days
- Sprint 4.3: 2 days
- Sprint 4.4: 3 days
- Sprint 4.5: 2 days
- Sprint 4.6: 3-4 days
- Sprint 4.7: 2-3 days
- Sprint 4.8: 2 days

## Next Steps

1. Review Sprint 4.1 completion
2. Begin Sprint 4.2: Auth + User + Identity
3. Set up WeChat developer account for testing
4. Prepare test data for integration tests
