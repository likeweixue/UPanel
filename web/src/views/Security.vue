<template>
  <div class="security-container">
    <!-- 顶部分类面板 -->
    <div class="top-panel">
      <div class="top-left">
        <el-button type="primary" @click="saveAllSettings" :loading="saving">
          <el-icon><Check /></el-icon> 保存设置
        </el-button>
        <el-button @click="refreshSettings">
          <el-icon><Refresh /></el-icon> 刷新
        </el-button>
      </div>
      <div class="top-right">
        <el-tabs v-model="activeTab" class="security-tabs">
          <el-tab-pane name="overview">
            <template #label>
              <span><el-icon><DataAnalysis /></el-icon> 安全概览</span>
            </template>
          </el-tab-pane>
          <el-tab-pane name="firewall">
            <template #label>
              <span><el-icon><Lock /></el-icon> 防火墙</span>
            </template>
          </el-tab-pane>
          <el-tab-pane name="ssh">
            <template #label>
              <span><el-icon><Monitor /></el-icon> SSH安全</span>
            </template>
          </el-tab-pane>
          <el-tab-pane name="login">
            <template #label>
              <span><el-icon><User /></el-icon> 登录保护</span>
            </template>
          </el-tab-pane>
          <el-tab-pane name="fail2ban">
            <template #label>
              <span><el-icon><Warning /></el-icon> Fail2ban</span>
            </template>
          </el-tab-pane>
          <el-tab-pane name="ssl">
            <template #label>
              <span><el-icon><Connection /></el-icon> SSL证书</span>
            </template>
          </el-tab-pane>
          <el-tab-pane name="backup">
            <template #label>
              <span><el-icon><Files /></el-icon> 安全备份</span>
            </template>
          </el-tab-pane>
        </el-tabs>
      </div>
    </div>

    <!-- 安全概览 -->
    <div class="content-panel" v-if="activeTab === 'overview'">
      <!-- 统计卡片 -->
      <div class="stats-grid">
        <div class="stat-card">
          <div class="stat-icon" style="background: #e8f4f4;">
            <el-icon color="#477779"><Warning /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-number">{{ stats.totalAttacks }}</div>
            <div class="stat-label">今日攻击</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon" style="background: #e8f4e8;">
            <el-icon color="#2e7d32"><Lock /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-number">{{ stats.blockedIps }}</div>
            <div class="stat-label">已封禁IP</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon" style="background: #fef3e8;">
            <el-icon color="#ed6c02"><Monitor /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-number">{{ stats.scanAttempts }}</div>
            <div class="stat-label">扫描尝试</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon" style="background: #f3e8f4;">
            <el-icon color="#7b1fa2"><Connection /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-number">{{ stats.sslCerts }}</div>
            <div class="stat-label">SSL证书</div>
          </div>
        </div>
      </div>

      <!-- 世界地图 -->
      <div class="map-section">
        <div class="section-header">
          <h3 class="section-title">攻击来源分布</h3>
          <div class="map-controls">
            <el-radio-group v-model="mapDataType" size="small">
              <el-radio-button label="attacks">攻击次数</el-radio-button>
              <el-radio-button label="ips">来源IP</el-radio-button>
            </el-radio-group>
          </div>
        </div>
        <div ref="mapChartRef" class="map-container"></div>
      </div>

      <!-- 攻击趋势图 -->
      <div class="chart-section">
        <div class="section-header">
          <h3 class="section-title">攻击趋势</h3>
          <el-select v-model="trendDays" size="small" style="width: 100px">
            <el-option label="最近7天" :value="7" />
            <el-option label="最近30天" :value="30" />
            <el-option label="最近90天" :value="90" />
          </el-select>
        </div>
        <div ref="trendChartRef" class="trend-container"></div>
      </div>

      <!-- 攻击类型分布和最近事件 -->
      <div class="two-columns">
        <div class="column">
          <div class="section-header">
            <h3 class="section-title">攻击类型分布</h3>
          </div>
          <div ref="typeChartRef" class="type-container"></div>
        </div>
        <div class="column">
          <div class="section-header">
            <h3 class="section-title">最近安全事件</h3>
            <el-button size="small" @click="clearEvents">清空</el-button>
          </div>
          <div class="events-list">
            <div v-for="event in recentEvents" :key="event.id" class="event-item" :class="event.level">
              <div class="event-time">{{ event.time }}</div>
              <div class="event-content">
                <span class="event-type">{{ event.type }}</span>
                <span class="event-desc">{{ event.description }}</span>
              </div>
              <div class="event-ip">{{ event.ip }}</div>
            </div>
            <div v-if="recentEvents.length === 0" class="empty">暂无安全事件</div>
          </div>
        </div>
      </div>

      <!-- 实时流量监控 -->
      <div class="chart-section">
        <div class="section-header">
          <h3 class="section-title">实时流量监控</h3>
          <el-button size="small" @click="startRealTime" :loading="realtimeLoading">开始监控</el-button>
        </div>
        <div ref="trafficChartRef" class="traffic-container"></div>
      </div>
    </div>

    <!-- 防火墙设置 -->
    <div class="content-panel" v-if="activeTab === 'firewall'">
      <!-- 防火墙设置内容保持不变 -->
      <div class="settings-section">
        <div class="section-header">
          <h3 class="section-title">防火墙状态</h3>
          <el-switch v-model="firewallForm.enabled" />
        </div>
        <div class="section-desc">控制系统的防火墙开关，关闭后所有端口都将开放</div>
      </div>

      <div class="settings-section">
        <h3 class="section-title">端口规则</h3>
        <div class="port-rules">
          <div class="rule-header">
            <span>端口</span>
            <span>协议</span>
            <span>来源IP</span>
            <span>操作</span>
          </div>
          <div v-for="(rule, idx) in firewallForm.rules" :key="idx" class="rule-row">
            <el-input v-model="rule.port" placeholder="80" size="small" style="width: 120px" />
            <el-select v-model="rule.protocol" size="small" style="width: 100px">
              <el-option label="TCP" value="tcp" />
              <el-option label="UDP" value="udp" />
              <el-option label="TCP+UDP" value="both" />
            </el-select>
            <el-input v-model="rule.source" placeholder="0.0.0.0/0" size="small" style="width: 150px" />
            <div class="rule-actions">
              <el-button size="small" text @click="addPortRule">+</el-button>
              <el-button size="small" text type="danger" @click="removePortRule(idx)">-</el-button>
            </div>
          </div>
        </div>
      </div>

      <div class="settings-section">
        <h3 class="section-title">IP黑白名单</h3>
        <el-tabs v-model="ipListType" class="ip-tabs">
          <el-tab-pane label="黑名单" name="blacklist">
            <div class="ip-list">
              <div v-for="(ip, idx) in firewallForm.blacklist" :key="idx" class="ip-item">
                <span>{{ ip }}</span>
                <el-button size="small" text type="danger" @click="removeBlacklist(idx)">删除</el-button>
              </div>
              <div class="add-ip">
                <el-input v-model="newBlacklistIp" placeholder="输入IP地址" size="small" style="width: 200px" />
                <el-button size="small" @click="addBlacklist">添加</el-button>
              </div>
            </div>
          </el-tab-pane>
          <el-tab-pane label="白名单" name="whitelist">
            <div class="ip-list">
              <div v-for="(ip, idx) in firewallForm.whitelist" :key="idx" class="ip-item">
                <span>{{ ip }}</span>
                <el-button size="small" text type="danger" @click="removeWhitelist(idx)">删除</el-button>
              </div>
              <div class="add-ip">
                <el-input v-model="newWhitelistIp" placeholder="输入IP地址" size="small" style="width: 200px" />
                <el-button size="small" @click="addWhitelist">添加</el-button>
              </div>
            </div>
          </el-tab-pane>
        </el-tabs>
      </div>
    </div>

    <!-- SSH安全设置（简化） -->
    <div class="content-panel" v-if="activeTab === 'ssh'">
      <div class="settings-section">
        <h3 class="section-title">SSH服务设置</h3>
        <el-form label-width="140px">
          <el-form-item label="SSH端口">
            <el-input-number v-model="sshForm.port" :min="1" :max="65535" />
          </el-form-item>
          <el-form-item label="允许密码登录">
            <el-switch v-model="sshForm.passwordAuth" />
          </el-form-item>
          <el-form-item label="允许Root登录">
            <el-switch v-model="sshForm.rootLogin" />
          </el-form-item>
          <el-form-item label="空闲超时">
            <el-input-number v-model="sshForm.idleTimeout" :min="0" :max="3600" /> 秒
          </el-form-item>
        </el-form>
      </div>
    </div>

    <!-- 登录保护设置（简化） -->
    <div class="content-panel" v-if="activeTab === 'login'">
      <div class="settings-section">
        <h3 class="section-title">登录限制</h3>
        <el-form label-width="160px">
          <el-form-item label="登录失败限制">
            <el-switch v-model="loginForm.loginLimit" />
          </el-form-item>
          <el-form-item label="最大失败次数" v-if="loginForm.loginLimit">
            <el-input-number v-model="loginForm.maxAttempts" :min="1" :max="20" /> 次
          </el-form-item>
          <el-form-item label="锁定时长" v-if="loginForm.loginLimit">
            <el-input-number v-model="loginForm.lockMinutes" :min="1" :max="60" /> 分钟
          </el-form-item>
        </el-form>
      </div>
    </div>

    <!-- Fail2ban设置（简化） -->
    <div class="content-panel" v-if="activeTab === 'fail2ban'">
      <div class="settings-section">
        <div class="section-header">
          <h3 class="section-title">Fail2ban状态</h3>
          <el-switch v-model="fail2banForm.enabled" />
        </div>
        <el-form label-width="160px">
          <el-form-item label="最大尝试次数">
            <el-input-number v-model="fail2banForm.maxRetry" :min="1" :max="10" /> 次
          </el-form-item>
          <el-form-item label="封禁时间">
            <el-input-number v-model="fail2banForm.banTime" :min="60" :max="86400" /> 秒
          </el-form-item>
        </el-form>
      </div>
    </div>

    <!-- SSL证书设置（简化） -->
    <div class="content-panel" v-if="activeTab === 'ssl'">
      <div class="settings-section">
        <h3 class="section-title">证书管理</h3>
        <el-table :data="sslForm.certificates" stripe style="width: 100%">
          <el-table-column prop="domain" label="域名" width="200" />
          <el-table-column prop="issuer" label="颁发机构" width="150" />
          <el-table-column prop="expireDate" label="过期时间" width="180" />
          <el-table-column label="操作" width="150">
            <template #default="{ row }">
              <el-button size="small" @click="renewCert(row)">续期</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <!-- 安全备份（简化） -->
    <div class="content-panel" v-if="activeTab === 'backup'">
      <div class="settings-section">
        <h3 class="section-title">安全配置备份</h3>
        <el-form label-width="140px">
          <el-form-item>
            <el-button @click="backupNow" :loading="backingUp">立即备份</el-button>
          </el-form-item>
        </el-form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Check, Refresh, Lock, Monitor, User, Warning, Connection, Files, DataAnalysis } from '@element-plus/icons-vue'
