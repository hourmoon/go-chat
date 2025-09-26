<template>
  <div class="group-chat-container">
    <!-- 群成员侧边栏 -->
    <div class="group-members-sidebar">
      <div class="group-header">
        <h3>{{ currentGroup?.Name || '群聊' }}</h3>
        <div class="group-actions">
          <el-button @click="goToGroups" size="small" type="primary">
            <el-icon><Back /></el-icon>
            返回群组列表
          </el-button>
        </div>
      </div>
      
      <div class="group-members">
        <el-tabs v-model="activeTab" class="member-tabs">
          <el-tab-pane label="在线成员" name="online">
            <div class="online-members">
              <h4>在线成员 ({{ onlineMembers.length }})</h4>
              <div class="member-list">
                <div 
                  v-for="member in onlineMembers" 
                  :key="member.id"
                  class="member-item"
                  :class="{ 'current-user': member.id === currentUserID }"
                >
                  <div class="member-avatar">
                    <img 
                      :src="member.avatar ? getFullAvatarUrl(member.avatar) : defaultAvatar" 
                      :alt="member.username"
                    />
                    <span class="member-status online"></span>
                  </div>
                  <div class="member-info">
                    <div class="member-name">{{ member.username }}</div>
                    <div class="member-status-text">在线</div>
                    <span v-if="member.id === currentUserID" class="you-label">(我)</span>
                  </div>
                </div>
              </div>
            </div>
          </el-tab-pane>
          
          <el-tab-pane label="成员管理" name="manage">
            <GroupMemberList
              :group-id="currentGroupId"
              :current-user-role="currentUserRole"
              :online-members="onlineMembers"
              @member-added="handleMemberAdded"
              @member-removed="handleMemberRemoved"
              @refresh-members="refreshOnlineMembers"
            />
          </el-tab-pane>
        </el-tabs>
      </div>
    </div>
    
    <!-- 主群聊区域 -->
    <div class="main-chat-area">
      <div class="chat-header">
        <h2>
          <el-icon><ChatRound /></el-icon>
          {{ currentGroup?.Name || '群聊' }}
          <span class="group-label">群聊</span>
        </h2>
        <div class="header-actions">
          <el-button @click="refreshOnlineMembers" size="small">
            <el-icon><Refresh /></el-icon>
            刷新成员
          </el-button>
          
          <!-- 根据用户角色显示不同操作 -->
          <el-button 
            v-if="isOwner" 
            @click="handleDeleteGroup" 
            type="danger" 
            size="small"
          >
            解散群组
          </el-button>
          <el-button 
            v-else 
            @click="handleLeaveGroup" 
            type="warning" 
            size="small"
          >
            退出群组
          </el-button>
          
          <el-button @click="goToProfile" type="primary" size="small">
            <el-icon><User /></el-icon>
            个人资料
          </el-button>
          <el-button @click="logout" type="danger" size="small">退出登录</el-button>
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
            'system-message': msg.isSystem,
            'group-message': true
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
        </div>
      </div>
      
      <div class="chat-input">
        <el-input
          v-model="message"
          placeholder="输入群聊消息并按回车发送..."
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
import { ref, onMounted, onUnmounted, nextTick, computed, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Loading, Upload, Document, User, ChatRound, Back, Refresh } from '@element-plus/icons-vue'
import request from '../utils/request'
import groupStore from '../stores/groupStore'
import * as groupApi from '../utils/groupApi'
import GroupMemberList from '../components/GroupMemberList.vue'
import { getAuthToken, getUsername, clearAuthToken, getCurrentUserId } from '../utils/auth'

const router = useRouter()
const route = useRoute()
const message = ref('')
const messages = ref([])
const socket = ref(null)
const isConnected = ref(false)
const chatBox = ref(null)
const loadingHistory = ref(false)
const onlineMembers = ref([])
const currentUserID = ref(0)
const currentGroupId = ref(null)
const activeTab = ref('online')
const currentUserRole = ref('member')
const defaultAvatar = 'https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png'

// 添加分页相关状态变量
const currentPage = ref(1)
const hasMoreMessages = ref(true)
const isLoadingMore = ref(false)

// 从路由参数获取群组ID
currentGroupId.value = parseInt(route.params.groupId)

// 从groupStore获取当前群组信息
const currentGroup = computed(() => groupStore.state.currentGroup)

// 从 auth.js 获取用户名（sessionStorage 优先，localStorage 兼容回退）
const username = computed(() => getUsername() || '未知用户')

