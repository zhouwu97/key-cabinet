import { userService } from '../../services/index'

Page({
  data: {
    name: '',
    studentNo: '',
    department: '',
    phone: '',
    departmentSuggestions: [
      '计算机科学与技术学院',
      '人工智能与自动化学院',
      '信息与通信工程学院',
      '实验教学与设备管理中心',
    ],
    submitting: false,
  },

  async onLoad() {
    try {
      const user = await userService.getCurrentUser()
      if (user) {
        this.setData({
          name: user.name && user.name !== '微信用户' ? user.name : '',
          studentNo: user.studentNo || '',
          department: user.department || '',
          phone: user.phone || '',
        })
      }
    } catch (e) {
      console.error('获取用户信息失败', e)
    }
  },

  onInputName(e: any) {
    this.setData({ name: e.detail.value.trim() })
  },

  onInputStudentNo(e: any) {
    this.setData({ studentNo: e.detail.value.trim() })
  },

  onInputDepartment(e: any) {
    this.setData({ department: e.detail.value.trim() })
  },

  onSelectDepartment(e: any) {
    const dep = e.currentTarget.dataset.dep
    this.setData({ department: dep })
  },

  onInputPhone(e: any) {
    this.setData({ phone: e.detail.value.trim() })
  },

  async onSubmit() {
    const { name, studentNo, department, phone, submitting } = this.data
    if (submitting) return

    if (!name) {
      wx.showToast({ title: '请输入真实姓名', icon: 'none' })
      return
    }

    if (!studentNo) {
      wx.showToast({ title: '请输入学号或工号', icon: 'none' })
      return
    }

    if (!department) {
      wx.showToast({ title: '请输入所属学院或部门', icon: 'none' })
      return
    }

    if (phone && !/^1\d{10}$/.test(phone)) {
      wx.showToast({ title: '请输入有效的11位手机号', icon: 'none' })
      return
    }

    this.setData({ submitting: true })
    wx.showLoading({ title: '正在提交认证...' })

    try {
      await userService.updateProfile({
        name,
        studentNo,
        department,
        phone,
      })

      wx.hideLoading()
      wx.showToast({ title: '认证成功', icon: 'success' })

      setTimeout(() => {
        const pages = getCurrentPages()
        if (pages.length > 1) {
          wx.navigateBack()
        } else {
          wx.switchTab({ url: '/pages/home/home' })
        }
      }, 1200)
    } catch (err: any) {
      wx.hideLoading()
      this.setData({ submitting: false })
      wx.showToast({ title: err.message || '认证失败，请重试', icon: 'none' })
    }
  },
})
