import { User } from '../../models/user'

export interface UserService {
  /** 获取当前用户 */
  getCurrentUser(): Promise<User | null>

  /** 获取用户信息 */
  getUserById(userId: string): Promise<User | null>

  /** 登录 */
  login(studentNo?: string): Promise<User>

  /** 完善/更新个人资料 */
  updateProfile(data: Partial<User>): Promise<User>
}
