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

function formatTime(timestamp?: number): string {
  if (!timestamp) return '--:--'
  const date = new Date(timestamp)
  const h = date.getHours().toString().padStart(2, '0')
  const m = date.getMinutes().toString().padStart(2, '0')
  return `${h}:${m}`
}

export interface ReservationViewModel extends Reservation {
  keyName: string
  roomNo: string
  pickupWindowStartText: string
  pickupWindowEndText: string
  statusLabel: string
  statusTone: string
  canPickup: boolean
  canCancel: boolean
}

export interface BorrowViewModel extends BorrowRecord {
  keyName: string
  roomNo: string
  isOverdue: boolean
  expectedReturnText: string
  borrowedAtText: string
  statusLabel: string
  statusTone: string
  canReturn: boolean
}

Page({
  data: {
    deviceName: '1号钥匙柜 (信息楼)',
    deviceLabel: '在线正常',
    tone: 'green',
    availableSlotCount: 7,
    totalSlotCount: 10,
    activeOperation: null as DeviceOperation | null,
    activeOperationKey: null as Key | null,
    activeReservations: [] as ReservationViewModel[],
    currentBorrows: [] as BorrowViewModel[],
    loading: true,
  },

  onLoad() {
    // Removed: loadData() will be called in onShow()
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
        const activeReservations: ReservationViewModel[] = reservations
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
              pickupWindowStartText: formatTime(r.pickupWindowStart),
              pickupWindowEndText: formatTime(r.pickupWindowEnd),
              statusLabel: r.status === ReservationStatus.APPROVED ? '已审批通过' : '待现场取钥',
              statusTone: 'blue',
              canPickup: true,
              canCancel: true,
            }
          })

        // 处理借用记录
        const currentBorrows: BorrowViewModel[] = borrows.map(b => {
          const key = keyMap.get(b.keyId)
          const overdue = isRecordOverdue(b)
          return {
            ...b,
            keyName: key?.name || b.keyId,
            roomNo: key?.roomNo || '',
            isOverdue: overdue,
            expectedReturnText: formatTime(b.expectedReturnAt),
            borrowedAtText: formatTime(b.borrowedAt),
            statusLabel: overdue ? '已逾期' : '借用中',
            statusTone: overdue ? 'red' : 'green',
            canReturn: true,
          }
        })

        const availableSlotCount = allKeys.filter(k => k.status === 'AVAILABLE').length

        this.setData({
          deviceName: device.name || '1号钥匙柜 (信息楼)',
          deviceLabel: DEVICE_STATUS_LABEL[device.status],
          tone,
          availableSlotCount: availableSlotCount || 7,
          totalSlotCount: allKeys.length || 10,
          activeOperation: activeOp,
          activeOperationKey: activeOpKey,
          activeReservations,
          currentBorrows,
          loading: false,
        })
      } else {
        this.setData({
          deviceName: device.name || '1号钥匙柜 (信息楼)',
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

  // 处理预约卡片事件
  async onReservationCardPickup(e: any) {
    const { id: rsvId, keyId } = e.detail
    try {
      const user = await userService.getCurrentUser()
      if (!user) return

      const key = await keyService.getKeyById(keyId)
      if (!key) return

      wx.showLoading({ title: '正在建立会话...' })
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

  async onReservationCardCancel(e: any) {
    const { id: rsvId } = e.detail
    wx.showModal({
      title: '取消预约',
      content: '确定要取消该笔钥匙预约吗？',
      success: async res => {
        if (res.confirm) {
          try {
            await reservationService.cancelReservation(rsvId)
            wx.showToast({ title: '已取消预约', icon: 'success' })
            this.loadData()
          } catch (err: any) {
            wx.showToast({ title: err.message || '取消失败', icon: 'none' })
          }
        }
      },
    })
  },

  // 处理借用卡片事件
  async onBorrowCardReturn(e: any) {
    const { id: borrowId, keyId } = e.detail
    try {
      const user = await userService.getCurrentUser()
      if (!user) return

      const key = await keyService.getKeyById(keyId)
      if (!key) return

      wx.showLoading({ title: '正在开启归还口...' })
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

  goKeys() {
    wx.switchTab({ url: '/pages/keys/keys' })
  },

  goMyReservations() {
    // 存储标记，供 records 页面进入时定位到 RESERVATIONS Tab
    try {
      wx.setStorageSync('kcab_records_initial_tab', 'RESERVATIONS')
    } catch (e) {}
    wx.switchTab({ url: '/pages/records/records' })
  },

  goRecords() {
    wx.switchTab({ url: '/pages/records/records' })
  },

  goHelp() {
    wx.switchTab({ url: '/pages/profile/profile' })
  },
})