import * as echarts from 'echarts'

const activeTab = ref('overview')
const ipListType = ref('blacklist')
const saving = ref(false)
const backingUp = ref(false)
const mapDataType = ref('attacks')
const trendDays = ref(7)
const realtimeLoading = ref(false)

// 图表实例
const mapChartRef = ref(null)
const trendChartRef = ref(null)
const typeChartRef = ref(null)
const trafficChartRef = ref(null)
let mapChart = null
let trendChart = null
let typeChart = null
let trafficChart = null
let realtimeInterval = null

// 统计数据
const stats = ref({
  totalAttacks: 1234,
  blockedIps: 56,
  scanAttempts: 789,
  sslCerts: 3
})

// 世界地图数据
const attackMapData = ref([
  { name: 'China', value: 456 },
  { name: 'United States', value: 234 },
  { name: 'Russia', value: 123 },
  { name: 'Germany', value: 89 },
  { name: 'France', value: 67 },
  { name: 'United Kingdom', value: 56 },
  { name: 'Japan', value: 45 },
  { name: 'South Korea', value: 34 },
  { name: 'Brazil', value: 23 },
  { name: 'India', value: 12 }
])

// 趋势数据
const trendData = ref({
  dates: ['01-10', '01-11', '01-12', '01-13', '01-14', '01-15', '01-16'],
  attacks: [45, 52, 38, 65, 43, 58, 48],
  blocks: [12, 15, 8, 22, 10, 18, 14]
})

