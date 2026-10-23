<template>
  <div class="page">
    <header class="page__header">
      <div>
        <div class="page__title-row">
          <h1 class="page__title">应用商店</h1>
          <span class="badge badge--primary">{{ apps.length }}</span>
        </div>
        <p class="page__subtitle">一键安装常用服务，端口默认只绑定 127.0.0.1</p>
      </div>
      <div class="page__actions">
        <UButton icon="refresh" label="刷新" :loading="loading" @click="loadAll" />
        <UButton v-if="activeTab === 'installed'" variant="primary" icon="store" label="去商店" @click="activeTab = 'store'" />
      </div>
    </header>

    <div v-if="error" class="alert alert--danger" role="alert">
      <UIcon name="alert" :size="16" class="alert__icon" />
      <div class="alert__body">{{ error }}</div>
    </div>

    <UTabs v-model="activeTab" :tabs="tabs" aria-label="应用商店标签页">
      <div v-if="activeTab === 'store'" class="stack">
        <div class="toolbar">
          <div class="toolbar__left">
            <div class="input-affix" style="width: 260px">
              <UIcon name="search" :size="15" class="subtle" style="position: absolute; left: 10px" />
              <input v-model="search" class="input" placeholder="搜索应用名称或描述" style="padding-left: 32px" />
            </div>
            <div class="chip-row" role="group" aria-label="应用分类筛选">
              <button
                v-for="category in categories"
                :key="category.value"
                type="button"
                class="chip"
                :class="{ 'chip--active': categoryFilter === category.value }"
                :aria-pressed="categoryFilter === category.value ? 'true' : 'false'"
                @click="categoryFilter = category.value"
              >
                {{ category.label }}
              </button>
            </div>
          </div>
        </div>

        <div v-if="loading && apps.length === 0" class="loading-block"><span class="spinner" /> 正在加载应用目录…</div>

        <div v-else-if="filteredApps.length === 0" class="table-wrap">
          <UEmpty icon="store" title="没有匹配的应用" description="换个关键词或分类试试。" />
        </div>

        <div v-else class="app-grid">
          <article v-for="app in filteredApps" :key="app.key" class="app-card">
            <div class="app-card__head">
              <span class="app-card__icon"><UIcon :name="app.icon || 'box'" :size="18" /></span>
              <div class="grow">
                <div class="split">
                  <h2 class="app-card__name">{{ app.name }}</h2>
                  <span v-if="app.installed" class="badge badge--success">已安装</span>
                  <span v-else class="badge">未安装</span>
                </div>
                <p class="subtle" style="font-size: 11px">{{ categoryLabel(app.category) }} · 默认端口 {{ app.hostPort }}</p>
              </div>
            </div>
            <p class="app-card__desc">{{ app.description }}</p>
            <div class="app-card__meta">
              <span class="mono">{{ app.versions.slice(0, 3).join(' / ') }}</span>
              <UButton
                v-if="!app.installed"
                variant="primary"
                size="sm"
                icon="plus"
                label="安装"
                @click="openInstall(app)"
              />
              <UButton v-else variant="secondary" size="sm" icon="settings" label="管理" @click="activeTab = 'installed'" />
            </div>
          </article>
        </div>

        <div class="alert alert--info">
          <UIcon name="shield" :size="16" class="alert__icon" />
          <div class="alert__body">
            MySQL / PostgreSQL / Redis / MinIO 会自动生成随机密码；数据库与工具容器均只监听
            <code class="code--inline">127.0.0.1</code>，避免直接暴露到公网。
          </div>
        </div>
      </div>

      <div v-else class="stack">
        <div v-if="installed.length === 0" class="table-wrap">
          <UEmpty icon="layers" title="还没有安装应用" description="从应用商店选择服务安装，安装完成后会出现在这里。">
            <UButton variant="primary" icon="store" label="浏览应用商店" @click="activeTab = 'store'" />
          </UEmpty>
        </div>
        <div v-else class="table-wrap">
          <table class="table">
            <caption class="sr-only">已安装应用</caption>
            <thead>
              <tr>
                <th scope="col">实例</th>
                <th scope="col">应用</th>
                <th scope="col">版本</th>
                <th scope="col">状态</th>
                <th scope="col">本机端口</th>
                <th scope="col">安装时间</th>
                <th scope="col" style="text-align: right">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in installed" :key="item.name">
                <td>
                  <div class="stack" style="gap: 0">
                    <span class="table__primary">{{ item.name }}</span>
                    <span class="subtle mono" style="font-size: 11px">{{ item.container }}</span>
                  </div>
                </td>
                <td><span class="badge badge--primary">{{ item.appKey }}</span></td>
                <td class="mono">{{ item.version }}</td>
                <td><UStatusDot :status="item.status" :text="statusLabel(item.status)" /></td>
                <td class="mono">{{ item.hostPort }}</td>
                <td class="muted">{{ formatDateShort(item.installedAt) }}</td>
                <td>
                  <div class="table__actions">
                    <UButton variant="ghost" size="sm" icon="key" label="凭据" @click="openCredentials(item)" />
                    <UButton
                      v-if="item.status === 'running'"
                      variant="ghost"
                      icon-only
                      size="sm"
                      icon="stop"
                      aria-label="停止应用"
                      title="停止"
                      :loading="busyName === item.name"
                      @click="toggleApp(item, 'stop')"
                    />
                    <UButton
                      v-else
                      variant="ghost"
                      icon-only
                      size="sm"
                      icon="play"
                      aria-label="启动应用"
                      title="启动"
                      :loading="busyName === item.name"
                      @click="toggleApp(item, 'start')"
                    />
                    <UButton
                      variant="ghost"
                      icon-only
                      size="sm"
                      icon="restart"
                      aria-label="重启应用"
                      title="重启"
                      :loading="busyName === item.name"
                      @click="toggleApp(item, 'restart')"
                    />
                    <UButton
                      variant="ghost"
                      icon-only
                      size="sm"
                      icon="trash"
                      aria-label="卸载应用"
                      title="卸载"
                      @click="openUninstall(item)"
                    />
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </UTabs>

    <!-- 安装 -->
    <UModal
      v-model="installOpen"
      :title="`安装 ${installApp?.name || ''}`"
      description="安装过程会拉取镜像，首次安装可能需要几分钟"
      size="lg"
      :busy="installing"
      :close-on-backdrop="!installing"
      @close="closeInstall"
    >
      <form id="install-app-form" class="stack" @submit.prevent="submitInstall">
        <div v-if="formError" class="alert alert--danger" role="alert">
          <UIcon name="alert" :size="16" class="alert__icon" />
          <div class="alert__body">{{ formError }}</div>
        </div>
        <div v-if="installApp" class="row">
          <span class="app-card__icon"><UIcon :name="installApp.icon || 'box'" :size="18" /></span>
          <div>
            <p class="table__primary">{{ installApp.name }}</p>
            <p class="subtle" style="font-size: 12px">{{ installApp.description }}</p>
          </div>
        </div>
        <div class="form-grid">
          <UField label="实例名称" for-id="app-name" required hint="用于容器名与数据目录">
            <UInput id="app-name" v-model="installForm.name"
              autofocus autocomplete="off" placeholder="myapp" />
          </UField>
          <UField label="版本" required>
            <UDropdownSelect
              v-model="installForm.version"
              :options="versionOptions"
              :searchable="false"
              aria-label="应用版本"
            />
          </UField>
          <UField label="宿主机端口" for-id="app-port" hint="仅监听 127.0.0.1，端口冲突时自动顺延">
            <UInput id="app-port" v-model.number="installForm.hostPort" type="number" min="1" max="65535" />
          </UField>
          <UField v-if="showField('username')" label="用户名" for-id="app-username">
            <UInput id="app-username" v-model="installForm.username" autocomplete="off" />
          </UField>
          <UField v-if="showField('password')" label="密码" for-id="app-password" hint="留空自动生成随机密码">
            <UInput id="app-password" v-model="installForm.password" type="password" revealable autocomplete="new-password" />
          </UField>
          <UField v-if="showField('database')" label="数据库名" for-id="app-database">
            <UInput id="app-database" v-model="installForm.database" autocomplete="off" />
          </UField>
          <UField v-if="showField('db_host')" label="数据库地址" for-id="app-db-host" required hint="例如 172.17.0.1:3306 或容器名">
            <UInput id="app-db-host" v-model="installForm.db_host" placeholder="172.17.0.1:3306" />
          </UField>
          <UField v-if="showField('db_user')" label="数据库用户名" for-id="app-db-user">
            <UInput id="app-db-user" v-model="installForm.db_user" placeholder="wordpress" />
          </UField>
          <UField v-if="showField('db_name')" label="数据库名" for-id="app-db-name">
            <UInput id="app-db-name" v-model="installForm.db_name" placeholder="wordpress" />
          </UField>
          <UField v-if="showField('db_password')" label="数据库密码" for-id="app-db-password" required>
            <UInput id="app-db-password" v-model="installForm.db_password" type="password" revealable autocomplete="new-password" />
          </UField>
        </div>

        <div v-if="installApp?.key === 'portainer'" class="alert alert--warning">
          <UIcon name="alert" :size="16" class="alert__icon" />
          <div class="alert__body">
            Portainer 会挂载 <code class="code--inline">/var/run/docker.sock</code>（等同于宿主机 root 权限），并仅监听 127.0.0.1。请通过 SSH 隧道访问。
          </div>
        </div>
        <div v-if="installing" class="alert alert--info" role="status">
          <span class="spinner" />
          <div class="alert__body">正在拉取镜像并启动容器，请不要关闭页面…</div>
        </div>
      </form>
      <template #footer>
        <UButton variant="secondary" :disabled="installing" @click="closeInstall">取消</UButton>
        <UButton variant="primary" :loading="installing" icon="plus" label="开始安装" @click="submitInstall" />
      </template>
    </UModal>

    <!-- 凭据 -->
    <UModal v-model="credentialsOpen" :title="`${credentialsName} · 连接凭据`" size="md">
      <div v-if="credentialsLoading" class="loading-block"><span class="spinner" /> 正在读取…</div>
      <div v-else-if="credentialRows.length === 0" class="empty" style="padding: 20px">
        <p class="empty__title">没有可显示的凭据</p>
        <p class="empty__desc">该应用不需要账号密码，或凭据未保存。</p>
      </div>
      <div v-else class="credential-grid">
        <div v-for="row in credentialRows" :key="row.key" class="credential-row">
          <span class="credential-row__label">{{ row.label }}</span>
          <span class="credential-row__value">
            <span>{{ row.masked && !revealCredentials ? '••••••••••••' : row.value }}</span>
            <UButton
              v-if="row.masked"
              variant="ghost"
              size="sm"
              icon-only
              :icon="revealCredentials ? 'eye-off' : 'eye'"
              :aria-label="revealCredentials ? '隐藏密码' : '显示密码'"
              @click="revealCredentials = !revealCredentials"
            />
            <button class="copy-btn" type="button" :aria-label="`复制${row.label}`" @click="copy(row.value)">
              <UIcon name="copy" :size="13" />
            </button>
          </span>
        </div>
      </div>
      <template #footer>
        <UButton variant="secondary" @click="credentialsOpen = false">关闭</UButton>
      </template>
    </UModal>

    <!-- 卸载 -->
    <UModal v-model="uninstallOpen" title="卸载应用" size="sm" :busy="submitting">
      <div class="stack">
        <p>即将卸载应用 <strong>{{ uninstallItem?.name }}</strong> 并删除其容器。</p>
        <USwitch v-model="uninstallKeepData" label="保留数据目录" />
        <div class="alert" :class="uninstallKeepData ? 'alert--info' : 'alert--danger'">
          <UIcon name="alert" :size="16" class="alert__icon" />
          <div class="alert__body">
            {{ uninstallKeepData ? '数据目录会保留在 data/apps 下。' : '数据目录会被永久删除，无法恢复。' }}
          </div>
        </div>
      </div>
      <template #footer>
        <UButton variant="secondary" @click="uninstallOpen = false">取消</UButton>
        <UButton variant="danger" :loading="submitting" icon="trash" label="确认卸载" @click="submitUninstall" />
      </template>
    </UModal>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import UButton from '@/components/ui/UButton.vue'
