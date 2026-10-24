<template>
  <div class="page">
    <div class="page-shell">
    <header class="page__header">
      <div>
        <div class="page__title-row">
          <h1 class="page__title">数据库</h1>
          <span class="badge badge--primary">{{ databases.length }}</span>
        </div>
        <p class="page__subtitle">MySQL / PostgreSQL / Redis 一键创建，默认仅监听 127.0.0.1</p>
      </div>
      <div class="page__actions">
        <UButton icon="refresh" label="刷新" :loading="loading" @click="loadDatabases" />
        <UButton variant="primary" icon="plus" label="创建数据库" @click="openCreate" />
      </div>
    </header>

    <div v-if="error" class="alert alert--danger" role="alert">
      <UIcon name="alert" :size="16" class="alert__icon" />
      <div class="alert__body">{{ error }}</div>
    </div>

    <UTabs v-model="activeTab" :tabs="tabs" aria-label="数据库管理标签页">
      <div v-if="activeTab === 'instances'" class="stack">
        <div class="toolbar">
          <div class="toolbar__left">
            <div class="chip-row" role="group" aria-label="按数据库类型筛选">
              <button
                v-for="option in typeFilters"
                :key="option.value"
                type="button"
                class="chip"
                :class="{ 'chip--active': filterType === option.value }"
                :aria-pressed="filterType === option.value ? 'true' : 'false'"
                @click="setTypeFilter(option.value)"
              >
                {{ option.label }}
                <span class="chip__count">{{ countByType(option.value) }}</span>
              </button>
            </div>
          </div>
          <div class="toolbar__right">
            <span class="subtle" style="font-size: 12px">{{ filteredDatabases.length }} 个实例</span>
          </div>
        </div>

        <div v-if="loading && databases.length === 0" class="loading-block"><span class="spinner" /> 正在加载数据库…</div>

        <div v-else-if="filteredDatabases.length === 0" class="table-wrap">
          <UEmpty
            icon="database"
            title="还没有数据库"
            description="创建 MySQL / PostgreSQL / Redis 实例，面板会自动分配本机端口并保存凭据。"
          >
            <UButton variant="primary" icon="plus" label="创建数据库" @click="openCreate" />
          </UEmpty>
        </div>

        <div v-else class="table-wrap">
          <table class="table">
            <caption class="sr-only">数据库实例列表</caption>
            <thead>
              <tr>
                <th scope="col">名称</th>
                <th scope="col">类型</th>
                <th scope="col">版本</th>
                <th scope="col">状态</th>
                <th scope="col">本机端口</th>
                <th scope="col">数据库</th>
                <th scope="col" style="text-align: right">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="db in filteredDatabases" :key="db.name">
                <td>
                  <div class="stack" style="gap: 0">
                    <span class="table__primary">{{ db.name }}</span>
                    <span class="subtle mono" style="font-size: 11px">{{ db.container }}</span>
                  </div>
                </td>
                <td><span class="badge" :class="dbBadge(db.type)">{{ typeLabel(db.type) }}</span></td>
                <td class="muted">{{ db.version || '—' }}</td>
                <td><UStatusDot :status="db.status" :text="statusLabel(db.status)" /></td>
                <td class="mono">{{ db.port }}</td>
                <td class="mono">{{ db.database }}</td>
                <td>
                  <div class="table__actions">
                    <UButton
                      variant="ghost"
                      size="sm"
                      icon="key"
                      label="连接"
                      @click="openConnection(db)"
                    />
                    <UButton
                      variant="ghost"
                      icon-only
                      size="sm"
                      icon="logs"
                      aria-label="查看数据库日志"
                      title="日志"
                      @click="openLogs(db)"
                    />
                    <UButton
                      v-if="db.status === 'running'"
                      variant="ghost"
                      icon-only
                      size="sm"
                      icon="stop"
                      aria-label="停止数据库"
                      title="停止"
                      :loading="busyName === db.name"
                      @click="toggleDb(db, 'stop')"
                    />
                    <UButton
                      v-else
                      variant="ghost"
                      icon-only
                      size="sm"
                      icon="play"
                      aria-label="启动数据库"
                      title="启动"
                      :loading="busyName === db.name"
                      @click="toggleDb(db, 'start')"
                    />
                    <UButton
                      variant="ghost"
                      icon-only
                      size="sm"
                      icon="restart"
                      aria-label="重启数据库"
                      title="重启"
                      :loading="busyName === db.name"
                      @click="toggleDb(db, 'restart')"
                    />
                    <UButton
                      variant="ghost"
                      icon-only
                      size="sm"
                      icon="trash"
                      aria-label="删除数据库"
                      title="删除"
                      @click="openDelete(db)"
                    />
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="alert alert--info">
          <UIcon name="shield" :size="16" class="alert__icon" />
          <div class="alert__body">
            数据库端口只绑定 <code class="code--inline">127.0.0.1</code>，公网无法直连。远程访问请使用 SSH 隧道，或在安全组中谨慎开放。
          </div>
        </div>
      </div>

      <div v-else class="stack">
        <div v-if="databases.length === 0" class="table-wrap">
          <UEmpty icon="key" title="暂无连接信息" description="创建数据库后，这里会展示本机地址、账号、密码与 DSN。" />
        </div>
        <div v-else class="table-wrap">
          <table class="table">
            <caption class="sr-only">数据库连接信息</caption>
            <thead>
              <tr>
                <th scope="col">实例</th>
                <th scope="col">类型</th>
                <th scope="col">主机:端口</th>
                <th scope="col">用户名</th>
                <th scope="col">数据库</th>
                <th scope="col" style="text-align: right">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="db in databases" :key="db.name">
                <td class="table__primary">{{ db.name }}</td>
                <td><span class="badge" :class="dbBadge(db.type)">{{ typeLabel(db.type) }}</span></td>
                <td class="mono">127.0.0.1:{{ db.port }}</td>
                <td class="mono">{{ db.username || '—' }}</td>
                <td class="mono">{{ db.database }}</td>
                <td>
                  <div class="table__actions">
                    <UButton variant="secondary" size="sm" icon="key" label="查看凭据" @click="openConnection(db)" />
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </UTabs>
    </div>

    <!-- 创建 -->
    <UModal v-model="createOpen" title="创建数据库" description="容器创建完成后会自动生成随机凭据（可自定义）" size="lg" :busy="submitting" @close="clearCreateQuery">
      <form id="create-db-form" class="stack" @submit.prevent="submitCreate">
        <div v-if="formError" class="alert alert--danger" role="alert">
          <UIcon name="alert" :size="16" class="alert__icon" />
          <div class="alert__body">{{ formError }}</div>
        </div>
        <div class="form-grid">
          <UField label="实例名称" for-id="db-name" required hint="用于容器名与数据目录，字母开头，可含 - _">
            <UInput id="db-name" v-model="createForm.name"
              autofocus placeholder="myapp-db" autocomplete="off" />
          </UField>
          <UField label="数据库类型" required>
            <UDropdownSelect
              v-model="createForm.type"
              :options="typeOptions"
              :searchable="false"
              aria-label="数据库类型"
              @change="onTypeChange"
            />
          </UField>
          <UField label="版本">
            <UDropdownSelect v-model="createForm.version" :options="versionOptions" :searchable="false" aria-label="数据库版本" />
          </UField>
          <UField label="数据库名" for-id="db-database" hint="留空则与实例名称相同">
            <UInput id="db-database" v-model="createForm.database" placeholder="myapp" />
          </UField>
          <UField label="密码" for-id="db-password" hint="留空自动生成 20 位随机密码（推荐）">
            <UInput id="db-password" v-model="createForm.password" type="password" revealable autocomplete="new-password" placeholder="至少 8 位，包含字母和数字" />
          </UField>
          <UField label="宿主机端口" for-id="db-port" hint="留空自动分配；仅监听 127.0.0.1">
            <UInput id="db-port" v-model.number="createForm.port" type="number" min="1" max="65535" placeholder="自动" />
          </UField>
        </div>
        <div class="alert alert--info">
          <UIcon name="info" :size="16" class="alert__icon" />
          <div class="alert__body">创建过程需要拉取镜像，首次安装可能需要几分钟。</div>
        </div>
      </form>
      <template #footer>
        <UButton variant="secondary" @click="createOpen = false">取消</UButton>
        <UButton variant="primary" :loading="submitting" icon="plus" label="创建并启动" @click="submitCreate" />
      </template>
    </UModal>

    <!-- 连接信息 -->
    <UModal v-model="connectionOpen" :title="`${connectionDb?.name || ''} · 连接信息`" size="lg">
      <div v-if="connectionLoading" class="loading-block"><span class="spinner" /> 正在读取凭据…</div>
      <div v-else class="stack">
        <div class="credential-grid">
          <div v-for="row in connectionRows" :key="row.key" class="credential-row">
            <span class="credential-row__label">{{ row.label }}</span>
            <span class="credential-row__value">
              <span>{{ row.masked && !revealPassword ? '••••••••••••' : row.value }}</span>
              <UButton
                v-if="row.masked"
                variant="ghost"
                size="sm"
                icon-only
                :icon="revealPassword ? 'eye-off' : 'eye'"
                :aria-label="revealPassword ? '隐藏密码' : '显示密码'"
                @click="revealPassword = !revealPassword"
              />
              <button class="copy-btn" type="button" :aria-label="`复制${row.label}`" @click="copy(row.value)">
                <UIcon name="copy" :size="13" />
              </button>
            </span>
          </div>
        </div>
        <div v-if="connection?.dsn" class="field">
          <span class="field__label">DSN</span>
          <div class="row" style="align-items: flex-start">
            <code class="code--inline grow" style="white-space: normal">{{ connection.dsn }}</code>
            <UButton variant="secondary" size="sm" icon="copy" label="复制" @click="copy(connection.dsn)" />
          </div>
        </div>
        <div v-if="connection?.hint" class="alert alert--info">
          <UIcon name="terminal" :size="16" class="alert__icon" />
          <div class="alert__body">{{ connection.hint }}</div>
        </div>
        <div v-if="connectionDb" class="muted" style="font-size: 12px">
          实例状态：{{ statusLabel(connectionDb.status) }} · 容器：{{ connectionDb.container }}
        </div>
      </div>
      <template #footer>
        <UButton variant="secondary" @click="connectionOpen = false">关闭</UButton>
        <UButton variant="primary" icon="refresh" label="刷新凭据" :loading="connectionLoading" @click="loadConnection" />
      </template>
    </UModal>

    <!-- 日志 -->
    <UModal v-model="logsOpen" :title="`${logsDb?.name || ''} · 容器日志`" size="xl">
      <div v-if="logsLoading" class="loading-block"><span class="spinner" /> 正在读取日志…</div>
      <pre v-else-if="logsContent" class="log">{{ logsContent }}</pre>
      <UEmpty v-else icon="logs" title="暂无日志" description="容器尚未产生输出。" />
      <template #footer>
        <UButton variant="secondary" @click="logsOpen = false">关闭</UButton>
        <UButton variant="primary" icon="refresh" label="刷新" :loading="logsLoading" @click="loadLogs" />
      </template>
    </UModal>

    <!-- 删除 -->
    <UModal v-model="deleteOpen" title="删除数据库" size="sm" :busy="submitting">
      <div class="stack">
        <p>即将删除数据库 <strong>{{ deleteDb?.name }}</strong> 的容器与元数据。</p>
        <USwitch v-model="deleteKeepData" label="保留数据目录" />
        <div class="alert" :class="deleteKeepData ? 'alert--info' : 'alert--danger'">
          <UIcon name="alert" :size="16" class="alert__icon" />
          <div class="alert__body">
            {{ deleteKeepData ? '数据目录会保留在 data/databases 下。' : '数据文件会被永久删除，无法恢复。' }}
          </div>
        </div>
      </div>
      <template #footer>
        <UButton variant="secondary" @click="deleteOpen = false">取消</UButton>
        <UButton variant="danger" :loading="submitting" icon="trash" label="确认删除" @click="submitDelete" />
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
import { dbApi, errText } from '@/api'
import { copyText, statusLabel } from '@/composables/useFormat'
import { toastError, toastSuccess } from '@/composables/useToast'

