import { currentConfig } from '../config/index'
import { KeyService } from './key/key-service'
import { MockKeyService } from './key/mock-key-service'
import { ApiKeyService } from './key/api-key-service'
import { ReservationService } from './reservation/reservation-service'
import { MockReservationService } from './reservation/mock-reservation-service'
import { ApiReservationService } from './reservation/api-reservation-service'
import { BorrowService } from './borrow/borrow-service'
import { MockBorrowService } from './borrow/mock-borrow-service'
import { ApiBorrowService } from './borrow/api-borrow-service'
import { MockUserService } from './user/mock-user-service'
import { ApiUserService } from './user/api-user-service'
import { UserService } from './user/user-service'
import { ApiDeviceService } from './device/api-device-service'
import { MockDeviceService } from './device/mock-device-service'
import { DeviceService } from './device/device-service'
import { ApiOperationService } from './operation/api-operation-service'
import { MockOperationService } from './operation/mock-operation-service'
import { OperationService } from './operation/operation-service'

const isApi = currentConfig.dataMode === 'api'

// 全局服务实例：模式切换必须覆盖整条业务链，避免真实登录后继续读 Mock 数据。
export const keyService: KeyService = isApi ? new ApiKeyService() : new MockKeyService()
export const reservationService: ReservationService = isApi
  ? new ApiReservationService()
  : new MockReservationService()
export const borrowService: BorrowService = isApi ? new ApiBorrowService() : new MockBorrowService()
export const userService: UserService =
  isApi ? new ApiUserService() : new MockUserService()
export const deviceService: DeviceService = isApi ? new ApiDeviceService() : new MockDeviceService()
export const operationService: OperationService = isApi
  ? new ApiOperationService()
  : new MockOperationService(deviceService, keyService, reservationService, borrowService)

export * from './key/index'
export * from './reservation/index'
export * from './borrow/index'
export * from './user/index'
export * from './device/index'
export * from './operation/index'
export * from './auth/index'
