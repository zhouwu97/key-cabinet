import { userService, borrowService, reservationService } from '../../services/index'
import { User } from '../../models/user'
import { BorrowRecordStatus } from '../../models/borrow-record'
import { ReservationStatus } from '../../models/reservation'

Page({
  data: {
    user: null as User | null,
    currentBorrowCount: 0,
    activeReservationCount: 0,
    totalBorrowCount: 0,
    loading: true,
  },

  onLoad() {
    this.loadUserData()
  },

  onShow() {
    this.loadUserData()
  },

  async loadUserData() {
    try {
      this.setData({ loading: true })
      const user = await userService.getCurrentUser()
      if (!user) {
        this.setData({ loading: false })
        return
      }

      const [borrows, reservations] = await Promise.all([
        borrowService.getUserBorrowRecords(user.id).catch(() => []),
        reservationService.getUserReservations(user.id).catch(() => []),
      ])

      const currentBorrowCount = borrows.filter(
        b =>
          b.status === BorrowRecordStatus.BORROWED ||
          b.status === BorrowRecordStatus.BORROWING ||
          b.status === BorrowRecordStatus.RETURNING,
      ).length

      const activeReservationCount = reservations.filter(
        r =>
          r.status === ReservationStatus.ACTIVE ||
          r.status === ReservationStatus.APPROVED,
      ).length

      const totalBorrowCount = borrows.length

      this.setData({
        user,
        currentBorrowCount,
        activeReservationCount,
        totalBorrowCount,
        loading: false,
      })
    } catch (e) {
      console.error('加载用户数据失败', e)
      this.setData({ loading: false })
    }
  },

  goCurrentBorrows() {
    try {
      wx.setStorageSync('kcab_records_initial_tab', 'CURRENT')
    } catch (e) {}
    wx.switchTab({ url: '/pages/records/records' })
  },

  goReservations() {
    try {
      wx.setStorageSync('kcab_records_initial_tab', 'RESERVATIONS')
    } catch (e) {}
    wx.switchTab({ url: '/pages/records/records' })
  },

  goHistory() {
    try {
      wx.setStorageSync('kcab_records_initial_tab', 'HISTORY')
    } catch (e) {}
    wx.switchTab({ url: '/pages/records/records' })
  },

  showHelpModal() {
    wx.showModal({
      title: '借还流程说明',
      content:
        '1. 找钥匙选择目标房间提交预约。\n2. 在取钥窗口期内前往钥匙柜点击取钥。\n3. 使用完毕后将钥匙插入归还口，通过 RFID 芯片自动识别完成结算。',
      showCancel: false,
      confirmText: '我知道了',
    })
  },

  showLocationsModal() {
    wx.showModal({
      title: '钥匙柜分布位置',
      content:
        '● 1号钥匙柜 (CAB001)：信息楼 1F 门厅东侧 (在线服务中)\n● 2号钥匙柜 (CAB002)：工程实训楼 1F 入口 (建设中)',
      showCancel: false,
      confirmText: '我知道了',
    })
  },

  showRulesModal() {
    wx.showModal({
      title: '实验室钥匙借用守则',
      content:
        '1. 钥匙仅限本人使用，严禁私自转借。\n2. 请于预约应还时间内归还，逾期将扣减信用分。\n3. 如遇钥匙丢失或损坏，请立即前往管理员控制台报障。',
      showCancel: false,
      confirmText: '遵守守则',
    })
  },

  showAbout() {
    wx.showModal({
      title: '智能钥匙自助借还系统',
      content: '版本：v0.3.0-product-ready\n架构：WeChat MiniProgram + Cloud Backend + MQTT ESP32 Cabinet\n状态：Mock 闭环 & 协议冻结完毕',
      showCancel: false,
      confirmText: '确定',
    })
  },

  goAdmin() {
    wx.navigateTo({ url: '/pages/admin/admin' })
  },
})
