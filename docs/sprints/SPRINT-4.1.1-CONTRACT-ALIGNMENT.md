# Sprint 4.1.1: 契约对齐与基础设施修复

**状态**: 🔴 Current Priority  
**前置条件**: Sprint 4.1 完成  
**阻塞**: Sprint 4.2 开始  
**预计时间**: 1-2 天  
**优先级**: P0 - 必须在任何新业务接口前完成

---

## 问题陈述

Sprint 4.1 建立了数据库基础设施，但在过程中产生了**四层契约漂移**：

```
冻结文档 DATA-MODEL.md
    ≠
小程序 models/*.ts
    ≠
Sprint 4.2 计划
    ≠
PostgreSQL Migration 000001
```

这种漂移如果不修复，会导致：
1. 前端发送的字段后端接收不到
2. 数据库字段与业务逻辑不匹配
3. Migration 冲突（000003 试图重建已存在的表）
4. 业务状态机无法正确运转

---

## 核心决策

### 决策 1: User 字段统一

**最终决定**:
- 学号/工号字段：**`student_no`** (数据库) / **`studentNo`** (Go/TS)
- 角色枚举：**`USER`** | **`ADMIN`** (保留 `MAINTAINER` 作为 v0.5 扩展)
- 状态枚举：**`ACTIVE`** | **`DISABLED`** (不使用 `SUSPENDED`，直接禁用)
- 必须保留：**`credit_score`** (信用评分，业务核心)

**理由**:
- `studentNo` 比 `studentId` 语义更明确（ID 容易与系统 ID 混淆）
- `USER`/`ADMIN` 符合小程序已有实现，改动最小
- `DISABLED` 比 `SUSPENDED` 更终局，业务逻辑更清晰

### 决策 2: API 响应格式统一

**最终决定**: 严格遵守 API Contract 文档的统一包装格式

```typescript
// 所有成功响应
{
  "code": 0,
  "message": "success",
  "data": T  // 实际业务数据
}

// 所有错误响应
{
  "code": 40001,
  "errorCode": "VALIDATION_ERROR",
  "message": "学号格式不正确",
  "data": null,
  "timestamp": "2026-09-03T10:00:00Z"
}
```

**理由**:
- 前后端统一错误处理逻辑
- 便于前端 HttpClient 统一拦截
- 符合已冻结的 API Contract

### 决策 3: 时间字段统一

**最终决定**: 全部使用 **`TIMESTAMPTZ`**

**Reservation 时间语义**:
- `pickup_window_start` - 取钥窗口开始
- `pickup_window_end` - 取钥窗口结束
- `expected_return_at` - 预期归还时间

删除模糊的 `start_time`/`end_time`。

**理由**:
- 预约系统有明确的"取钥窗口"和"归还期限"两个独立时间概念
- `TIMESTAMPTZ` 避免时区混乱

### 决策 4: BorrowRecord.borrowed_at 可空

**最终决定**: `borrowed_at TIMESTAMPTZ NULL`

**业务语义**:
```
BORROWING (borrowed_at = NULL) → 开锁中，用户还没拿走
        ↓ KEY_REMOVED
BORROWED (borrowed_at = NOW()) → 真正借出
```

### 决策 5: DeviceOperation 幂等性

**最终决定**: 必须添加 `request_id VARCHAR(64) UNIQUE NOT NULL`

**业务逻辑**:
```typescript
// 前端生成唯一请求 ID
const requestId = `REQ_${Date.now()}_${Math.random()}`
POST /pickup { requestId, reservationId }

// 后端先检查 request_id 是否已存在
// 存在 → 返回已有 operation
// 不存在 → 创建新 operation
```

防止网络重试导致重复开锁。

---

## 任务清单

### Phase 1: Migration 修复（P0）

#### Task 4.1.1.1: 修正 000001_init.up.sql

**不能新建 000003**，因为会冲突。应该：

**选项 A（如果数据库还没有生产数据）**:
```bash
# 回滚所有 migration
migrate -path ./migrations -database $DATABASE_URL down

# 修正 000001_init.up.sql
# 修正 000002_reservation_constraint.up.sql

# 重新应用
migrate -path ./migrations -database $DATABASE_URL up
```

