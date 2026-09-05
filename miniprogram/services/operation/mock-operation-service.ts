import {
  DeviceOperation,
  DeviceOperationAction,
  DeviceOperationStatus,
  StartOperationInput,
} from '../../models/device-operation'
import {
  DeviceCommandMessage,
  DeviceEvent,
  DeviceEventMessage,
  DeviceStatus,
} from '../../models/device'
import { OperationErrorCode } from '../../models/operation-error'
import { MockScenario } from '../../mocks/mock-scenarios'
import { OperationEventListener, OperationService } from './operation-service'
import { DeviceService } from '../device/device-service'
import { KeyService } from '../key/key-service'
import { ReservationService } from '../reservation/reservation-service'
import { BorrowService } from '../borrow/borrow-service'
import { STORAGE_KEYS } from '../../mocks/mock-data'
import { KeyStatus } from '../../models/key'
import { KeyPresenceState } from '../../models/key-presence'
import { BorrowRecordStatus } from '../../models/borrow-record'
import { generateRequestId } from '../../utils/request-id'

export class MockOperationService implements OperationService {
  private operations: DeviceOperation[] = []
  private operationListeners = new Map<string, Set<OperationEventListener>>()
  private pendingLocks = new Set<string>()

  constructor(
    private deviceService: DeviceService,
    private keyService: KeyService,
    private reservationService: ReservationService,
    private borrowService: BorrowService,
  ) {
    this.loadFromStorage()
  }

  private loadFromStorage(): void {
    try {
      const stored = wx.getStorageSync(STORAGE_KEYS.OPERATIONS)
      this.operations = Array.isArray(stored) ? stored : []
    } catch (e) {
      console.error('加载操作记录失败', e)
      this.operations = []
    }
  }

  private saveToStorage(): void {
    try {
      wx.setStorageSync(STORAGE_KEYS.OPERATIONS, this.operations)
    } catch (e) {
      console.error('保存操作记录失败', e)
    }
  }

  private generateOperationId(): string {
    return `OP${Date.now()}${Math.random().toString(36).substring(2, 6).toUpperCase()}`
  }