// 攻击类型数据
const attackTypeData = ref([
  { name: '暴力破解', value: 345 },
  { name: 'DDoS攻击', value: 234 },
  { name: 'SQL注入', value: 123 },
  { name: 'XSS攻击', value: 89 },
  { name: 'CC攻击', value: 67 }
])

// 最近事件
const recentEvents = ref([
  { id: 1, time: '10:23:45', type: '暴力破解', description: '来自 203.0.113.50 的多次登录尝试', ip: '203.0.113.50', level: 'high' },
  { id: 2, time: '09:15:22', type: '端口扫描', description: '检测到端口扫描行为', ip: '185.45.12.3', level: 'medium' },
  { id: 3, time: '08:30:10', type: '异常请求', description: '异常请求模式检测', ip: '104.28.16.5', level: 'low' }
])

// 防火墙表单
const firewallForm = reactive({
  enabled: true,
  rules: [
    { port: '22', protocol: 'tcp', source: '0.0.0.0/0' },
    { port: '80', protocol: 'tcp', source: '0.0.0.0/0' },
    { port: '443', protocol: 'tcp', source: '0.0.0.0/0' }
  ],
  blacklist: [],
  whitelist: ['192.168.1.0/24']
})

const newBlacklistIp = ref('')
const newWhitelistIp = ref('')

