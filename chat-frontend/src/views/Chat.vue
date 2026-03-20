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
          <el-button @click="goToGroups" type="info" size="small">
            <el-icon><ChatRound /></el-icon>
            群组
          </el-button>
          <el-button @click="goToProfile" type="primary" size="small">
            <el-icon><User /></el-icon>
            个人资料
          </el-button>
          <el-button @click="logout" type="danger" size="small">退出</el-button>
        </div>
      </div>
      
      <div class="chat-box" ref="chatBox" @scroll="handleScroll">
        <div v-if="isLoadingMore" class="loading-more">
          <el-icon class="is-loading"><Loading /></el-icon>
          <span>加载更多消息...</span>
        </div>
        <div v-if="loadingHistory" class="loading-indicator">
          <el-icon class="is-loading"><Loading /></el-icon>
          <span>加载历史消息中...</span>
        </div>
        
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
          <div v-if="msg.messageType === 'text'" class="message-content">{{ msg.content }}</div>
          <div v-else-if="msg.messageType === 'image'" class="image-message">
            <img :src="getFullFileUrl(msg.fileUrl)" :alt="msg.fileName" @load="scrollToBottom" />
            <div class="image-info">{{ msg.fileName }}</div>
          </div>
          <div v-else-if="msg.messageType === 'file'" class="file-message">
            <div class="file-icon"><el-icon><Document /></el-icon></div>
            <div class="file-info">
              <a :href="getFullFileUrl(msg.fileUrl)" target="_blank" class="file-name">{{ msg.fileName }}</a>
              <div class="file-size">{{ formatFileSize(msg.fileSize) }}</div>
            </div>
          </div>
          <div v-if="msg.isOwn && !msg.isSystem" class="message-read-status">
            <span v-if="msg.readCount > 0" class="read-status read">✓✓ 已读</span>
            <span v-else class="read-status unread">✓ 未读</span>
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
            <el-upload action="#" :show-file-list="false" :before-upload="beforeUpload" :http-request="handleUpload">
              <el-button :disabled="!isConnected"><el-icon><Upload /></el-icon></el-button>
            </el-upload>
          </template>
          <template #append>
            <el-button @click="sendMessage" :disabled="!message.trim() || !isConnected" type="primary">发送</el-button>
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
import { Loading, Upload, Document, User, ChatRound } from '@element-plus/icons-vue'
import request from '../utils/request'
import { getAuthToken, clearAuthToken } from '../utils/auth'
import { useUserStore } from '../stores/userStore'
import { useChatStore } from '../stores/chatStore'

const router = useRouter()
const userStore = useUserStore()
const chatStore = useChatStore()

const message = ref('')
const socket = ref(null)
const isConnected = ref(false)
const chatBox = ref(null)
const loadingHistory = ref(false)
const onlineUsers = ref([])
const privateTarget = ref(0)
const privateTargetName = ref('')
const onlineUsersRefreshInterval = ref(null)
const isLoadingMore = ref(false)
const readRequestedIds = new Set()
const defaultAvatar = 'https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png'

const currentUserID = computed(() => userStore.profile.id)
const messages = computed(() => chatStore.messages)

