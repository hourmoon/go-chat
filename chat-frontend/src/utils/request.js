import axios from 'axios'

const service = axios.create({
  baseURL: 'http://localhost:8080',
  timeout: 5000,
  withCredentials: true // 确保发送凭据（cookies）
})

// 请求拦截器：带 token
service.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// 响应拦截器：直接返回 data
service.interceptors.response.use(
  response => response.data, 
  error => {
    console.error('API 错误:', error)
    return Promise.reject(error.response?.data || error)
  }
)

export default service