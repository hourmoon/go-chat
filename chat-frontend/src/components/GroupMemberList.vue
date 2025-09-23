<template>
  <div class="group-member-list">
    <div class="member-header">
      <h4>群成员 ({{ members.length }})</h4>
      <el-button 
        @click="showInviteDialog = true" 
        type="primary" 
        size="small"
        :disabled="!canInvite"
      >
        <el-icon><Plus /></el-icon>
        添加成员
      </el-button>
    </div>
    
    <div class="member-list">
      <div 
        v-for="member in members" 
        :key="getUserId(member)"
        class="member-item"
      >
        <div class="member-avatar">
          <img 
            :src="getUserAvatar(member)" 
            :alt="getUserName(member)"
          />
          <span :class="['member-status', isOnline(getUserId(member)) ? 'online' : 'offline']"></span>
        </div>
        <div class="member-info">
          <div class="member-name">{{ getUserName(member) }}</div>
          <div class="member-role">
            <el-tag 
              :type="getRoleType(member.Role)" 
              size="small"
            >
              {{ getRoleText(member.Role) }}
            </el-tag>
          </div>
          <div class="member-join-time">
            {{ formatJoinTime(member.JoinedAt) }}
          </div>
        </div>
        <div class="member-actions" v-if="showActions(member)">
          <!-- 角色切换按钮（仅群主可见） -->
          <div v-if="currentUserRole === 'owner' && member.Role !== 'owner'">
            <el-button
              v-if="member.Role === 'member'"
              @click="changeRole(member, 'admin')"
              type="warning"
              size="small"
            >
              设为管理员
            </el-button>
            <el-button
              v-else-if="member.Role === 'admin'"
              @click="changeRole(member, 'member')"
              type="info"
              size="small"
            >
              取消管理员
            </el-button>
            
            <!-- 转让群主按钮 -->
            <el-button
              @click="confirmTransferOwner(member)"
              type="success"
              size="small"
              style="margin-left: 8px;"
            >
              转让群主
            </el-button>
          </div>
          
          <!-- 移除成员按钮 -->
          <el-button 
            v-if="canRemoveMember(member)"
            @click="confirmRemoveMember(member)" 
            type="danger" 
            size="small"
            :disabled="member.Role === 'owner'"
            :style="currentUserRole === 'owner' && member.Role !== 'owner' ? 'margin-left: 8px;' : ''"
          >
            移除
          </el-button>
        </div>
      </div>
    </div>

    <!-- 邀请成员对话框 -->
    <el-dialog
      v-model="showInviteDialog"
      title="邀请新成员"
      width="400px"
    >
      <el-form :model="inviteForm" label-width="80px">
        <el-form-item label="用户ID">
          <el-input
            v-model="inviteForm.userId"
            placeholder="请输入要邀请的用户ID"
            type="number"
          />
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="inviteForm.role" placeholder="选择角色">
            <el-option label="普通成员" value="member" />
            <el-option label="管理员" value="admin" v-if="currentUserRole === 'owner'" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="showInviteDialog = false">取消</el-button>
          <el-button 
            type="primary" 
            @click="inviteMember"
            :loading="inviting"
            :disabled="!inviteForm.userId"
          >
            邀请
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, defineProps, defineEmits } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import * as groupApi from '../utils/groupApi'

