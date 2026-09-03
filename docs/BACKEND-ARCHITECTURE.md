# Backend Architecture

**Version**: v0.4.0  
**Status**: Approved  
**Last Updated**: 2026-09-03

## Overview

本文档定义智能钥匙柜后端服务的技术架构、核心设计决策及实现规范。

v0.4 目标：**建立完整的 Go 后端服务，成为系统唯一的授权中心与数据源，使小程序完全退出 Mock 数据依赖。**

---

## Technology Stack

| Component         | Technology              | Rationale                                                                 |
|-------------------|-------------------------|---------------------------------------------------------------------------|
| Language          | **Go 1.22+**            | 原生并发支持、标准库完善、官方推荐 `cmd/ + internal/` 布局                   |
| HTTP Framework    | **Gin**                 | 路由分组、中间件、JSON 校验，适合 REST API 与 JWT 认证                      |
| ORM               | **GORM**                | 事务支持、`FOR UPDATE` 行锁、Repository 实现                                |
| Database          | **PostgreSQL 15+**      | `tstzrange` + `EXCLUDE USING gist` 直接保证预约时间不重叠                   |
| Migration         | **golang-migrate/migrate** | 版本化 SQL Migration，避免 `AutoMigrate` 在生产环境的不可控性             |
| Auth              | **JWT (golang-jwt)**    | 微信 code → Backend → 自有 JWT，支持前端无状态认证                          |
| Device (v0.4)     | **MockDeviceGateway**   | 模拟设备事件，验证业务流程完整性                                              |
| Device (v0.5+)    | MQTTDeviceGateway       | 接入真实 ESP32 设备                                                        |

### Why Go + Gin + GORM + PostgreSQL?

1. **Go**：适合状态机、事务、并发预约、设备事件的服务端场景
2. **Gin**：路由分组天然对应 `/auth/*`、`/api/v1/*`、`/admin/*`
3. **GORM**：事务 + `FOR UPDATE` 适合取钥授权、归还结算的并发更新
4. **PostgreSQL**：`exclusion constraint` 直接解决"同一钥匙预约时间不重叠"问题

---

## Project Structure

```
key-cabinet/
├── miniprogram/              # 微信小程序
├── server/                   # Go 后端服务
│   ├── cmd/
│   │   ├── api/
│   │   │   └── main.go       # HTTP 服务入口
│   │   └── migrate/
│   │       └── main.go       # Migration 工具入口
│   │
│   ├── internal/
│   │   ├── config/           # 配置加载（env/yaml）
│   │   │   ├── config.go
│   │   │   └── config.yaml
│   │   │
│   │   ├── domain/           # Domain Model + Domain Logic
│   │   │   ├── user/
│   │   │   │   ├── user.go
│   │   │   │   └── identity.go
│   │   │   ├── key/
│   │   │   │   ├── key.go
│   │   │   │   ├── slot.go
│   │   │   │   └── device.go
│   │   │   ├── reservation/
│   │   │   │   ├── reservation.go
│   │   │   │   └── status.go
│   │   │   ├── borrow/
│   │   │   │   ├── borrow_record.go
│   │   │   │   └── rfid_verification.go
│   │   │   └── operation/
│   │   │       ├── device_operation.go
│   │   │       ├── device_command.go
│   │   │       └── device_event.go
│   │   │
│   │   ├── service/          # Application Service（编排 Domain + Repository + Gateway）
│   │   │   ├── auth_service.go
│   │   │   ├── user_service.go
│   │   │   ├── key_service.go
│   │   │   ├── reservation_service.go
│   │   │   ├── borrow_service.go
│   │   │   └── operation_service.go
│   │   │
│   │   ├── repository/       # Repository Interface + GORM Implementation
│   │   │   ├── user_repository.go
│   │   │   ├── key_repository.go
│   │   │   ├── reservation_repository.go
│   │   │   ├── borrow_repository.go
│   │   │   └── operation_repository.go
│   │   │
│   │   ├── transport/        # HTTP 传输层
│   │   │   └── http/
│   │   │       ├── handler/
│   │   │       │   ├── auth_handler.go
│   │   │       │   ├── user_handler.go
│   │   │       │   ├── key_handler.go
│   │   │       │   ├── reservation_handler.go
│   │   │       │   ├── borrow_handler.go
│   │   │       │   └── operation_handler.go
│   │   │       ├── middleware/
│   │   │       │   ├── auth.go
│   │   │       │   ├── error.go
│   │   │       │   └── logger.go
│   │   │       ├── dto/
│   │   │       │   ├── auth_dto.go
│   │   │       │   ├── user_dto.go
│   │   │       │   ├── key_dto.go
│   │   │       │   ├── reservation_dto.go
│   │   │       │   ├── borrow_dto.go
│   │   │       │   └── operation_dto.go
│   │   │       ├── mapper/
│   │   │       │   └── reservation_mapper.go
│   │   │       └── router.go
│   │   │
│   │   ├── infrastructure/   # 外部系统适配器
│   │   │   ├── postgres/
│   │   │   │   ├── db.go
│   │   │   │   └── transaction.go
│   │   │   ├── wechat/
│   │   │   │   └── client.go
│   │   │   └── device/
│   │   │       ├── gateway.go          # DeviceGateway Interface
│   │   │       ├── mock_gateway.go     # v0.4 Mock 实现
│   │   │       └── mqtt_gateway.go     # v0.5 MQTT 实现
│   │   │
│   │   └── platform/         # 基础设施工具
│   │       ├── jwt/
│   │       │   ├── token.go
│   │       │   └── claims.go
│   │       ├── clock/
│   │       │   └── clock.go
│   │       ├── idgen/
│   │       │   └── snowflake.go
│   │       └── errors/
│   │           ├── error.go
│   │           └── codes.go
│   │
│   ├── migrations/           # 版本化 SQL Migration
│   │   ├── 000001_init.up.sql
│   │   ├── 000001_init.down.sql
│   │   ├── 000002_reservation_constraint.up.sql
│   │   └── 000002_reservation_constraint.down.sql
│   │
│   ├── tests/
│   │   ├── integration/      # 集成测试
│   │   │   ├── auth_test.go
│   │   │   ├── reservation_test.go
│   │   │   └── borrow_test.go
│   │   └── contract/         # 契约测试（验证 API 契约）
│   │       └── api_contract_test.go
│   │
│   ├── go.mod
│   ├── go.sum
│   └── Makefile
│
└── docs/
    ├── BACKEND-ARCHITECTURE.md       # 本文档
    ├── API-SPECIFICATION.md          # API 契约
    └── sprints/
        └── SPRINT-4.1-PLAN.md
```