const route = useRoute()
const router = useRouter()

const databases = ref([])
const loading = ref(false)
const error = ref('')
const busyName = ref('')
const filterType = ref(['mysql', 'postgres', 'redis'].includes(route.query.type) ? route.query.type : 'all')
const activeTab = ref(route.query.tab === 'connections' ? 'connections' : 'instances')

const tabs = computed(() => [
  { id: 'instances', label: '数据库实例', count: databases.value.length },
  { id: 'connections', label: '连接信息', icon: 'key' },
])

const typeOptions = [
  { value: 'mysql', label: 'MySQL', description: '关系型数据库，兼容 MariaDB' },
  { value: 'postgres', label: 'PostgreSQL', description: '功能强大的开源关系型数据库' },
  { value: 'redis', label: 'Redis', description: '内存键值数据库，自动启用 requirepass' },
]
const versionsByType = {
  mysql: ['8.0', '8.4', '5.7'],
  postgres: ['16', '15', '14'],
  redis: ['7', '6'],
}
const typeFilters = [
  { value: 'all', label: '全部' },
  { value: 'mysql', label: 'MySQL' },
  { value: 'postgres', label: 'PostgreSQL' },
  { value: 'redis', label: 'Redis' },
]

const filteredDatabases = computed(() =>
  filterType.value === 'all'
    ? databases.value
    : databases.value.filter((db) => db.type === filterType.value)
)

