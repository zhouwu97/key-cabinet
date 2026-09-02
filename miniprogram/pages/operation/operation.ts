import { operationService, keyService, deviceService } from '../../services'
import { DeviceOperation, DeviceOperationAction, DeviceOperationStatus } from '../../models/device-operation'
import { Key } from '../../models/key'
import { Device, DeviceEvent, DeviceEventMessage } from '../../models/device'

interface StepItem {
  key: string
  title: string
  desc: string
  status: 'wait' | 'process' | 'finish' | 'error'
}

Page({
  data: {
    operationId: '',
    operation: null as DeviceOperation | null,
    key: null as Key | null,
    device: null as Device | null,
    deviceName: '1号钥匙柜 (信息楼一楼大厅)',
    currentStepNumber: 1,
    steps: [] as StepItem[],
    userPromptTitle: '正在建立设备会话',
    userPromptDesc: '系统正在核验您的预约权限并连接钥匙柜，请稍候',
    loading: true,
    isFinished: false,
    hasError: false,
    errorMessage: '',
  },

  onLoad(options: any) {
    const opId = options.operationId
    this.initOperation(opId)
  },

  onUnload() {
    if (this.data.operationId) {
      operationService.unsubscribeOperation(this.data.operationId, this.onDeviceEvent)
    }
  },

  async initOperation(opId?: string) {
    try {
      this.setData({ loading: true })

      let operation: DeviceOperation | null = null
      if (opId) {
        operation = await operationService.getOperation(opId)
      }

      if (!operation) {
        operation = await operationService.resumeActiveOperation()
      }

      if (!operation) {
        this.setData({
          loading: false,
          hasError: true,
          errorMessage: '未找到待执行的操作会话',
        })
        return
      }

      const [key, device] = await Promise.all([
        keyService.getKeyById(operation.keyId).catch(() => null),
        deviceService.getDeviceStatus(operation.deviceId).catch(() => null),
      ])

      const isFinished = operation.status === DeviceOperationStatus.SUCCESS
      const hasError = operation.status === DeviceOperationStatus.FAILED
      const deviceName = device?.name || (operation.deviceId === 'CAB001' ? '1号钥匙柜 (信息楼一楼大厅)' : operation.deviceId)

      this.setData({
        operationId: operation.id,
        operation,
        key,
        device,
        deviceName,
        loading: false,
        isFinished,
        hasError,
        errorMessage: operation.errorMessage || '',
      })

      this.initSteps(operation.action)

      // 订阅事件流
      operationService.subscribeOperation(operation.id, this.onDeviceEvent)

      if (isFinished) {
        this.markAllStepsFinished()
      } else if (hasError) {
        this.markStepError(operation.errorMessage || '操作执行中断')
      }
    } catch (e: any) {
      console.error('初始化操作页面失败', e)
      this.setData({
        loading: false,
        hasError: true,
        errorMessage: e.message || '初始化操作失败',
      })
    }
  },

  initSteps(action: DeviceOperationAction) {
    const isPickup = action === DeviceOperationAction.PICKUP
    const steps: StepItem[] = [
      {
        key: 'AUTH',
        title: '身份与权限核验',
        desc: '系统已确认预约单与用户借用权限',
        status: 'process',
      },
      {
        key: 'DEVICE_READY',
        title: '设备响应就绪',
        desc: '钥匙柜已就绪，建立安全通信会话',
        status: 'wait',
      },
      {
        key: 'POSITION',
        title: '正在定位钥匙',
        desc: isPickup ? '机械转盘旋转移动至取钥窗口' : '机械转盘旋转对准归还槽位',
        status: 'wait',
      },
      {
        key: 'DOOR',
        title: '柜门开启',
        desc: '电磁锁开启安全防护门，等待操作',
        status: 'wait',
      },
      {
        key: 'KEY_ACTION',
        title: isPickup ? '等待取走钥匙' : 'RFID 芯片扫描核验',
        desc: isPickup ? '请从槽位取走钥匙并关闭柜门' : '请将钥匙插入归还口，扫描 RFID 芯片',
        status: 'wait',
      },
      {
        key: 'DONE',
        title: '操作完成与结算',
        desc: isPickup ? '钥匙已取出，借用台账正式生效' : 'RFID 核对一致，归还成功并结算',
        status: 'wait',
      },
    ]

    this.setData({
      steps,
      currentStepNumber: 1,
      userPromptTitle: '正在进行身份与权限核验',
      userPromptDesc: '系统正在核验预约权限并建立设备安全通信...',
    })
  },

  onDeviceEvent(msg: DeviceEventMessage) {
    this.handleEventProgress(msg)
  },

  handleEventProgress(msg: DeviceEventMessage) {
    const isPickup = this.data.operation?.action === DeviceOperationAction.PICKUP
    const keyName = this.data.key ? `${this.data.key.roomNo} ${this.data.key.name}` : '钥匙'
    const steps = [...this.data.steps]

    switch (msg.event) {
      case DeviceEvent.RECEIVED:
        steps[0].status = 'finish'
        steps[1].status = 'process'
        this.setData({
          steps,
          currentStepNumber: 2,
          userPromptTitle: '钥匙柜响应就绪',
          userPromptDesc: '已与 1 号钥匙柜建立连接，即将启动机械寻位',
        })
        break

      case DeviceEvent.AUTH_CONFIRMED:
        steps[0].status = 'finish'
        steps[1].status = 'finish'
        steps[2].status = 'process'
        this.setData({
          steps,
          currentStepNumber: 3,
          userPromptTitle: '正在定位钥匙',
          userPromptDesc: `机械转盘正在旋转将 ${keyName} 对准取还口，请勿触碰柜体`,
        })
        break

      case DeviceEvent.POSITIONING:
        steps[2].status = 'process'
        this.setData({
          steps,
          currentStepNumber: 3,
          userPromptTitle: '正在定位钥匙',
          userPromptDesc: '机械转盘运动寻位中，请稍候...',
        })
        break

      case DeviceEvent.POSITIONED:
        steps[2].status = 'finish'
        steps[3].status = 'process'
        this.setData({
          steps,
          currentStepNumber: 4,
          userPromptTitle: '槽位对准就绪',
          userPromptDesc: '槽位已对准到位，正在开启安全柜门...',
        })
        break

      case DeviceEvent.DOOR_OPEN:
        steps[3].status = 'finish'
        steps[4].status = 'process'
        this.setData({
          steps,
          currentStepNumber: 5,
          userPromptTitle: '柜门已开启',
          userPromptDesc: isPickup
            ? `柜门已打开，请取走 ${keyName} 并在 60 秒内关闭柜门`
            : '柜门已打开，请将钥匙插入归还口槽位',
        })
        break

      case DeviceEvent.WAITING_REMOVE:
        this.setData({
          userPromptTitle: '等待取走钥匙',
          userPromptDesc: `请取走槽位中的 ${keyName} 并随手关闭安全门`,
        })
        break

      case DeviceEvent.KEY_REMOVED:
        steps[4].status = 'finish'
        steps[5].status = 'process'
        this.setData({
          steps,
          currentStepNumber: 5,
          userPromptTitle: '钥匙已取走',
          userPromptDesc: '传感器已检测到钥匙离柜，正在关闭安全门并复位...',
        })
        break

      case DeviceEvent.KEY_RETURNED:
        this.setData({
          userPromptTitle: '正在扫描 RFID 芯片',
          userPromptDesc: '检测到钥匙已放入归还口，正在扫描芯片 UID 校验...',
        })
        break

      case DeviceEvent.RFID_CONFIRMED:
        steps[4].status = 'finish'
        steps[5].status = 'process'
        this.setData({
          steps,
          currentStepNumber: 5,
          userPromptTitle: 'RFID 芯片验证通过',
          userPromptDesc: '钥匙身份核对正确，正在关闭柜门完成归还结算...',
        })
        break

      case DeviceEvent.DOOR_CLOSED:
      case DeviceEvent.HOMING:
        this.setData({
          userPromptTitle: '设备复位中',
          userPromptDesc: '柜门已安全锁止，机构正在归零复位...',
        })
        break

      case DeviceEvent.SUCCESS:
        this.markAllStepsFinished()
        this.setData({
          isFinished: true,
          currentStepNumber: 6,
          userPromptTitle: isPickup ? '取钥成功！借用已生效' : '归还成功！台账已结算',
          userPromptDesc: isPickup
            ? '请妥善保管钥匙并按时归还。'
            : '钥匙已安全入库，感谢您的配合。',
        })
        wx.showToast({ title: isPickup ? '取钥成功' : '归还成功', icon: 'success' })
        break

      case DeviceEvent.FAILED:
        this.markStepError(msg.errorMessage || '操作失败')
        this.setData({
          hasError: true,
          errorMessage: msg.errorMessage || '设备操作异常终止',
          userPromptTitle: '操作未能成功完成',
          userPromptDesc: msg.errorMessage || '设备执行遇到异常，请检查并重试',
        })
        break
    }
  },

  markAllStepsFinished() {
    const steps = this.data.steps.map(s => ({ ...s, status: 'finish' as const }))
    this.setData({
      steps,
      currentStepNumber: 6,
      isFinished: true,
      userPromptTitle: '操作已成功完成',
      userPromptDesc: '钥匙借还事务处理完毕。',
    })
  },

  markStepError(errorMsg: string) {
    const steps = [...this.data.steps]
    const idx = Math.min(this.data.currentStepNumber - 1, steps.length - 1)
    if (steps[idx]) {
      steps[idx].status = 'error'
      steps[idx].desc = errorMsg
    }
    this.setData({ steps, hasError: true })
  },

  goHome() {
    wx.switchTab({ url: '/pages/home/home' })
  },

  goRecords() {
    wx.switchTab({ url: '/pages/records/records' })
  },
})
