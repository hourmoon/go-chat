import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import '@/utils/request'  // 初始化 axios 拦截器

const app = createApp(App)

app.use(router)
app.use(ElementPlus)
app.mount('#app')