const props = defineProps({
  groupId: {
    type: Number,
    required: true
  },
  currentUserRole: {
    type: String,
    default: 'member'
  },
  onlineMembers: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['memberAdded', 'memberRemoved', 'refreshMembers'])

const members = ref([])
const showInviteDialog = ref(false)
const inviting = ref(false)
const inviteForm = ref({
  userId: '',
  role: 'member'
})

const defaultAvatar = 'https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png'

// 获取完整头像URL
const getFullAvatarUrl = (avatar) => {
  if (!avatar) return defaultAvatar
  if (avatar.startsWith('http')) return avatar
  return `http://localhost:8080${avatar}`
}

// 获取用户ID（兼容大小写字段名）
const getUserId = (member) => {
  return member.UserID || member.user_id
}

// 获取用户名（兼容大小写字段名）
const getUserName = (member) => {
  const user = member.User || member.user
  return user?.Username || user?.username || '未知用户'
}

// 获取用户头像（兼容大小写字段名）
const getUserAvatar = (member) => {
  const user = member.User || member.user
  const avatar = user?.Avatar || user?.avatar
  return avatar ? getFullAvatarUrl(avatar) : defaultAvatar
}

// 检查用户是否在线
const isOnline = (userId) => {
  return props.onlineMembers.some(member => member.id === userId)
}

// 获取角色标签类型
const getRoleType = (role) => {
  const roleTypes = {
    'owner': 'danger',
    'admin': 'warning',
    'member': 'info'
  }
  return roleTypes[role] || 'info'
}

// 获取角色文本
const getRoleText = (role) => {
  const roleTexts = {
    'owner': '群主',
    'admin': '管理员',
    'member': '成员'
  }
  return roleTexts[role] || '成员'
}

// 格式化加入时间
const formatJoinTime = (joinTime) => {
  if (!joinTime) return ''
  const date = new Date(joinTime)
  return date.toLocaleDateString()
}

// 检查当前用户是否可以邀请成员（任何群成员都可以）
const canInvite = computed(() => {
  return props.currentUserRole && ['owner', 'admin', 'member'].includes(props.currentUserRole)
})

// 检查是否显示操作按钮
const showActions = (member) => {
  return canRemoveMember(member) || (props.currentUserRole === 'owner' && member.Role !== 'owner')
}

// 检查是否可以移除某个成员
const canRemoveMember = (member) => {
  // 只有群主和管理员可以移除成员
  if (!['owner', 'admin'].includes(props.currentUserRole)) {
    return false
  }
  
  // 群主不能被移除
  if (member.Role === 'owner') {
    return false
  }
  
  // 管理员不能移除其他管理员，只有群主可以
  if (member.Role === 'admin' && props.currentUserRole !== 'owner') {
    return false
  }
  
  return true
}

// 获取群成员列表
const fetchMembers = async () => {
  try {
    const response = await groupApi.getGroupMembers(props.groupId)
    members.value = response.members || []
  } catch (error) {
    console.error('获取群成员失败:', error)
    ElMessage.error('获取群成员失败')
  }
}

// 邀请成员
const inviteMember = async () => {
  if (!inviteForm.value.userId) {
    ElMessage.warning('请输入用户ID')
    return
  }

  inviting.value = true
  try {
    await groupApi.addGroupMember(props.groupId, {
      user_id: parseInt(inviteForm.value.userId),
      role: inviteForm.value.role
    })
    
    ElMessage.success('成员邀请成功')
    showInviteDialog.value = false
    inviteForm.value = { userId: '', role: 'member' }
    
    // 刷新成员列表
    await fetchMembers()
    emit('memberAdded')
    emit('refreshMembers')
  } catch (error) {
    console.error('邀请成员失败:', error)
    // 错误信息已在request拦截器中处理
  } finally {
    inviting.value = false
  }
}

// 确认移除成员
const confirmRemoveMember = async (member) => {
  try {
    await ElMessageBox.confirm(
      `确定要移除成员 "${getUserName(member)}" 吗？`,
      '确认移除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    await removeMember(member)
  } catch (error) {
    // 用户取消操作
    if (error === 'cancel') {
      return
    }
    console.error('移除成员失败:', error)
  }
}

// 移除成员
const removeMember = async (member) => {
  try {
    await groupApi.removeGroupMember(props.groupId, getUserId(member))
    ElMessage.success('成员移除成功')
    
    // 刷新成员列表
    await fetchMembers()
    emit('memberRemoved')
    emit('refreshMembers')
  } catch (error) {
    console.error('移除成员失败:', error)
    // 错误信息已在request拦截器中处理
  }
}

// 修改成员角色
const changeRole = async (member, newRole) => {
  try {
    await ElMessageBox.confirm(
      `确定要${newRole === 'admin' ? '设置' : '取消'}"${getUserName(member)}"为管理员吗？`,
      '确认角色变更',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    await groupApi.updateMemberRole(props.groupId, getUserId(member), newRole)
    ElMessage.success(`成员角色${newRole === 'admin' ? '设置' : '取消'}成功`)
    
    // 刷新成员列表
    await fetchMembers()
    emit('refreshMembers')
    
  } catch (error) {
    if (error === 'cancel') {
      return
    }
    console.error('修改成员角色失败:', error)
    // 错误信息已在request拦截器中处理
  }
}

// 确认转让群主
const confirmTransferOwner = async (member) => {
  try {
    await ElMessageBox.confirm(
      `确定要转让群主给"${getUserName(member)}"吗？转让后您将变为管理员，此操作不可撤销！`,
      '确认转让群主',
      {
        confirmButtonText: '确定转让',
        cancelButtonText: '取消',
        type: 'warning',
        dangerouslyUseHTMLString: false
      }
    )
    
    await groupApi.transferGroupOwnership(props.groupId, getUserId(member))
    ElMessage.success('群主转让成功')
    
    // 刷新成员列表
    await fetchMembers()
    emit('refreshMembers')
    
  } catch (error) {
    if (error === 'cancel') {
      return
    }
    console.error('转让群主失败:', error)
    // 错误信息已在request拦截器中处理
  }
}

// 生命周期
onMounted(() => {
  fetchMembers()
})

// 暴露刷新方法给父组件
defineExpose({
  fetchMembers
})
</script>

<style scoped>
.group-member-list {
  padding: 16px;
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.member-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #eee;
}

.member-header h4 {
  margin: 0;
  color: #333;
  font-size: 16px;
}

.member-list {
  max-height: 400px;
  overflow-y: auto;
}

.member-item {
  display: flex;
  align-items: center;
  padding: 12px 0;
  border-bottom: 1px solid #f5f5f5;
  transition: background-color 0.2s;
}

.member-item:hover {
  background-color: #fafafa;
  padding: 12px 8px;
  margin: 0 -8px;
  border-radius: 4px;
}

.member-item:last-child {
  border-bottom: none;
}

.member-avatar {
  position: relative;
  margin-right: 12px;
  flex-shrink: 0;
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
  width: 10px;
  height: 10px;
  border-radius: 50%;
  border: 2px solid #fff;
}

.member-status.online {
  background-color: #52c41a;
}

.member-status.offline {
  background-color: #d9d9d9;
}

.member-info {
  flex: 1;
  min-width: 0;
}

.member-name {
  font-weight: bold;
  color: #333;
  margin-bottom: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.member-role {
  margin-bottom: 4px;
}

.member-join-time {
  font-size: 12px;
  color: #999;
}

.member-actions {
  margin-left: 12px;
  flex-shrink: 0;
}

.dialog-footer {
  text-align: right;
}

@media (max-width: 768px) {
  .member-header {
    flex-direction: column;
    gap: 12px;
    text-align: center;
  }
  
  .member-item {
    flex-direction: column;
    text-align: center;
    padding: 16px 0;
  }
  
  .member-avatar {
    margin-right: 0;
    margin-bottom: 8px;
  }
  
  .member-actions {
    margin-left: 0;
    margin-top: 8px;
  }
}
</style>
