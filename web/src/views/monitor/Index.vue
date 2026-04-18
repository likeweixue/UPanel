<template>
  <div class="monitor-container">
    <!-- 顶部时间选择 -->
    <div class="top-bar">
      <el-radio-group v-model="timeRange" size="default" @change="onTimeRangeChange">
        <el-radio-button value="1h">1小时</el-radio-button>
        <el-radio-button value="6h">6小时</el-radio-button>
        <el-radio-button value="24h">24小时</el-radio-button>
        <el-radio-button value="7d">7天</el-radio-button>
        <el-radio-button value="30d">30天</el-radio-button>
      </el-radio-group>
      <el-button @click="refreshData" :loading="loading">
        <el-icon><Refresh /></el-icon> 刷新
      </el-button>
    </div>

    <!-- 平均负载图表 -->
    <div class="chart-card">
      <div class="card-header">
        <span class="title">平均负载</span>
        <span class="unit">负载值</span>
      </div>
      <div ref="loadChartRef" class="chart-container"></div>
    </div>

    <!-- CPU 和内存 两列布局 -->
    <div class="two-columns">
      <div class="chart-card">
        <div class="card-header">
          <span class="title">CPU 使用率</span>
          <span class="unit">百分比 (%)</span>
        </div>
        <div ref="cpuChartRef" class="chart-container"></div>
      </div>
      <div class="chart-card">
        <div class="card-header">
          <span class="title">内存使用率</span>
          <span class="unit">百分比 (%)</span>
        </div>
        <div ref="memoryChartRef" class="chart-container"></div>
      </div>
    </div>

    <!-- 磁盘 I/O 和网络 两列布局 -->
    <div class="two-columns">
      <div class="chart-card">
        <div class="card-header">
          <span class="title">磁盘 I/O</span>
          <span class="unit">MB/s</span>
        </div>
        <div ref="diskChartRef" class="chart-container"></div>
      </div>
      <div class="chart-card">
        <div class="card-header">
          <span class="title">网络流量</span>
          <span class="unit">KB/s</span>
        </div>
        <div ref="networkChartRef" class="chart-container"></div>
      </div>
    </div>

    <!-- 进程列表 -->
    <div class="process-card">
      <div class="card-header">
        <span class="title">进程列表</span>
        <el-input v-model="processSearch" placeholder="搜索进程" prefix-icon="Search" clearable style="width: 200px" />
      </div>
      <el-table :data="filteredProcesses" stripe v-loading="loading" max-height="400">
        <el-table-column prop="pid" label="PID" width="80" />
        <el-table-column prop="user" label="用户" width="100" />
        <el-table-column prop="name" label="进程名称" min-width="200" />
        <el-table-column prop="cpu" label="CPU %" width="100">
          <template #default="{ row }">
            <span :style="{ color: row.cpu > 50 ? '#c62828' : '#477779' }">{{ row.cpu }}%</span>
          </template>
        </el-table-column>
        <el-table-column prop="memory" label="内存 %" width="100">
          <template #default="{ row }">
            <span :style="{ color: row.memory > 50 ? '#c62828' : '#477779' }">{{ row.memory }}%</span>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import * as echarts from 'echarts'

const timeRange = ref('24h')
const loading = ref(false)
const processSearch = ref('')

// 图表实例
const loadChartRef = ref(null)
const cpuChartRef = ref(null)
const memoryChartRef = ref(null)
const diskChartRef = ref(null)
const networkChartRef = ref(null)

let loadChart = null
let cpuChart = null
let memoryChart = null
let diskChart = null
let networkChart = null
let refreshInterval = null

// 冷色调颜色
const colors = {
  load1: '#1b8f3c',
  load5: '#28a745',
  load15: '#34ce57',
  cpu: '#17a2b8',
  memory: '#20c997',
  diskRead: '#5bc0de',
  diskWrite: '#5cb85c',
  netUpload: '#5bc0de',
  netDownload: '#5cb85c'
}

// 模拟数据
const loadData = ref({
  times: [],
  load1: [],
  load5: [],
  load15: []
})

const cpuData = ref({
  times: [],
  values: []
})

const memoryData = ref({
  times: [],
  values: []
})

const diskData = ref({
  times: [],
  read: [],
  write: []
})

const networkData = ref({
  times: [],
  upload: [],
  download: []
})

const processes = ref([
  { pid: 1211905, user: 'root', name: 'AliYunDunMonitor', cpu: 6.5, memory: 2.1 },
  { pid: 1211859, user: 'root', name: 'AliYunDun', cpu: 0.81, memory: 1.2 },
  { pid: 1137, user: 'root', name: '/usr/local/cloud', cpu: 0.81, memory: 0.5 },
  { pid: 2697091, user: 'lxd', name: 'mysqld', cpu: 0.73, memory: 8.5 },
  { pid: 3700100, user: 'root', name: '1panel-core', cpu: 0.23, memory: 1.8 },
  { pid: 1234, user: 'www', name: 'nginx', cpu: 0.15, memory: 0.8 },
  { pid: 5678, user: 'www', name: 'php-fpm', cpu: 0.32, memory: 1.2 }
])

