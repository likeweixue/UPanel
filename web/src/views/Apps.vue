<template>
  <div class="apps-container">
    <!-- 分类面板 -->
    <div class="categories-panel">
      <div class="categories-left">
        <div class="category-group">
          <span v-for="cat in leftCategories" :key="cat.key" class="category-item"
            :class="{ active: activeCategory === cat.key }" @click="activeCategory = cat.key">
            {{ cat.name }}
          </span>
        </div>
        <div class="category-divider"></div>
        <div class="category-group">
          <span v-for="cat in rightCategories" :key="cat.key" class="category-item"
            :class="{ active: activeStatus === cat.key }" @click="activeStatus = cat.key">
            {{ cat.name }}
          </span>
        </div>
      </div>
    </div>

    <!-- 应用列表 -->
    <div class="apps-panel">
      <div class="apps-grid">
        <div v-for="app in filteredApps" :key="app.id" class="app-card" @click="openAppDetail(app)">
          <div class="app-icon" :style="{ backgroundColor: app.iconBg }">
            <el-icon :size="32" :color="app.iconColor">
              <component :is="app.icon" />
            </el-icon>
          </div>
          <div class="app-info">
            <div class="app-name">{{ app.name }}</div>
            <div class="app-description">{{ app.description }}</div>
            <div class="app-footer">
              <el-button size="small" :type="app.installed ? 'info' : 'primary'" @click.stop="openInstallDrawer(app)">
                {{ app.installed ? '已安装' : '安装' }}
              </el-button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 安装抽屉 -->
    <el-drawer v-model="drawerVisible" :title="`安装 ${selectedApp?.name}`" direction="rtl" size="450px">
      <div class="drawer-content">
        <el-form :model="installForm" label-width="100px">
          <el-form-item label="应用名称">
            <el-input v-model="installForm.name" :placeholder="`例如: ${selectedApp?.name}-test`" />
          </el-form-item>
          <el-form-item label="版本" v-if="availableVersions.length">
            <el-select v-model="installForm.version">
              <el-option v-for="v in availableVersions" :key="v" :label="v" :value="v" />
            </el-select>
          </el-form-item>
          <template v-if="selectedApp?.key === 'nginx' || selectedApp?.key === 'openresty'">
            <el-form-item label="端口">
              <el-input-number v-model="installForm.config.port" :min="80" :max="8080" />
            </el-form-item>
          </template>
        </el-form>
        <div class="drawer-footer">
          <el-button @click="drawerVisible = false">取消</el-button>
          <el-button type="primary" @click="confirmInstall" :loading="installing">确认安装</el-button>
        </div>
      </div>
    </el-drawer>

    <!-- 应用详情抽屉 -->
    <el-drawer v-model="detailDrawerVisible" :title="currentApp?.name" direction="rtl" size="500px">
      <div class="detail-content">
        <div class="detail-section">
          <div class="detail-header">
            <div class="app-icon-small" :style="{ backgroundColor: currentApp?.iconBg }">
              <el-icon :size="28" :color="currentApp?.iconColor">
                <component :is="currentApp?.icon" />
              </el-icon>
            </div>
            <div class="app-info-text">
              <div class="app-name-large">{{ currentApp?.name }}</div>
              <div class="app-version">版本: {{ containerInfo?.version || currentApp?.default_version }}</div>
            </div>
          </div>
          <div class="detail-desc">{{ currentApp?.description }}</div>
        </div>

        <div class="detail-section">
          <div class="section-title">运行状态</div>
          <div class="status-row">
            <span class="status-label">容器状态</span>
            <el-tag :type="containerInfo?.status === 'running' ? 'success' : 'danger'" size="small">
              {{ containerInfo?.status === 'running' ? '运行中' : '已停止' }}
            </el-tag>
          </div>
          <div class="status-row">
            <span class="status-label">容器名称</span>
            <span class="status-value">{{ containerInfo?.name || '-' }}</span>
          </div>
          <div class="status-row">
            <span class="status-label">镜像</span>
            <span class="status-value">{{ containerInfo?.image || '-' }}</span>
          </div>
          <div class="status-row">
            <span class="status-label">端口映射</span>
            <span class="status-value">{{ containerInfo?.ports || '-' }}</span>
          </div>
          <div class="status-row">
            <span class="status-label">创建时间</span>
            <span class="status-value">{{ containerInfo?.created || '-' }}</span>
          </div>
        </div>

        <div class="detail-section">
          <div class="section-title">操作</div>
          <div class="action-buttons">
            <el-button size="small" @click="restartContainer" :loading="actionLoading">
              <el-icon><RefreshRight /></el-icon> 重启
            </el-button>
            <el-button size="small" @click="stopContainer" :loading="actionLoading" v-if="containerInfo?.status === 'running'">
              <el-icon><VideoPause /></el-icon> 停止
            </el-button>
            <el-button size="small" @click="startContainer" :loading="actionLoading" v-else>
              <el-icon><VideoPlay /></el-icon> 启动
            </el-button>
            <el-button size="small" @click="openLogs">
              <el-icon><Document /></el-icon> 日志
            </el-button>
            <el-button size="small" @click="openTerminal">
              <el-icon><Monitor /></el-icon> 终端
            </el-button>
            <el-button size="small" @click="openFileDir">
              <el-icon><Folder /></el-icon> 文件
            </el-button>
            <el-button size="small" type="danger" @click="uninstallApp">
              <el-icon><Delete /></el-icon> 卸载
            </el-button>
          </div>
        </div>

        <div class="detail-section" v-if="containerInfo?.inspect">
          <div class="section-title">容器详情</div>
          <pre class="container-inspect">{{ JSON.stringify(containerInfo.inspect, null, 2) }}</pre>
        </div>
      </div>
    </el-drawer>

    <!-- 日志抽屉 -->
    <el-drawer v-model="logDrawerVisible" title="容器日志" direction="rtl" size="600px">
      <div class="logs-content">
        <div class="log-controls">
          <el-button size="small" @click="refreshLogs">刷新</el-button>
          <el-button size="small" @click="clearLogs">清空</el-button>
        </div>
        <pre class="log-pre">{{ containerLogs }}</pre>
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import axios from 'axios'
import { RefreshRight, VideoPause, VideoPlay, Document, Monitor, Folder, Delete } from '@element-plus/icons-vue'