---

## Layered Architecture

### 1. Domain Layer (`internal/domain/`)

**职责**：领域模型 + 领域逻辑（状态机、业务规则、领域事件）

#### 示例：`domain/reservation/reservation.go`

```go
package reservation

import (
    "time"
    "github.com/yourusername/key-cabinet/server/internal/platform/errors"
)

type ReservationStatus string

const (
    StatusPending   ReservationStatus = "PENDING"
    StatusActive    ReservationStatus = "ACTIVE"
    StatusUsed      ReservationStatus = "USED"
    StatusCancelled ReservationStatus = "CANCELLED"
    StatusExpired   ReservationStatus = "EXPIRED"
)

type Reservation struct {
    ID        string
    UserID    string
    KeyID     string
    Status    ReservationStatus
    StartTime time.Time
    EndTime   time.Time
    Purpose   string
    CreatedAt time.Time
    UpdatedAt time.Time
}

// CanPickup 领域逻辑：检查预约是否可以取钥
func (r *Reservation) CanPickup(now time.Time) error {
    if r.Status != StatusActive {
        return errors.New(errors.CodeInvalidState, "预约状态必须为 ACTIVE")
    }
    if now.Before(r.StartTime) {
        return errors.New(errors.CodeTooEarly, "未到预约开始时间")
    }
    if now.After(r.EndTime) {
        return errors.New(errors.CodeExpired, "预约已过期")
    }
    return nil
}

// CanCancel 领域逻辑：检查预约是否可以取消
func (r *Reservation) CanCancel() error {
    if r.Status == StatusUsed {
        return errors.New(errors.CodeInvalidState, "已使用的预约不能取消")
    }
    if r.Status == StatusCancelled {
        return errors.New(errors.CodeInvalidState, "预约已取消")
    }
    return nil
}

// MarkAsUsed 领域逻辑：标记预约为已使用
func (r *Reservation) MarkAsUsed() {
    r.Status = StatusUsed
    r.UpdatedAt = time.Now()
}
```

**原则**：
- Domain Model 是**富模型**，不是贫血的 struct
- 业务规则放在 Domain，不要分散到 Service
- Domain 不依赖任何基础设施（DB、HTTP、MQTT）

---

### 2. Service Layer (`internal/service/`)

**职责**：应用服务，编排 Domain + Repository + Gateway

#### 示例：`service/reservation_service.go`

