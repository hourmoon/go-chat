<template>
  <div class="groups-container">
    <div class="groups-header">
      <h1>我的群组</h1>
      <div class="header-actions">
        <el-button @click="showCreateDialog = true" type="success" size="small">
          <el-icon><Plus /></el-icon>
          创建群组
        </el-button>
        <el-button @click="goToChat" type="info" size="small">
          <el-icon><ChatRound /></el-icon>
          全局聊天
        </el-button>
        <el-button @click="goToProfile" type="primary" size="small">
          <el-icon><User /></el-icon>
          个人资料
        </el-button>
        <el-button @click="logout" type="danger" size="small">退出</el-button>
      </div>
    </div>
    
    <div class="groups-content">
      <div v-if="groupStore.state.loading.groupList" class="loading-indicator">
        <el-icon class="is-loading"><Loading /></el-icon>
        <span>加载群组列表中...</span>
      </div>
      
      <div v-else-if="groupStore.state.groupList.length === 0" class="empty-state">
        <el-icon><ChatDotRound /></el-icon>
        <h3>暂无群组</h3>
        <p>您还没有加入任何群组</p>
      </div>
      
      <div v-else class="groups-grid">
        <div 
          v-for="group in groupStore.state.groupList" 
          :key="group.id || group.ID"
          class="group-card"
          @click="enterGroup(group)"
        >
          <div class="group-avatar">
            <img 
              v-if="group.avatar || group.Avatar" 
              :src="getFullAvatarUrl(group.avatar || group.Avatar)" 
              :alt="group.name || group.Name"
            />
            <div v-else class="default-avatar">
              <el-icon><ChatDotRound /></el-icon>
            </div>
          </div>
          
          <div class="group-info">
            <h3 class="group-name">{{ group.name || group.Name }}</h3>
            <p class="group-description">{{ group.description || group.Description || '暂无描述' }}</p>
            <div class="group-meta">
              <span class="member-count">
                <el-icon><User /></el-icon>
                成员数量暂不可见
              </span>
            </div>
          </div>
          
          <div class="group-actions">
            <el-button type="primary" size="small">
              <el-icon><Right /></el-icon>
              进入群聊
            </el-button>
            
            <!-- 根据用户角色显示不同操作 -->
            <el-dropdown @command="(command) => handleGroupAction(command, group)" trigger="click">
              <el-button size="small" type="info" plain>
                <el-icon><MoreFilled /></el-icon>
                更多
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item 
                    v-if="isGroupOwner(group)" 
                    command="delete"
                    class="danger-item"
                  >
                    <el-icon><Delete /></el-icon>
                    解散群组
                  </el-dropdown-item>
                  <el-dropdown-item 
                    v-else 
                    command="leave"
                    class="warning-item"
                  >
                    <el-icon><Close /></el-icon>
                    退出群组
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>
      </div>
    </div>

    <!-- 创建群组对话框 -->
    <el-dialog
      v-model="showCreateDialog"
      title="创建群组"
      width="500px"
      @close="resetCreateForm"
    >
      <el-form
        ref="createFormRef"
        :model="createForm"
        :rules="createFormRules"
        label-width="80px"
      >
        <el-form-item label="群组名称" prop="name">
          <el-input
            v-model="createForm.name"
            placeholder="请输入群组名称"
            maxlength="50"
            show-word-limit
          />
        </el-form-item>
        
        <el-form-item label="群组描述" prop="description">
          <el-input
            v-model="createForm.description"
            type="textarea"
            placeholder="请输入群组描述（可选）"
            maxlength="200"
            show-word-limit
            :rows="3"
          />
        </el-form-item>
        
        <el-form-item label="群组头像" prop="avatar">
          <el-input
            v-model="createForm.avatar"
            placeholder="请输入头像URL（可选）"
          />
        </el-form-item>
      </el-form>
      
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="showCreateDialog = false">取消</el-button>
          <el-button
            type="primary"
            @click="createGroup"
            :loading="creating"
          >
            创建
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Loading, ChatDotRound, ChatRound, User, Right, Plus, MoreFilled, Delete, Close } from '@element-plus/icons-vue'
import groupStore from '../stores/groupStore'
import * as groupApi from '../utils/groupApi'

const router = useRouter()
const defaultAvatar = 'https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png'

// 创建群组相关状态
const showCreateDialog = ref(false)
const creating = ref(false)
const createFormRef = ref(null)
const createForm = reactive({
  name: '',
  description: '',
  avatar: ''
})

// 表单验证规则
const createFormRules = {
  name: [
    { required: true, message: '请输入群组名称', trigger: 'blur' },
    { min: 1, max: 50, message: '群组名称长度在1到50个字符', trigger: 'blur' }
  ]
}

// 获取完整头像URL
const getFullAvatarUrl = (avatar) => {
  if (!avatar) return defaultAvatar
  if (avatar.startsWith('http')) return avatar
  return `http://localhost:8080${avatar}`
}

// 进入群聊
const enterGroup = (group) => {
  const groupName = group.name || group.Name
  const groupId = group.id || group.ID
  ElMessage.info(`正在进入群组: ${groupName}`)
  router.push(`/group/${groupId}`)
}

// 跳转到全局聊天
const goToChat = () => {
  router.push('/chat')
}

// 跳转到个人资料页面
const goToProfile = () => {
  router.push('/profile')
}

// 退出登录
const logout = () => {
  localStorage.removeItem('token')
  localStorage.removeItem('username')
  groupStore.actions.resetState()
  router.push('/')
}