**选项 B（如果已有数据）**:
```sql
-- 000003_fix_schema.up.sql
ALTER TABLE users RENAME COLUMN student_id TO student_no;
ALTER TABLE users ADD COLUMN credit_score INTEGER NOT NULL DEFAULT 100;
ALTER TABLE users ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE users ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

ALTER TABLE user_identities RENAME COLUMN identity_type TO provider;
ALTER TABLE user_identities RENAME COLUMN provider_user_id TO subject;
ALTER TABLE user_identities ALTER COLUMN created_at TYPE TIMESTAMPTZ;

ALTER TABLE reservations ADD COLUMN pickup_window_start TIMESTAMPTZ;
ALTER TABLE reservations ADD COLUMN pickup_window_end TIMESTAMPTZ;
ALTER TABLE reservations ADD COLUMN expected_return_at TIMESTAMPTZ;
UPDATE reservations SET 
  pickup_window_start = start_time,
  pickup_window_end = start_time + INTERVAL '2 hours',
  expected_return_at = end_time;
ALTER TABLE reservations DROP COLUMN start_time;
ALTER TABLE reservations DROP COLUMN end_time;
ALTER TABLE reservations ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE reservations ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

ALTER TABLE borrow_records ALTER COLUMN borrowed_at DROP NOT NULL;
ALTER TABLE borrow_records ALTER COLUMN borrowed_at TYPE TIMESTAMPTZ;
ALTER TABLE borrow_records ALTER COLUMN expected_return_at TYPE TIMESTAMPTZ;
ALTER TABLE borrow_records ALTER COLUMN returned_at TYPE TIMESTAMPTZ;
ALTER TABLE borrow_records ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE borrow_records ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

ALTER TABLE device_operations ADD COLUMN request_id VARCHAR(64);
ALTER TABLE device_operations ADD COLUMN user_id VARCHAR(64);
ALTER TABLE device_operations ADD COLUMN error_code VARCHAR(50);
CREATE UNIQUE INDEX idx_device_operations_request_id ON device_operations(request_id);
ALTER TABLE device_operations ALTER COLUMN initiated_at TYPE TIMESTAMPTZ;
ALTER TABLE device_operations ALTER COLUMN completed_at TYPE TIMESTAMPTZ;
ALTER TABLE device_operations ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE device_operations ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

-- 全部其他表的 TIMESTAMP 改 TIMESTAMPTZ
ALTER TABLE devices ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE devices ALTER COLUMN updated_at TYPE TIMESTAMPTZ;
ALTER TABLE devices ALTER COLUMN last_heartbeat_at TYPE TIMESTAMPTZ USING last_heartbeat_at AT TIME ZONE 'UTC';

ALTER TABLE slots ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE slots ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

ALTER TABLE keys ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE keys ALTER COLUMN updated_at TYPE TIMESTAMPTZ;
```

**验收标准**:
```bash
migrate -path ./migrations -database $DATABASE_URL version
# 应显示当前版本

psql $DATABASE_URL -c "\d users"
# 应看到 student_no, credit_score, TIMESTAMPTZ
```

---

#### Task 4.1.1.2: 创建 operation_events 表

```sql
-- 000004_create_operation_events.up.sql
CREATE TABLE operation_events (
    id VARCHAR(64) PRIMARY KEY,
    operation_id VARCHAR(64) NOT NULL REFERENCES device_operations(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    event_data JSONB,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_operation_events_operation_id ON operation_events(operation_id);
CREATE INDEX idx_operation_events_occurred_at ON operation_events(occurred_at);
```

---

### Phase 2: Go Module 修复（P0）

#### Task 4.1.1.3: 修正 go.mod module path

```bash
cd server

# 查找所有 import 路径
grep -r "github.com/yourusername" . --include="*.go"

# 全局替换
find . -name "*.go" -exec sed -i 's|github.com/yourusername/key-cabinet/server|github.com/zhouwu97/key-cabinet/server|g' {} +

# 修改 go.mod
sed -i 's|github.com/yourusername/key-cabinet/server|github.com/zhouwu97/key-cabinet/server|g' go.mod

go mod tidy
```

