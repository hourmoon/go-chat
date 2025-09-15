<template>
  <div class="chat-container">
    <!-- 在线用户侧边栏 -->
    <div class="online-users-sidebar">
      <h3>在线用户 ({{ onlineUsers.length }})</h3>
      <div class="user-list">
        <div 
          v-for="user in onlineUsers" 
          :key="user.id"
          class="user-item"
          :class="{ 
            active: privateTarget === user.id,
            'current-user': user.id === currentUserID
          }"
          @click="startPrivateChat(user)"
        >
          <div class="user-avatar">
            <img 
              :src="user.avatar ? getFullAvatarUrl(user.avatar) : defaultAvatar" 
              :alt="user.username"
            />
            <span :class="['user-status', user.status]"></span>
          </div>
          <div class="user-info">
            <div class="user-name">{{ user.username }}</div>
            <div class="user-bio" v-if="user.bio">{{ user.bio }}</div>
            <div class="user-status-text">
              <span :class="['status-text', user.status]">
                {{ getStatusText(user.status) }}
              </span>
              <span v-if="user.id === currentUserID" class="you-label">(我)</span>
            </div>
          </div>
        </div>
      </div>
    </div>
    
    <!-- 主聊天区域 -->
    <div class="main-chat-area">
      <div class="chat-header">
        <h2>
          {{ privateTarget ? `与 ${privateTargetName} 的私聊` : '群聊' }}
          <el-button v-if="privateTarget" @click="exitPrivateChat" size="small" type="info">
            返回群聊
          </el-button>
        </h2>
        <div class="header-actions">
          <el-button @click="goToProfile" type="primary" size="small">
            <el-icon><User /></el-icon>
            个人资料
          </el-button>
          <el-button @click="logout" type="danger" size="small">退出</el-button>
        </div>
      </div>
      
      <div class="chat-box" ref="chatBox" @scroll="handleScroll">
        <!-- 加载更多指示器 -->
        <div v-if="isLoadingMore" class="loading-more">
          <el-icon class="is-loading"><Loading /></el-icon>
          <span>加载更多消息...</span>
        </div>
        
        <!-- 加载指示器 -->
        <div v-if="loadingHistory" class="loading-indicator">
          <el-icon class="is-loading"><Loading /></el-icon>
          <span>加载历史消息中...</span>
        </div>
        
        <!-- 消息列表 -->
        <div 
          v-for="(msg, index) in messages" 
          :key="index" 
          class="message" 
          :class="{ 
            'own-message': msg.isOwn, 
            'private-message': msg.isPrivate,
            'system-message': msg.isSystem
          }"
        >
          <div class="message-meta">
            <span class="message-sender">{{ msg.sender }}</span>
            <span class="message-time">{{ formatDisplayTime(msg.timestamp) }}</span>
          </div>
          
          <!-- 文本消息 -->
          <div v-if="msg.messageType === 'text'" class="message-content">{{ msg.content }}</div>
          
          <!-- 图片消息 -->
          <div v-else-if="msg.messageType === 'image'" class="image-message">
            <img :src="getFullFileUrl(msg.fileUrl)" :alt="msg.fileName" @load="scrollToBottom" />
            <div class="image-info">{{ msg.fileName }}</div>
          </div>
          
          <!-- 文件消息 -->
          <div v-else-if="msg.messageType === 'file'" class="file-message">
            <div class="file-icon">
              <el-icon><Document /></el-icon>
            </div>
            <div class="file-info">
              <a :href="getFullFileUrl(msg.fileUrl)" target="_blank" class="file-name">{{ msg.fileName }}</a>
              <div class="file-size">{{ formatFileSize(msg.fileSize) }}</div>
            </div>
          </div>
          
          <div v-if="msg.isPrivate" class="private-label">私聊</div>
        </div>
      </div>
      
      <div class="chat-input">
        <el-input
          v-model="message"
          :placeholder="privateTarget ? `发送给 ${privateTargetName}...` : '输入消息并按回车发送...'"
          @keyup.enter="sendMessage"
          :disabled="!isConnected"
        >
          <template #prepend>
            <el-upload
              action="#"
              :show-file-list="false"
              :before-upload="beforeUpload"
              :http-request="handleUpload"
            >
              <el-button :disabled="!isConnected">
                <el-icon><Upload /></el-icon>
              </el-button>
            </el-upload>
          </template>
          <template #append>
            <el-button 
              @click="sendMessage" 
              :disabled="!message.trim() || !isConnected"
              type="primary"
            >
              发送
            </el-button>
          </template>
        </el-input>
      </div>
      
      <div class="connection-status">
        <span :class="['status-dot', isConnected ? 'connected' : 'disconnected']"></span>
        {{ isConnected ? '已连接' : '未连接' }}
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, nextTick, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Loading, Upload, Document, User } from '@element-plus/icons-vue'
import request from '../utils/request'

