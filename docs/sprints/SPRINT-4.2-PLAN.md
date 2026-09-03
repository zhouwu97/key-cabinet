# Sprint 4.2: 微信登录与用户管理

**状态**: 🟡 待开始  
**前置条件**: Sprint 4.1 完成 + 白屏问题修复验证通过  
**预计时间**: 2-3 天  
**负责人**: Backend + Frontend

---

## 目标

实现完整的微信登录流程，让小程序通过真实后端接口完成用户认证，并在个人中心显示真实用户信息。

### 核心验收标准

```bash
# 删除 MockUserService
rm miniprogram/services/user/mock-user-service.ts

# 小程序仍然正常工作
- ✅ 启动自动微信登录
- ✅ 个人中心显示真实用户信息（从后端返回）
- ✅ JWT 自动携带到所有需要认证的接口
```

---

## 技术方案

### 1. 认证流程设计

```
┌─────────────┐
│  小程序启动   │
└──────┬──────┘
       │
       ▼
┌─────────────────┐
│ 检查本地 Token   │
│ 是否存在且有效   │
└──────┬──────────┘
       │
       ├─ 有效 ─────────────────────┐
       │                            │
       └─ 无效/不存在                │
              │                     │
              ▼                     │
       ┌─────────────┐             │
       │ wx.login()  │             │
       │ 获取 code   │             │
       └──────┬──────┘             │
              │                     │
              ▼                     │
       ┌─────────────────────┐     │
       │ POST /auth/wechat   │     │
       │ Body: { code }      │     │
       └──────┬──────────────┘     │
              │                     │
              ▼                     │
       ┌─────────────────────┐     │
       │ Backend             │     │
       │ - code2Session      │     │
       │ - 获取 openid       │     │
       │ - 查询/创建 User    │     │
       │ - 签发 JWT          │     │
       └──────┬──────────────┘     │
              │                     │
              ▼                     │
       ┌─────────────────────┐     │
       │ 返回 accessToken    │     │
       │ + User 信息         │     │
       └──────┬──────────────┘     │
              │                     │
              ▼                     │
       ┌─────────────────────┐     │
       │ 小程序保存 Token    │     │
       │ 保存 User 信息      │     │
       └──────┬──────────────┘     │
              │                     │
              └─────────────────────┤
                                    │
                                    ▼
                          ┌──────────────────┐
                          │ 后续所有请求携带  │
                          │ Authorization     │
                          └──────────────────┘
```

### 2. 数据模型设计

#### User (Domain)

```go
type User struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    StudentNo string    `json:"studentNo"`
    Phone     string    `json:"phone"`
    Role      UserRole  `json:"role"`
    Status    UserStatus `json:"status"`
    CreatedAt time.Time `json:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt"`
}

type UserRole string
const (
    RoleStudent UserRole = "STUDENT"
    RoleAdmin   UserRole = "ADMIN"
)

type UserStatus string
const (
    StatusActive    UserStatus = "ACTIVE"
    StatusSuspended UserStatus = "SUSPENDED"
)
```

#### UserIdentity (Domain)

```go
type UserIdentity struct {
    ID        string           `json:"id"`
    UserID    string           `json:"userId"`
    Provider  IdentityProvider `json:"provider"`
    Subject   string           `json:"subject"` // openid for WECHAT
    CreatedAt time.Time        `json:"createdAt"`
}