// SSH表单
const sshForm = reactive({
  port: 22,
  passwordAuth: true,
  rootLogin: true,
  idleTimeout: 300
})

// 登录保护表单
const loginForm = reactive({
  loginLimit: true,
  maxAttempts: 5,
  lockMinutes: 15
})

// Fail2ban表单
const fail2banForm = reactive({
  enabled: true,
  maxRetry: 3,
  banTime: 600
})

// SSL表单
const sslForm = reactive({
  certificates: [
    { domain: 'example.com', issuer: "Let's Encrypt", expireDate: '2025-01-15' }
  ]
})

// 初始化地图
const initMapChart = () => {
  if (!mapChartRef.value) return
  if (mapChart) mapChart.dispose()
  
  mapChart = echarts.init(mapChartRef.value)
  mapChart.setOption({
    title: { show: false },
    tooltip: { trigger: 'item' },
    visualMap: {
      min: 0,
      max: 500,
      left: 'left',
      top: 'bottom',
      calculable: true,
      inRange: { color: ['#e8f4f4', '#477779', '#2d5a5c'] }
    },
    series: [{
      name: '攻击来源',
      type: 'map',
      map: 'world',
      roam: true,
      data: attackMapData.value,
      label: { show: false },
      emphasis: { label: { show: true } }
    }]
  })
}

// 初始化趋势图
const initTrendChart = () => {
  if (!trendChartRef.value) return
  if (trendChart) trendChart.dispose()
  
  trendChart = echarts.init(trendChartRef.value)
  trendChart.setOption({
    tooltip: { trigger: 'axis' },
    legend: { data: ['攻击次数', '封禁IP'], bottom: 0 },
    grid: { left: '3%', right: '4%', top: '10%', bottom: '10%', containLabel: true },
    xAxis: { type: 'category', data: trendData.value.dates },
    yAxis: { type: 'value' },
    series: [
      { name: '攻击次数', type: 'line', smooth: true, data: trendData.value.attacks, lineStyle: { color: '#c62828' }, areaStyle: { opacity: 0.3, color: '#c62828' } },
      { name: '封禁IP', type: 'line', smooth: true, data: trendData.value.blocks, lineStyle: { color: '#477779' }, areaStyle: { opacity: 0.3, color: '#477779' } }
    ]
  })
}