**验收**:
```bash
go build ./cmd/api
# 应成功编译
```

---

#### Task 4.1.1.4: 统一 Go 版本

**Dockerfile**:
```dockerfile
FROM golang:1.26-alpine AS builder  # 改为 1.26
```

**go.mod**:
```go
go 1.26.2  // 保持
```

---

### Phase 3: 配置管理修复（P0）

#### Task 4.1.1.5: 统一环境变量命名

**docker-compose.yml**:
```yaml
environment:
  - KC_DATABASE_HOST=postgres
  - KC_DATABASE_PORT=5432
  - KC_DATABASE_USER=postgres
  - KC_DATABASE_PASSWORD=postgres
  - KC_DATABASE_DBNAME=keycabinet
  - KC_JWT_SECRET=docker-development-secret-change-in-production
  - KC_WECHAT_APP_ID=
  - KC_WECHAT_APP_SECRET=
  - KC_DEVICE_GATEWAY_TYPE=mock
```

**internal/config/config.go**:
```go
func Load() (*Config, error) {
    viper.SetEnvPrefix("KC")
    viper.AutomaticEnv()
    viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
    
    // Bind explicit keys
    viper.BindEnv("database.host", "KC_DATABASE_HOST")
    viper.BindEnv("database.port", "KC_DATABASE_PORT")
    // ...
}
```

---

### Phase 4: Domain Model 收敛（P0）

#### Task 4.1.1.6: 统一 User 模型

**小程序 models/user.ts**:
```typescript
export type UserRole = 'USER' | 'ADMIN'
export type UserStatus = 'ACTIVE' | 'DISABLED'

export interface User {
  id: string
  name: string
  studentNo: string          // ✅ 统一
  phone?: string
  role: UserRole
  status: UserStatus
  department?: string        // ✅ 添加
  creditScore: number        // ✅ 添加
  createdAt?: string
  updatedAt?: string
}
```

**Go domain (新建 internal/domain/user.go)**:
```go
package domain

type UserRole string

const (
	UserRoleUser  UserRole = "USER"
	UserRoleAdmin UserRole = "ADMIN"
)

type UserStatus string

const (
	UserStatusActive   UserStatus = "ACTIVE"
	UserStatusDisabled UserStatus = "DISABLED"
)

type User struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	StudentNo   string     `json:"studentNo"`
	Phone       string     `json:"phone"`
	Role        UserRole   `json:"role"`
	Status      UserStatus `json:"status"`
	Department  string     `json:"department"`
	CreditScore int        `json:"creditScore"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}
```

---

#### Task 4.1.1.7: 更新冻结文档

修改 `docs/02-DATA-MODEL.md`:

```diff
- studentId: string;
+ studentNo: string;

- role: 'USER' | 'ADMIN' | 'MAINTAINER';
+ role: 'USER' | 'ADMIN';  // MAINTAINER reserved for v0.5

+ status: 'ACTIVE' | 'DISABLED';
```

修改 `docs/04-API-CONTRACT.md`:

确保所有示例响应都使用统一包装格式。

---

### Phase 5: 前端基础设施（P1）

#### Task 4.1.1.8: 创建 HttpClient

```typescript
// miniprogram/api/http-client.ts
export interface ApiResponse<T> {
  code: number
  message: string
  data: T
  timestamp?: string
}

export interface ApiError {
  code: number
  errorCode: string
  message: string
  data: null
  timestamp: string
}

export class HttpClient {
  private baseURL = 'http://localhost:8080'
  
  async request<T>(options: RequestOptions): Promise<T> {
    const token = wx.getStorageSync('accessToken')
    
    const res = await wx.request({
      url: `${this.baseURL}${options.url}`,
      method: options.method || 'GET',
      data: options.data,
      header: {
        'Content-Type': 'application/json',
        ...(token && { Authorization: `Bearer ${token}` }),
        ...options.headers,
      },
    })
    
    // 401 自动重新登录
    if (res.statusCode === 401) {
      await this.refreshAuth()
      return this.request(options)
    }
    
    // 业务错误
    if (res.statusCode >= 400) {
      const error = res.data as ApiError
      throw new Error(error.message || '请求失败')
    }
    
    // 统一解包 data
    const response = res.data as ApiResponse<T>
    return response.data
  }
  
