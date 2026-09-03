# Sprint 4.1.1 执行检查清单

## 已完成 ✅

### P0: 契约收敛
- [x] 确定 User 字段: studentNo (不是 studentId)
- [x] 确定角色: USER | ADMIN (MAINTAINER 保留 v0.5)
- [x] 确定状态: ACTIVE | DISABLED
- [x] Reservation 时间语义统一: pickup_window_start/end + expected_return_at
- [x] BorrowRecord.borrowed_at 改为 nullable
- [x] DeviceOperation 字段统一: request_id/user_id/error_code
- [x] API Response envelope 统一

### P0: Database
- [x] 创建 000003_fix_schema_alignment.up.sql
  - [x] users.student_id → student_no
  - [x] users 添加 credit_score, department
  - [x] user_identities: identity_type → provider, provider_user_id → subject
  - [x] 全部时间字段改 TIMESTAMPTZ
  - [x] reservations 添加 pickup_window_start/end/expected_return_at
  - [x] reservations 删除 start_time/end_time
  - [x] borrow_records.borrowed_at 改 nullable
  - [x] device_operations 添加 request_id (UNIQUE), user_id, error_code
- [x] 创建 000004_create_operation_events.up.sql
- [x] 创建 000005_fix_reservation_constraint.up.sql
- [x] 所有 down migration 文件

### P0: Backend Foundation
- [x] module 改成 github.com/zhouwu97/key-cabinet/server
- [x] Go 版本统一: 1.26
- [x] Dockerfile Go 版本: 1.26-alpine
- [x] docker-compose env 添加 KC_ 前缀

### P1: Engineering
- [x] 添加 .github/workflows/ci.yml
  - [x] Frontend check/test
  - [x] Backend test/vet/fmt
  - [x] Migration CI
  - [x] Docker build CI

### P1: Frontend Integration Architecture
- [x] 创建 HttpClient (api/http-client.ts)
- [x] 创建 ApiError 类型
- [x] 创建 AuthService (services/auth/auth-service.ts)
- [x] 创建 ApiUserService (services/user/api-user-service.ts)
- [x] 创建 config/index.ts
- [x] 保留 Mock Service (不删除)

### P1: Documentation
- [x] 更新 DATA-MODEL.md: studentId → studentNo
- [x] 更新 API-CONTRACT.md: studentId → studentNo
- [x] 更新前端 models/user.ts
- [x] 创建 SPRINT-4.1.1-CONTRACT-ALIGNMENT.md
- [x] 创建 SPRINT-4.1.1-SUMMARY.md
- [x] 创建验收脚本 verify-sprint-4.1.1.sh

## Git 状态

```bash
Commit: 29e9377
Message: feat(sprint-4.1.1): contract alignment and infrastructure setup
Files Changed: 31 files, 2018 insertions(+), 34 deletions(-)
Status: ✅ Committed to main branch
```

## 下一步行动

### 立即执行
1. [ ] Push to GitHub: `git push origin main`
2. [ ] 验证 GitHub Actions CI 通过
3. [ ] 应用 migration 到本地数据库

### Sprint 4.2 准备
1. [ ] 注册微信小程序测试账号
2. [ ] 配置 KC_WECHAT_APP_ID 和 KC_WECHAT_APP_SECRET
3. [ ] 阅读 Sprint 4.2 计划（已简化）

## 验收命令

```bash
# 本地验证
bash scripts/verify-sprint-4.1.1.sh

# 数据库验证（需 PostgreSQL）
docker-compose -f server/docker-compose.yml up postgres -d
cd server
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/keycabinet?sslmode=disable"
migrate -path ./migrations -database "$DATABASE_URL" up

# 验证表结构
psql "$DATABASE_URL" -c "\d users"
psql "$DATABASE_URL" -c "\d reservations"
psql "$DATABASE_URL" -c "\d device_operations"
psql "$DATABASE_URL" -c "\d operation_events"
```

## Sprint 4.1.1 完成 ✅

**契约对齐**: ✅ 完成  
**基础设施**: ✅ 就绪  
**CI/CD**: ✅ 建立  
**文档**: ✅ 同步

**可以进入 Sprint 4.2**: ✅
