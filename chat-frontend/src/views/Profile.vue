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
              <el-button type="primary" size="small" icon="el-icon-camera">
                更换头像
              </el-button>
            </el-upload>
          </div>
        </div>

        <!-- 用户信息表单 -->
        <el-form :model="profile" :rules="profileRules" ref="profileForm" label-width="80px">
          <el-form-item label="用户名" prop="username">
            <el-input v-model="profile.username" disabled></el-input>
          </el-form-item>

          <el-form-item label="个性签名" prop="bio">
            <el-input
              v-model="profile.bio"
              type="textarea"
              :rows="3"
              placeholder="请输入个性签名"
              maxlength="200"
              show-word-limit
            ></el-input>
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
            <el-button type="primary" @click="updateProfile" :loading="loading">
              保存修改
            </el-button>
            <el-button @click="resetForm">重置</el-button>
          </el-form-item>
        </el-form>

        <!-- 好友管理区域 -->
        <div class="friends-section">
          <h3>好友管理</h3>
          
          <!-- 好友请求区域 -->
          <div v-if="pendingRequests.length > 0" class="pending-requests">
            <h4>待处理的好友请求 ({{ pendingRequests.length }})</h4>
            <div 
              v-for="request in pendingRequests" 
              :key="request.id"
              class="friend-request-item"
            >
              <div class="friend-avatar">
                <img 
                  :src="request.avatar ? getFullAvatarUrl(request.avatar) : defaultAvatar" 
                  :alt="request.username"
                />
              </div>
              <div class="friend-info">
                <div class="friend-name">{{ request.username }}</div>
                <div class="friend-bio">{{ request.bio || '这个人很懒，什么都没有写' }}</div>
              </div>
              <div class="friend-actions">
                <el-button 
                  @click="handleFriendRequest(request.id, 'accept')" 
                  type="success" 
                  size="small"
                >
                  接受
                </el-button>
                <el-button 
                  @click="handleFriendRequest(request.id, 'reject')" 
                  type="danger" 
                  size="small"
                >
                  拒绝
                </el-button>
              </div>
            </div>
          </div>
          
          <div class="add-friend">
            <el-input
              v-model="newFriendUsername"
              placeholder="输入用户名添加好友"
              @keyup.enter="addFriend"
            >
              <template #append>
                <el-button @click="addFriend" :loading="addingFriend">添加</el-button>
              </template>
            </el-input>
          </div>

          <div class="friends-list">
            <div v-if="friends.length === 0" class="no-friends">
              暂无好友
            </div>
            <div v-else>
              <div 
                v-for="friend in friends" 
                :key="friend.id"
                class="friend-item"
              >
                <div class="friend-avatar">
                  <img 
                    :src="friend.avatar ? getFullAvatarUrl(friend.avatar) : defaultAvatar" 
                    :alt="friend.username"
                  />
                  <span :class="['status-dot', friend.status]"></span>
                </div>
                <div class="friend-info">
                  <div class="friend-name">{{ friend.username }}</div>
                  <div class="friend-bio">{{ friend.bio || '这个人很懒，什么都没有写' }}</div>
                  <div class="friend-status">
                    <span :class="['status-text', friend.status]">
                      {{ getStatusText(friend.status) }}
                    </span>
                    <span class="last-seen" v-if="friend.status === 'offline'">
                      {{ formatLastSeen(friend.last_seen) }}
                    </span>
                  </div>
                </div>
                <div class="friend-actions">
                  <el-button 
                    @click="startChatWithFriend(friend)" 
                    type="primary" 
                    size="small"
                  >
                    聊天
                  </el-button>
                  <el-button 
                    @click="removeFriend(friend.id)" 
                    type="danger" 
                    size="small"
                  >
                    删除
                  </el-button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script>
import request from '@/utils/request'

