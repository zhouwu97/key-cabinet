import { User } from '../../models/user'

export interface UserService {
  /** 获取当前用户 */
  getCurrentUser(): Promise<User | null>

  /** 获取用户信息 */
  getUserById(userId: string): Promise<User | null>

  /** 模拟登录 */
  login(studentNo: string): Promise<User>
}
