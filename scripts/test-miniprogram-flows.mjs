import test, { beforeEach } from 'node:test'
import assert from 'node:assert/strict'
import { createRequire } from 'node:module'

const require = createRequire(import.meta.url)
const storage = new Map()
globalThis.wx = {
  getStorageSync: key => structuredClone(storage.get(key)),
  setStorageSync: (key, value) => storage.set(key, structuredClone(value)),
  removeStorageSync: key => storage.delete(key),
  showToast() {}, showLoading() {}, hideLoading() {},
  showModal: async () => ({ confirm: true }),
  switchTab() {}, navigateTo() {}, redirectTo() {}, stopPullDownRefresh() {},
}

const services = require('../.test-dist/services/index.js')
const { httpClient } = require('../.test-dist/api/http-client.js')
const { ApiOperationService } = require('../.test-dist/services/operation/api-operation-service.js')
const { MockReservationService } = require('../.test-dist/services/reservation/mock-reservation-service.js')
const { MockOperationService } = require('../.test-dist/services/operation/mock-operation-service.js')
const { formatDate, formatTime, parseLocalDateTime } = require('../.test-dist/utils/date.js')
const { canPickupReservation } = require('../.test-dist/models/reservation.js')
const { STORAGE_KEYS, MOCK_KEYS, MOCK_DEVICES, MOCK_BORROW_RECORDS, MOCK_RESERVATIONS } = require('../.test-dist/mocks/mock-data.js')
const flush = () => new Promise(resolve => setImmediate(resolve))
const user = { id: 'U001', name: '测试用户', identityVerified: true }
const key = { ...MOCK_KEYS[0] }
const reservation = (status = 'ACTIVE') => ({
  id: 'R1', userId: user.id, keyId: key.id, status, createdAt: Date.now(),
  pickupWindowStart: Date.now() - 1000, pickupWindowEnd: Date.now() + 1800000,
  expectedReturnAt: Date.now() + 7200000, purpose: '实验准备',
})
const borrow = (status = 'BORROWED') => ({ ...MOCK_BORROW_RECORDS[0], status })
const operation = (status = 'EXECUTING') => ({
  id: 'OP1', requestId: 'REQ1', action: 'PICKUP', userId: user.id, keyId: key.id,
  deviceId: key.deviceId, slotId: key.slotId, createdAt: Date.now(), status,
})

function loadPage(name) {
  let definition
  globalThis.Page = page => { definition = page }
  const path = require.resolve(`../.test-dist/pages/${name}/${name}.js`)
  delete require.cache[path]
  require(path)
  const page = { ...definition, data: structuredClone(definition.data) }
  page.setData = values => Object.assign(page.data, values)
  for (const [name, value] of Object.entries(definition)) {
    if (typeof value === 'function') page[name] = value.bind(page)
  }
  return page
}

function stubContext(t, reservations = [reservation()], borrows = []) {
  t.mock.method(services.userService, 'getCurrentUser', async () => user)
  t.mock.method(services.keyService, 'getKeys', async () => [key])
  t.mock.method(services.keyService, 'getKeyById', async () => key)
  t.mock.method(services.reservationService, 'getUserReservations', async () => structuredClone(reservations))
  t.mock.method(services.borrowService, 'getUserBorrowRecords', async () => structuredClone(borrows))
  t.mock.method(services.borrowService, 'getCurrentBorrows', async () => structuredClone(borrows))
  t.mock.method(services.operationService, 'getActiveOperation', async () => null)
  t.mock.method(services.deviceService, 'getDeviceStatus', async () => ({ ...MOCK_DEVICES[0], status: 'ONLINE' }))
  t.mock.method(services.deviceService, 'listDevices', async () => [MOCK_DEVICES[0]])
}

beforeEach(() => {
  storage.clear()
  wx.setStorageSync(STORAGE_KEYS.KEYS, MOCK_KEYS)
  wx.setStorageSync(STORAGE_KEYS.DEVICES, MOCK_DEVICES)
  wx.setStorageSync(STORAGE_KEYS.RESERVATIONS, [])
  wx.setStorageSync(STORAGE_KEYS.BORROW_RECORDS, [])
})

test('预约时间：明确跨日，拒绝非法日历日期和非法时刻', () => {
  const value = parseLocalDateTime('2026-09-11', '01:30')
  assert.equal(formatDate(value), '2026-09-11')
  assert.equal(formatTime(value), '01:30')
  assert.ok(Number.isNaN(parseLocalDateTime('2026-02-30', '09:00')))
  assert.ok(Number.isNaN(parseLocalDateTime('2026-09-11', '24:00')))
  const r = reservation()
  assert.equal(canPickupReservation(r, r.pickupWindowStart - 1), false)
  assert.equal(canPickupReservation(r, r.pickupWindowStart), true)
  assert.equal(canPickupReservation(r, r.pickupWindowEnd + 1), false)
  assert.equal(canPickupReservation({ ...r, status: 'PENDING' }), false)
})

