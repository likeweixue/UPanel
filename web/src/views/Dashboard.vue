<template>
  <div class="dashboard">
    <!-- 暗色模式开关 -->
    
    <div class="dashboard-main">
      <!-- 左侧区域 -->
      <div class="left-panel">
        <!-- d 面板：统计卡片 -->
        <div class="stats-panel">
          <div class="panel-header"><span class="panel-title">资源统计</span></div>
          <div class="stats-grid">
            <div class="stat-item">
              <div class="stat-icon website"><el-icon><Grid /></el-icon></div>
              <div class="stat-number">{{ stats.websites }}</div>
              <div class="stat-label">网站数量</div>
            </div>
            <div class="stat-divider"></div>
            <div class="stat-item">
              <div class="stat-icon database"><el-icon><Coin /></el-icon></div>
              <div class="stat-number">{{ stats.databases }}</div>
              <div class="stat-label">数据库数量</div>
            </div>
            <div class="stat-divider"></div>
            <div class="stat-item">
              <div class="stat-icon container"><el-icon><Box /></el-icon></div>
              <div class="stat-number">{{ stats.containers }}</div>
              <div class="stat-label">容器数量</div>
            </div>
            <div class="stat-divider"></div>
            <div class="stat-item">
              <div class="stat-icon task"><el-icon><Calendar /></el-icon></div>
              <div class="stat-number">{{ stats.tasks }}</div>
              <div class="stat-label">计划任务</div>
            </div>
          </div>
        </div>

        <!-- e 面板：系统资源仪表 -->
        <div class="gauges-panel">
          <div class="panel-header"><span class="panel-title">系统资源</span></div>
          <div class="gauges-grid">
            <div class="gauge-item">
              <div class="gauge-label">系统负荷</div>
              <div class="gauge-progress">
                <svg viewBox="0 0 100 100" class="gauge-svg">
                  <circle cx="50" cy="50" r="42" class="gauge-bg"/>
                  <circle cx="50" cy="50" r="42" class="gauge-fill" :style="{ strokeDasharray: `${circumference}`, strokeDashoffset: `${circumference * (1 - (systemStats.load_percent || 0) / 100)}` }"/>
                </svg>
                <div class="gauge-percent">{{ systemStats.load_avg }}</div>
              </div>
              <div class="gauge-unit">平均负载</div>
            </div>
            <div class="gauge-item">
              <div class="gauge-label">CPU 使用率</div>
              <div class="gauge-progress">
                <svg viewBox="0 0 100 100" class="gauge-svg">
                  <circle cx="50" cy="50" r="42" class="gauge-bg"/>
                  <circle cx="50" cy="50" r="42" class="gauge-fill" :style="{ strokeDasharray: `${circumference}`, strokeDashoffset: `${circumference * (1 - (systemStats.cpu_percent || 0) / 100)}` }"/>
                </svg>
                <div class="gauge-percent">{{ Math.round(systemStats.cpu_percent || 0) }}%</div>
              </div>
            </div>
            <div class="gauge-item">
              <div class="gauge-label">内存使用率</div>
              <div class="gauge-progress">
                <svg viewBox="0 0 100 100" class="gauge-svg">
                  <circle cx="50" cy="50" r="42" class="gauge-bg"/>
                  <circle cx="50" cy="50" r="42" class="gauge-fill" :style="{ strokeDasharray: `${circumference}`, strokeDashoffset: `${circumference * (1 - (systemStats.memory_percent || 0) / 100)}` }"/>
                </svg>
                <div class="gauge-percent">{{ Math.round(systemStats.memory_percent || 0) }}%</div>
              </div>
            </div>
            <div class="gauge-item">
              <div class="gauge-label">存储使用率</div>
              <div class="gauge-progress">
                <svg viewBox="0 0 100 100" class="gauge-svg">
                  <circle cx="50" cy="50" r="42" class="gauge-bg"/>
                  <circle cx="50" cy="50" r="42" class="gauge-fill" :style="{ strokeDasharray: `${circumference}`, strokeDashoffset: `${circumference * (1 - (systemStats.disk_percent || 0) / 100)}` }"/>
                </svg>
                <div class="gauge-percent">{{ Math.round(systemStats.disk_percent || 0) }}%</div>
              </div>
            </div>
          </div>
        </div>

        <!-- f 面板：网络流量 -->
        <div class="chart-panel">
          <div class="panel-header"><span class="panel-title">{{ chartType === 'network' ? '网络流量' : '磁盘读写' }}</span><div class="panel-tabs"><span :class="{ active: chartType === 'network' }" @click="switchChart('network')">网络</span><span :class="{ active: chartType === 'disk' }" @click="switchChart('disk')">磁盘</span></div></div>
          <div ref="chartRef" class="chart-container"></div>
        </div>
      </div>

      <!-- 右侧区域 -->
      <div class="right-panel">
        <div class="info-panel"><div class="panel-header"><span class="panel-title">服务器信息</span></div><div class="info-content"><div class="info-row"><span class="info-label">主机名</span><span class="info-value">{{ serverInfo.hostname }}</span></div><div class="info-row"><span class="info-label">操作系统</span><span class="info-value">{{ serverInfo.os }}</span></div><div class="info-row"><span class="info-label">Docker 版本</span><span class="info-value">{{ serverInfo.dockerVersion }}</span></div><div class="info-row"><span class="info-label">运行时间</span><span class="info-value">{{ serverInfo.uptime }}</span></div></div></div>
        <div class="notification-panel"><div class="panel-header"><span class="panel-title">系统通知</span><div class="panel-tabs"><span :class="{ active: notificationTab === 'notice' }" @click="notificationTab = 'notice'">通知</span><span :class="{ active: notificationTab === 'todo' }" @click="notificationTab = 'todo'">待办</span></div></div><div class="notification-content"><div v-if="notificationTab === 'notice'"><div v-for="notice in notices" class="notice-item"><div class="notice-title">{{ notice.title }}</div><div class="notice-time">{{ notice.time }}</div></div></div><div v-else><div v-for="todo in todos" class="todo-item"><el-checkbox :checked="todo.done">{{ todo.title }}</el-checkbox></div></div></div></div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { Grid, Coin, Calendar, Box } from '@element-plus/icons-vue'
