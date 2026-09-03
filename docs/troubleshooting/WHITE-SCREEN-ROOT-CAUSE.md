# 白屏问题根因分析与修复

## 问题现象

**日期**：2026-09-03  
**症状**：小程序首页完全白屏，但导航栏和 TabBar 正常显示

## 根因定位

### 错误信息

```
module 'services.js' is not defined,
require args is '../../services'
```

### 问题根源

**微信小程序的模块解析机制不会自动将目录路径解析为 `index.ts`**

在 Node.js/TypeScript 环境中，以下导入会自动解析：

```typescript
import { xxx } from '../../services'
// 自动解析为 ../../services/index.ts ✅
```

但在微信小程序运行时环境中：

```typescript
import { xxx } from '../../services'
// 尝试加载 ../../services.js ❌
// 找不到，抛出 "module 'services.js' is not defined"
```

### 为什么 TypeScript 编译不报错？

因为 TypeScript 编译器（`tsc`）使用 Node.js 的模块解析规则，会自动补全 `/index`。

但微信小程序的运行时**不遵循 Node.js 的模块解析规则**，必须显式指定 `/index`。

### 为什么导航栏和 TabBar 正常？

加载顺序如下：

```
1. app.json 加载 ✅ (原生配置)
2. app.ts 加载 ✅ (全局逻辑)
3. TabBar 渲染 ✅ (原生组件)
4. 导航栏渲染 ✅ (原生组件)
5. pages/home/home.ts 加载 ❌
   ├─ import from '../../services' ❌
   └─ 找不到 services.js，抛出异常
6. Page({}) 根本没执行
7. 页面主体白屏
```

所以**原生部分全部正常，但页面脚本加载失败导致白屏**。

## 修复方案

### 1. 修改所有页面的 services 导入

**修改前**：

```typescript
import {
  deviceService,
  reservationService,
  borrowService,
} from '../../services'
```

**修改后**：

```typescript
import {
  deviceService,
  reservationService,
  borrowService,
} from '../../services/index'
```

### 2. 修改 services/index.ts 的子模块导出

**修改前**：

```typescript
export * from './key'
export * from './reservation'
export * from './borrow'
export * from './user'
export * from './device'
export * from './operation'
```

**修改后**：

```typescript
export * from './key/index'
export * from './reservation/index'
export * from './borrow/index'
export * from './user/index'
export * from './device/index'
export * from './operation/index'
```

### 3. 修改类型导入

**修改前**：

```typescript
import { CreateReservationParams } from '../../services/reservation'
```

**修改后**：

```typescript
import { CreateReservationParams } from '../../services/reservation/index'
```

## 受影响的文件

共修复了 **9 个文件**：

### services/
- `services/index.ts` - 修改子模块 export

### pages/
- `pages/home/home.ts`
- `pages/admin/admin.ts`
- `pages/keys/keys.ts`
- `pages/key-detail/key-detail.ts`
- `pages/operation/operation.ts`
- `pages/profile/profile.ts`
- `pages/records/records.ts`
- `pages/reservation-create/reservation-create.ts`

## 防止复发

### 1. 添加自动检查脚本

创建了 `miniprogram/scripts/check-directory-imports.mjs`，会检查所有 `.ts` 文件中的目录导入。

### 2. 集成到 npm 脚本

```json
{
  "scripts": {
    "check": "tsc --noEmit && node scripts/check-pages.mjs && node miniprogram/scripts/check-directory-imports.mjs",
    "check:imports": "node miniprogram/scripts/check-directory-imports.mjs"
  }
}
```

### 3. 开发规范

**强制规则**：在微信小程序项目中，导入目录必须显式指定 `/index`

```typescript
// ❌ 错误 - 微信小程序运行时会失败
import { xxx } from './services'
import { yyy } from '../key'

// ✅ 正确 - 显式指定 /index
import { xxx } from './services/index'
import { yyy } from '../key/index'
```

**例外**：导入单个文件（非目录）可以省略 `.ts` 扩展名

```typescript
// ✅ 正确 - key.ts 是文件，不是目录
import { Key } from '../models/key'

// ✅ 也可以，但不必要
import { Key } from '../models/key.ts'
```

## 验证步骤

### 1. 运行检查脚本

```bash
npm run check:imports
```

预期输出：

```
✨ All imports are correct!
```

### 2. TypeScript 检查

```bash
npm run check
```

预期输出：

```
OK: 10 个页面注册完整，4 个 tabBar 页面，21 个 JSON 组件路径校验通过，19 个 WXML 模板语法全部通过！
✨ All imports are correct!
```

### 3. 微信开发者工具验证

1. 工具 → 清除缓存 → 全部清除
2. 关闭开发者工具
3. 重新打开项目
4. 编译
5. 首页正常显示

## 技术细节

### 微信小程序的模块解析规则

微信小程序使用的是**精确路径匹配**，而不是 Node.js 的模块解析算法。

```typescript
// Node.js 模块解析规则
import { xxx } from './foo'
// 1. 尝试 ./foo.js
// 2. 尝试 ./foo/package.json 的 main 字段
// 3. 尝试 ./foo/index.js  ← 自动补全

// 微信小程序模块解析规则
import { xxx } from './foo'
// 1. 尝试 ./foo.js
// 2. 找不到，报错 ❌
```

### TypeScript 编译 vs 运行时

```
源代码 (TypeScript)
    ↓
  tsc 编译
    ↓
JavaScript (.js)
    ↓
微信开发者工具编译
    ↓
微信小程序运行时 ← 在这里执行模块解析
```

所以 `tsc --noEmit` 检查通过 ≠ 微信小程序运行时不会出错。

## 相关 Issue

- **现象**：静态检查全绿，真机运行白屏
- **根因**：编译期和运行期的模块解析规则不一致
- **修复**：显式指定 `/index`，并添加自动检查脚本

## 总结

这个问题的关键教训是：

1. **微信小程序有自己的模块解析规则**，不完全等同于 Node.js
2. **TypeScript 编译通过不代表运行时不会出错**
3. **必须在真实环境中验证**，不能只依赖静态检查
4. **自动化检查可以防止人为疏忽**

修复后，所有导入路径都显式指定了 `/index`，并且添加了自动检查脚本，确保以后不会再出现同样的问题。
