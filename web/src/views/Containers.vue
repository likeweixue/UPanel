<template>
  <div class="containers-container">
    <!-- 顶部分类面板 -->
    <div class="top-panel">
      <div class="top-left">
        <el-button type="primary" @click="openCreateContainer">
          <el-icon><Plus /></el-icon> 创建容器
        </el-button>
        <el-button @click="refreshAll">
          <el-icon><Refresh /></el-icon> 刷新
        </el-button>
      </div>
      <div class="top-right">
        <el-tabs v-model="activeTab" class="container-tabs">
          <el-tab-pane name="containers">
            <template #label>
              <span><el-icon><Box /></el-icon> 容器 ({{ containers.length }})</span>
            </template>
          </el-tab-pane>
          <el-tab-pane name="images">
            <template #label>
              <span><el-icon><Picture /></el-icon> 镜像 ({{ images.length }})</span>
            </template>
          </el-tab-pane>
          <el-tab-pane name="volumes">
            <template #label>
              <span><el-icon><Folder /></el-icon> 数据卷 ({{ volumes.length }})</span>
            </template>
          </el-tab-pane>
          <el-tab-pane name="networks">
            <template #label>
              <span><el-icon><Connection /></el-icon> 网络 ({{ networks.length }})</span>
            </template>
          </el-tab-pane>
          <el-tab-pane name="compose">
            <template #label>
              <span><el-icon><Files /></el-icon> 编排</span>
            </template>
          </el-tab-pane>
        </el-tabs>
        <el-input 
          v-if="activeTab !== 'compose'"
          v-model="searchKeyword" 
          :placeholder="searchPlaceholder"
          prefix-icon="Search"
          clearable
          style="width: 200px; margin-left: 16px;"
        />
      </div>
    </div>

    <!-- 容器列表 -->
    <div class="content-panel" v-if="activeTab === 'containers'">
      <el-table :data="filteredContainers" stripe style="width: 100%" @selection-change="handleSelectionChange">
        <el-table-column type="selection" width="55" />
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <span class="status-badge" :class="row.status">
              {{ row.status === 'running' ? '运行中' : row.status === 'exited' ? '已停止' : '异常' }}
            </span>
          </template>
        </el-table-column>
        <el-table-column prop="name" label="容器名" min-width="180" />
        <el-table-column prop="image" label="镜像" min-width="200" />
        <el-table-column label="端口映射" min-width="150">
          <template #default="{ row }">
            <span class="ports">{{ row.ports || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="created" label="创建时间" width="160" />
        <el-table-column label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <el-button size="small" :type="row.status === 'running' ? 'warning' : 'success'" @click="toggleContainer(row)">
              {{ row.status === 'running' ? '停止' : '启动' }}
            </el-button>
            <el-button size="small" @click="restartContainer(row)">重启</el-button>
            <el-dropdown @command="(cmd) => handleContainerCommand(cmd, row)">
              <el-button size="small">
                更多 <el-icon class="el-icon--right"><ArrowDown /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="logs">查看日志</el-dropdown-item>
                  <el-dropdown-item command="terminal">终端</el-dropdown-item>
                  <el-dropdown-item command="inspect">详情</el-dropdown-item>
                  <el-dropdown-item command="rename">重命名</el-dropdown-item>
                  <el-dropdown-item command="export">导出</el-dropdown-item>
                  <el-dropdown-item command="delete" divided>删除</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 镜像列表 -->
    <div class="content-panel" v-if="activeTab === 'images'">
      <div class="panel-actions">
        <el-button size="small" @click="openPullImage">
          <el-icon><Download /></el-icon> 拉取镜像
        </el-button>
        <el-button size="small" @click="refreshImages">
          <el-icon><Refresh /></el-icon> 刷新
        </el-button>
      </div>
      <el-table :data="filteredImages" stripe style="width: 100%">
        <el-table-column prop="repository" label="仓库" min-width="200" />
        <el-table-column prop="tag" label="标签" width="120" />
        <el-table-column prop="size" label="大小" width="100" />
        <el-table-column prop="created" label="创建时间" width="160" />
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="runImage(row)">运行</el-button>
            <el-button size="small" type="danger" @click="deleteImage(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 数据卷列表 -->
    <div class="content-panel" v-if="activeTab === 'volumes'">
      <el-table :data="volumes" stripe style="width: 100%">
        <el-table-column prop="name" label="名称" min-width="200" />
        <el-table-column prop="driver" label="驱动" width="120" />
        <el-table-column prop="mountpoint" label="挂载点" min-width="250" />
        <el-table-column prop="created" label="创建时间" width="160" />
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="danger" @click="deleteVolume(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 网络列表 -->
    <div class="content-panel" v-if="activeTab === 'networks'">
      <el-table :data="networks" stripe style="width: 100%">
        <el-table-column prop="name" label="名称" min-width="200" />
        <el-table-column prop="driver" label="驱动" width="120" />
        <el-table-column prop="subnet" label="子网" width="180" />
        <el-table-column prop="gateway" label="网关" width="160" />
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="danger" @click="deleteNetwork(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 编排列表 -->
    <div class="content-panel" v-if="activeTab === 'compose'">
      <div class="panel-actions">
        <el-button type="primary" size="small" @click="openCreateCompose">
          <el-icon><Plus /></el-icon> 新建编排
        </el-button>
        <el-button size="small" @click="refreshCompose">
          <el-icon><Refresh /></el-icon> 刷新
        </el-button>
      </div>
      <el-table :data="composeProjects" stripe style="width: 100%">
        <el-table-column prop="name" label="项目名称" min-width="200" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <span class="status-badge" :class="row.status === 'running' ? 'running' : 'exited'">
              {{ row.status === 'running' ? '运行中' : '已停止' }}
            </span>
          </template>
        </el-table-column>
        <el-table-column prop="services" label="服务数" width="100" />
        <el-table-column prop="created" label="创建时间" width="160" />
        <el-table-column label="操作" width="250" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="startCompose(row)">启动</el-button>
            <el-button size="small" type="warning" @click="stopCompose(row)">停止</el-button>
            <el-button size="small" type="danger" @click="deleteCompose(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 创建容器抽屉 -->
    <el-drawer v-model="createContainerDrawer" title="创建容器" direction="rtl" size="500px">
      <div class="drawer-content">
        <el-form :model="containerForm" label-width="100px">
          <el-form-item label="容器名称" required>
            <el-input v-model="containerForm.name" placeholder="my-container" />
          </el-form-item>
          <el-form-item label="镜像" required>
            <el-input v-model="containerForm.image" placeholder="nginx:alpine" />
          </el-form-item>
          <el-form-item label="端口映射">
            <el-input v-model="containerForm.ports" placeholder="8080:80" />
            <div class="form-tip">格式: 宿主机端口:容器端口，多个用逗号分隔</div>
          </el-form-item>
          <el-form-item label="环境变量">
            <el-input v-model="containerForm.env" placeholder="KEY1=value1,KEY2=value2" />
          </el-form-item>
          <el-form-item label="数据卷">
            <el-input v-model="containerForm.volumes" placeholder="/host/path:/container/path" />
          </el-form-item>
          <el-form-item label="重启策略">
            <el-select v-model="containerForm.restart">
              <el-option label="无" value="no" />
              <el-option label="总是" value="always" />
              <el-option label="失败时" value="on-failure" />
            </el-select>
          </el-form-item>
        </el-form>
        <div class="drawer-footer">
          <el-button @click="createContainerDrawer = false">取消</el-button>
          <el-button type="primary" @click="confirmCreateContainer">创建</el-button>
        </div>
      </div>
    </el-drawer>

    <!-- 拉取镜像抽屉 -->
    <el-drawer v-model="pullImageDrawer" title="拉取镜像" direction="rtl" size="400px">
      <div class="drawer-content">
        <el-form :model="pullImageForm" label-width="80px">
          <el-form-item label="镜像名" required>
            <el-input v-model="pullImageForm.name" placeholder="nginx" />
          </el-form-item>
          <el-form-item label="标签">
            <el-input v-model="pullImageForm.tag" placeholder="latest" />
          </el-form-item>
        </el-form>
        <div class="drawer-footer">
          <el-button @click="pullImageDrawer = false">取消</el-button>
          <el-button type="primary" @click="confirmPullImage">拉取</el-button>
        </div>
      </div>
    </el-drawer>

    <!-- 日志抽屉 -->
    <el-drawer v-model="logsDrawer" title="容器日志" direction="rtl" size="600px">
      <div class="logs-content">
        <pre class="logs-pre">{{ currentLogs }}</pre>
      </div>
    </el-drawer>

    <!-- 详情抽屉 -->
    <el-drawer v-model="detailDrawer" title="容器详情" direction="rtl" size="500px">
      <div class="detail-content">
        <pre class="detail-pre">{{ containerDetail }}</pre>
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh, Box, Picture, Folder, Connection, Files, Download, ArrowDown } from '@element-plus/icons-vue'

const activeTab = ref('containers')
const searchKeyword = ref('')
const createContainerDrawer = ref(false)
const pullImageDrawer = ref(false)
const logsDrawer = ref(false)
const detailDrawer = ref(false)
const currentLogs = ref('')
const containerDetail = ref('')
const selectedRows = ref([])

// 搜索占位符
const searchPlaceholder = computed(() => {
  const placeholders = {
    containers: '搜索容器...',
    images: '搜索镜像...',
    volumes: '搜索数据卷...',
    networks: '搜索网络...'
  }
  return placeholders[activeTab.value] || '搜索...'
})

// 容器数据
const containers = ref([
  { id: 1, name: 'nginx-web', image: 'nginx:alpine', status: 'running', ports: '0.0.0.0:8080->80/tcp', created: '2024-01-15' },
  { id: 2, name: 'mysql-db', image: 'mysql:8.0', status: 'running', ports: '0.0.0.0:3306->3306/tcp', created: '2024-01-20' },
  { id: 3, name: 'redis-cache', image: 'redis:alpine', status: 'exited', ports: '', created: '2024-02-01' },
  { id: 4, name: 'php-app', image: 'php:8.2-fpm', status: 'running', ports: '9000/tcp', created: '2024-02-10' }
])

// 镜像数据
const images = ref([
  { repository: 'nginx', tag: 'alpine', size: '23MB', created: '2024-01-10' },
  { repository: 'mysql', tag: '8.0', size: '450MB', created: '2024-01-15' },
  { repository: 'redis', tag: 'alpine', size: '32MB', created: '2024-01-20' },
  { repository: 'php', tag: '8.2-fpm', size: '420MB', created: '2024-01-25' }
])

// 数据卷数据
const volumes = ref([
  { name: 'mysql_data', driver: 'local', mountpoint: '/var/lib/docker/volumes/mysql_data', created: '2024-01-20' },
  { name: 'nginx_html', driver: 'local', mountpoint: '/var/lib/docker/volumes/nginx_html', created: '2024-01-15' }
])

// 网络数据
const networks = ref([
  { name: 'bridge', driver: 'bridge', subnet: '172.17.0.0/16', gateway: '172.17.0.1' },
  { name: 'upanel_network', driver: 'bridge', subnet: '172.18.0.0/16', gateway: '172.18.0.1' }
])

// 编排项目数据
const composeProjects = ref([
  { name: 'wordpress', status: 'running', services: 2, created: '2024-02-01' },
  { name: 'nextcloud', status: 'stopped', services: 3, created: '2024-02-10' }
])

// 表单数据
const containerForm = ref({
  name: '',
  image: '',
  ports: '',
  env: '',
  volumes: '',
  restart: 'no'
})

const pullImageForm = ref({
  name: '',
  tag: 'latest'
})

// 过滤数据
const filteredContainers = computed(() => {
  if (!searchKeyword.value) return containers.value
  return containers.value.filter(c => c.name.includes(searchKeyword.value) || c.image.includes(searchKeyword.value))
})

const filteredImages = computed(() => {
  if (!searchKeyword.value) return images.value
  return images.value.filter(i => i.repository.includes(searchKeyword.value))
})

const handleSelectionChange = (selection) => {
  selectedRows.value = selection
}

// 容器操作
const toggleContainer = async (row) => {
  try {
    await new Promise(resolve => setTimeout(resolve, 500))
    row.status = row.status === 'running' ? 'exited' : 'running'
    ElMessage.success(`容器已${row.status === 'running' ? '启动' : '停止'}`)
  } catch (err) {
    ElMessage.error('操作失败')
  }
}

const restartContainer = async (row) => {
  try {
    await new Promise(resolve => setTimeout(resolve, 500))
    ElMessage.success('容器已重启')
  } catch (err) {
    ElMessage.error('重启失败')
  }
}

const handleContainerCommand = (cmd, row) => {
  switch (cmd) {
    case 'logs':
      currentLogs.value = `[${row.name}] 日志内容示例...\n2024-01-01 00:00:00 Container started\n2024-01-01 00:00:01 Listening on port 80`
      logsDrawer.value = true
      break
    case 'terminal':
      ElMessage.info('终端功能开发中')
      break
    case 'inspect':
      containerDetail.value = JSON.stringify({
        Name: row.name,
        Image: row.image,
        State: { Status: row.status },
        NetworkSettings: { Ports: row.ports }
      }, null, 2)
      detailDrawer.value = true
      break
    case 'rename':
      ElMessage.info('重命名功能开发中')
      break
    case 'export':
      ElMessage.info('导出功能开发中')
      break
    case 'delete':
      ElMessageBox.confirm(`确定删除容器 ${row.name} 吗？`, '警告', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(async () => {
        const index = containers.value.findIndex(c => c.id === row.id)
        if (index !== -1) {
          containers.value.splice(index, 1)
          ElMessage.success('容器已删除')
        }
      }).catch(() => {})
      break
  }
}

// 镜像操作
const openPullImage = () => {
  pullImageForm.value = { name: '', tag: 'latest' }
  pullImageDrawer.value = true
}

const confirmPullImage = async () => {
  if (!pullImageForm.value.name) {
    ElMessage.warning('请输入镜像名')
    return
  }
  try {
    await new Promise(resolve => setTimeout(resolve, 2000))
    images.value.unshift({
      repository: pullImageForm.value.name,
      tag: pullImageForm.value.tag,
      size: '未知',
      created: new Date().toISOString().split('T')[0]
    })
    ElMessage.success('镜像拉取成功')
    pullImageDrawer.value = false
  } catch (err) {
    ElMessage.error('拉取失败')
  }
}

const refreshImages = () => {
  ElMessage.success('刷新成功')
}

const runImage = (row) => {
  containerForm.value = {
    name: row.repository.replace('/', '_'),
    image: `${row.repository}:${row.tag}`,
    ports: '',
    env: '',
    volumes: '',
    restart: 'no'
  }
  createContainerDrawer.value = true
}

const deleteImage = async (row) => {
  await ElMessageBox.confirm(`确定删除镜像 ${row.repository}:${row.tag} 吗？`, '警告', {
    type: 'warning'
  })
  const index = images.value.findIndex(i => i.repository === row.repository && i.tag === row.tag)
  if (index !== -1) {
    images.value.splice(index, 1)
    ElMessage.success('镜像已删除')
  }
}

// 容器创建
const openCreateContainer = () => {
  containerForm.value = { name: '', image: '', ports: '', env: '', volumes: '', restart: 'no' }
  createContainerDrawer.value = true
}

const confirmCreateContainer = async () => {
  if (!containerForm.value.name || !containerForm.value.image) {
    ElMessage.warning('请填写容器名称和镜像')
    return
  }
  try {
    await new Promise(resolve => setTimeout(resolve, 1000))
    containers.value.unshift({
      id: Date.now(),
      name: containerForm.value.name,
      image: containerForm.value.image,
      status: 'running',
      ports: containerForm.value.ports || '-',
      created: new Date().toISOString().split('T')[0]
    })
    ElMessage.success('容器创建成功')
    createContainerDrawer.value = false
  } catch (err) {
    ElMessage.error('创建失败')
  }
}

// 数据卷操作
const deleteVolume = async (row) => {
  await ElMessageBox.confirm(`确定删除数据卷 ${row.name} 吗？`, '警告', { type: 'warning' })
  const index = volumes.value.findIndex(v => v.name === row.name)
  if (index !== -1) {
    volumes.value.splice(index, 1)
    ElMessage.success('数据卷已删除')
  }
}

// 网络操作
const deleteNetwork = async (row) => {
  await ElMessageBox.confirm(`确定删除网络 ${row.name} 吗？`, '警告', { type: 'warning' })
  const index = networks.value.findIndex(n => n.name === row.name)
  if (index !== -1) {
    networks.value.splice(index, 1)
    ElMessage.success('网络已删除')
  }
}

// 编排操作
const openCreateCompose = () => {
  ElMessage.info('编排创建功能开发中')
}

const refreshCompose = () => {
  ElMessage.success('刷新成功')
}

const startCompose = async (row) => {
  row.status = 'running'
  ElMessage.success(`${row.name} 已启动`)
}

const stopCompose = async (row) => {
  row.status = 'stopped'
  ElMessage.success(`${row.name} 已停止`)
}

const deleteCompose = async (row) => {
  await ElMessageBox.confirm(`确定删除编排项目 ${row.name} 吗？`, '警告', { type: 'warning' })
  const index = composeProjects.value.findIndex(c => c.name === row.name)
  if (index !== -1) {
    composeProjects.value.splice(index, 1)
    ElMessage.success('编排项目已删除')
  }
}

// 刷新所有
const refreshAll = () => {
  ElMessage.success('刷新成功')
}
</script>

<style scoped>
.containers-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.top-panel {
  background: #ffffff;
  border-radius: 4px;
  padding: 12px 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.top-left {
  display: flex;
  gap: 8px;
}

.top-right {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

.container-tabs {
  margin-right: 16px;
}

.content-panel {
  background: #ffffff;
  border-radius: 4px;
  padding: 16px;
  flex: 1;
  overflow-y: auto;
}

.panel-actions {
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
}

.status-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
}

.status-badge.running {
  background-color: #e8f4e8;
  color: #2e7d32;
}

.status-badge.exited {
  background-color: #f4e8e8;
  color: #c62828;
}

.ports {
  font-size: 12px;
  font-family: monospace;
}

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

.logs-content {
  height: 100%;
}

.logs-pre {
  background-color: #1e1e1e;
  color: #d4d4d4;
  padding: 16px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 12px;
  overflow-x: auto;
  white-space: pre-wrap;
  word-break: break-all;
}

.detail-content {
  height: 100%;
}

.detail-pre {
  background-color: #f5f5f5;
  padding: 16px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 12px;
  overflow-x: auto;
}
</style>