function countByType(type) {
  return type === 'all' ? databases.value.length : databases.value.filter((db) => db.type === type).length
}

function typeLabel(type) {
  return { mysql: 'MySQL', postgres: 'PostgreSQL', redis: 'Redis' }[type] || type
}
function dbBadge(type) {
  return type === 'mysql' ? 'badge--info' : type === 'postgres' ? 'badge--primary' : 'badge--danger'
}

function setTypeFilter(type) {
  filterType.value = type
  const query = { ...route.query }
  if (type === 'all') delete query.type
  else query.type = type
  router.replace({ path: route.path, query })
}

// create
const createOpen = ref(false)
const createForm = reactive({ name: '', type: 'mysql', version: '8.0', database: '', password: '', port: '' })
const formError = ref('')
const submitting = ref(false)

const versionOptions = computed(() =>
  (versionsByType[createForm.type] || []).map((v) => ({ value: v, label: v }))
)

function onTypeChange(type) {
  createForm.version = (versionsByType[type] || [])[0] || ''
}

function openCreate() {
  createForm.name = ''
  createForm.type = 'mysql'
  createForm.version = '8.0'
  createForm.database = ''
  createForm.password = ''
  createForm.port = ''
  formError.value = ''
  createOpen.value = true
  if (route.query.action !== 'create') {
    router.replace({ path: route.path, query: { ...route.query, action: 'create' } })
  }
}