// 创建群组
const createGroup = async () => {
  if (!createFormRef.value) return
  
  try {
    // 验证表单
    await createFormRef.value.validate()
    
    creating.value = true
    
    // 调用群组store的创建方法
    await groupStore.actions.createGroup({
      name: createForm.name,
      description: createForm.description || undefined,
      avatar: createForm.avatar || undefined
    })
    
    ElMessage.success('群组创建成功')
    
    // 关闭对话框和清空表单
    showCreateDialog.value = false
    resetCreateForm()
    
    // 刷新群组列表
    await groupStore.actions.fetchUserGroups()
    
  } catch (error) {
    console.error('创建群组失败:', error)
    // 错误信息已在store中处理
  } finally {
    creating.value = false
  }
}

// 重置创建表单
const resetCreateForm = () => {
  if (createFormRef.value) {
    createFormRef.value.clearValidate()
  }
  Object.assign(createForm, {
    name: '',
    description: '',
    avatar: ''
  })
}

// 判断是否为群主
const isGroupOwner = (group) => {
  // 从 localStorage 获取当前用户信息
  const token = localStorage.getItem('token')
  if (!token) return false
  
  try {
    const payload = JSON.parse(atob(token.split('.')[1]))
    const currentUserId = payload.userID || 0
    return group.OwnerID === currentUserId || group.owner_id === currentUserId
  } catch (error) {
    console.error('解析用户ID失败:', error)
    return false
  }
}

// 处理群组操作
const handleGroupAction = (command, group) => {
  const groupId = group.ID || group.id
  const groupName = group.Name || group.name
  
  if (command === 'leave') {
    // 退出群组
    ElMessageBox.confirm(
      `确定要退出群组"${groupName}"吗？退出后将无法接收群消息。`,
      '退出群组',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    ).then(async () => {
      try {
        await groupApi.leaveGroup(groupId)
        ElMessage.success('已退出群组')
        // 刷新群组列表
        await groupStore.actions.fetchUserGroups()
      } catch (error) {
        ElMessage.error(error?.error || '退出群组失败')
      }
    }).catch(() => {
      // 用户取消操作
    })
  } else if (command === 'delete') {
    // 解散群组
    ElMessageBox.confirm(
      `确定要解散群组"${groupName}"吗？解散后所有成员都将离开群组，此操作不可恢复。`,
      '解散群组',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'error',
      }
    ).then(async () => {
      try {
        await groupApi.deleteGroup(groupId)
        ElMessage.success('群组已解散')
        // 刷新群组列表
        await groupStore.actions.fetchUserGroups()
      } catch (error) {
        ElMessage.error(error?.error || '解散群组失败')
      }
    }).catch(() => {
      // 用户取消操作
    })
  }
}

// 生命周期钩子
onMounted(() => {
  // 加载用户群组列表
  groupStore.actions.fetchUserGroups()
})
</script>

<style scoped>
.groups-container {
  min-height: 100vh;
  background-color: #f5f5f5;
  padding: 20px;
}

.groups-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 30px;
  padding: 20px;
  background-color: white;
  border-radius: 8px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
}

.groups-header h1 {
  margin: 0;
  color: #333;
  font-size: 24px;
}

.header-actions {
  display: flex;
  gap: 10px;
}

.groups-content {
  background-color: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
  min-height: 400px;
}

.loading-indicator {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 50px;
  color: #666;
  flex-direction: column;
  gap: 10px;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 50px;
  color: #999;
  text-align: center;
}

.empty-state .el-icon {
  font-size: 64px;
  margin-bottom: 16px;
  color: #ddd;
}

.empty-state h3 {
  margin: 0 0 8px 0;
  font-size: 18px;
  color: #666;
}

.empty-state p {
  margin: 0;
  color: #999;
}

.groups-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 20px;
}

.group-card {
  display: flex;
  align-items: center;
  padding: 20px;
  border: 1px solid #e8e8e8;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.3s ease;
  background-color: #fafafa;
}

.group-card:hover {
  border-color: #1890ff;
  box-shadow: 0 4px 12px rgba(24, 144, 255, 0.15);
  transform: translateY(-2px);
}

.group-avatar {
  margin-right: 16px;
  flex-shrink: 0;
}

.group-avatar img {
  width: 50px;
  height: 50px;
  border-radius: 8px;
  object-fit: cover;
}

.default-avatar {
  width: 50px;
  height: 50px;
  border-radius: 8px;
  background: linear-gradient(45deg, #1890ff, #52c41a);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 24px;
}

.group-info {
  flex: 1;
  min-width: 0;
}

.group-name {
  margin: 0 0 8px 0;
  font-size: 16px;
  font-weight: bold;
  color: #333;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.group-description {
  margin: 0 0 8px 0;
  font-size: 12px;
  color: #666;
  line-height: 1.4;
  max-height: 2.8em;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.group-meta {
  display: flex;
  align-items: center;
  gap: 12px;
}

.member-count {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: #999;
}

.group-actions {
  margin-left: 16px;
  flex-shrink: 0;
}

@media (max-width: 768px) {
  .groups-container {
    padding: 10px;
  }
  
  .groups-header {
    flex-direction: column;
    gap: 16px;
    text-align: center;
  }
  
  .groups-grid {
    grid-template-columns: 1fr;
  }
  
  .group-card {
    flex-direction: column;
    text-align: center;
  }
  
  .group-avatar {
    margin-right: 0;
    margin-bottom: 12px;
  }
  
  .group-actions {
    margin-left: 0;
    margin-top: 12px;
  }
}

/* 下拉菜单项样式 */
:deep(.danger-item) {
  color: #f56c6c;
}

:deep(.danger-item:hover) {
  color: #f56c6c;
  background-color: #fef0f0;
}

:deep(.warning-item) {
  color: #e6a23c;
}

:deep(.warning-item:hover) {
  color: #e6a23c;
  background-color: #fdf6ec;
}
</style>
