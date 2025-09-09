<template>
  <div class="chat-page">
    <div class="chat-header">
      <h2>聊天室 - {{ username }}</h2>
      <el-button @click="logout" type="danger" size="small">退出</el-button>
    </div>
    
    <div class="chat-box" ref="chatBox">
      <!-- 加载指示器 -->
      <div v-if="loadingHistory" class="loading-indicator">
        <el-icon class="is-loading"><Loading /></el-icon>
        <span>加载历史消息中...</span>
      </div>
      
      <!-- 消息列表 -->
      <div v-for="(msg, index) in messages" :key="index" class="message" :class="{ 'own-message': msg.isOwn }">
        <div class="message-meta">
          <span class="message-sender">{{ msg.sender }}</span>
          <span class="message-time">{{ formatDisplayTime(msg.timestamp) }}</span>
        </div>
        <div class="message-content">{{ msg.content }}</div>
      </div>
    </div>
    
    <div class="chat-input">
      <el-input
        v-model="message"
        placeholder="输入消息并按回车发送..."
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
const loadingHistory = ref(false) // 加载状态

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
      timestamp: new Date(msg.created_at), // 保存时间戳对象
      time: formatTime(msg.created_at),
      isOwn: msg.username === username.value
    }))
    scrollToBottom()
  } catch (error) {
    console.error('获取历史消息失败:', error)
    ElMessage.error('获取历史消息失败')
  } finally {
    loadingHistory.value = false
  }
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
        
        // 处理不同类型的消息
        if (messageData.type === 'message') {
          const newMessage = {
            content: messageData.content,
            sender: messageData.username,
            timestamp: new Date(), // 当前时间
            time: formatTime(new Date()),
            isOwn: messageData.username === username.value
          }
          messages.value.push(newMessage)
          scrollToBottom()
        } else if (messageData.type === 'system') {
          // 处理系统消息
          ElMessage.info(messageData.content)
        }
      } catch (error) {
        // 如果不是 JSON，当作普通文本处理
        const newMessage = {
          content: event.data,
          sender: '系统',
          timestamp: new Date(),
          time: formatTime(new Date()),
          isOwn: false
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
    socket.value.send(message.value)
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
  fetchHistoryMessages() // 获取历史消息
  initWebSocket()        // 初始化 WebSocket 连接
})

onUnmounted(() => {
  if (socket.value) {
    socket.value.close()
  }
})
</script>

<style scoped>
.chat-page {
  display: flex;
  flex-direction: column;
  height: 100vh;
  max-width: 800px;
  margin: 0 auto;
  padding: 20px;
  background-color: #f5f5f5;
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
}

.own-message {
  background-color: #1890ff;
  color: white;
  margin-left: auto;
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