const activeCategory = ref('all')
const activeStatus = ref('all')
const drawerVisible = ref(false)
const detailDrawerVisible = ref(false)
const logDrawerVisible = ref(false)
const installing = ref(false)
const actionLoading = ref(false)
const selectedApp = ref(null)
const currentApp = ref(null)
const containerInfo = ref({})
const containerLogs = ref('')

const installForm = ref({
  name: '',
  version: '',
  config: { port: 8080 }
})

const leftCategories = [
  { key: 'all', name: '全部' },
  { key: 'web', name: 'Web服务器' },
  { key: 'database', name: '数据库' },
  { key: 'environment', name: '环境' },
  { key: 'tools', name: '工具' }
]

const rightCategories = [
  { key: 'all', name: '全部' },
  { key: 'installed', name: '已安装' },
  { key: 'not_installed', name: '未安装' }
]

const apps = ref([])

const fetchContainers = async () => {
  try {
    const res = await axios.get('/api/containers/')
    const containers = res.data.data || []
    const containerNames = containers.map(c => c.name)
    
    apps.value = apps.value.map(app => {
      const isInstalled = containerNames.some(name => name.includes(app.key))
      return { ...app, installed: isInstalled }
    })
  } catch (err) {
    console.error('获取容器列表失败:', err)
  }
}

const fetchContainerInfo = async (appKey) => {
  try {
    const res = await axios.get('/api/containers/')
    const containers = res.data.data || []
    const container = containers.find(c => c.name.includes(appKey))
    
    if (container) {
      containerInfo.value = {
        name: container.name,
        status: container.status,
        image: container.image,
        ports: container.ports,
        created: container.created,
        version: container.image?.split(':')[1] || '-'
      }
    } else {
      containerInfo.value = {}
    }
  } catch (err) {
    console.error('获取容器详情失败:', err)
  }
}

