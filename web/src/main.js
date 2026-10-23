import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import { initTheme } from './composables/useTheme'
import { useAuth } from './composables/useAuth'

import './styles/tokens.css'
import './styles/base.css'
import './styles/components.css'
import './styles/layout.css'

initTheme()

// API 返回 401 时统一清理会话并回到登录页
window.addEventListener('upanel:unauthorized', () => {
  const { clear } = useAuth()
  clear()
  const current = router.currentRoute.value
  if (current.path !== '/login') {
    router.replace({ path: '/login', query: { redirect: current.fullPath } })
  }
})

const app = createApp(App)
app.use(router)
app.mount('#app')
