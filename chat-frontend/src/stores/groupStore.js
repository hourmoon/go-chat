import { reactive, computed } from 'vue'
import { ElMessage } from 'element-plus'
import * as groupApi from '../utils/groupApi'

/**
 * 群组状态管理
 * 使用 Vue 3 的 reactive API 实现简单的状态管理
 */

// 群组状态
const state = reactive({
  // 用户加入的群组列表
  groupList: [],
  // 当前选中的群组ID，null表示没有选中群组，0表示全局聊天
  currentGroupId: null,
  // 各个群组的消息缓存，以groupId为键存储消息数组
  groupMessages: {},
  // 加载状态
  loading: {
    groupList: false,
    messages: false
  },
  // 当前选中的群组信息
  currentGroup: null
})

// 计算属性
const getters = {
  // 获取当前群组的消息
  currentGroupMessages: computed(() => {
    if (!state.currentGroupId) return []
    return state.groupMessages[state.currentGroupId] || []
  }),
  
  // 检查是否已加载群组列表
  hasGroups: computed(() => state.groupList.length > 0),
  
  // 获取当前群组信息
  getCurrentGroup: computed(() => state.currentGroup)
}

// Actions
const actions = {
  // 获取用户的群组列表
  async fetchUserGroups() {
    if (state.loading.groupList) return
    
    state.loading.groupList = true
    try {
      const response = await groupApi.getUserGroups()
      state.groupList = response.groups || []
      console.log('✅ 群组列表加载成功:', state.groupList)
    } catch (error) {
      console.error('❌ 获取群组列表失败:', error)
      ElMessage.error('获取群组列表失败')
      state.groupList = []
    } finally {
      state.loading.groupList = false
    }
  },

  // 选择群组
  async selectGroup(groupId) {
    if (state.currentGroupId === groupId) return
    
    // 设置当前群组ID
    state.currentGroupId = groupId
    
    if (groupId) {
      // 查找群组详情（兼容大小写字段名）
      const group = state.groupList.find(g => (g.ID || g.id) === groupId)
      state.currentGroup = group || null
      
      // 加载群组消息
      await actions.fetchGroupMessages(groupId)
      const groupName = group?.name || group?.Name || '未知群组'
      ElMessage.info(`已切换到群组: ${groupName}`)
    } else {
      // 切换到全局聊天
      state.currentGroup = null
      ElMessage.info('已切换到全局聊天')
    }
  },

  // 获取群组消息
  async fetchGroupMessages(groupId, page = 1) {
    if (!groupId || state.loading.messages) return
    
    state.loading.messages = true
    try {
      const response = await groupApi.getGroupMessages(groupId, page)
      
      // 将消息转换为前端格式
      const messages = response.messages.map(msg => ({
        id: msg.id,
        content: msg.content,
        sender: msg.username,
        timestamp: new Date(msg.created_at),
        time: formatTime(msg.created_at),
        isOwn: msg.user_id === getCurrentUserId(),
        isPrivate: false, // 群消息不是私聊
        isSystem: false,
        messageType: msg.message_type || 'text',
        fileUrl: msg.file_url,
        fileName: msg.file_name,
        fileSize: msg.file_size,
        groupId: msg.group_id
      }))
      
      // 存储到状态中（消息按时间倒序，最新的在前面）
      state.groupMessages[groupId] = messages.reverse()
      
      console.log(`✅ 群组 ${groupId} 消息加载成功:`, messages.length, '条')
      return response.pagination
    } catch (error) {
      console.error('❌ 获取群组消息失败:', error)
      ElMessage.error('获取群组消息失败')
      state.groupMessages[groupId] = []
    } finally {
      state.loading.messages = false
    }
  },

  // 添加新消息到当前群组
  addMessageToGroup(groupId, message) {
    if (!groupId) return
    
    if (!state.groupMessages[groupId]) {
      state.groupMessages[groupId] = []
    }
    
    // ✅ 重新添加消息格式化逻辑，确保所有消息对象结构一致
    const formattedMessage = {
      id: message.ID || message.id || Date.now(), // 兼容 gorm.Model 的 ID
      content: message.content,
      sender: message.username,
      timestamp: new Date(message.created_at || new Date()),
      time: formatTime(message.created_at || new Date()),
      isOwn: message.user_id === getCurrentUserId(),
      isPrivate: false,
      isSystem: false,
      messageType: message.message_type || 'text',
      fileUrl: message.file_url,
      fileName: message.file_name,
      fileSize: message.file_size,
      groupId: message.group_id
    };

    state.groupMessages[groupId].push(formattedMessage)
  },
  // 创建群组
  async createGroup(groupData) {
    try {
      const response = await groupApi.createGroup(groupData)
      ElMessage.success('群组创建成功')
      
      // 刷新群组列表
      await actions.fetchUserGroups()
      
      return response.group
    } catch (error) {
      console.error('❌ 创建群组失败:', error)
      ElMessage.error('创建群组失败')
      throw error
    }
  },

  // 退出群组选择
  exitGroup() {
    state.currentGroupId = null
    state.currentGroup = null
  },

  // 清空群组消息缓存
  clearGroupMessages(groupId) {
    if (groupId) {
      delete state.groupMessages[groupId]
    } else {
      state.groupMessages = {}
    }
  },

  // 重置所有状态
  resetState() {
    state.groupList = []
    state.currentGroupId = null
    state.groupMessages = {}
    state.currentGroup = null
    state.loading = {
      groupList: false,
      messages: false
    }
  }
}

// 辅助函数
function formatTime(timeString) {
  const date = new Date(timeString)
  return date.toLocaleTimeString()
}

function getCurrentUserId() {
  // 从 localStorage 中的 token 解析用户ID
  const token = localStorage.getItem('token')
  if (!token) return 0
  
  try {
    const payload = JSON.parse(atob(token.split('.')[1]))
    return payload.userID || 0
  } catch (error) {
    console.error('解析用户ID失败:', error)
    return 0
  }
}

// 导出状态管理对象
export default {
  state,
  getters,
  actions
}
