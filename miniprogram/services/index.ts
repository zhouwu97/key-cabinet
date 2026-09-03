import { MockKeyService } from './key/mock-key-service'
import { MockReservationService } from './reservation/mock-reservation-service'
import { MockBorrowService } from './borrow/mock-borrow-service'
import { MockUserService } from './user/mock-user-service'
import { MockDeviceService } from './device/mock-device-service'
import { MockOperationService } from './operation/mock-operation-service'

// 全局服务实例
export const keyService = new MockKeyService()
export const reservationService = new MockReservationService()
export const borrowService = new MockBorrowService()
export const userService = new MockUserService()
export const deviceService = new MockDeviceService()
export const operationService = new MockOperationService(
  deviceService,
  keyService,
  reservationService,
  borrowService,
)

export * from './key/index'
export * from './reservation/index'
export * from './borrow/index'
export * from './user/index'
export * from './device/index'
export * from './operation/index'
