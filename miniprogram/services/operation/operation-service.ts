import { DeviceOperation, StartOperationInput } from '../../models/device-operation'
import { DeviceEventMessage } from '../../models/device'
import { MockScenario } from '../../mocks/mock-scenarios'

export type OperationEventListener = (message: DeviceEventMessage) => void

export interface OperationService {
  /** 发起一次取钥或归还设备操作 */
  startOperation(input: StartOperationInput, scenario?: MockScenario): Promise<DeviceOperation>

  /** 根据操作ID获取操作实体 */
  getOperation(operationId: string): Promise<DeviceOperation | null>

  /** 获取当前正在执行中的活动操作 */
  getActiveOperation(userId?: string): Promise<DeviceOperation | null>

  /** 恢复中断的活动操作上下文 */
  resumeActiveOperation(): Promise<DeviceOperation | null>

  /** 取消或终止未完成的操作 */
  cancelOperation(operationId: string): Promise<void>

  /** 订阅指定操作的实时事件流 */
  subscribeOperation(operationId: string, listener: OperationEventListener): void

  /** 取消订阅操作事件流 */
  unsubscribeOperation(operationId: string, listener: OperationEventListener): void
}