test('预约表单：午夜默认日期正确，过早归还不提交，成功后防止重复预约', async t => {
  stubContext(t)
  const now = new Date(2026, 8, 10, 23, 30).getTime()
  t.mock.method(Date, 'now', () => now)
  const page = loadPage('reservation-create')
  let loading
  page.loadKeyInfo = () => { loading = Promise.resolve() }
  page.onLoad({ keyId: key.id })
  await loading
  assert.equal(page.data.returnDate, '2026-09-11')
  assert.equal(page.data.returnTime, '02:30')
  page.setData({ key, loading: false, returnDate: '2026-09-11', returnTime: '00:00' })
  const create = t.mock.method(services.reservationService, 'createReservation', async () => reservation('PENDING'))
  await page.submitReservation()
  assert.equal(create.mock.callCount(), 0)
  page.onReturnTimeChange({ detail: { value: '00:31' } })
  await page.submitReservation()
  assert.equal(create.mock.calls[0].arguments[0].expectedReturnAt, parseLocalDateTime('2026-09-11', '00:31'))
  assert.equal(page.data.createdReservation.status, 'PENDING')
  await page.submitReservation()
  assert.equal(create.mock.callCount(), 1)
  const navigate = t.mock.method(wx, 'switchTab')
  page.goRecords()
  assert.equal(navigate.mock.calls[0].arguments[0].url, '/pages/records/records')
})

test('记录页：历史预约与拒绝原因完整，已批准可取消，设备动作中不可归还', async t => {
  const statuses = ['ACTIVE', 'APPROVED', 'PENDING', 'USED', 'CANCELLED', 'REJECTED', 'EXPIRED']
  const reservations = statuses.map((status, i) => ({ ...reservation(status), id: `R${i}`, rejectionReason: status === 'REJECTED' ? '用途不符' : '' }))
  const borrows = ['BORROWED', 'BORROWING', 'RETURNING', 'EXCEPTION', 'COMPLETED'].map((status, i) => ({ ...borrow(status), id: `B${i}`, expectedReturnAt: Date.now() - 1000 }))
  stubContext(t, reservations, borrows)
  const page = loadPage('records')
  await page.loadData()
  assert.equal(page.data.activeReservations.length, 3)
  assert.equal(page.data.historyReservations.length, 4)
  assert.equal(page.data.historyTotalCount, 5)
  assert.equal(page.data.historyReservations.find(r => r.status === 'REJECTED').rejectionReason, '用途不符')
  assert.equal(page.data.activeReservations.find(r => r.status === 'APPROVED').canCancel, true)
  assert.equal(page.data.activeReservations.find(r => r.status === 'PENDING').canPickup, false)
  assert.deepEqual(page.data.currentBorrows.map(b => b.canReturn), [true, false, false])
  assert.equal(page.data.exceptionBorrows.find(b => b.status === 'EXCEPTION').canReturn, false)
  assert.equal(page.data.currentBorrows[0].returnedAtText, '')
})

test('记录页：取消已批准预约后进入历史，退出登录后清空旧用户记录', async t => {
  const reservations = [reservation('APPROVED')]
  stubContext(t, reservations)
  t.mock.method(services.reservationService, 'cancelReservation', async () => { reservations[0].status = 'CANCELLED' })
  const page = loadPage('records')
  await page.loadData()
  await page.onReservationCancel({ detail: { id: 'R1' } })
  assert.equal(page.data.currentTotalCount, 0)
  assert.equal(page.data.historyReservations[0].status, 'CANCELLED')
  t.mock.method(services.userService, 'getCurrentUser', async () => null)
  await page.loadData()
  assert.equal(page.data.loggedIn, false)
  assert.equal(page.data.historyTotalCount, 0)
  assert.deepEqual(page.data.keyMap, {})
})

test('记录页：网络错误可重试，下拉刷新结束，不让旧请求覆盖新状态', async t => {
  stubContext(t)
  const page = loadPage('records')
  t.mock.method(console, 'error', () => {})
  const request = t.mock.method(services.reservationService, 'getUserReservations', async () => { throw new Error('network') })
  const stop = t.mock.method(wx, 'stopPullDownRefresh')
  await page.onPullDownRefresh()
  assert.ok(page.data.loadError)
  assert.equal(stop.mock.callCount(), 1)
  request.mock.mockImplementation(async () => [])
  await page.loadData()
  assert.equal(page.data.loadError, '')
  let release
  request.mock.mockImplementationOnce(() => new Promise(resolve => { release = resolve }))
  const oldRequest = page.loadData()
  await flush()
  await page.loadData()
  release([reservation()])
  await oldRequest
  assert.equal(page.data.activeReservations.length, 0)
})

