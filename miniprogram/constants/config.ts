export const USE_MOCK = true

export const MQTT_TOPIC_PREFIX = 'keybox'

export type MqttChannel = 'command' | 'event' | 'status' | 'heartbeat'

export function deviceTopic(deviceId: string, channel: MqttChannel): string {
  return `${MQTT_TOPIC_PREFIX}/${deviceId}/${channel}`
}
