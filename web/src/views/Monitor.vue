<template>
  <div class="page">
    <div class="page-shell">
    <header class="page__header">
      <div>
        <div class="page__title-row">
          <h1 class="page__title">资源监控</h1>
          <UStatusDot :status="info ? 'running' : 'unknown'" :text="info ? '实时采样' : '等待数据'" />
        </div>
        <p class="page__subtitle">每 5 秒自动刷新 · 数据来自宿主机与 Docker</p>
      </div>
      <div class="page__actions">
        <UButton icon="refresh" label="立即刷新" :loading="loading" @click="loadInfo" />
      </div>
    </header>

    <div v-if="error" class="alert alert--danger" role="alert">
      <UIcon name="alert" :size="16" class="alert__icon" />
      <div class="alert__body">{{ error }}</div>
    </div>

    <UTabs v-model="activeTab" :tabs="tabs" aria-label="监控标签页">
      <div v-if="activeTab === 'overview'" class="widget-grid">
        <UWidget title="CPU" subtitle="宿主机整体使用率" icon="cpu" :span="6">
          <div class="stack">
            <div class="monitor__big">{{ formatPercent(info && info.cpu_percent, 1) }}</div>
            <div class="meter">
              <div class="meter__track">
                <div
                  class="meter__fill"
                  :class="`meter__fill--${formatMeterTone(info && info.cpu_percent)}`"
                  :style="{ width: `${clampPercent(info && info.cpu_percent)}%` }"
                />
              </div>
              <p class="meter__meta">{{ info?.cpu || '—' }} 个逻辑核心 · 负载 {{ loadText }}</p>
            </div>
            <USparkline :values="history.cpu" label="CPU 使用率历史" :max="100" />
          </div>
        </UWidget>

        <UWidget title="内存" subtitle="物理内存使用情况" icon="memory" :span="6">
          <div class="stack">
            <div class="monitor__big">{{ formatPercent(info && info.memory_percent, 1) }}</div>
            <div class="meter">
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
            <USparkline :values="history.memory" label="内存使用率历史" :max="100" />
          </div>
        </UWidget>

        <UWidget title="磁盘" subtitle="根分区容量" icon="disk" :span="4">
          <div class="stack">
            <div class="monitor__big">{{ formatPercent(info && info.disk_percent, 1) }}</div>
            <div class="meter">
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
        </UWidget>

        <UWidget title="负载" subtitle="1 / 5 / 15 分钟平均负载" icon="activity" :span="4">
          <div class="stack">
            <div class="monitor__load-row">
              <div><span class="subtle">1m</span><strong>{{ numberText(info && info.load1) }}</strong></div>
              <div><span class="subtle">5m</span><strong>{{ numberText(info && info.load5) }}</strong></div>
              <div><span class="subtle">15m</span><strong>{{ numberText(info && info.load15) }}</strong></div>
            </div>
            <p class="muted" style="font-size: 12px">负载持续高于核心数时，说明 CPU 或 IO 可能成为瓶颈。</p>
          </div>
        </UWidget>

        <UWidget title="运行时间" subtitle="主机与面板进程" icon="clock" :span="4">
          <dl class="info-list">
            <div class="info-list__row">
              <dt class="info-list__label">主机</dt>
              <dd class="info-list__value">{{ info?.uptime || '—' }}</dd>
            </div>
            <div class="info-list__row">
              <dt class="info-list__label">面板</dt>
              <dd class="info-list__value">{{ info?.panel_uptime || '—' }}</dd>
            </div>
            <div class="info-list__row">
              <dt class="info-list__label">系统</dt>
              <dd class="info-list__value">{{ info?.platform || info?.os || '—' }}</dd>
            </div>
          </dl>
        </UWidget>
      </div>

      <div v-else-if="activeTab === 'disk'" class="widget-grid">
        <UWidget title="根分区" subtitle="挂载点 /" icon="disk" :span="7">
          <div class="stack">
            <div class="monitor__big">{{ formatPercent(info && info.disk_percent, 1) }}</div>
            <div class="meter">
              <div class="meter__track" style="height: 10px">
                <div
                  class="meter__fill"
                  :class="`meter__fill--${formatMeterTone(info && info.disk_percent)}`"
                  :style="{ width: `${clampPercent(info && info.disk_percent)}%` }"
                />
              </div>
            </div>
            <div class="stat-grid" style="margin-top: 6px">
              <div class="stat">
                <p class="stat__label">总容量</p>
                <p class="stat__value" style="font-size: 18px">{{ formatBytes(info && info.disk_total) }}</p>
              </div>
              <div class="stat">
                <p class="stat__label">已使用</p>
                <p class="stat__value" style="font-size: 18px">{{ formatBytes(info && info.disk_used) }}</p>
              </div>
              <div class="stat">
                <p class="stat__label">可用</p>
                <p class="stat__value" style="font-size: 18px">{{ formatBytes(diskFree) }}</p>
              </div>
            </div>
            <div class="alert alert--info">
              <UIcon name="info" :size="16" class="alert__icon" />
              <div class="alert__body">
                磁盘占用超过 90% 可能影响数据库与日志写入。建议定期清理 Docker 镜像、站点日志与旧备份。
              </div>
            </div>
          </div>
        </UWidget>
        <UWidget title="Swap" subtitle="交换分区使用情况" icon="memory" :span="5">
          <div class="stack">
            <div class="monitor__big">{{ formatPercent(info && info.swap_percent, 1) }}</div>
            <div class="meter">
              <div class="meter__track">
                <div
                  class="meter__fill"
                  :class="`meter__fill--${formatMeterTone(info && info.swap_percent)}`"
                  :style="{ width: `${clampPercent(info && info.swap_percent)}%` }"
                />
              </div>
              <p class="meter__meta">
                {{ formatBytes(info && info.swap_used) }} / {{ formatBytes(info && info.swap_total) }}
              </p>
            </div>
            <p class="muted" style="font-size: 12px">Swap 使用过高通常意味着内存不足。</p>
          </div>
        </UWidget>
      </div>

      <div v-else class="widget-grid">
        <UWidget title="网络吞吐" subtitle="所有网卡合计（实时速率）" icon="network" :span="7">
          <div class="stack">
            <div class="monitor__net-row">
              <div>
                <p class="subtle">下行 / 接收</p>
                <p class="monitor__net-value">{{ formatBytes(net.rxRate) }}/s</p>
                <p class="subtle" style="font-size: 11px">累计 {{ formatBytes(info && info.network_rx_bytes) }}</p>
              </div>
              <div>
                <p class="subtle">上行 / 发送</p>
                <p class="monitor__net-value">{{ formatBytes(net.txRate) }}/s</p>
                <p class="subtle" style="font-size: 11px">累计 {{ formatBytes(info && info.network_tx_bytes) }}</p>
              </div>
            </div>
            <USparkline :values="history.rx" label="下行速率历史" :color="'var(--color-info)'" :max="netMax" />
            <USparkline :values="history.tx" label="上行速率历史" :color="'var(--color-success)'" :max="netMax" />
            <p class="muted" style="font-size: 12px">采样间隔 5 秒；速率为两次采样的差值。</p>
          </div>
        </UWidget>
        <UWidget title="连接状态" subtitle="Docker 与主机网络" icon="server" :span="5">
          <dl class="info-list">
            <div class="info-list__row">
              <dt class="info-list__label">主机名</dt>
              <dd class="info-list__value">{{ info?.hostname || '—' }}</dd>
            </div>
            <div class="info-list__row">
              <dt class="info-list__label">架构</dt>
              <dd class="info-list__value">{{ info?.kernel || info?.arch || '—' }}</dd>
            </div>
            <div class="info-list__row">
              <dt class="info-list__label">Docker</dt>
              <dd class="info-list__value">
                <UStatusDot
                  :status="info && info.docker_status === 'running' ? 'running' : 'error'"
                  :text="info?.docker_version || '未连接'"
                />
              </dd>
            </div>
            <div class="info-list__row">
              <dt class="info-list__label">容器</dt>
              <dd class="info-list__value">{{ info?.running || 0 }} 运行 / {{ info?.containers || 0 }} 总计</dd>
            </div>
          </dl>
        </UWidget>
      </div>
    </UTabs>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import UButton from '@/components/ui/UButton.vue'
