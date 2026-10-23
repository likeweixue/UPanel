<template>
  <div class="page dashboard">
    <header class="page__header">
      <div>
        <div class="page__title-row">
          <h1 class="page__title">总览</h1>
          <span v-if="info && info.hostname" class="chip">
            <UIcon name="server" :size="13" /> {{ info.hostname }}
          </span>
        </div>
        <p class="page__subtitle">今天是 {{ today }} · 面板与站点运行状态一览</p>
      </div>
      <div class="page__actions">
        <UButton icon="refresh" :loading="loading" label="刷新" @click="loadAll" />
        <UButton icon="plus" variant="primary" label="创建网站" @click="goCreateSite" />
      </div>
    </header>

    <div v-if="loadError" class="alert alert--warning" role="alert">
      <UIcon name="alert" :size="16" class="alert__icon" />
      <div class="alert__body">
        <p class="alert__title">部分数据加载失败</p>
        <p>{{ loadError }}</p>
      </div>
    </div>

    <div class="widget-grid">
      <UWidget title="资源概况" subtitle="当前面板管理的资源数量" icon="dashboard" :span="12" flush>
        <template #actions>
          <UButton variant="ghost" size="sm" icon="refresh" label="刷新" @click="loadAll" />
        </template>
        <div class="stat-grid">
          <div class="stat">
            <p class="stat__label"><UIcon name="globe" :size="14" /> 网站</p>
            <p class="stat__value">{{ stats.sites }}</p>
            <p class="stat__meta">{{ stats.runningSites }} 个运行中 · {{ stats.sslSites }} 个已启用 HTTPS</p>
          </div>
          <div class="stat">
            <p class="stat__label"><UIcon name="database" :size="14" /> 数据库</p>
            <p class="stat__value">{{ stats.databases }}</p>
            <p class="stat__meta">{{ stats.runningDatabases }} 个运行中</p>
          </div>
          <div class="stat">
            <p class="stat__label"><UIcon name="box" :size="14" /> 容器</p>
            <p class="stat__value">{{ stats.containers }}</p>
            <p class="stat__meta">{{ stats.runningContainers }} 个运行中</p>
          </div>
          <div class="stat">
            <p class="stat__label"><UIcon name="store" :size="14" /> 应用</p>
            <p class="stat__value">{{ stats.apps }}</p>
            <p class="stat__meta">{{ stats.runningApps }} 个运行中</p>
          </div>
        </div>
      </UWidget>

      <UWidget id="health" title="系统资源" subtitle="CPU、内存、磁盘与负载" icon="activity" :span="7">
        <template #actions>
          <UStatusDot :status="info ? 'running' : 'unknown'" :text="info ? '实时' : '等待数据'" />
        </template>
        <div class="stack">
          <div class="row" style="justify-content: space-between; align-items: flex-end">
            <div class="stack" style="gap: 14px; flex: 1">
              <div class="meter">
                <div class="meter__head">
                  <span class="meter__label">CPU</span>
                  <span class="meter__value">{{ formatPercent(info && info.cpu_percent, 1) }}</span>
                </div>
                <div class="meter__track">
                  <div
                    class="meter__fill"
                    :class="`meter__fill--${formatMeterTone(info && info.cpu_percent)}`"
                    :style="{ width: `${clampPercent(info && info.cpu_percent)}%` }"
                  />
                </div>
              </div>
              <div class="meter">
                <div class="meter__head">
                  <span class="meter__label">内存</span>
                  <span class="meter__value">{{ formatPercent(info && info.memory_percent, 1) }}</span>
                </div>
                <div class="meter__track">
                  <div
                    class="meter__fill"
                    :class="`meter__fill--${formatMeterTone(info && info.memory_percent)}`"
                    :style="{ width: `${clampPercent(info && info.memory_percent)}%` }"
                  />
                </div>
                <p class="meter__meta">
                  {{ formatBytes(info && info.memory_used) }} / {{ formatBytes(info && info.memory_total) }}
                </p>
              </div>
              <div class="meter">
                <div class="meter__head">
                  <span class="meter__label">磁盘</span>
                  <span class="meter__value">{{ formatPercent(info && info.disk_percent, 1) }}</span>
                </div>
                <div class="meter__track">
                  <div
                    class="meter__fill"
                    :class="`meter__fill--${formatMeterTone(info && info.disk_percent)}`"
                    :style="{ width: `${clampPercent(info && info.disk_percent)}%` }"
                  />
                </div>
                <p class="meter__meta">
                  {{ formatBytes(info && info.disk_used) }} / {{ formatBytes(info && info.disk_total) }}
                </p>
              </div>
            </div>
            <div class="dashboard__spark" aria-hidden="true">
              <USparkline :values="cpuSamples" label="CPU 使用率趋势" :max="100" />
            </div>
          </div>
          <div class="dashboard__mini-grid">
            <div class="dashboard__mini">
              <span class="muted">负载 (1m / 5m / 15m)</span>
              <strong>{{ loadText }}</strong>
            </div>
            <div class="dashboard__mini">
              <span class="muted">主机运行时间</span>
              <strong>{{ (info && info.uptime) || '—' }}</strong>
            </div>
            <div class="dashboard__mini">
              <span class="muted">面板运行时间</span>
              <strong>{{ (info && info.panel_uptime) || '—' }}</strong>
            </div>
          </div>
        </div>
      </UWidget>

      <UWidget title="网站网关" subtitle="统一 Nginx 网关，承载全部站点" icon="network" :span="5">
        <template #actions>
          <UButton
            v-if="gateway.state !== 'running'"
            variant="primary"
            size="sm"
            icon="wrench"
            label="修复网关"
            :loading="repairing"
            @click="repairGateway"
          />
        </template>
        <div class="stack">
          <div class="dashboard__gateway">
            <UStatusDot :status="gatewayTone" :text="gatewayText" />
            <span v-if="gateway.state === 'running'" class="badge badge--success">80 / 443</span>
            <span v-else class="badge badge--warning">需要处理</span>
          </div>
          <dl class="info-list">
            <div class="info-list__row">
              <dt class="info-list__label">容器</dt>
              <dd class="info-list__value mono">{{ gateway.container || 'upanel-gateway' }}</dd>
            </div>
            <div class="info-list__row">
              <dt class="info-list__label">HTTP 端口</dt>
              <dd class="info-list__value">{{ gateway.httpPort || '—' }}</dd>
            </div>
            <div class="info-list__row">
              <dt class="info-list__label">HTTPS 端口</dt>
              <dd class="info-list__value">{{ gateway.httpsPort || '—' }}</dd>
            </div>
            <div class="info-list__row">
              <dt class="info-list__label">镜像</dt>
              <dd class="info-list__value mono">{{ gateway.image || 'nginx:alpine' }}</dd>
            </div>
          </dl>
          <div v-if="gateway.error" class="alert alert--warning">
            <UIcon name="alert" :size="15" class="alert__icon" />
            <div class="alert__body">{{ gateway.error }}</div>
          </div>
          <div class="row">
            <UButton variant="secondary" size="sm" icon="network" label="查看站点" @click="go('/websites?tab=gateway')" />
            <UButton variant="ghost" size="sm" icon="refresh" label="刷新" @click="loadGateway" />
          </div>
        </div>
      </UWidget>

      <UWidget title="最近网站" subtitle="按域名排序的站点状态" icon="globe" :span="7">
        <template #actions>
          <UButton variant="ghost" size="sm" label="全部网站" icon="arrow-right" @click="go('/websites')" />
        </template>
        <div v-if="sites.length === 0" class="empty" style="padding: 26px 12px">
          <p class="empty__title">还没有网站</p>
          <p class="empty__desc">创建第一个网站，域名解析后即可对外访问。</p>
          <UButton variant="primary" size="sm" icon="plus" label="创建网站" @click="goCreateSite" />
        </div>
        <div v-else class="table-wrap" style="border: 0">
          <table class="table table--compact">
            <caption class="sr-only">最近网站状态</caption>
            <thead>
              <tr>
                <th scope="col">域名</th>
                <th scope="col">类型</th>
                <th scope="col">状态</th>
                <th scope="col">HTTPS</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="site in sites.slice(0, 5)" :key="site.domain">
                <td>
                  <router-link class="table__primary" :to="`/websites?tab=domains`">{{ site.domain }}</router-link>
                </td>
                <td><span class="badge">{{ siteTypeLabel(site.type) }}</span></td>
                <td><UStatusDot :status="site.status" :text="statusLabel(site.status)" /></td>
                <td>
                  <span :class="['badge', site.sslEnabled ? 'badge--success' : 'badge--warning']">
                    {{ site.sslEnabled ? '已启用' : '未启用' }}
                  </span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </UWidget>

      <UWidget title="快捷操作" subtitle="常用动作一键直达" icon="zap" :span="5">
        <div class="quick-actions">
          <button type="button" class="quick-action" @click="goCreateSite">
            <span class="quick-action__icon"><UIcon name="plus" :size="15" /></span>
            创建网站
          </button>
          <button type="button" class="quick-action" @click="go('/databases?action=create')">
            <span class="quick-action__icon"><UIcon name="database" :size="15" /></span>
            创建数据库
          </button>
          <button type="button" class="quick-action" @click="go('/files?action=upload')">
            <span class="quick-action__icon"><UIcon name="upload" :size="15" /></span>
            上传文件
          </button>
          <button type="button" class="quick-action" @click="go('/apps')">
            <span class="quick-action__icon"><UIcon name="store" :size="15" /></span>
            应用商店
          </button>
          <button type="button" class="quick-action" @click="go('/security')">
            <span class="quick-action__icon"><UIcon name="shield" :size="15" /></span>
            安全检查
          </button>
          <button type="button" class="quick-action" @click="go('/settings?tab=gateway')">
            <span class="quick-action__icon"><UIcon name="settings" :size="15" /></span>
            网关设置
          </button>
        </div>
      </UWidget>

      <UWidget title="安全提示" subtitle="来自安全中心的风险项" icon="shield" :span="6">
        <template #actions>
          <UButton variant="ghost" size="sm" icon="arrow-right" label="安全中心" @click="go('/security')" />
        </template>
        <div v-if="securityWarnings.length === 0" class="alert alert--success">
          <UIcon name="check-circle" :size="16" class="alert__icon" />
          <div class="alert__body">未发现明显风险项</div>
        </div>
        <div v-else class="warning-list">
          <div v-for="(warning, index) in securityWarnings.slice(0, 4)" :key="index" class="warning-item">
            <UIcon name="alert" :size="15" />
            <span>{{ warning }}</span>
          </div>
          <p v-if="securityWarnings.length > 4" class="muted" style="font-size: 12px">
            还有 {{ securityWarnings.length - 4 }} 项风险，前往安全中心查看
          </p>
        </div>
      </UWidget>

      <UWidget title="主机信息" subtitle="系统与 Docker 版本" icon="server" :span="6">
        <template #actions>
          <UButton variant="ghost" size="sm" icon="arrow-right" label="监控" @click="go('/monitor')" />
        </template>
        <dl class="info-list">
          <div class="info-list__row">
            <dt class="info-list__label">操作系统</dt>
            <dd class="info-list__value">{{ (info && info.platform) || (info && info.os) || '—' }}</dd>
          </div>
          <div class="info-list__row">
            <dt class="info-list__label">内核 / 架构</dt>
            <dd class="info-list__value">{{ (info && info.kernel) || '—' }}</dd>
          </div>
          <div class="info-list__row">
            <dt class="info-list__label">CPU 核心</dt>
            <dd class="info-list__value">{{ (info && info.cpu) || '—' }}</dd>
          </div>
          <div class="info-list__row">
            <dt class="info-list__label">Docker</dt>
            <dd class="info-list__value">
              <UStatusDot
                :status="info && info.docker_status === 'running' ? 'running' : 'error'"
                :text="(info && info.docker_version) || '未连接'"
              />
            </dd>
          </div>
          <div class="info-list__row">
            <dt class="info-list__label">网络收发</dt>
            <dd class="info-list__value">
              ↓ {{ formatBytes(info && info.network_rx_bytes) }} · ↑ {{ formatBytes(info && info.network_tx_bytes) }}
            </dd>
          </div>
        </dl>
      </UWidget>

      <UWidget id="activity" title="最近活动" subtitle="最近的登录审计记录" icon="clock" :span="12">
        <template #actions>
          <UButton variant="ghost" size="sm" icon="arrow-right" label="登录审计" @click="go('/security?tab=audit')" />
        </template>
        <div v-if="activity.length === 0" class="empty" style="padding: 24px 12px">
          <p class="empty__title">暂无登录记录</p>
          <p class="empty__desc">登录成功或失败后，记录会出现在这里。</p>
        </div>
        <div v-else class="timeline">
          <div v-for="(entry, index) in activity.slice(0, 6)" :key="index" class="timeline__item">
            <span :class="['timeline__dot', entry.success ? 'timeline__dot--success' : 'timeline__dot--danger']" />
            <div class="timeline__body">
              <div class="split">
                <span>
                  {{ entry.success ? '登录成功' : '登录失败' }}
                  <span class="muted">· {{ entry.username || '未知用户' }}</span>
                </span>
                <span class="timeline__meta">{{ relativeTime(entry.time) }}</span>
              </div>
              <div class="timeline__meta">
                {{ entry.ip || '未知 IP' }}<span v-if="entry.reason"> · {{ entry.reason }}</span>
              </div>
            </div>
          </div>
        </div>
      </UWidget>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import UButton from '@/components/ui/UButton.vue'