  async startOperation(
    input: StartOperationInput,
    scenario?: MockScenario,
  ): Promise<DeviceOperation> {
    const lockKey = `${input.deviceId}_${input.keyId}`
    if (
      this.pendingLocks.has(lockKey) ||
      this.pendingLocks.has(input.deviceId) ||
      this.pendingLocks.has(input.keyId)
    ) {
      throw new Error(OperationErrorCode.OPERATION_DUPLICATED)
    }

    // 同步加锁防连续点击竞态
    this.pendingLocks.add(lockKey)
    this.pendingLocks.add(input.deviceId)
    this.pendingLocks.add(input.keyId)

    try {
      this.loadFromStorage()
      const now = Date.now()

      // 1. 并发防抖与唯一性校验：同一钥匙或同一设备当前是否有正在执行的 Operation
      const activeOp = this.operations.find(
        op =>
          (op.keyId === input.keyId || op.deviceId === input.deviceId) &&
          (op.status === DeviceOperationStatus.EXECUTING ||
            op.status === DeviceOperationStatus.SENT ||
            op.status === DeviceOperationStatus.AUTHORIZED),
      )
      if (activeOp) {
        throw new Error(OperationErrorCode.OPERATION_DUPLICATED)
      }

      // 2. 检查设备状态
      const device = await this.deviceService.getDeviceStatus(input.deviceId)
      if (device.status === DeviceStatus.OFFLINE) {
        throw new Error(OperationErrorCode.DEVICE_OFFLINE)
      }
      if (this.deviceService.isDeviceBusy(input.deviceId)) {
        throw new Error(OperationErrorCode.DEVICE_BUSY)
      }

      // 3. 检查钥匙与槽位
      const key = await this.keyService.getKeyById(input.keyId)
      if (!key || !key.enabled) {
        throw new Error(OperationErrorCode.KEY_NOT_AVAILABLE)
      }
      const slot = await this.keyService.getKeySlot(input.slotId || key.slotId)
      const slotId = slot ? slot.id : (input.slotId || key.slotId)

      let reservationId = input.reservationId
      let borrowRecordId = input.borrowRecordId

      // 4. 业务前置校验
      if (input.action === DeviceOperationAction.PICKUP) {
        // 取钥流程：必须具备有效的活跃预约
        if (!reservationId) {
          const activeRes = await this.reservationService.getActiveReservation(
            input.userId,
            input.keyId,
          )
          if (!activeRes) {
            throw new Error(OperationErrorCode.RESERVATION_NOT_ACTIVE)
          }
          reservationId = activeRes.id
        }

        // 创建初始借还记录 (BORROWING)
        const borrowRecord = await this.borrowService.createBorrowRecord(
          input.userId,
          input.keyId,
          input.deviceId,
          slotId,
          reservationId,
          '实验科研借用',
        )
        borrowRecordId = borrowRecord.id
      } else if (input.action === DeviceOperationAction.RETURN) {
        // 归还流程：必须存在进行中的借用记录
        if (!borrowRecordId) {
          const activeBorrow = await this.borrowService.getActiveBorrowByKey(input.keyId)
          if (!activeBorrow || activeBorrow.userId !== input.userId) {
            throw new Error(OperationErrorCode.OPERATION_USER_MISMATCH)
          }
          borrowRecordId = activeBorrow.id
        } else {
          const record = await this.borrowService.getBorrowRecordById(borrowRecordId)
          if (!record || record.userId !== input.userId) {
            throw new Error(OperationErrorCode.OPERATION_USER_MISMATCH)
          }
        }
        // 更新借还记录为 RETURNING
        await this.borrowService.updateBorrowStatus(
          borrowRecordId,
          BorrowRecordStatus.RETURNING,
        )
      }

      // 5. 构建可持久化的 DeviceOperation 实体
      const operation: DeviceOperation = {
        id: this.generateOperationId(),
        requestId: input.requestId || generateRequestId(),
        action: input.action,
        userId: input.userId,
        keyId: input.keyId,
        slotId,
        deviceId: input.deviceId,
        reservationId,
        borrowRecordId,
        status: DeviceOperationStatus.EXECUTING,
        createdAt: now,
        startedAt: now,
      }

      this.operations.unshift(operation)
      this.saveToStorage()

      // 记录全局当前活跃 Operation ID
      try {
        wx.setStorageSync(STORAGE_KEYS.ACTIVE_OPERATION_ID, operation.id)
      } catch (e) {
        console.error('保存活跃操作ID失败', e)
      }

      // 6. 订阅底层设备事件并流转业务事实
      const onDeviceEvent = async (msg: DeviceEventMessage) => {
        if (msg.operationId !== operation.id && msg.requestId !== operation.requestId) {
          return
        }

        // 分发给上层操作监听者
        this.emitToOperationListeners(operation.id, msg)

        // 业务状态同步流转
        if (msg.event === DeviceEvent.KEY_REMOVED && operation.action === DeviceOperationAction.PICKUP) {
          // 钥匙已取走 -> 物理在位状态 ABSENT
          const slots = wx.getStorageSync(STORAGE_KEYS.KEY_SLOTS) || []
          const currentSlot = slots.find((s: any) => s.id === operation.slotId)
          if (currentSlot) {
            currentSlot.presence = KeyPresenceState.ABSENT
            currentSlot.lastUpdated = Date.now()
            wx.setStorageSync(STORAGE_KEYS.KEY_SLOTS, slots)
          }
        }

        if (msg.event === DeviceEvent.SUCCESS) {
          // 操作圆满完成
          operation.status = DeviceOperationStatus.SUCCESS
          operation.finishedAt = Date.now()
          this.saveToStorage()
          this.pendingLocks.delete(lockKey)
          this.pendingLocks.delete(operation.deviceId)
          this.pendingLocks.delete(operation.keyId)
          wx.removeStorageSync(STORAGE_KEYS.ACTIVE_OPERATION_ID)
          this.deviceService.unsubscribeDevice(operation.deviceId, onDeviceEvent)

          if (operation.action === DeviceOperationAction.PICKUP) {
            // 取钥成功：核销预约 -> 借用中 -> 钥匙状态借出
            if (operation.reservationId) {
              await this.reservationService.markReservationUsed(operation.reservationId)
            }
            if (operation.borrowRecordId) {
              await this.borrowService.updateBorrowStatus(
                operation.borrowRecordId,
                BorrowRecordStatus.BORROWED,
              )
            }
            await this.keyService.updateKeyStatus(operation.keyId, KeyStatus.BORROWED)
          } else if (operation.action === DeviceOperationAction.RETURN) {
            // 归还成功：完成借还单 -> 钥匙恢复可借 -> 物理在位恢复
            if (operation.borrowRecordId) {
              await this.borrowService.completeBorrowRecord(operation.borrowRecordId)
            }
            await this.keyService.updateKeyStatus(operation.keyId, KeyStatus.AVAILABLE)
          }
        } else if (msg.event === DeviceEvent.FAILED) {
          // 操作失败
          operation.status = DeviceOperationStatus.FAILED
          operation.errorCode = msg.errorCode || OperationErrorCode.SYSTEM_ERROR
          operation.errorMessage = msg.errorMessage || '设备操作未完成'
          operation.finishedAt = Date.now()
          this.saveToStorage()
          this.pendingLocks.delete(lockKey)
          this.pendingLocks.delete(operation.deviceId)
          this.pendingLocks.delete(operation.keyId)
          wx.removeStorageSync(STORAGE_KEYS.ACTIVE_OPERATION_ID)
          this.deviceService.unsubscribeDevice(operation.deviceId, onDeviceEvent)

          // 异常回退保护
          if (operation.action === DeviceOperationAction.PICKUP) {
            if (operation.borrowRecordId) {
              await this.borrowService.updateBorrowStatus(
                operation.borrowRecordId,
                BorrowRecordStatus.EXCEPTION,
                operation.errorMessage,
              )
            }
          } else if (operation.action === DeviceOperationAction.RETURN) {
            if (operation.borrowRecordId) {
              // 归还失败，若为 RFID 错误仍保持借出状态并记录说明
              await this.borrowService.updateBorrowStatus(
                operation.borrowRecordId,
                BorrowRecordStatus.BORROWED,
                `归还异常: ${operation.errorMessage}`,
              )
            }
          }
        }
      }

      this.deviceService.subscribeDevice(operation.deviceId, onDeviceEvent)

      // 7. 发送控制指令给设备
      const command: DeviceCommandMessage = {
        version: '1.0',
        operationId: operation.id,
        requestId: operation.requestId,
        action: operation.action,
        keyId: operation.keyId,
        slotId: operation.slotId,
        deviceId: operation.deviceId,
        timestamp: now,
      }

      this.deviceService.executeCommand(command, scenario)

      return { ...operation }
    } catch (err) {
      this.pendingLocks.delete(lockKey)
      this.pendingLocks.delete(input.deviceId)
      this.pendingLocks.delete(input.keyId)
      throw err
    }
  }