const filteredProcesses = computed(() => {
  if (!processSearch.value) return processes.value
  return processes.value.filter(p => 
    p.name.toLowerCase().includes(processSearch.value.toLowerCase()) ||
    p.user.toLowerCase().includes(processSearch.value.toLowerCase())
  )
})

// 生成模拟数据
const generateMockData = () => {
  const now = new Date()
  const hours = timeRange.value === '1h' ? 60 : 
                timeRange.value === '6h' ? 360 : 
                timeRange.value === '24h' ? 1440 : 
                timeRange.value === '7d' ? 10080 : 43200
  const points = Math.min(60, Math.floor(hours / 10))
  const times = []
  const load1 = []
  const load5 = []
  const load15 = []
  const cpu = []
  const memory = []
  const diskRead = []
  const diskWrite = []
  const netUpload = []
  const netDownload = []
  
  for (let i = points; i >= 0; i--) {
    const date = new Date(now.getTime() - i * (hours / points) * 60 * 1000)
    times.push(date.toLocaleTimeString())
    load1.push((Math.random() * 2).toFixed(2))
    load5.push((Math.random() * 1.5).toFixed(2))
    load15.push((Math.random() * 1).toFixed(2))
    cpu.push(Math.floor(Math.random() * 60) + 10)
    memory.push(Math.floor(Math.random() * 70) + 20)
    diskRead.push(Math.floor(Math.random() * 50))
    diskWrite.push(Math.floor(Math.random() * 30))
    netUpload.push(Math.floor(Math.random() * 200) + 50)
    netDownload.push(Math.floor(Math.random() * 500) + 100)
  }
  
  loadData.value = { times, load1, load5, load15 }
  cpuData.value = { times, values: cpu }
  memoryData.value = { times, values: memory }
  diskData.value = { times, read: diskRead, write: diskWrite }
  networkData.value = { times, upload: netUpload, download: netDownload }
}

// 初始化所有图表（折线图，冷色调）
const initCharts = () => {
  // 平均负载图表
  if (loadChartRef.value) {
    if (loadChart) loadChart.dispose()
    loadChart = echarts.init(loadChartRef.value)
    loadChart.setOption({
      tooltip: { trigger: 'axis' },
      legend: { data: ['1分钟', '5分钟', '15分钟'], bottom: 0, textStyle: { color: '#6b7280' } },
      grid: { left: '5%', right: '5%', top: '10%', bottom: '10%', containLabel: true },
      xAxis: { type: 'category', data: loadData.value.times, axisLabel: { rotate: 45, interval: Math.floor(loadData.value.times.length / 10), color: '#9ca3af' } },
      yAxis: { type: 'value', name: '负载值', nameTextStyle: { color: '#9ca3af' }, axisLabel: { color: '#9ca3af' } },
      series: [
        { name: '1分钟', type: 'line', smooth: false, data: loadData.value.load1, lineStyle: { width: 2, color: colors.load1 }, symbol: 'circle', symbolSize: 4 },
        { name: '5分钟', type: 'line', smooth: false, data: loadData.value.load5, lineStyle: { width: 2, color: colors.load5 }, symbol: 'circle', symbolSize: 4 },
        { name: '15分钟', type: 'line', smooth: false, data: loadData.value.load15, lineStyle: { width: 2, color: colors.load15 }, symbol: 'circle', symbolSize: 4 }
      ]
    })
  }
  
  // CPU 图表
  if (cpuChartRef.value) {
    if (cpuChart) cpuChart.dispose()
    cpuChart = echarts.init(cpuChartRef.value)
    cpuChart.setOption({
      tooltip: { trigger: 'axis' },
      grid: { left: '5%', right: '5%', top: '10%', bottom: '5%', containLabel: true },
      xAxis: { type: 'category', data: cpuData.value.times, axisLabel: { rotate: 45, interval: Math.floor(cpuData.value.times.length / 10), color: '#9ca3af' } },
      yAxis: { type: 'value', name: '百分比 (%)', max: 100, nameTextStyle: { color: '#9ca3af' }, axisLabel: { color: '#9ca3af' } },
      series: [{ type: 'line', smooth: false, data: cpuData.value.values, lineStyle: { width: 2, color: colors.cpu }, symbol: 'circle', symbolSize: 4, areaStyle: { opacity: 0.1, color: colors.cpu } }]
    })
  }
  
  // 内存图表
  if (memoryChartRef.value) {
    if (memoryChart) memoryChart.dispose()
    memoryChart = echarts.init(memoryChartRef.value)
    memoryChart.setOption({
      tooltip: { trigger: 'axis' },
      grid: { left: '5%', right: '5%', top: '10%', bottom: '5%', containLabel: true },
      xAxis: { type: 'category', data: memoryData.value.times, axisLabel: { rotate: 45, interval: Math.floor(memoryData.value.times.length / 10), color: '#9ca3af' } },
      yAxis: { type: 'value', name: '百分比 (%)', max: 100, nameTextStyle: { color: '#9ca3af' }, axisLabel: { color: '#9ca3af' } },
      series: [{ type: 'line', smooth: false, data: memoryData.value.values, lineStyle: { width: 2, color: colors.memory }, symbol: 'circle', symbolSize: 4, areaStyle: { opacity: 0.1, color: colors.memory } }]
    })
  }
  
  // 磁盘 I/O 图表
  if (diskChartRef.value) {
    if (diskChart) diskChart.dispose()
    diskChart = echarts.init(diskChartRef.value)
    diskChart.setOption({
      tooltip: { trigger: 'axis' },
      legend: { data: ['读取', '写入'], bottom: 0, textStyle: { color: '#6b7280' } },
      grid: { left: '5%', right: '5%', top: '10%', bottom: '10%', containLabel: true },
      xAxis: { type: 'category', data: diskData.value.times, axisLabel: { rotate: 45, interval: Math.floor(diskData.value.times.length / 10), color: '#9ca3af' } },
      yAxis: { type: 'value', name: 'MB/s', nameTextStyle: { color: '#9ca3af' }, axisLabel: { color: '#9ca3af' } },
      series: [
        { name: '读取', type: 'line', smooth: false, data: diskData.value.read, lineStyle: { width: 2, color: colors.diskRead }, symbol: 'circle', symbolSize: 4 },
        { name: '写入', type: 'line', smooth: false, data: diskData.value.write, lineStyle: { width: 2, color: colors.diskWrite }, symbol: 'circle', symbolSize: 4 }
      ]
    })
  }
  
  // 网络图表
  if (networkChartRef.value) {
    if (networkChart) networkChart.dispose()
    networkChart = echarts.init(networkChartRef.value)
    networkChart.setOption({
      tooltip: { trigger: 'axis' },
      legend: { data: ['上行', '下行'], bottom: 0, textStyle: { color: '#6b7280' } },
      grid: { left: '5%', right: '5%', top: '10%', bottom: '10%', containLabel: true },
      xAxis: { type: 'category', data: networkData.value.times, axisLabel: { rotate: 45, interval: Math.floor(networkData.value.times.length / 10), color: '#9ca3af' } },
      yAxis: { type: 'value', name: 'KB/s', nameTextStyle: { color: '#9ca3af' }, axisLabel: { color: '#9ca3af' } },
      series: [
        { name: '上行', type: 'line', smooth: false, data: networkData.value.upload, lineStyle: { width: 2, color: colors.netUpload }, symbol: 'circle', symbolSize: 4 },
        { name: '下行', type: 'line', smooth: false, data: networkData.value.download, lineStyle: { width: 2, color: colors.netDownload }, symbol: 'circle', symbolSize: 4 }
      ]
    })
  }
}

