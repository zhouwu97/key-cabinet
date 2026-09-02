import {
  reservationService,
  borrowService,
  userService,
  keyService,
} from '../../services'
import { Reservation, ReservationStatus } from '../../models/reservation'
import { BorrowRecord, BorrowRecordStatus } from '../../models/borrow-record'
import { Key } from '../../models/key'

type TabType = 'RESERVATIONS' | 'CURRENT' | 'HISTORY' | 'EXCEPTION'

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
        r =>
          r.status === BorrowRecordStatus.OVERDUE ||
          r.status === BorrowRecordStatus.EXCEPTION,
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

  async cancelReservation(e: any) {
    const id = e.currentTarget.dataset.id
    try {
      await wx.showModal({
        title: '取消预约',
        content: '确认取消该预约吗？',
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
    const labels: Record<ReservationStatus, string> = {
      [ReservationStatus.ACTIVE]: '待使用',
      [ReservationStatus.USED]: '已使用',
      [ReservationStatus.CANCELLED]: '已取消',
      [ReservationStatus.EXPIRED]: '已过期',
    }
    return labels[status] || '未知'
  },

  getBorrowStatusLabel(status: BorrowRecordStatus): string {
    const labels: Record<BorrowRecordStatus, string> = {
      [BorrowRecordStatus.BORROWING]: '借用中',
      [BorrowRecordStatus.BORROWED]: '借用中',
      [BorrowRecordStatus.OVERDUE]: '逾期',
      [BorrowRecordStatus.RETURNING]: '归还中',
      [BorrowRecordStatus.COMPLETED]: '已完成',
      [BorrowRecordStatus.EXCEPTION]: '异常',
    }
    return labels[status] || '未知'
  },

  getStatusTone(
    status: ReservationStatus | BorrowRecordStatus,
  ): string {
    if (status === ReservationStatus.ACTIVE) return 'blue'
    if (status === BorrowRecordStatus.BORROWED) return 'orange'
    if (status === BorrowRecordStatus.OVERDUE) return 'red'
    if (status === BorrowRecordStatus.COMPLETED) return 'green'
    if (status === BorrowRecordStatus.EXCEPTION) return 'red'
    return 'gray'
  },
})
