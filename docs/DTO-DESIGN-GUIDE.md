# DTO 设计指南 (Data Transfer Object Design Guide)

> **版本**: v0.3.1  
> **适用范围**: 微信小程序前端 API 层设计

---

## 1. 设计原则

### 1.1 职责分离

前端应明确区分三层数据结构：

```text
HTTP JSON (API 响应)
  ↓
DTO (Data Transfer Object)
  ↓ mapper
Domain Model (领域模型)
  ↓
UI (页面/组件)
```

**职责划分**:
- **DTO**: 与后台 API 契约一致，字段类型严格对应 HTTP JSON
- **Domain Model**: 前端业务逻辑层使用的实体，字段类型适配 TypeScript/JavaScript
- **Mapper**: 负责 DTO → Domain Model 的转换逻辑

### 1.2 为什么需要 DTO

**核心问题**: API 返回 ISO 时间字符串，前端需要 number 时间戳

```typescript
// API 返回
{
  "pickupWindowStart": "2026-09-02T20:00:00.000Z"
}

// 前端需要
{
  pickupWindowStart: 1725307200000  // number
}
```

**如果没有 DTO 层**:
- 页面/Service 到处写 `new Date(api.xxx).getTime()`
- 时间转换逻辑分散，难以维护
- 测试时需要同时构造 ISO 字符串和时间戳

**有了 DTO 层**:
- 转换逻辑集中在 mapper
- 领域模型保持纯粹，不含转换逻辑
- 测试时可以直接构造领域模型

---

## 2. 实现结构

### 2.1 目录组织

```text
miniprogram/
├── api/
│   ├── dto/                    # DTO 定义
│   │   ├── reservation.dto.ts
│   │   ├── borrow-record.dto.ts
│   │   ├── device-operation.dto.ts
│   │   └── key.dto.ts
│   ├── mappers/                # DTO → Domain 转换器
│   │   ├── reservation.mapper.ts
│   │   ├── borrow-record.mapper.ts
│   │   ├── device-operation.mapper.ts
│   │   └── key.mapper.ts
│   └── http.service.ts         # HTTP 客户端
├── models/                     # Domain Model (现有)
│   ├── reservation.ts
│   ├── borrow-record.ts
│   └── ...
└── services/                   # 业务 Service
    ├── reservation.service.ts
    └── ...
```

### 2.2 DTO 定义示例

```typescript
// api/dto/reservation.dto.ts

/**
 * 预约 DTO (与后台 API 契约严格对应)
 */
export interface ReservationDTO {
  id: string
  userId: string
  keyId: string
  keyName?: string
  roomNo?: string
  deviceId: string
  status: ReservationStatusDTO
  purpose: string
  pickupWindowStart: string  // ISO8601
  pickupWindowEnd: string    // ISO8601
  expectedReturnAt: string   // ISO8601
  createdAt: string          // ISO8601
  approvedAt?: string        // ISO8601
  usedAt?: string           // ISO8601
  cancelledAt?: string      // ISO8601
}

export type ReservationStatusDTO = 
  | 'PENDING'
  | 'APPROVED'
  | 'ACTIVE'
  | 'USED'
  | 'REJECTED'
  | 'CANCELLED'
  | 'EXPIRED'
```

### 2.3 Mapper 实现示例

