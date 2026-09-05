import {
  keyService,
  deviceService,
  userService,
  operationService,
  reservationService,
  borrowService,
} from '../../services/index'
import { DeviceOperationAction, DeviceOperationStatus } from '../../models/device-operation'
import { ReservationStatus } from '../../models/reservation'
import { BorrowRecordStatus } from '../../models/borrow-record'
import { Key } from '../../models/key'
import { Device } from '../../models/device'

Page({
  data: {
    mode: 'PICKUP' as 'PICKUP' | 'RETURN',
    reservationId: '',
    borrowRecordId: '',
    keyId: '',
    expectedDeviceId: '',
    key: null as Key | null,
    device: null as Device | null,
    userName: '',
    cabinetName: '1号钥匙柜',
    cabinetLocation: '信息楼 1F 大厅东侧',
    isOnline: true,
    scannedCabinetId: '',
    verified: false,
    verifyError: '',
    starting: false,
  },

  async onLoad(options: any) {
    const mode = (options.mode === 'RETURN' ? 'RETURN' : 'PICKUP') as 'PICKUP' | 'RETURN'
    const reservationId = options.reservationId || ''
    const borrowRecordId = options.borrowRecordId || ''
    const keyId = options.keyId || ''
    const expectedDeviceId = options.expectedDeviceId || 'CAB001'

    this.setData({
      mode,
      reservationId,
      borrowRecordId,
      keyId,
      expectedDeviceId,
    })

    await this.loadContext()
  },

  async loadContext() {
    try {
      const [user, key, device] = await Promise.all([
        userService.getCurrentUser(),
        this.data.keyId ? keyService.getKeyById(this.data.keyId) : Promise.resolve(null),
        deviceService.getDeviceStatus(this.data.expectedDeviceId || 'CAB001'),
      ])

      this.setData({
        userName: user?.name || '',
        key: key || null,
        device: device || null,
        cabinetName: device?.name || '1号钥匙柜',
        cabinetLocation: '信息楼 1F 大厅东侧',
        isOnline: device?.status === 'ONLINE',
      })
    } catch (e) {
      console.error('加载核验上下文失败', e)
    }
  },

  // 启动微信扫码
  startScan() {
    this.setData({ verifyError: '' })
    wx.scanCode({
      onlyFromCamera: true,
      scanType: ['qrCode'],
      success: res => {
        this.handleScanRawResult(res.result)
      },
      fail: err => {
        if (err.errMsg && !err.errMsg.includes('cancel')) {
          this.setData({ verifyError: '调用扫码失败，请重试或开启相机权限' })
        }
      },
    })
  },

  // 开发/测试模拟扫码识别 CAB001
  simulateScanSuccess() {
    this.handleScanRawResult(JSON.stringify({ cabinetId: this.data.expectedDeviceId || 'CAB001' }))
  },

  // 解析扫码内容
  handleScanRawResult(raw: string) {
    let cabinetId = ''
    try {
      const parsed = JSON.parse(raw)
      cabinetId = parsed.cabinetId || parsed.deviceId || ''
    } catch {
      // 纯字符串格式 CAB001
      cabinetId = raw.trim()
    }

    if (!cabinetId) {
      this.setData({
        verifyError: '二维码格式无效，未识别到钥匙柜编号',
        verified: false,
      })
      return
    }

    this.verifyCabinet(cabinetId)
  },

  // 核心核验逻辑
  async verifyCabinet(scannedCabinetId: string) {
    wx.showLoading({ title: '正在核验设备...' })
    this.setData({ verifyError: '' })

    try {
      // 1. 验证设备是否在线
      const device = await deviceService.getDeviceStatus(scannedCabinetId)
      if (!device) {
        throw new Error(`未找到编号为 [${scannedCabinetId}] 的钥匙柜设备`)
      }
      if (device.status !== 'ONLINE') {
        throw new Error(`钥匙柜 [${device.name || scannedCabinetId}] 当前处于离线状态，暂时无法提供自助借还服务`)
      }

      // 2. 验证扫码设备 == 当前钥匙所在柜
      const expected = this.data.expectedDeviceId || this.data.key?.deviceId || 'CAB001'
      if (scannedCabinetId !== expected) {
        throw new Error(`设备不匹配！目标钥匙存放在 [${expected}]，您扫描的是 [${scannedCabinetId}]，请前往指定钥匙柜扫码`)
      }

      // 3. 验证未完成 Operation 中断检查
      const activeOp = await operationService.getActiveOperation()
      const isOpInProgress =
        activeOp &&
        [
          DeviceOperationStatus.CREATED,
          DeviceOperationStatus.AUTHORIZED,
          DeviceOperationStatus.SENT,
          DeviceOperationStatus.EXECUTING,
        ].includes(activeOp.status)

      if (isOpInProgress) {
        throw new Error('检测到您当前有正在进行的钥匙柜操作，请先恢复并完成当前会话')
      }

      // 4. 验证预约/借用状态
      const user = await userService.getCurrentUser()
      if (!user) {
        throw new Error('用户登录态已失效，请重新登录')
      }

      if (this.data.mode === 'PICKUP' && this.data.reservationId) {
        const reservations = await reservationService.getUserReservations(user.id)
        const rsv = reservations.find(r => r.id === this.data.reservationId)
        if (rsv && rsv.status !== ReservationStatus.ACTIVE && rsv.status !== ReservationStatus.APPROVED) {
          throw new Error('当前预约状态不可取钥（已取消、已过期或已完成）')
        }
      } else if (this.data.mode === 'RETURN' && this.data.borrowRecordId) {
        const borrows = await borrowService.getUserBorrowRecords(user.id)
        const record = borrows.find(b => b.id === this.data.borrowRecordId)
        if (record && record.status === BorrowRecordStatus.COMPLETED) {
          throw new Error('该笔借用记录已归还，无需重复操作')
        }
      }

      wx.hideLoading()

      // 核验通过，展示确认卡片
      this.setData({
        scannedCabinetId,
        cabinetName: device.name || '1号钥匙柜',
        cabinetLocation: '信息楼 1F 大厅东侧',
        isOnline: true,
        verified: true,
        verifyError: '',
      })
    } catch (err: any) {
      wx.hideLoading()
      this.setData({
        verified: false,
        verifyError: err.message || '核验失败，请重试',
      })
    }
  },

  // 重新扫码
  reScan() {
    this.setData({
      verified: false,
      verifyError: '',
    })
    this.startScan()
  },

  // 用户确认开始取钥 / 归还
  async onConfirmOperation() {
    if (this.data.starting) return
    this.setData({ starting: true })

    try {
      const user = await userService.getCurrentUser()
      if (!user) {
        wx.showToast({ title: '登录态失效', icon: 'none' })
        this.setData({ starting: false })
        return
      }

      const key = this.data.key
      const keyId = this.data.keyId || key?.id || 'KEY001'
      const deviceId = this.data.scannedCabinetId || this.data.expectedDeviceId || 'CAB001'
      const slotId = key?.slotId || 'SLOT01'

      wx.showLoading({ title: '正在建立柜机连接...' })

      let op
      if (this.data.mode === 'PICKUP') {
        op = await operationService.startOperation({
          action: DeviceOperationAction.PICKUP,
          userId: user.id,
          keyId,
          deviceId,
          slotId,
          reservationId: this.data.reservationId || undefined,
        })
      } else {
        op = await operationService.startOperation({
          action: DeviceOperationAction.RETURN,
          userId: user.id,
          keyId,
          deviceId,
          slotId,
          borrowRecordId: this.data.borrowRecordId || undefined,
        })
      }

      wx.hideLoading()

      // 跳转至设备操作进度页面
      wx.redirectTo({
        url: `/pages/operation/operation?operationId=${op.id}`,
      })
    } catch (err: any) {
      wx.hideLoading()
      this.setData({ starting: false })
      wx.showToast({ title: err.message || '发起操作失败，请重试', icon: 'none' })
    }
  },
})