  private async refreshAuth() {
    const { authService } = await import('../services/auth/auth-service')
    await authService.login()
  }
}

export const httpClient = new HttpClient()
```

---

#### Task 4.1.1.9: 保留 Mock Service，添加 Api Service

**不删除** `mock-user-service.ts`，而是：

```typescript
// miniprogram/services/user/api-user-service.ts
import { httpClient } from '../../api/http-client'
import type { User } from '../../models/user'
import type { IUserService } from './user-service'

export class ApiUserService implements IUserService {
  async getCurrentUser(): Promise<User> {
    return httpClient.request<User>({
      url: '/api/v1/me',
    })
  }
  
  async updateUser(user: Partial<User>): Promise<User> {
    return httpClient.request<User>({
      url: '/api/v1/me',
      method: 'PATCH',
      data: user,
    })
  }
}
```

```typescript
// miniprogram/services/index.ts
import { MockUserService } from './user/mock-user-service'
import { ApiUserService } from './user/api-user-service'
import { MockKeyService } from './key/mock-key-service'
// ...

const USE_MOCK = true  // 开发时切换

export const userService = USE_MOCK 
  ? new MockUserService() 
  : new ApiUserService()
  
export const keyService = new MockKeyService()  // 暂时保留
// ...
```

---

### Phase 6: CI 基础设施（P1）

#### Task 4.1.1.10: 添加 GitHub Actions

```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  frontend:
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: ./miniprogram
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '18'
      - run: npm ci
      - run: npm run check
      - run: npm test

  backend:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:14-alpine
        env:
          POSTGRES_USER: postgres
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: keycabinet_test
        ports:
          - 5432:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    
    defaults:
      run:
        working-directory: ./server
    
    steps:
      - uses: actions/checkout@v3
      
      - uses: actions/setup-go@v4
        with:
          go-version: '1.26'
      
      - name: Install migrate
        run: |
          curl -L https://github.com/golang-migrate/migrate/releases/download/v4.15.2/migrate.linux-amd64.tar.gz | tar xvz
          sudo mv migrate /usr/local/bin/
      
      - name: Run migrations
        env:
          DATABASE_URL: postgres://postgres:postgres@localhost:5432/keycabinet_test?sslmode=disable
        run: migrate -path ./migrations -database $DATABASE_URL up
      
      - name: Go fmt check
        run: |
          if [ "$(gofmt -s -l . | wc -l)" -gt 0 ]; then
            echo "Run 'gofmt -s -w .'"
            gofmt -s -l .
            exit 1
          fi
      
      - name: Go vet
        run: go vet ./...
      
      - name: Go test
        env:
          KC_DATABASE_HOST: localhost
          KC_DATABASE_PORT: 5432
          KC_DATABASE_USER: postgres
          KC_DATABASE_PASSWORD: postgres
          KC_DATABASE_DBNAME: keycabinet_test
          KC_JWT_SECRET: test-secret
        run: go test ./... -v

  docker-build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Build Docker image
        run: docker build -t keycabinet-api:test ./server
```

---

### Phase 7: OpenAPI Schema（P2）

#### Task 4.1.1.11: 创建 OpenAPI 规范

```yaml
# api/openapi.yaml
openapi: 3.0.3
info:
  title: Key Cabinet API
  version: 1.0.0
  description: 智能钥匙自助借还系统 API

servers:
  - url: http://localhost:8080/api/v1
    description: 开发环境