```typescript
// api/mappers/reservation.mapper.ts

import { ReservationDTO, ReservationStatusDTO } from '../dto/reservation.dto'
import { Reservation, ReservationStatus } from '../../models/reservation'

/**
 * 将 ISO 时间字符串转换为时间戳
 */
function parseTimestamp(isoString: string): number {
  return new Date(isoString).getTime()
}

/**
 * 将 DTO 状态转换为领域模型状态
 */
function mapStatus(dtoStatus: ReservationStatusDTO): ReservationStatus {
  // 如果状态枚举完全一致，直接返回
  return dtoStatus as ReservationStatus
}

/**
 * 将预约 DTO 转换为领域模型
 */
export function reservationFromDTO(dto: ReservationDTO): Reservation {
  return {
    id: dto.id,
    userId: dto.userId,
    keyId: dto.keyId,
    status: mapStatus(dto.status),
    purpose: dto.purpose,
    pickupWindowStart: parseTimestamp(dto.pickupWindowStart),
    pickupWindowEnd: parseTimestamp(dto.pickupWindowEnd),
    expectedReturnAt: parseTimestamp(dto.expectedReturnAt),
    createdAt: parseTimestamp(dto.createdAt),
    approvedAt: dto.approvedAt ? parseTimestamp(dto.approvedAt) : undefined,
    usedAt: dto.usedAt ? parseTimestamp(dto.usedAt) : undefined,
    cancelledAt: dto.cancelledAt ? parseTimestamp(dto.cancelledAt) : undefined,
  }
}

/**
 * 批量转换
 */
export function reservationsFromDTO(dtos: ReservationDTO[]): Reservation[] {
  return dtos.map(reservationFromDTO)
}
```

### 2.4 Service 使用示例

```typescript
// services/reservation.service.ts

import { httpService } from '../api/http.service'
import { ReservationDTO } from '../api/dto/reservation.dto'
import { reservationFromDTO, reservationsFromDTO } from '../api/mappers/reservation.mapper'
import { Reservation } from '../models/reservation'

export class ReservationService {
  /**
   * 获取当前用户预约列表
   */
  async getMyReservations(): Promise<Reservation[]> {
    const response = await httpService.get<ReservationDTO[]>('/me/reservations')
    return reservationsFromDTO(response.data)
  }

  /**
   * 创建预约
   */
  async createReservation(input: {
    keyId: string
    pickupWindowStart: number
    pickupWindowEnd: number
    expectedReturnAt: number
    purpose: string
  }): Promise<Reservation> {
    // 发送时将 number 转回 ISO string
    const requestBody = {
      keyId: input.keyId,
      pickupWindowStart: new Date(input.pickupWindowStart).toISOString(),
      pickupWindowEnd: new Date(input.pickupWindowEnd).toISOString(),
      expectedReturnAt: new Date(input.expectedReturnAt).toISOString(),
      purpose: input.purpose,
    }
    
    const response = await httpService.post<ReservationDTO>('/reservations', requestBody)
    return reservationFromDTO(response.data)
  }
}
```

---

## 3. 关键转换场景

### 3.1 时间字段转换

**API → 前端** (ISO string → number):
```typescript
pickupWindowStart: parseTimestamp(dto.pickupWindowStart)
```

**前端 → API** (number → ISO string):
```typescript
pickupWindowStart: new Date(model.pickupWindowStart).toISOString()
```

### 3.2 可选字段处理

```typescript
approvedAt: dto.approvedAt ? parseTimestamp(dto.approvedAt) : undefined
```

### 3.3 嵌套对象转换

```typescript
// DTO
export interface BorrowRecordDTO {
  id: string
  keyInfo: KeyDTO  // 嵌套对象
  borrowedAt: string
}

// Mapper
export function borrowRecordFromDTO(dto: BorrowRecordDTO): BorrowRecord {
  return {
    id: dto.id,
    key: keyFromDTO(dto.keyInfo),  // 递归转换
    borrowedAt: parseTimestamp(dto.borrowedAt),
  }
}
```

### 3.4 枚举值映射

**当 API 枚举与前端枚举完全一致时**:
```typescript
status: dto.status as ReservationStatus
```

**当 API 枚举与前端枚举不一致时**:
```typescript
function mapStatus(dtoStatus: string): ReservationStatus {
  switch (dtoStatus) {
    case 'PENDING_APPROVAL': return ReservationStatus.PENDING
    case 'ACTIVE': return ReservationStatus.ACTIVE
    // ...
    default: throw new Error(`Unknown status: ${dtoStatus}`)
  }
}
```

---

## 4. 测试策略

### 4.1 Mapper 单元测试