// 初始化攻击类型饼图
const initTypeChart = () => {
  if (!typeChartRef.value) return
  if (typeChart) typeChart.dispose()
  
  typeChart = echarts.init(typeChartRef.value)
  typeChart.setOption({
    tooltip: { trigger: 'item' },
    legend: { orient: 'vertical', left: 'left' },
    series: [{
      type: 'pie',
      radius: '50%',
      data: attackTypeData.value,
      emphasis: { scale: true },
      label: { show: true, formatter: '{b}: {d}%' },
      color: ['#c62828', '#ed6c02', '#477779', '#7b1fa2', '#2e7d32']
    }]
  })
}

// 初始化流量图
const initTrafficChart = () => {
  if (!trafficChartRef.value) return
  if (trafficChart) trafficChart.dispose()
  
  trafficChart = echarts.init(trafficChartRef.value)
  trafficChart.setOption({
    tooltip: { trigger: 'axis' },
    legend: { data: ['入站流量', '出站流量'], bottom: 0 },
    grid: { left: '3%', right: '4%', top: '10%', bottom: '10%', containLabel: true },
    xAxis: { type: 'category', data: Array.from({ length: 60 }, (_, i) => `${i}s`) },
    yAxis: { type: 'value', name: 'Mbps' },
    series: [
      { name: '入站流量', type: 'line', smooth: true, data: Array.from({ length: 60 }, () => Math.random() * 100), lineStyle: { color: '#477779' }, areaStyle: { opacity: 0.3 } },
      { name: '出站流量', type: 'line', smooth: true, data: Array.from({ length: 60 }, () => Math.random() * 80), lineStyle: { color: '#ed6c02' }, areaStyle: { opacity: 0.3 } }
    ]
  })
}

// 开始实时监控
const startRealTime = () => {
  if (realtimeInterval) clearInterval(realtimeInterval)
  realtimeLoading.value = true
  
  setTimeout(() => {
    realtimeInterval = setInterval(() => {
      if (trafficChart) {
        const newInbound = Math.random() * 100
        const newOutbound = Math.random() * 80
        const option = trafficChart.getOption()
        const inboundData = option.series[0].data
        const outboundData = option.series[1].data
        inboundData.push(newInbound)
        outboundData.push(newOutbound)
        if (inboundData.length > 60) inboundData.shift()
        if (outboundData.length > 60) outboundData.shift()
        trafficChart.setOption({ series: [{ data: inboundData }, { data: outboundData }] })
      }
    }, 1000)
    realtimeLoading.value = false
    ElMessage.success('实时监控已启动')
  }, 500)
}

// 保存设置
const saveAllSettings = async () => {
  saving.value = true
  await new Promise(resolve => setTimeout(resolve, 1000))
  ElMessage.success('设置已保存')
  saving.value = false
}

const refreshSettings = () => {
  ElMessage.success('刷新成功')
  // 刷新图表数据
  // initMapChart()
  initTrendChart()
  initTypeChart()
}

// 防火墙操作
const addPortRule = () => {
  firewallForm.rules.push({ port: '', protocol: 'tcp', source: '0.0.0.0/0' })
}

const removePortRule = (idx) => {
  firewallForm.rules.splice(idx, 1)
}

const addBlacklist = () => {
  if (newBlacklistIp.value) {
    firewallForm.blacklist.push(newBlacklistIp.value)
    newBlacklistIp.value = ''
  }
}

const removeBlacklist = (idx) => {
  firewallForm.blacklist.splice(idx, 1)
}

const addWhitelist = () => {
  if (newWhitelistIp.value) {
    firewallForm.whitelist.push(newWhitelistIp.value)
    newWhitelistIp.value = ''
  }
}

const removeWhitelist = (idx) => {
  firewallForm.whitelist.splice(idx, 1)
}

// 其他操作
const renewCert = (cert) => {
  ElMessage.info(`续期 ${cert.domain}`)
}

const backupNow = async () => {
  backingUp.value = true
  await new Promise(resolve => setTimeout(resolve, 2000))
  ElMessage.success('备份创建成功')
  backingUp.value = false
}

const clearEvents = () => {
  recentEvents.value = []
  ElMessage.success('事件列表已清空')
}

