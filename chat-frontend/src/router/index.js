import { createRouter, createWebHistory } from 'vue-router'
import Login from '../views/login.vue'
import Chat from '../views/Chat.vue'
import Profile from '../views/Profile.vue'
import Groups from '../views/Groups.vue'
import GroupChat from '../views/GroupChat.vue'
import { ElMessage } from 'element-plus'
import { getAuthToken } from '../utils/auth'
import { useUserStore } from '../stores/userStore'

const routes = [
  { path: '/', name: 'Login', component: Login },
  {
    path: '/chat',
    name: 'Chat',
    component: Chat,
    meta: { requiresAuth: true }
  },
  {
    path: '/profile',
    name: 'Profile',
    component: Profile,
    meta: { requiresAuth: true }
  },
  {
    path: '/groups',
    name: 'Groups',
    component: Groups,
    meta: { requiresAuth: true }
  },
  {
    path: '/group/:groupId',
    name: 'GroupChat',
    component: GroupChat,
    meta: { requiresAuth: true }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// ✅ 全局路由守卫
router.beforeEach(async (to, from, next) => {
  const token = await getAuthToken()

  if (to.meta.requiresAuth && !token) {
    ElMessage.error('请先登录')
    next('/')
  } else {
    // 已登录且跳转到受保护路由时，预加载用户信息
    if (token && to.meta.requiresAuth) {
      const userStore = useUserStore()
      if (!userStore.profile.id) {
        userStore.fetchProfile().catch(() => {})
      }
    }
    next()
  }
})

export default router