import axios from 'axios'
import * as echarts from 'echarts'
import { useTheme } from '@/composables/useTheme'

const { isDark, toggleTheme, initTheme } = useTheme()
const isDarkMode = ref(isDark.value)

const toggleDarkMode = (val) => {
  toggleTheme()
  isDarkMode.value = val
  // 刷新图表以适应暗色模式
  setTimeout(() => {
    if (chartInstance) initChart()
  }, 100)
}

const stats = ref({ websites: 0, databases: 0, containers: 0, tasks: 0 })
const systemStats = ref({ load_avg: '0.00', load_percent: 0, cpu_percent: 0, memory_percent: 0, disk_percent: 0 })
const serverInfo = ref({ hostname: '加载中...', os: '加载中...', dockerVersion: '加载中...', uptime: '加载中...' })
const notices = ref([{ id: 1, title: '欢迎使用 UPanel', time: '刚刚' }])
const todos = ref([{ id: 1, title: '配置 Docker 镜像加速', done: false }, { id: 2, title: '创建第一个网站', done: false }])
const chartType = ref('network')
const notificationTab = ref('notice')
const chartRef = ref(null)
let chartInstance = null

const networkData = ref({ upload: [120, 135, 118, 142, 158, 145, 132, 128, 145, 162, 155, 148], download: [320, 335, 318, 342, 358, 345, 332, 328, 345, 362, 355, 348] })
const diskData = ref({ read: [45, 52, 48, 55, 62, 58, 51, 49, 56, 63, 59, 54], write: [38, 42, 39, 45, 48, 44, 41, 40, 46, 50, 47, 43] })
const circumference = 2 * Math.PI * 42

const initChart = () => {
  if (!chartRef.value) return
  if (chartInstance) chartInstance.dispose()
  chartInstance = echarts.init(chartRef.value)
  const isDarkMode = document.body.classList.contains('dark-mode')
  const textColor = isDarkMode ? '#e5e7eb' : '#1f2937'
  const axisColor = isDarkMode ? '#9ca3af' : '#6b7280'
  const gridColor = isDarkMode ? '#2a2a2a' : '#e5e7eb'
  
  const option = { 
    tooltip: { trigger: 'axis' }, 
    grid: { left: '5%', right: '5%', top: '10%', bottom: '5%', containLabel: true, borderColor: gridColor },
    xAxis: { type: 'category', data: ['12:00', '12:05', '12:10', '12:15', '12:20', '12:25', '12:30', '12:35', '12:40', '12:45', '12:50', '12:55'], axisLabel: { fontSize: 10, rotate: 45, color: axisColor } },
    yAxis: { type: 'value', name: chartType.value === 'network' ? 'KB/s' : 'MB/s', nameTextStyle: { color: axisColor }, axisLabel: { color: axisColor } },
    series: []
  }
  if (chartType.value === 'network') {
    option.series = [
      { name: '上行', type: 'line', smooth: true, symbol: 'none', lineStyle: { width: 2, color: '#5bc0de' }, areaStyle: { opacity: 0.3, color: '#5bc0de' }, data: networkData.value.upload },
      { name: '下行', type: 'line', smooth: true, symbol: 'none', lineStyle: { width: 2, color: '#5cb85c' }, areaStyle: { opacity: 0.3, color: '#5cb85c' }, data: networkData.value.download }
    ]
  } else {
    option.series = [
      { name: '读取', type: 'line', smooth: true, symbol: 'none', lineStyle: { width: 2, color: '#5bc0de' }, areaStyle: { opacity: 0.3, color: '#5bc0de' }, data: diskData.value.read },
      { name: '写入', type: 'line', smooth: true, symbol: 'none', lineStyle: { width: 2, color: '#5cb85c' }, areaStyle: { opacity: 0.3, color: '#5cb85c' }, data: diskData.value.write }
    ]
  }
  chartInstance.setOption(option)
}

