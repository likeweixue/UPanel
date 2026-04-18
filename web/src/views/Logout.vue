<template>
  <div class="logout-container">
    <el-drawer
      v-model="drawerVisible"
      title="退出确认"
      direction="rtl"
      size="400px"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
      @close="handleClose"
    >
      <div class="drawer-content">
        <div class="logout-icon">
          <el-icon :size="64" color="#f59e0b"><Warning /></el-icon>
        </div>
        <div class="logout-title">确定要退出登录吗？</div>
        <div class="logout-desc">退出后需要重新登录才能继续使用面板</div>
        
        <div class="logout-info">
          <div class="info-item">
            <span class="info-label">当前用户</span>
            <span class="info-value">admin</span>
          </div>
          <div class="info-item">
            <span class="info-label">会话时长</span>
            <span class="info-value">{{ sessionDuration }}</span>
          </div>
        </div>

        <div class="drawer-footer">
          <el-button @click="cancelLogout">取消</el-button>
          <el-button type="danger" @click="confirmLogout" :loading="loggingOut">
            确认退出
          </el-button>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Warning } from '@element-plus/icons-vue'

const router = useRouter()
const drawerVisible = ref(true)
const loggingOut = ref(false)
const sessionDuration = ref('')

// 计算会话时长
const calculateSessionDuration = () => {
  const loginTime = localStorage.getItem('loginTime')
  if (loginTime) {
    const start = new Date(parseInt(loginTime))
    const now = new Date()
    const diff = Math.floor((now - start) / 1000)
    const hours = Math.floor(diff / 3600)
    const minutes = Math.floor((diff % 3600) / 60)
    const seconds = diff % 60
    
    if (hours > 0) {
      sessionDuration.value = `${hours}小时${minutes}分钟`
    } else if (minutes > 0) {
      sessionDuration.value = `${minutes}分钟${seconds}秒`
    } else {
      sessionDuration.value = `${seconds}秒`
    }
  } else {
    sessionDuration.value = '未知'
  }
}

// 确认退出
const confirmLogout = async () => {
  loggingOut.value = true
  try {
    // 模拟退出请求
    await new Promise(resolve => setTimeout(resolve, 500))
    
    // 清除本地存储
    localStorage.removeItem('token')
    localStorage.removeItem('loginTime')
    
    ElMessage.success('已安全退出')
    drawerVisible.value = false
    
    // 跳转到登录页
    setTimeout(() => {
      router.push('/login')
    }, 300)
  } catch (err) {
    ElMessage.error('退出失败: ' + err.message)
    drawerVisible.value = false
  } finally {
    loggingOut.value = false
  }
}

// 取消退出
const cancelLogout = () => {
  drawerVisible.value = false
  router.push('/')
}

// 关闭抽屉时的处理
const handleClose = () => {
  if (!loggingOut.value) {
    router.push('/')
  }
}

onMounted(() => {
  calculateSessionDuration()
})
</script>

<style scoped>
.logout-container {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.drawer-content {
  text-align: center;
  padding: 20px 0;
}

.logout-icon {
  margin-bottom: 20px;
}

.logout-title {
  font-size: 20px;
  font-weight: 600;
  color: #1f2937;
  margin-bottom: 12px;
}

.logout-desc {
  font-size: 14px;
  color: #9ca3af;
  margin-bottom: 24px;
}

.logout-info {
  background-color: #f9fafb;
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 24px;
  text-align: left;
}

.info-item {
  display: flex;
  justify-content: space-between;
  padding: 8px 0;
}

.info-item:first-child {
  padding-top: 0;
}

.info-item:last-child {
  padding-bottom: 0;
}

.info-label {
  font-size: 13px;
  color: #9ca3af;
}

.info-value {
  font-size: 13px;
  color: #1f2937;
  font-weight: 500;
}

.drawer-footer {
  display: flex;
  justify-content: center;
  gap: 16px;
  margin-top: 8px;
}
</style>

<style scoped>
/* 暗色模式支持 */
body.dark-mode .logout-container {
  background-color: #1a1a1a;
}

body.dark-mode .logout-info {
  background-color: #2a2a2a;
}

body.dark-mode .info-label {
  color: #9ca3af;
}

body.dark-mode .info-value {
  color: #e5e7eb;
}

body.dark-mode .logout-title {
  color: #e5e7eb;
}

body.dark-mode .logout-desc {
  color: #9ca3af;
}
</style>