test('扫码核验：二维码类型、预约归属和时间窗口均需有效', async t => {
  const reservations = [reservation()]
  stubContext(t, reservations)
  const page = loadPage('scan')
  await page.onLoad({ keyId: key.id, reservationId: 'R1', expectedDeviceId: key.deviceId })
  for (const raw of ['null', '{"cabinetId":123}', '[]']) {
    await page.handleScanRawResult(raw)
    assert.equal(page.data.verified, false)
  }
  reservations[0].pickupWindowEnd = Date.now() - 1
  assert.equal(await page.verifyCabinet(key.deviceId), false)
  reservations[0] = { ...reservation(), keyId: 'OTHER' }
  assert.equal(await page.verifyCabinet(key.deviceId), false)
  reservations[0] = reservation()
  assert.equal(await page.verifyCabinet(key.deviceId), true)
})

test('扫码确认：未扫码不能发指令，扫码后预约被取消必须重新核验', async t => {
  const reservations = [reservation()]
  stubContext(t, reservations)
  const start = t.mock.method(services.operationService, 'startOperation', async () => operation())
  const page = loadPage('scan')
  await page.onLoad({ keyId: key.id, reservationId: 'R1', expectedDeviceId: key.deviceId })
  await page.onConfirmOperation()
  assert.equal(start.mock.callCount(), 0)
  await page.verifyCabinet(key.deviceId)
  reservations[0].status = 'CANCELLED'
  await page.onConfirmOperation()
  assert.equal(start.mock.callCount(), 0)
  assert.equal(page.data.verified, false)
  assert.equal(page.data.starting, false)
})

test('归还：沿用借出柜机槽位，连续确认只发起一次操作', async t => {
  const record = { ...borrow(), keyId: key.id, deviceId: key.deviceId, slotId: 'ORIGINAL_SLOT' }
  stubContext(t, [], [record])
  const start = t.mock.method(services.operationService, 'startOperation', async () => operation())
  const page = loadPage('scan')
  await page.onLoad({ mode: 'RETURN', keyId: key.id, borrowRecordId: record.id, expectedDeviceId: record.deviceId })
  assert.equal(await page.verifyCabinet(record.deviceId), true)
  await Promise.all([page.onConfirmOperation(), page.onConfirmOperation()])
  assert.equal(start.mock.callCount(), 1)
  assert.equal(start.mock.calls[0].arguments[0].slotId, 'ORIGINAL_SLOT')
})

test('操作恢复：设备忙碌时仍显示已有操作的恢复入口', async t => {
  stubContext(t)
  t.mock.method(services.operationService, 'getActiveOperation', async () => operation())
  t.mock.method(services.deviceService, 'getDeviceStatus', async () => ({ status: 'BUSY' }))
  const page = loadPage('scan')
  assert.equal(await page.verifyCabinet(key.deviceId), false)
  assert.equal(page.data.activeOperationId, 'OP1')
  const navigate = t.mock.method(wx, 'redirectTo')
  page.resumeOperation()
  assert.match(navigate.mock.calls[0].arguments[0].url, /operationId=OP1/)
})

test('首页：待审批预约可见，归还中不能重复归还，扫码优先恢复已有操作', async t => {
  stubContext(t, [reservation('PENDING')], [borrow('RETURNING')])
  const page = loadPage('home')
  await page.loadData()
  assert.equal(page.data.activeReservations[0].canPickup, false)
  assert.equal(page.data.normalBorrows[0].canReturn, false)
  assert.equal(page.data.normalBorrows[0].statusLabel, '入柜中')
  page.setData({ activeOperation: operation() })
  const navigate = t.mock.method(wx, 'navigateTo')
  page.onMainScanTap()
  assert.match(navigate.mock.calls[0].arguments[0].url, /operationId=OP1/)
})

test('Mock 预约：审批、冲突、过期释放和逾期占用与真实流程一致', async () => {
  wx.setStorageSync(STORAGE_KEYS.KEYS, [{ ...key, requiresApproval: true }])
  const service = new MockReservationService()
  assert.deepEqual(await service.getUserReservations(user.id), [])
  const params = { ...reservation(), expectedDuration: 7200000 }
  const created = await service.createReservation(params)
  assert.equal(created.status, 'PENDING')
  assert.equal(created.approvedAt, undefined)
  await assert.rejects(service.createReservation(params), /RESERVATION_CONFLICT/)
  wx.setStorageSync(STORAGE_KEYS.RESERVATIONS, [{ ...created, pickupWindowEnd: Date.now() - 1 }])
  assert.equal((await service.getUserReservations(user.id))[0].status, 'EXPIRED')
  assert.equal(wx.getStorageSync(STORAGE_KEYS.KEYS)[0].status, 'AVAILABLE')
  wx.setStorageSync(STORAGE_KEYS.BORROW_RECORDS, [{ ...borrow(), keyId: key.id, expectedReturnAt: Date.now() - 1 }])
  await assert.rejects(service.createReservation(params), /KEY_ALREADY_BORROWED/)
})