```go
package service

import (
    "context"
    "time"
    "github.com/yourusername/key-cabinet/server/internal/domain/reservation"
    "github.com/yourusername/key-cabinet/server/internal/repository"
    "github.com/yourusername/key-cabinet/server/internal/platform/errors"
)

type ReservationService struct {
    reservationRepo repository.ReservationRepository
    keyRepo         repository.KeyRepository
    userRepo        repository.UserRepository
}

func NewReservationService(
    reservationRepo repository.ReservationRepository,
    keyRepo repository.KeyRepository,
    userRepo repository.UserRepository,
) *ReservationService {
    return &ReservationService{
        reservationRepo: reservationRepo,
        keyRepo:         keyRepo,
        userRepo:        userRepo,
    }
}

func (s *ReservationService) CreateReservation(ctx context.Context, req CreateReservationRequest) (*reservation.Reservation, error) {
    // 1. 校验用户是否存在
    user, err := s.userRepo.FindByID(ctx, req.UserID)
    if err != nil {
        return nil, errors.Wrap(err, "用户不存在")
    }
    if user.Status != "ACTIVE" {
        return nil, errors.New(errors.CodeForbidden, "用户状态异常")
    }

    // 2. 校验钥匙是否存在且可用
    key, err := s.keyRepo.FindByID(ctx, req.KeyID)
    if err != nil {
        return nil, errors.Wrap(err, "钥匙不存在")
    }
    if key.Status != "AVAILABLE" {
        return nil, errors.New(errors.CodeConflict, "钥匙当前不可用")
    }

    // 3. 校验时间范围
    if req.StartTime.Before(time.Now()) {
        return nil, errors.New(errors.CodeInvalidInput, "预约开始时间不能早于当前时间")
    }
    if req.EndTime.Before(req.StartTime) {
        return nil, errors.New(errors.CodeInvalidInput, "结束时间不能早于开始时间")
    }

    // 4. 检查时间冲突（应用层友好提示）
    conflicts, err := s.reservationRepo.FindConflicts(ctx, req.KeyID, req.StartTime, req.EndTime)
    if err != nil {
        return nil, errors.Wrap(err, "检查冲突失败")
    }
    if len(conflicts) > 0 {
        return nil, errors.New(errors.CodeConflict, "预约时间与现有预约冲突")
    }

    // 5. 创建预约
    r := &reservation.Reservation{
        UserID:    req.UserID,
        KeyID:     req.KeyID,
        Status:    reservation.StatusActive,
        StartTime: req.StartTime,
        EndTime:   req.EndTime,
        Purpose:   req.Purpose,
    }

    // 6. 保存到数据库（数据库层会再次校验 exclusion constraint）
    if err := s.reservationRepo.Create(ctx, r); err != nil {
        return nil, errors.Wrap(err, "创建预约失败")
    }

    return r, nil
}
```

**原则**：
- Service 编排 Domain 对象，不包含业务逻辑
- Service 调用 Repository，不直接操作数据库
- Service 是事务边界

---

### 3. Repository Layer (`internal/repository/`)

**职责**：Repository Interface + GORM Implementation

#### Interface：`repository/reservation_repository.go`

```go
package repository

import (
    "context"
    "time"
    "github.com/yourusername/key-cabinet/server/internal/domain/reservation"
)

type ReservationRepository interface {
    Create(ctx context.Context, r *reservation.Reservation) error
    FindByID(ctx context.Context, id string) (*reservation.Reservation, error)
    FindConflicts(ctx context.Context, keyID string, startTime, endTime time.Time) ([]*reservation.Reservation, error)
    Update(ctx context.Context, r *reservation.Reservation) error
    Delete(ctx context.Context, id string) error
}
```

#### GORM Implementation：`repository/reservation_repository_impl.go`

```go
package repository

import (
    "context"
    "time"
    "gorm.io/gorm"
    "github.com/yourusername/key-cabinet/server/internal/domain/reservation"
)

type reservationRepositoryImpl struct {
    db *gorm.DB
}

func NewReservationRepository(db *gorm.DB) ReservationRepository {
    return &reservationRepositoryImpl{db: db}
}

func (r *reservationRepositoryImpl) Create(ctx context.Context, res *reservation.Reservation) error {
    return r.db.WithContext(ctx).Create(res).Error
}

func (r *reservationRepositoryImpl) FindConflicts(ctx context.Context, keyID string, startTime, endTime time.Time) ([]*reservation.Reservation, error) {
    var results []*reservation.Reservation
    err := r.db.WithContext(ctx).
        Where("key_id = ?", keyID).
        Where("status IN (?)", []string{"ACTIVE", "PENDING"}).
        Where("NOT (end_time <= ? OR start_time >= ?)", startTime, endTime).
        Find(&results).Error
    return results, err
}
```

**原则**：
- Repository 只负责数据持久化，不包含业务逻辑
- Interface 定义在 `repository/`，Implementation 也在 `repository/`
- 事务通过 `*gorm.DB` 传递，Service 层控制事务边界

---

