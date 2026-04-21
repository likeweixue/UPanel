<template>
  <div class="app-container">
    <aside class="sidebar">
      <div class="logo-area"><h1 class="logo">UPanel</h1></div>
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
            <div class="submenu-item" :class="{ active: $route.path === '/files' }" @click="navigate('/files')">文件</div>
            <div class="submenu-item" :class="{ active: $route.path === '/monitor' }" @click="navigate('/monitor')">监控</div>
            <div class="submenu-item" :class="{ active: $route.path === '/ssh' }" @click="navigate('/ssh')">SSH</div>
            <div class="submenu-item" :class="{ active: $route.path === '/security' }" @click="navigate('/security')">安全</div>
          </div>
        </div>

        <!-- 工具箱子菜单 -->
        <div class="menu-group">
          <div class="menu-item" :class="{ active: isToolboxActive }" @click="toggleToolboxMenu">
            <el-icon class="menu-icon"><Tools /></el-icon>
            <span class="menu-text">工具箱</span>
            <el-icon class="menu-arrow"><ArrowDown v-if="!showToolboxMenu" /><ArrowUp v-else /></el-icon>
          </div>
          <div v-show="showToolboxMenu" class="submenu">
            <div class="submenu-item" :class="{ active: $route.path === '/email-notify' }" @click="navigate('/email-notify')">邮件通知</div>
          </div>
        </div>
        
        <div v-for="item in bottomMenus" :key="item.path" class="menu-item" :class="{ active: isActive(item.path) }" @click="navigate(item.path)">
          <el-icon class="menu-icon"><component :is="item.icon" /></el-icon>
          <span class="menu-text">{{ item.name }}</span>
        </div>
      </nav>
    </aside>
    
    <div class="main-area">
      <header class="top-bar">
        <h2 class="page-title">{{ currentTitle }}</h2>
        <div class="user-info">
          <el-dropdown @command="handleCommand">
            <span class="user-name">{{ username }} <el-icon><ArrowDown /></el-icon></span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="settings">面板设置</el-dropdown-item>
                <el-dropdown-item command="logout" divided>退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </header>
      
      <main class="content-area"><router-view /></main>
      
      <footer class="footer">
        <span>UPanel 开源容器面板</span>
        <a href="https://github.com/likeweixue/UPanel" target="_blank" class="footer-link">GitHub</a>
        <span class="footer-version">v0.1.3</span>
      </footer>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Monitor, DataBoard, Shop, Setting, SwitchButton, ArrowDown, ArrowUp, Grid, Coin, Calendar, Odometer, Tools } from '@element-plus/icons-vue'

const router = useRouter()
const route = useRoute()
const username = ref('admin')
const showHostMenu = ref(false)
const showToolboxMenu = ref(false)

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
  if (found) return found.name
  if (route.path === '/monitor') return '监控'
  if (route.path === '/files') return '文件管理'
  if (route.path === '/ssh') return 'SSH管理'
  if (route.path === '/security') return '安全设置'
  if (route.path === '/email-notify') return '邮件通知'
  return '总览'
})

const isActive = (path) => route.path.includes(path)
const isHostActive = computed(() => ['/files', '/monitor', '/ssh', '/security'].includes(route.path))
const isToolboxActive = computed(() => ['/email-notify'].includes(route.path))

const toggleHostMenu = () => { showHostMenu.value = !showHostMenu.value }
const toggleToolboxMenu = () => { showToolboxMenu.value = !showToolboxMenu.value }
const navigate = (path) => { router.push(path) }
const handleCommand = (command) => {
  if (command === 'settings') router.push('/settings')
  else if (command === 'logout') router.push('/logout')
}
</script>

<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
html, body, #app { width: 100%; height: 100%; overflow: hidden; }
</style>

<style scoped>
.app-container { display: flex; width: 100%; height: 100%; background-color: #f1f1f1; overflow: hidden; }
.sidebar { width: 180px; background-color: #ffffff; display: flex; flex-direction: column; flex-shrink: 0; height: 100%; overflow-y: auto; border-right: 1px solid #e5e7eb; }
.logo-area { padding: 20px 16px; text-align: center; border-bottom: 1px solid #e5e7eb; }
.logo { font-size: 18px; font-weight: bold; color: #477779; margin: 0; }
.menu { flex: 1; padding: 12px 8px; }
.menu-item { display: flex; align-items: center; padding: 10px 12px; margin-bottom: 2px; border-radius: 6px; cursor: pointer; color: #4b5563; transition: all 0.2s; }
.menu-item:hover { background-color: #f3f4f6; }
.menu-item.active { background-color: #477779; color: white; }
.menu-icon { font-size: 18px; margin-right: 10px; }
.menu-text { font-size: 13px; font-weight: 500; flex: 1; }
.menu-arrow { font-size: 12px; margin-left: 4px; }
.submenu { padding-left: 40px; margin-bottom: 4px; }
.submenu-item { padding: 8px 12px; margin-bottom: 2px; border-radius: 4px; cursor: pointer; font-size: 12px; color: #6b7280; }
.submenu-item:hover { background-color: #f3f4f6; color: #477779; }
.submenu-item.active { background-color: #e8f4f4; color: #477779; font-weight: 500; }
.main-area { flex: 1; display: flex; flex-direction: column; height: 100%; min-width: 0; overflow: hidden; }
.top-bar { background-color: #ffffff; padding: 12px 24px; display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid #e5e7eb; flex-shrink: 0; }
.page-title { font-size: 18px; font-weight: 600; color: #1f2937; margin: 0; }
.user-name { cursor: pointer; display: flex; align-items: center; gap: 4px; color: #4b5563; font-size: 13px; }
.content-area { flex: 1; overflow-y: auto; padding: 20px; min-height: 0; }
.footer { background-color: #ffffff; padding: 8px 24px; display: flex; justify-content: space-between; align-items: center; font-size: 11px; color: #9ca3af; border-top: 1px solid #e5e7eb; flex-shrink: 0; }
.footer-link { color: #477779; text-decoration: none; }
.footer-link:hover { text-decoration: underline; }
.footer-version { margin-left: auto; }

.el-button--small { height: 28px !important; line-height: 28px !important; padding: 0 12px !important; font-size: 12px !important; }
.el-input__wrapper { height: 28px !important; padding: 0 8px !important; }
.el-tabs__item { height: 32px !important; line-height: 32px !important; font-size: 13px !important; }
</style>
