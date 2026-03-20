<template>
  <div class="profile-page">
    <el-card class="profile-card">
      <div slot="header" class="clearfix">
        <span class="profile-title">个人资料</span>
        <el-button @click="goBack" style="float: right;" type="primary" size="small">
          返回聊天
        </el-button>
      </div>
      
      <div class="profile-content">
        <!-- 头像区域 -->
        <div class="avatar-section">
          <div class="avatar-container">
            <img 
              :src="profile.avatar ? getFullAvatarUrl(profile.avatar) : defaultAvatar" 
              :alt="profile.username"
              class="avatar"
            />
            <el-upload
              action="#"
              :show-file-list="false"
              :before-upload="beforeAvatarUpload"
              :http-request="handleAvatarUpload"
              class="avatar-upload"
            >
              <el-button type="primary" size="small">更换头像</el-button>
            </el-upload>
          </div>
        </div>

        <!-- 用户信息表单 -->
        <el-form :model="profile" :rules="profileRules" ref="profileFormRef" label-width="80px">
          <el-form-item label="用户名" prop="username">
            <el-input v-model="profile.username" disabled></el-input>
          </el-form-item>
          <el-form-item label="个性签名" prop="bio">
            <el-input v-model="profile.bio" type="textarea" :rows="3"
              placeholder="请输入个性签名" maxlength="200" show-word-limit>
            </el-input>
          </el-form-item>
          <el-form-item label="在线状态" prop="status">
            <el-select v-model="profile.status" placeholder="选择状态">
              <el-option label="在线" value="online"></el-option>
              <el-option label="忙碌" value="busy"></el-option>
              <el-option label="离开" value="away"></el-option>
              <el-option label="离线" value="offline"></el-option>
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleUpdateProfile" :loading="userStore.loading">保存修改</el-button>
            <el-button @click="resetForm">重置</el-button>
          </el-form-item>
        </el-form>

        <!-- 好友管理区域 -->
        <div class="friends-section">
          <h3>好友管理</h3>
          
          <!-- 待处理好友请求 -->
          <div v-if="pendingRequests.length > 0" class="pending-requests">
            <h4>待处理的好友请求 ({{ pendingRequests.length }})</h4>
            <div v-for="req in pendingRequests" :key="req.id" class="friend-request-item">
              <div class="friend-avatar">
                <img :src="req.avatar ? getFullAvatarUrl(req.avatar) : defaultAvatar" :alt="req.username" />
              </div>
              <div class="friend-info">
                <div class="friend-name">{{ req.username }}</div>
                <div class="friend-bio">{{ req.bio || '这个人很懒，什么都没有写' }}</div>
              </div>
              <div class="friend-actions">
                <el-button @click="handleFriendRequest(req.id, 'accept')" type="success" size="small">接受</el-button>
                <el-button @click="handleFriendRequest(req.id, 'reject')" type="danger" size="small">拒绝</el-button>
              </div>
            </div>
          </div>
          
          <!-- 添加好友 -->
          <div class="add-friend">
            <el-input v-model="newFriendUsername" placeholder="输入用户名添加好友" @keyup.enter="addFriend">
              <template #append>
                <el-button @click="addFriend" :loading="addingFriend">添加</el-button>
              </template>
            </el-input>
          </div>

          <!-- 好友列表 -->
          <div class="friends-list">
            <div v-if="friends.length === 0" class="no-friends">暂无好友</div>
            <div v-else>
              <div v-for="friend in friends" :key="friend.id" class="friend-item">
                <div class="friend-avatar">
                  <img :src="friend.avatar ? getFullAvatarUrl(friend.avatar) : defaultAvatar" :alt="friend.username" />
                  <span :class="['status-dot', friend.status]"></span>
                </div>
                <div class="friend-info">
                  <div class="friend-name">{{ friend.username }}</div>
                  <div class="friend-bio">{{ friend.bio || '这个人很懒，什么都没有写' }}</div>
                  <div class="friend-status">
                    <span :class="['status-text', friend.status]">{{ getStatusText(friend.status) }}</span>
                    <span class="last-seen" v-if="friend.status === 'offline'">{{ formatLastSeen(friend.last_seen) }}</span>
                  </div>
                </div>
                <div class="friend-actions">
                  <el-button @click="startChatWithFriend(friend)" type="primary" size="small">聊天</el-button>
                  <el-button @click="removeFriend(friend.id)" type="danger" size="small">删除</el-button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useUserStore } from '../stores/userStore'
import { useFriendStore } from '../stores/friendStore'

const router = useRouter()
const userStore = useUserStore()
const friendStore = useFriendStore()

// ── 模板需要的直接绑定 ────────────────────────────────
const profile = computed(() => userStore.profile)
const friends = computed(() => friendStore.friends)
const pendingRequests = computed(() => friendStore.pendingRequests)

// ── 本地 UI 状态 ─────────────────────────────────────
const profileFormRef = ref(null)
const newFriendUsername = ref('')
const addingFriend = ref(false)
const defaultAvatar = 'https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png'

const profileRules = {
  bio: [{ max: 200, message: '个性签名不能超过200字', trigger: 'blur' }]
}

// ── 辅助函数 ──────────────────────────────────────────
const getFullAvatarUrl = (avatar) => {
  if (!avatar) return defaultAvatar
  if (avatar.startsWith('http')) return avatar
  return `http://localhost:8080${avatar}`
}

