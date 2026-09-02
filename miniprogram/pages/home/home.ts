import {
  deviceService,
  reservationService,
  borrowService,
  userService,
  operationService,
  keyService,
} from '../../services'
import { DEVICE_STATUS_LABEL } from '../../constants/labels'
import { DeviceStatus } from '../../models/device'
import { Reservation, ReservationStatus } from '../../models/reservation'
import { BorrowRecord, isRecordOverdue } from '../../models/borrow-record'
import { DeviceOperation, DeviceOperationAction } from '../../models/device-operation'
import { Key } from '../../models/key'

Page({
  data: {
    deviceName: '一号钥匙柜',
    deviceLabel: '检测中',
    tone: 'gray',
    activeOperation: null as DeviceOperation | null,
    activeOperationKey: null as Key | null,
    activeReservations: [] as Array<Reservation & { keyName?: string; roomNo?: string }>,
    currentBorrows: [] as Array<BorrowRecord & { keyName?: string; roomNo?: string; isOverdue?: boolean }>,
    loading: true,
  },

  onLoad() {
    this.loadData()
  },

  onShow() {
    this.loadData()
  },

  async loadData() {
    try {
      this.setData({ loading: true })

      // 1. 检查是否存在未完成的活动操作 (P0)
      const activeOp = await operationService.resumeActiveOperation()
      let activeOpKey: Key | null = null
      if (activeOp) {
        activeOpKey = await keyService.getKeyById(activeOp.keyId)
      }

      // 2. 加载设备状态 (P4)
      const device = await deviceService.getDeviceStatus('CAB001')
      const tone =
        device.status === DeviceStatus.ONLINE
          ? 'green'
          : device.status === DeviceStatus.FAULT
            ? 'red'
            : 'gray'

      // 3. 加载用户预约与借用数据 (P1, P2)
      const user = await userService.getCurrentUser()
      if (user) {
        const [reservations, borrows, allKeys] = await Promise.all([
          reservationService.getUserReservations(user.id),
          borrowService.getCurrentBorrows(user.id),
          keyService.getKeys(),
        ])

        const keyMap = new Map<string, Key>()
        allKeys.forEach(k => keyMap.set(k.id, k))

        // 筛选可取钥的预约 (ACTIVE / APPROVED)
        const activeReservations = reservations
          .filter(
            r =>
              r.status === ReservationStatus.ACTIVE ||
              r.status === ReservationStatus.APPROVED,
          )
          .map(r => {
            const key = keyMap.get(r.keyId)
            return {
              ...r,
              keyName: key?.name || r.keyId,
              roomNo: key?.roomNo || '',
            }
          })

        // 处理借用记录
        const currentBorrows = borrows.map(b => {
          const key = keyMap.get(b.keyId)
          return {
            ...b,
            keyName: key?.name || b.keyId,
            roomNo: key?.roomNo || '',
            isOverdue: isRecordOverdue(b),
          }
        })

        this.setData({
          deviceName: device.name,
          deviceLabel: DEVICE_STATUS_LABEL[device.status],
          tone,
          activeOperation: activeOp,
          activeOperationKey: activeOpKey,
          activeReservations,
          currentBorrows,
          loading: false,
        })
      } else {
        this.setData({
          deviceName: device.name,
          deviceLabel: DEVICE_STATUS_LABEL[device.status],
          tone,
          activeOperation: activeOp,
          activeOperationKey: activeOpKey,
          loading: false,
        })
      }
    } catch (e) {
      console.error('加载首页数据失败', e)
      this.setData({ loading: false })
    }
  },

  // 点击未完成操作横幅 -> 恢复操作
  resumeOperation() {
    if (this.data.activeOperation) {
      wx.navigateTo({
        url: `/pages/operation/operation?operationId=${this.data.activeOperation.id}`,
      })
    }
  },

  // 点击预约卡片上的“发起取钥”
  async startPickupFromReservation(e: any) {
    const rsvId = e.currentTarget.dataset.id
    const keyId = e.currentTarget.dataset.keyid
    try {
      const user = await userService.getCurrentUser()
      if (!user) return

      const key = await keyService.getKeyById(keyId)
      if (!key) return

      wx.showLoading({ title: '准备就绪，正在连接...' })
      const op = await operationService.startOperation({
        action: DeviceOperationAction.PICKUP,
        userId: user.id,
        keyId,
        deviceId: key.deviceId,
        slotId: key.slotId,
        reservationId: rsvId,
      })
      wx.hideLoading()

      wx.navigateTo({
        url: `/pages/operation/operation?operationId=${op.id}`,
      })
    } catch (e: any) {
      wx.hideLoading()
      wx.showToast({ title: e.message || '发起取钥失败', icon: 'none' })
    }
  },

  // 点击借用卡片上的“发起归还”
  async startReturnFromBorrow(e: any) {
    const borrowId = e.currentTarget.dataset.id
    const keyId = e.currentTarget.dataset.keyid
    try {
      const user = await userService.getCurrentUser()
      if (!user) return

      const key = await keyService.getKeyById(keyId)
      if (!key) return

      wx.showLoading({ title: '正在开启归还通道...' })
      const op = await operationService.startOperation({
        action: DeviceOperationAction.RETURN,
        userId: user.id,
        keyId,
        deviceId: key.deviceId,
        slotId: key.slotId,
        borrowRecordId: borrowId,
      })
      wx.hideLoading()

      wx.navigateTo({
        url: `/pages/operation/operation?operationId=${op.id}`,
      })
    } catch (e: any) {
      wx.hideLoading()
      wx.showToast({ title: e.message || '发起归还失败', icon: 'none' })
    }
  },

  goScan() {
    wx.navigateTo({ url: '/pages/scan/scan' })
  },

  goKeys() {
    wx.switchTab({ url: '/pages/keys/keys' })
  },

  goRecords() {
    wx.switchTab({ url: '/pages/records/records' })
  },

  formatTime(timestamp: number): string {
    const date = new Date(timestamp)
    const h = date.getHours().toString().padStart(2, '0')
    const m = date.getMinutes().toString().padStart(2, '0')
    return `${h}:${m}`
  },
})