const getFullFileUrl = (fileUrl) => {
  if (!fileUrl) return ''
  if (fileUrl.startsWith('http')) return fileUrl
  return `http://localhost:8080${fileUrl}`
}
const getFullAvatarUrl = (avatar) => {
  if (!avatar) return defaultAvatar
  if (avatar.startsWith('http')) return avatar
  return `http://localhost:8080${avatar}`
}
const getStatusText = (status) => {
  const map = { online: '在线', busy: '忙碌', away: '离开', offline: '离线' }
  return map[status] || '未知'
}
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
const formatFileSize = (bytes) => {
  if (!bytes || bytes === 0) return '0 Bytes'
  const k = 1024
  const sizes = ['Bytes', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const scrollToBottom = () => {
  nextTick(() => { if (chatBox.value) chatBox.value.scrollTop = chatBox.value.scrollHeight })
}
const isScrolledToBottom = () => {
  if (!chatBox.value) return false
  const { scrollTop, scrollHeight, clientHeight } = chatBox.value
  return scrollHeight - scrollTop - clientHeight < 20
}

const markMessageAsRead = async (messageId) => {
  if (!messageId) return
  try {
    const res = await request.put(`/messages/${messageId}/read`)
    if (res.success) console.log(`✅ 消息已标记为已读: messageID=${messageId}`)
  } catch (error) {
    console.error('标记消息已读失败:', error)
  }
}
const markVisibleMessagesAsRead = () => {
  if (!isScrolledToBottom()) return
  chatStore.messages.forEach(msg => {
    if (!msg.isOwn && !msg.isSystem && msg.id && !readRequestedIds.has(msg.id)) {
      readRequestedIds.add(msg.id)
      markMessageAsRead(msg.id)
    }
  })
}

const fetchHistoryMessages = async (loadMore = false) => {
  if (loadMore) {
    if (!chatStore.hasMore || isLoadingMore.value) return
    isLoadingMore.value = true
  } else {
    loadingHistory.value = true
  }
  try {
    await chatStore.fetchMessages(loadMore, currentUserID.value)
    if (!loadMore) {
      scrollToBottom()
      nextTick(() => markVisibleMessagesAsRead())
    }
  } catch (error) {
    console.error('获取历史消息失败:', error)
  } finally {
    loadingHistory.value = false
    isLoadingMore.value = false
  }
}

const handleScroll = () => {
  if (!isLoadingMore.value && chatStore.hasMore && chatBox.value?.scrollTop < 100) {
    fetchHistoryMessages(true)
  }
  markVisibleMessagesAsRead()
}

const fetchOnlineUsers = async () => {
  try {
    const res = await request.get('/online-users')
    onlineUsers.value = res.success ? res.data : res
  } catch (error) {
    console.error('获取在线用户失败:', error)
  }
}

const startPrivateChat = (user) => {
  if (user.id === currentUserID.value) return
  privateTarget.value = user.id
  privateTargetName.value = user.username
  ElMessage.info(`开始与 ${user.username} 私聊`)
}
const exitPrivateChat = () => {
  privateTarget.value = 0
  privateTargetName.value = ''
  ElMessage.info('已返回群聊')
}

const beforeUpload = (file) => {
  if (file.size / 1024 / 1024 >= 10) { ElMessage.error('文件大小不能超过10MB!'); return false }
  return true
}
const handleUpload = async (options) => {
  const formData = new FormData()
  formData.append('file', options.file)
  try {
    const token = await getAuthToken()
    const res = await request.post('/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data', Authorization: `Bearer ${token}` }
    })
    if (res.success) {
      const msgType = res.file_name.match(/\.(jpg|jpeg|png|gif|bmp|webp)$/i) ? 'image' : 'file'
      if (socket.value && socket.value.readyState === WebSocket.OPEN) {
        socket.value.send(JSON.stringify({
          type: 'message', content: res.file_name, messageType: msgType,
          fileUrl: res.file_url, fileName: res.file_name, fileSize: res.file_size,
          target: privateTarget.value
        }))
        ElMessage.success('文件发送成功')
      }
    } else {
      ElMessage.error('文件上传失败')
    }
  } catch (error) {
    console.error('文件上传错误:', error)
    ElMessage.error('文件上传失败')
  }
}

const sendMessage = () => {
  if (!message.value.trim() || !isConnected.value) return
  if (socket.value && socket.value.readyState === WebSocket.OPEN) {
    socket.value.send(JSON.stringify({ type: 'message', content: message.value, target: privateTarget.value }))
    message.value = ''
  } else {
    ElMessage.warning('连接未就绪，请稍后再试')
  }
}

const goToGroups = () => router.push('/groups')
const goToProfile = () => router.push('/profile')
const logout = () => {
  clearAuthToken()
  userStore.logout()
  if (socket.value) socket.value.close()
  router.push('/')
}

const initWebSocket = async () => {
  const token = await getAuthToken()
  if (!token) { ElMessage.error('请先登录'); router.push('/'); return }
  socket.value = new WebSocket('ws://localhost:8080/ws')
  socket.value.onopen = () => {
    socket.value.send(JSON.stringify({ type: 'auth', token }))
    isConnected.value = true
    ElMessage.success('连接成功')
  }
  socket.value.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data)
      if (data.type === 'user_joined' || data.type === 'user_left') {
        fetchOnlineUsers()
        chatStore.addMessage(
          { content: data.content, username: '系统', created_at: new Date().toISOString(), message_type: 'text', user_id: -1 },
          currentUserID.value
        )
        const msgs = chatStore.messages
        if (msgs.length > 0) msgs[msgs.length - 1].isSystem = true
        scrollToBottom()
        return
      }
      if (data.type === 'read_receipt') {
        chatStore.updateReadStatus(data.id, data.read_count)
        return
      }
      if (data.type === 'message') {
        if (data.group_id && data.group_id > 0) return
        chatStore.addMessage(data, currentUserID.value)
        scrollToBottom()
        if (data.user_id !== currentUserID.value && data.id) {
          nextTick(() => {
            if (isScrolledToBottom() && !readRequestedIds.has(data.id)) {
              readRequestedIds.add(data.id)
              markMessageAsRead(data.id)
            }
          })
        }
      }
    } catch (error) {
      console.error('消息解析错误:', error)
      chatStore.addMessage(
        { content: event.data, username: '系统', created_at: new Date().toISOString(), message_type: 'text', user_id: -1 },
        currentUserID.value
      )
      const msgs = chatStore.messages
      if (msgs.length > 0) msgs[msgs.length - 1].isSystem = true
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
}

onMounted(async () => {
  if (!userStore.profile.id) await userStore.fetchProfile()
  if (chatStore.messages.length === 0) {
    await fetchHistoryMessages()
  } else {
    scrollToBottom()
  }
  await initWebSocket()
  fetchOnlineUsers()
  onlineUsersRefreshInterval.value = setInterval(fetchOnlineUsers, 5000)
})

