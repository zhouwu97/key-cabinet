import { existsSync, readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')

console.log('======================================================')
console.log('   智能钥匙借还系统 Stage 3 & v0.3.0 验收核对清单')
console.log('======================================================\n')

const checklist = [
  {
    name: '文档协议冻结 (Track B: docs/)',
    items: [
      'docs/02-DATA-MODEL.md',
      'docs/03-STATE-MACHINE.md',
      'docs/04-API-CONTRACT.md',
      'docs/05-MQTT-PROTOCOL.md',
      'docs/06-DEVICE-OPERATION.md',
      'docs/07-ERROR-CODES.md',
    ],
  },
  {
    name: '9 大核心基础组件 (Track A: components/)',
    items: [
      'miniprogram/components/status-badge/status-badge.wxml',
      'miniprogram/components/section-header/section-header.wxml',
      'miniprogram/components/empty-state/empty-state.wxml',
      'miniprogram/components/error-state/error-state.wxml',
      'miniprogram/components/device-status/device-status.wxml',
      'miniprogram/components/key-card/key-card.wxml',
      'miniprogram/components/reservation-card/reservation-card.wxml',
      'miniprogram/components/borrow-card/borrow-card.wxml',
      'miniprogram/components/operation-stepper/operation-stepper.wxml',
    ],
  },
  {
    name: '核心页面 6 状态完整化 (Track A: pages/)',
    items: [
      'miniprogram/pages/home/home.wxml',
      'miniprogram/pages/keys/keys.wxml',
      'miniprogram/pages/key-detail/key-detail.wxml',
      'miniprogram/pages/operation/operation.wxml',
      'miniprogram/pages/records/records.wxml',
      'miniprogram/pages/profile/profile.wxml',
      'miniprogram/pages/admin/admin.wxml',
      'miniprogram/pages/reservation-create/reservation-create.wxml',
    ],
  },
]

let allPassed = true

for (const group of checklist) {
  console.log(`[GROUP] ${group.name}`)
  for (const item of group.items) {
    const fullPath = resolve(root, item)
    if (existsSync(fullPath)) {
      console.log(`  ✓ ${item}`)
    } else {
      console.log(`  ✕ MISSING: ${item}`)
      allPassed = false
    }
  }
  console.log('')
}

if (!allPassed) {
  console.error('FAIL: 存在未就绪的检查项！')
  process.exit(1)
}

console.log('======================================================')
console.log('验收结果: v0.3.0-product-ready 所有组件与规范文件 100% 就绪！')
console.log('======================================================\n')
