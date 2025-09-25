import axios from 'axios'
import { ElMessage } from 'element-plus'
import router from '../router'
import { getAuthToken, clearAuthToken } from './auth'

// 创建axios实例
const request = axios.create({
  baseURL: 'http://localhost:8080',
  timeout: 10000
})

// 请求拦截器
request.interceptors.request.use(
  async (config) => {
    // 使用 auth.js 获取token（sessionStorage 优先，localStorage 兼容回退）
    const token = await getAuthToken()
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器
request.interceptors.response.use(
  (response) => {
    return response.data
  },
  (error) => {
    if (error.response?.status === 401) {
      ElMessage.error('登录已过期，请重新登录')
      // 使用 auth.js 清理认证状态（同时清理 sessionStorage 和 localStorage）
      clearAuthToken()
      router.push('/')
    } else if (error.response?.data?.error) {
      ElMessage.error(error.response.data.error)
    } else {
      ElMessage.error('网络错误，请稍后重试')
    }
    return Promise.reject(error)
  }
)

export default request