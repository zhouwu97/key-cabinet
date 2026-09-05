import { currentConfig } from '../config/index'
import { MockKeyService } from './key/mock-key-service'
import { MockReservationService } from './reservation/mock-reservation-service'
import { MockBorrowService } from './borrow/mock-borrow-service'
import { MockUserService } from './user/mock-user-service'
import { ApiUserService } from './user/api-user-service'
import { UserService } from './user/user-service'
import { MockDeviceService } from './device/mock-device-service'
import { MockOperationService } from './operation/mock-operation-service'

// 全局服务实例（根据 config.dataMode 自动切换 Mock 或 API 模式）
export const keyService = new MockKeyService()
export const reservationService = new MockReservationService()
export const borrowService = new MockBorrowService()
export const userService: UserService =
  currentConfig.dataMode === 'api' ? new ApiUserService() : new MockUserService()
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
export * from './auth/index'
