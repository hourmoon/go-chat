import { defineStore } from 'pinia'
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import request from '../utils/request'
import { getUsername } from '../utils/auth'

/**
 * 全局聊天 + 私聊消息状态管理
 * setup 风格 defineStore
 */
export const useChatStore = defineStore('chat', () => {
  // ── 状态 ──────────────────────────────────────────────
  /** 全局聊天 + 私聊消息列表（展示在 Chat.vue 中） */
  const messages = ref([])
  /** 未读消息数 */
  const unreadCount = ref(0)
  /** 当前已加载页码 */
  const currentPage = ref(1)
  /** 是否还有更多历史消息 */
  const hasMore = ref(true)

  // ── 辅助函数 ──────────────────────────────────────────

  /** 格式化存储时间 */
  function _formatTime(timeString) {
    return new Date(timeString).toLocaleTimeString()
  }

  /**
   * 统一消息格式转换
   * @param {object} raw 原始消息（HTTP 或 WebSocket）
   * @param {number} currentUserId 当前用户 ID
   */
  function formatMessage(raw, currentUserId) {
    const username = getUsername() || ''
    return {
      id: raw.id || raw.ID || null,
      content: raw.content,
      sender: raw.username,
      timestamp: new Date(raw.created_at || new Date()),
      time: _formatTime(raw.created_at || new Date()),
      // HTTP 历史消息用 username 比对，WS 消息用 user_id 比对
      isOwn: currentUserId
        ? (raw.user_id === currentUserId)
        : (raw.username === username),
      isPrivate: !!(raw.target && raw.target > 0),
      isSystem: false,
      messageType: raw.message_type || 'text',
      fileUrl: raw.file_url || null,
      fileName: raw.file_name || null,
      fileSize: raw.file_size || 0,
      readCount: raw.read_count || 0
    }
  }

  // ── Actions ───────────────────────────────────────────

  /**
   * 拉取历史消息
   * @param {boolean} loadMore 是否追加加载（上拉分页）
   * @param {number}  currentUserId 当前用户 ID（用于 isOwn 判断）
   */
  async function fetchMessages(loadMore = false, currentUserId = 0) {
    if (loadMore) {
      currentPage.value += 1
    } else {
      currentPage.value = 1
      hasMore.value = true
    }

    try {
      const res = await request.get(`/messages?page=${currentPage.value}&pageSize=50`)
      // 只保留全局聊天消息（group_id 为 0 或 null）
      const filtered = (res.messages || []).filter(
        msg => !msg.group_id || msg.group_id === 0
      )
      const newMsgs = filtered.map(msg => formatMessage(msg, currentUserId))

      if (loadMore) {
        messages.value = [...newMsgs, ...messages.value]
      } else {
        messages.value = newMsgs
      }

      // 更新分页状态
      if (res.pagination) {
        hasMore.value = currentPage.value < res.pagination.totalPages
      } else {
        hasMore.value = false
      }

      console.log(
        `✅ 消息加载成功: page=${currentPage.value}, count=${newMsgs.length}, hasMore=${hasMore.value}`
      )
    } catch (error) {
      // 分页失败时回滚页码
      if (loadMore) currentPage.value -= 1
      console.error('❌ 获取历史消息失败:', error)
      ElMessage.error('获取历史消息失败')
      throw error
    }
  }

  /** 拉取未读消息数 */
  async function fetchUnreadCount() {
    try {
      const res = await request.get('/messages/unread-count')
      if (res.success) {
        unreadCount.value = res.unread_count || 0
      }
    } catch (error) {
      console.error('❌ 获取未读数失败:', error)
    }
  }

  /**
   * 添加一条消息（WebSocket 推送时调用）
   * @param {object} raw 原始 WS 消息
   * @param {number} currentUserId 当前用户 ID
   */
  function addMessage(raw, currentUserId = 0) {
    const msg = formatMessage(raw, currentUserId)
    messages.value.push(msg)
  }

  /**
   * 更新消息已读状态（收到 read_receipt 时调用）
   * @param {number} messageId
   * @param {number} readCount
   */
  function updateReadStatus(messageId, readCount) {
    const idx = messages.value.findIndex(m => m.id === messageId)
    if (idx !== -1) {
      messages.value[idx].readCount = readCount
      console.log(`✅ 消息已读状态更新: messageID=${messageId}, readCount=${readCount}`)
    }
  }

  /** 重置所有状态（退出登录时调用） */
  function resetState() {
    messages.value = []
    unreadCount.value = 0
    currentPage.value = 1
    hasMore.value = true
  }

  return {
    messages,
    unreadCount,
    currentPage,
    hasMore,
    formatMessage,
    fetchMessages,
    fetchUnreadCount,
    addMessage,
    updateReadStatus,
    resetState
  }
})