### 4. Transport Layer (`internal/transport/http/`)

**职责**：HTTP 请求/响应处理、DTO 转换、路由定义

#### Handler：`transport/http/handler/reservation_handler.go`

```go
package handler

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/yourusername/key-cabinet/server/internal/service"
    "github.com/yourusername/key-cabinet/server/internal/transport/http/dto"
    "github.com/yourusername/key-cabinet/server/internal/transport/http/mapper"
)

type ReservationHandler struct {
    reservationService *service.ReservationService
}

func NewReservationHandler(reservationService *service.ReservationService) *ReservationHandler {
    return &ReservationHandler{reservationService: reservationService}
}

func (h *ReservationHandler) CreateReservation(c *gin.Context) {
    var req dto.CreateReservationRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, dto.ErrorResponse{
            Error: dto.ErrorDetail{
                Code:      "INVALID_INPUT",
                Message:   err.Error(),
                Timestamp: time.Now().Format(time.RFC3339),
            },
        })
        return
    }

    // DTO → Domain
    serviceReq := mapper.ToCreateReservationServiceRequest(req)

    // 调用 Service
    reservation, err := h.reservationService.CreateReservation(c.Request.Context(), serviceReq)
    if err != nil {
        // 统一错误处理在 middleware/error.go
        c.Error(err)
        return
    }

    // Domain → DTO
    resp := mapper.ToReservationResponse(reservation)
    c.JSON(http.StatusCreated, resp)
}
```

#### DTO：`transport/http/dto/reservation_dto.go`

```go
package dto

type CreateReservationRequest struct {
    KeyID     string `json:"keyId" binding:"required"`
    StartTime string `json:"startTime" binding:"required"` // ISO8601
    EndTime   string `json:"endTime" binding:"required"`   // ISO8601
    Purpose   string `json:"purpose"`
}

type ReservationResponse struct {
    ID        string `json:"id"`
    UserID    string `json:"userId"`
    KeyID     string `json:"keyId"`
    Status    string `json:"status"`
    StartTime string `json:"startTime"` // ISO8601
    EndTime   string `json:"endTime"`   // ISO8601
    Purpose   string `json:"purpose"`
    CreatedAt string `json:"createdAt"`
}
```

#### Mapper：`transport/http/mapper/reservation_mapper.go`

```go
package mapper

import (
    "time"
    "github.com/yourusername/key-cabinet/server/internal/domain/reservation"
    "github.com/yourusername/key-cabinet/server/internal/transport/http/dto"
)

func ToReservationResponse(r *reservation.Reservation) dto.ReservationResponse {
    return dto.ReservationResponse{
        ID:        r.ID,
        UserID:    r.UserID,
        KeyID:     r.KeyID,
        Status:    string(r.Status),
        StartTime: r.StartTime.Format(time.RFC3339),
        EndTime:   r.EndTime.Format(time.RFC3339),
        Purpose:   r.Purpose,
        CreatedAt: r.CreatedAt.Format(time.RFC3339),
    }
}
```

**原则**：
- Handler 只做：解析 DTO → 调用 Service → 返回 DTO
- 严格：`API DTO → Mapper → Domain Model`，三者分离
- 不要让 `API DTO == DB Entity == Domain Model`

---

### 5. Infrastructure Layer (`internal/infrastructure/`)

**职责**：外部系统适配器（微信、设备网关、PostgreSQL）

#### DeviceGateway Interface：`infrastructure/device/gateway.go`

```go
package device

import (
    "context"
    "github.com/yourusername/key-cabinet/server/internal/domain/operation"
)

// DeviceGateway 设备网关接口
type DeviceGateway interface {
    // StartPickup 发起取钥命令
    StartPickup(ctx context.Context, cmd operation.DeviceCommand) error

    // StartReturn 发起归还命令
    StartReturn(ctx context.Context, cmd operation.DeviceCommand) error

    // GetDeviceStatus 获取设备状态
    GetDeviceStatus(ctx context.Context, deviceID string) (*DeviceStatus, error)

    // RegisterEventHandler 注册设备事件处理器
    RegisterEventHandler(handler DeviceEventHandler)
}

// DeviceEventHandler 设备事件处理器
type DeviceEventHandler interface {
    OnPickupSuccess(ctx context.Context, event operation.DeviceEvent) error
    OnPickupFailed(ctx context.Context, event operation.DeviceEvent) error
    OnReturnSuccess(ctx context.Context, event operation.DeviceEvent) error
    OnReturnFailed(ctx context.Context, event operation.DeviceEvent) error
}

type DeviceStatus struct {
    DeviceID  string
    Online    bool
    LastSeen  time.Time
}
```

#### MockDeviceGateway (v0.4)：`infrastructure/device/mock_gateway.go`