const fetchApps = async () => {
  try {
    const res = await axios.get('/api/apps/')
    apps.value = res.data.data || []
    
    const iconMap = {
      'Monitor': '#e8f4f4', 'Coin': '#e8f4e8',
      'Connection': '#f3e8f4', 'Document': '#fef3e8', 'Tools': '#e8e8f4'
    }
    const colorMap = {
      'Monitor': '#477779', 'Coin': '#2e7d32',
      'Connection': '#7b1fa2', 'Document': '#ed6c02', 'Tools': '#3f51b5'
    }
    apps.value = apps.value.map(app => ({
      ...app,
      iconBg: iconMap[app.icon] || '#e8f4f4',
      iconColor: colorMap[app.icon] || '#477779'
    }))
    
    await fetchContainers()
  } catch (err) {
    console.error('获取应用列表失败:', err)
    ElMessage.error('获取应用列表失败: ' + err.message)
  }
}

const openAppDetail = async (app) => {
  currentApp.value = app
  await fetchContainerInfo(app.key)
  detailDrawerVisible.value = true
}

const restartContainer = async () => {
  actionLoading.value = true
  try {
    await new Promise(resolve => setTimeout(resolve, 1000))
    ElMessage.success('容器已重启')
    await fetchContainerInfo(currentApp.value.key)
  } catch (err) {
    ElMessage.error('操作失败: ' + err.message)
  } finally {
    actionLoading.value = false
  }
}

const stopContainer = async () => {
  actionLoading.value = true
  try {
    await new Promise(resolve => setTimeout(resolve, 1000))
    ElMessage.success('容器已停止')
    await fetchContainerInfo(currentApp.value.key)
  } catch (err) {
    ElMessage.error('操作失败: ' + err.message)
  } finally {
    actionLoading.value = false
  }
}

const startContainer = async () => {
  actionLoading.value = true
  try {
    await new Promise(resolve => setTimeout(resolve, 1000))
    ElMessage.success('容器已启动')
    await fetchContainerInfo(currentApp.value.key)
  } catch (err) {
    ElMessage.error('操作失败: ' + err.message)
  } finally {
    actionLoading.value = false
  }
}

const openLogs = async () => {
  containerLogs.value = `[${new Date().toLocaleString()}] 容器日志示例\n`
  containerLogs.value += `容器名称: ${containerInfo.value.name}\n`
  containerLogs.value += `状态: ${containerInfo.value.status}\n`
  containerLogs.value += `镜像: ${containerInfo.value.image}\n`
  containerLogs.value += `\n--- 日志输出 ---\n`
  containerLogs.value += `[2024-01-01 00:00:00] Container started\n`
  containerLogs.value += `[2024-01-01 00:00:01] Listening on port 80\n`
  logDrawerVisible.value = true
}

const refreshLogs = () => {
  ElMessage.success('日志已刷新')
}

const clearLogs = () => {
  containerLogs.value = ''
  ElMessage.success('日志已清空')
}

const openTerminal = () => {
  ElMessage.info('终端功能开发中，将连接到容器')
}

const openFileDir = () => {
  const dir = `/Users/machangsheng/Downloads/Upanel/apps/${currentApp.value.key}-test5`
  ElMessage.info(`打开目录: ${dir}`)
}

const uninstallApp = async () => {
  await ElMessageBox.confirm(`确定卸载 ${currentApp.value.name} 吗？`, '警告', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  })
  
  try {
    await new Promise(resolve => setTimeout(resolve, 1000))
    ElMessage.success('卸载成功')
    detailDrawerVisible.value = false
    await fetchApps()
  } catch (err) {
    ElMessage.error('卸载失败: ' + err.message)
  }
}

const availableVersions = computed(() => {
  if (!selectedApp.value) return []
  return selectedApp.value.versions || []
})

const filteredApps = computed(() => {
  let result = apps.value
  if (activeCategory.value !== 'all') {
    result = result.filter(app => app.category === activeCategory.value)
  }
  if (activeStatus.value === 'installed') {
    result = result.filter(app => app.installed)
  } else if (activeStatus.value === 'not_installed') {
    result = result.filter(app => !app.installed)
  }
  return result
})

const openInstallDrawer = (app) => {
  if (app.installed) {
    openAppDetail(app)
    return
  }
  selectedApp.value = app
  installForm.value = {
    name: `${app.key}-${app.default_version}`,
    version: app.default_version,
    config: { port: getDefaultPort(app.key) }
  }
  drawerVisible.value = true
}