import UIcon from '@/components/ui/UIcon.vue'
import UStatusDot from '@/components/ui/UStatusDot.vue'
import UWidget from '@/components/ui/UWidget.vue'
import USparkline from '@/components/ui/USparkline.vue'
import {
  appApi,
  containerApi,
  dbApi,
  errText,
  gatewayApi,
  securityApi,
  siteApi,
  systemApi,
} from '@/api'
import {
  clampPercent,
  formatBytes,
  formatMeterTone,
  formatPercent,
  relativeTime,
  statusLabel,
} from '@/composables/useFormat'
import { toastError, toastSuccess } from '@/composables/useToast'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const repairing = ref(false)
const loadError = ref('')
const info = ref(null)
const sites = ref([])
const databases = ref([])
const containers = ref([])
const apps = ref([])
const gateway = reactive({ state: 'unknown', httpPort: '', httpsPort: '', image: '', container: '', error: '' })
const security = ref(null)
const activity = ref([])
const cpuSamples = ref([])
const netPrev = reactive({ rx: null, tx: null, at: null, rxRate: 0, txRate: 0 })

const stats = computed(() => ({
  sites: sites.value.length,
  runningSites: sites.value.filter((s) => s.status === 'running').length,
  sslSites: sites.value.filter((s) => s.sslEnabled).length,
  databases: databases.value.length,
  runningDatabases: databases.value.filter((d) => d.status === 'running').length,
  containers: containers.value.length,
  runningContainers: containers.value.filter((c) => c.state === 'running').length,
  apps: apps.value.length,
  runningApps: apps.value.filter((a) => a.status === 'running').length,
}))