```go
package device

import (
    "context"
    "time"
    "github.com/yourusername/key-cabinet/server/internal/domain/operation"
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

func (g *MockDeviceGateway) StartPickup(ctx context.Context, cmd operation.DeviceCommand) error {
    // 模拟设备处理延迟
    go func() {
        time.Sleep(100 * time.Millisecond)
        event := operation.DeviceEvent{
            OperationID: cmd.OperationID,
            DeviceID:    cmd.DeviceID,
            EventType:   "PICKUP_SUCCESS",
            Timestamp:   time.Now(),
        }
        if g.handler != nil {
            g.handler.OnPickupSuccess(context.Background(), event)
        }
    }()
    return nil
}

func (g *MockDeviceGateway) StartReturn(ctx context.Context, cmd operation.DeviceCommand) error {
    // 模拟设备处理延迟
    go func() {
        time.Sleep(100 * time.Millisecond)
        event := operation.DeviceEvent{
            OperationID: cmd.OperationID,
            DeviceID:    cmd.DeviceID,
            EventType:   "RETURN_SUCCESS",
            Timestamp:   time.Now(),
        }
        if g.handler != nil {
            g.handler.OnReturnSuccess(context.Background(), event)
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

**关键设计**：
- **v0.4**：MockDeviceGateway 在 `StartPickup` 后 100ms 调用 `OnPickupSuccess`
- **v0.5**：MQTTDeviceGateway 在收到 MQTT 消息时调用 `OnPickupSuccess`
- **业务代码完全不变**

---

## Concurrency Control Strategy

### 1. Reservation 时间冲突：PostgreSQL Exclusion Constraint

```sql
-- migrations/000002_reservation_constraint.up.sql
CREATE EXTENSION IF NOT EXISTS btree_gist;

ALTER TABLE reservations
ADD CONSTRAINT reservations_no_overlap
EXCLUDE USING gist (
    key_id WITH =,
    tstzrange(start_time, end_time) WITH &&
)
WHERE (status IN ('ACTIVE', 'PENDING'));
```

**效果**：
- 数据库自动拒绝时间重叠的预约
- 即使两个并发请求同时通过应用层检查，数据库也只允许一个成功
- 应用层仍然先检查冲突，提供友好提示

---

### 2. 取钥事务：Transaction + FOR UPDATE

```go
func (s *BorrowService) Pickup(ctx context.Context, req PickupRequest) error {
    return s.db.Transaction(func(tx *gorm.DB) error {
        // 1. 锁定预约
        var reservation reservation.Reservation
        if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
            Where("id = ?", req.ReservationID).
            First(&reservation).Error; err != nil {
            return err
        }

        // 2. 检查预约状态
        if err := reservation.CanPickup(time.Now()); err != nil {
            return err
        }

        // 3. 锁定钥匙
        var key key.Key
        if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
            Where("id = ?", reservation.KeyID).
            First(&key).Error; err != nil {
            return err
        }

        // 4. 检查不存在 Active Operation
        var count int64
        if err := tx.Model(&operation.DeviceOperation{}).
            Where("reservation_id = ?", req.ReservationID).
            Where("status IN (?)", []string{"AUTHORIZED", "SENT", "EXECUTING"}).
            Count(&count).Error; err != nil {
            return err
        }
        if count > 0 {
            return errors.New(errors.CodeConflict, "存在进行中的操作")
        }

        // 5. 创建 DeviceOperation
        op := &operation.DeviceOperation{
            ReservationID: req.ReservationID,
            DeviceID:      key.DeviceID,
            Type:          "PICKUP",
            Status:        "CREATED",
        }
        if err := tx.Create(op).Error; err != nil {
            return err
        }

        // 6. 发送设备命令
        cmd := operation.DeviceCommand{
            OperationID: op.ID,
            DeviceID:    key.DeviceID,
            SlotID:      key.SlotID,
            Type:        "PICKUP",
        }
        if err := s.deviceGateway.StartPickup(ctx, cmd); err != nil {
            return err
        }

        // 7. 更新 Operation 状态
        op.Status = "SENT"
        if err := tx.Save(op).Error; err != nil {
            return err
        }

        return nil
    })
}
```

**原则**：
- 一个数据库事务内完成：锁定 → 校验 → 创建 → 发送命令 → 更新状态
- 任何一步失败，整个事务回滚
- 不会出现"一半成功"的状态

---

### 3. 设备事件回调：更新状态

```go
func (h *BorrowEventHandler) OnPickupSuccess(ctx context.Context, event operation.DeviceEvent) error {
    return h.db.Transaction(func(tx *gorm.DB) error {
        // 1. 更新 Operation
        if err := tx.Model(&operation.DeviceOperation{}).
            Where("id = ?", event.OperationID).
            Updates(map[string]interface{}{
                "status":       "SUCCESS",
                "completed_at": event.Timestamp,
            }).Error; err != nil {
            return err
        }

        // 2. 获取 Operation
        var op operation.DeviceOperation
        if err := tx.First(&op, "id = ?", event.OperationID).Error; err != nil {
            return err
        }

        // 3. 更新 Reservation
        if err := tx.Model(&reservation.Reservation{}).
            Where("id = ?", op.ReservationID).
            Update("status", "USED").Error; err != nil {
            return err
        }

        // 4. 创建 BorrowRecord
        borrowRecord := &borrow.BorrowRecord{
            ReservationID: op.ReservationID,
            Status:        "BORROWED",
            BorrowedAt:    event.Timestamp,
        }
        if err := tx.Create(borrowRecord).Error; err != nil {
            return err
        }

        // 5. 更新 Key
        if err := tx.Model(&key.Key{}).
            Where("id = ?", op.KeyID).
            Update("status", "BORROWED").Error; err != nil {
            return err
        }

        // 6. 更新 Slot
        if err := tx.Model(&key.Slot).
            Where("id = ?", op.SlotID).
            Update("status", "ABSENT").Error; err != nil {
            return err
        }

        return nil
    })
}
```

**原则**：
- 设备事件回调也在一个事务内完成所有状态更新
- `Reservation → USED`、`BorrowRecord → BORROWED`、`Key → BORROWED`、`Slot → ABSENT` 要么全部成功，要么全部失败

---

## Authentication Flow

### Wechat Login → Backend → JWT

```
小程序
 ↓
