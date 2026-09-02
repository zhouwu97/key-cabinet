import { operationService, keyService, deviceService, userService } from '../../services'
import { DeviceOperation, DeviceOperationAction, DeviceOperationStatus } from '../../models/device-operation'
import { Key } from '../../models/key'
import { Device, DeviceEvent, DeviceEventMessage } from '../../models/device'
import { DEVICE_EVENT_LABEL } from '../../constants/labels'
import { MockScenario, MOCK_SCENARIO_LABEL } from '../../mocks/mock-scenarios'

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
    statusText: '正在连接设备...',
    statusTone: 'blue',
    latestEvent: '',
    currentStepIndex: 0,
    steps: [] as StepItem[],
    eventLogs: [] as Array<{ time: string; text: string; seq: number }>,
    loading: true,
    isFinished: false,
    hasError: false,
    errorMessage: '',

    // 调试与异常场景注入
    scenarios: Object.entries(MOCK_SCENARIO_LABEL).map(([key, label]) => ({ key, label })),
    selectedScenario: MockScenario.SUCCESS,
    scenarioIndex: 0,
    showDebugPanel: true,
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
        // 尝试恢复中断中的活跃操作
        operation = await operationService.resumeActiveOperation()
      }

      if (!operation) {
        this.setData({
          loading: false,
          hasError: true,
          errorMessage: '未找到待执行的操作记录',
        })
        return
      }

      const [key, device] = await Promise.all([
        keyService.getKeyById(operation.keyId),
        deviceService.getDeviceStatus(operation.deviceId),
      ])

      this.setData({
        operationId: operation.id,
        operation,
        key,
        device,
        loading: false,
        isFinished: operation.status === DeviceOperationStatus.SUCCESS,
        hasError: operation.status === DeviceOperationStatus.FAILED,
        errorMessage: operation.errorMessage || '',
      })

      this.initSteps(operation.action)

      // 订阅事件
      operationService.subscribeOperation(operation.id, this.onDeviceEvent)

      // 若已经处于成功或失败状态，直接展示最终步骤
      if (operation.status === DeviceOperationStatus.SUCCESS) {
        this.markAllStepsFinished()
      } else if (operation.status === DeviceOperationStatus.FAILED) {
        this.markStepError(operation.errorMessage || '操作执行失败')
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
        title: '身份认证',
        desc: '核验用户身份与预约权限',
        status: 'process',
      },
      {
        key: 'POSITION',
        title: '钥匙定位',
        desc: isPickup ? '转盘移动至取钥槽位' : '转盘移动至归还槽位',
        status: 'wait',
      },
      {
        key: 'DOOR',
        title: '安全门开',
        desc: '电磁锁开启安全防护门',
        status: 'wait',
      },
      {
        key: 'KEY_ACTION',
        title: isPickup ? '取走钥匙' : '放入并核验',
        desc: isPickup ? '请从槽位取出钥匙' : 'RFID 读取并核对钥匙标签',
        status: 'wait',
      },
      {
        key: 'HOMING',
        title: '关门归位',
        desc: '安全门关闭，设备复位',
        status: 'wait',
      },
      {
        key: 'DONE',
        title: '操作完成',
        desc: isPickup ? '已记录借出，请妥善保管' : '已成功归还入库',
        status: 'wait',
      },
    ]

    this.setData({ steps, currentStepIndex: 0, statusText: '正在核验身份与下发指令...' })
  },

  onDeviceEvent(msg: DeviceEventMessage) {
    const timeStr = new Date(msg.timestamp).toLocaleTimeString()
    const logText = msg.errorMessage
      ? `${DEVICE_EVENT_LABEL[msg.event] || msg.event} (${msg.errorMessage})`
      : DEVICE_EVENT_LABEL[msg.event] || msg.event

    const newLogs = [...this.data.eventLogs, { time: timeStr, text: logText, seq: msg.seq }]

    this.setData({
      eventLogs: newLogs,
      latestEvent: logText,
    })

    this.handleEventProgress(msg)
  },

  handleEventProgress(msg: DeviceEventMessage) {
    const keyName = this.data.key ? `${this.data.key.roomNo} ${this.data.key.name}` : '钥匙'
    const isPickup = this.data.operation?.action === DeviceOperationAction.PICKUP
    const steps = [...this.data.steps]

    switch (msg.event) {
      case DeviceEvent.RECEIVED:
        this.setData({ statusText: '设备已接收操作指令，准备就绪' })
        break

      case DeviceEvent.AUTH_CONFIRMED:
        steps[0].status = 'finish'
        steps[1].status = 'process'
        this.setData({
          steps,
          currentStepIndex: 1,
          statusText: `身份验证成功，正在将 ${keyName} 定位至窗口...`,
        })
        break

      case DeviceEvent.POSITIONING:
        steps[1].status = 'process'
        this.setData({
          steps,
          statusText: `转盘电机正在旋转定位 ${keyName}...`,
        })
        break

      case DeviceEvent.POSITIONED:
        steps[1].status = 'finish'
        steps[2].status = 'process'
        this.setData({
          steps,
          currentStepIndex: 2,
          statusText: '槽位对准到位，正在开启安全柜门...',
        })
        break

      case DeviceEvent.DOOR_OPEN:
        steps[2].status = 'finish'
        steps[3].status = 'process'
        this.setData({
          steps,
          currentStepIndex: 3,
          statusText: isPickup ? `柜门已打开，请取走 ${keyName}` : '柜门已打开，请将钥匙放入归还口',
        })
        break

      case DeviceEvent.WAITING_REMOVE:
        this.setData({ statusText: `等待取走 ${keyName}...` })
        break

      case DeviceEvent.KEY_REMOVED:
        steps[3].status = 'finish'
        steps[4].status = 'process'
        this.setData({
          steps,
          currentStepIndex: 4,
          statusText: '检测到钥匙已被取走，正在关闭柜门...',
        })
        break

      case DeviceEvent.KEY_RETURNED:
        this.setData({ statusText: '检测到钥匙已放入，正在进行 RFID 芯片认证...' })
        break

      case DeviceEvent.RFID_CONFIRMED:
        steps[3].status = 'finish'
        steps[4].status = 'process'
        this.setData({
          steps,
          currentStepIndex: 4,
          statusText: 'RFID 身份校验通过，钥匙匹配正确！',
        })
        break

      case DeviceEvent.DOOR_CLOSED:
        this.setData({ statusText: '柜门已安全关闭锁止，机构归零中...' })
        break

      case DeviceEvent.HOMING:
        steps[4].status = 'process'
        this.setData({ steps, statusText: '转盘正在执行复位归零...' })
        break

      case DeviceEvent.SUCCESS:
        this.markAllStepsFinished()
        this.setData({
          isFinished: true,
          statusText: isPickup ? '取钥成功！借用计时已开始' : '归还成功！借还单已完成',
          statusTone: 'green',
        })
        wx.showToast({ title: isPickup ? '取钥成功' : '归还成功', icon: 'success' })
        break

      case DeviceEvent.FAILED:
        this.markStepError(msg.errorMessage || '操作失败')
        this.setData({
          hasError: true,
          errorMessage: msg.errorMessage || '设备操作异常终止',
          statusText: `操作失败: ${msg.errorMessage || '设备异常'}`,
          statusTone: 'red',
        })
        break
    }
  },

  markAllStepsFinished() {
    const steps = this.data.steps.map(s => ({ ...s, status: 'finish' as const }))
    this.setData({ steps, currentStepIndex: steps.length - 1, isFinished: true })
  },

  markStepError(errorMsg: string) {
    const steps = [...this.data.steps]
    const idx = Math.min(this.data.currentStepIndex, steps.length - 1)
    if (steps[idx]) {
      steps[idx].status = 'error'
      steps[idx].desc = errorMsg
    }
    this.setData({ steps, hasError: true })
  },

  onScenarioChange(e: any) {
    const idx = parseInt(e.detail.value)
    const item = this.data.scenarios[idx]
    if (item) {
      this.setData({
        scenarioIndex: idx,
        selectedScenario: item.key as MockScenario,
      })
      deviceService.setGlobalScenario(item.key as MockScenario)
      wx.showToast({ title: `已设置场景：${item.label}`, icon: 'none' })
    }
  },

  // 调试辅助：从当前页面一键发起测试取钥/归还
  async triggerDemoPickup() {
    try {
      const user = await userService.getCurrentUser()
      if (!user) return

      const op = await operationService.startOperation(
        {
          action: DeviceOperationAction.PICKUP,
          userId: user.id,
          keyId: 'KEY103',
          deviceId: 'CAB001',
          slotId: 'SLOT03',
        },
        this.data.selectedScenario,
      )

      this.initOperation(op.id)
    } catch (e: any) {
      wx.showToast({ title: e.message || '发起失败', icon: 'none' })
    }
  },

  async triggerDemoReturn() {
    try {
      const user = await userService.getCurrentUser()
      if (!user) return

      const op = await operationService.startOperation(
        {
          action: DeviceOperationAction.RETURN,
          userId: user.id,
          keyId: 'KEY104',
          deviceId: 'CAB001',
          slotId: 'SLOT04',
        },
        this.data.selectedScenario,
      )

      this.initOperation(op.id)
    } catch (e: any) {
      wx.showToast({ title: e.message || '发起失败', icon: 'none' })
    }
  },

  goHome() {
    wx.switchTab({ url: '/pages/home/home' })
  },

  goRecords() {
    wx.switchTab({ url: '/pages/records/records' })
  },
})
