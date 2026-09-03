/**
 * API 统一响应格式（成功）
 */
export interface ApiResponse<T> {
  code: number
  message: string
  data: T
}

/**
 * API 错误响应格式
 */
export interface ApiError {
  code: number
  errorCode: string
  message: string
  data: null
  timestamp: string
}

/**
 * HTTP 请求配置
 */
export interface RequestOptions {
  url: string
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  data?: any
  headers?: Record<string, string>
}

/**
 * 统一 HTTP 客户端
 *
 * 功能：
 * - 自动添加 Authorization 头
 * - 统一错误处理
 * - 401 自动重新登录
 * - 解包 ApiResponse.data
 */
export class HttpClient {
  private baseURL = 'http://localhost:8080'
  private refreshing = false

  /**
   * 发起 HTTP 请求
   */
  async request<T>(options: RequestOptions): Promise<T> {
    const token = wx.getStorageSync('accessToken')

    try {
      const res = await wx.request({
        url: `${this.baseURL}${options.url}`,
        method: options.method || 'GET',
        data: options.data,
        header: {
          'Content-Type': 'application/json',
          ...(token && { Authorization: `Bearer ${token}` }),
          ...options.headers,
        },
      })

      // 401 Token 失效，自动重新登录
      if (res.statusCode === 401 && !this.refreshing) {
        this.refreshing = true
        try {
          await this.refreshAuth()
          this.refreshing = false
          // 重试原请求
          return this.request(options)
        } catch (e) {
          this.refreshing = false
          throw e
        }
      }

      // HTTP 错误
      if (res.statusCode >= 400) {
        const error = res.data as ApiError
        throw new Error(error.message || `请求失败 (${res.statusCode})`)
      }

      // 成功响应，解包 data
      const response = res.data as ApiResponse<T>
      if (response.code !== 0) {
        throw new Error(response.message || '业务错误')
      }

      return response.data
    } catch (err: any) {
      console.error('HTTP Request Error:', err)
      throw err
    }
  }

  /**
   * 刷新认证
   */
  private async refreshAuth() {
    // 动态导入避免循环依赖
    const { authService } = await import('../services/auth/auth-service')
    await authService.login()
  }

  /**
   * 设置 BaseURL（用于环境切换）
   */
  setBaseURL(url: string) {
    this.baseURL = url
  }
}

export const httpClient = new HttpClient()
