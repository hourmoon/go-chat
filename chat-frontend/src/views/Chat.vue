<template>
  <div class="chat-page">
    <div class="chat-header">
      <h2>聊天室 - {{ username }}</h2>
      <el-button @click="logout" type="danger" size="small">退出</el-button>
    </div>
    
    <div class="chat-box" ref="chatBox">
      <div v-for="(msg, index) in messages" :key="index" class="message" :class="{ 'own-message': msg.isOwn }">
        <div class="message-sender">{{ msg.sender }}</div>
        <div class="message-content">{{ msg.content }}</div>
        <div class="message-time">{{ msg.time }}</div>
      </div>
    </div>
    
    <div class="chat-input">
      <el-input
        v-model="message"
        placeholder="输入消息..."
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

const router = useRouter()
const message = ref('')
const messages = ref([])
const socket = ref(null)
const isConnected = ref(false)
const chatBox = ref(null)

// 从 localStorage 获取用户名
const username = computed(() => localStorage.getItem('username') || '未知用户')

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
            sender: messageData.sender || '系统',
            time: new Date().toLocaleTimeString(),
            isOwn: false
          }
          messages.value.push(newMessage)
          scrollToBottom()
        } else if (messageData.type === 'system') {
          // 处理系统消息
          ElMessage.info(messageData.content)
        }
      }  catch (error) {
    console.error('WebSocket 初始化错误:', error)
    ElMessage.error('连接失败: ' + error.message)
  }
    }
    
    socket.value.onerror = (error) => {
      console.error('WebSocket 错误:', error)
      ElMessage.error('连接错误，3秒后重试...')
      isConnected.value = false
      
      // 3秒后重试
      setTimeout(() => {
        if (!isConnected.value) {
          initWebSocket()
        }
      }, 3000)
    }
    
    socket.value.onclose = (event) => {
      console.log('WebSocket 连接关闭:', event.code, event.reason)
      isConnected.value = false
      
      // 如果不是正常关闭，尝试重连
      if (event.code !== 1000) {
        ElMessage.warning('连接已断开，5秒后重试...')
        setTimeout(() => {
          if (!isConnected.value) {
            initWebSocket()
          }
        }, 5000)
      }
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
    
    // 添加到消息列表（自己的消息）
    messages.value.push({
      content: message.value,
      sender: username.value,
      time: new Date().toLocaleTimeString(),
      isOwn: true
    })
    
    message.value = ''
    scrollToBottom()
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
  initWebSocket()
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

.message {
  margin-bottom: 15px;
  padding: 10px;
  border-radius: 8px;
  background-color: #e9e9e9;
}

.own-message {
  background-color: #d1ecf1;
  text-align: right;
}

.message-sender {
  font-weight: bold;
  font-size: 12px;
  margin-bottom: 5px;
}

.message-content {
  margin-bottom: 5px;
}

.message-time {
  font-size: 11px;
  color: #666;
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