const securityWarnings = computed(() => (security.value && security.value.warnings) || [])

const loadText = computed(() => {
  if (!info.value) return '—'
  return [info.value.load1, info.value.load5, info.value.load15]
    .map((value) => (Number.isFinite(Number(value)) ? Number(value).toFixed(2) : '0.00'))
    .join(' / ')
})

const gatewayTone = computed(() => {
  if (gateway.state === 'running') return 'running'
  if (gateway.state === 'docker_unavailable') return 'error'
  return 'warning'
})

const gatewayText = computed(() => {
  if (gateway.state === 'running') return '网关运行中'
  if (gateway.state === 'not_installed') return '网关尚未创建'
  if (gateway.state === 'docker_unavailable') return 'Docker 未连接'
  if (gateway.state === 'exited' || gateway.state === 'stopped') return '网关已停止'
  return statusLabel(gateway.state)
})

const today = new Intl.DateTimeFormat('zh-CN', {
  year: 'numeric',
  month: 'long',
  day: 'numeric',
  weekday: 'long',
}).format(new Date())

function siteTypeLabel(type) {
  return { static: '静态', php: 'PHP', proxy: '反代' }[type] || type
}

function go(path) {
  router.push(path)
}

function goCreateSite() {
  router.push({ path: '/websites', query: { action: 'create' } })
}

