import { User } from '../../models/user'
import { UserService } from './user-service'
import { MOCK_USERS, CURRENT_USER, STORAGE_KEYS } from '../../mocks/mock-data'

export class MockUserService implements UserService {
  private currentUser: User | null = null

  constructor() {
    this.loadFromStorage()
  }

  private loadFromStorage(): void {
    try {
      const stored = wx.getStorageSync(STORAGE_KEYS.CURRENT_USER)
      this.currentUser = stored || CURRENT_USER
      this.saveToStorage()
    } catch (e) {
      console.error('加载用户数据失败', e)
      this.currentUser = CURRENT_USER
    }
  }

  private saveToStorage(): void {
    try {
      wx.setStorageSync(STORAGE_KEYS.CURRENT_USER, this.currentUser)
    } catch (e) {
      console.error('保存用户数据失败', e)
    }
  }

  async getCurrentUser(): Promise<User | null> {
    return new Promise(resolve => {
      setTimeout(() => {
        resolve(this.currentUser ? { ...this.currentUser } : null)
      }, 100)
    })
  }

  async getUserById(userId: string): Promise<User | null> {
    return new Promise(resolve => {
      setTimeout(() => {
        const user = MOCK_USERS.find(u => u.id === userId)
        resolve(user ? { ...user } : null)
      }, 150)
    })
  }

  async login(studentNo: string): Promise<User> {
    return new Promise((resolve, reject) => {
      setTimeout(() => {
        const user = MOCK_USERS.find(u => u.studentNo === studentNo)
        if (user) {
          this.currentUser = user
          this.saveToStorage()
          resolve({ ...user })
        } else {
          reject(new Error('用户不存在'))
        }
      }, 300)
    })
  }
}
