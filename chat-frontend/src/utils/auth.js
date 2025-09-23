import { ref, nextTick } from 'vue'

/**
 * 认证状态管理器
 * 解决登录后token时序问题
 */

// 认证状态
const isAuthenticated = ref(false)
const authToken = ref('')

// 认证状态检查队列
const authCheckQueue = []

/**
 * 设置认证token
 * @param {string} token JWT token
 */
export const setAuthToken = async (token) => {
  // 存储到localStorage
  localStorage.setItem('token', token)
  
  // 等待DOM更新
  await nextTick()
  
  // 再等待一个微任务，确保localStorage写入完成
  await new Promise(resolve => setTimeout(resolve, 10))
  
  // 更新状态
  authToken.value = token
  isAuthenticated.value = true
  
  // 处理队列中的请求
  while (authCheckQueue.length > 0) {
    const resolver = authCheckQueue.shift()
    resolver(token)
  }
  
  console.log('✅ 认证token已就绪:', token.substring(0, 20) + '...')
}

/**
 * 清除认证token
 */
export const clearAuthToken = () => {
  localStorage.removeItem('token')
  localStorage.removeItem('username')
  authToken.value = ''
  isAuthenticated.value = false
  console.log('🔒 认证token已清除')
}

/**
 * 获取认证token
 * @returns {Promise<string|null>} 返回token或null
 */
export const getAuthToken = () => {
  return new Promise((resolve) => {
    // 如果已经有token，直接返回
    if (authToken.value) {
      resolve(authToken.value)
      return
    }
    
    // 尝试从localStorage获取
    const storedToken = localStorage.getItem('token')
    if (storedToken) {
      authToken.value = storedToken
      isAuthenticated.value = true
      resolve(storedToken)
      return
    }
    
    // 如果没有token，加入等待队列
    authCheckQueue.push(resolve)
    
    // 设置超时，避免无限等待
    setTimeout(() => {
      const index = authCheckQueue.indexOf(resolve)
      if (index > -1) {
        authCheckQueue.splice(index, 1)
        resolve(null) // 超时返回null
      }
    }, 1000) // 1秒超时
  })
}

/**
 * 检查是否已认证
 * @returns {boolean}
 */
export const checkAuth = () => {
  return isAuthenticated.value || !!localStorage.getItem('token')
}

/**
 * 等待认证就绪
 * @returns {Promise<boolean>} 认证是否成功
 */
export const waitForAuth = async () => {
  const token = await getAuthToken()
  return !!token
}