export default {
  name: 'Profile',
  data() {
    return {
      loading: false,
      addingFriend: false,
      profile: {
        id: 0,
        username: '',
        avatar: '',
        bio: '',
        status: 'online'
      },
      friends: [],
      pendingRequests: [],
      newFriendUsername: '',
      defaultAvatar: 'https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png',
      profileRules: {
        bio: [
          { max: 200, message: '个性签名不能超过200个字符', trigger: 'blur' }
        ]
      }
    }
  },
  mounted() {
    this.loadProfile()
    this.loadFriends()
    this.loadPendingRequests()
  },
  methods: {
    // 获取完整头像URL
    getFullAvatarUrl(avatar) {
      if (!avatar) return this.defaultAvatar
      if (avatar.startsWith('http')) return avatar
      return `http://localhost:8080${avatar}`
    },

    // 加载用户资料
    async loadProfile() {
      try {
        const response = await request.get('/profile')
        if (response.success) {
          this.profile = response.data
        }
      } catch (error) {
        console.error('加载资料失败:', error)
        this.$message.error('加载资料失败')
      }
    },

    // 加载好友列表
    async loadFriends() {
      try {
        const response = await request.get('/friends')
        if (response.success) {
          this.friends = response.data
        }
      } catch (error) {
        console.error('加载好友列表失败:', error)
        this.$message.error('加载好友列表失败')
      }
    },

    // 加载待处理的好友请求
    async loadPendingRequests() {
      try {
        const response = await request.get('/friends/pending')
        if (response.success) {
          this.pendingRequests = response.data
        }
      } catch (error) {
        console.error('加载好友请求失败:', error)
        this.$message.error('加载好友请求失败')
      }
    },

    // 更新资料
    async updateProfile() {
      this.$refs.profileForm.validate().then(async () => {
        this.loading = true
        try {
          // 更新基本资料
          await request.put('/profile', {
            bio: this.profile.bio
          })

          // 更新状态
          await request.put('/profile/status', {
            status: this.profile.status
          })

          this.$message.success('资料更新成功')
        } catch (error) {
          console.error('更新失败:', error)
          this.$message.error(error?.error || '更新失败')
        } finally {
          this.loading = false
        }
      }).catch(() => {
        this.$message.warning('请检查输入信息')
      })
    },

    // 重置表单
    resetForm() {
      this.loadProfile()
    },

    // 头像上传前验证
    beforeAvatarUpload(file) {
      const isImage = file.type.startsWith('image/')
      const isLt2M = file.size / 1024 / 1024 < 2

      if (!isImage) {
        this.$message.error('只能上传图片文件!')
        return false
      }
      if (!isLt2M) {
        this.$message.error('图片大小不能超过2MB!')
        return false
      }
      return true
    },

    // 处理头像上传
    async handleAvatarUpload(options) {
      const formData = new FormData()
      formData.append('avatar', options.file)
      
      try {
        const response = await request.post('/profile/avatar', formData, {
          headers: {
            'Content-Type': 'multipart/form-data'
          }
        })
        
        if (response.success) {
          this.profile.avatar = response.avatar
          this.$message.success('头像上传成功')
        } else {
          this.$message.error('头像上传失败')
        }
      } catch (error) {
        console.error('头像上传错误:', error)
        this.$message.error('头像上传失败')
      }
    },

    // 添加好友
    async addFriend() {
      if (!this.newFriendUsername.trim()) {
        this.$message.warning('请输入用户名')
        return
      }

      this.addingFriend = true
      try {
        const response = await request.post('/friends', {
          username: this.newFriendUsername
        })
        
        if (response.success) {
          this.$message.success('好友请求已发送')
          this.newFriendUsername = ''
        } else {
          this.$message.error(response.error || '添加好友失败')
        }
      } catch (error) {
        console.error('添加好友失败:', error)
        this.$message.error(error?.error || '添加好友失败')
      } finally {
        this.addingFriend = false
      }
    },

    // 处理好友请求
    async handleFriendRequest(friendId, action) {
      try {
        const response = await request.post(`/friends/${friendId}/action`, {
          friend_id: friendId,
          action: action
        })
        
        if (response.success) {
          this.$message.success(action === 'accept' ? '已接受好友请求' : '已拒绝好友请求')
          this.loadPendingRequests() // 重新加载待处理请求
          this.loadFriends() // 重新加载好友列表
        } else {
          this.$message.error('处理好友请求失败')
        }
      } catch (error) {
        console.error('处理好友请求失败:', error)
        this.$message.error('处理好友请求失败')
      }
    },

    // 删除好友
    async removeFriend(friendId) {
      try {
        await this.$confirm('确定要删除这个好友吗？', '提示', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        })

        const response = await request.delete(`/friends/${friendId}`)
        if (response.success) {
          this.$message.success('好友删除成功')
          this.loadFriends() // 重新加载好友列表
        } else {
          this.$message.error('删除失败')
        }
      } catch (error) {
        if (error !== 'cancel') {
          console.error('删除好友失败:', error)
          this.$message.error('删除失败')
        }
      }
    },

    // 与好友开始聊天
    startChatWithFriend(friend) {
      this.$router.push({
        name: 'Chat',
        query: { friendId: friend.id, friendName: friend.username }
      })
    },

    // 获取状态文本
    getStatusText(status) {
      const statusMap = {
        'online': '在线',
        'busy': '忙碌',
        'away': '离开',
        'offline': '离线'
      }
      return statusMap[status] || '未知'
    },

    // 格式化最后在线时间
    formatLastSeen(lastSeen) {
      if (!lastSeen) return ''
      const date = new Date(lastSeen)
      const now = new Date()
      const diff = now - date
      const diffMins = Math.floor(diff / 60000)
      const diffHours = Math.floor(diff / 3600000)
      const diffDays = Math.floor(diff / 86400000)
      
      if (diffMins < 1) return '刚刚'
      if (diffMins < 60) return `${diffMins}分钟前`
      if (diffHours < 24) return `${diffHours}小时前`
      if (diffDays < 7) return `${diffDays}天前`
      
      return date.toLocaleDateString()
    },

    // 返回聊天页面
    goBack() {
      this.$router.push('/chat')
    }
  }
}
</script>

