<template>
  <div style="display: flex; height: 100vh; width: 100%; margin: 0; padding: 0; background: #f5f7fa;">
    <!-- 左侧菜单栏 -->
    <div style="width: 260px; background: white; border-right: 1px solid #e5e7eb; display: flex; flex-direction: column; flex-shrink: 0; box-shadow: 2px 0 8px rgba(0,0,0,0.05);">
      <div style="padding: 20px; border-bottom: 1px solid #e5e7eb;">
        <h1 style="font-size: 20px; font-weight: bold; color: #477779; margin: 0;">UPanel</h1>
        <p style="font-size: 12px; color: #9ca3af; margin-top: 4px; margin-bottom: 0;">轻量级容器面板</p>
      </div>
      
      <nav style="flex: 1; padding: 16px 12px;">
        <div v-for="item in menuItems" :key="item.path" style="margin-bottom: 8px;">
          <router-link 
            :to="item.path"
            style="display: flex; align-items: center; gap: 12px; padding: 12px 16px; border-radius: 10px; text-decoration: none; transition: all 0.2s ease; background: white; box-shadow: 0 1px 2px rgba(0,0,0,0.05);"
            :style="$route.path === item.path || ($route.path === '/' && item.path === '/dashboard') ? { background: '#477779', color: 'white', boxShadow: '0 4px 12px rgba(71,119,121,0.3)' } : { color: '#4b5563', boxShadow: '0 1px 2px rgba(0,0,0,0.05)' }"
          >
            <el-icon style="font-size: 18px;"><component :is="item.icon" /></el-icon>
            <span style="font-size: 14px; font-weight: 500;">{{ item.name }}</span>
          </router-link>
        </div>
      </nav>
      
      <div style="padding: 16px 12px; border-top: 1px solid #e5e7eb;">
        <router-link to="/logout" style="display: flex; align-items: center; gap: 12px; padding: 12px 16px; border-radius: 10px; color: #4b5563; text-decoration: none; background: white; box-shadow: 0 1px 2px rgba(0,0,0,0.05); transition: all 0.2s ease;">
          <el-icon style="font-size: 18px;"><SwitchButton /></el-icon>
          <span style="font-size: 14px; font-weight: 500;">退出面板</span>
        </router-link>
      </div>
    </div>
    
    <!-- 右侧内容区域 -->
    <div style="flex: 1; overflow: auto; background: #f5f7fa;">
      <div style="padding: 24px;">
        <router-view />
      </div>
    </div>
  </div>
</template>

<script setup>
import { 
  Monitor, Shop, DataBoard, Folder, Setting, SwitchButton 
} from '@element-plus/icons-vue'

const menuItems = [
  { path: '/dashboard', name: '总览', icon: Monitor },
  { path: '/apps', name: '应用商店', icon: Shop },
  { path: '/websites', name: '网站', icon: DataBoard },
  { path: '/databases', name: '数据库', icon: DataBoard },
  { path: '/files', name: '文件管理', icon: Folder },
  { path: '/settings', name: '面板设置', icon: Setting },
]
</script>