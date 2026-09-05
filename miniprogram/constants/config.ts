import { currentConfig } from '../config/index'

/** 兼容旧设备适配器的只读别名，数据模式仍以 currentConfig 为唯一来源。 */
export const USE_MOCK = currentConfig.dataMode === 'mock'

export const MQTT_TOPIC_PREFIX = 'keybox'

export type MqttChannel = 'command' | 'event' | 'status' | 'heartbeat'

export function deviceTopic(deviceId: string, channel: MqttChannel): string {
  return `${MQTT_TOPIC_PREFIX}/${deviceId}/${channel}`
}