const getStatusText = (status) => {
  const map = { online: '在线', busy: '忙碌', away: '离开', offline: '离线' }
  return map[status] || '未知'
}

const formatLastSeen = (lastSeen) => {
  if (!lastSeen) return ''
  const d = new Date(lastSeen)
  return d.toLocaleString()
}

// ── 用户资料操作 ──────────────────────────────────────

/** 保存签名 + 状态 */
const handleUpdateProfile = async () => {
  try {
    await userStore.updateProfile(profile.value.bio)
    await userStore.updateStatus(profile.value.status)
  } catch {}
}

/** 重置表单到 store 中的当前值（重新拉取） */
const resetForm = async () => {
  await userStore.fetchProfile()
}

/** 上传头像前校验 */
const beforeAvatarUpload = (file) => {
  const isImage = file.type.startsWith('image/')
  const isLt2M = file.size / 1024 / 1024 < 2
  if (!isImage) {
    ElMessage.error('只能上传图片文件!')
    return false
  }
  if (!isLt2M) {
    ElMessage.error('图片大小不能超过 2MB!')
    return false
  }
  return true
}

/** 执行头像上传 */
const handleAvatarUpload = async (options) => {
  try {
    await userStore.uploadAvatar(options.file)
  } catch {}
}

// ── 好友操作 ──────────────────────────────────────────

/** 发送好友申请 */
const addFriend = async () => {
  const name = newFriendUsername.value.trim()
  if (!name) {
    ElMessage.warning('请输入用户名')
    return
  }
  addingFriend.value = true
  try {
    await friendStore.addFriend(name)
    newFriendUsername.value = ''
  } catch {} finally {
    addingFriend.value = false
  }
}

/** 接受 / 拒绝好友请求 */
const handleFriendRequest = async (friendId, action) => {
  try {
    await friendStore.handleRequest(friendId, action)
  } catch {}
}

/** 删除好友 */
const removeFriend = async (friendId) => {
  try {
    await ElMessageBox.confirm('确定要删除该好友吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await friendStore.removeFriend(friendId)
  } catch {}
}

/** 跳转到私聊 */
const startChatWithFriend = (friend) => {
  router.push({ path: '/chat', query: { friendId: friend.id, friendName: friend.username } })
}

// ── 导航 ─────────────────────────────────────────────
const goBack = () => router.push('/chat')

// ── 初始化 ───────────────────────────────────────────
onMounted(async () => {
  // 若 profile 尚未加载（id === 0），则拉取
  if (!userStore.profile.id) {
    await userStore.fetchProfile()
  }
  // 并行拉取好友数据
  await friendStore.fetchAll()
})
</script>

<style scoped>
.profile-page {
  min-height: 100vh;
  background-color: #f5f5f5;
  padding: 20px;
}

.profile-card {
  max-width: 800px;
  margin: 0 auto;
}

.profile-title {
  font-size: 18px;
  font-weight: bold;
}

.profile-content {
  padding: 10px;
}

/* 头像区域 */
.avatar-section {
  display: flex;
  justify-content: center;
  margin-bottom: 30px;
}

.avatar-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

.avatar {
  width: 100px;
  height: 100px;
  border-radius: 50%;
  object-fit: cover;
  border: 3px solid #409EFF;
}

/* 好友管理 */
.friends-section {
  margin-top: 30px;
  border-top: 1px solid #eee;
  padding-top: 20px;
}

.friends-section h3 {
  margin-bottom: 16px;
  color: #333;
}

.pending-requests {
  background-color: #fff9e6;
  border: 1px solid #ffd666;
  border-radius: 8px;
  padding: 12px;
  margin-bottom: 16px;
}

.pending-requests h4 {
  margin: 0 0 12px;
  color: #d48806;
}

.friend-request-item,
.friend-item {
  display: flex;
  align-items: center;
  padding: 10px;
  border-radius: 8px;
  margin-bottom: 8px;
  background-color: #fafafa;
  border: 1px solid #f0f0f0;
}

.friend-avatar {
  position: relative;
  margin-right: 12px;
}

.friend-avatar img {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  object-fit: cover;
}

.status-dot {
  position: absolute;
  bottom: 1px;
  right: 1px;
  width: 11px;
  height: 11px;
  border-radius: 50%;
  border: 2px solid #fff;
}

.status-dot.online  { background-color: #52c41a; }
.status-dot.busy    { background-color: #ff4d4f; }
.status-dot.away    { background-color: #faad14; }
.status-dot.offline { background-color: #d9d9d9; }

.friend-info {
  flex: 1;
  min-width: 0;
}

.friend-name {
  font-weight: bold;
  color: #333;
  margin-bottom: 2px;
}

.friend-bio {
  font-size: 12px;
  color: #888;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.friend-status {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
}

.status-text {
  font-size: 11px;
  padding: 1px 6px;
  border-radius: 10px;
  color: #fff;
}

.status-text.online  { background-color: #52c41a; }
.status-text.busy    { background-color: #ff4d4f; }
.status-text.away    { background-color: #faad14; }
.status-text.offline { background-color: #d9d9d9; color: #666; }

.last-seen {
  font-size: 11px;
  color: #aaa;
}

.friend-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.add-friend {
  margin-bottom: 16px;
}

.no-friends {
  text-align: center;
  color: #aaa;
  padding: 24px 0;
}
</style>