const switchChart = (type) => { chartType.value = type; if (chartInstance) initChart() }
const handleResize = () => { if (chartInstance) chartInstance.resize() }

onMounted(async () => {
  initTheme()
  try {
    const sysRes = await axios.get('/api/system/info')
    if (sysRes.data.data) {
      const data = sysRes.data.data
      serverInfo.value = { hostname: 'localhost', os: data.os || 'macOS', dockerVersion: data.docker_version || '未连接', uptime: data.uptime || '0 分钟' }
    }
    const [websitesRes, databasesRes, containersRes] = await Promise.all([axios.get('/api/websites/'), axios.get('/api/databases/'), axios.get('/api/containers/')])
    stats.value = { websites: websitesRes.data.data?.length || 0, databases: databasesRes.data.data?.length || 0, containers: containersRes.data.data?.length || 0, tasks: 0 }
  } catch (err) { console.error('加载失败:', err) }
  setTimeout(() => { initChart() }, 500)
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  if (chartInstance) chartInstance.dispose()
})
</script>

<style scoped>
.dashboard { height: 100%; width: 100%; position: relative; }
.theme-switch { position: fixed; top: 12px; right: 120px; z-index: 1000; z-index: 100; }
.dashboard-main { display: flex; gap: 16px; width: 100%; }
.left-panel { flex: 2; display: flex; flex-direction: column; gap: 16px; min-width: 0; }
.right-panel { flex: 1; display: flex; flex-direction: column; gap: 16px; min-width: 0; }
.stats-panel, .gauges-panel, .chart-panel, .info-panel, .notification-panel { background: #ffffff; border-radius: 4px; overflow: hidden; }
.panel-header { display: flex; justify-content: space-between; align-items: center; padding: 12px 16px; border-bottom: 1px solid #e5e7eb; }
.panel-title { font-size: 14px; font-weight: 600; color: #1f2937; }
.panel-tabs { display: flex; gap: 12px; }
.panel-tabs span { font-size: 12px; color: #9ca3af; cursor: pointer; padding: 2px 8px; border-radius: 4px; }
.panel-tabs span.active { background-color: #e8f4f4; color: #477779; }
.stats-grid { display: flex; align-items: center; justify-content: space-around; padding: 16px 12px; }
.stat-item { flex: 1; text-align: center; }
.stat-icon { width: 44px; height: 44px; border-radius: 8px; display: flex; align-items: center; justify-content: center; font-size: 22px; margin: 0 auto 6px; }
.stat-icon.website { background-color: #e8f4f4; color: #477779; }
.stat-icon.database { background-color: #e8f4e8; color: #2e7d32; }
.stat-icon.container { background-color: #e8f4f4; color: #477779; }
.stat-icon.task { background-color: #fef3e8; color: #ed6c02; }
.stat-number { font-size: 26px; font-weight: bold; color: #1f2937; }
.stat-label { font-size: 12px; color: #9ca3af; margin-top: 2px; }
.stat-divider { width: 1px; height: 48px; background-color: #e5e7eb; }
.gauges-grid { display: flex; justify-content: space-around; padding: 16px 12px; }
.gauge-item { text-align: center; flex: 1; }
.gauge-label { font-size: 12px; color: #9ca3af; margin-bottom: 12px; }
.load-value { font-size: 24px; font-weight: bold; color: #1f2937; margin-top: 20px; }
.gauge-progress { position: relative; display: inline-block; width: 80px; height: 80px; }
.gauge-svg { width: 100%; height: 100%; transform: rotate(-90deg); }
.gauge-bg { fill: none; stroke: #e5e7eb; stroke-width: 8; }
.gauge-fill { fill: none; stroke: #477779; stroke-width: 8; stroke-linecap: round; }
.gauge-percent { position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%); font-size: 14px; font-weight: bold; color: #1f2937; }
.gauge-unit { font-size: 10px; color: #9ca3af; margin-top: 6px; }
.chart-container { width: 100%; height: 260px; padding: 6px; }
.info-content { padding: 14px; display: flex; flex-direction: column; gap: 10px; }
.info-row { display: flex; justify-content: space-between; font-size: 13px; }
.info-label { color: #9ca3af; }
.info-value { color: #1f2937; font-weight: 500; }
.notification-content { padding: 6px 14px; max-height: 260px; overflow-y: auto; }
.notice-item, .todo-item { padding: 10px 0; border-bottom: 1px solid #e5e7eb; }
.notice-title { font-size: 13px; color: #1f2937; }
.notice-time { font-size: 10px; color: #9ca3af; }
</style>

// 在 onMounted 中添加主题同步
const syncTheme = () => {
  const saved = localStorage.getItem('theme')
  if (saved === 'davinci') {
    document.body.classList.add('theme-davinci')
    document.body.classList.remove('theme-light')
  } else {
    document.body.classList.add('theme-light')
    document.body.classList.remove('theme-davinci')
  }
}

onMounted(() => {
  // ... 原有代码
  syncTheme()
})