onUnmounted(() => {
  if (onlineUsersRefreshInterval.value) clearInterval(onlineUsersRefreshInterval.value)
  if (socket.value) socket.value.close()
})
</script>

<style scoped>
.chat-container { display: flex; height: 100vh; background-color: #f5f5f5; }
.online-users-sidebar {
  width: 250px; background-color: white;
  border-right: 1px solid #e0e0e0; padding: 20px; overflow-y: auto;
}
.online-users-sidebar h3 { margin-top: 0; margin-bottom: 15px; color: #333; }
.user-list { display: flex; flex-direction: column; gap: 8px; }
.user-item {
  display: flex; align-items: center; padding: 12px;
  border-radius: 8px; cursor: pointer; transition: background-color 0.2s; margin-bottom: 8px;
}
.user-item:hover { background-color: #f0f0f0; }
.user-item.active { background-color: #e3f2fd; font-weight: bold; }
.user-item.current-user { background-color: #f5f5f5; cursor: default; }
.user-avatar { position: relative; margin-right: 12px; }
.user-avatar img { width: 40px; height: 40px; border-radius: 50%; object-fit: cover; }
.user-status {
  position: absolute; bottom: 2px; right: 2px;
  width: 12px; height: 12px; border-radius: 50%; border: 2px solid #fff;
}
.user-status.online { background-color: #52c41a; }
.user-status.busy { background-color: #ff4d4f; }
.user-status.away { background-color: #faad14; }
.user-status.offline { background-color: #d9d9d9; }
.user-info { flex: 1; min-width: 0; }
.user-name { font-weight: bold; color: #333; margin-bottom: 2px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.user-bio { font-size: 11px; color: #666; margin-bottom: 4px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.user-status-text { display: flex; align-items: center; justify-content: space-between; }
.status-text { font-size: 11px; padding: 2px 6px; border-radius: 10px; color: #fff; }
.status-text.online { background-color: #52c41a; }
.status-text.busy { background-color: #ff4d4f; }
.status-text.away { background-color: #faad14; }
.status-text.offline { background-color: #d9d9d9; color: #666; }
.you-label { font-size: 11px; color: #999; }
.main-chat-area { flex: 1; display: flex; flex-direction: column; padding: 20px; }
.chat-header {
  display: flex; justify-content: space-between; align-items: center;
  margin-bottom: 20px; padding-bottom: 10px; border-bottom: 1px solid #ddd;
}
.header-actions { display: flex; gap: 10px; }
.chat-box {
  flex: 1; overflow-y: auto; background-color: white;
  border-radius: 8px; padding: 15px; margin-bottom: 15px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
}
.loading-more, .loading-indicator {
  display: flex; align-items: center; justify-content: center; padding: 10px; color: #666;
}
.loading-more { background-color: #f9f9f9; border-bottom: 1px solid #eee; }
.message {
  margin-bottom: 15px; padding: 10px 15px; border-radius: 18px;
  background-color: #e9e9e9; max-width: 80%; word-wrap: break-word; position: relative;
}
.own-message { background-color: #1890ff; color: white; margin-left: auto; }
.private-message { border: 2px solid #ffa500; }
.system-message { background-color: #f0f0f0; font-style: italic; margin: 0 auto; text-align: center; max-width: 95%; }
.message-meta { display: flex; justify-content: space-between; margin-bottom: 5px; font-size: 12px; }
.message-sender { font-weight: bold; }
.own-message .message-sender { color: rgba(255,255,255,0.8); }
.message-time { color: #666; }
.own-message .message-time { color: rgba(255,255,255,0.8); }
.image-message { max-width: 300px; }
.image-message img { max-width: 100%; border-radius: 4px; }
.image-info { font-size: 12px; color: #666; margin-top: 5px; }
.file-message { display: flex; align-items: center; padding: 10px; background-color: #f9f9f9; border-radius: 6px; max-width: 300px; }
.file-icon { margin-right: 10px; font-size: 24px; color: #409EFF; }
.file-info { flex: 1; }
.file-name { display: block; font-weight: bold; color: #409EFF; text-decoration: none; margin-bottom: 5px; }
.file-name:hover { text-decoration: underline; }
.file-size { font-size: 12px; color: #666; }
.private-label {
  position: absolute; top: -8px; right: 10px;
  background-color: #ffa500; color: white; font-size: 10px; padding: 2px 6px; border-radius: 10px;
}
.chat-input { margin-bottom: 15px; }
.connection-status { display: flex; align-items: center; justify-content: center; font-size: 14px; color: #666; }
.status-dot { width: 10px; height: 10px; border-radius: 50%; margin-right: 8px; }
.status-dot.connected { background-color: #52c41a; }
.status-dot.disconnected { background-color: #f5222d; }
.message-read-status { margin-top: 5px; font-size: 12px; text-align: right; }
.read-status { display: inline-block; padding: 2px 6px; border-radius: 4px; font-size: 11px; }
.read-status.read { color: #52c41a; font-weight: 500; }
.read-status.unread { color: #999; }
</style>
