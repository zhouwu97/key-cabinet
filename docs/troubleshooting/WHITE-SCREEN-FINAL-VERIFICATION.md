# 白屏问题最终验收清单

**状态**: 🟡 等待微信开发者工具验证  
**日期**: 2026-09-03  
**提交**: fd28a41

---

## 已修复的问题

### 1. ✅ 首页/钥匙页白屏（根因）
- **问题**: `module 'services.js' is not defined`
- **根因**: 微信小程序不会自动解析目录导入 `'../../services'` → `'../../services/index'`
- **修复**: 12 个文件全部改为显式 `/index` 导入
- **提交**: `5d5e5f0`

### 2. ✅ 个人中心白屏
- **问题**: 
  - `onLoad()` 和 `onShow()` 重复加载数据
  - 缺少 loading/empty 状态渲染
  - User 字段映射错误：`studentId` → `studentNo`
  - 引用了不存在的 `department` 字段
- **修复**: 
  - 移除 `profile.ts` 的 `onLoad()`
  - 添加 `wx:if/wx:elif/wx:else` 状态分支
  - 修正字段为 `user.studentNo || '--'`
  - 移除 `department` 字段显示
- **提交**: `fd28a41`

### 3. ✅ status-badge null 警告
- **问题**: 组件初始化时 `text/tone` 属性为 null
- **修复**: 所有调用点添加 fallback
  - `key-card`: `statusLabel || ''`, `statusTone || 'gray'`
  - `reservation-card`: `statusLabel || ''`
  - `device-status`: `statusLabel || ''`, `tone || 'gray'`
- **提交**: `fd28a41`

---

## 验收标准（请逐项验证）

### 前置步骤（必须执行）

```bash
# 1. 确认代码已拉取
git pull origin main
git log --oneline -1
# 应显示: fd28a41 fix(miniprogram): fix profile page white screen and status-badge null warnings
```

**在微信开发者工具中**:

```
2. 工具 → 清除缓存 → 全部清除
3. 关闭微信开发者工具
4. 重新打开项目
5. 编译
```

---

### 验收项

#### ✅ 基础验收（必须全部通过）

1. **首页渲染**
   - [ ] 设备状态卡片显示
   - [ ] "我的借用"区域显示（即使为空）
   - [ ] "我的预约"区域显示（即使为空）
   - [ ] "快捷服务"按钮组显示
   - [ ] 底部 TabBar 正常

2. **钥匙页渲染**
   - [ ] 钥匙列表显示
   - [ ] 筛选按钮正常
   - [ ] 搜索框正常
   - [ ] 每张卡片的 status-badge 正常

3. **记录页渲染**
   - [ ] 顶部 Tab 切换正常
   - [ ] 借用记录列表显示
   - [ ] 预约记录列表显示
   - [ ] status-badge 正常

4. **个人中心渲染**
   - [ ] 用户信息卡片显示
   - [ ] 显示"张三"和学号（Mock 数据）
   - [ ] 统计看板显示（当前借用/待履约/累计/归还率）
   - [ ] "常见服务与指南"菜单显示
   - [ ] "管理员与系统控制台"菜单显示

#### ✅ Console 检查（必须通过）

5. **错误检查**
   - [ ] Console 红色错误数 = **0**
   - [ ] 没有 `module 'services.js' is not defined`
   - [ ] 没有 `page register error`

6. **警告检查**
   - [ ] 没有 `status-badge` 的 `property "text" ... get null value`
   - [ ] 没有 `status-badge` 的 `property "tone" ... get null value`
   - [ ] 可以忽略微信开发者工具自己的警告（如 `WAServiceMainContext preload`）

#### ✅ 稳定性验收（必须通过）

7. **Tab 切换测试**
   - [ ] 首页 → 钥匙 → 记录 → 我的 → 首页，连续切换 **10 次**
   - [ ] Console 不新增红色错误
   - [ ] 每个页面都正常渲染

8. **冷启动测试**
   - [ ] 工具 → 清除缓存 → 全部清除
   - [ ] 编译
   - [ ] 所有页面正常
   - [ ] Console 无红色错误

---

## 验收结果记录

**测试人**: ___________  
**测试时间**: ___________  

### 结果

- [ ] ✅ **全部通过** → 可以进入 Sprint 4.2
- [ ] ❌ **未通过** → 继续调试

### 如果未通过，记录以下信息

**失败的验收项**: ___________

**Console 错误截图**: （粘贴到下方）

**页面截图**: （粘贴到下方）

**具体现象描述**:

```
（描述问题）
```

---

## 通过后的下一步：Sprint 4.2

### Sprint 4.2 目标

**完整实现微信登录 → 真实后端 → 个人中心显示真实用户**

### 核心链路

```
小程序
  ↓
wx.login() → code
  ↓
POST /api/v1/auth/wechat-login
  ↓
Go Backend
  ↓
微信 code2Session → openid
  ↓
查找/创建 UserIdentity (WECHAT, openid)
  ↓
查找/创建 User
  ↓
签发 JWT
  ↓
小程序保存 accessToken
  ↓
GET /api/v1/me (Authorization: Bearer <JWT>)
  ↓
个人中心显示真实用户信息
```

### Sprint 4.2 验收标准

- ✅ 删除 `MockUserService`
- ✅ 小程序启动自动微信登录
- ✅ 后端 `/api/v1/auth/wechat-login` 接口工作
- ✅ 后端 `/api/v1/me` 接口工作
- ✅ 个人中心显示真实 User 数据（从后端返回）
- ✅ JWT 过期后自动重新登录
- ✅ 所有需要认证的接口携带 JWT

---

## 技术债务清单（Sprint 4.3+ 处理）

1. **User.department 字段**: 
   - 当前已移除显示
   - 如需要，在 Sprint 4.2 User API 设计时正式加入

2. **Mock 数据清理**:
   - Sprint 4.2: MockUserService → 真实 UserService
   - Sprint 4.3: MockKeyService → 真实 KeyService
   - Sprint 4.4: MockReservationService → 真实 ReservationService
   - Sprint 4.5: MockBorrowService → 真实 BorrowService

3. **重复数据加载优化**:
   - 当前 `keys/records` 页面仍然 `onLoad()` + `onShow()` 都加载
   - 建议统一改为只在 `onShow()` 加载

---

**此文档用于最终验收，通过后归档到 `docs/troubleshooting/` 作为白屏问题的完整解决记录。**