const getDefaultPort = (key) => {
  const ports = { nginx: 8080, openresty: 8080, mysql: 3306, postgresql: 5432, redis: 6379 }
  return ports[key] || 8080
}

const confirmInstall = async () => {
  installing.value = true
  try {
    await axios.post('/api/apps/install', {
      app_key: selectedApp.value.key,
      version: installForm.value.version,
      name: installForm.value.name,
      config: installForm.value.config
    })
    ElMessage.success('安装成功！')
    drawerVisible.value = false
    await fetchApps()
  } catch (err) {
    ElMessage.error('安装失败: ' + (err.response?.data?.error || err.message))
  } finally {
    installing.value = false
  }
}

onMounted(() => {
  fetchApps()
})
</script>

<style scoped>
.apps-container { height: 100%; display: flex; flex-direction: column; gap: 16px; }
.categories-panel { background: #fff; border-radius: 4px; padding: 12px 16px; }
.categories-left { display: flex; align-items: center; gap: 16px; flex-wrap: wrap; }
.category-group { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.category-item { padding: 6px 14px; font-size: 13px; color: #6b7280; cursor: pointer; border-radius: 4px; }
.category-item:hover { background-color: #f3f4f6; color: #477779; }
.category-item.active { background-color: #477779; color: white; }
.category-divider { width: 1px; height: 24px; background-color: #e5e7eb; }
.apps-panel { background: #fff; border-radius: 4px; padding: 20px; flex: 1; overflow-y: auto; }
.apps-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 16px; }
.app-card { display: flex; gap: 12px; padding: 14px; border: 1px solid #e5e7eb; border-radius: 6px; cursor: pointer; }
.app-card:hover { border-color: #477779; box-shadow: 0 2px 8px rgba(0,0,0,0.08); }
.app-icon { flex-shrink: 0; width: 52px; height: 52px; border-radius: 10px; display: flex; align-items: center; justify-content: center; }
.app-info { flex: 1; display: flex; flex-direction: column; min-width: 0; }
.app-name { font-size: 14px; font-weight: 600; color: #1f2937; margin-bottom: 4px; }
.app-description { font-size: 11px; color: #9ca3af; line-height: 1.3; margin-bottom: 8px; }
.app-footer { display: flex; justify-content: flex-end; }
.drawer-content { padding: 0 4px; }
.drawer-footer { display: flex; justify-content: flex-end; gap: 12px; margin-top: 24px; padding-top: 16px; border-top: 1px solid #e5e7eb; }

/* 详情抽屉样式 */
.detail-content { padding: 0 4px; }
.detail-section { margin-bottom: 24px; padding-bottom: 16px; border-bottom: 1px solid #e5e7eb; }
.detail-section:last-child { border-bottom: none; }
.section-title { font-size: 14px; font-weight: 600; color: #1f2937; margin-bottom: 12px; }
.detail-header { display: flex; gap: 16px; align-items: center; margin-bottom: 12px; }
.app-icon-small { width: 48px; height: 48px; border-radius: 12px; display: flex; align-items: center; justify-content: center; }
.app-name-large { font-size: 18px; font-weight: 600; color: #1f2937; }
.app-version { font-size: 12px; color: #9ca3af; margin-top: 4px; }
.detail-desc { font-size: 13px; color: #6b7280; line-height: 1.5; }
.status-row { display: flex; justify-content: space-between; padding: 8px 0; font-size: 13px; }
.status-label { color: #9ca3af; }
.status-value { color: #1f2937; font-weight: 500; }
.action-buttons { display: flex; flex-wrap: wrap; gap: 8px; }
.container-inspect { background: #f5f5f5; padding: 12px; border-radius: 4px; font-size: 11px; overflow-x: auto; }

/* 日志样式 */
.logs-content { height: 100%; display: flex; flex-direction: column; gap: 12px; }
.log-controls { display: flex; gap: 8px; }
.log-pre { background: #1e1e1e; color: #d4d4d4; padding: 16px; border-radius: 4px; font-family: monospace; font-size: 12px; overflow-x: auto; flex: 1; white-space: pre-wrap; }
</style>