// 窗口大小变化时重新调整图表
const handleResize = () => {
  if (mapChart) mapChart.resize()
  if (trendChart) trendChart.resize()
  if (typeChart) typeChart.resize()
  if (trafficChart) trafficChart.resize()
}

onMounted(() => {
  setTimeout(() => {
    // initMapChart()
    initTrendChart()
    initTypeChart()
    initTrafficChart()
  }, 500)
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  if (realtimeInterval) clearInterval(realtimeInterval)
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped>
.security-container {
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
  flex-wrap: wrap;
}

.top-left {
  display: flex;
  gap: 8px;
}

.top-right {
  flex-shrink: 0;
}

.security-tabs {
  margin-right: 16px;
}

.content-panel {
  background: #ffffff;
  border-radius: 4px;
  padding: 20px;
  flex: 1;
  overflow-y: auto;
}

/* 统计卡片 */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  background: #f9fafb;
  border-radius: 8px;
  padding: 16px;
  display: flex;
  align-items: center;
  gap: 16px;
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
}

.stat-number {
  font-size: 28px;
  font-weight: bold;
  color: #1f2937;
}

.stat-label {
  font-size: 12px;
  color: #9ca3af;
  margin-top: 4px;
}

/* 图表区域 */
.map-section, .chart-section, .two-columns {
  margin-bottom: 24px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: #1f2937;
  margin: 0;
  padding-left: 8px;
  border-left: 3px solid #477779;
}

.map-container {
  width: 100%;
  height: 400px;
  background: #f5f5f5;
  border-radius: 8px;
}

.trend-container {
  width: 100%;
  height: 300px;
}

.type-container {
  width: 100%;
  height: 280px;
}

.traffic-container {
  width: 100%;
  height: 300px;
}

.two-columns {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}

.column {
  background: #f9fafb;
  border-radius: 8px;
  padding: 16px;
}

.events-list {
  max-height: 280px;
  overflow-y: auto;
}

.event-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border-bottom: 1px solid #e5e7eb;
  font-size: 13px;
}

.event-item.high {
  border-left: 3px solid #c62828;
  background: #fef3f3;
}

.event-item.medium {
  border-left: 3px solid #ed6c02;
}

.event-item.low {
  border-left: 3px solid #477779;
}

.event-time {
  width: 70px;
  color: #9ca3af;
  font-size: 11px;
}

.event-content {
  flex: 1;
}

.event-type {
  font-weight: 500;
  color: #1f2937;
  margin-right: 8px;
}

.event-desc {
  color: #6b7280;
  font-size: 12px;
}

.event-ip {
  font-family: monospace;
  color: #477779;
  font-size: 11px;
}

.empty {
  text-align: center;
  color: #9ca3af;
  padding: 40px;
}

/* 防火墙设置样式 */
.settings-section {
  margin-bottom: 32px;
  padding-bottom: 24px;
  border-bottom: 1px solid #e5e7eb;
}

.section-desc {
  font-size: 12px;
  color: #9ca3af;
  margin-top: 8px;
}

.port-rules {
  margin-bottom: 16px;
}

.rule-header {
  display: grid;
  grid-template-columns: 120px 100px 150px 80px;
  gap: 12px;
  padding: 8px 0;
  font-size: 13px;
  color: #9ca3af;
  border-bottom: 1px solid #e5e7eb;
}

.rule-row {
  display: grid;
  grid-template-columns: 120px 100px 150px 80px;
  gap: 12px;
  align-items: center;
  padding: 8px 0;
}

.rule-actions {
  display: flex;
  gap: 4px;
}

.ip-list {
  border: 1px solid #e5e7eb;
  border-radius: 4px;
  padding: 12px;
}

.ip-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 0;
  border-bottom: 1px solid #f0f0f0;
}

.ip-item:last-child {
  border-bottom: none;
}

.add-ip {
  display: flex;
  gap: 8px;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid #e5e7eb;
}
</style>
<style>
.top-panel .el-button {
  height: 28px !important;
  line-height: 28px !important;
  padding: 0 12px !important;
  font-size: 12px !important;
}
.top-panel .el-tabs__item {
  height: 32px !important;
  line-height: 32px !important;
  font-size: 13px !important;
}
</style>