<style scoped>
.profile-page {
  background-image: linear-gradient(180deg, #2af598 0%, #009efd 100%);
  min-height: 100vh;
  padding: 20px;
}

.profile-card {
  max-width: 800px;
  margin: 0 auto;
}

.profile-title {
  font-size: 20px;
  font-weight: bold;
}

.profile-content {
  padding: 20px 0;
}

.avatar-section {
  text-align: center;
  margin-bottom: 30px;
}

.avatar-container {
  display: inline-block;
  position: relative;
}

.avatar {
  width: 100px;
  height: 100px;
  border-radius: 50%;
  object-fit: cover;
  border: 3px solid #fff;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
}

.avatar-upload {
  margin-top: 10px;
}

.friends-section {
  margin-top: 40px;
  padding-top: 20px;
  border-top: 1px solid #eee;
}

.friends-section h3 {
  margin-bottom: 20px;
  color: #333;
}

.pending-requests {
  margin-bottom: 20px;
  padding: 15px;
  background-color: #f8f9fa;
  border-radius: 8px;
  border: 1px solid #e9ecef;
}

.pending-requests h4 {
  margin-bottom: 15px;
  color: #495057;
  font-size: 14px;
}

.friend-request-item {
  display: flex;
  align-items: center;
  padding: 10px;
  border: 1px solid #dee2e6;
  border-radius: 6px;
  margin-bottom: 8px;
  background-color: #fff;
  transition: background-color 0.2s;
}

.friend-request-item:hover {
  background-color: #f8f9fa;
}

.add-friend {
  margin-bottom: 20px;
}

.friends-list {
  max-height: 400px;
  overflow-y: auto;
}

.no-friends {
  text-align: center;
  color: #999;
  padding: 20px;
}

.friend-item {
  display: flex;
  align-items: center;
  padding: 15px;
  border: 1px solid #eee;
  border-radius: 8px;
  margin-bottom: 10px;
  background-color: #fafafa;
  transition: background-color 0.2s;
}

.friend-item:hover {
  background-color: #f0f0f0;
}

.friend-avatar {
  position: relative;
  margin-right: 15px;
}

.friend-avatar img {
  width: 50px;
  height: 50px;
  border-radius: 50%;
  object-fit: cover;
}

.status-dot {
  position: absolute;
  bottom: 2px;
  right: 2px;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  border: 2px solid #fff;
}

.status-dot.online {
  background-color: #52c41a;
}

.status-dot.busy {
  background-color: #ff4d4f;
}

.status-dot.away {
  background-color: #faad14;
}

.status-dot.offline {
  background-color: #d9d9d9;
}

.friend-info {
  flex: 1;
}

.friend-name {
  font-weight: bold;
  color: #333;
  margin-bottom: 5px;
}

.friend-bio {
  color: #666;
  font-size: 12px;
  margin-bottom: 5px;
}

.friend-status {
  display: flex;
  align-items: center;
  gap: 10px;
}

.status-text {
  font-size: 12px;
  padding: 2px 6px;
  border-radius: 10px;
  color: #fff;
}

.status-text.online {
  background-color: #52c41a;
}

.status-text.busy {
  background-color: #ff4d4f;
}

.status-text.away {
  background-color: #faad14;
}

.status-text.offline {
  background-color: #d9d9d9;
  color: #666;
}

.last-seen {
  font-size: 11px;
  color: #999;
}

.friend-actions {
  display: flex;
  gap: 5px;
}
</style>