function clearCreateQuery() {
  if (route.query.action === 'create') {
    const query = { ...route.query }
    delete query.action
    router.replace({ path: route.path, query })
  }
}

async function submitCreate() {
  if (submitting.value) return
  formError.value = ''
  if (!createForm.name.trim()) {
    formError.value = '请输入实例名称'
    return
  }
  const payload = {
    name: createForm.name.trim(),
    type: createForm.type,
    version: createForm.version,
  }
  if (createForm.database.trim()) payload.database = createForm.database.trim()
  if (createForm.password.trim()) payload.password = createForm.password.trim()
  if (createForm.port) payload.port = Number(createForm.port)
  submitting.value = true
  try {
    await dbApi.create(payload)
    toastSuccess('数据库创建成功', '正在打开连接凭据')
    createOpen.value = false
    clearCreateQuery()
    await loadDatabases()
    const created = databases.value.find((db) => db.name === payload.name)
    if (created) openConnection(created)
  } catch (e) {
    formError.value = errText(e, '创建数据库失败')
  } finally {
    submitting.value = false
  }
}

// connection
const connectionOpen = ref(false)
const connectionLoading = ref(false)
const connectionDb = ref(null)
const connection = ref(null)
const revealPassword = ref(false)

const connectionRows = computed(() => {
  if (!connection.value) return []
  const c = connection.value
  const rows = [
    { key: 'host', label: '主机', value: c.host || '127.0.0.1' },
    { key: 'port', label: '端口', value: String(c.port || '') },
    { key: 'database', label: '数据库', value: c.database || '' },
  ]
  if (c.username) rows.push({ key: 'username', label: '用户名', value: c.username })
  if (c.password) rows.push({ key: 'password', label: '密码', value: c.password, masked: true })
  return rows
})