import UDropdownSelect from '@/components/ui/UDropdownSelect.vue'
import UEmpty from '@/components/ui/UEmpty.vue'
import UField from '@/components/ui/UField.vue'
import UIcon from '@/components/ui/UIcon.vue'
import UInput from '@/components/ui/UInput.vue'
import UModal from '@/components/ui/UModal.vue'
import UStatusDot from '@/components/ui/UStatusDot.vue'
import USwitch from '@/components/ui/USwitch.vue'
import UTabs from '@/components/ui/UTabs.vue'
import { appApi, errText } from '@/api'
import { copyText, formatDateShort, statusLabel } from '@/composables/useFormat'
import { toastError, toastSuccess } from '@/composables/useToast'

const route = useRoute()
const router = useRouter()

const apps = ref([])
const installed = ref([])
const loading = ref(false)
const error = ref('')
const activeTab = ref(route.query.tab === 'installed' ? 'installed' : 'store')
const search = ref('')
const categoryFilter = ref('all')
const busyName = ref('')

const tabs = computed(() => [
  { id: 'store', label: '应用商店', count: apps.value.length },
  { id: 'installed', label: '已安装', count: installed.value.length },
])

const categories = [
  { value: 'all', label: '全部' },
  { value: 'web', label: 'Web' },
  { value: 'database', label: '数据库' },
  { value: 'tools', label: '工具' },
  { value: 'website', label: '建站' },
  { value: 'storage', label: '存储' },
]

