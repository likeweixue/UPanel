<template>
  <div class="page">
    <div class="page-shell">
    <header class="page__header">
      <div>
        <div class="page__title-row">
          <h1 class="page__title">安全中心</h1>
          <UStatusDot
            :status="riskCount === 0 ? 'running' : 'warning'"
            :text="riskCount === 0 ? '基线良好' : `${riskCount} 项待处理`"
          />
        </div>
        <p class="page__subtitle">安全基线检查、登录审计与防火墙状态（只读）</p>
      </div>
      <div class="page__actions">
        <UButton icon="refresh" label="刷新" :loading="loading" @click="loadAll" />
        <UButton icon="key" label="修改密码" @click="openPassword" />
        <UButton variant="primary" icon="settings" label="面板设置" @click="go('/settings')" />
      </div>
    </header>

    <div v-if="error" class="alert alert--danger" role="alert">
      <UIcon name="alert" :size="16" class="alert__icon" />
      <div class="alert__body">{{ error }}</div>
    </div>

    <UTabs v-model="activeTab" :tabs="tabs" aria-label="安全中心标签页">
      <div v-if="activeTab === 'overview'" class="stack">
        <div class="widget-grid">
          <UWidget title="安全基线" subtitle="逐项检查面板常见风险" icon="shield" :span="7">
            <div class="checklist">
              <div v-for="check in checks" :key="check.id" class="checklist__item">
                <span
                  class="checklist__icon"
                  :class="check.ok ? 'checklist__icon--ok' : 'checklist__icon--warn'"
                >
                  <UIcon :name="check.ok ? 'check' : 'alert'" :size="13" />
                </span>
                <div class="checklist__body">
                  <p class="checklist__title">{{ check.title }}</p>
                  <p class="checklist__desc">{{ check.detail }}</p>
                </div>
                <span :class="['badge', check.ok ? 'badge--success' : 'badge--warning']">
                  {{ check.ok ? '通过' : '待处理' }}
                </span>
              </div>
            </div>
          </UWidget>

          <UWidget title="登录与风险" subtitle="最近 24 小时" icon="user" :span="5">
            <div class="stat-grid" style="grid-template-columns: repeat(2, minmax(0, 1fr))">
              <div class="stat">
                <p class="stat__label">登录失败</p>
                <p class="stat__value">{{ stats.failures24h || 0 }}</p>
                <p class="stat__meta">失败会被限流并记录</p>
              </div>
              <div class="stat">
                <p class="stat__label">登录成功</p>
                <p class="stat__value">{{ stats.successes24h || 0 }}</p>
                <p class="stat__meta">最近登录见审计</p>
              </div>
            </div>
            <div v-if="overview && overview.warnings && overview.warnings.length" class="warning-list" style="margin-top: 12px">
              <div v-for="(warning, index) in overview.warnings" :key="index" class="warning-item">
                <UIcon name="alert" :size="15" />
                <span>{{ warning }}</span>
              </div>
            </div>
            <div v-else class="alert alert--success" style="margin-top: 12px">
              <UIcon name="check-circle" :size="16" class="alert__icon" />
              <div class="alert__body">未发现明显风险项</div>
            </div>
          </UWidget>

          <UWidget title="HTTPS 覆盖" subtitle="站点证书状态" icon="ssl" :span="4">
            <div class="stack">
              <div class="monitor__big">{{ httpsRatio }}%</div>
              <div class="meter">
                <div class="meter__track">
                  <div
                    class="meter__fill"
                    :class="httpsRatio === 100 ? 'meter__fill--success' : 'meter__fill--warning'"
                    :style="{ width: `${httpsRatio}%` }"
                  />
                </div>
                <p class="meter__meta">
                  {{ overview?.httpsSites || 0 }} / {{ overview?.totalSites || 0 }} 个站点已启用 SSL
                </p>
              </div>
              <UButton variant="secondary" size="sm" icon="ssl" label="管理证书" @click="go('/websites?tab=ssl')" />
            </div>
          </UWidget>

          <UWidget title="SSH 安全" subtitle="sshd 配置只读检查" icon="terminal" :span="4">
            <div class="stack">
              <div class="row">
                <UStatusDot
                  :status="overview?.ssh?.serviceActive ? 'running' : 'warning'"
                  :text="overview?.ssh?.serviceActive ? `运行中 · ${overview?.ssh?.port}` : '未检测到服务'"
                />
              </div>
              <dl class="info-list">
                <div class="info-list__row">
                  <dt class="info-list__label">密码登录</dt>
                  <dd class="info-list__value">{{ overview?.ssh?.passwordAuthentication || '—' }}</dd>
                </div>
                <div class="info-list__row">
                  <dt class="info-list__label">root 登录</dt>
                  <dd class="info-list__value">{{ overview?.ssh?.permitRootLogin || '—' }}</dd>
                </div>
              </dl>
              <UButton variant="secondary" size="sm" icon="terminal" label="查看详情" @click="go('/ssh')" />
            </div>
          </UWidget>

          <UWidget title="防火墙" subtitle="只读状态" icon="shield" :span="4">
            <div class="stack">
              <div class="row">
                <UStatusDot
                  :status="overview?.firewall?.active ? 'running' : 'warning'"
                  :text="overview?.firewall?.active ? `${overview?.firewall?.backend} 已启用` : '未检测到已启用防火墙'"
                />
              </div>
              <p class="muted" style="font-size: 12px">
                {{ overview?.firewall?.error || `规则数：${overview?.firewall?.rules?.length || 0}` }}
              </p>
              <UButton variant="secondary" size="sm" icon="shield" label="查看规则" @click="activeTab = 'firewall'" />
            </div>
          </UWidget>
        </div>
      </div>

      <div v-else-if="activeTab === 'audit'" class="stack">
        <div class="toolbar">
          <div class="toolbar__left">
            <span class="chip">失败 {{ stats.failures24h || 0 }}</span>
            <span class="chip">成功 {{ stats.successes24h || 0 }}</span>
          </div>
          <div class="toolbar__right">
            <UButton icon="refresh" label="刷新" :loading="loading" @click="loadAll" />
          </div>
        </div>
        <div v-if="audit.length === 0" class="table-wrap">
          <UEmpty icon="user" title="暂无登录记录" description="登录成功或失败后会记录 IP、时间和结果。" />
        </div>
        <div v-else class="table-wrap">
          <table class="table">
            <caption class="sr-only">登录审计记录</caption>
            <thead>
              <tr>
                <th scope="col">时间</th>
                <th scope="col">结果</th>
                <th scope="col">用户名</th>
                <th scope="col">来源 IP</th>
                <th scope="col">原因</th>
                <th scope="col">User-Agent</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(entry, index) in audit" :key="index">
                <td class="muted nowrap">{{ formatDate(entry.time) }}</td>
                <td>
                  <UStatusDot :status="entry.success ? 'running' : 'error'" :text="entry.success ? '成功' : '失败'" />
                </td>
                <td class="mono">{{ entry.username || '—' }}</td>
                <td class="mono">{{ entry.ip || '—' }}</td>
                <td class="muted">{{ entry.reason || '—' }}</td>
                <td class="muted truncate" style="max-width: 260px">{{ entry.userAgent || '—' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="alert alert--info">
          <UIcon name="info" :size="16" class="alert__icon" />
          <div class="alert__body">审计记录保存在 <code class="code--inline">data/logs/auth.log</code>，最多保留最近 500 条到内存。</div>
        </div>
      </div>

      <div v-else class="stack">
        <div class="widget-grid">
          <UWidget title="防火墙状态" subtitle="只读检查，不提供修改操作" icon="shield" :span="12">
            <div class="stack">
              <div class="row">
                <UStatusDot
                  :status="overview?.firewall?.active ? 'running' : 'warning'"
                  :text="overview?.firewall?.active ? '已启用' : '未启用或不可用'"
                />
                <span class="badge">{{ overview?.firewall?.backend || 'none' }}</span>
              </div>
              <div v-if="overview?.firewall?.error" class="alert alert--warning">
                <UIcon name="alert" :size="16" class="alert__icon" />
                <div class="alert__body">{{ overview.firewall.error }}</div>
              </div>
              <pre v-if="overview?.firewall?.summary" class="log log--light">{{ overview.firewall.summary }}</pre>
              <div v-else class="empty" style="padding: 22px">
                <p class="empty__title">没有可显示的防火墙信息</p>
                <p class="empty__desc">面板会尝试读取 ufw / firewalld / nftables 的状态。</p>
              </div>
              <div class="alert alert--info">
                <UIcon name="info" :size="16" class="alert__icon" />
                <div class="alert__body">
                  云服务器通常由安全组控制入站规则，本页只展示系统防火墙状态，修改请使用 SSH 或云控制台。
                </div>
              </div>
            </div>
          </UWidget>
        </div>
      </div>
    </UTabs>
    </div>

    <!-- 修改密码 -->
    <UModal v-model="passwordOpen" title="修改管理员密码" description="修改成功后所有已登录会话会立即失效" size="md" :busy="passwordSubmitting">
      <form id="password-form" class="stack" @submit.prevent="submitPassword">
        <div v-if="passwordError" class="alert alert--danger" role="alert">
          <UIcon name="alert" :size="16" class="alert__icon" />
          <div class="alert__body">{{ passwordError }}</div>
        </div>
        <UField label="当前密码" for-id="old-password" required>
          <UInput id="old-password" v-model="passwordForm.oldPassword" type="password" revealable autocomplete="current-password" />
        </UField>
        <UField label="新密码" for-id="new-password" required hint="至少 8 位，必须同时包含字母和数字">
          <UInput id="new-password" v-model="passwordForm.newPassword" type="password" revealable autocomplete="new-password" />
        </UField>
        <UField label="确认新密码" for-id="confirm-password" required>
          <UInput id="confirm-password" v-model="passwordForm.confirmPassword" type="password" revealable autocomplete="new-password" />
        </UField>
        <div class="alert alert--warning">
          <UIcon name="alert" :size="16" class="alert__icon" />
          <div class="alert__body">修改密码后需要重新登录。若忘记密码，可在服务器执行 <code class="code--inline">up 9 password 新密码</code>。</div>
        </div>
      </form>
      <template #footer>
        <UButton variant="secondary" @click="passwordOpen = false">取消</UButton>
        <UButton variant="primary" :loading="passwordSubmitting" icon="key" label="修改密码" @click="submitPassword" />
      </template>
    </UModal>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import UButton from '@/components/ui/UButton.vue'
import UEmpty from '@/components/ui/UEmpty.vue'
import UField from '@/components/ui/UField.vue'
import UIcon from '@/components/ui/UIcon.vue'
import UInput from '@/components/ui/UInput.vue'
import UModal from '@/components/ui/UModal.vue'
import UStatusDot from '@/components/ui/UStatusDot.vue'
import UTabs from '@/components/ui/UTabs.vue'
import UWidget from '@/components/ui/UWidget.vue'
import { authApi, errText, securityApi } from '@/api'
import { useAuth } from '@/composables/useAuth'
import { formatDate } from '@/composables/useFormat'
import { toastError, toastSuccess } from '@/composables/useToast'

const route = useRoute()
const router = useRouter()
const { clear } = useAuth()

const loading = ref(false)
const error = ref('')
const overview = ref(null)
const audit = ref([])
const stats = ref({})
const activeTab = ref(['overview', 'audit', 'firewall'].includes(route.query.tab) ? route.query.tab : 'overview')
const tabs = [
  { id: 'overview', label: '概览', icon: 'shield' },
  { id: 'audit', label: '登录审计', icon: 'user' },
  { id: 'firewall', label: '防火墙', icon: 'network' },
]

const httpsRatio = computed(() => {
  if (!overview.value || !overview.value.totalSites) return 0
  return Math.round((overview.value.httpsSites / overview.value.totalSites) * 100)
})

const checks = computed(() => {
  const o = overview.value || {}
  const ssh = o.ssh || {}
  return [
    {
      id: 'entry',
      title: '安全入口',
      detail: o.securityEntry ? `已启用：${o.securityEntry}` : '未启用，登录页对公网可见',
      ok: !!o.securityEntry,
    },
    {
      id: 'bind',
      title: '面板监听地址',
      detail: o.bindIP && o.bindIP !== '0.0.0.0' ? `仅监听 ${o.bindIP}` : '监听所有网卡，建议限制来源 IP',
      ok: o.bindIP && o.bindIP !== '0.0.0.0',
    },
    {
      id: 'https',
      title: 'HTTPS 覆盖',
      detail: !o.totalSites ? '尚无站点' : `${o.httpsSites || 0}/${o.totalSites} 个站点启用 SSL`,
      ok: !o.totalSites || o.httpsSites === o.totalSites,
    },
    {
      id: 'initial-password',
      title: '初始密码文件',
      detail: o.initialPasswordExists ? 'data/initial-password.txt 仍存在，请删除' : '未发现残留密码文件',
      ok: !o.initialPasswordExists,
    },
    {
      id: 'firewall',
      title: '系统防火墙',
      detail: o.firewall?.active ? `${o.firewall.backend} 已启用` : '未检测到启用的 ufw/firewalld',
      ok: !!o.firewall?.active,
    },
    {
      id: 'ssh',
      title: 'SSH 配置',
      detail: ssh.error || `${ssh.port || '22'} 端口 · 密码登录 ${ssh.passwordAuthentication || '—'}`,
      ok: !ssh.error && ssh.installed && (!ssh.warnings || ssh.warnings.length === 0),
    },
  ]
})

const riskCount = computed(() => checks.value.filter((check) => !check.ok).length)

async function loadAll() {
  loading.value = true
  error.value = ''
  try {
    const [overviewRes, auditRes] = await Promise.all([securityApi.overview(), securityApi.audit(100)])
    overview.value = overviewRes.data || null
    audit.value = auditRes.data || []
    stats.value = auditRes.stats || {}
  } catch (e) {
    error.value = errText(e, '读取安全信息失败')
  } finally {
    loading.value = false
  }
}

function go(path) {
  router.push(path)
}

// password
const passwordOpen = ref(false)
const passwordSubmitting = ref(false)
const passwordError = ref('')
const passwordForm = reactive({ oldPassword: '', newPassword: '', confirmPassword: '' })

function openPassword() {
  passwordForm.oldPassword = ''
  passwordForm.newPassword = ''
  passwordForm.confirmPassword = ''
  passwordError.value = ''
  passwordOpen.value = true
}

async function submitPassword() {
  if (passwordSubmitting.value) return
  passwordError.value = ''
  if (!passwordForm.oldPassword) {
    passwordError.value = '请输入当前密码'
    return
  }
  if (passwordForm.newPassword.length < 8) {
    passwordError.value = '新密码至少 8 位'
    return
  }
  if (!/[A-Za-z]/.test(passwordForm.newPassword) || !/[0-9]/.test(passwordForm.newPassword)) {
    passwordError.value = '新密码必须同时包含字母和数字'
    return
  }
  if (passwordForm.newPassword !== passwordForm.confirmPassword) {
    passwordError.value = '两次输入的新密码不一致'
    return
  }
  passwordSubmitting.value = true
  try {
    await authApi.changePassword(passwordForm.oldPassword, passwordForm.newPassword)
    toastSuccess('密码已修改', '请使用新密码重新登录')
    passwordOpen.value = false
    clear()
    router.replace('/login')
  } catch (e) {
    passwordError.value = errText(e, '修改密码失败')
  } finally {
    passwordSubmitting.value = false
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
    if (['overview', 'audit', 'firewall'].includes(tab) && tab !== activeTab.value) activeTab.value = tab
  }
)

onMounted(loadAll)
</script>