```typescript
// api/mappers/__tests__/reservation.mapper.test.ts

import { reservationFromDTO } from '../reservation.mapper'
import { ReservationDTO } from '../../dto/reservation.dto'

describe('reservationFromDTO', () => {
  it('should convert DTO to domain model', () => {
    const dto: ReservationDTO = {
      id: 'RES001',
      userId: 'U001',
      keyId: 'KEY103',
      deviceId: 'CAB001',
      status: 'ACTIVE',
      purpose: '测试',
      pickupWindowStart: '2026-09-02T20:00:00.000Z',
      pickupWindowEnd: '2026-09-02T22:00:00.000Z',
      expectedReturnAt: '2026-09-03T08:00:00.000Z',
      createdAt: '2026-09-02T18:00:00.000Z',
    }

    const model = reservationFromDTO(dto)

    expect(model.id).toBe('RES001')
    expect(model.pickupWindowStart).toBe(1725307200000) // number
    expect(model.status).toBe('ACTIVE')
  })

  it('should handle optional fields', () => {
    const dto: ReservationDTO = {
      // ... 必填字段
      approvedAt: '2026-09-02T18:30:00.000Z',
    }

    const model = reservationFromDTO(dto)

    expect(model.approvedAt).toBe(1725305400000)
  })
})
```

### 4.2 Service 集成测试

Service 测试时直接构造领域模型，不需要关心 DTO：

```typescript
// services/__tests__/reservation.service.test.ts

describe('ReservationService', () => {
  it('should get my reservations', async () => {
    // Mock HTTP 响应 (DTO 格式)
    mockHttp.get.mockResolvedValue({
      data: [
        {
          id: 'RES001',
          // ... DTO 字段
          pickupWindowStart: '2026-09-02T20:00:00.000Z',
        }
      ]
    })

    const reservations = await service.getMyReservations()

    // 验证返回的是领域模型
    expect(reservations[0].pickupWindowStart).toBeTypeOf('number')
  })
})
```

---

## 5. 迁移策略

### 5.1 现状评估

当前 `v0.3.0` 使用 Mock Service，数据层已经是领域模型格式。

### 5.2 迁移步骤

**Phase 1**: 创建 DTO 和 Mapper (不影响现有代码)
```text
1. 创建 api/dto/ 目录
2. 根据 04-API-CONTRACT.md 定义所有 DTO
3. 创建 api/mappers/ 目录
4. 实现所有 mapper 函数
5. 编写 mapper 单元测试
```

**Phase 2**: 创建 HttpService (与 MockService 并存)
```text
1. 实现 api/http.service.ts
2. 在 Service 中使用 mapper 转换
3. 通过配置切换 Mock / Http
```

**Phase 3**: 切换到真实后台
```text
1. 后台 API 就绪后，切换配置
2. 现有页面完全不用修改
3. 领域模型和业务逻辑保持不变
```

### 5.3 配置切换示例

```typescript
// config/env.ts
export const USE_MOCK = false  // 切换这里即可

// services/reservation.service.ts
import { USE_MOCK } from '../config/env'
import { MockReservationService } from './mock/reservation.mock.service'
import { HttpReservationService } from './http/reservation.http.service'

export const reservationService = USE_MOCK
  ? new MockReservationService()
  : new HttpReservationService()
```

---

## 6. 最佳实践

### 6.1 DO

✅ DTO 严格对应 API 契约，字段类型与 JSON 一致  
✅ Mapper 集中处理所有转换逻辑  
✅ 领域模型保持纯粹，不含 API 相关逻辑  
✅ 为 mapper 编写单元测试  
✅ 枚举值完全一致时直接 `as` 转换  

### 6.2 DON'T

❌ 不要在页面/组件中直接调用 API  
❌ 不要在领域模型中混入 DTO 字段  
❌ 不要在多处重复写时间转换逻辑  
❌ 不要跳过 mapper 直接使用 DTO  
❌ 不要在 DTO 中定义业务逻辑方法  

---

## 7. 参考资料

- [02-DATA-MODEL.md](./02-DATA-MODEL.md) - 领域模型定义
- [04-API-CONTRACT.md](./04-API-CONTRACT.md) - API 契约规范
- [CONTRACT-ALIGNMENT-REPORT.md](./CONTRACT-ALIGNMENT-REPORT.md) - 契约对齐报告

---

**最后更新**: 2026-09-03 (v0.3.1-contract-alignment)
