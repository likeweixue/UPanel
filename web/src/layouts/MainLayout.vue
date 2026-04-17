<template>
  <div class="app-container">
    <!-- a 面板：左侧菜单 -->
    <aside class="sidebar">
      <div class="logo-area">
        <h1 class="logo">UPanel</h1>
      </div>
      
      <nav class="menu">
        <div v-for="item in mainMenus" :key="item.path" class="menu-item" :class="{ active: isActive(item.path) }" @click="navigate(item.path)">
          <el-icon class="menu-icon"><component :is="item.icon" /></el-icon>
          <span class="menu-text">{{ item.name }}</span>
        </div>
        
        <!-- 主机子菜单 -->
        <div class="menu-group">
          <div class="menu-item" :class="{ active: isHostActive }" @click="toggleHostMenu">
            <el-icon class="menu-icon"><Monitor /></el-icon>
            <span class="menu-text">主机</span>
            <el-icon class="menu-arrow"><ArrowDown v-if="!showHostMenu" /><ArrowUp v-else /></el-icon>
          </div>
          <div v-show="showHostMenu" class="submenu">
            <div class="submenu-item" :class="{ active: $route.path === '/files' }" @click="navigate('/files')">
              文件
            </div>
            <div class="submenu-item" :class="{ active: $route.path === '/ssh' }" @click="navigate('/ssh')">
              SSH
            </div>
            <div class="submenu-item" :class="{ active: $route.path === '/security' }" @click="navigate('/security')">
              安全
            </div>
          </div>
        </div>
        
        <div v-for="item in bottomMenus" :key="item.path" class="menu-item" :class="{ active: isActive(item.path) }" @click="navigate(item.path)">
          <el-icon class="menu-icon"><component :is="item.icon" /></el-icon>
          <span class="menu-text">{{ item.name }}</span>
        </div>
      </nav>
    </aside>
    
    <!-- 右侧主内容区 -->
    <div class="main-area">
      <header class="top-bar">
        <h2 class="page-title">{{ currentTitle }}</h2>
        <div class="user-info">
          <el-dropdown @command="handleCommand">
            <span class="user-name">
              {{ username }} <el-icon><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="settings">面板设置</el-dropdown-item>
                <el-dropdown-item command="logout" divided>退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </header>
      
      <main class="content-area">
        <router-view />
      </main>
      
      <footer class="footer">
        <span>UPanel 开源容器面板</span>
        <a href="https://github.com/likeweixue/UPanel" target="_blank" class="footer-link">GitHub</a>
        <span class="footer-version">v0.1.0</span>
      </footer>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { 
  Monitor, DataBoard, Shop, Folder, 
  Setting, SwitchButton, ArrowDown, ArrowUp,
  Grid, Coin, Calendar, Odometer
} from '@element-plus/icons-vue'

const router = useRouter()
const route = useRoute()
const username = ref('admin')
const showHostMenu = ref(false)

const mainMenus = [
  { path: '/dashboard', name: '总览', icon: Odometer },
  { path: '/apps', name: '应用商店', icon: Shop },
  { path: '/websites', name: '网站', icon: Grid },
  { path: '/databases', name: '数据库', icon: Coin },
  { path: '/containers', name: '容器', icon: DataBoard }
]

const bottomMenus = [
  { path: '/cron', name: '计划任务', icon: Calendar },
  { path: '/settings', name: '面板设置', icon: Setting },
  { path: '/logout', name: '退出', icon: SwitchButton }
]

const currentTitle = computed(() => {
  const allMenus = [...mainMenus, ...bottomMenus]
  const found = allMenus.find(m => route.path.includes(m.path))
  return found ? found.name : '总览'
})

const isActive = (path) => {
  return route.path.includes(path)
}

const isHostActive = computed(() => {
  return route.path === '/files' || route.path === '/ssh' || route.path === '/security'
})

const toggleHostMenu = () => {
  showHostMenu.value = !showHostMenu.value
}

const navigate = (path) => {
  router.push(path)
}

const handleCommand = (command) => {
  if (command === 'settings') {
    router.push('/settings')
  } else if (command === 'logout') {
    router.push('/logout')
  }
}
</script>

<style scoped>
.app-container {
  display: flex;
  height: 100vh;
  background-color: #f1f1f1;
  margin: 0;
  padding: 0;
}

/* a 面板：左侧菜单 - 宽度再缩短 1/3（从180px减到120px） */
.sidebar {
  width: 120px;
  background-color: #ffffff;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  border-right: 1px solid #e5e7eb;  /* 添加分隔线 */
}

.logo-area {
  padding: 16px 12px;
  text-align: center;
  border-bottom: 1px solid #e5e7eb;
}

.logo {
  font-size: 16px;
  font-weight: bold;
  color: #477779;
  margin: 0;
}

.menu {
  flex: 1;
  padding: 8px 6px;
  overflow-y: auto;
}

.menu-item {
  display: flex;
  align-items: center;
  padding: 8px 8px;
  margin-bottom: 2px;
  border-radius: 6px;
  cursor: pointer;
  color: #4b5563;
  transition: all 0.2s;
}

.menu-item:hover {
  background-color: #f3f4f6;
}

.menu-item.active {
  background-color: #477779;
  color: white;
}

.menu-icon {
  font-size: 16px;
  margin-right: 8px;
}

.menu-text {
  font-size: 12px;
  font-weight: 500;
  flex: 1;
}

.menu-arrow {
  font-size: 10px;
  margin-left: 2px;
}

.menu-group {
  margin-bottom: 2px;
}

.submenu {
  padding-left: 30px;
  margin-bottom: 4px;
}

.submenu-item {
  padding: 6px 8px;
  margin-bottom: 2px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 11px;
  color: #6b7280;
}

.submenu-item:hover {
  background-color: #f3f4f6;
  color: #477779;
}

.submenu-item.active {
  background-color: #e8f4f4;
  color: #477779;
  font-weight: 500;
}

/* 右侧区域 */
.main-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* b 面板：顶部导航栏 */
.top-bar {
  background-color: #ffffff;
  padding: 12px 24px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-bottom: 1px solid #e5e7eb;
}

.page-title {
  font-size: 18px;
  font-weight: 600;
  color: #1f2937;
  margin: 0;
}

.user-name {
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 4px;
  color: #4b5563;
  font-size: 13px;
}

/* 内容区域 */
.content-area {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

/* c 面板：底部页脚 */
.footer {
  background-color: #ffffff;
  padding: 8px 24px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 11px;
  color: #9ca3af;
  border-top: 1px solid #e5e7eb;
}

.footer-link {
  color: #477779;
  text-decoration: none;
}

.footer-link:hover {
  text-decoration: underline;
}

.footer-version {
  margin-left: auto;
}
</style>