const router = useRouter()
const message = ref('')
const messages = ref([])
const socket = ref(null)
const isConnected = ref(false)
const chatBox = ref(null)
const loadingHistory = ref(false)
const onlineUsers = ref([])
const privateTarget = ref(0) // 0表示群聊，>0表示私聊目标用户ID
const privateTargetName = ref('')
const currentUserID = ref(0)
const onlineUsersRefreshInterval = ref(null)
const defaultAvatar = 'https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png'

// 添加分页相关状态变量
const currentPage = ref(1)
const hasMoreMessages = ref(true)
const isLoadingMore = ref(false)

// 从 localStorage 获取用户名
const username = computed(() => localStorage.getItem('username') || '未知用户')

// 获取完整的文件URL
const getFullFileUrl = (fileUrl) => {
  if (!fileUrl) return ''
  if (fileUrl.startsWith('http')) return fileUrl
  return `http://localhost:8080${fileUrl}`
}

// 获取完整头像URL
const getFullAvatarUrl = (avatar) => {
  if (!avatar) return defaultAvatar
  if (avatar.startsWith('http')) return avatar
  return `http://localhost:8080${avatar}`
}

// 获取状态文本
const getStatusText = (status) => {
  const statusMap = {
    'online': '在线',
    'busy': '忙碌',
    'away': '离开',
    'offline': '离线'
  }
  return statusMap[status] || '未知'
}

// 获取历史消息
const fetchHistoryMessages = async (loadMore = false) => {
  if (loadMore) {
    isLoadingMore.value = true
    currentPage.value += 1
  } else {
    loadingHistory.value = true
    currentPage.value = 1
    hasMoreMessages.value = true
  }
  
  try {
    const response = await request.get(`/messages?page=${currentPage.value}&pageSize=50`)
    
    const newMessages = response.messages.map(msg => ({
      id: msg.id,
      content: msg.content,
      sender: msg.username,
      timestamp: new Date(msg.created_at),
      time: formatTime(msg.created_at),
      isOwn: msg.username === username.value,
      isPrivate: false,
      isSystem: false,
      messageType: msg.message_type || 'text',
      fileUrl: msg.file_url,
      fileName: msg.file_name,
      fileSize: msg.file_size
    }))
    
    if (loadMore) {
      // 将新消息添加到列表开头
      messages.value = [...newMessages, ...messages.value]
    } else {
      messages.value = newMessages
      scrollToBottom()
    }
    
    // 检查是否还有更多消息
    hasMoreMessages.value = currentPage.value < response.pagination.totalPages
  } catch (error) {
    console.error('获取历史消息失败:', error)
    ElMessage.error('获取历史消息失败')
  } finally {
    loadingHistory.value = false
    isLoadingMore.value = false
  }
}

// 添加滚动监听，实现无限滚动
const handleScroll = () => {
  if (!chatBox.value || isLoadingMore.value || !hasMoreMessages.value) return
  
  const scrollTop = chatBox.value.scrollTop
  if (scrollTop < 100) {
    fetchHistoryMessages(true)
  }
}

// 获取在线用户列表
const fetchOnlineUsers = async () => {
  try {
    const response = await request.get('/online-users')
    if (response.success) {
      onlineUsers.value = response.data
    } else {
      onlineUsers.value = response
    }
  } catch (error) {
    console.error('获取在线用户失败:', error)
  }
}