wx.login()
 ↓
临时 code
 ↓
POST /api/v1/auth/wechat-login
 ↓
Go Backend
 ↓
调用微信 code2Session
 ↓
获取 openid
 ↓
查找 UserIdentity (type=WECHAT, provider_user_id=openid)
 ↓
存在？
 ├─ 是：读取 User
 └─ 否：创建 UserIdentity + User
 ↓
签发 JWT (claims: user_id, role)
 ↓
返回 {accessToken, expiresIn, user}
 ↓
小程序保存 accessToken
```

### JWT Middleware

```go
func AuthMiddleware() gin.HandlerFunc {
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
        claims, err := jwt.ValidateToken(tokenString)
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
```

**原则**：
- 微信 session_key 不传播到前端
- 所有 API 调用使用 `Authorization: Bearer <JWT>`
- JWT 包含 `user_id` + `role`，用于权限控制

---

## User Identity Design

### Why UserIdentity Table?

为了支持多种登录方式（微信、人脸识别），不要直接 `users.openid`。

```sql
-- users
CREATE TABLE users (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    student_id VARCHAR(50),
    role VARCHAR(20) NOT NULL DEFAULT 'STUDENT',
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- user_identities
CREATE TABLE user_identities (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL REFERENCES users(id),
    identity_type VARCHAR(20) NOT NULL, -- WECHAT, FACE
    provider_user_id VARCHAR(255) NOT NULL, -- openid, face_profile_id
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(identity_type, provider_user_id)
);
```

**效果**：
- **v0.4**：微信登录 → `WECHAT / openid123`
- **v0.6**：人脸登录 → `FACE / FACE_PROFILE_001`
- 两者都指向同一个 `User`

---

## Error Handling

### Unified AppError

```go
// platform/errors/error.go
package errors

type Code string

const (
    CodeInvalidInput    Code = "INVALID_INPUT"
    CodeUnauthorized    Code = "UNAUTHORIZED"
    CodeForbidden       Code = "FORBIDDEN"
    CodeNotFound        Code = "RESOURCE_NOT_FOUND"
    CodeConflict        Code = "CONFLICT"
    CodeInvalidState    Code = "INVALID_STATE"
    CodeTooEarly        Code = "TOO_EARLY"
    CodeExpired         Code = "EXPIRED"
    CodeInternalError   Code = "INTERNAL_ERROR"
)

type AppError struct {
    Code    Code
    Message string
    Cause   error
}

func (e *AppError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func New(code Code, message string) *AppError {
    return &AppError{Code: code, Message: message}
}

func Wrap(err error, message string) *AppError {
    return &AppError{Code: CodeInternalError, Message: message, Cause: err}
}
```

### Error Middleware

```go
// transport/http/middleware/error.go
func ErrorMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()

        if len(c.Errors) > 0 {
            err := c.Errors.Last().Err
            appErr, ok := err.(*errors.AppError)
            if !ok {
                appErr = &errors.AppError{
                    Code:    errors.CodeInternalError,
                    Message: err.Error(),
                }
            }

            statusCode := mapErrorToHTTPStatus(appErr.Code)
            c.JSON(statusCode, dto.ErrorResponse{
                Error: dto.ErrorDetail{
                    Code:      string(appErr.Code),
                    Message:   appErr.Message,
                    Timestamp: time.Now().Format(time.RFC3339),
                },
            })
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
    case errors.CodeConflict:
        return http.StatusConflict
    default:
        return http.StatusInternalServerError
    }
}
```

**原则**：
- Handler 调用 `c.Error(err)`，不直接返回 JSON
- Middleware 统一处理，保证错误格式一致
- 所有错误都符合 API 契约

---

## Testing Strategy

### 1. Unit Test

针对 Domain 层和 Service 层的单元测试。

```bash
# 运行所有单元测试
go test ./internal/domain/... ./internal/service/...
```

### 2. Integration Test

针对完整业务流程的集成测试，使用真实 PostgreSQL（testcontainers）。

```bash
# 运行集成测试
go test ./tests/integration/...
```

#### 示例：`tests/integration/reservation_test.go`

```go
package integration

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestReservationConcurrency(t *testing.T) {
    // R003 两个 goroutine 同时预约 → 只能一个成功
    db := setupTestDB(t)
    defer teardownTestDB(t, db)

    service := setupReservationService(db)

    var wg sync.WaitGroup
    errors := make(chan error, 2)

    for i := 0; i < 2; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            _, err := service.CreateReservation(context.Background(), CreateReservationRequest{
                UserID:    "U001",
                KeyID:     "KEY103",
                StartTime: "2026-09-03T14:00:00Z",
                EndTime:   "2026-09-03T16:00:00Z",
            })
            errors <- err
        }()
    }

    wg.Wait()
    close(errors)

    successCount := 0
    for err := range errors {
        if err == nil {
            successCount++
        }
    }

    assert.Equal(t, 1, successCount, "只能有一个预约成功")
}
```

### 3. Contract Test

验证 API 契约是否符合 `docs/API-SPECIFICATION.md`。

```bash
# 运行契约测试
go test ./tests/contract/...
```

---

## Migration Management

### Tool: golang-migrate/migrate

```bash
# 安装
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# 创建 migration
migrate create -ext sql -dir server/migrations -seq init

