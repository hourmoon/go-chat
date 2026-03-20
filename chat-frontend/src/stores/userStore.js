import { defineStore } from 'pinia'
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { ElMessageBox } from 'element-plus'
import request from '../utils/request'
import { getAuthToken, clearAuthToken, getCurrentUserId } from '../utils/auth'

/**
 * 用户个人信息状态管理
 * setup 风格 defineStore
 */
export const useUserStore = defineStore('user', () => {
  // ── 状态 ──────────────────────────────────────────────
  const profile = ref({
    id: 0,
    username: '',
    avatar: '',
    bio: '',
    status: 'online'
  })
  const isLoggedIn = ref(false)
  const loading = ref(false)

  // ── Actions ───────────────────────────────────────────

  /** 初始化：从 token 读取用户 ID，判断登录态 */
  async function init() {
    const token = await getAuthToken()
    if (!token) {
      isLoggedIn.value = false
      return
    }
    const userId = await getCurrentUserId()
    if (userId) {
      profile.value.id = userId
      isLoggedIn.value = true
    }
  }

  /** 拉取完整用户资料 */
  async function fetchProfile() {
    try {
      const res = await request.get('/profile')
      if (res.success && res.data) {
        profile.value = {
          id: res.data.id,
          username: res.data.username,
          avatar: res.data.avatar || '',
          bio: res.data.bio || '',
          status: res.data.status || 'online'
        }
        isLoggedIn.value = true
        console.log('✅ 用户资料加载成功: userID=', profile.value.id)
      }
    } catch (error) {
      console.error('❌ 获取用户资料失败:', error)
      ElMessage.error('获取用户资料失败')
    }
  }

  /** 更新个性签名 */
  async function updateProfile(bio) {
    loading.value = true
    try {
      const res = await request.put('/profile', { bio })
      if (res.success) {
        profile.value.bio = bio
        ElMessage.success('保存成功')
        console.log('✅ 用户资料已更新: userID=', profile.value.id)
      }
    } catch (error) {
      console.error('❌ 更新用户资料失败:', error)
      ElMessage.error('保存失败')
      throw error
    } finally {
      loading.value = false
    }
  }

  /** 更新在线状态 */
  async function updateStatus(status) {
    try {
      const res = await request.put('/profile/status', { status })
      if (res.success) {
        profile.value.status = status
        console.log('✅ 状态已更新: status=', status)
      }
    } catch (error) {
      console.error('❌ 更新状态失败:', error)
      ElMessage.error('状态更新失败')
      throw error
    }
  }

  /** 上传头像 */
  async function uploadAvatar(file) {
    const formData = new FormData()
    formData.append('avatar', file)
    loading.value = true
    try {
      const res = await request.post('/profile/avatar', formData, {
        headers: { 'Content-Type': 'multipart/form-data' }
      })
      if (res.success) {
        profile.value.avatar = res.avatar
        ElMessage.success('头像上传成功')
        console.log('✅ 头像已更新: userID=', profile.value.id)
      }
    } catch (error) {
      console.error('❌ 上传头像失败:', error)
      ElMessage.error('头像上传失败')
      throw error
    } finally {
      loading.value = false
    }
  }

  /** 退出登录：清除 token + 重置所有状态 */
  function logout() {
    clearAuthToken()
    profile.value = { id: 0, username: '', avatar: '', bio: '', status: 'online' }
    isLoggedIn.value = false
    console.log('🔒 用户已退出登录')
  }

  return {
    profile,
    isLoggedIn,
    loading,
    init,
    fetchProfile,
    updateProfile,
    updateStatus,
    uploadAvatar,
    logout
  }
})