// 判断当前用户是否为群主（兜底判断）
const isOwner = computed(() => {
  // 方式1：通过角色判断
  if (currentUserRole.value === 'owner') {
    return true
  }
  
  // 方式2：通过群组信息兜底判断（兼容大小写字段名）
  const groupOwnerID = currentGroup.value?.OwnerID || currentGroup.value?.owner_id
  if (groupOwnerID && currentUserID.value && groupOwnerID === currentUserID.value) {
    return true
  }
  
  return false
})

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

// 获取群聊历史消息
const fetchHistoryMessages = async (loadMore = false) => {
  if (!currentGroupId.value) return
  
  if (loadMore) {
    isLoadingMore.value = true
    currentPage.value += 1
  } else {
    loadingHistory.value = true
    currentPage.value = 1
    hasMoreMessages.value = true
  }
  
  try {
    const response = await groupApi.getGroupMessages(currentGroupId.value, currentPage.value, 50)
    
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
      fileSize: msg.file_size,
      groupId: msg.group_id
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
    console.error('获取群聊历史消息失败:', error)
    ElMessage.error('获取群聊历史消息失败')
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

// 获取群在线成员列表
const fetchOnlineMembers = async () => {
  if (!currentGroupId.value) return
  
  try {
    const response = await groupApi.getOnlineMembers(currentGroupId.value)
    onlineMembers.value = response.onlineMembers || []
  } catch (error) {
    console.error('获取群在线成员失败:', error)
    ElMessage.error('获取群在线成员失败')
  }
}

// 刷新在线成员
const refreshOnlineMembers = () => {
  fetchOnlineMembers()
  ElMessage.success('已刷新在线成员列表')
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
    const token = await getAuthToken()
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
      group_id: currentGroupId.value
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
const initWebSocket = async () => {
  const token = await getAuthToken()
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
        
        // 处理群组解散消息
        if (messageData.type === 'group_dissolved') {
          // 只处理当前群组的解散消息
          if (messageData.group_id === currentGroupId.value) {
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
            
            // 延迟跳转，让用户看到解散消息
            setTimeout(() => {
              ElMessage.warning('群组已解散，即将返回群组列表')
              // 刷新群组列表
              groupStore.actions.fetchUserGroups()
              // 跳转到群组列表页
              router.push('/groups')
            }, 2000)
          }
          return
        }
        
        // 处理群成员变动消息
        if (messageData.type === 'group_member_joined' || messageData.type === 'group_member_left') {
          // 只处理当前群组的成员变动
          if (messageData.group_id === currentGroupId.value) {
            fetchOnlineMembers() // 刷新在线成员列表
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
          }
          return
        }
        
        // 处理用户上下线消息（全局事件）
        if (messageData.type === 'user_joined' || messageData.type === 'user_left') {
          fetchOnlineMembers() // 刷新在线成员列表
          return
        }
        
        // 处理普通消息
        if (messageData.type === 'message') {
          // 只显示当前群组的消息
          if (messageData.group_id === currentGroupId.value) {
            const newMessage = {
              content: messageData.content,
              sender: messageData.username,
              timestamp: new Date(messageData.created_at || new Date()),
              time: formatTime(new Date(messageData.created_at || new Date())),
              isOwn: messageData.user_id === currentUserID.value,
              isPrivate: false,
              isSystem: false,
              messageType: messageData.message_type || 'text',
              fileUrl: messageData.file_url,
              fileName: messageData.file_name,
              fileSize: messageData.file_size,
              groupId: messageData.group_id
            }
            
            messages.value.push(newMessage)
            scrollToBottom()
          } else if (messageData.group_id && messageData.group_id !== currentGroupId.value) {
            // 将其他群组的消息添加到groupStore中
            groupStore.actions.addMessageToGroup(messageData.group_id, messageData)
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
      group_id: currentGroupId.value
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

// 跳转到群组列表
const goToGroups = () => {
  router.push('/groups')
}

// 跳转到个人资料页面
const goToProfile = () => {
  router.push('/profile')
}

// 退出登录
const logout = () => {
  clearAuthToken()
  if (socket.value) {
    socket.value.close()
  }
  router.push('/')
}

// 监听路由变化
watch(() => route.params.groupId, async (newGroupId) => {
  if (newGroupId) {
    currentGroupId.value = parseInt(newGroupId)
    // 重新加载群组信息和消息
    await groupStore.actions.selectGroup(currentGroupId.value)
    // 刷新用户角色（切换群聊后需要重新获取角色）
    await fetchCurrentUserRole()
    fetchHistoryMessages()
    fetchOnlineMembers()
  }
})

// 处理成员添加事件
const handleMemberAdded = () => {
  ElMessage.success('成员添加成功')
  fetchOnlineMembers()
}

// 处理成员移除事件
const handleMemberRemoved = () => {
  ElMessage.success('成员移除成功')
  fetchOnlineMembers()
}

// 获取当前用户在群组中的角色
const fetchCurrentUserRole = async () => {
  try {
    const response = await groupApi.getGroupMembers(currentGroupId.value)
    // 兼容后端大小写字段名
    const currentMember = response.members.find(member => 
      (member.UserID || member.user_id) === currentUserID.value
    )
    if (currentMember) {
      // 兼容后端大小写字段名
      currentUserRole.value = currentMember.Role || currentMember.role
    }
  } catch (error) {
    console.error('获取用户角色失败:', error)
  }
}

// 处理退出群组
const handleLeaveGroup = () => {
  ElMessageBox.confirm(
    '确定要退出该群组吗？退出后将无法接收群消息。',
    '退出群组',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    }
  ).then(async () => {
    try {
      await groupApi.leaveGroup(currentGroupId.value)
      ElMessage.success('已退出群组')
      // 刷新群组列表
      await groupStore.actions.fetchUserGroups()
      // 跳转到群组列表页
      router.push('/groups')
    } catch (error) {
      ElMessage.error(error?.error || '退出群组失败')
    }
  }).catch(() => {
    // 用户取消操作
  })
}

// 处理解散群组
const handleDeleteGroup = () => {
  ElMessageBox.confirm(
    '确定要解散该群组吗？解散后所有成员都将离开群组，此操作不可恢复。',
    '解散群组',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'error',
    }
  ).then(async () => {
    try {
      await groupApi.deleteGroup(currentGroupId.value)
      ElMessage.success('群组已解散')
      // 刷新群组列表
      await groupStore.actions.fetchUserGroups()
      // 跳转到群组列表页
      router.push('/groups')
    } catch (error) {
      ElMessage.error(error?.error || '解散群组失败')
    }
  }).catch(() => {
    // 用户取消操作
  })
}

// 生命周期钩子
onMounted(async () => {
  // 步骤1：从token中解析用户ID（使用 auth.js）
  const userId = await getCurrentUserId()
  if (userId) {
    currentUserID.value = userId
  }
  
  // 步骤2：初始化群组状态
  await groupStore.actions.selectGroup(currentGroupId.value)
  
  // 步骤3：获取用户角色（必须在群组状态初始化后）
  await fetchCurrentUserRole()
  
  // 步骤4：初始化 WebSocket 连接
  await initWebSocket()
  
  // 步骤5：加载历史消息和在线成员
  fetchHistoryMessages()
  fetchOnlineMembers()
  
  // 定期刷新在线成员列表
  const refreshInterval = setInterval(fetchOnlineMembers, 10000)
  
  onUnmounted(() => {
    clearInterval(refreshInterval)
    if (socket.value) {
      socket.value.close()
    }
  })
})
</script>

<style scoped>
.group-chat-container {
  display: flex;
  height: 100vh;
  background-color: #f5f5f5;
}

.group-members-sidebar {
  width: 250px;
  background-color: white;
  border-right: 1px solid #e0e0e0;
  padding: 20px;
  overflow-y: auto;
}

.group-header {
  margin-bottom: 20px;
  padding-bottom: 15px;
  border-bottom: 1px solid #eee;
}

.group-header h3 {
  margin: 0 0 10px 0;
  color: #333;
  font-size: 18px;
}

.group-actions {
  margin-top: 10px;
}

.group-members h4 {
  margin: 0 0 15px 0;
  color: #666;
  font-size: 14px;
}

.member-tabs {
  height: 100%;
}

.member-tabs .el-tabs__content {
  height: calc(100% - 40px);
  overflow-y: auto;
}

.online-members {
  padding: 8px 0;
}

.member-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.member-item {
  display: flex;
  align-items: center;
  padding: 12px;
  border-radius: 8px;
  transition: background-color 0.2s;
  margin-bottom: 8px;
}

.member-item:hover {
  background-color: #f0f0f0;
}

.member-item.current-user {
  background-color: #f5f5f5;
}

.member-avatar {
  position: relative;
  margin-right: 12px;
}

.member-avatar img {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
}

.member-status {
  position: absolute;
  bottom: 2px;
  right: 2px;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  border: 2px solid #fff;
}

.member-status.online {
  background-color: #52c41a;
}

.member-info {
  flex: 1;
  min-width: 0;
}

.member-name {
  font-weight: bold;
  color: #333;
  margin-bottom: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.member-status-text {
  font-size: 11px;
  color: #52c41a;
  margin-bottom: 4px;
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

.chat-header h2 {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  color: #333;
}

.group-label {
  background: linear-gradient(45deg, #1890ff, #52c41a);
  color: white;
  font-size: 12px;
  padding: 4px 8px;
  border-radius: 12px;
  font-weight: normal;
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

.group-message {
  border-left: 4px solid #52c41a;
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
