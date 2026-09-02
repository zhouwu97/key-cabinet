/**
 * 智能钥匙自助借还系统 阶段 2 (2A/2B/2C) 全场景自动化测试套件
 * 运行方式: node scripts/run-stage2-tests.mjs
 */

import { execSync } from 'node:child_process'
import { createRequire } from 'node:module'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import assert from 'node:assert/strict'

const __dirname = dirname(fileURLToPath(import.meta.url))
const rootDir = resolve(__dirname, '..')
const require = createRequire(import.meta.url)

// 1. 编译 TypeScript 到测试目录
console.log('Compiling TypeScript for test suite...')
execSync('npx tsc -p tsconfig.test.json', { cwd: rootDir, stdio: 'inherit' })

// 2. 初始化微信小程序全局环境 Mock
const storage = new Map()

globalThis.wx = {
  getStorageSync: key => storage.get(key) || null,
  setStorageSync: (key, val) => storage.set(key, JSON.parse(JSON.stringify(val))),
  removeStorageSync: key => storage.delete(key),
  clearStorageSync: () => storage.clear(),
  showToast: () => {},
  showLoading: () => {},
  hideLoading: () => {},
  showModal: async () => ({ confirm: true }),
}

// 3. 动态加载编译后的模块
const {
  MOCK_USERS,
  MOCK_KEYS,
  MOCK_KEY_SLOTS,
  MOCK_DEVICES,
  MOCK_RESERVATIONS,
  MOCK_BORROW_RECORDS,
  STORAGE_KEYS,
} = require(resolve(rootDir, '.test-dist/mocks/mock-data.js'))

const { MockScenario } = require(resolve(rootDir, '.test-dist/mocks/mock-scenarios.js'))
const { KeyStatus } = require(resolve(rootDir, '.test-dist/models/key.js'))
const { KeyPresenceState } = require(resolve(rootDir, '.test-dist/models/key-presence.js'))
const { ReservationStatus } = require(resolve(rootDir, '.test-dist/models/reservation.js'))
const { BorrowRecordStatus, isRecordOverdue } = require(resolve(rootDir, '.test-dist/models/borrow-record.js'))
const { DeviceOperationAction, DeviceOperationStatus } = require(resolve(rootDir, '.test-dist/models/device-operation.js'))
const { OperationErrorCode } = require(resolve(rootDir, '.test-dist/models/operation-error.js'))

const { MockKeyService } = require(resolve(rootDir, '.test-dist/services/key/mock-key-service.js'))
const { MockReservationService } = require(resolve(rootDir, '.test-dist/services/reservation/mock-reservation-service.js'))
const { MockBorrowService } = require(resolve(rootDir, '.test-dist/services/borrow/mock-borrow-service.js'))
const { MockDeviceService } = require(resolve(rootDir, '.test-dist/services/device/mock-device-service.js'))
const { MockOperationService } = require(resolve(rootDir, '.test-dist/services/operation/mock-operation-service.js'))

function resetEnvironment() {
  wx.clearStorageSync()
  wx.setStorageSync(STORAGE_KEYS.USERS, JSON.parse(JSON.stringify(MOCK_USERS)))
  wx.setStorageSync(STORAGE_KEYS.KEYS, JSON.parse(JSON.stringify(MOCK_KEYS)))
  wx.setStorageSync(STORAGE_KEYS.KEY_SLOTS, JSON.parse(JSON.stringify(MOCK_KEY_SLOTS)))
  wx.setStorageSync(STORAGE_KEYS.DEVICES, JSON.parse(JSON.stringify(MOCK_DEVICES)))
  wx.setStorageSync(STORAGE_KEYS.RESERVATIONS, JSON.parse(JSON.stringify(MOCK_RESERVATIONS)))
  wx.setStorageSync(STORAGE_KEYS.BORROW_RECORDS, JSON.parse(JSON.stringify(MOCK_BORROW_RECORDS)))
  wx.setStorageSync(STORAGE_KEYS.OPERATIONS, [])
  wx.removeStorageSync(STORAGE_KEYS.ACTIVE_OPERATION_ID)

  const keyService = new MockKeyService()
  const reservationService = new MockReservationService()
  const borrowService = new MockBorrowService()
  const deviceService = new MockDeviceService()
  const operationService = new MockOperationService(
    deviceService,
    keyService,
    reservationService,
    borrowService,
  )

  return { keyService, reservationService, borrowService, deviceService, operationService }
}