// 文件上传前的验证
const beforeUpload = (file) => {
  const isLt10M = file.size / 1024 / 1024 < 10
  if (!isLt10M) {
    ElMessage.error('文件大小不能超过10MB!')
    return false
  }
  return true
}

// 处理文件上传
const handleUpload = async (options) => {
  const formData = new FormData()
  formData.append('file', options.file)
  
  try {
    const token = localStorage.getItem('token')
    const response = await request.post('/upload', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
        'Authorization': `Bearer ${token}`
      }
    })
    
    if (response.success) {
      // 发送文件消息
      sendFileMessage(response.file_url, response.file_name, response.file_size)
    } else {
      ElMessage.error('文件上传失败')
    }
  } catch (error) {
    console.error('文件上传错误:', error)
    ElMessage.error('文件上传失败')
  }
}

// 发送文件消息
const sendFileMessage = (fileUrl, fileName, fileSize) => {
  if (!isConnected.value) return
  
  // 确定消息类型
  const messageType = fileName.match(/\.(jpg|jpeg|png|gif|bmp|webp)$/i) ? 'image' : 'file'
  
  if (socket.value && socket.value.readyState === WebSocket.OPEN) {
    const messageData = {
      type: 'message',
      content: fileName,
      messageType: messageType,
      fileUrl: fileUrl,
      fileName: fileName,
      fileSize: fileSize,
      target: privateTarget.value
    }
    
    socket.value.send(JSON.stringify(messageData))
    ElMessage.success('文件发送成功')
  } else {
    ElMessage.warning('连接未就绪，请稍后再试')
  }
}

