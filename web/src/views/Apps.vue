<template>
  <div class="apps-container">
    <!-- 顶部分类面板 -->
    <div class="categories-panel">
      <div class="categories-left">
        <div class="category-group">
          <span 
            v-for="cat in leftCategories" 
            :key="cat.key"
            class="category-item"
            :class="{ active: activeCategory === cat.key }"
            @click="activeCategory = cat.key"
          >
            {{ cat.name }}
          </span>
        </div>
        
        <div class="category-divider"></div>
        
        <div class="category-group">
          <span 
            v-for="cat in rightCategories" 
            :key="cat.key"
            class="category-item"
            :class="{ active: activeStatus === cat.key }"
            @click="activeStatus = cat.key"
          >
            {{ cat.name }}
          </span>
        </div>
      </div>
    </div>

    <!-- 下方应用列表面板 -->
    <div class="apps-panel">
      <div class="apps-grid">
        <div 
          v-for="app in filteredApps" 
          :key="app.id"
          class="app-card"
          @click="openInstallDrawer(app)"
        >
          <div class="app-icon" :style="{ backgroundColor: app.iconBg }">
            <el-icon :size="32" :color="app.iconColor">
              <component :is="app.icon" />
            </el-icon>
          </div>
          
          <div class="app-info">
            <div class="app-name">{{ app.name }}</div>
            <div class="app-description">{{ app.description }}</div>
            <div class="app-footer">
              <el-button 
                size="small" 
                :type="app.installed ? 'info' : 'primary'"
                :disabled="app.installed"
                @click.stop="openInstallDrawer(app)"
              >
                {{ app.installed ? '已安装' : '安装' }}
              </el-button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 右侧安装抽屉 -->
    <el-drawer
      v-model="drawerVisible"
      :title="`安装 ${selectedApp?.name}`"
      direction="rtl"
      size="400px"
    >
      <div class="drawer-content">
        <!-- 软件名称 -->
        <el-form label-width="100px" label-position="left">
          <el-form-item label="软件名称">
            <el-input 
              v-model="installForm.name" 
              :placeholder="`例如: ${selectedApp?.name}-8.2`"
            />
            <div class="form-tip">用于区分不同版本，例如: php-7.4, php-8.2</div>
          </el-form-item>

          <!-- 主版本号 -->
          <el-form-item label="主版本">
            <el-select v-model="installForm.majorVersion" placeholder="请选择主版本" @change="onMajorVersionChange">
              <el-option 
                v-for="v in availableMajorVersions" 
                :key="v.value" 
                :label="v.label" 
                :value="v.value"
              />
            </el-select>
          </el-form-item>

          <!-- 次版本号 -->
          <el-form-item label="具体版本" v-if="availableMinorVersions.length">
            <el-select v-model="installForm.minorVersion" placeholder="请选择具体版本">
              <el-option 
                v-for="v in availableMinorVersions" 
                :key="v.value" 
                :label="v.label" 
                :value="v.value"
              />
            </el-select>
          </el-form-item>

          <!-- 扩展选择 -->
          <el-form-item label="扩展选择">
            <el-checkbox-group v-model="installForm.extensions">
              <el-checkbox 
                v-for="ext in availableExtensions" 
                :key="ext.value" 
                :value="ext.value"
                :label="ext.label"
              />
            </el-checkbox-group>
            <div class="form-tip">可选扩展，根据需要勾选</div>
          </el-form-item>
        </el-form>

        <div class="drawer-footer">
          <el-button @click="drawerVisible = false">取消</el-button>
          <el-button type="primary" @click="confirmInstall" :loading="installing">
            确认安装
          </el-button>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref, computed, reactive } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Monitor, Download, Coin, Tools, 
  Bell, MoreFilled, Connection, Files,
  Document, Setting, User, DataAnalysis, VideoCamera
} from '@element-plus/icons-vue'

// 左侧分类
const leftCategories = [
  { key: 'all', name: '全部' },
  { key: 'environment', name: '环境' },
  { key: 'database', name: '数据库' },
  { key: 'tools', name: '工具' },
  { key: 'website', name: '建站' },
  { key: 'security', name: '安全' },
  { key: 'entertainment', name: '娱乐' },
  { key: 'other', name: '其他' }
]

