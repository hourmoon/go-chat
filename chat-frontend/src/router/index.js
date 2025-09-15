import { createRouter, createWebHistory } from 'vue-router'
import Login from '../views/login.vue'
import Chat from '../views/Chat.vue'
import Profile from '../views/Profile.vue'

const routes = [
  { path: '/', name: 'Login', component: Login },
  {
    path: '/chat',
    name: 'Chat',
    component: Chat,
    meta: { requiresAuth: true } // ✅ 标记这个路由需要登录
  },
  {
    path: '/profile',
    name: 'Profile',
    component: Profile,
    meta: { requiresAuth: true }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// ✅ 添加全局守卫（关键补充）
router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('token')

  if (to.meta.requiresAuth && !token) {
    ElMessage.error('请先登录')
    next('/') // 重定向到登录页
  } else {
    next() // 允许跳转
  }
})

export default router
