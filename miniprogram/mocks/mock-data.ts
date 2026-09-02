import { User, UserRole, UserStatus } from '../models/user'
import { Key, KeyStatus } from '../models/key'
import { Device, DeviceStatus } from '../models/device'
import { Reservation, ReservationStatus } from '../models/reservation'
import { BorrowRecord, BorrowRecordStatus } from '../models/borrow-record'
import { KeySlot } from '../models/key-slot'
import { KeyPresenceState } from '../models/key-presence'
import { DeviceOperation } from '../models/device-operation'

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
    name: '一号钥匙柜（信息楼）',
    status: DeviceStatus.ONLINE,
    lastHeartbeat: Date.now(),
  },
  {
    id: 'CAB002',
    name: '二号钥匙柜（实验楼）',
    status: DeviceStatus.OFFLINE,
    lastHeartbeat: Date.now() - 300000,
  },
]

// ==================== 物理槽位 ====================
export const MOCK_KEY_SLOTS: KeySlot[] = [
  { id: 'SLOT01', deviceId: 'CAB001', slotNo: 1, keyId: 'KEY101', presence: KeyPresenceState.PRESENT, enabled: true, lastUpdated: Date.now() },
  { id: 'SLOT02', deviceId: 'CAB001', slotNo: 2, keyId: 'KEY102', presence: KeyPresenceState.PRESENT, enabled: true, lastUpdated: Date.now() },
  { id: 'SLOT03', deviceId: 'CAB001', slotNo: 3, keyId: 'KEY103', presence: KeyPresenceState.PRESENT, enabled: true, lastUpdated: Date.now() },
  { id: 'SLOT04', deviceId: 'CAB001', slotNo: 4, keyId: 'KEY104', presence: KeyPresenceState.ABSENT, enabled: true, lastUpdated: Date.now() },
  { id: 'SLOT05', deviceId: 'CAB001', slotNo: 5, keyId: 'KEY105', presence: KeyPresenceState.PRESENT, enabled: true, lastUpdated: Date.now() },
  { id: 'SLOT06', deviceId: 'CAB001', slotNo: 6, keyId: 'KEY106', presence: KeyPresenceState.ABSENT, enabled: true, lastUpdated: Date.now() },
  { id: 'SLOT07', deviceId: 'CAB001', slotNo: 7, keyId: 'KEY107', presence: KeyPresenceState.PRESENT, enabled: false, lastUpdated: Date.now() },
  { id: 'SLOT08', deviceId: 'CAB001', slotNo: 8, keyId: 'KEY108', presence: KeyPresenceState.PRESENT, enabled: true, lastUpdated: Date.now() },
  { id: 'SLOT09', deviceId: 'CAB001', slotNo: 9, keyId: 'KEY109', presence: KeyPresenceState.PRESENT, enabled: true, lastUpdated: Date.now() },
  { id: 'SLOT10', deviceId: 'CAB001', slotNo: 10, keyId: 'KEY110', presence: KeyPresenceState.PRESENT, enabled: false, lastUpdated: Date.now() },
]

