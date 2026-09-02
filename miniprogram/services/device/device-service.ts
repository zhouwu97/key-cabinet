import { Device, DeviceCommandMessage, DeviceEventMessage } from '../../models/device'
import { MockScenario } from '../../mocks/mock-scenarios'

export type DeviceEventListener = (message: DeviceEventMessage) => void

export interface DeviceService {
  /** 获取设备状态 */
  getDeviceStatus(deviceId: string): Promise<Device>

  /** 检查指定设备是否正在执行操作中 */
  isDeviceBusy(deviceId: string): boolean

  /** 执行设备指令（支持指定/全局注入的 MockScenario） */
  executeCommand(command: DeviceCommandMessage, scenario?: MockScenario): Promise<void>

  /** 订阅设备实时事件 */
  subscribeDevice(deviceId: string, listener: DeviceEventListener): void

  /** 取消订阅设备事件 */
  unsubscribeDevice(deviceId: string, listener: DeviceEventListener): void

  /** 设置全局 Mock 故障注入场景 */
  setGlobalScenario(scenario: MockScenario): void

  /** 获取当前注入的 Mock 场景 */
  getGlobalScenario(): MockScenario
}