// 刷新所有图表数据
const updateCharts = () => {
  generateMockData()
  if (loadChart) {
    loadChart.setOption({ xAxis: { data: loadData.value.times }, series: [{ data: loadData.value.load1 }, { data: loadData.value.load5 }, { data: loadData.value.load15 }] })
  }
  if (cpuChart) {
    cpuChart.setOption({ xAxis: { data: cpuData.value.times }, series: [{ data: cpuData.value.values }] })
  }
  if (memoryChart) {
    memoryChart.setOption({ xAxis: { data: memoryData.value.times }, series: [{ data: memoryData.value.values }] })
  }
  if (diskChart) {
    diskChart.setOption({ xAxis: { data: diskData.value.times }, series: [{ data: diskData.value.read }, { data: diskData.value.write }] })
  }
  if (networkChart) {
    networkChart.setOption({ xAxis: { data: networkData.value.times }, series: [{ data: networkData.value.upload }, { data: networkData.value.download }] })
  }
}

const onTimeRangeChange = () => {
  generateMockData()
  initCharts()
}

const refreshData = () => {
  loading.value = true
  setTimeout(() => {
    updateCharts()
    loading.value = false
  }, 500)
}

// 窗口大小变化时重新调整图表
const handleResize = () => {
  if (loadChart) loadChart.resize()
  if (cpuChart) cpuChart.resize()
  if (memoryChart) memoryChart.resize()
  if (diskChart) diskChart.resize()
  if (networkChart) networkChart.resize()
}

onMounted(() => {
  generateMockData()
  setTimeout(() => initCharts(), 100)
  window.addEventListener('resize', handleResize)
  refreshInterval = setInterval(() => updateCharts(), 30000)
})

onUnmounted(() => {
  if (refreshInterval) clearInterval(refreshInterval)
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped>
.monitor-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 16px;
  overflow-y: auto;
}

.top-bar {
  background: #ffffff;
  border-radius: 4px;
  padding: 12px 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.chart-card {
  background: #ffffff;
  border-radius: 4px;
  padding: 16px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #e5e7eb;
}

.card-header .title {
  font-size: 16px;
  font-weight: 600;
  color: #1f2937;
}

.card-header .unit {
  font-size: 12px;
  color: #9ca3af;
}

.chart-container {
  width: 100%;
  height: 320px;
}

.two-columns {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.process-card {
  background: #ffffff;
  border-radius: 4px;
  padding: 16px;
}

.process-card .card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
</style>
