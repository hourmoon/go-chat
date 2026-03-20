import { defineStore } from 'pinia'
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import request from '../utils/request'

/**
 * 好友状态管理
 * setup 风格 defineStore
 */
export const useFriendStore = defineStore('friend', () => {
  // ── 状态 ──────────────────────────────────────────────
  const friends = ref([])
  const pendingRequests = ref([])
  const loading = ref(false)

  // ── Actions ───────────────────────────────────────────

  /** 并行拉取好友列表 + 待处理请求 */
  async function fetchAll() {
    loading.value = true
    try {
      const [friendsRes, pendingRes] = await Promise.all([
        request.get('/friends'),
        request.get('/friends/pending')
      ])
      friends.value = friendsRes.data || []
      pendingRequests.value = pendingRes.data || []
      console.log('✅ 好友列表加载成功:', friends.value.length, '位好友,', pendingRequests.value.length, '条待处理')
    } catch (error) {
      console.error('❌ 获取好友数据失败:', error)
      ElMessage.error('获取好友列表失败')
    } finally {
      loading.value = false
    }
  }

  /** 仅拉取好友列表 */
  async function fetchFriends() {
    try {
      const res = await request.get('/friends')
      friends.value = res.data || []
      console.log('✅ 好友列表刷新成功:', friends.value.length, '位')
    } catch (error) {
      console.error('❌ 获取好友列表失败:', error)
      ElMessage.error('获取好友列表失败')
    }
  }

  /** 仅拉取待处理请求 */
  async function fetchPendingRequests() {
    try {
      const res = await request.get('/friends/pending')
      pendingRequests.value = res.data || []
    } catch (error) {
      console.error('❌ 获取待处理好友请求失败:', error)
      ElMessage.error('获取好友请求失败')
    }
  }

  /** 发送好友申请 */
  async function addFriend(username) {
    try {
      await request.post('/friends', { username })
      ElMessage.success('好友申请已发送')
      console.log('✅ 好友申请已发送: username=', username)
    } catch (error) {
      console.error('❌ 发送好友申请失败:', error)
      ElMessage.error('发送好友申请失败')
      throw error
    }
  }

  /** 处理好友请求（接受/拒绝） */
  async function handleRequest(friendId, action) {
    try {
      await request.post(`/friends/${friendId}/action`, { friend_id: friendId, action })
      // 从待处理列表移除
      pendingRequests.value = pendingRequests.value.filter(r => r.id !== friendId)
      if (action === 'accept') {
        ElMessage.success('已接受好友请求')
        // 接受后立即刷新好友列表
        await fetchFriends()
      } else {
        ElMessage.info('已拒绝好友请求')
      }
      console.log('✅ 好友请求已处理: friendID=', friendId, 'action=', action)
    } catch (error) {
      console.error('❌ 处理好友请求失败:', error)
      ElMessage.error('操作失败')
      throw error
    }
  }

  /** 删除好友 */
  async function removeFriend(friendId) {
    try {
      await request.delete(`/friends/${friendId}`)
      friends.value = friends.value.filter(f => f.id !== friendId)
      ElMessage.success('已删除好友')
      console.log('✅ 好友已删除: friendID=', friendId)
    } catch (error) {
      console.error('❌ 删除好友失败:', error)
      ElMessage.error('删除好友失败')
      throw error
    }
  }

  /** 重置所有状态（退出登录时调用） */
  function resetState() {
    friends.value = []
    pendingRequests.value = []
    loading.value = false
  }

  return {
    friends,
    pendingRequests,
    loading,
    fetchAll,
    fetchFriends,
    fetchPendingRequests,
    addFriend,
    handleRequest,
    removeFriend,
    resetState
  }
})