# 执行 up
migrate -path server/migrations -database "postgresql://user:pass@localhost:5432/keycabinet?sslmode=disable" up

# 回滚 down
migrate -path server/migrations -database "postgresql://user:pass@localhost:5432/keycabinet?sslmode=disable" down 1
```

### Migration Files

```sql
-- migrations/000001_init.up.sql
CREATE TABLE users (...);
CREATE TABLE user_identities (...);
CREATE TABLE keys (...);
CREATE TABLE slots (...);
CREATE TABLE devices (...);
CREATE TABLE reservations (...);
CREATE TABLE borrow_records (...);
CREATE TABLE device_operations (...);

-- migrations/000002_reservation_constraint.up.sql
CREATE EXTENSION IF NOT EXISTS btree_gist;

ALTER TABLE reservations
ADD CONSTRAINT reservations_no_overlap
EXCLUDE USING gist (
    key_id WITH =,
    tstzrange(start_time, end_time) WITH &&
)
WHERE (status IN ('ACTIVE', 'PENDING'));
```

**原则**：
- 每个 migration 有 `up` 和 `down`
- 生产环境只执行 `up`，本地开发可以 `down` 回滚
- 不依赖 `AutoMigrate`

---

## Dependencies

### Core Dependencies

```go
// server/go.mod
module github.com/yourusername/key-cabinet/server

go 1.22

