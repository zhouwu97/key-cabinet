import { httpClient } from '../../api/http-client'
import type { User } from '../../models/user'
import type { UserService } from './user-service'

/**
 * API 版本的 UserService（对接真实后端）
 */
export class ApiUserService implements UserService {
  async getCurrentUser(): Promise<User | null> {
    try {
      // 先尝试从本地缓存读取
      const cached = wx.getStorageSync('user')
      if (cached) {
        return cached as User
      }

      // 否则从后端获取
      const user = await httpClient.request<User>({
        url: '/api/v1/me',
      })

      // 更新缓存
      wx.setStorageSync('user', user)

      return user
    } catch (err) {
      console.error('Failed to get current user:', err)
      return null
    }
  }

  async getUserById(userId: string): Promise<User | null> {
    try {
      return await httpClient.request<User>({
        url: `/api/v1/users/${userId}`,
      })
    } catch (err) {
      console.error(`Failed to get user ${userId}:`, err)
      return null
    }
  }

  async login(_studentNo?: string): Promise<User> {
    // API 版本通过微信登录
    const { authService } = await import('../auth/auth-service')
    const response = await authService.login()
    return response.user
  }

  async updateProfile(data: Partial<User>): Promise<User> {
    const updatedUser = await httpClient.request<User>({
      url: '/api/v1/me',
      method: 'PATCH',
      data,
    })

    // 更新本地缓存
    wx.setStorageSync('user', updatedUser)

    return updatedUser
  }
}