async function openConnection(db) {
  connectionDb.value = db
  connection.value = null
  revealPassword.value = false
  connectionOpen.value = true
  await loadConnection()
}

async function loadConnection() {
  if (!connectionDb.value) return
  connectionLoading.value = true
  try {
    const res = await dbApi.connection(connectionDb.value.name)
    connection.value = res.data || null
  } catch (e) {
    toastError('读取连接信息失败', errText(e))
  } finally {
    connectionLoading.value = false
  }
}

// logs
const logsOpen = ref(false)
const logsLoading = ref(false)
const logsDb = ref(null)
const logsContent = ref('')

async function openLogs(db) {
  logsDb.value = db
  logsContent.value = ''
  logsOpen.value = true
  await loadLogs()
}

async function loadLogs() {
  if (!logsDb.value) return
  logsLoading.value = true
  try {
    const res = await dbApi.logs(logsDb.value.name, 300)
    logsContent.value = res.data || ''
  } catch (e) {
    toastError('读取日志失败', errText(e))
  } finally {
    logsLoading.value = false
  }
}

// delete
const deleteOpen = ref(false)
const deleteDb = ref(null)
const deleteKeepData = ref(true)

function openDelete(db) {
  deleteDb.value = db
  deleteKeepData.value = true
  deleteOpen.value = true
}

async function submitDelete() {
  if (submitting.value || !deleteDb.value) return
  submitting.value = true
  try {
    await dbApi.remove(deleteDb.value.name, deleteKeepData.value)
    toastSuccess('数据库已删除', deleteKeepData.value ? '数据目录已保留' : '数据已删除')
    deleteOpen.value = false
    await loadDatabases()
  } catch (e) {
    toastError('删除数据库失败', errText(e))
  } finally {
    submitting.value = false
  }
}

async function toggleDb(db, action) {
  busyName.value = db.name
  try {
    if (action === 'start') await dbApi.start(db.name)
    else if (action === 'stop') await dbApi.stop(db.name)
    else await dbApi.restart(db.name)
    toastSuccess('操作成功', db.name)
    await loadDatabases()
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

async function loadDatabases() {
  loading.value = true
  error.value = ''
  try {
    const res = await dbApi.list()
    databases.value = res.data || []
  } catch (e) {
    error.value = errText(e, '加载数据库列表失败')
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
    if (tab === 'connections' && activeTab.value !== 'connections') activeTab.value = 'connections'
    if (!tab && activeTab.value !== 'instances') activeTab.value = 'instances'
  }
)

watch(
  () => route.query.action,
  (action) => {
    if (action === 'create' && !createOpen.value) openCreate()
  }
)

onMounted(async () => {
  await loadDatabases()
  if (route.query.action === 'create') openCreate()
})
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

.chip__count {
  padding: 0 5px;
  border-radius: var(--radius-pill);
  background: var(--color-surface-3);
  font-size: 10px;
  line-height: 16px;
}

.chip--active .chip__count {
  background: var(--color-surface);
}
</style>
