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
          <span class="user-status"></span>
          {{ user.username }}
          <span v-if="user.id === currentUserID" class="you-label">(我)</span>
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
        <el-button @click="logout" type="danger" size="small">退出</el-button>
      </div>
      
      <div class="chat-box" ref="chatBox">
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
          <div class="message-content">{{ msg.content }}</div>
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
import { Loading } from '@element-plus/icons-vue'
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

// 从 localStorage 获取用户名
const username = computed(() => localStorage.getItem('username') || '未知用户')

// 获取历史消息
const fetchHistoryMessages = async () => {
  loadingHistory.value = true
  try {
    const response = await request.get('/messages?limit=100')
    messages.value = response.map(msg => ({
      content: msg.content,
      sender: msg.username,
      timestamp: new Date(msg.created_at),
      time: formatTime(msg.created_at),
      isOwn: msg.username === username.value,
      isPrivate: false,
      isSystem: false
    }))
    scrollToBottom()
  } catch (error) {
    console.error('获取历史消息失败:', error)
    ElMessage.error('获取历史消息失败')
  } finally {
    loadingHistory.value = false
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
            isSystem: false
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
  padding: 8px 12px;
  border-radius: 6px;
  cursor: pointer;
  transition: background-color 0.2s;
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

.user-status {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: #4caf50;
  margin-right: 8px;
}

.you-label {
  margin-left: auto;
  font-size: 12px;
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

.chat-box {
  flex: 1;
  overflow-y: auto;
  background-color: white;
  border-radius: 8px;
  padding: 15px;
  margin-bottom: 15px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
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