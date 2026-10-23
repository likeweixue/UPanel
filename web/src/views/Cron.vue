<template>
  <div class="page">
    <header class="page__header">
      <div>
        <div class="page__title-row">
          <h1 class="page__title">计划任务</h1>
          <span class="badge badge--primary">{{ jobs.length }}</span>
        </div>
        <p class="page__subtitle">按 cron 表达式定时执行 shell 命令（以面板进程权限运行）</p>
      </div>
      <div class="page__actions">
        <UButton icon="refresh" label="刷新" :loading="loading" @click="loadJobs" />
        <UButton variant="primary" icon="plus" label="新建任务" @click="openCreate" />
      </div>
    </header>

    <div v-if="error" class="alert alert--danger" role="alert">
      <UIcon name="alert" :size="16" class="alert__icon" />
      <div class="alert__body">{{ error }}</div>
    </div>

    <div class="alert alert--warning">
      <UIcon name="alert" :size="16" class="alert__icon" />
      <div class="alert__body">
        计划任务可以执行任意 shell 命令。面板通常以 root 运行，请只添加你完全信任的命令，并避免把敏感信息写进命令行。
      </div>
    </div>

    <div v-if="loading && jobs.length === 0" class="loading-block"><span class="spinner" /> 正在加载计划任务…</div>

    <div v-else-if="jobs.length === 0" class="table-wrap">
      <UEmpty icon="clock" title="还没有计划任务" description="可以定时清理日志、备份数据库或执行自定义脚本。">
        <UButton variant="primary" icon="plus" label="新建任务" @click="openCreate" />
      </UEmpty>
    </div>

    <div v-else class="table-wrap">
      <table class="table">
        <caption class="sr-only">计划任务列表</caption>
        <thead>
          <tr>
            <th scope="col">任务</th>
            <th scope="col">表达式</th>
            <th scope="col">命令</th>
            <th scope="col">启用</th>
            <th scope="col">上次执行</th>
            <th scope="col" style="text-align: right">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="job in jobs" :key="job.id">
            <td>
              <div class="stack" style="gap: 0">
                <span class="table__primary">{{ job.name }}</span>
                <span class="subtle mono" style="font-size: 11px">{{ job.id }}</span>
              </div>
            </td>
            <td>
              <code class="code--inline">{{ job.schedule }}</code>
              <div class="subtle" style="font-size: 11px">{{ scheduleLabel(job.schedule) }}</div>
            </td>
            <td>
              <code class="code--inline truncate" style="max-width: 280px">{{ job.command }}</code>
            </td>
            <td>
              <USwitch
                :model-value="job.enabled"
                :aria-label="`${job.enabled ? '禁用' : '启用'}任务 ${job.name}`"
                @update:model-value="(value) => toggleEnabled(job, value)"
              />
            </td>
            <td>
              <div class="stack" style="gap: 2px">
                <UStatusDot :status="job.lastStatus || 'unknown'" :text="runStatusText(job.lastStatus)" />
                <span class="subtle" style="font-size: 11px">{{ job.lastRun ? relativeTime(job.lastRun) : '从未执行' }}</span>
              </div>
            </td>
            <td>
              <div class="table__actions">
                <UButton
                  variant="ghost"
                  icon-only
                  size="sm"
                  icon="play"
                  aria-label="立即执行"
                  title="立即执行"
                  :loading="busyId === job.id"
                  @click="runJob(job)"
                />
                <UButton
                  variant="ghost"
                  icon-only
                  size="sm"
                  icon="logs"
                  aria-label="查看上次输出"
                  title="上次输出"
                  @click="openOutput(job)"
                />
                <UButton
                  variant="ghost"
                  icon-only
                  size="sm"
                  icon="edit"
                  aria-label="编辑任务"
                  title="编辑"
                  @click="openEdit(job)"
                />
                <UButton
                  variant="ghost"
                  icon-only
                  size="sm"
                  icon="trash"
                  aria-label="删除任务"
                  title="删除"
                  @click="openDelete(job)"
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
        调度器每 20 秒检查一次，按分钟匹配表达式；同一分钟内不会重复执行。单次命令最长运行 10 分钟，输出保留最近 16KB。
      </div>
    </div>

    <!-- 新建/编辑 -->
    <UModal
      v-model="formOpen"
      :title="editingId ? '编辑计划任务' : '新建计划任务'"
      size="lg"
      :busy="submitting"
      @close="clearFormError"
    >
      <form id="cron-form" class="stack" @submit.prevent="submitForm">
        <div v-if="formError" class="alert alert--danger" role="alert">
          <UIcon name="alert" :size="16" class="alert__icon" />
          <div class="alert__body">{{ formError }}</div>
        </div>
        <div class="form-grid">
          <UField label="任务名称" for-id="cron-name" required>
            <UInput id="cron-name" v-model="form.name"
              autofocus placeholder="例如：每天清理日志" />
          </UField>
          <UField label="cron 表达式" for-id="cron-schedule" required hint="5 段：分 时 日 月 周">
            <UInput id="cron-schedule" v-model="form.schedule" class="mono" placeholder="0 3 * * *" />
          </UField>
        </div>
        <div class="chip-row">
          <button
            v-for="preset in presets"
            :key="preset.value"
            type="button"
            class="chip"
            :class="{ 'chip--active': form.schedule === preset.value }"
            @click="form.schedule = preset.value"
          >
            {{ preset.label }}
          </button>
        </div>
        <UField label="命令" for-id="cron-command" required hint="通过 /bin/sh -c 执行，支持管道与重定向">
          <UTextarea
            id="cron-command"
            v-model="form.command"
            code
            :rows="6"
            placeholder="find /opt/upanel/data/logs -type f -mtime +7 -delete"
          />
        </UField>
        <USwitch v-model="form.enabled" label="创建后立即启用" />
      </form>
      <template #footer>
        <UButton variant="secondary" @click="formOpen = false">取消</UButton>
        <UButton variant="primary" :loading="submitting" icon="save" :label="editingId ? '保存' : '创建任务'" @click="submitForm" />
      </template>
    </UModal>

    <!-- 输出 -->
    <UModal v-model="outputOpen" :title="`${outputJob?.name || ''} · 上次输出`" size="xl">
      <div class="stack">
        <div class="row">
          <UStatusDot :status="outputJob?.lastStatus || 'unknown'" :text="runStatusText(outputJob?.lastStatus)" />
          <span class="subtle" style="font-size: 12px">{{ outputJob?.lastRun ? formatDate(outputJob.lastRun) : '尚未执行' }}</span>
        </div>
        <pre v-if="outputJob?.lastOutput" class="log">{{ outputJob.lastOutput }}</pre>
        <UEmpty v-else icon="logs" title="暂无输出" description="任务执行后，stdout/stderr 会显示在这里。" />
      </div>
      <template #footer>
        <UButton variant="secondary" @click="outputOpen = false">关闭</UButton>
      </template>
    </UModal>

    <!-- 删除 -->
    <UModal v-model="deleteOpen" title="删除计划任务" size="sm" :busy="submitting">
      <p>确定删除任务 <strong>{{ deleteJob?.name }}</strong> 吗？删除后不会再执行。</p>
      <template #footer>
        <UButton variant="secondary" @click="deleteOpen = false">取消</UButton>
        <UButton variant="danger" :loading="submitting" icon="trash" label="删除" @click="submitDelete" />
      </template>
    </UModal>
  </div>
