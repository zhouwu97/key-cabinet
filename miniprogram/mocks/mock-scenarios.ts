/**
 * Mock 设备执行场景枚举
 * 用于在开发、调试、测试与演示时注入预期行为与异常
 */
export enum MockScenario {
  /** 正常成功执行 */
  SUCCESS = 'SUCCESS',
  /** 设备离线 */
  DEVICE_OFFLINE = 'DEVICE_OFFLINE',
  /** 设备正忙（已被其他任务占用） */
  DEVICE_BUSY = 'DEVICE_BUSY',
  /** 定位失败（电机卡死或传感器丢失） */
  POSITION_ERROR = 'POSITION_ERROR',
  /** 柜门打开失败（门锁故障） */
  DOOR_ERROR = 'DOOR_ERROR',
  /** RFID 未检测到钥匙 */
  RFID_NOT_FOUND = 'RFID_NOT_FOUND',
  /** RFID 检测到错误钥匙（归还错误钥匙） */
  RFID_WRONG_KEY = 'RFID_WRONG_KEY',
  /** 操作整体超时未响应 */
  TIMEOUT = 'TIMEOUT',
}

export const MOCK_SCENARIO_LABEL: Record<MockScenario, string> = {
  [MockScenario.SUCCESS]: '正常流程（成功）',
  [MockScenario.DEVICE_OFFLINE]: '异常：设备离线',
  [MockScenario.DEVICE_BUSY]: '异常：设备繁忙',
  [MockScenario.POSITION_ERROR]: '异常：电机定位失败',
  [MockScenario.DOOR_ERROR]: '异常：柜门开启失败',
  [MockScenario.RFID_NOT_FOUND]: '异常：RFID 未感应到',
  [MockScenario.RFID_WRONG_KEY]: '异常：归还错误钥匙',
  [MockScenario.TIMEOUT]: '异常：操作超时',
}
