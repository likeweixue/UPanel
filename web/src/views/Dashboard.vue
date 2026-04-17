<template>
  <div class="dashboard">
    <div class="dashboard-main">
      <!-- 左侧区域：占比更大 -->
      <div class="left-panel">
        <!-- d 面板：统计卡片 -->
        <div class="stats-panel">
          <div class="panel-header">
            <span class="panel-title">资源统计</span>
          </div>
          <div class="stats-grid">
            <div class="stat-item">
              <div class="stat-icon website">
                <el-icon><Grid /></el-icon>
              </div>
              <div class="stat-number">{{ stats.websites }}</div>
              <div class="stat-label">网站数量</div>
            </div>
            <div class="stat-divider"></div>
            <div class="stat-item">
              <div class="stat-icon database">
                <el-icon><Coin /></el-icon>
              </div>
              <div class="stat-number">{{ stats.databases }}</div>
              <div class="stat-label">数据库数量</div>
            </div>
            <div class="stat-divider"></div>
            <div class="stat-item">
              <div class="stat-icon task">
                <el-icon><Calendar /></el-icon>
              </div>
              <div class="stat-number">{{ stats.tasks }}</div>
              <div class="stat-label">计划任务</div>
            </div>
          </div>
        </div>

        <!-- e 面板：系统资源仪表 -->
        <div class="gauges-panel">
          <div class="panel-header">
            <span class="panel-title">系统资源</span>
          </div>
          <div class="gauges-grid">
            <div class="gauge-item">
              <div class="gauge-label">系统负荷</div>
              <div class="gauge-progress">
                <svg viewBox="0 0 100 100" class="gauge-svg">
                  <circle cx="50" cy="50" r="42" class="gauge-bg"/>
                  <circle cx="50" cy="50" r="42" class="gauge-fill" :style="{ strokeDasharray: `${circumference}`, strokeDashoffset: `${circumference * (1 - systemStats.loadPercent / 100)}` }"/>
                </svg>
                <div class="gauge-percent">{{ systemStats.load }}</div>
              </div>
              <div class="gauge-unit">平均负载</div>
            </div>
            <div class="gauge-item">
              <div class="gauge-label">CPU 使用率</div>
              <div class="gauge-progress">
                <svg viewBox="0 0 100 100" class="gauge-svg">
                  <circle cx="50" cy="50" r="42" class="gauge-bg"/>
                  <circle cx="50" cy="50" r="42" class="gauge-fill" :style="{ strokeDasharray: `${circumference}`, strokeDashoffset: `${circumference * (1 - systemStats.cpuUsage / 100)}` }"/>
                </svg>
                <div class="gauge-percent">{{ systemStats.cpuUsage }}%</div>
              </div>
            </div>
            <div class="gauge-item">
              <div class="gauge-label">内存使用率</div>
              <div class="gauge-progress">
                <svg viewBox="0 0 100 100" class="gauge-svg">
                  <circle cx="50" cy="50" r="42" class="gauge-bg"/>
                  <circle cx="50" cy="50" r="42" class="gauge-fill" :style="{ strokeDasharray: `${circumference}`, strokeDashoffset: `${circumference * (1 - systemStats.memUsage / 100)}` }"/>
                </svg>
                <div class="gauge-percent">{{ systemStats.memUsage }}%</div>
              </div>
            </div>
            <div class="gauge-item">
              <div class="gauge-label">存储使用率</div>
              <div class="gauge-progress">
                <svg viewBox="0 0 100 100" class="gauge-svg">
                  <circle cx="50" cy="50" r="42" class="gauge-bg"/>
                  <circle cx="50" cy="50" r="42" class="gauge-fill" :style="{ strokeDasharray: `${circumference}`, strokeDashoffset: `${circumference * (1 - systemStats.diskUsage / 100)}` }"/>
                </svg>
                <div class="gauge-percent">{{ systemStats.diskUsage }}%</div>
              </div>
            </div>
          </div>
        </div>

        <!-- f 面板：网络流量 / 磁盘读写（面积图） -->
        <div class="chart-panel">
          <div class="panel-header">
            <span class="panel-title">{{ chartType === 'network' ? '网络流量' : '磁盘读写' }}</span>
            <div class="panel-tabs">
              <span :class="{ active: chartType === 'network' }" @click="switchChart('network')">网络</span>
              <span :class="{ active: chartType === 'disk' }" @click="switchChart('disk')">磁盘</span>
            </div>
          </div>
          <div ref="chartRef" class="chart-container"></div>
        </div>
      </div>

      <!-- 右侧区域：占比更小 -->
      <div class="right-panel">
        <div class="info-panel">
          <div class="panel-header">
            <span class="panel-title">服务器信息</span>
          </div>
          <div class="info-content">
            <div class="info-row">
              <span class="info-label">主机名</span>
              <span class="info-value">{{ serverInfo.hostname }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">操作系统</span>
              <span class="info-value">{{ serverInfo.os }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">内核版本</span>
              <span class="info-value">{{ serverInfo.kernel }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">Docker 版本</span>
              <span class="info-value">{{ serverInfo.dockerVersion }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">Go 版本</span>
              <span class="info-value">{{ serverInfo.goVersion }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">运行时间</span>
              <span class="info-value">{{ serverInfo.uptime }}</span>
            </div>
          </div>
        </div>

        <div class="notification-panel">
          <div class="panel-header">
            <span class="panel-title">系统通知</span>
            <div class="panel-tabs">
              <span :class="{ active: notificationTab === 'notice' }" @click="notificationTab = 'notice'">通知</span>
              <span :class="{ active: notificationTab === 'todo' }" @click="notificationTab = 'todo'">待办</span>
            </div>
          </div>
          <div class="notification-content">
            <div v-if="notificationTab === 'notice'" class="notice-list">
              <div v-for="notice in notices" :key="notice.id" class="notice-item">
                <div class="notice-title">{{ notice.title }}</div>
                <div class="notice-time">{{ notice.time }}</div>
              </div>
              <div v-if="notices.length === 0" class="empty">暂无通知</div>
            </div>
            <div v-else class="todo-list">
              <div v-for="todo in todos" :key="todo.id" class="todo-item">
                <el-checkbox :checked="todo.done">{{ todo.title }}</el-checkbox>
              </div>
              <div v-if="todos.length === 0" class="empty">暂无待办</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { Grid, Coin, Calendar } from '@element-plus/icons-vue'
import axios from 'axios'
import * as echarts from 'echarts'

const stats = ref({
  websites: 0,
  databases: 0,
  tasks: 0
})

const systemStats = ref({
  load: '0.00',
  loadPercent: 0,
  cpuUsage: 0,
  memUsage: 0,
  diskUsage: 0
})

const serverInfo = ref({
  hostname: '加载中...',
  os: '加载中...',
  kernel: '加载中...',
  dockerVersion: '加载中...',
  goVersion: '加载中...',
  uptime: '加载中...'
})

const notices = ref([
  { id: 1, title: '欢迎使用 UPanel', time: '刚刚' }
])

const todos = ref([
  { id: 1, title: '配置 Docker 镜像加速', done: false },
  { id: 2, title: '创建第一个网站', done: false }
])

const chartType = ref('network')
const notificationTab = ref('notice')

const chartRef = ref(null)
let chartInstance = null

const networkData = ref({
  upload: [120, 135, 118, 142, 158, 145, 132, 128, 145, 162, 155, 148],
  download: [320, 335, 318, 342, 358, 345, 332, 328, 345, 362, 355, 348]
})

const diskData = ref({
  read: [45, 52, 48, 55, 62, 58, 51, 49, 56, 63, 59, 54],
  write: [38, 42, 39, 45, 48, 44, 41, 40, 46, 50, 47, 43]
})

const circumference = 2 * Math.PI * 42

const initChart = () => {
  if (!chartRef.value) return
  
  if (chartInstance) {
    chartInstance.dispose()
  }
  
  chartInstance = echarts.init(chartRef.value)
  
  const option = {
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow' }
    },
    grid: {
      left: '5%',
      right: '5%',
      top: '10%',
      bottom: '5%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: ['12:00', '12:05', '12:10', '12:15', '12:20', '12:25', '12:30', '12:35', '12:40', '12:45', '12:50', '12:55'],
      axisLabel: {
        fontSize: 10,
        rotate: 45
      }
    },
    yAxis: {
      type: 'value',
      name: chartType.value === 'network' ? 'KB/s' : 'MB/s',
      nameTextStyle: {
        fontSize: 10
      },
      axisLabel: {
        fontSize: 10
      }
    },
    series: []
  }
  
  if (chartType.value === 'network') {
    option.series = [
      {
        name: '上行',
        type: 'line',
        smooth: true,
        symbol: 'none',
        lineStyle: { width: 2, color: '#477779' },
        areaStyle: { opacity: 0.3, color: '#477779' },
        data: networkData.value.upload
      },
      {
        name: '下行',
        type: 'line',
        smooth: true,
        symbol: 'none',
        lineStyle: { width: 2, color: '#f59e0b' },
        areaStyle: { opacity: 0.3, color: '#f59e0b' },
        data: networkData.value.download
      }
    ]
  } else {
    option.series = [
      {
        name: '读取',
        type: 'line',
        smooth: true,
        symbol: 'none',
        lineStyle: { width: 2, color: '#477779' },
        areaStyle: { opacity: 0.3, color: '#477779' },
        data: diskData.value.read
      },
      {
        name: '写入',
        type: 'line',
        smooth: true,
        symbol: 'none',
        lineStyle: { width: 2, color: '#f59e0b' },
        areaStyle: { opacity: 0.3, color: '#f59e0b' },
        data: diskData.value.write
      }
    ]
  }
  
  chartInstance.setOption(option)
}

const switchChart = (type) => {
  chartType.value = type
  if (chartInstance) {
    initChart()
  }
}

let intervalId = null

const startRealTimeData = () => {
  intervalId = setInterval(() => {
    const newUpload = Math.floor(Math.random() * 80) + 100
    const newDownload = Math.floor(Math.random() * 100) + 300
    const newRead = Math.floor(Math.random() * 30) + 40
    const newWrite = Math.floor(Math.random() * 25) + 35
    
    networkData.value.upload.push(newUpload)
    networkData.value.download.push(newDownload)
    diskData.value.read.push(newRead)
    diskData.value.write.push(newWrite)
    
    if (networkData.value.upload.length > 12) networkData.value.upload.shift()
    if (networkData.value.download.length > 12) networkData.value.download.shift()
    if (diskData.value.read.length > 12) diskData.value.read.shift()
    if (diskData.value.write.length > 12) diskData.value.write.shift()
    
    if (chartInstance) {
      if (chartType.value === 'network') {
        chartInstance.setOption({
          series: [
            { data: networkData.value.upload },
            { data: networkData.value.download }
          ]
        })
      } else {
        chartInstance.setOption({
          series: [
            { data: diskData.value.read },
            { data: diskData.value.write }
          ]
        })
      }
    }
    
    const loadVal = Math.random() * 2
    systemStats.value.load = loadVal.toFixed(2)
    systemStats.value.loadPercent = Math.min(100, Math.floor(loadVal * 50))
    systemStats.value.cpuUsage = Math.floor(Math.random() * 60) + 20
    systemStats.value.memUsage = Math.floor(Math.random() * 70) + 10
    systemStats.value.diskUsage = Math.floor(Math.random() * 50) + 30
  }, 3000)
}

const loadDashboardData = async () => {
  try {
    const sysRes = await axios.get('/api/system/info')
    if (sysRes.data.data) {
      const data = sysRes.data.data
      serverInfo.value = {
        hostname: 'localhost',
        os: data.os || 'macOS',
        kernel: data.arch || 'arm64',
        dockerVersion: data.docker_version || '未连接',
        goVersion: data.go_version || '1.21',
        uptime: data.uptime || '0 分钟'
      }
      
      const loadVal = Math.random() * 2
      systemStats.value.load = loadVal.toFixed(2)
      systemStats.value.loadPercent = Math.min(100, Math.floor(loadVal * 50))
      systemStats.value.cpuUsage = Math.floor(Math.random() * 60) + 20
      systemStats.value.memUsage = Math.floor(Math.random() * 70) + 10
      systemStats.value.diskUsage = Math.floor(Math.random() * 50) + 30
    }
    
    const websitesRes = await axios.get('/api/websites/')
    stats.value.websites = websitesRes.data.data?.length || 0
    
    const databasesRes = await axios.get('/api/databases/')
    stats.value.databases = databasesRes.data.data?.length || 0
    
    const containersRes = await axios.get('/api/containers/')
    stats.value.tasks = containersRes.data.data?.length || 0
    
  } catch (err) {
    console.error('加载总览数据失败:', err)
  }
}

onMounted(() => {
  loadDashboardData()
  setTimeout(() => {
    initChart()
    startRealTimeData()
  }, 500)
})

onUnmounted(() => {
  if (intervalId) {
    clearInterval(intervalId)
  }
  if (chartInstance) {
    chartInstance.dispose()
  }
})
</script>

<style scoped>
.dashboard {
  height: 100%;
}

.dashboard-main {
  display: flex;
  gap: 12px;
}

/* 左侧区域：占比 70% */
.left-panel {
  flex: 8;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* 右侧区域：占比 30%（原来是 50%，减小 1/4 后约 30%） */
.right-panel {
  flex: 2;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.stats-panel,
.gauges-panel,
.chart-panel,
.info-panel,
.notification-panel {
  background: #ffffff;
  border-radius: 4px;
  overflow: hidden;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid #e5e7eb;
}

.panel-title {
  font-size: 14px;
  font-weight: 600;
  color: #1f2937;
}

.panel-tabs {
  display: flex;
  gap: 12px;
}

.panel-tabs span {
  font-size: 12px;
  color: #9ca3af;
  cursor: pointer;
  padding: 2px 8px;
  border-radius: 4px;
}

.panel-tabs span.active {
  background-color: #e8f4f4;
  color: #477779;
}

/* 统计卡片 */
.stats-grid {
  display: flex;
  align-items: center;
  justify-content: space-around;
  padding: 16px 12px;
}

.stat-item {
  flex: 1;
  text-align: center;
}

.stat-icon {
  width: 44px;
  height: 44px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
  margin: 0 auto 6px;
}

.stat-icon.website {
  background-color: #e8f4f4;
  color: #477779;
}

.stat-icon.database {
  background-color: #e8f4e8;
  color: #2e7d32;
}

.stat-icon.task {
  background-color: #fef3e8;
  color: #ed6c02;
}

.stat-number {
  font-size: 26px;
  font-weight: bold;
  color: #1f2937;
}

.stat-label {
  font-size: 12px;
  color: #9ca3af;
  margin-top: 2px;
}

.stat-divider {
  width: 1px;
  height: 48px;
  background-color: #e5e7eb;
}

/* 系统资源仪表 */
.gauges-grid {
  display: flex;
  justify-content: space-around;
  padding: 16px 12px;
}

.gauge-item {
  text-align: center;
  flex: 1;
}

.gauge-label {
  font-size: 12px;
  color: #9ca3af;
  margin-bottom: 8px;
}

.gauge-progress {
  position: relative;
  display: inline-block;
  width: 72px;
  height: 72px;
}

.gauge-svg {
  width: 100%;
  height: 100%;
  transform: rotate(-90deg);
}

.gauge-bg {
  fill: none;
  stroke: #e5e7eb;
  stroke-width: 8;
}

.gauge-fill {
  fill: none;
  stroke: #477779;
  stroke-width: 8;
  stroke-linecap: round;
}

.gauge-percent {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  font-size: 14px;
  font-weight: bold;
  color: #1f2937;
}

.gauge-unit {
  font-size: 10px;
  color: #9ca3af;
  margin-top: 6px;
}

/* 图表面板 */
.chart-container {
  width: 100%;
  height: 260px;
  padding: 6px;
}

/* 服务器信息 */
.info-content {
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.info-row {
  display: flex;
  justify-content: space-between;
  font-size: 13px;
}

.info-label {
  color: #9ca3af;
}

.info-value {
  color: #1f2937;
  font-weight: 500;
}

/* 通知 */
.notification-content {
  padding: 6px 14px;
  max-height: 240px;
  overflow-y: auto;
}

.notice-item, .todo-item {
  padding: 10px 0;
  border-bottom: 1px solid #e5e7eb;
}

.notice-item:last-child, .todo-item:last-child {
  border-bottom: none;
}

.notice-title {
  font-size: 13px;
  color: #1f2937;
  margin-bottom: 4px;
}

.notice-time {
  font-size: 10px;
  color: #9ca3af;
}

.empty {
  text-align: center;
  color: #9ca3af;
  padding: 24px;
  font-size: 12px;
}
</style>
