import { httpClient } from '../../api/http-client'
import type { User } from '../../models/user'

/**
 * 登录响应
 */
export interface LoginResponse {
  accessToken: string
  expiresIn: number
  user: User
}

/**
 * 认证服务接口
 */
export interface IAuthService {
  login(): Promise<LoginResponse>
  getMe(): Promise<User>
  logout(): void
}

/**
 * 真实认证服务（对接后端 API）
 */
export class AuthService implements IAuthService {
	private loginPromise: Promise<LoginResponse> | null = null

  /**
   * 微信登录
   */
	login(): Promise<LoginResponse> {
		// 启动阶段优先复用本地会话，服务端返回 401 时由 HttpClient 再触发登录刷新。
		const accessToken = wx.getStorageSync('accessToken')
		const cachedUser = wx.getStorageSync('user') as User | undefined
		if (accessToken && cachedUser) {
			return Promise.resolve({ accessToken, expiresIn: 0, user: cachedUser })
		}
		if (!this.loginPromise) {
			this.loginPromise = this.performLogin().finally(() => {
				this.loginPromise = null
			})
		}
		return this.loginPromise
	}

	private async performLogin(): Promise<LoginResponse> {
    // 1. 调用微信登录获取 code
    const { code } = await wx.login()

    // 2. 发送 code 到后端换取 token
    const response = await httpClient.request<LoginResponse>({
      url: '/api/v1/auth/wechat-login',
      method: 'POST',
      data: { code },
    })

    // 3. 保存 token 和用户信息
    wx.setStorageSync('accessToken', response.accessToken)
    wx.setStorageSync('user', response.user)

    return response
  }

  /**
   * 获取当前用户信息
   */
  async getMe(): Promise<User> {
    const user = await httpClient.request<User>({
      url: '/api/v1/me',
    })

    // 更新本地缓存
    wx.setStorageSync('user', user)

    return user
  }

  /**
   * 登出
   */
  logout(): void {
    wx.removeStorageSync('accessToken')
    wx.removeStorageSync('user')
  }
}

export const authService = new AuthService()