async function loadSystem() {
  const res = await systemApi.info()
  const data = res.data || {}
  info.value = data
  if (Number.isFinite(Number(data.cpu_percent))) {
    cpuSamples.value.push(Number(data.cpu_percent))
    if (cpuSamples.value.length > 40) cpuSamples.value.shift()
  }
  if (Number.isFinite(Number(data.network_rx_bytes))) {
    const now = Date.now()
    if (netPrev.at) {
      const seconds = Math.max(1, (now - netPrev.at) / 1000)
      netPrev.rxRate = Math.max(0, (data.network_rx_bytes - netPrev.rx) / seconds)
      netPrev.txRate = Math.max(0, (data.network_tx_bytes - netPrev.tx) / seconds)
    }
    netPrev.rx = data.network_rx_bytes
    netPrev.tx = data.network_tx_bytes
    netPrev.at = now
  }
}

async function loadGateway() {
  try {
    const res = await gatewayApi.status()
    Object.assign(gateway, { error: '' }, res.data || {})
  } catch (error) {
    gateway.state = 'docker_unavailable'
    gateway.error = errText(error)
  }
}

async function loadAll() {
  if (loading.value) return
  loading.value = true
  loadError.value = ''
  const failures = []
  const results = await Promise.allSettled([
    loadSystem(),
    siteApi.list().then((res) => (sites.value = res.data || [])),
    dbApi.list().then((res) => (databases.value = res.data || [])),
    containerApi.list().then((res) => (containers.value = res.data || [])),
    appApi.installed().then((res) => (apps.value = res.data || [])),
    loadGateway(),
    securityApi.overview().then((res) => (security.value = res.data || null)),
    securityApi.audit(10).then((res) => (activity.value = res.data || [])),
  ])
  const labels = ['系统信息', '网站', '数据库', '容器', '应用', '网关', '安全中心', '登录审计']
  results.forEach((result, index) => {
    if (result.status === 'rejected') {
      failures.push(`${labels[index]}：${errText(result.reason)}`)
    }
  })
  if (failures.length > 0) {
    const nonDocker = failures.filter((item) => !item.includes('Docker 未连接'))
    if (nonDocker.length === 0) {
      loadError.value = 'Docker 未连接：容器管理与应用安装暂不可用，已有网站、数据库列表仍可查看。'
    } else {
      const unique = [...new Set(nonDocker)]
      loadError.value = unique.join('；')
    }
  }
  loading.value = false
}

