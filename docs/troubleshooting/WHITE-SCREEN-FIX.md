# 白屏问题修复指南

## 问题描述
小程序首页出现白屏，顶部标题和底部 TabBar 正常显示，但页面内容区域空白。

## 根因分析

### 1. 全局组件注册冲突
在 commit `2aa08b4` 中，将 9 个自定义组件全部注册到了 `app.json` 的全局 `usingComponents`：

```json
"usingComponents": {
  "status-badge": "/components/status-badge/status-badge",
  "section-header": "/components/section-header/section-header",
  "empty-state": "/components/empty-state/empty-state",
  "error-state": "/components/error-state/error-state",
  "device-status": "/components/device-status/device-status",
  "key-card": "/components/key-card/key-card",
  "reservation-card": "/components/reservation-card/reservation-card",
  "borrow-card": "/components/borrow-card/borrow-card",
  "operation-stepper": "/components/operation-stepper/operation-stepper"
}
```

同时，各个页面的 `.json` 文件也注册了这些组件（例如 `home.json`）。

**问题**：全局和局部重复注册导致组件路径解析冲突，加上 `lazyCodeLoading: "requiredComponents"` 配置，可能导致组件加载失败。

### 2. 重复数据加载
`pages/home/home.ts` 中同时在 `onLoad()` 和 `onShow()` 调用 `loadData()`：

```typescript
onLoad() {
  this.loadData()  // 第一次调用
},

onShow() {
  this.loadData()  // 第二次调用
},
```

首次进入首页时，小程序生命周期会依次触发 `onLoad` → `onShow`，导致 `loadData()` 被执行两次，造成：
- API 重复请求
- Mock Service 状态重复变化
- 潜在的竞态条件

## 已实施的修复

### 修复 1: 移除全局组件注册
**文件**: `miniprogram/app.json`

**修改**:
```diff
  "pages": [...],
- "usingComponents": {
-   "status-badge": "/components/status-badge/status-badge",
-   ...9个组件
- },
  "window": {...}
```

**原理**: 
- 让每个页面在自己的 `.json` 文件中按需注册组件
- 避免全局和局部注册冲突
- 配合 `lazyCodeLoading: "requiredComponents"` 实现按需加载

### 修复 2: 移除重复的 onLoad 数据加载
**文件**: `miniprogram/pages/home/home.ts`

**修改**:
```diff
  onLoad() {
-   this.loadData()
+   // Removed: loadData() will be called in onShow()
  },

  onShow() {
    this.loadData()
  },
```

**原理**:
- 只在 `onShow()` 加载数据
- 首次进入和每次显示页面时都会执行 `onShow()`
- 避免重复请求和状态混乱

## 验证步骤

### 1. 清除缓存并重新编译
在微信开发者工具中：
1. 点击菜单：**工具 → 清除缓存 → 全部清除**
2. 点击 **编译** 按钮

### 2. 检查 Console 错误
打开开发者工具底部的 **Console** 面板，查看是否还有红色错误信息。

如果仍有错误，请记录：
- 错误信息完整内容
- 错误发生的文件和行号
- 错误类型（组件加载失败/API 请求失败/其他）

### 3. 二分定位法（如果仍然白屏）

#### 步骤 1: 简化页面
临时修改 `miniprogram/pages/home/home.wxml`，只保留基本内容：

```xml
<view style="padding:40rpx;background:#fff;color:#000;">
  首页渲染正常
</view>
```

同时修改 `miniprogram/pages/home/home.json`，移除所有组件：

```json
{
  "navigationBarTitleText": "智能钥匙自助借还"
}
```

**预期结果**: 如果这时显示 "首页渲染正常"，说明问题出在自定义组件或复杂的 WXML 结构上。

#### 步骤 2: 逐个添加组件
按以下顺序逐个添加组件到 `home.json` 和 `home.wxml`：

1. `section-header`
2. `device-status`
3. `borrow-card`
4. `reservation-card`
5. `status-badge`
6. `empty-state`

每添加一个组件就编译一次，观察哪个组件导致白屏。

### 4. 检查组件文件完整性
确认以下文件都存在且格式正确：

```
components/
├── status-badge/
│   ├── status-badge.wxml
│   ├── status-badge.ts
│   ├── status-badge.json
│   └── status-badge.wxss
├── section-header/
│   ├── section-header.wxml
│   ├── section-header.ts
│   ├── section-header.json
│   └── section-header.wxss
...
```

特别检查每个组件的 `.json` 文件是否包含 `"component": true`：

```json
{
  "component": true,
  "usingComponents": {}
}
```

## 常见问题

### Q1: 清除缓存后仍然白屏
**A**: 使用二分定位法，从最简单的页面开始，逐步添加内容，找出具体是哪个组件或代码块导致问题。

### Q2: Console 报 "Component is not found in path"
**A**: 检查组件路径是否正确，路径必须以 `/` 开头，指向 `miniprogram/` 根目录的相对路径。

### Q3: 组件路径正确但仍然加载失败
**A**: 检查组件的 4 个文件（.wxml, .ts, .json, .wxss）是否都存在，特别是 `.json` 文件必须包含 `"component": true`。

### Q4: 修复后数据不显示
**A**: 这是正常的，目前前端还在使用 Mock Service。数据显示问题属于业务逻辑层，与白屏问题（渲染层）是两个层面的问题。

## 后续建议

### 1. 组件注册原则
- **不要使用全局 `usingComponents`**，除非是真正在所有页面都使用的基础组件
- 每个页面在自己的 `.json` 中按需注册
- 配合 `lazyCodeLoading: "requiredComponents"` 实现按需加载

### 2. 生命周期使用规范
- `onLoad()`: 只做一次性的初始化工作（获取页面参数、设置初始状态）
- `onShow()`: 每次显示页面时需要更新的数据加载
- `onReady()`: DOM 渲染完成后的操作

### 3. 错误监控
在 `app.ts` 中添加全局错误监听：

```typescript
App({
  onError(error: string) {
    console.error('全局错误捕获:', error)
    // 可以上报到错误监控服务
  },
})
```

## 提交记录
- Commit: `39b94bb` - fix(miniprogram): remove global component registration and duplicate onLoad data fetching
- Commit: `401ff3c` - feat(backend): complete Sprint 4.1 - Backend Foundation

## 相关文档
- [微信小程序自定义组件](https://developers.weixin.qq.com/miniprogram/dev/framework/custom-component/)
- [微信小程序页面生命周期](https://developers.weixin.qq.com/miniprogram/dev/framework/app-service/page-life-cycle.html)
- [代码分包加载](https://developers.weixin.qq.com/miniprogram/dev/framework/subpackages/basic.html)
