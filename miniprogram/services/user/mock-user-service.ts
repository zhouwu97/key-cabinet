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

  async login(studentNo?: string): Promise<User> {
    return new Promise((resolve, reject) => {
      setTimeout(() => {
        const targetNo = studentNo || '2021001'
        const user = MOCK_USERS.find(u => u.studentNo === targetNo) || MOCK_USERS[0]
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

  async updateProfile(data: Partial<User>): Promise<User> {
    return new Promise(resolve => {
      setTimeout(() => {
        if (this.currentUser) {
          this.currentUser = {
            ...this.currentUser,
            ...data,
            profileCompleted: true,
          }
        } else {
          this.currentUser = {
            id: 'U001',
            name: data.name || '张三',
            studentNo: data.studentNo || '2021001',
            phone: data.phone || '13800000000',
            department: data.department || '计算机学院',
            role: 'USER',
            status: 'ACTIVE',
            creditScore: 100,
            profileCompleted: true,
          }
        }
        this.saveToStorage()
        resolve({ ...this.currentUser })
      }, 200)
    })
  }
}
