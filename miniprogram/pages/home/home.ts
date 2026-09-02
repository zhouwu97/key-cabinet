import { deviceService } from '../../services'
import {
  reservationService,
  borrowService,
  userService,
} from '../../services'
import { DEVICE_STATUS_LABEL } from '../../constants/labels'
import { DeviceStatus } from '../../models/device'
import { Reservation, ReservationStatus } from '../../models/reservation'
import { BorrowRecord } from '../../models/borrow-record'

Page({
  data: {
    deviceName: '一号钥匙柜',
    deviceLabel: '检测中',
    tone: 'gray',
    activeReservations: [] as Reservation[],
    currentBorrows: [] as BorrowRecord[],
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

      // 加载设备状态
      const device = await deviceService.getDeviceStatus('CAB001')
      const tone =
        device.status === DeviceStatus.ONLINE
          ? 'green'
          : device.status === DeviceStatus.FAULT
            ? 'red'
            : 'gray'

      // 加载用户数据
      const user = await userService.getCurrentUser()
      if (user) {
        const [reservations, borrows] = await Promise.all([
          reservationService.getUserReservations(user.id),
          borrowService.getCurrentBorrows(user.id),
        ])

        const activeReservations = reservations.filter(
          r => r.status === ReservationStatus.ACTIVE,
        )

        this.setData({
          deviceName: device.name,
          deviceLabel: DEVICE_STATUS_LABEL[device.status],
          tone,
          activeReservations,
          currentBorrows: borrows,
          loading: false,
        })
      } else {
        this.setData({
          deviceName: device.name,
          deviceLabel: DEVICE_STATUS_LABEL[device.status],
          tone,
          loading: false,
        })
      }
    } catch (e) {
      console.error('加载首页数据失败', e)
      this.setData({ loading: false })
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

  goReservationDetail(e: any) {
    const id = e.currentTarget.dataset.id
    // TODO: 预约详情页
    console.log('查看预约详情', id)
  },

  goBorrowDetail(e: any) {
    const id = e.currentTarget.dataset.id
    // TODO: 借用详情页
    console.log('查看借用详情', id)
  },

  formatTime(timestamp: number): string {
    const date = new Date(timestamp)
    const h = date.getHours().toString().padStart(2, '0')
    const m = date.getMinutes().toString().padStart(2, '0')
    return `${h}:${m}`
  },

  isOverdue(expectedReturnAt: number): boolean {
    return Date.now() > expectedReturnAt
  },
})
