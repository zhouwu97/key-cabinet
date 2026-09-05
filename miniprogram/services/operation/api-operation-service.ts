import { httpClient } from '../../api/http-client'
import { toTimestamp } from '../../api/serializers'
import {
  DeviceOperation,
  DeviceOperationAction,
  DeviceOperationStatus,
  StartOperationInput,
} from '../../models/device-operation'
import { DeviceEvent, DeviceEventMessage } from '../../models/device'
import { MockScenario } from '../../mocks/mock-scenarios'
import { generateRequestId } from '../../utils/request-id'
import { OperationEventListener, OperationService } from './operation-service'

type ApiOperation = DeviceOperation & {
  createdAt?: string | number
  startedAt?: string | number
  finishedAt?: string | number
  events?: Array<{
    event?: DeviceEvent | string
    type?: DeviceEvent | string
    seq?: number
    timestamp?: string | number
    eventId?: string
    errorCode?: string
    errorMessage?: string
  }>
}

function normalizeOperation(data: ApiOperation): DeviceOperation {
  return {
    ...data,
    createdAt: toTimestamp(data.createdAt),
    startedAt: data.startedAt === undefined ? undefined : toTimestamp(data.startedAt),
    finishedAt: data.finishedAt === undefined ? undefined : toTimestamp(data.finishedAt),
  }
}

function isTerminal(status: DeviceOperationStatus): boolean {
  return [
    DeviceOperationStatus.SUCCESS,
    DeviceOperationStatus.FAILED,
    DeviceOperationStatus.TIMEOUT,
    DeviceOperationStatus.CANCELLED,
  ].includes(status)
}

function eventForStatus(status: DeviceOperationStatus): DeviceEvent {
  switch (status) {
    case DeviceOperationStatus.SUCCESS:
      return DeviceEvent.SUCCESS
    case DeviceOperationStatus.FAILED:
    case DeviceOperationStatus.TIMEOUT:
    case DeviceOperationStatus.CANCELLED:
      return DeviceEvent.FAILED
    default:
      return DeviceEvent.RECEIVED
  }
}

function normalizeEvent(value: unknown, status: DeviceOperationStatus): DeviceEvent {
  return typeof value === 'string' && Object.values(DeviceEvent).includes(value as DeviceEvent)
    ? (value as DeviceEvent)
    : eventForStatus(status)
}

/** 真实操作服务使用后台事务和 HTTP Polling，不在客户端重复执行借还状态机。 */
export class ApiOperationService implements OperationService {
  private readonly listeners = new Map<string, Set<OperationEventListener>>()
  private readonly pollTimers = new Map<string, ReturnType<typeof setInterval>>()
  private readonly lastStatus = new Map<string, DeviceOperationStatus>()
  private readonly lastEventSeq = new Map<string, number>()

  async startOperation(input: StartOperationInput, _scenario?: MockScenario): Promise<DeviceOperation> {
    const action = input.action === DeviceOperationAction.PICKUP ? 'pickup' : 'return'
    const clientRequestId = input.requestId || generateRequestId()
    const body =
      input.action === DeviceOperationAction.PICKUP
        ? {
            reservationId: input.reservationId,
            clientRequestId,
          }
        : {
            borrowRecordId: input.borrowRecordId,
            deviceId: input.deviceId,
            clientRequestId,
          }
    const operation = await httpClient.request<ApiOperation>({
      url: `/api/v1/device-operations/${action}`,
      method: 'POST',
      data: body,
    })
    return normalizeOperation(operation)
  }

  async getOperation(operationId: string): Promise<DeviceOperation | null> {
    return this.fetchOperation(operationId, true)
  }

  private async fetchOperation(operationId: string, logError: boolean): Promise<ApiOperation | null> {
    try {
      const operation = await httpClient.request<ApiOperation>({
        url: `/api/v1/device-operations/${encodeURIComponent(operationId)}`,
      })
      return operation
    } catch (error) {
      if (logError) {
        console.error(`Failed to get operation ${operationId}:`, error)
      }
      return null
    }
  }