const filteredApps = computed(() => {
  const q = search.value.trim().toLowerCase()
  return apps.value.filter((app) => {
    if (categoryFilter.value !== 'all' && app.category !== categoryFilter.value) return false
    if (!q) return true
    return [app.name, app.key, app.description].join(' ').toLowerCase().includes(q)
  })
})

function categoryLabel(category) {
  return { web: 'Web 服务', database: '数据库', tools: '管理工具', website: '建站', storage: '存储' }[category] || category
}

// install
const installOpen = ref(false)
const installApp = ref(null)
const installForm = reactive({
  name: '',
  version: '',
  hostPort: 80,
  username: '',
  password: '',
  database: '',
  db_host: '',
  db_user: '',
  db_name: '',
  db_password: '',
})
const installing = ref(false)
const submitting = ref(false)
const formError = ref('')

const versionOptions = computed(() =>
  (installApp.value?.versions || []).map((version) => ({ value: version, label: version }))
)

const FIELD_MAP = {
  mysql: ['password'],
  mariadb: ['password'],
  postgres: ['username', 'password', 'database'],
  redis: ['password'],
  minio: ['username', 'password'],
  wordpress: ['db_host', 'db_user', 'db_name', 'db_password'],
}

function showField(name) {
  const key = installApp.value?.key
  return (FIELD_MAP[key] || []).includes(name)
}