import UIcon from '@/components/ui/UIcon.vue'
import UStatusDot from '@/components/ui/UStatusDot.vue'
import UTabs from '@/components/ui/UTabs.vue'
import UWidget from '@/components/ui/UWidget.vue'
import USparkline from '@/components/ui/USparkline.vue'
import { errText, systemApi } from '@/api'
import {
  clampPercent,
  formatBytes,
  formatMeterTone,
  formatPercent,
} from '@/composables/useFormat'

const route = useRoute()
const router = useRouter()

const info = ref(null)
const loading = ref(false)
const error = ref('')
const activeTab = ref(['overview', 'disk', 'network'].includes(route.query.tab) ? route.query.tab : 'overview')
const tabs = [
  { id: 'overview', label: '概览', icon: 'activity' },
  { id: 'disk', label: '磁盘', icon: 'disk' },
  { id: 'network', label: '网络', icon: 'network' },
]

const history = reactive({ cpu: [], memory: [], rx: [], tx: [] })
const net = reactive({ rxRate: 0, txRate: 0, prevRx: null, prevTx: null, at: null })
let timer = null

const loadText = computed(() =>
  info.value ? `${Number(info.value.load1 || 0).toFixed(2)}` : '—'
)
const diskFree = computed(() => {
  if (!info.value) return 0
  return Math.max(0, Number(info.value.disk_total) - Number(info.value.disk_used))
})
const netMax = computed(() => {
  const values = [...history.rx, ...history.tx, 1024]
  return Math.max(...values) * 1.15
})