// 右侧分类
const rightCategories = [
  { key: 'all', name: '全部' },
  { key: 'not_installed', name: '未安装' },
  { key: 'installed', name: '已安装' },
  { key: 'updatable', name: '可更新' }
]

const activeCategory = ref('all')
const activeStatus = ref('all')
const drawerVisible = ref(false)
const installing = ref(false)
const selectedApp = ref(null)

// 安装表单
const installForm = reactive({
  name: '',
  majorVersion: '',
  minorVersion: '',
  extensions: []
})

// 根据应用类型获取可用版本
const getAvailableVersions = (appName) => {
  const versionsMap = {
    PHP: {
      major: [
        { value: 'php7', label: 'PHP 7' },
        { value: 'php8', label: 'PHP 8' }
      ],
      minor: {
        php7: [
          { value: '7.2', label: '7.2' },
          { value: '7.3', label: '7.3' },
          { value: '7.4', label: '7.4' }
        ],
        php8: [
          { value: '8.0', label: '8.0' },
          { value: '8.1', label: '8.1' },
          { value: '8.2', label: '8.2' },
          { value: '8.3', label: '8.3' }
        ]
      }
    },
    MySQL: {
      major: [
        { value: 'mysql5', label: 'MySQL 5' },
        { value: 'mysql8', label: 'MySQL 8' }
      ],
      minor: {
        mysql5: [
          { value: '5.6', label: '5.6' },
          { value: '5.7', label: '5.7' }
        ],
        mysql8: [
          { value: '8.0', label: '8.0' }
        ]
      }
    },
    Nginx: {
      major: [
        { value: 'nginx1', label: 'Nginx 1.x' }
      ],
      minor: {
        nginx1: [
          { value: '1.20', label: '1.20' },
          { value: '1.22', label: '1.22' },
          { value: '1.24', label: '1.24' }
        ]
      }
    }
  }
  return versionsMap[appName] || { major: [], minor: {} }
}

// 可用主版本
const availableMajorVersions = computed(() => {
  if (!selectedApp.value) return []
  return getAvailableVersions(selectedApp.value.name).major
})

// 可用次版本
const availableMinorVersions = computed(() => {
  if (!selectedApp.value || !installForm.majorVersion) return []
  const versions = getAvailableVersions(selectedApp.value.name)
  return versions.minor[installForm.majorVersion] || []
})

// 可用扩展
const availableExtensions = computed(() => {
  if (!selectedApp.value) return []
  const extensionsMap = {
    PHP: [
      { value: 'mysql', label: 'MySQL 扩展' },
      { value: 'pgsql', label: 'PostgreSQL 扩展' },
      { value: 'redis', label: 'Redis 扩展' },
      { value: 'gd', label: 'GD 图像库' },
      { value: 'curl', label: 'cURL' },
      { value: 'zip', label: 'Zip' },
      { value: 'mbstring', label: 'MBString' }
    ],
    MySQL: [
      { value: 'innodb', label: 'InnoDB 引擎' },
      { value: 'myisam', label: 'MyISAM 引擎' }
    ],
    Nginx: [
      { value: 'http2', label: 'HTTP/2 支持' },
      { value: 'ssl', label: 'SSL 支持' },
      { value: 'gzip', label: 'Gzip 压缩' }
    ]
  }
  return extensionsMap[selectedApp.value.name] || []
})

// 打开安装抽屉
const openInstallDrawer = (app) => {
  if (app.installed) {
    ElMessage.info(`${app.name} 已安装`)
    return
  }
  selectedApp.value = app
  installForm.name = `${app.name}-${app.defaultVersion || 'latest'}`
  installForm.majorVersion = ''
  installForm.minorVersion = ''
  installForm.extensions = []
  drawerVisible.value = true
}

// 主版本变更
const onMajorVersionChange = () => {
  installForm.minorVersion = ''
}

// 确认安装
const confirmInstall = async () => {
  if (!installForm.name) {
    ElMessage.warning('请输入软件名称')
    return
  }
  
  installing.value = true
  try {
    // TODO: 调用后端 API 安装
    console.log('安装参数:', {
      app: selectedApp.value.name,
      name: installForm.name,
      majorVersion: installForm.majorVersion,
      minorVersion: installForm.minorVersion,
      extensions: installForm.extensions
    })
    
    await new Promise(resolve => setTimeout(resolve, 1500))
    ElMessage.success(`${selectedApp.value.name} 安装成功！`)
    drawerVisible.value = false
    // 更新安装状态
    selectedApp.value.installed = true
  } catch (err) {
    ElMessage.error('安装失败: ' + err.message)
  } finally {
    installing.value = false
  }
}