// 格式化文件大小
const formatFileSize = (bytes) => {
  if (bytes === 0) return '0 Bytes'
  const k = 1024
  const sizes = ['Bytes', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// 开始私聊
const startPrivateChat = (user) => {
  if (user.id === currentUserID.value) return // 不能和自己私聊
  
  privateTarget.value = user.id
  privateTargetName.value = user.username
  ElMessage.info(`开始与 ${user.username} 私聊`)
}

// 退出私聊
const exitPrivateChat = () => {
  privateTarget.value = 0
  privateTargetName.value = ''
  ElMessage.info('已返回群聊')
}

// 格式化存储时间
const formatTime = (timeString) => {
  const date = new Date(timeString)
  return date.toLocaleTimeString()
}

// 格式化显示时间（相对时间或简洁格式）
const formatDisplayTime = (timestamp) => {
  const now = new Date()
  const msgTime = new Date(timestamp)
  const diff = now - msgTime
  const diffMins = Math.floor(diff / 60000)
  const diffHours = Math.floor(diff / 3600000)
  const diffDays = Math.floor(diff / 86400000)
  
  if (diffMins < 1) return '刚刚'
  if (diffMins < 60) return `${diffMins}分钟前`
  if (diffHours < 24) return `${diffHours}小时前`
  if (diffDays < 7) return `${diffDays}天前`
  
  return msgTime.toLocaleDateString()
}

// 初始化 WebSocket 连接
const initWebSocket = () => {
  const token = localStorage.getItem('token')
  if (!token) {
    ElMessage.error('请先登录')
    router.push('/')
    return
  }

  try {
    // 建立 WebSocket 连接
    socket.value = new WebSocket('ws://localhost:8080/ws')
    
    socket.value.onopen = () => {
      console.log('WebSocket 连接已建立')
      // 连接建立后发送认证消息
      socket.value.send(JSON.stringify({
        type: 'auth',
        token: token
      }))
      isConnected.value = true
      ElMessage.success('连接成功')
    }
    
    socket.value.onmessage = (event) => {
      console.log('收到消息:', event.data)
      try {
        // 尝试解析 JSON 消息
        const messageData = JSON.parse(event.data)
        
        // 处理用户上下线消息
        if (messageData.type === 'user_joined' || messageData.type === 'user_left') {
          fetchOnlineUsers() // 刷新在线用户列表
          // 显示系统消息
          const newMessage = {
            content: messageData.content,
            sender: '系统',
            timestamp: new Date(),
            time: formatTime(new Date()),
            isOwn: false,
            isPrivate: false,
            isSystem: true
          }
          messages.value.push(newMessage)
          scrollToBottom()
          return
        }
        
        // 处理普通消息
        if (messageData.type === 'message') {
          const newMessage = {
            content: messageData.content,
            sender: messageData.username,
            timestamp: new Date(messageData.created_at || new Date()),
            time: formatTime(new Date(messageData.created_at || new Date())),
            isOwn: messageData.user_id === currentUserID.value,
            isPrivate: messageData.target > 0 && messageData.target !== currentUserID.value,
            isSystem: false,
            messageType: messageData.message_type || 'text',
            fileUrl: messageData.file_url,
            fileName: messageData.file_name,
            fileSize: messageData.file_size
          }
          
          // 如果是私聊消息，只有相关用户能看到
          if (!messageData.target || 
              messageData.target === currentUserID.value || 
              messageData.user_id === currentUserID.value) {
            messages.value.push(newMessage)
            scrollToBottom()
          }
        }
      } catch (error) {
        // 处理非JSON消息
        console.error('消息解析错误:', error)
        const newMessage = {
          content: event.data,
          sender: '系统',
          timestamp: new Date(),
          time: formatTime(new Date()),
          isOwn: false,
          isPrivate: false,
          isSystem: true
        }
        messages.value.push(newMessage)
        scrollToBottom()
      }
    }
    
    socket.value.onerror = (error) => {
      console.error('WebSocket 错误:', error)
      ElMessage.error('连接错误')
      isConnected.value = false
    }
    
    socket.value.onclose = () => {
      console.log('WebSocket 连接关闭')
      isConnected.value = false
    }
  } catch (error) {
    console.error('WebSocket 初始化错误:', error)
    ElMessage.error('连接失败: ' + error.message)
  }
}

// 发送消息
const sendMessage = () => {
  if (!message.value.trim() || !isConnected.value) return
  
  if (socket.value && socket.value.readyState === WebSocket.OPEN) {
    // 构建消息对象
    const messageData = {
      type: 'message',
      content: message.value,
      target: privateTarget.value // 添加目标字段
    }
    
    socket.value.send(JSON.stringify(messageData))
    message.value = ''
  } else {
    ElMessage.warning('连接未就绪，请稍后再试')
  }
}

// 滚动到底部
const scrollToBottom = () => {
  nextTick(() => {
    if (chatBox.value) {
      chatBox.value.scrollTop = chatBox.value.scrollHeight
    }
  })
}

// 跳转到个人资料页面
const goToProfile = () => {
  router.push('/profile')
}

// 退出登录
const logout = () => {
  localStorage.removeItem('token')
  localStorage.removeItem('username')
  if (socket.value) {
    socket.value.close()
  }
  router.push('/')
}

// 生命周期钩子
onMounted(() => {
  fetchHistoryMessages()
  initWebSocket()
  
  // 从token中解析用户ID（这里需要根据你的JWT结构进行调整）
  const token = localStorage.getItem('token')
  if (token) {
    try {
      // 简单解析JWT获取用户ID（实际应用中应该使用更安全的方式）
      const payload = JSON.parse(atob(token.split('.')[1]))
      currentUserID.value = payload.userID || 0
    } catch (error) {
      console.error('解析用户ID失败:', error)
    }
  }
  
  // 定期刷新在线用户列表
  onlineUsersRefreshInterval.value = setInterval(fetchOnlineUsers, 5000)
})

onUnmounted(() => {
  if (onlineUsersRefreshInterval.value) {
    clearInterval(onlineUsersRefreshInterval.value)
  }
  if (socket.value) {
    socket.value.close()
  }
  if (chatBox.value) {
    chatBox.value.removeEventListener('scroll', handleScroll)
  }
})
</script>

<style scoped>
.chat-container {
  display: flex;
  height: 100vh;
  background-color: #f5f5f5;
}

.online-users-sidebar {
  width: 250px;
  background-color: white;
  border-right: 1px solid #e0e0e0;
  padding: 20px;
  overflow-y: auto;
}

.online-users-sidebar h3 {
  margin-top: 0;
  margin-bottom: 15px;
  color: #333;
}

.user-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.user-item {
  display: flex;
  align-items: center;
  padding: 12px;
  border-radius: 8px;
  cursor: pointer;
  transition: background-color 0.2s;
  margin-bottom: 8px;
}

.user-item:hover {
  background-color: #f0f0f0;
}

.user-item.active {
  background-color: #e3f2fd;
  font-weight: bold;
}

.user-item.current-user {
  background-color: #f5f5f5;
  cursor: default;
}

.user-avatar {
  position: relative;
  margin-right: 12px;
}

.user-avatar img {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
}

.user-status {
  position: absolute;
  bottom: 2px;
  right: 2px;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  border: 2px solid #fff;
}

.user-status.online {
  background-color: #52c41a;
}

.user-status.busy {
  background-color: #ff4d4f;
}

.user-status.away {
  background-color: #faad14;
}

.user-status.offline {
  background-color: #d9d9d9;
}

.user-info {
  flex: 1;
  min-width: 0;
}

.user-name {
  font-weight: bold;
  color: #333;
  margin-bottom: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.user-bio {
  font-size: 11px;
  color: #666;
  margin-bottom: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.user-status-text {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.status-text {
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 10px;
  color: #fff;
}

.status-text.online {
  background-color: #52c41a;
}

.status-text.busy {
  background-color: #ff4d4f;
}

.status-text.away {
  background-color: #faad14;
}

.status-text.offline {
  background-color: #d9d9d9;
  color: #666;
}

.you-label {
  font-size: 11px;
  color: #999;
}

.main-chat-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 20px;
}

.chat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 10px;
  border-bottom: 1px solid #ddd;
}

.header-actions {
  display: flex;
  gap: 10px;
}

.chat-box {
  flex: 1;
  overflow-y: auto;
  background-color: white;
  border-radius: 8px;
  padding: 15px;
  margin-bottom: 15px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
}

/* 添加加载更多样式 */
.loading-more {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 10px;
  color: #666;
  background-color: #f9f9f9;
  border-bottom: 1px solid #eee;
}

.loading-indicator {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 10px;
  color: #666;
}

.message {
  margin-bottom: 15px;
  padding: 10px 15px;
  border-radius: 18px;
  background-color: #e9e9e9;
  max-width: 80%;
  word-wrap: break-word;
  position: relative;
}

.own-message {
  background-color: #1890ff;
  color: white;
  margin-left: auto;
}

.private-message {
  border: 2px solid #ffa500;
}

.system-message {
  background-color: #f0f0f0;
  font-style: italic;
  margin: 0 auto;
  text-align: center;
  max-width: 95%;
}

.message-meta {
  display: flex;
  justify-content: space-between;
  margin-bottom: 5px;
  font-size: 12px;
}

.message-sender {
  font-weight: bold;
}

.own-message .message-sender {
  color: rgba(255, 255, 255, 0.8);
}

.message-time {
  color: #666;
}

.own-message .message-time {
  color: rgba(255, 255, 255, 0.8);
}

.message-content {
  word-wrap: break-word;
}

/* 图片消息样式 */
.image-message {
  max-width: 300px;
}

.image-message img {
  max-width: 100%;
  border-radius: 4px;
}

.image-info {
  font-size: 12px;
  color: #666;
  margin-top: 5px;
}

/* 文件消息样式 */
.file-message {
  display: flex;
  align-items: center;
  padding: 10px;
  background-color: #f9f9f9;
  border-radius: 6px;
  max-width: 300px;
}

.file-icon {
  margin-right: 10px;
  font-size: 24px;
  color: #409EFF;
}

.file-info {
  flex: 1;
}

.file-name {
  display: block;
  font-weight: bold;
  color: #409EFF;
  text-decoration: none;
  margin-bottom: 5px;
}

.file-name:hover {
  text-decoration: underline;
}

.file-size {
  font-size: 12px;
  color: #666;
}

.private-label {
  position: absolute;
  top: -8px;
  right: 10px;
  background-color: #ffa500;
  color: white;
  font-size: 10px;
  padding: 2px 6px;
  border-radius: 10px;
}

.chat-input {
  margin-bottom: 15px;
}

.connection-status {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  color: #666;
}

.status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  margin-right: 8px;
}

.status-dot.connected {
  background-color: #52c41a;
}

.status-dot.disconnected {
  background-color: #f5222d;
}
</style>