function numberText(value) {
  const n = Number(value)
  return Number.isFinite(n) ? n.toFixed(2) : '0.00'
}

function pushSample(list, value) {
  list.push(Number(value) || 0)
  if (list.length > 60) list.shift()
}

async function loadInfo() {
  if (loading.value) return
  loading.value = true
  try {
    const res = await systemApi.info()
    const data = res.data || {}
    info.value = data
    error.value = ''
    pushSample(history.cpu, data.cpu_percent)
    pushSample(history.memory, data.memory_percent)
    const now = Date.now()
    if (net.at && Number.isFinite(Number(data.network_rx_bytes))) {
      const seconds = Math.max(1, (now - net.at) / 1000)
      net.rxRate = Math.max(0, (data.network_rx_bytes - net.prevRx) / seconds)
      net.txRate = Math.max(0, (data.network_tx_bytes - net.prevTx) / seconds)
      pushSample(history.rx, net.rxRate)
      pushSample(history.tx, net.txRate)
    }
    net.prevRx = Number(data.network_rx_bytes) || 0
    net.prevTx = Number(data.network_tx_bytes) || 0
    net.at = now
  } catch (e) {
    error.value = errText(e, '读取系统信息失败')
  } finally {
    loading.value = false
  }
}

watch(activeTab, (tab) => {
  if (route.query.tab !== tab) router.replace({ path: route.path, query: { ...route.query, tab } })
})

watch(
  () => route.query.tab,
  (tab) => {
    if (!tab) {
      activeTab.value = 'overview'
      return
    }
    if (['overview', 'disk', 'network'].includes(tab) && tab !== activeTab.value) activeTab.value = tab
  }
)

onMounted(() => {
  loadInfo()
  timer = setInterval(() => {
    if (document.visibilityState === 'visible') loadInfo()
  }, 5000)
})

onBeforeUnmount(() => {
  if (timer) clearInterval(timer)
})
</script>

<style scoped>
.monitor__big {
  font-size: 32px;
  font-weight: 650;
  letter-spacing: -0.02em;
  line-height: 1.1;
}

.monitor__load-row {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 1px;
  overflow: hidden;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-border);
}

.monitor__load-row > div {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 10px 12px;
  background: var(--color-surface);
  font-size: 12px;
}

.monitor__load-row strong {
  font-size: 18px;
  font-weight: 650;
  font-variant-numeric: tabular-nums;
}

.monitor__net-row {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
  padding-bottom: 6px;
  border-bottom: 1px solid var(--color-border);
}

.monitor__net-value {
  margin: 2px 0;
  font-size: 24px;
  font-weight: 650;
  font-variant-numeric: tabular-nums;
}
</style>
