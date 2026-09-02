export type UserRole = 'USER' | 'ADMIN'

export type UserStatus = 'ACTIVE' | 'DISABLED'

export interface User {
  id: string
  name: string
  studentNo: string
  phone?: string
  role: UserRole
  status: UserStatus
}
