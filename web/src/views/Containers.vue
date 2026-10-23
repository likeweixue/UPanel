<template>
  <div class="page">
    <header class="page__header">
      <div>
        <div class="page__title-row">
          <h1 class="page__title">容器</h1>
          <span class="badge badge--primary">{{ containers.length }}</span>
        </div>
        <p class="page__subtitle">管理 Docker 容器、镜像与数据卷</p>
      </div>
      <div class="page__actions">
        <UButton icon="refresh" label="刷新" :loading="loading" @click="loadAll" />
      </div>
    </header>

    <div v-if="error" class="alert alert--danger" role="alert">
      <UIcon name="alert" :size="16" class="alert__icon" />
      <div class="alert__body">{{ error }}</div>
    </div>

    <UTabs v-model="activeTab" :tabs="tabs" aria-label="容器资源标签页">
      <div v-if="activeTab === 'containers'" class="stack">
        <div class="toolbar">
          <div class="toolbar__left">
            <div class="input-affix" style="width: 260px">
              <UIcon name="search" :size="15" class="subtle" style="position: absolute; left: 10px" />
              <input v-model="search" class="input" placeholder="搜索容器名称或镜像" style="padding-left: 32px" />
            </div>
          </div>
          <div class="toolbar__right">
            <span class="subtle" style="font-size: 12px">{{ runningCount }} 个运行中 / {{ containers.length }} 个容器</span>
          </div>
        </div>

        <div v-if="loading && containers.length === 0" class="loading-block"><span class="spinner" /> 正在读取 Docker 容器…</div>

        <div v-else-if="filteredContainers.length === 0" class="table-wrap">
          <UEmpty
            icon="box"
            title="没有容器"
            description="Docker 中还没有容器，或 Docker 服务未连接。请先在应用商店或网站中创建服务。"
          />
        </div>

        <div v-else class="table-wrap">
          <table class="table">
            <caption class="sr-only">Docker 容器列表</caption>
            <thead>
              <tr>
                <th scope="col">容器</th>
                <th scope="col">镜像</th>
                <th scope="col">状态</th>
                <th scope="col">端口</th>
                <th scope="col">来源</th>
                <th scope="col" style="text-align: right">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="container in filteredContainers" :key="container.id">
                <td>
                  <div class="stack" style="gap: 0">
                    <span class="table__primary">{{ container.name }}</span>
                    <span class="subtle mono" style="font-size: 11px">{{ container.id }}</span>
                  </div>
                </td>
                <td>
                  <code class="code--inline truncate" style="max-width: 220px">{{ container.image }}</code>
                </td>
                <td>
                  <UStatusDot :status="container.state" :text="statusLabel(container.state)" />
                  <div class="subtle" style="font-size: 11px">{{ container.status }}</div>
                </td>
                <td class="mono" style="font-size: 11px">
                  <span v-if="container.ports && container.ports.length">{{ container.ports.join(', ') }}</span>
                  <span v-else class="subtle">—</span>
                </td>
                <td>
                  <span :class="['badge', container.managed ? 'badge--primary' : '']">
                    {{ container.managed ? 'UPanel' : '外部' }}
                  </span>
                </td>
                <td>
                  <div class="table__actions">
                    <UButton
                      variant="ghost"
                      icon-only
                      size="sm"
                      icon="logs"
                      aria-label="查看容器日志"
                      title="日志"
                      @click="openLogs(container)"
                    />
                    <UButton
                      v-if="container.state === 'running'"
                      variant="ghost"
                      icon-only
                      size="sm"
                      icon="stop"
                      aria-label="停止容器"
                      title="停止"
                      :loading="busyId === container.id"
                      @click="act(container, 'stop')"
                    />
                    <UButton
                      v-else
                      variant="ghost"
                      icon-only
                      size="sm"
                      icon="play"
                      aria-label="启动容器"
                      title="启动"
                      :loading="busyId === container.id"
                      @click="act(container, 'start')"
                    />
                    <UButton
                      variant="ghost"
                      icon-only
                      size="sm"
                      icon="restart"
                      aria-label="重启容器"
                      title="重启"
                      :loading="busyId === container.id"
                      @click="act(container, 'restart')"
                    />
                    <UButton
                      variant="ghost"
                      icon-only
                      size="sm"
                      icon="trash"
                      aria-label="删除容器"
                      title="删除"
                      @click="confirmRemove(container)"
                    />
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="alert alert--info">
          <UIcon name="info" :size="16" class="alert__icon" />
          <div class="alert__body">
            UPanel 创建的容器带有 <code class="code--inline">managed-by=upanel</code> 标签。删除容器时不会删除宿主机上的网站文件或数据目录。
          </div>
        </div>
      </div>

      <div v-else-if="activeTab === 'images'" class="stack">
        <div v-if="images.length === 0" class="table-wrap">
          <UEmpty icon="image" title="没有本地镜像" description="安装应用或创建 PHP / 反代站点后，镜像会出现在这里。" />
        </div>
        <div v-else class="table-wrap">
          <table class="table">
            <caption class="sr-only">Docker 镜像列表</caption>
            <thead>
              <tr>
                <th scope="col">标签</th>
                <th scope="col">镜像 ID</th>
                <th scope="col">大小</th>
                <th scope="col">创建时间</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="image in images" :key="image.id">
                <td>
                  <div class="row" style="gap: 4px">
                    <span v-for="tag in image.tags" :key="tag" class="chip mono">{{ tag }}</span>
                  </div>
                </td>
                <td class="mono">{{ image.id }}</td>
                <td>{{ formatBytes(image.size) }}</td>
                <td class="muted">{{ formatDate(image.created * 1000) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div v-else class="stack">
        <div v-if="volumes.length === 0" class="table-wrap">
          <UEmpty icon="layers" title="没有数据卷" description="使用命名卷的容器会在这里显示。" />
        </div>
        <div v-else class="table-wrap">
          <table class="table">
            <caption class="sr-only">Docker 数据卷列表</caption>
            <thead>
              <tr>
                <th scope="col">名称</th>
                <th scope="col">驱动</th>
                <th scope="col">挂载路径</th>
                <th scope="col">创建时间</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="volume in volumes" :key="volume.name">
                <td class="table__primary">{{ volume.name }}</td>
                <td>{{ volume.driver }}</td>
                <td class="mono" style="font-size: 11px">{{ volume.mountpoint }}</td>
                <td class="muted">{{ volume.createdAt || '—' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </UTabs>

    <!-- 日志 -->
    <UModal v-model="logsOpen" :title="`${logsContainer?.name || ''} · 容器日志`" size="xl">
      <div v-if="logsLoading" class="loading-block"><span class="spinner" /> 正在读取日志…</div>
      <pre v-else-if="logsContent" class="log">{{ logsContent }}</pre>
      <UEmpty v-else icon="logs" title="暂无日志" description="容器可能尚未产生输出。" />
      <template #footer>
        <UButton variant="secondary" @click="logsOpen = false">关闭</UButton>
        <UButton variant="primary" icon="refresh" label="刷新" :loading="logsLoading" @click="loadLogs" />
      </template>
    </UModal>

    <!-- 删除确认 -->
    <UModal v-model="removeOpen" title="删除容器" size="sm">
      <div class="stack">
        <p>即将删除容器 <strong>{{ removeTarget?.name }}</strong>。</p>
        <div class="alert alert--warning">
          <UIcon name="alert" :size="16" class="alert__icon" />
          <div class="alert__body">容器会被强制删除，但宿主机上的数据目录不会删除。此操作不可撤销。</div>
        </div>
      </div>
      <template #footer>
        <UButton variant="secondary" @click="removeOpen = false">取消</UButton>
        <UButton variant="danger" :loading="removing" icon="trash" label="删除容器" @click="removeContainer" />
      </template>
    </UModal>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import UButton from '@/components/ui/UButton.vue'
import UEmpty from '@/components/ui/UEmpty.vue'
import UIcon from '@/components/ui/UIcon.vue'
import UModal from '@/components/ui/UModal.vue'
import UStatusDot from '@/components/ui/UStatusDot.vue'
import UTabs from '@/components/ui/UTabs.vue'
import { containerApi, errText } from '@/api'
import { formatBytes, formatDate, statusLabel } from '@/composables/useFormat'
import { toastError, toastSuccess } from '@/composables/useToast'

const route = useRoute()
const router = useRouter()

const containers = ref([])
const images = ref([])
const volumes = ref([])
const loading = ref(false)
const error = ref('')
const search = ref('')
const busyId = ref('')
const activeTab = ref(['containers', 'images', 'volumes'].includes(route.query.tab) ? route.query.tab : 'containers')

const tabs = computed(() => [
  { id: 'containers', label: '容器', count: containers.value.length },
  { id: 'images', label: '镜像', count: images.value.length },
  { id: 'volumes', label: '数据卷', count: volumes.value.length },
])

const runningCount = computed(() => containers.value.filter((c) => c.state === 'running').length)
const filteredContainers = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return containers.value
  return containers.value.filter((c) => `${c.name} ${c.image} ${c.id}`.toLowerCase().includes(q))
})

async function loadAll() {
  loading.value = true
  error.value = ''
  const results = await Promise.allSettled([
    containerApi.list(),
    containerApi.images(),
    containerApi.volumes(),
  ])
  if (results[0].status === 'fulfilled') containers.value = results[0].value.data || []
  if (results[1].status === 'fulfilled') images.value = results[1].value.data || []
  if (results[2].status === 'fulfilled') volumes.value = results[2].value.data || []
  const failures = results.filter((r) => r.status === 'rejected')
  if (failures.length === 3) {
    error.value = errText(failures[0].reason, 'Docker 未连接')
  } else if (failures.length > 0) {
    error.value = errText(failures[0].reason, '部分资源加载失败')
  }
  loading.value = false
}

async function act(container, action) {
  busyId.value = container.id
  try {
    if (action === 'start') await containerApi.start(container.id)
    else if (action === 'stop') await containerApi.stop(container.id)
    else await containerApi.restart(container.id)
    toastSuccess('操作成功', container.name)
    await loadAll()
  } catch (e) {
    toastError('操作失败', errText(e))
  } finally {
    busyId.value = ''
  }
}

// logs
const logsOpen = ref(false)
const logsLoading = ref(false)
const logsContainer = ref(null)
const logsContent = ref('')

async function openLogs(container) {
  logsContainer.value = container
  logsContent.value = ''
  logsOpen.value = true
  await loadLogs()
}

async function loadLogs() {
  if (!logsContainer.value) return
  logsLoading.value = true
  try {
    const res = await containerApi.logs(logsContainer.value.id, 300)
    logsContent.value = res.logs || ''
  } catch (e) {
    toastError('读取日志失败', errText(e))
  } finally {
    logsLoading.value = false
  }
}

// remove
const removeOpen = ref(false)
const removeTarget = ref(null)
const removing = ref(false)

function confirmRemove(container) {
  removeTarget.value = container
  removeOpen.value = true
}

async function removeContainer() {
  if (removing.value || !removeTarget.value) return
  removing.value = true
  try {
    await containerApi.remove(removeTarget.value.id)
    toastSuccess('容器已删除', removeTarget.value.name)
    removeOpen.value = false
    await loadAll()
  } catch (e) {
    toastError('删除容器失败', errText(e))
  } finally {
    removing.value = false
  }
}

watch(activeTab, (tab) => {
  if (route.query.tab !== tab) router.replace({ path: route.path, query: { ...route.query, tab } })
})

watch(
  () => route.query.tab,
  (tab) => {
    if (!tab) {
      activeTab.value = 'containers'
      return
    }
    if (['containers', 'images', 'volumes'].includes(tab) && tab !== activeTab.value) activeTab.value = tab
  }
)

onMounted(loadAll)
</script>
