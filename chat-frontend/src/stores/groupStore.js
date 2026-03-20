import { defineStore, getActivePinia } from 'pinia'
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import * as groupApi from '../utils/groupApi'
import { getCurrentUserId } from '../utils/auth'

/**
 * 群组状态管理
 * 使用 Pinia defineStore（setup 风格）
 * 默认导出兼容层，保持 Groups.vue/GroupChat.vue 的原有调用方式不变：
 *   groupStore.state.xxx
 *   groupStore.actions.xxx
 */
export const useGroupStore = defineStore('group', () => {
  // ── 状态 ──────────────────────────────────────────────
  const groupList = ref([])
  const currentGroupId = ref(null)
  const groupMessages = ref({})
  const currentGroup = ref(null)
  const loading = ref({
    groupList: false,
    messages: false
  })

  // ── 计算属性 ──────────────────────────────────────────
  const currentGroupMessages = computed(() => {
    if (!currentGroupId.value) return []
    return groupMessages.value[currentGroupId.value] || []
  })

  const hasGroups = computed(() => groupList.value.length > 0)

  // ── 辅助函数 ──────────────────────────────────────────
  function _formatTime(timeString) {
    return new Date(timeString).toLocaleTimeString()
  }

  // ── Actions ───────────────────────────────────────────

  /** 获取用户的群组列表 */
  async function fetchUserGroups() {
    if (loading.value.groupList) return
    loading.value.groupList = true
    try {
      const response = await groupApi.getUserGroups()
      groupList.value = response.groups || []
      console.log('✅ 群组列表加载成功:', groupList.value.length, '个')
    } catch (error) {
      console.error('❌ 获取群组列表失败:', error)
      ElMessage.error('获取群组列表失败')
      groupList.value = []
    } finally {
      loading.value.groupList = false
    }
  }

  /** 选择群组（切换当前群组） */
  async function selectGroup(groupId) {
    if (currentGroupId.value === groupId) return
    currentGroupId.value = groupId

    if (groupId) {
      const group = groupList.value.find(g => (g.ID || g.id) === groupId)
      currentGroup.value = group || null
      await fetchGroupMessages(groupId)
      const groupName = group?.name || group?.Name || '未知群组'
      ElMessage.info(`已切换到群组: ${groupName}`)
    } else {
      currentGroup.value = null
      ElMessage.info('已切换到全局聊天')
    }
  }

  /** 获取群组消息（页码默认1） */
  async function fetchGroupMessages(groupId, page = 1) {
    if (!groupId || loading.value.messages) return
    loading.value.messages = true
    try {
      const response = await groupApi.getGroupMessages(groupId, page)
      const currentUserId = await getCurrentUserId()

      const msgs = response.messages.map(msg => ({
        id: msg.id,
        content: msg.content,
        sender: msg.username,
        timestamp: new Date(msg.created_at),
        time: _formatTime(msg.created_at),
        isOwn: msg.user_id === currentUserId,
        isPrivate: false,
        isSystem: false,
        messageType: msg.message_type || 'text',
        fileUrl: msg.file_url,
        fileName: msg.file_name,
        fileSize: msg.file_size,
        groupId: msg.group_id
      }))

      groupMessages.value[groupId] = msgs.reverse()
      console.log(`✅ 群组 ${groupId} 消息加载成功:`, msgs.length, '条')
      return response.pagination
    } catch (error) {
      console.error('❌ 获取群组消息失败:', error)
      ElMessage.error('获取群组消息失败')
      groupMessages.value[groupId] = []
    } finally {
      loading.value.messages = false
    }
  }

  /** 添加新消息到指定群组（WebSocket 推送时调用） */
  async function addMessageToGroup(groupId, message) {
    if (!groupId) return
    if (!groupMessages.value[groupId]) {
      groupMessages.value[groupId] = []
    }
    const currentUserId = await getCurrentUserId()
    const formatted = {
      id: message.ID || message.id || Date.now(),
      content: message.content,
      sender: message.username,
      timestamp: new Date(message.created_at || new Date()),
      time: _formatTime(message.created_at || new Date()),
      isOwn: message.user_id === currentUserId,
      isPrivate: false,
      isSystem: false,
      messageType: message.message_type || 'text',
      fileUrl: message.file_url,
      fileName: message.file_name,
      fileSize: message.file_size,
      groupId: message.group_id
    }
    groupMessages.value[groupId].push(formatted)
  }

  /** 创建群组 */
  async function createGroup(groupData) {
    try {
      const response = await groupApi.createGroup(groupData)
      ElMessage.success('群组创建成功')
      await fetchUserGroups()
      return response.group
    } catch (error) {
      console.error('❌ 创建群组失败:', error)
      ElMessage.error('创建群组失败')
      throw error
    }
  }

  /** 退出群组选择 */
  function exitGroup() {
    currentGroupId.value = null
    currentGroup.value = null
  }

  /** 清空群组消息缓存 */
  function clearGroupMessages(groupId) {
    if (groupId) {
      delete groupMessages.value[groupId]
    } else {
      groupMessages.value = {}
    }
  }

  /** 重置所有状态（退出登录时调用） */
  function resetState() {
    groupList.value = []
    currentGroupId.value = null
    groupMessages.value = {}
    currentGroup.value = null
    loading.value = { groupList: false, messages: false }
  }

  return {
    groupList,
    currentGroupId,
    groupMessages,
    currentGroup,
    loading,
    currentGroupMessages,
    hasGroups,
    fetchUserGroups,
    selectGroup,
    fetchGroupMessages,
    addMessageToGroup,
    createGroup,
    exitGroup,
    clearGroupMessages,
    resetState
  }
})

/**
 * 兼容层默认导出
 * Groups.vue / GroupChat.vue 中的 groupStore.state.xxx 和 groupStore.actions.xxx
 * 均通过此代理对象透明地访问真正的 Pinia store 实例。
 *
 * Pinia store 实例在组件外（setup 之外）调用时，需要传入 pinia 实例。
 * 这里使用 getActivePinia() 获取应用级 Pinia 实例，在 app.use(pinia) 之后始终可用。
 */
function getStore() {
  return useGroupStore(getActivePinia())
}

const groupStoreCompat = {
  get state() {
    return getStore()
  },
  get actions() {
    return getStore()
  },
  get getters() {
    return getStore()
  }
}

export default groupStoreCompat