async function repairGateway() {
  repairing.value = true
  try {
    await gatewayApi.ensure()
    toastSuccess('网关已就绪', '站点配置已重新加载')
    await loadGateway()
    await loadAll()
  } catch (error) {
    toastError('修复网关失败', errText(error))
  } finally {
    repairing.value = false
  }
}

function scrollToSection() {
  const section = route.query.section
  if (!section) return
  setTimeout(() => {
    const el = document.getElementById(String(section))
    if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }, 250)
}

let pollTimer = null

onMounted(async () => {
  await loadAll()
  scrollToSection()
  pollTimer = setInterval(() => {
    if (document.visibilityState === 'visible') loadSystem().catch(() => {})
  }, 10000)
})

onBeforeUnmount(() => {
  if (pollTimer) clearInterval(pollTimer)
})

watch(() => route.query.section, scrollToSection)
</script>

<style scoped>
.dashboard__spark {
  width: 180px;
  flex: 0 0 auto;
  padding: 0 0 2px 16px;
}

.dashboard__mini-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 1px;
  overflow: hidden;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-border);
}

.dashboard__mini {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 9px 11px;
  background: var(--color-surface);
  font-size: var(--text-sm);
}

.dashboard__mini strong {
  font-size: var(--text-md);
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.dashboard__gateway {
  display: flex;
  align-items: center;
  gap: 10px;
}

@media (max-width: 720px) {
  .dashboard__spark {
    display: none;
  }

  .dashboard__mini-grid {
    grid-template-columns: 1fr;
  }
}
</style>