const sleep = ms => new Promise(r => setTimeout(r, ms))

let passedCount = 0
let failedCount = 0

async function runTest(testId, description, fn) {
  process.stdout.write(`  [TEST] ${testId}: ${description} ... `)
  try {
    await fn()
    console.log('\x1b[32mPASSED\x1b[0m')
    passedCount++
  } catch (err) {
    console.log('\x1b[31mFAILED\x1b[0m')
    console.error('    Error:', err.message || err)
    if (err.stack) console.error('    Stack:', err.stack.split('\n')[1])
    failedCount++
  }
}

async function main() {
  console.log('\n======================================================')
  console.log('   智能钥匙自助借还系统 阶段 2 (2A/2B/2C) 自动化验证套件')
  console.log('======================================================\n')

  // ---------------------------------------------------------
  // 一、正常业务流程测试 (T001 ~ T007)
  // ---------------------------------------------------------
  console.log('--- 1. 正常业务闭环测试 ---')

  await runTest('T001', '创建预约 (Create Reservation)', async () => {
    const { reservationService } = resetEnvironment()
    const rsv = await reservationService.createReservation({
      userId: 'U001',
      keyId: 'KEY101',
      purpose: '测试实验课',
      expectedDuration: 7200000,
    })
    assert.ok(rsv.id.startsWith('RSV'))
    assert.equal(rsv.keyId, 'KEY101')
    assert.equal(rsv.status, ReservationStatus.ACTIVE)
  })

  await runTest('T002', '取消预约 (Cancel Reservation)', async () => {
    const { reservationService, keyService } = resetEnvironment()
    const rsv = await reservationService.createReservation({
      userId: 'U001',
      keyId: 'KEY102',
      purpose: '测试取消',
    })
    await reservationService.cancelReservation(rsv.id)
    const cancelled = await reservationService.getReservationById(rsv.id)
    assert.equal(cancelled.status, ReservationStatus.CANCELLED)

    const key = await keyService.getKeyById('KEY102')
    assert.equal(key.status, KeyStatus.AVAILABLE)
  })

  await runTest('T003', '预约 → 发起取钥 (Reservation -> Pickup Operation)', async () => {
    const { reservationService, operationService } = resetEnvironment()
    const rsv = await reservationService.createReservation({
      userId: 'U001',
      keyId: 'KEY103',
      purpose: '取钥测试',
    })
    const op = await operationService.startOperation({
      action: DeviceOperationAction.PICKUP,
      userId: 'U001',
      keyId: 'KEY103',
      deviceId: 'CAB001',
      reservationId: rsv.id,
    })
    assert.ok(op.id.startsWith('OP'))
    assert.equal(op.action, DeviceOperationAction.PICKUP)
    assert.equal(op.status, DeviceOperationStatus.EXECUTING)
  })

  await runTest('T004', '取钥事件流执行 → 借用中 & 钥匙离柜 (Pickup Success)', async () => {
    const { reservationService, operationService, keyService, borrowService } = resetEnvironment()
    const rsv = await reservationService.createReservation({
      userId: 'U001',
      keyId: 'KEY101',
      purpose: '出柜测试',
    })
    const op = await operationService.startOperation({
      action: DeviceOperationAction.PICKUP,
      userId: 'U001',
      keyId: 'KEY101',
      deviceId: 'CAB001',
      reservationId: rsv.id,
    }, MockScenario.SUCCESS)

    // 等待模拟事件时序走完
    await sleep(8500)

    const updatedOp = await operationService.getOperation(op.id)
    assert.equal(updatedOp.status, DeviceOperationStatus.SUCCESS)

    const key = await keyService.getKeyById('KEY101')
    assert.equal(key.status, KeyStatus.BORROWED)

    const slot = await keyService.getKeySlot('SLOT01')
    assert.equal(slot.presence, KeyPresenceState.ABSENT)

    const updatedRsv = await reservationService.getReservationById(rsv.id)
    assert.equal(updatedRsv.status, ReservationStatus.USED)

    const borrows = await borrowService.getCurrentBorrows('U001')
    const activeBorrow = borrows.find(b => b.keyId === 'KEY101')
    assert.ok(activeBorrow)
    assert.equal(activeBorrow.status, BorrowRecordStatus.BORROWED)
  })

  await runTest('T005', '借用中 → 发起归还 (BorrowRecord -> Return Operation)', async () => {
    const { operationService, borrowService } = resetEnvironment()
    const borrows = await borrowService.getCurrentBorrows('U001')
    const borrow = borrows.find(b => b.keyId === 'KEY104')
    assert.ok(borrow)

    const op = await operationService.startOperation({
      action: DeviceOperationAction.RETURN,
      userId: 'U001',
      keyId: 'KEY104',
      deviceId: 'CAB001',
      borrowRecordId: borrow.id,
    })
    assert.equal(op.action, DeviceOperationAction.RETURN)
  })

  await runTest('T006', 'RFID 确认 → 归还成功完整闭环 (Return Success)', async () => {
    const { operationService, keyService, borrowService } = resetEnvironment()
    const borrows = await borrowService.getCurrentBorrows('U001')
    const borrow = borrows.find(b => b.keyId === 'KEY104')

    await operationService.startOperation({
      action: DeviceOperationAction.RETURN,
      userId: 'U001',
      keyId: 'KEY104',
      deviceId: 'CAB001',
      borrowRecordId: borrow.id,
    }, MockScenario.SUCCESS)

    await sleep(8500)

    const key = await keyService.getKeyById('KEY104')
    assert.equal(key.status, KeyStatus.AVAILABLE)

    const slot = await keyService.getKeySlot('SLOT04')
    assert.equal(slot.presence, KeyPresenceState.PRESENT)

    const updatedBorrow = await borrowService.getBorrowRecordById(borrow.id)
    assert.equal(updatedBorrow.status, BorrowRecordStatus.COMPLETED)
  })

  await runTest('T007', '逾期订单归还 (Overdue -> Return)', async () => {
    const { operationService, borrowService, keyService } = resetEnvironment()
    const record = await borrowService.getBorrowRecordById('BR002')
    assert.ok(isRecordOverdue(record))

    await operationService.startOperation({
      action: DeviceOperationAction.RETURN,
      userId: 'U001',
      keyId: 'KEY106',
      deviceId: 'CAB001',
      borrowRecordId: record.id,
    }, MockScenario.SUCCESS)

    await sleep(8500)

    const completed = await borrowService.getBorrowRecordById('BR002')
    assert.equal(completed.status, BorrowRecordStatus.COMPLETED)
    const key = await keyService.getKeyById('KEY106')
    assert.equal(key.status, KeyStatus.AVAILABLE)
  })

  // ---------------------------------------------------------
  // 二、业务异常场景测试 (T101 ~ T106)
  // ---------------------------------------------------------
  console.log('\n--- 2. 业务异常场景测试 ---')

  await runTest('T101', '同一用户重复预约同一把钥匙应拦截', async () => {
    const { reservationService } = resetEnvironment()
    await reservationService.createReservation({
      userId: 'U001',
      keyId: 'KEY108',
      purpose: '首个预约',
    })
    await assert.rejects(
      async () => {
        await reservationService.createReservation({
          userId: 'U001',
          keyId: 'KEY108',
          purpose: '重复预约',
        })
      },
      err => err.message === OperationErrorCode.RESERVATION_CONFLICT,
    )
  })

  await runTest('T102', '两用户预约时间重叠冲突检测', async () => {
    const { reservationService } = resetEnvironment()
    const now = Date.now()
    await reservationService.createReservation({
      userId: 'U001',
      keyId: 'KEY109',
      pickupWindowStart: now,
      pickupWindowEnd: now + 1800000,
      expectedReturnAt: now + 7200000,
    })

    await assert.rejects(
      async () => {
        await reservationService.createReservation({
          userId: 'A001',
          keyId: 'KEY109',
          pickupWindowStart: now + 3600000, // 处于前一个预约的时间窗内部
          expectedReturnAt: now + 10800000,
        })
      },
      err => err.message === OperationErrorCode.RESERVATION_CONFLICT,
    )
  })

  await runTest('T103', '已停用/维护中钥匙不可预约', async () => {
    const { reservationService } = resetEnvironment()
    await assert.rejects(
      async () => {
        await reservationService.createReservation({
          userId: 'U001',
          keyId: 'KEY110', // 已停用
        })
      },
      err => err.message === OperationErrorCode.KEY_NOT_AVAILABLE,
    )
  })

  await runTest('T104', '已被借出的钥匙不可重复预约', async () => {
    const { reservationService } = resetEnvironment()
    await assert.rejects(
      async () => {
        await reservationService.createReservation({
          userId: 'U001',
          keyId: 'KEY104', // 借出中
        })
      },
      err => err.message === OperationErrorCode.KEY_ALREADY_BORROWED,
    )
  })

  await runTest('T105', '无有效预约的用户发起取钥被拦截', async () => {
    const { operationService } = resetEnvironment()
    await assert.rejects(
      async () => {
        await operationService.startOperation({
          action: DeviceOperationAction.PICKUP,
          userId: 'A001', // 该用户无 KEY108 的活跃预约
          keyId: 'KEY108',
          deviceId: 'CAB001',
        })
      },
      err => err.message === OperationErrorCode.RESERVATION_NOT_ACTIVE,
    )
  })

  await runTest('T106', '用户尝试归还非本人的借还单被拦截', async () => {
    const { operationService } = resetEnvironment()
    await assert.rejects(
      async () => {
        await operationService.startOperation({
          action: DeviceOperationAction.RETURN,
          userId: 'A001', // 不是 BR001 的借用人
          keyId: 'KEY104',
          deviceId: 'CAB001',
          borrowRecordId: 'BR001',
        })
      },
      err => err.message === OperationErrorCode.OPERATION_USER_MISMATCH,
    )
  })

  // ---------------------------------------------------------
  // 三、设备异常与故障注入测试 (T201 ~ T210)
  // ---------------------------------------------------------
  console.log('\n--- 3. 设备与硬件异常故障注入测试 ---')

  await runTest('T201', '设备离线故障注入 (DEVICE_OFFLINE)', async () => {
    const { operationService } = resetEnvironment()
    await assert.rejects(
      async () => {
        await operationService.startOperation({
          action: DeviceOperationAction.PICKUP,
          userId: 'U001',
          keyId: 'KEY105',
          deviceId: 'CAB002', // CAB002 为离线设备
          reservationId: 'RSV001',
        })
      },
      err => err.message === OperationErrorCode.DEVICE_OFFLINE,
    )
  })

  await runTest('T202', '设备正忙故障注入 (DEVICE_BUSY)', async () => {
    const { operationService } = resetEnvironment()
    // 先发起一个正常的任务占用设备
    await operationService.startOperation({
      action: DeviceOperationAction.PICKUP,
      userId: 'U001',
      keyId: 'KEY105',
      deviceId: 'CAB001',
      reservationId: 'RSV001',
    })

    // 立即发起第二个任务
    await assert.rejects(
      async () => {
        await operationService.startOperation({
          action: DeviceOperationAction.PICKUP,
          userId: 'U001',
          keyId: 'KEY101',
          deviceId: 'CAB001',
        })
      },
      err => err.message === OperationErrorCode.OPERATION_DUPLICATED || err.message === OperationErrorCode.DEVICE_BUSY,
    )
  })

  await runTest('T203', '电机定位失败 (MOTOR_POSITION_FAILED)', async () => {
    const { operationService } = resetEnvironment()
    const op = await operationService.startOperation({
      action: DeviceOperationAction.PICKUP,
      userId: 'U001',
      keyId: 'KEY105',
      deviceId: 'CAB001',
      reservationId: 'RSV001',
    }, MockScenario.POSITION_ERROR)

    await sleep(3000)

    const updated = await operationService.getOperation(op.id)
    assert.equal(updated.status, DeviceOperationStatus.FAILED)
    assert.equal(updated.errorCode, OperationErrorCode.MOTOR_POSITION_FAILED)
  })

  await runTest('T204', '安全柜门开启失败 (DOOR_OPEN_FAILED)', async () => {
    const { operationService } = resetEnvironment()
    const op = await operationService.startOperation({
      action: DeviceOperationAction.PICKUP,
      userId: 'U001',
      keyId: 'KEY105',
      deviceId: 'CAB001',
      reservationId: 'RSV001',
    }, MockScenario.DOOR_ERROR)

    await sleep(3500)

    const updated = await operationService.getOperation(op.id)
    assert.equal(updated.status, DeviceOperationStatus.FAILED)
    assert.equal(updated.errorCode, OperationErrorCode.DOOR_OPEN_FAILED)
  })

  await runTest('T207', 'RFID 未检测到标签 (RFID_NOT_FOUND)', async () => {
    const { operationService, borrowService } = resetEnvironment()
    const borrows = await borrowService.getCurrentBorrows('U001')
    const borrow = borrows.find(b => b.keyId === 'KEY104')

    const op = await operationService.startOperation({
      action: DeviceOperationAction.RETURN,
      userId: 'U001',
      keyId: 'KEY104',
      deviceId: 'CAB001',
      borrowRecordId: borrow.id,
    }, MockScenario.RFID_NOT_FOUND)

    await sleep(6000)

    const updated = await operationService.getOperation(op.id)
    assert.equal(updated.status, DeviceOperationStatus.FAILED)
    assert.equal(updated.errorCode, OperationErrorCode.RFID_NOT_FOUND)
  })

  await runTest('T208', 'RFID 检测到错误钥匙拒绝归还 (RFID_WRONG_KEY)', async () => {
    const { operationService, borrowService, keyService } = resetEnvironment()
    const borrows = await borrowService.getCurrentBorrows('U001')
    const borrow = borrows.find(b => b.keyId === 'KEY104')

    const op = await operationService.startOperation({
      action: DeviceOperationAction.RETURN,
      userId: 'U001',
      keyId: 'KEY104',
      deviceId: 'CAB001',
      borrowRecordId: borrow.id,
    }, MockScenario.RFID_WRONG_KEY)

    await sleep(6000)

    const updated = await operationService.getOperation(op.id)
    assert.equal(updated.status, DeviceOperationStatus.FAILED)
    assert.equal(updated.errorCode, OperationErrorCode.RFID_WRONG_KEY)

    // 关键校验：归还错误钥匙绝不能将借还记录置为 COMPLETED
    const borrowCheck = await borrowService.getBorrowRecordById(borrow.id)
    assert.notEqual(borrowCheck.status, BorrowRecordStatus.COMPLETED)

    // 钥匙在位状态保持不变
    const slot = await keyService.getKeySlot('SLOT04')
    assert.equal(slot.presence, KeyPresenceState.ABSENT)
  })

  await runTest('T210', '操作响应超时 (OPERATION_TIMEOUT)', async () => {
    const { operationService } = resetEnvironment()
    const op = await operationService.startOperation({
      action: DeviceOperationAction.PICKUP,
      userId: 'U001',
      keyId: 'KEY105',
      deviceId: 'CAB001',
      reservationId: 'RSV001',
    }, MockScenario.TIMEOUT)

    await sleep(4500)

    const updated = await operationService.getOperation(op.id)
    assert.equal(updated.status, DeviceOperationStatus.FAILED)
    assert.equal(updated.errorCode, OperationErrorCode.OPERATION_TIMEOUT)
  })

  // ---------------------------------------------------------
  // 四、恢复与幂等性测试 (T301 ~ T307)
  // ---------------------------------------------------------
  console.log('\n--- 4. 恢复与幂等性测试 ---')

  await runTest('T301', '连续点击发起操作防抖与幂等', async () => {
    const { operationService } = resetEnvironment()
    const p1 = operationService.startOperation({
      action: DeviceOperationAction.PICKUP,
      userId: 'U001',
      keyId: 'KEY105',
      deviceId: 'CAB001',
      reservationId: 'RSV001',
    })
    const p2 = operationService.startOperation({
      action: DeviceOperationAction.PICKUP,
      userId: 'U001',
      keyId: 'KEY105',
      deviceId: 'CAB001',
      reservationId: 'RSV001',
    })

    const results = await Promise.allSettled([p1, p2])
    const fulfilled = results.filter(r => r.status === 'fulfilled')
    const rejected = results.filter(r => r.status === 'rejected')
    assert.equal(fulfilled.length, 1)
    assert.equal(rejected.length, 1)
  })

  await runTest('T305/T306', '操作执行中退出小程序后重启恢复现场 (Operation Recovery)', async () => {
    const { operationService } = resetEnvironment()
    const op = await operationService.startOperation({
      action: DeviceOperationAction.PICKUP,
      userId: 'U001',
      keyId: 'KEY105',
      deviceId: 'CAB001',
      reservationId: 'RSV001',
    })

    // 模拟应用重启，重新实例化 Service
    const restoredService = new MockOperationService(
      new MockDeviceService(),
      new MockKeyService(),
      new MockReservationService(),
      new MockBorrowService(),
    )

    const activeOp = await restoredService.resumeActiveOperation()
    assert.ok(activeOp)
    assert.equal(activeOp.id, op.id)
    assert.equal(activeOp.status, DeviceOperationStatus.EXECUTING)
  })

  // ---------------------------------------------------------
  // 五、多设备并发隔离测试 (T401 ~ T403)
  // ---------------------------------------------------------
  console.log('\n--- 5. 多设备并发隔离测试 ---')

  await runTest('T401', 'CAB001 忙不影响 CAB002 设备状态查询与独立性', async () => {
    const { operationService, deviceService } = resetEnvironment()
    // CAB001 开启任务
    await operationService.startOperation({
      action: DeviceOperationAction.PICKUP,
      userId: 'U001',
      keyId: 'KEY105',
      deviceId: 'CAB001',
      reservationId: 'RSV001',
    })

    assert.equal(deviceService.isDeviceBusy('CAB001'), true)
    assert.equal(deviceService.isDeviceBusy('CAB002'), false)
  })

  await runTest('T402', '同一 Key 严禁产生多个并发活动 Operation', async () => {
    const { operationService } = resetEnvironment()
    await operationService.startOperation({
      action: DeviceOperationAction.PICKUP,
      userId: 'U001',
      keyId: 'KEY105',
      deviceId: 'CAB001',
      reservationId: 'RSV001',
    })

    await assert.rejects(
      async () => {
        await operationService.startOperation({
          action: DeviceOperationAction.PICKUP,
          userId: 'U001',
          keyId: 'KEY105',
          deviceId: 'CAB001',
          reservationId: 'RSV001',
        })
      },
      err => err.message === OperationErrorCode.OPERATION_DUPLICATED,
    )
  })

  console.log('\n======================================================')
  console.log(`测试完成: 共 ${passedCount + failedCount} 项用例, \x1b[32m${passedCount} 通过\x1b[0m, \x1b[${failedCount > 0 ? '31' : '32'}m${failedCount} 失败\x1b[0m`)
  console.log('======================================================\n')

  if (failedCount > 0) {
    process.exit(1)
  }
}

main().catch(err => {
  console.error('Fatal test error:', err)
  process.exit(1)
})