require (
    github.com/gin-gonic/gin v1.10.0
    gorm.io/gorm v1.25.12
    gorm.io/driver/postgres v1.5.9
    github.com/golang-jwt/jwt/v5 v5.2.1
    github.com/golang-migrate/migrate/v4 v4.18.1
    github.com/google/uuid v1.6.0
    github.com/spf13/viper v1.19.0
    go.uber.org/zap v1.27.0
)
```

---

## API Routes

```go
// transport/http/router.go
func SetupRouter(
    authHandler *handler.AuthHandler,
    userHandler *handler.UserHandler,
    keyHandler *handler.KeyHandler,
    reservationHandler *handler.ReservationHandler,
    borrowHandler *handler.BorrowHandler,
    operationHandler *handler.OperationHandler,
) *gin.Engine {
    r := gin.Default()

    // Middleware
    r.Use(middleware.ErrorMiddleware())
    r.Use(middleware.LoggerMiddleware())

    // Public routes
    r.GET("/health", healthHandler)
    auth := r.Group("/api/v1/auth")
    {
        auth.POST("/wechat-login", authHandler.WechatLogin)
    }

    // Protected routes
    api := r.Group("/api/v1")
    api.Use(middleware.AuthMiddleware())
    {
        // User
        api.GET("/me", userHandler.GetMe)
        api.PUT("/me", userHandler.UpdateMe)

        // Keys
        api.GET("/keys", keyHandler.ListKeys)
        api.GET("/keys/:id", keyHandler.GetKey)

        // Reservations
        api.GET("/reservations", reservationHandler.ListReservations)
        api.POST("/reservations", reservationHandler.CreateReservation)
        api.GET("/reservations/:id", reservationHandler.GetReservation)
        api.PUT("/reservations/:id/cancel", reservationHandler.CancelReservation)

        // Borrow
        api.GET("/borrows", borrowHandler.ListBorrows)
        api.GET("/borrows/:id", borrowHandler.GetBorrow)
        api.POST("/borrows/pickup", borrowHandler.Pickup)
        api.POST("/borrows/:id/return", borrowHandler.Return)

        // Operations
        api.GET("/operations", operationHandler.ListOperations)
        api.GET("/operations/:id", operationHandler.GetOperation)
    }

    // Admin routes
    admin := r.Group("/admin")
    admin.Use(middleware.AuthMiddleware())
    admin.Use(middleware.AdminMiddleware())
    {
        // Admin APIs (v0.5+)
    }

    return r
}
```

---

## Configuration

### config/config.yaml

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
  secret: your-secret-key
  expiration: 86400 # 24h

wechat:
  app_id: your-wechat-app-id
  app_secret: your-wechat-app-secret

device:
  gateway_type: mock # mock / mqtt
  mqtt_broker: tcp://localhost:1883
  mqtt_username: ""
  mqtt_password: ""
```

### config/config.go

```go
package config

import (
    "github.com/spf13/viper"
)

type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    JWT      JWTConfig
    Wechat   WechatConfig
    Device   DeviceConfig
}

type ServerConfig struct {
    Port int
    Mode string
}

type DatabaseConfig struct {
    Host         string
    Port         int
    User         string
    Password     string
    DBName       string
    SSLMode      string
    MaxOpenConns int
    MaxIdleConns int
}

type JWTConfig struct {
    Secret     string
    Expiration int
}

type WechatConfig struct {
    AppID     string
    AppSecret string
}

type DeviceConfig struct {
    GatewayType  string
    MQTTBroker   string
    MQTTUsername string
    MQTTPassword string
}

func Load(configPath string) (*Config, error) {
    viper.SetConfigFile(configPath)
    viper.AutomaticEnv()

    if err := viper.ReadInConfig(); err != nil {
        return nil, err
    }

    var config Config
    if err := viper.Unmarshal(&config); err != nil {
        return nil, err
    }

    return &config, nil
}
```

---

## v0.4 Scope

### What v0.4 Must Deliver

- ✅ Complete Go backend with Gin + GORM + PostgreSQL
- ✅ JWT-based authentication (Wechat login → Backend JWT)
- ✅ Core business APIs (Key, Reservation, Borrow, Operation)
- ✅ MockDeviceGateway (模拟设备事件)
- ✅ Concurrency control (DB constraint + Transaction + FOR UPDATE)
- ✅ UserIdentity table (支持未来人脸登录)
- ✅ Unified error handling
- ✅ Integration tests (并发预约、取钥事务)
- ✅ 小程序 HttpService + DTO Mapper (退出 Mock 依赖)
- ✅ Full E2E verification (小程序 → 后端 → Mock 设备 → 小程序)

### What v0.4 Does NOT Deliver

- ❌ Real MQTT device integration (v0.5)
- ❌ Admin dashboard (v0.5+)
- ❌ Face recognition login (v0.6)
- ❌ Real-time progress via WebSocket (v0.4 使用 polling)

---

## Next Steps

1. ✅ 完成 `docs/BACKEND-ARCHITECTURE.md`（本文档）
2. 📝 编写 `docs/sprints/SPRINT-4.1-PLAN.md`
3. 🏗️ 实施 Sprint 4.1：Backend Foundation
4. 🧪 验证 Sprint 4.1：Health API + 测试通过
5. 🔁 继续 Sprint 4.2-4.8

---

## References

- [Go Project Layout](https://go.dev/doc/modules/layout)
- [Gin Web Framework](https://gin-gonic.com/)
- [GORM](https://gorm.io/)
- [PostgreSQL Constraints](https://www.postgresql.org/docs/current/ddl-constraints.html)
- [golang-migrate](https://github.com/golang-migrate/migrate)
- [JWT](https://jwt.io/)

---

**End of Document**
