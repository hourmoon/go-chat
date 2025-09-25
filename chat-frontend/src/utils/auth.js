import { ref, nextTick } from 'vue'

/**
 * 认证状态管理器
 * 支持每标签独立会话：sessionStorage 优先，localStorage 兼容回退
 */

// 认证状态
const isAuthenticated = ref(false)
const authToken = ref('')
const username = ref('')

// 认证状态检查队列
const authCheckQueue = []

/**
 * 设置认证token
 * @param {string} token JWT token
 */
export const setAuthToken = async (token) => {
  // 优先写入sessionStorage（每标签独立）
  sessionStorage.setItem('token', token)
  
  // 清理localStorage中的旧token（避免跨标签污染）
  localStorage.removeItem('token')
  
  // 等待DOM更新
  await nextTick()
  
  // 再等待一个微任务，确保存储写入完成
  await new Promise(resolve => setTimeout(resolve, 10))
  
  // 更新状态
  authToken.value = token
  isAuthenticated.value = true
  
  // 处理队列中的请求
  while (authCheckQueue.length > 0) {
    const resolver = authCheckQueue.shift()
    resolver(token)
  }
  
  console.log('✅ 认证token已就绪（sessionStorage）:', token.substring(0, 20) + '...')
}

/**
 * 设置用户名
 * @param {string} name 用户名
 */
export const setUsername = (name) => {
  // 优先写入sessionStorage（每标签独立）
  sessionStorage.setItem('username', name)
  
  // 清理localStorage中的旧username（避免跨标签污染）
  localStorage.removeItem('username')
  
  // 更新状态
  username.value = name
  
  console.log('✅ 用户名已设置（sessionStorage）:', name)
}

/**
 * 清除认证token
 */
export const clearAuthToken = () => {
  // 同时清理两处存储
  sessionStorage.removeItem('token')
  sessionStorage.removeItem('username')
  localStorage.removeItem('token')
  localStorage.removeItem('username')
  
  // 清理状态
  authToken.value = ''
  username.value = ''
  isAuthenticated.value = false
  
  console.log('🔒 认证token已清除（sessionStorage + localStorage）')
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
    
    // 优先从sessionStorage获取
    let storedToken = sessionStorage.getItem('token')
    if (storedToken) {
      authToken.value = storedToken
      isAuthenticated.value = true
      resolve(storedToken)
      return
    }
    
    // 兼容回退：从localStorage获取（为兼容旧数据）
    storedToken = localStorage.getItem('token')
    if (storedToken) {
      authToken.value = storedToken
      isAuthenticated.value = true
      // 迁移到sessionStorage
      sessionStorage.setItem('token', storedToken)
      localStorage.removeItem('token')
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
 * 获取用户名
 * @returns {string|null} 返回用户名或null
 */
export const getUsername = () => {
  // 如果已经有用户名，直接返回
  if (username.value) {
    return username.value
  }
  
  // 优先从sessionStorage获取
  let storedUsername = sessionStorage.getItem('username')
  if (storedUsername) {
    username.value = storedUsername
    return storedUsername
  }
  
  // 兼容回退：从localStorage获取（为兼容旧数据）
  storedUsername = localStorage.getItem('username')
  if (storedUsername) {
    username.value = storedUsername
    // 迁移到sessionStorage
    sessionStorage.setItem('username', storedUsername)
    localStorage.removeItem('username')
    return storedUsername
  }
  
  return null
}

/**
 * 检查是否已认证
 * @returns {boolean}
 */
export const checkAuth = () => {
  return isAuthenticated.value || !!sessionStorage.getItem('token') || !!localStorage.getItem('token')
}

/**
 * 等待认证就绪
 * @returns {Promise<boolean>} 认证是否成功
 */
export const waitForAuth = async () => {
  const token = await getAuthToken()
  return !!token
}

/**
 * 获取当前用户ID（从token解析）
 * @returns {number|null} 用户ID或null
 */
export const getCurrentUserId = async () => {
  const token = await getAuthToken()
  if (!token) return null
  
  try {
    const payload = JSON.parse(atob(token.split('.')[1]))
    return payload.userID || 0
  } catch (error) {
    console.error('解析用户ID失败:', error)
    return null
  }
}