  async getOperation(operationId: string): Promise<DeviceOperation | null> {
    this.loadFromStorage()
    const op = this.operations.find(o => o.id === operationId)
    return op ? { ...op } : null
  }

  async getActiveOperation(userId?: string): Promise<DeviceOperation | null> {
    this.loadFromStorage()
    const activeOp = this.operations.find(
      op =>
        (!userId || op.userId === userId) &&
        (op.status === DeviceOperationStatus.EXECUTING ||
          op.status === DeviceOperationStatus.SENT ||
          op.status === DeviceOperationStatus.AUTHORIZED),
    )
    return activeOp ? { ...activeOp } : null
  }

  async resumeActiveOperation(): Promise<DeviceOperation | null> {
    this.loadFromStorage()
    let activeId: string | null = null
    try {
      activeId = wx.getStorageSync(STORAGE_KEYS.ACTIVE_OPERATION_ID)
    } catch {
      activeId = null
    }

    if (activeId) {
      const op = this.operations.find(o => o.id === activeId)
      if (
        op &&
        (op.status === DeviceOperationStatus.EXECUTING ||
          op.status === DeviceOperationStatus.SENT ||
          op.status === DeviceOperationStatus.AUTHORIZED)
      ) {
        return { ...op }
      }
    }

    return this.getActiveOperation()
  }

  async cancelOperation(operationId: string): Promise<void> {
    this.loadFromStorage()
    const op = this.operations.find(o => o.id === operationId)
    if (op && op.status === DeviceOperationStatus.EXECUTING) {
      op.status = DeviceOperationStatus.CANCELLED
      op.finishedAt = Date.now()
      this.saveToStorage()
      wx.removeStorageSync(STORAGE_KEYS.ACTIVE_OPERATION_ID)
    }
  }

  subscribeOperation(
    operationId: string,
    listener: OperationEventListener,
  ): void {
    if (!this.operationListeners.has(operationId)) {
      this.operationListeners.set(operationId, new Set())
    }
    this.operationListeners.get(operationId)!.add(listener)
  }

  unsubscribeOperation(
    operationId: string,
    listener: OperationEventListener,
  ): void {
    this.operationListeners.get(operationId)?.delete(listener)
  }

  private emitToOperationListeners(
    operationId: string,
    msg: DeviceEventMessage,
  ): void {
    this.operationListeners.get(operationId)?.forEach(listener => {
      try {
        listener(msg)
      } catch (e) {
        console.error('Operation listener exception', e)
      }
    })
  }
}