type IdentityProvider string
const (
    ProviderWechat IdentityProvider = "WECHAT"
    ProviderFace   IdentityProvider = "FACE"  // v0.6
)
```

#### Database Schema

```sql
-- users 表
CREATE TABLE users (
    id VARCHAR(32) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    student_no VARCHAR(50),
    phone VARCHAR(20),
    role VARCHAR(20) NOT NULL DEFAULT 'STUDENT',
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_student_no ON users(student_no);
CREATE INDEX idx_users_status ON users(status);

-- user_identities 表
CREATE TABLE user_identities (
    id VARCHAR(32) PRIMARY KEY,
    user_id VARCHAR(32) NOT NULL,
    provider VARCHAR(20) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX idx_user_identities_provider_subject 
ON user_identities(provider, subject);

CREATE INDEX idx_user_identities_user_id ON user_identities(user_id);
```

### 3. API 设计

#### POST /api/v1/auth/wechat-login

**Request**:
```json
{
  "code": "081xAw100..."
}
```

**Response**:
```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIs...",
  "expiresIn": 86400,
  "user": {
    "id": "U001",
    "name": "张三",
    "studentNo": "20230001",
    "phone": "13800138000",
    "role": "STUDENT",
    "status": "ACTIVE"
  }
}
```

**错误响应**:
```json
{
  "error": {
    "code": "WECHAT_LOGIN_FAILED",
    "message": "微信登录失败，请重试",
    "timestamp": "2026-09-03T10:30:00Z"
  }
}
```

#### GET /api/v1/me

**Headers**:
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

**Response**:
```json
{
  "id": "U001",
  "name": "张三",
  "studentNo": "20230001",
  "phone": "13800138000",
  "role": "STUDENT",
  "status": "ACTIVE",
  "createdAt": "2026-09-01T10:00:00Z",
  "updatedAt": "2026-09-01T10:00:00Z"
}
```

---

## 任务拆解

### Phase 1: Backend - 数据层 (4小时)

#### Task 4.2.1: User Domain Model
- [ ] `internal/domain/user/user.go` - User 结构体
- [ ] `internal/domain/user/role.go` - UserRole 枚举
- [ ] `internal/domain/user/status.go` - UserStatus 枚举
- [ ] `internal/domain/user/errors.go` - 领域错误

#### Task 4.2.2: UserIdentity Domain Model
- [ ] `internal/domain/user/identity.go` - UserIdentity 结构体
- [ ] `internal/domain/user/provider.go` - IdentityProvider 枚举

#### Task 4.2.3: Database Migration
- [ ] `migrations/000003_create_users_tables.up.sql`
- [ ] `migrations/000003_create_users_tables.down.sql`
- [ ] 测试 migration up/down

#### Task 4.2.4: Repository Interface
- [ ] `internal/repository/user_repository.go` - Interface 定义
  ```go
  type UserRepository interface {
      Create(ctx context.Context, user *domain.User) error
      FindByID(ctx context.Context, id string) (*domain.User, error)
      Update(ctx context.Context, user *domain.User) error
  }
  ```
- [ ] `internal/repository/user_identity_repository.go` - Interface 定义
  ```go
  type UserIdentityRepository interface {
      Create(ctx context.Context, identity *domain.UserIdentity) error
      FindByProviderAndSubject(ctx context.Context, provider, subject string) (*domain.UserIdentity, error)
      FindByUserID(ctx context.Context, userID string) ([]*domain.UserIdentity, error)
  }
  ```

#### Task 4.2.5: GORM Implementation
- [ ] `internal/infrastructure/postgres/user_repository.go`
- [ ] `internal/infrastructure/postgres/user_identity_repository.go`
- [ ] `internal/infrastructure/postgres/user_entity.go` - GORM Entity
- [ ] `internal/infrastructure/postgres/user_identity_entity.go` - GORM Entity
- [ ] Domain ↔ Entity 映射函数

**验收**:
```bash
go test ./internal/infrastructure/postgres/... -v
```

---

### Phase 2: Backend - 微信 API 集成 (3小时)

#### Task 4.2.6: 微信 API Client
- [ ] `internal/infrastructure/wechat/client.go`
  ```go
  type WechatClient interface {
      Code2Session(ctx context.Context, code string) (*SessionResponse, error)
  }
  
  type SessionResponse struct {
      OpenID     string `json:"openid"`
      SessionKey string `json:"session_key"`
      UnionID    string `json:"unionid"`
      ErrCode    int    `json:"errcode"`
      ErrMsg     string `json:"errmsg"`
  }
  ```
- [ ] 配置：AppID、AppSecret
- [ ] HTTP 客户端实现
- [ ] 错误处理

#### Task 4.2.7: Mock Wechat Client
- [ ] `internal/infrastructure/wechat/mock_client.go`
  ```go
  // code → 固定的 mock openid
  // 用于本地开发和测试
  ```

**验收**:
```bash
go test ./internal/infrastructure/wechat/... -v
```

---

### Phase 3: Backend - 业务逻辑 (4小时)

#### Task 4.2.8: AuthService
- [ ] `internal/service/auth_service.go`
  ```go
  type AuthService struct {
      wechatClient         wechat.WechatClient
      userRepo             repository.UserRepository
      userIdentityRepo     repository.UserIdentityRepository
      jwtSecret            string
      jwtExpirationSeconds int
  }
  
  func (s *AuthService) WechatLogin(ctx context.Context, code string) (*LoginResponse, error)
  ```

**业务逻辑**:
1. 调用 `wechatClient.Code2Session(code)` 获取 openid
2. 查询 `UserIdentity` by `(WECHAT, openid)`
3. 如果存在 → 查询 User
4. 如果不存在 → 创建新 User + UserIdentity
5. 签发 JWT (包含 userID)
6. 返回 `accessToken` + `user`

#### Task 4.2.9: UserService
- [ ] `internal/service/user_service.go`
  ```go
  type UserService struct {
      userRepo repository.UserRepository
  }
  
  func (s *UserService) GetUserByID(ctx context.Context, id string) (*domain.User, error)
  func (s *UserService) UpdateUser(ctx context.Context, user *domain.User) error
  ```

**验收**:
```bash
go test ./internal/service/... -v
```

---

### Phase 4: Backend - HTTP Transport (3小时)

#### Task 4.2.10: Auth DTO
- [ ] `internal/transport/http/dto/auth_dto.go`
  ```go
  type WechatLoginRequest struct {
      Code string `json:"code" binding:"required"`
  }
  
  type LoginResponse struct {
      AccessToken string   `json:"accessToken"`
      ExpiresIn   int      `json:"expiresIn"`
      User        UserDTO  `json:"user"`
  }
  ```

#### Task 4.2.11: User DTO
- [ ] `internal/transport/http/dto/user_dto.go`
  ```go
  type UserDTO struct {
      ID        string `json:"id"`
      Name      string `json:"name"`
      StudentNo string `json:"studentNo"`
      Phone     string `json:"phone"`
      Role      string `json:"role"`
      Status    string `json:"status"`
      CreatedAt string `json:"createdAt"` // ISO8601
      UpdatedAt string `json:"updatedAt"` // ISO8601
  }
  ```

#### Task 4.2.12: Auth Handler
- [ ] `internal/transport/http/handler/auth_handler.go`
  ```go
  func (h *AuthHandler) WechatLogin(c *gin.Context)
  ```

#### Task 4.2.13: User Handler
- [ ] `internal/transport/http/handler/user_handler.go`
  ```go
  func (h *UserHandler) GetMe(c *gin.Context)
  ```

#### Task 4.2.14: JWT Middleware 完整实现
- [ ] `internal/transport/http/middleware/auth.go`
  ```go
  func RequireAuth(jwtSecret string) gin.HandlerFunc
  ```
- [ ] 从 `Authorization: Bearer <token>` 提取 JWT
- [ ] 验证 JWT
- [ ] 提取 userID 注入到 `c.Set("userID", userID)`
- [ ] 401 错误处理

#### Task 4.2.15: 路由注册
- [ ] `internal/transport/http/router.go`
  ```go
  // Public routes
  authGroup := r.Group("/api/v1/auth")
  {
      authGroup.POST("/wechat-login", authHandler.WechatLogin)
  }
  
  // Protected routes
  apiGroup := r.Group("/api/v1")
  apiGroup.Use(middleware.RequireAuth(cfg.JWTSecret))
  {
      apiGroup.GET("/me", userHandler.GetMe)
  }
  ```

**验收**:
```bash
# 启动服务器
go run cmd/api/main.go

# 测试微信登录
curl -X POST http://localhost:8080/api/v1/auth/wechat-login \
  -H "Content-Type: application/json" \
  -d '{"code":"test_code_123"}'

# 应返回 accessToken 和 user

# 测试 /me
curl http://localhost:8080/api/v1/me \
  -H "Authorization: Bearer <上面返回的token>"

# 应返回用户信息
```

---

### Phase 5: Frontend - 认证服务 (3小时)

#### Task 4.2.16: HTTP Service
- [ ] `miniprogram/services/http/http-service.ts`
  ```ts
  export class HttpService {
    private baseURL = 'http://localhost:8080' // 开发环境
    
    async request<T>(options: RequestOptions): Promise<T> {
      const token = wx.getStorageSync('accessToken')
      const headers = {
        'Content-Type': 'application/json',
        ...(token && { Authorization: `Bearer ${token}` }),
        ...options.headers,
      }
      
      const res = await wx.request({
        url: `${this.baseURL}${options.url}`,
        method: options.method || 'GET',
        data: options.data,
        header: headers,
      })
      
      if (res.statusCode === 401) {
        // Token 过期，重新登录
        await this.refreshAuth()
        // 重试请求
        return this.request(options)
      }
      
      if (res.statusCode >= 400) {
        throw new Error(res.data.error?.message || '请求失败')
      }
      
      return res.data as T
    }
    
    private async refreshAuth() {
      // 调用 AuthService.login()
    }
  }
  
  export const httpService = new HttpService()
  ```

#### Task 4.2.17: Real AuthService
- [ ] `miniprogram/services/auth/auth-service.ts`
  ```ts
  export interface IAuthService {
    login(): Promise<LoginResponse>
    getMe(): Promise<User>
    logout(): Promise<void>
  }
  
  export class AuthService implements IAuthService {
    async login(): Promise<LoginResponse> {
      // 1. wx.login() 获取 code
      const { code } = await wx.login()
      
      // 2. POST /auth/wechat-login
      const res = await httpService.request<LoginResponse>({
        url: '/api/v1/auth/wechat-login',
        method: 'POST',
        data: { code },
      })
      
      // 3. 保存 token 和 user
      wx.setStorageSync('accessToken', res.accessToken)
      wx.setStorageSync('user', res.user)
      
      return res
    }
    
    async getMe(): Promise<User> {
      return httpService.request<User>({
        url: '/api/v1/me',
      })
    }
    
    logout() {
      wx.removeStorageSync('accessToken')
      wx.removeStorageSync('user')
    }
  }
  
  export const authService = new AuthService()
  ```

#### Task 4.2.18: 删除 MockUserService
- [ ] 删除 `miniprogram/services/user/mock-user-service.ts`
- [ ] 更新 `miniprogram/services/user/index.ts`
  ```ts
  export * from './user-service'  // Interface
  // 移除 mock-user-service 的导出
  ```

#### Task 4.2.19: Real UserService
- [ ] `miniprogram/services/user/user-service.ts`
  ```ts
  export class UserService implements IUserService {
    async getCurrentUser(): Promise<User> {
      // 先尝试从本地读取
      const cached = wx.getStorageSync('user')
      if (cached) return cached
      
      // 否则请求后端
      const user = await authService.getMe()
      wx.setStorageSync('user', user)
      return user
    }
  }
  
  export const userService = new UserService()
  ```

#### Task 4.2.20: 更新 services/index.ts
- [ ] 替换所有 Mock 为真实实现
  ```ts
  import { authService } from './auth/auth-service'
  import { userService } from './user/user-service'
  import { MockKeyService } from './key/mock-key-service'  // 暂时保留
  // ...
  
  export {
    authService,
    userService,
    keyService,  // 仍然是 Mock
    // ...
  }
  ```

**验收**:
```bash
npm run check
```

---

### Phase 6: Frontend - 启动流程集成 (2小时)

#### Task 4.2.21: App 启动登录
- [ ] `miniprogram/app.ts`
  ```ts
  import { authService } from './services/index'
  
  App({
    async onLaunch() {
      console.log('App Launch')
      
      try {
        // 自动登录
        await authService.login()
        console.log('用户登录成功')
      } catch (e) {
        console.error('登录失败', e)
        // 不阻塞启动
      }
    },
  })
  ```

#### Task 4.2.22: Profile 页面对接真实数据
- [ ] `miniprogram/pages/profile/profile.ts`
  ```ts
  import { userService } from '../../services/index'
  
  Page({
    data: {
      user: null,
      loading: true,
    },
    
    onShow() {
      this.loadUserData()
    },
    
    async loadUserData() {
      try {
        this.setData({ loading: true })
        const user = await userService.getCurrentUser()
        this.setData({ user, loading: false })
      } catch (e) {
        console.error('加载用户数据失败', e)
        this.setData({ loading: false })
      }
    },
  })
  ```

**验收**:
1. 启动小程序
2. Console 显示"用户登录成功"
3. 切到"我的"页面
4. 显示真实用户信息（从后端返回的）

---

## 测试计划

### Unit Tests

```bash
# Backend
go test ./internal/domain/user/... -v
go test ./internal/infrastructure/postgres/... -v
go test ./internal/infrastructure/wechat/... -v
go test ./internal/service/... -v
```

### Integration Tests

```bash
# 创建测试数据库
createdb key_cabinet_test

# 运行迁移
DATABASE_URL=postgres://localhost/key_cabinet_test?sslmode=disable \
  go run cmd/migrate/main.go -command up

# 运行集成测试
go test ./tests/integration/auth_test.go -v
```

**测试场景**:
1. **首次登录**: code → 创建 User + UserIdentity → 返回 token
2. **重复登录**: code → 查询已存在 User → 返回 token
3. **Token 验证**: 携带 token 访问 /me → 返回用户信息
4. **Token 过期**: 过期 token 访问 /me → 401 → 自动重新登录
5. **无效 code**: 无效 code → 微信返回错误 → 400

### E2E Tests (手动)

1. **冷启动登录**
   - 清除小程序缓存
   - 启动小程序
   - 检查 Console: "用户登录成功"
   - 检查后端日志: 微信登录请求

2. **查看个人中心**
   - 切到"我的"页面
   - 检查用户信息是否显示
   - 检查网络请求: GET /api/v1/me

3. **Token 持久化**
   - 关闭小程序
   - 重新打开
   - 不应该重新请求微信登录
   - 直接使用本地 token

4. **Token 过期处理**
   - 手动清除后端 token
   - 访问个人中心
   - 应自动重新登录
   - 显示新的用户信息

---

## 配置管理

### Backend Config

```yaml
# config/config.yaml
wechat:
  app_id: "wxXXXXXXXXXXXXXXXX"
  app_secret: "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"

jwt:
  secret: "your-secret-key-change-in-production"
  expiration_seconds: 86400  # 24 hours
```

### Frontend Config

```typescript
// miniprogram/config/index.ts
export const config = {
  // 开发环境
  dev: {
    apiBaseURL: 'http://localhost:8080',
  },
  // 生产环境
  prod: {
    apiBaseURL: 'https://api.yourdomain.com',
  },
}

export const currentConfig = config.dev  // 根据环境切换
```

---

## 风险与缓解

### 风险1: 微信 AppID/AppSecret 未准备

**影响**: 无法调用真实微信 code2Session API

**缓解**:
- 优先使用 MockWechatClient
- code → 固定的 mock openid
- 功能完全可测试
- 真实 AppID 到位后直接替换

### 风险2: JWT Secret 泄露

**影响**: 安全风险

**缓解**:
- 开发环境使用固定 secret
- 生产环境从环境变量读取
- 不提交到 Git

### 风险3: Token 过期处理不当

**影响**: 用户体验差

**缓解**:
- 统一在 HttpService 处理 401
- 自动重新登录
- 重试失败的请求
- 最多重试 1 次，避免死循环

---

## 完成标准

### 功能完整性

- ✅ 小程序启动自动微信登录
- ✅ 后端 `/auth/wechat-login` 接口工作
- ✅ 后端 `/me` 接口工作
- ✅ JWT 自动携带到所有请求
- ✅ Token 过期自动刷新
- ✅ 个人中心显示真实用户信息

### 代码质量

- ✅ 所有测试通过
- ✅ TypeScript 无编译错误
- ✅ Go 无编译错误
- ✅ 无 lint 警告

### 文档完整

- ✅ API 文档更新
- ✅ 数据库 Schema 文档
- ✅ Sprint 4.2 总结文档

---

## 下一步: Sprint 4.3

**目标**: 钥匙管理 + 设备管理真实接口

**核心**:
- `GET /api/v1/keys` - 钥匙列表
- `GET /api/v1/keys/:id` - 钥匙详情
- `GET /api/v1/devices/:id` - 设备状态
- 删除 `MockKeyService` + `MockDeviceService`
- 钥匙页/首页对接真实后端

---

**Sprint 4.2 准备就绪，白屏修复验证通过后即可开始！**
