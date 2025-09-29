<template>
  <div class="group-list">
    <div class="group-list-header">
      <h3>我的群组</h3>
      <el-button 
        @click="showCreateDialog = true" 
        type="primary" 
        size="small"
        :disabled="loading"
      >
        <el-icon><Plus /></el-icon>
        创建群组
      </el-button>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="loading-state">
      <el-icon class="is-loading"><Loading /></el-icon>
      <span>加载群组中...</span>
    </div>

    <!-- 群组列表 -->
    <div v-else class="groups">
      <div 
        v-for="group in groupList" 
        :key="group.ID"
        class="group-item"
        :class="{ active: currentGroupId === group.ID }"
        @click="selectGroup(group)"
      >
        <div class="group-avatar">
          <img 
            :src="group.Avatar || defaultGroupAvatar" 
            :alt="group.Name"
          />
        </div>
        <div class="group-info">
          <div class="group-name">{{ group.Name }}</div>
          <div class="group-description" v-if="group.Description">
            {{ group.Description }}
          </div>
          <div class="group-meta">
            <span class="member-count">成员数量加载中...</span>
          </div>
        </div>
      </div>
      
      <!-- 空状态 -->
      <div v-if="groupList.length === 0" class="empty-state">
        <el-icon><UserGroup /></el-icon>
        <p>还没有加入任何群组</p>
        <p class="hint">点击上方按钮创建群组</p>
      </div>
    </div>

    <!-- 创建群组对话框 -->
    <el-dialog
      v-model="showCreateDialog"
      title="创建群组"
      width="400px"
      :before-close="handleCloseDialog"
    >
      <el-form
        ref="createFormRef"
        :model="createForm"
        :rules="createRules"
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
            :rows="3"
            placeholder="请输入群组描述（可选）"
            maxlength="200"
            show-word-limit
          />
        </el-form-item>
      </el-form>
      
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="showCreateDialog = false">取消</el-button>
          <el-button 
            type="primary" 
            @click="handleCreateGroup"
            :loading="creating"
          >
            创建
          </el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Loading, Avatar as UserGroup } from '@element-plus/icons-vue'
import groupStore from '../stores/groupStore'
import { getAuthToken } from '../utils/auth'

// 响应式数据
const showCreateDialog = ref(false)
const creating = ref(false)
const createFormRef = ref(null)
const defaultGroupAvatar = 'https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png'

// 创建群组表单
const createForm = ref({
  name: '',
  description: ''
})

// 表单验证规则
const createRules = {
  name: [
    { required: true, message: '请输入群组名称', trigger: 'blur' },
    { min: 2, max: 50, message: '群组名称长度在 2 到 50 个字符', trigger: 'blur' }
  ]
}

// 计算属性
const groupList = computed(() => groupStore.state.groupList)
const currentGroupId = computed(() => groupStore.state.currentGroupId)
const loading = computed(() => groupStore.state.loading.groupList)

// 方法
const selectGroup = (group) => {
  groupStore.actions.selectGroup(group.ID)
}

const handleCreateGroup = async () => {
  if (!createFormRef.value) return
  
  try {
    await createFormRef.value.validate()
    creating.value = true
    
    await groupStore.actions.createGroup({
      name: createForm.value.name,
      description: createForm.value.description
    })
    
    // 重置表单并关闭对话框
    createForm.value = { name: '', description: '' }
    showCreateDialog.value = false
    createFormRef.value.resetFields()
    
  } catch (error) {
    if (error !== false) { // 不是表单验证失败
      console.error('创建群组失败:', error)
    }
  } finally {
    creating.value = false
  }
}

const handleCloseDialog = (done) => {
  if (creating.value) {
    ElMessageBox.confirm('正在创建群组，确定要关闭吗？')
      .then(() => done())
      .catch(() => {})
  } else {
    createForm.value = { name: '', description: '' }
    if (createFormRef.value) {
      createFormRef.value.resetFields()
    }
    done()
  }
}

// 生命周期
onMounted(async () => {
  try {
    // 使用认证管理器等待token就绪
    const { waitForAuth } = await import('../utils/auth.js')
    const isAuthenticated = await waitForAuth()
    
    if (isAuthenticated) {
      console.log('🔐 认证就绪，开始加载群组列表')
      groupStore.actions.fetchUserGroups()
    } else {
      console.warn('⚠️ 认证失败，跳过群组列表加载')
    }
  } catch (error) {
    console.error('❌ 认证检查失败:', error)
    // 降级处理：使用 auth.js 方式
    const token = await getAuthToken()
    if (token) {
      groupStore.actions.fetchUserGroups()
    }
  }
})
</script>

<style scoped>
.group-list {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.group-list-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 0 15px 0;
  margin-bottom: 15px;
  border-bottom: 1px solid #eee;
}

.group-list-header h3 {
  margin: 0;
  color: #333;
}

.loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  color: #666;
}

.loading-state .el-icon {
  margin-right: 8px;
}

.groups {
  flex: 1;
  overflow-y: auto;
}

.group-item {
  display: flex;
  align-items: center;
  padding: 12px;
  border-radius: 8px;
  cursor: pointer;
  transition: background-color 0.2s;
  margin-bottom: 8px;
}

.group-item:hover {
  background-color: #f0f0f0;
}

.group-item.active {
  background-color: #e3f2fd;
  border: 1px solid #1890ff;
}

.group-avatar {
  margin-right: 12px;
}

.group-avatar img {
  width: 45px;
  height: 45px;
  border-radius: 8px;
  object-fit: cover;
}

.group-info {
  flex: 1;
  min-width: 0;
}

.group-name {
  font-weight: bold;
  color: #333;
  margin-bottom: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.group-description {
  font-size: 12px;
  color: #666;
  margin-bottom: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.group-meta {
  font-size: 11px;
  color: #999;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  text-align: center;
  color: #999;
}

.empty-state .el-icon {
  font-size: 48px;
  margin-bottom: 16px;
  color: #ddd;
}

.empty-state p {
  margin: 4px 0;
}

.empty-state .hint {
  font-size: 12px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
