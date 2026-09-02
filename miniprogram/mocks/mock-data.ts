import { User, UserRole, UserStatus } from '../models/user'
import { Key, KeyStatus } from '../models/key'
import { Device, DeviceStatus } from '../models/device'
import { Reservation, ReservationStatus } from '../models/reservation'
import { BorrowRecord, BorrowRecordStatus } from '../models/borrow-record'
import { KeyPhysicalState, KeyLocation } from '../models/key-physical-state'

// ==================== 用户 ====================
export const MOCK_USERS: User[] = [
  {
    id: 'U001',
    name: '张三',
    studentNo: '2021001',
    phone: '13800138001',
    role: 'USER' as UserRole,
    status: 'ACTIVE' as UserStatus,
  },
  {
    id: 'A001',
    name: '管理员',
    studentNo: 'ADMIN001',
    phone: '13800138000',
    role: 'ADMIN' as UserRole,
    status: 'ACTIVE' as UserStatus,
  },
]

// 当前登录用户
export const CURRENT_USER = MOCK_USERS[0]

// ==================== 设备 ====================
export const MOCK_DEVICES: Device[] = [
  {
    id: 'CAB001',
    name: '一号钥匙柜',
    status: DeviceStatus.ONLINE,
    lastHeartbeat: Date.now(),
  },
  {
    id: 'CAB002',
    name: '二号钥匙柜',
    status: DeviceStatus.OFFLINE,
    lastHeartbeat: Date.now() - 300000,
  },
]

// ==================== 钥匙 ====================
export const MOCK_KEYS: Key[] = [
  {
    id: 'KEY101',
    name: '101室钥匙',
    roomNo: '101',
    deviceId: 'CAB001',
    status: KeyStatus.AVAILABLE,
    enabled: true,
    requiresApproval: false,
    rfidTag: 'TAG101',
    description: '实验室101',
  },
  {
    id: 'KEY102',
    name: '102室钥匙',
    roomNo: '102',
    deviceId: 'CAB001',
    status: KeyStatus.AVAILABLE,
    enabled: true,
    requiresApproval: false,
    rfidTag: 'TAG102',
    description: '实验室102',
  },
  {
    id: 'KEY103',
    name: '103室钥匙',
    roomNo: '103',
    deviceId: 'CAB001',
    status: KeyStatus.AVAILABLE,
    enabled: true,
    requiresApproval: false,
    rfidTag: 'TAG103',
    description: '实验室103',
  },
  {
    id: 'KEY104',
    name: '104室钥匙',
    roomNo: '104',
    deviceId: 'CAB001',
    status: KeyStatus.BORROWED,
    enabled: true,
    requiresApproval: false,
    rfidTag: 'TAG104',
    description: '办公室104',
  },
  {
    id: 'KEY105',
    name: '105室钥匙',
    roomNo: '105',
    deviceId: 'CAB001',
    status: KeyStatus.RESERVED,
    enabled: true,
    requiresApproval: true,
    rfidTag: 'TAG105',
    description: '会议室105',
  },
  {
    id: 'KEY106',
    name: '106室钥匙',
    roomNo: '106',
    deviceId: 'CAB001',
    status: KeyStatus.OVERDUE,
    enabled: true,
    requiresApproval: false,
    rfidTag: 'TAG106',
    description: '实验室106',
  },
  {
    id: 'KEY107',
    name: '107室钥匙',
    roomNo: '107',
    deviceId: 'CAB001',
    status: KeyStatus.MAINTENANCE,
    enabled: false,
    requiresApproval: false,
    rfidTag: 'TAG107',
    description: '实验室107（维护中）',
  },
  {
    id: 'KEY108',
    name: '108室钥匙',
    roomNo: '108',
    deviceId: 'CAB001',
    status: KeyStatus.AVAILABLE,
    enabled: true,
    requiresApproval: false,
    rfidTag: 'TAG108',
    description: '办公室108',
  },
  {
    id: 'KEY109',
    name: '109室钥匙',
    roomNo: '109',
    deviceId: 'CAB001',
    status: KeyStatus.AVAILABLE,
    enabled: true,
    requiresApproval: false,
    rfidTag: 'TAG109',
    description: '实验室109',
  },
  {
    id: 'KEY110',
    name: '110室钥匙',
    roomNo: '110',
    deviceId: 'CAB001',
    status: KeyStatus.DISABLED,
    enabled: false,
    requiresApproval: false,
    rfidTag: 'TAG110',
    description: '仓库110（已停用）',
  },
]

// ==================== 钥匙物理状态 ====================
export const MOCK_KEY_LOCATIONS: KeyLocation[] = MOCK_KEYS.map((key, index) => ({
  keyId: key.id,
  deviceId: key.deviceId,
  physicalState:
    key.status === KeyStatus.BORROWED || key.status === KeyStatus.OVERDUE
      ? KeyPhysicalState.OUT
      : KeyPhysicalState.IN_CABINET,
  slotPosition: index + 1,
  lastUpdated: Date.now(),
}))

// ==================== 预约 ====================
const now = Date.now()
const oneHour = 3600000
const oneDay = 86400000

export const MOCK_RESERVATIONS: Reservation[] = [
  {
    id: 'RSV001',
    userId: 'U001',
    keyId: 'KEY105',
    status: ReservationStatus.ACTIVE,
    purpose: '实验准备',
    reservedAt: now - oneHour,
    expiresAt: now + oneHour * 2,
  },
]

// ==================== 借还记录 ====================
export const MOCK_BORROW_RECORDS: BorrowRecord[] = [
  // 当前借用
  {
    id: 'BR001',
    userId: 'U001',
    keyId: 'KEY104',
    deviceId: 'CAB001',
    status: BorrowRecordStatus.BORROWED,
    purpose: '日常办公',
    borrowedAt: now - oneHour * 3,
    expectedReturnAt: now + oneHour * 5,
  },
  // 逾期
  {
    id: 'BR002',
    userId: 'U001',
    keyId: 'KEY106',
    deviceId: 'CAB001',
    status: BorrowRecordStatus.OVERDUE,
    purpose: '设备检修',
    borrowedAt: now - oneDay * 2,
    expectedReturnAt: now - oneDay,
  },
  // 已完成
  {
    id: 'BR003',
    userId: 'U001',
    keyId: 'KEY102',
    deviceId: 'CAB001',
    status: BorrowRecordStatus.COMPLETED,
    purpose: '清洁打扫',
    borrowedAt: now - oneDay * 3,
    expectedReturnAt: now - oneDay * 2,
    returnedAt: now - oneDay * 2 - oneHour,
  },
  {
    id: 'BR004',
    userId: 'U001',
    keyId: 'KEY101',
    deviceId: 'CAB001',
    status: BorrowRecordStatus.COMPLETED,
    purpose: '实验准备',
    borrowedAt: now - oneDay * 5,
    expectedReturnAt: now - oneDay * 4,
    returnedAt: now - oneDay * 4 - oneHour * 2,
  },
]

// ==================== 存储键名 ====================
export const STORAGE_KEYS = {
  USERS: 'mock_users',
  KEYS: 'mock_keys',
  KEY_LOCATIONS: 'mock_key_locations',
  DEVICES: 'mock_devices',
  RESERVATIONS: 'mock_reservations',
  BORROW_RECORDS: 'mock_borrow_records',
  CURRENT_USER: 'current_user',
}