</template>

<script setup>
import { onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import UButton from '@/components/ui/UButton.vue'
import UEmpty from '@/components/ui/UEmpty.vue'
import UField from '@/components/ui/UField.vue'
import UIcon from '@/components/ui/UIcon.vue'
import UInput from '@/components/ui/UInput.vue'
import UModal from '@/components/ui/UModal.vue'
import UStatusDot from '@/components/ui/UStatusDot.vue'
import USwitch from '@/components/ui/USwitch.vue'
import UTextarea from '@/components/ui/UTextarea.vue'
import { cronApi, errText } from '@/api'
import { formatDate, relativeTime } from '@/composables/useFormat'
import { toastError, toastSuccess } from '@/composables/useToast'

const jobs = ref([])
const loading = ref(false)
const error = ref('')
const busyId = ref('')
let timer = null

const presets = [
  { label: '每 5 分钟', value: '*/5 * * * *' },
  { label: '每小时', value: '0 * * * *' },
  { label: '每天 03:00', value: '0 3 * * *' },
  { label: '每周一 03:00', value: '0 3 * * 1' },
  { label: '每月 1 日', value: '0 3 1 * *' },
]

const SCHEDULE_LABELS = {
  '*/5 * * * *': '每 5 分钟',
  '0 * * * *': '每小时整点',
  '0 3 * * *': '每天 03:00',
  '0 0 * * *': '每天 00:00',
  '0 3 * * 1': '每周一 03:00',
  '0 3 1 * *': '每月 1 日 03:00',
  '@daily': '每天 00:00',
  '@hourly': '每小时',
  '@weekly': '每周日 00:00',
  '@monthly': '每月 1 日 00:00',
}

function scheduleLabel(schedule) {
  return SCHEDULE_LABELS[schedule] || '自定义表达式'
}

function runStatusText(status) {
  return (
    {
      success: '成功',
      failed: '失败',
      timeout: '超时',
      running: '执行中',
      '': '未执行',
    }[status || ''] || status
  )
}

async function loadJobs() {
  loading.value = true
  error.value = ''
  try {
    const res = await cronApi.list()
    jobs.value = res.data || []
  } catch (e) {
    error.value = errText(e, '加载计划任务失败')
  } finally {
    loading.value = false
  }
}

// form
const formOpen = ref(false)
const editingId = ref('')
const form = reactive({ name: '', schedule: '0 3 * * *', command: '', enabled: true })
const formError = ref('')
const submitting = ref(false)

function openCreate() {
  editingId.value = ''
  form.name = ''
  form.schedule = '0 3 * * *'
  form.command = ''
  form.enabled = true
  formError.value = ''
  formOpen.value = true
}

function openEdit(job) {
  editingId.value = job.id
  form.name = job.name
  form.schedule = job.schedule
  form.command = job.command
  form.enabled = job.enabled
  formError.value = ''
  formOpen.value = true
}

function clearFormError() {
  formError.value = ''
}

async function submitForm() {
  if (submitting.value) return
  formError.value = ''
  if (!form.name.trim()) {
    formError.value = '请输入任务名称'
    return
  }
  if (!form.schedule.trim()) {
    formError.value = '请输入 cron 表达式'
    return
  }
  if (!form.command.trim()) {
    formError.value = '请输入要执行的命令'
    return
  }
  const payload = {
    name: form.name.trim(),
    schedule: form.schedule.trim(),
    command: form.command,
    enabled: form.enabled,
  }
  submitting.value = true
  try {
    if (editingId.value) await cronApi.update(editingId.value, payload)
    else await cronApi.create(payload)
    toastSuccess(editingId.value ? '任务已更新' : '任务已创建')
    formOpen.value = false
    await loadJobs()
  } catch (e) {
    formError.value = errText(e, '保存任务失败')
  } finally {
    submitting.value = false
  }
}

async function toggleEnabled(job, value) {
  try {
    await cronApi.update(job.id, {
      name: job.name,
      schedule: job.schedule,
      command: job.command,
      enabled: value,
    })
    job.enabled = value
    toastSuccess(value ? '任务已启用' : '任务已禁用')
  } catch (e) {
    toastError('操作失败', errText(e))
    await loadJobs()
  }
}

async function runJob(job) {
  busyId.value = job.id
  try {
    await cronApi.run(job.id)
    toastSuccess('任务已开始执行', '稍后可刷新查看输出')
    setTimeout(loadJobs, 1500)
  } catch (e) {
    toastError('执行失败', errText(e))
  } finally {
    setTimeout(() => {
      busyId.value = ''
    }, 1500)
  }
}

// output
const outputOpen = ref(false)
const outputJob = ref(null)

function openOutput(job) {
  outputJob.value = job
  outputOpen.value = true
}

// delete
const deleteOpen = ref(false)
const deleteJob = ref(null)

function openDelete(job) {
  deleteJob.value = job
  deleteOpen.value = true
}

async function submitDelete() {
  if (submitting.value || !deleteJob.value) return
  submitting.value = true
  try {
    await cronApi.remove(deleteJob.value.id)
    toastSuccess('任务已删除')
    deleteOpen.value = false
    await loadJobs()
  } catch (e) {
    toastError('删除失败', errText(e))
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  loadJobs()
  timer = setInterval(() => {
    if (document.visibilityState === 'visible' && !formOpen.value) loadJobs()
  }, 20000)
})

onBeforeUnmount(() => {
  if (timer) clearInterval(timer)
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
</style>