test('Mock 取钥：沿用预约用途和归还时间，拒绝未批准的显式预约编号', async () => {
  const r = { ...MOCK_RESERVATIONS[0], expectedReturnAt: Date.now() + 86400000, purpose: '通宵设备调试' }
  wx.setStorageSync(STORAGE_KEYS.RESERVATIONS, [r])
  const reservationService = new MockReservationService()
  let createdBorrowArgs
  const service = new MockOperationService(
    { getDeviceStatus: async () => ({ status: 'ONLINE' }), isDeviceBusy: () => false, subscribeDevice() {}, executeCommand() {} },
    { getKeyById: async () => MOCK_KEYS.find(k => k.id === r.keyId), getKeySlot: async () => ({ id: 'SLOT05' }) },
    reservationService,
    { createBorrowRecord: async (...args) => { createdBorrowArgs = args; return { id: 'B1' } } },
  )
  const input = { action: 'PICKUP', userId: r.userId, keyId: r.keyId, deviceId: 'CAB001', reservationId: r.id }
  wx.setStorageSync(STORAGE_KEYS.RESERVATIONS, [{ ...r, status: 'PENDING' }])
  await assert.rejects(service.startOperation(input), /RESERVATION_NOT_ACTIVE/)
  wx.setStorageSync(STORAGE_KEYS.RESERVATIONS, [r])
  await service.startOperation(input)
  assert.equal(createdBorrowArgs[5], r.purpose)
  assert.equal(createdBorrowArgs[6], r.expectedReturnAt)
})

function fakePolling(t) {
  const timers = new Map()
  let id = 0
  t.mock.method(globalThis, 'setInterval', callback => { timers.set(++id, callback); return id })
  t.mock.method(globalThis, 'clearInterval', timer => timers.delete(timer))
  return timers
}

test('API 取消被拒绝后继续轮询，即使事件缺少 SUCCESS 也以服务端终态收敛', async t => {
  const timers = fakePolling(t)
  let snapshot = operation()
  t.mock.method(httpClient, 'request', async ({ method }) => {
    if (method === 'POST') throw new Error('无法安全取消')
    return snapshot
  })
  const service = new ApiOperationService()
  const events = []
  service.subscribeOperation('OP1', message => events.push(message.event))
  await flush()
  await assert.rejects(service.cancelOperation('OP1'), /无法安全取消/)
  assert.equal(timers.size, 1)
  snapshot = { ...operation('SUCCESS'), events: [{ event: 'HOMING', seq: 2 }, { event: 'DOOR_CLOSED', seq: 1 }] }
  for (const callback of timers.values()) callback()
  await flush()
  assert.deepEqual(events.slice(-3), ['DOOR_CLOSED', 'HOMING', 'SUCCESS'])
  assert.equal(timers.size, 0)
})

test('API 慢请求不会并发轮询，离开页面后的响应不再更新界面', async t => {
  const timers = fakePolling(t)
  let release
  const request = t.mock.method(httpClient, 'request', () => new Promise(resolve => { release = resolve }))
  const service = new ApiOperationService()
  const events = []
  const listener = event => events.push(event)
  service.subscribeOperation('OP1', listener)
  for (const callback of timers.values()) { callback(); callback() }
  assert.equal(request.mock.callCount(), 1)
  service.unsubscribeOperation('OP1', listener)
  release(operation('SUCCESS'))
  await flush()
  assert.deepEqual(events, [])
  assert.equal(timers.size, 0)
})

test('进度页：恢复中间步骤，提示手动关门，迟到事件不覆盖成功', () => {
  const page = loadPage('operation')
  page.setData({ operationId: 'OP1', operation: operation(), key })
  page.initSteps('PICKUP')
  page.handleEventProgress({ operationId: 'OP1', event: 'KEY_REMOVED' })
  assert.equal(page.data.steps.slice(0, 5).every(step => step.status === 'finish'), true)
  assert.match(page.data.userPromptDesc, /请关闭安全门/)
  page.handleEventProgress({ operationId: 'OP1', event: 'SUCCESS' })
  page.handleEventProgress({ operationId: 'OP1', event: 'DOOR_OPEN' })
  assert.equal(page.data.isFinished, true)
  assert.equal(page.data.currentStepNumber, 6)
})