components:
  schemas:
    UserDTO:
      type: object
      required:
        - id
        - name
        - studentNo
        - role
        - status
        - creditScore
      properties:
        id:
          type: string
        name:
          type: string
        studentNo:
          type: string
        phone:
          type: string
        role:
          type: string
          enum: [USER, ADMIN]
        status:
          type: string
          enum: [ACTIVE, DISABLED]
        department:
          type: string
        creditScore:
          type: integer
        createdAt:
          type: string
          format: date-time
        updatedAt:
          type: string
          format: date-time
    
    ApiResponse:
      type: object
      required:
        - code
        - message
        - data
      properties:
        code:
          type: integer
        message:
          type: string
        data:
          type: object
    
    ApiError:
      type: object
      required:
        - code
        - errorCode
        - message
      properties:
        code:
          type: integer
        errorCode:
          type: string
        message:
          type: string
        data:
          nullable: true
        timestamp:
          type: string
          format: date-time

paths:
  /auth/wechat-login:
    post:
      summary: 微信小程序登录
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required:
                - code
              properties:
                code:
                  type: string
      responses:
        '200':
          description: 登录成功
          content:
            application/json:
              schema:
                allOf:
                  - $ref: '#/components/schemas/ApiResponse'
                  - properties:
                      data:
                        type: object
                        properties:
                          accessToken:
                            type: string
                          expiresIn:
                            type: integer
                          user:
                            $ref: '#/components/schemas/UserDTO'
  
  /me:
    get:
      summary: 获取当前用户信息
      security:
        - BearerAuth: []
      responses:
        '200':
          description: 成功
          content:
            application/json:
              schema:
                allOf:
                  - $ref: '#/components/schemas/ApiResponse'
                  - properties:
                      data:
                        $ref: '#/components/schemas/UserDTO'

securitySchemes:
  BearerAuth:
    type: http
    scheme: bearer
    bearerFormat: JWT
```

---

## 验收标准

### ✅ Migration 通过
```bash
cd server
migrate -path ./migrations -database $DATABASE_URL up
# 无错误

psql $DATABASE_URL -c "\d users" | grep student_no
psql $DATABASE_URL -c "\d users" | grep credit_score
psql $DATABASE_URL -c "\d borrow_records" | grep "borrowed_at.*timestamp with time zone"
psql $DATABASE_URL -c "\d device_operations" | grep request_id
```

### ✅ Go 编译通过
```bash
cd server
go build ./cmd/api
# 成功
```

### ✅ 前端类型检查通过
```bash
cd miniprogram
npm run check
# 无错误
```

### ✅ CI 通过
```bash
# Push 到 GitHub
git push origin main
# Actions 全绿
```

### ✅ 契约一致性
- [ ] User 字段在 DB / Go / TS / 文档中完全一致
- [ ] API 响应格式在文档 / Go DTO / TS HttpClient 中一致
- [ ] 时间字段全部使用 TIMESTAMPTZ
- [ ] DeviceOperation 有 request_id 幂等性保护

---

## 风险与缓解

### 风险 1: Migration 已经在生产环境应用

**影响**: 不能直接修改 000001

**缓解**:
- 使用 ALTER 方式创建 000003_fix_schema.up.sql
- 保留 000001/000002 不变
- 确保 down migration 可回滚

### 风险 2: 前端已有代码使用旧字段名

**影响**: 全局搜索替换可能遗漏

**缓解**:
```bash
cd miniprogram
grep -r "studentId" . --include="*.ts" --include="*.js"
# 确认全部替换为 studentNo
```

### 风险 3: Go module 路径修改后历史 commit 无法编译

**影响**: git bisect 困难

**缓解**:
- 单独一个 commit 修改 module path
- commit message 注明：`chore: fix module path to real GitHub account`

---

## 下一步: Sprint 4.2

完成 Sprint 4.1.1 后，Sprint 4.2 计划需要相应调整：

**删除的任务**:
- ~~Task 4.2.3: 创建 000003_create_users_tables.up.sql~~（已在 4.1.1 修复）
- ~~Task 4.2.16: 创建 HTTP Service~~（已在 4.1.1 创建 HttpClient）

**新增的前置条件**:
- ✅ 契约已收敛
- ✅ HttpClient 已就绪
- ✅ CI 已建立

Sprint 4.2 真正专注于：
1. 微信 API 集成
2. AuthService 业务逻辑
3. JWT Middleware
4. `/auth/wechat-login` 和 `/me` 端到端联调

---

**Sprint 4.1.1 准备就绪，建议立即开始！**