function openInstall(app) {
  installApp.value = app
  installForm.name = app.key
  installForm.version = app.defaultVersion
  installForm.hostPort = app.hostPort
  installForm.username = app.key === 'postgres' ? 'postgres' : app.key === 'minio' ? 'minioadmin' : ''
  installForm.password = ''
  installForm.database = app.key === 'postgres' ? 'postgres' : ''
  installForm.db_host = ''
  installForm.db_user = 'wordpress'
  installForm.db_name = 'wordpress'
  installForm.db_password = ''
  formError.value = ''
  installOpen.value = true
}

function closeInstall() {
  if (installing.value) return
  installOpen.value = false
}

async function submitInstall() {
  if (installing.value) return
  formError.value = ''
  if (!installForm.name.trim()) {
    formError.value = '请输入实例名称'
    return
  }
  const config = { host_port: Number(installForm.hostPort) || installApp.value.hostPort }
  for (const field of FIELD_MAP[installApp.value.key] || []) {
    const value = String(installForm[field] || '').trim()
    if (value) config[field] = value
  }
  if (installApp.value.key === 'wordpress' && !config.db_password) {
    formError.value = 'WordPress 需要填写数据库密码'
    return
  }
  installing.value = true
  try {
    const res = await appApi.install({
      app_key: installApp.value.key,
      version: installForm.version,
      name: installForm.name.trim(),
      config,
    })
    toastSuccess('应用安装成功', `${installApp.value.name} 已启动`)
    installOpen.value = false
    await loadAll()
    activeTab.value = 'installed'
    if (res.data && res.data.credentials && Object.keys(res.data.credentials).length > 0) {
      openCredentials({ name: res.data.name }, res.data.credentials)
    }
  } catch (e) {
    formError.value = errText(e, '安装失败')
  } finally {
    installing.value = false
  }
}