// ==================== 钥匙 ====================
export const MOCK_KEYS: Key[] = [
  {
    id: 'KEY101',
    name: '101室钥匙',
    roomNo: '101',
    deviceId: 'CAB001',
    slotId: 'SLOT01',
    status: KeyStatus.AVAILABLE,
    enabled: true,
    requiresApproval: false,
    rfidTag: 'TAG101',
    description: '数字电路实验室101',
  },
  {
    id: 'KEY102',
    name: '102室钥匙',
    roomNo: '102',
    deviceId: 'CAB001',
    slotId: 'SLOT02',
    status: KeyStatus.AVAILABLE,
    enabled: true,
    requiresApproval: false,
    rfidTag: 'TAG102',
    description: '微机原理实验室102',
  },
  {
    id: 'KEY103',
    name: '103室钥匙',
    roomNo: '103',
    deviceId: 'CAB001',
    slotId: 'SLOT03',
    status: KeyStatus.AVAILABLE,
    enabled: true,
    requiresApproval: false,
    rfidTag: 'TAG103',
    description: '嵌入式系统实验室103',
  },
  {
    id: 'KEY104',
    name: '104室钥匙',
    roomNo: '104',
    deviceId: 'CAB001',
    slotId: 'SLOT04',
    status: KeyStatus.BORROWED,
    enabled: true,
    requiresApproval: false,
    rfidTag: 'TAG104',
    description: '教师办公研讨室104',
  },
  {
    id: 'KEY105',
    name: '105室钥匙',
    roomNo: '105',
    deviceId: 'CAB001',
    slotId: 'SLOT05',
    status: KeyStatus.RESERVED,
    enabled: true,
    requiresApproval: false,
    rfidTag: 'TAG105',
    description: '多媒体学术会议室105',
  },
  {
    id: 'KEY106',
    name: '106室钥匙',
    roomNo: '106',
    deviceId: 'CAB001',
    slotId: 'SLOT06',
    status: KeyStatus.OVERDUE,
    enabled: true,
    requiresApproval: false,
    rfidTag: 'TAG106',
    description: '网络工程实验室106',
  },
  {
    id: 'KEY107',
    name: '107室钥匙',
    roomNo: '107',
    deviceId: 'CAB001',
    slotId: 'SLOT07',
    status: KeyStatus.MAINTENANCE,
    enabled: false,
    requiresApproval: false,
    rfidTag: 'TAG107',
    description: '实验室107（槽位维护中）',
  },
  {
    id: 'KEY108',
    name: '108室钥匙',
    roomNo: '108',
    deviceId: 'CAB001',
    slotId: 'SLOT08',
    status: KeyStatus.AVAILABLE,
    enabled: true,
    requiresApproval: false,
    rfidTag: 'TAG108',
    description: '辅导员办公室108',
  },
  {
    id: 'KEY109',
    name: '109室钥匙',
    roomNo: '109',
    deviceId: 'CAB001',
    slotId: 'SLOT09',
    status: KeyStatus.AVAILABLE,
    enabled: true,
    requiresApproval: false,
    rfidTag: 'TAG109',
    description: '研究生工作站109',
  },
  {
    id: 'KEY110',
    name: '110室钥匙',
    roomNo: '110',
    deviceId: 'CAB001',
    slotId: 'SLOT10',
    status: KeyStatus.DISABLED,
    enabled: false,
    requiresApproval: false,
    rfidTag: 'TAG110',
    description: '实验耗材仓库110（已停用）',
  },
]

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
    purpose: '实验准备与课程演示',
    createdAt: now - oneHour,
    pickupWindowStart: now - oneHour,
    pickupWindowEnd: now + oneHour * 2,
    expectedReturnAt: now + oneHour * 4,
    approvedAt: now - oneHour,
  },
]

// ==================== 借还记录 ====================
export const MOCK_BORROW_RECORDS: BorrowRecord[] = [
  // 当前借用 (未逾期)
  {
    id: 'BR001',
    userId: 'U001',
    keyId: 'KEY104',
    slotId: 'SLOT04',
    deviceId: 'CAB001',
    status: BorrowRecordStatus.BORROWED,
    purpose: '日常办公与研讨',
    borrowedAt: now - oneHour * 3,
    expectedReturnAt: now + oneHour * 5,
  },
  // 逾期借用
  {
    id: 'BR002',
    userId: 'U001',
    keyId: 'KEY106',
    slotId: 'SLOT06',
    deviceId: 'CAB001',
    status: BorrowRecordStatus.BORROWED,
    purpose: '设备检修排障',
    borrowedAt: now - oneDay * 2,
    expectedReturnAt: now - oneDay,
    overdueAt: now - oneDay,
  },
  // 已完成
  {
    id: 'BR003',
    userId: 'U001',
    keyId: 'KEY102',
    slotId: 'SLOT02',
    deviceId: 'CAB001',
    status: BorrowRecordStatus.COMPLETED,
    purpose: '卫生清洁与打扫',
    borrowedAt: now - oneDay * 3,
    expectedReturnAt: now - oneDay * 2,
    returnedAt: now - oneDay * 2 - oneHour,
  },
  {
    id: 'BR004',
    userId: 'U001',
    keyId: 'KEY101',
    slotId: 'SLOT01',
    deviceId: 'CAB001',
    status: BorrowRecordStatus.COMPLETED,
    purpose: '课程实验准备',
    borrowedAt: now - oneDay * 5,
    expectedReturnAt: now - oneDay * 4,
    returnedAt: now - oneDay * 4 - oneHour * 2,
  },
]

// ==================== 操作记录 ====================
export const MOCK_OPERATIONS: DeviceOperation[] = []

// ==================== 存储键名 ====================
export const STORAGE_KEYS = {
  USERS: 'mock_users',
  KEYS: 'mock_keys',
  KEY_SLOTS: 'mock_key_slots',
  DEVICES: 'mock_devices',
  RESERVATIONS: 'mock_reservations',
  BORROW_RECORDS: 'mock_borrow_records',
  OPERATIONS: 'mock_operations',
  ACTIVE_OPERATION_ID: 'mock_active_operation_id',
  CURRENT_USER: 'current_user',
  MOCK_SCENARIO: 'mock_scenario',
}
