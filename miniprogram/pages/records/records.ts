import {
  reservationService,
  borrowService,
  userService,
  keyService,
  operationService,
} from '../../services'
import { Reservation, ReservationStatus } from '../../models/reservation'
import { BorrowRecord, BorrowRecordStatus, isRecordOverdue } from '../../models/borrow-record'
import { Key } from '../../models/key'
import {
  RESERVATION_STATUS_LABEL,
  RESERVATION_STATUS_TONE,
  getBorrowRecordDisplayStatus,
} from '../../constants/labels'
import { DeviceOperationAction } from '../../models/device-operation'

type TabType = 'CURRENT' | 'RESERVATIONS' | 'HISTORY' | 'EXCEPTION'

Page({
  data: {
    activeTab: 'CURRENT' as TabType,
    reservations: [] as Reservation[],
    currentBorrows: [] as BorrowRecord[],
    historyBorrows: [] as BorrowRecord[],
    exceptionBorrows: [] as BorrowRecord[],
    keyMap: {} as Record<string, Key>,
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

      const user = await userService.getCurrentUser()
      if (!user) {
        this.setData({ loading: false })
        return
      }

      const [reservations, borrowRecords, keys] = await Promise.all([
        reservationService.getUserReservations(user.id),
        borrowService.getUserBorrowRecords(user.id),
        keyService.getKeys(),
      ])

      // 构建钥匙映射
      const keyMap: Record<string, Key> = {}
      keys.forEach(key => {
        keyMap[key.id] = key
      })

      // 分类借还记录
      const currentBorrows = borrowRecords.filter(
        r =>
          r.status === BorrowRecordStatus.BORROWED ||
          r.status === BorrowRecordStatus.BORROWING ||
          r.status === BorrowRecordStatus.RETURNING,
      )

      const historyBorrows = borrowRecords.filter(
        r => r.status === BorrowRecordStatus.COMPLETED,
      )

      const exceptionBorrows = borrowRecords.filter(
        r => isRecordOverdue(r) || r.status === BorrowRecordStatus.EXCEPTION,
      )

      this.setData({
        reservations,
        currentBorrows,
        historyBorrows,
        exceptionBorrows,
        keyMap,
        loading: false,
      })
    } catch (e) {
      console.error('加载记录数据失败', e)
      this.setData({ loading: false })
      wx.showToast({ title: '加载失败', icon: 'none' })
    }
  },

  onTabChange(e: any) {
    const tab = e.currentTarget.dataset.tab as TabType
    this.setData({ activeTab: tab })
  },

  // 快捷发起取钥
  async startPickup(e: any) {
    const rsvId = e.currentTarget.dataset.id
    const keyId = e.currentTarget.dataset.keyid
    try {
      const user = await userService.getCurrentUser()
      if (!user) return

      const key = this.data.keyMap[keyId]
      if (!key) return

      const op = await operationService.startOperation({
        action: DeviceOperationAction.PICKUP,
        userId: user.id,
        keyId,
        deviceId: key.deviceId,
        slotId: key.slotId,
        reservationId: rsvId,
      })

      wx.navigateTo({
        url: `/pages/operation/operation?operationId=${op.id}`,
      })
    } catch (e: any) {
      wx.showToast({ title: e.message || '发起取钥失败', icon: 'none' })
    }
  },

  // 快捷发起归还
  async startReturn(e: any) {
    const borrowId = e.currentTarget.dataset.id
    const keyId = e.currentTarget.dataset.keyid
    try {
      const user = await userService.getCurrentUser()
      if (!user) return

      const key = this.data.keyMap[keyId]
      if (!key) return

      const op = await operationService.startOperation({
        action: DeviceOperationAction.RETURN,
        userId: user.id,
        keyId,
        deviceId: key.deviceId,
        slotId: key.slotId,
        borrowRecordId: borrowId,
      })

      wx.navigateTo({
        url: `/pages/operation/operation?operationId=${op.id}`,
      })
    } catch (e: any) {
      wx.showToast({ title: e.message || '发起归还失败', icon: 'none' })
    }
  },

  async cancelReservation(e: any) {
    const id = e.currentTarget.dataset.id
    try {
      await wx.showModal({
        title: '取消预约',
        content: '确认取消该预约吗？取消后该时段钥匙将重新对他人开放。',
      })

      await reservationService.cancelReservation(id)
      wx.showToast({ title: '已取消', icon: 'success' })
      this.loadData()
    } catch (e: any) {
      if (e.errMsg && e.errMsg.includes('cancel')) {
        return
      }
      console.error('取消预约失败', e)
      wx.showToast({ title: '取消失败', icon: 'none' })
    }
  },

  formatDate(timestamp: number): string {
    const date = new Date(timestamp)
    const m = (date.getMonth() + 1).toString().padStart(2, '0')
    const d = date.getDate().toString().padStart(2, '0')
    return `${m}-${d}`
  },

  formatTime(timestamp: number): string {
    const date = new Date(timestamp)
    const h = date.getHours().toString().padStart(2, '0')
    const min = date.getMinutes().toString().padStart(2, '0')
    return `${h}:${min}`
  },

  formatDateTime(timestamp: number): string {
    return `${this.formatDate(timestamp)} ${this.formatTime(timestamp)}`
  },

  getKeyName(keyId: string): string {
    const key = this.data.keyMap[keyId]
    return key ? `${key.roomNo} ${key.name}` : keyId
  },

  getReservationStatusLabel(status: ReservationStatus): string {
    return RESERVATION_STATUS_LABEL[status] || '未知'
  },

  getReservationTone(status: ReservationStatus): string {
    return RESERVATION_STATUS_TONE[status] || 'gray'
  },

  getBorrowStatusInfo(record: BorrowRecord) {
    return getBorrowRecordDisplayStatus(record)
  },
})