// 应用列表
const apps = ref([
  {
    id: 1,
    name: 'Nginx',
    description: '高性能的HTTP和反向代理服务器',
    category: 'environment',
    installed: false,
    icon: 'Monitor',
    iconBg: '#e8f4f4',
    iconColor: '#477779',
    defaultVersion: '1.24'
  },
  {
    id: 2,
    name: 'MySQL',
    description: '流行的关系型数据库管理系统',
    category: 'database',
    installed: false,
    icon: 'Coin',
    iconBg: '#e8f4e8',
    iconColor: '#2e7d32',
    defaultVersion: '8.0'
  },
  {
    id: 3,
    name: 'PHP',
    description: '流行的通用脚本语言',
    category: 'environment',
    installed: false,
    icon: 'Document',
    iconBg: '#fef3e8',
    iconColor: '#ed6c02',
    defaultVersion: '8.2'
  },
  {
    id: 4,
    name: 'Redis',
    description: '高性能的键值对数据库',
    category: 'database',
    installed: false,
    icon: 'Connection',
    iconBg: '#f3e8f4',
    iconColor: '#7b1fa2',
    defaultVersion: '7.2'
  },
  {
    id: 5,
    name: 'WordPress',
    description: '流行的博客和内容管理系统',
    category: 'website',
    installed: false,
    icon: 'Files',
    iconBg: '#e8f4f4',
    iconColor: '#477779',
    defaultVersion: '6.4'
  },
  {
    id: 6,
    name: 'Portainer',
    description: '轻量级Docker管理界面',
    category: 'tools',
    installed: false,
    icon: 'Tools',
    iconBg: '#e8e8f4',
    iconColor: '#3f51b5',
    defaultVersion: 'latest'
  },
  {
    id: 7,
    name: 'Fail2ban',
    description: '入侵防御软件',
    category: 'security',
    installed: false,
    icon: 'Bell',
    iconBg: '#f4e8e8',
    iconColor: '#c62828',
    defaultVersion: 'latest'
  },
  {
    id: 8,
    name: 'Jellyfin',
    description: '开源媒体服务器',
    category: 'entertainment',
    installed: false,
    icon: 'VideoCamera',
    iconBg: '#e8f4e8',
    iconColor: '#2e7d32',
    defaultVersion: 'latest'
  }
])

// 过滤应用
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
</script>

<style scoped>
.apps-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* 顶部分类面板 */
.categories-panel {
  background: #ffffff;
  border-radius: 4px;
  padding: 12px 16px;
}

.categories-left {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}

.category-group {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.category-item {
  padding: 6px 14px;
  font-size: 13px;
  color: #6b7280;
  cursor: pointer;
  border-radius: 4px;
  transition: all 0.2s;
}

.category-item:hover {
  background-color: #f3f4f6;
  color: #477779;
}

.category-item.active {
  background-color: #477779;
  color: white;
}

.category-divider {
  width: 1px;
  height: 24px;
  background-color: #e5e7eb;
}

/* 下方应用列表面板 */
.apps-panel {
  background: #ffffff;
  border-radius: 4px;
  padding: 20px;
  flex: 1;
  overflow-y: auto;
}

.apps-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 16px;
}

/* 应用卡片 */
.app-card {
  display: flex;
  gap: 16px;
  padding: 16px;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  transition: all 0.2s;
  cursor: pointer;
}

.app-card:hover {
  border-color: #477779;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

.app-icon {
  flex-shrink: 0;
  width: 64px;
  height: 64px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.app-info {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.app-name {
  font-size: 16px;
  font-weight: 600;
  color: #1f2937;
  margin-bottom: 6px;
}

.app-description {
  font-size: 12px;
  color: #9ca3af;
  line-height: 1.4;
  margin-bottom: 12px;
  flex: 1;
}

.app-footer {
  display: flex;
  justify-content: flex-end;
}

/* 抽屉内容 */
.drawer-content {
  padding: 0 4px;
}

.form-tip {
  font-size: 11px;
  color: #9ca3af;
  margin-top: 4px;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 24px;
  padding-top: 16px;
  border-top: 1px solid #e5e7eb;
}
</style>