  async getActiveOperation(_userId?: string): Promise<DeviceOperation | null> {
    try {
      const operation = await httpClient.request<ApiOperation>({
        url: '/api/v1/device-operations/active',
      })
      return operation ? normalizeOperation(operation) : null
    } catch (error) {
      console.error('Failed to get active operation:', error)
      return null
    }
  }

  async resumeActiveOperation(): Promise<DeviceOperation | null> {
    return this.getActiveOperation()
  }

  async cancelOperation(operationId: string): Promise<void> {
    this.stopPolling(operationId)
    await httpClient.request<unknown>({
      url: `/api/v1/device-operations/${encodeURIComponent(operationId)}/cancel`,
      method: 'POST',
    })
  }

  subscribeOperation(operationId: string, listener: OperationEventListener): void {
    if (!this.listeners.has(operationId)) {
      this.listeners.set(operationId, new Set())
    }
    this.listeners.get(operationId)!.add(listener)

    if (!this.pollTimers.has(operationId)) {
      const timer = setInterval(() => {
        void this.pollOperation(operationId)
      }, 1500)
      this.pollTimers.set(operationId, timer)
      void this.pollOperation(operationId)
    }
  }

  unsubscribeOperation(operationId: string, listener: OperationEventListener): void {
    this.listeners.get(operationId)?.delete(listener)
    if (!this.listeners.get(operationId)?.size) {
      this.stopPolling(operationId)
    }
  }

  private async pollOperation(operationId: string): Promise<void> {
    const snapshot = await this.fetchOperation(operationId, false)
    if (!snapshot) return
    const operation = normalizeOperation(snapshot)

    const events = snapshot.events || []
    const previousEventSeq = this.lastEventSeq.get(operationId) || 0
    const newEvents = events.filter(event => typeof event.seq === 'number' && event.seq > previousEventSeq)
    if (newEvents.length > 0) {
      this.lastEventSeq.set(
        operationId,
        Math.max(...newEvents.map(event => event.seq as number)),
      )
      newEvents.forEach(event => {
        const message: DeviceEventMessage = {
          version: '1.0',
          operationId: operation.id,
          requestId: operation.requestId,
          eventId: event.eventId || `poll_${operation.id}_${event.seq}`,
          seq: event.seq || 0,
          deviceId: operation.deviceId,
          event: normalizeEvent(event.event || event.type, operation.status),
          errorCode: event.errorCode || operation.errorCode,
          errorMessage: event.errorMessage || operation.errorMessage,
          timestamp: toTimestamp(event.timestamp),
        }
        this.listeners.get(operationId)?.forEach(listener => listener(message))
      })
    } else if (this.lastStatus.get(operationId) !== operation.status) {
      // 后端暂未返回事件数组时，至少通过状态变化驱动终态和恢复提示。
      const message: DeviceEventMessage = {
        version: '1.0',
        operationId: operation.id,
        requestId: operation.requestId,
        eventId: `poll_${operation.id}_${operation.status}`,
        seq: 0,
        deviceId: operation.deviceId,
        event: eventForStatus(operation.status),
        errorCode: operation.errorCode,
        errorMessage: operation.errorMessage,
        timestamp: Date.now(),
      }
      this.listeners.get(operationId)?.forEach(listener => listener(message))
    }
    this.lastStatus.set(operationId, operation.status)

    if (isTerminal(operation.status)) {
      this.stopPolling(operationId)
    }
  }

  private stopPolling(operationId: string): void {
    const timer = this.pollTimers.get(operationId)
    if (timer) {
      clearInterval(timer)
      this.pollTimers.delete(operationId)
    }
    this.lastStatus.delete(operationId)
    this.lastEventSeq.delete(operationId)
  }
}