// credentials
const credentialsOpen = ref(false)
const credentialsLoading = ref(false)
const credentialsName = ref('')
const credentials = ref({})
const revealCredentials = ref(false)

const credentialRows = computed(() =>
  Object.entries(credentials.value || {}).map(([key, value]) => ({
    key,
    label: credentialLabel(key),
    value: String(value),
    masked: key.toLowerCase().includes('password') || key.toLowerCase().includes('secret'),
  }))
)

function credentialLabel(key) {
  return (
    {
      username: '用户名',
      password: '密码',
      db_password: '数据库密码',
      access_key: 'Access Key',
      secret_key: 'Secret Key',
    }[key] || key
  )
}

async function openCredentials(item, preset) {
  credentialsName.value = item.name
  credentials.value = preset || {}
  revealCredentials.value = false
  credentialsOpen.value = true
  if (preset) return
  credentialsLoading.value = true
  try {
    const res = await appApi.credentials(item.name)
    credentials.value = res.data || {}
  } catch (e) {
    toastError('读取凭据失败', errText(e))
  } finally {
    credentialsLoading.value = false
  }
}

// uninstall
const uninstallOpen = ref(false)
const uninstallItem = ref(null)
const uninstallKeepData = ref(true)

function openUninstall(item) {
  uninstallItem.value = item
  uninstallKeepData.value = true
  uninstallOpen.value = true
}

async function submitUninstall() {
  if (submitting.value || !uninstallItem.value) return
  submitting.value = true
  try {
    await appApi.remove(uninstallItem.value.name, uninstallKeepData.value)
    toastSuccess('应用已卸载', uninstallKeepData.value ? '数据目录已保留' : '数据目录已删除')
    uninstallOpen.value = false
    await loadAll()
  } catch (e) {
    toastError('卸载失败', errText(e))
  } finally {
    submitting.value = false
  }
}

async function toggleApp(item, action) {
  busyName.value = item.name
  try {
    if (action === 'start') await appApi.start(item.name)
    else if (action === 'stop') await appApi.stop(item.name)
    else await appApi.restart(item.name)
    toastSuccess('操作成功', item.name)
    await loadAll()
  } catch (e) {
    toastError('操作失败', errText(e))
  } finally {
    busyName.value = ''
  }
}

async function copy(text) {
  try {
    await copyText(text)
    toastSuccess('已复制')
  } catch (e) {
    toastError('复制失败', '请手动选择文本复制')
  }
}

async function loadAll() {
  loading.value = true
  error.value = ''
  try {
    const [storeRes, installedRes] = await Promise.allSettled([appApi.list(), appApi.installed()])
    if (storeRes.status === 'fulfilled') apps.value = storeRes.value.data || []
    if (installedRes.status === 'fulfilled') installed.value = installedRes.value.data || []
    if (storeRes.status === 'rejected' && installedRes.status === 'rejected') {
      error.value = errText(storeRes.reason, '加载应用商店失败')
    }
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
    if (tab === 'installed' && activeTab.value !== 'installed') activeTab.value = 'installed'
    if (!tab && activeTab.value !== 'store') activeTab.value = 'store'
  }
)

onMounted(loadAll)
</script>

<style scoped>
.chip-row {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.chip--active {
  border-color: var(--color-primary);
  background: var(--color-primary-soft);
  color: var(--color-primary);
}
</style>
