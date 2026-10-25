<template>
  <div class="page">
    <header class="page__header">
      <div>
        <div class="page__title-row">
          <h1 class="page__title">文件管理</h1>
          <span class="badge">{{ rootPath || '/' }}</span>
        </div>
        <p class="page__subtitle truncate">{{ currentAbs || '正在读取文件系统…' }}</p>
      </div>
      <div class="page__actions">
        <UButton icon="refresh" label="刷新" :loading="loading" @click="loadPath(currentPath)" />
        <UButton icon="folder" label="新建文件夹" @click="openCreate('folder')" />
        <UButton icon="file" label="新建文件" @click="openCreate('file')" />
        <UButton variant="primary" icon="upload" label="上传文件" @click="triggerUpload" />
      </div>
    </header>

    <div v-if="error" class="alert alert--danger" role="alert">
      <UIcon name="alert" :size="16" class="alert__icon" />
      <div class="alert__body">{{ error }}</div>
    </div>

    <div class="toolbar">
      <nav class="breadcrumb" aria-label="当前目录">
        <button type="button" class="breadcrumb__item" @click="loadPath('')">
          <UIcon name="home" :size="14" /> 根目录
        </button>
        <template v-for="(crumb, index) in breadcrumbs" :key="crumb.path">
          <UIcon name="chevron-right" :size="12" class="subtle" aria-hidden="true" />
          <button
            type="button"
            class="breadcrumb__item"
            :class="{ 'breadcrumb__item--current': index === breadcrumbs.length - 1 }"
            @click="loadPath(crumb.path)"
          >
            {{ crumb.label }}
          </button>
        </template>
      </nav>
      <div class="toolbar__right">
        <UButton
          variant="secondary"
          icon="arrow-up"
          label="上级目录"
          :disabled="!currentPath"
          @click="goUp"
        />
        <div class="input-affix" style="width: 200px">
          <UIcon name="search" :size="14" class="subtle" style="position: absolute; left: 10px" />
          <input v-model="search" class="input input--sm" placeholder="筛选当前目录" style="padding-left: 30px" />
        </div>
      </div>
    </div>

    <div v-if="loading" class="loading-block"><span class="spinner" /> 正在读取目录…</div>

    <div v-else-if="filteredEntries.length === 0" class="table-wrap">
      <UEmpty
        icon="folder-open"
        :title="search ? '没有匹配的文件' : '这个目录是空的'"
        :description="search ? '换个关键词试试。' : '上传文件或新建文件夹开始使用。'"
      >
        <div class="row">
          <UButton icon="upload" label="上传文件" @click="triggerUpload" />
          <UButton icon="plus" label="新建文件夹" @click="openCreate('folder')" />
        </div>
      </UEmpty>
    </div>

    <div v-else class="table-wrap">
      <table class="table">
        <caption class="sr-only">目录文件列表</caption>
        <thead>
          <tr>
            <th scope="col">名称</th>
            <th scope="col">大小</th>
            <th scope="col">修改时间</th>
            <th scope="col">权限</th>
            <th scope="col" style="text-align: right">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="entry in filteredEntries" :key="entry.path">
            <td>
              <button type="button" class="file-open" @click="openEntry(entry)">
                <span class="file-open__icon" :class="entry.isDir ? 'file-open__icon--dir' : ''">
                  <UIcon :name="fileIcon(entry)" :size="16" />
                </span>
                <span class="truncate">{{ entry.name }}</span>
                <span v-if="entry.isDir" class="subtle" style="font-size: 11px">目录</span>
              </button>
            </td>
            <td class="muted">{{ entry.isDir ? '—' : formatBytes(entry.size) }}</td>
            <td class="muted">{{ formatDate(entry.modTime) }}</td>
            <td><span class="chip mono" style="font-size: 10px">{{ entry.mode }}</span></td>
            <td>
              <div class="table__actions">
                <UButton
                  v-if="!entry.isDir"
                  variant="ghost"
                  icon-only
                  size="sm"
                  icon="eye"
                  aria-label="编辑文件内容"
                  title="编辑"
                  @click="openEditor(entry)"
                />
                <UButton
                  v-if="!entry.isDir"
                  variant="ghost"
                  icon-only
                  size="sm"
                  icon="download"
                  :aria-label="`下载 ${entry.name}`"
                  title="下载"
                  @click="downloadEntry(entry)"
                />
                <UButton
                  variant="ghost"
                  icon-only
                  size="sm"
                  icon="edit"
                  aria-label="重命名"
                  title="重命名"
                  @click="openRename(entry)"
                />
                <UButton
                  variant="ghost"
                  icon-only
                  size="sm"
                  icon="trash"
                  aria-label="删除"
                  title="删除"
                  @click="openDelete(entry)"
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
        文件管理以面板进程权限运行（通常是 root），请谨慎修改系统目录。单个文件读取上限 8MB，上传上限 2GB。
      </div>
    </div>

    <!-- 上传 -->
    <UDrawer
      v-model="uploadOpen"
      title="上传文件"
      :description="`目标目录：${currentAbs || '/'}`"
      size="sm"
      :busy="uploading"
      @close="resetUpload"
    >
      <div class="stack">
        <label
          class="upload-drop"
          @dragover.prevent
          @drop.prevent="onDropUpload"
        >
          <UIcon name="upload" :size="24" />
          <strong>{{ uploadTarget ? uploadTarget.name : '点击选择，或拖拽文件到这里' }}</strong>
          <span v-if="uploadTarget" class="subtle">
            {{ formatBytes(uploadTarget.size) }}
          </span>
          <span v-else class="field__hint">单个文件最大 2GB</span>
          <input
            ref="uploadInput"
            type="file"
            class="sr-only"
            tabindex="-1"
            @change="onPickUpload"
          />
        </label>
        <p v-if="uploadError" class="field__error" role="alert">{{ uploadError }}</p>
        <p class="field__hint">上传会以面板进程权限写入当前目录，请确认目标路径。</p>
      </div>
      <template #footer>
        <UButton variant="secondary" :disabled="uploading" @click="closeUpload">取消</UButton>
        <UButton
          variant="primary"
          icon="upload"
          label="开始上传"
          :loading="uploading"
          :disabled="!uploadTarget"
          @click="submitUpload"
        />
      </template>
    </UDrawer>

    <!-- 新建 -->
    <UDrawer v-model="createOpen" :title="createType === 'folder' ? '新建文件夹' : '新建文件'" size="sm" :busy="submitting">
      <form id="create-entry-form" @submit.prevent="submitCreate">
        <UField :label="createType === 'folder' ? '文件夹名称' : '文件名'" for-id="entry-name" required>
          <UInput id="entry-name" v-model="createName"
              autofocus autocomplete="off" placeholder="例如 app" @keyup.enter="submitCreate" />
        </UField>
        <p v-if="modalError" class="field__error" role="alert" style="margin-top: 8px">{{ modalError }}</p>
      </form>
      <template #footer>
        <UButton variant="secondary" @click="createOpen = false">取消</UButton>
        <UButton variant="primary" :loading="submitting" label="创建" @click="submitCreate" />
      </template>
    </UDrawer>

    <!-- 重命名 -->
    <UDrawer v-model="renameOpen" title="重命名" size="sm" :busy="submitting">
      <form id="rename-form" @submit.prevent="submitRename">
        <UField label="新名称" for-id="rename-name" required>
          <UInput id="rename-name" v-model="renameName" autocomplete="off" @keyup.enter="submitRename" />
        </UField>
        <p v-if="modalError" class="field__error" role="alert" style="margin-top: 8px">{{ modalError }}</p>
      </form>
      <template #footer>
        <UButton variant="secondary" @click="renameOpen = false">取消</UButton>
        <UButton variant="primary" :loading="submitting" label="保存" @click="submitRename" />
      </template>
    </UDrawer>

    <!-- 删除 -->
    <UModal v-model="deleteOpen" title="删除确认" size="sm" :busy="submitting">
      <div class="stack">
        <p>确定删除 <strong>{{ deleteTarget?.name }}</strong> 吗？</p>
        <div class="alert alert--danger">
          <UIcon name="alert" :size="16" class="alert__icon" />
          <div class="alert__body">
            {{ deleteTarget?.isDir ? '目录及其全部内容会被递归删除。' : '文件会被永久删除。' }}此操作不可撤销。
          </div>
        </div>
      </div>
      <template #footer>
        <UButton variant="secondary" @click="deleteOpen = false">取消</UButton>
        <UButton variant="danger" :loading="submitting" icon="trash" label="删除" @click="submitDelete" />
      </template>
    </UModal>

    <!-- 编辑器 -->
    <UDrawer v-model="editorOpen" :title="editorEntry?.name || '编辑文件'" size="xl" :busy="submitting">
      <div class="stack">
        <p class="subtle mono" style="font-size: 11px">{{ editorEntry?.path }}</p>
        <div v-if="editorLoading" class="loading-block"><span class="spinner" /> 正在读取文件…</div>
        <UTextarea
          v-else
          v-model="editorContent"
          code
          :rows="20"
          aria-label="文件内容"
          spellcheck="false"
        />
        <p v-if="modalError" class="field__error" role="alert">{{ modalError }}</p>
        <div class="row" style="justify-content: space-between">
          <span class="subtle" style="font-size: 12px">{{ editorContent.length }} 字符 · 最大 8MB</span>
          <span class="subtle" style="font-size: 12px">保存后将直接写入服务器文件</span>
        </div>
      </div>
      <template #footer>
        <UButton variant="secondary" @click="editorOpen = false">关闭</UButton>
        <UButton variant="primary" :loading="submitting" icon="save" label="保存文件" @click="submitEditor" />
      </template>
    </UDrawer>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import UButton from '@/components/ui/UButton.vue'
import UEmpty from '@/components/ui/UEmpty.vue'
import UField from '@/components/ui/UField.vue'
import UIcon from '@/components/ui/UIcon.vue'
import UInput from '@/components/ui/UInput.vue'
import UDrawer from '@/components/ui/UDrawer.vue'
import UModal from '@/components/ui/UModal.vue'
import UTextarea from '@/components/ui/UTextarea.vue'
import { errText, fileApi } from '@/api'
import { formatBytes, formatDate } from '@/composables/useFormat'
import { toastError, toastSuccess } from '@/composables/useToast'

const route = useRoute()
const router = useRouter()

const rootPath = ref('/')
const currentPath = ref('')
const currentAbs = ref('')
const entries = ref([])
const loading = ref(false)
const error = ref('')
const search = ref('')
const uploadInput = ref(null)
const uploadOpen = ref(false)
const uploadTarget = ref(null)
const uploadError = ref('')
const uploading = ref(false)

const breadcrumbs = computed(() => {
  const parts = currentPath.value.split('/').filter(Boolean)
  return parts.map((part, index) => ({
    label: part,
    path: parts.slice(0, index + 1).join('/'),
  }))
})

const filteredEntries = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return entries.value
  return entries.value.filter((entry) => entry.name.toLowerCase().includes(q))
})

function fileIcon(entry) {
  if (entry.isDir) return 'folder'
  const ext = (entry.extension || '').toLowerCase()
  if (['png', 'jpg', 'jpeg', 'gif', 'svg', 'webp', 'ico'].includes(ext)) return 'image'
  if (['js', 'ts', 'vue', 'go', 'php', 'py', 'sh', 'json', 'yml', 'yaml', 'css', 'html'].includes(ext)) return 'code'
  if (['zip', 'tar', 'gz', 'bz2', 'xz', '7z'].includes(ext)) return 'box'
  return 'file'
}

async function loadPath(path) {
  loading.value = true
  error.value = ''
  search.value = ''
  try {
    const res = await fileApi.list(path || '')
    entries.value = res.data || []
    currentPath.value = res.relative_path || path || ''
    currentAbs.value = res.current_path || ''
  } catch (e) {
    error.value = errText(e, '读取目录失败')
    entries.value = []
  } finally {
    loading.value = false
  }
}

function goUp() {
  if (!currentPath.value) return
  const parts = currentPath.value.split('/').filter(Boolean)
  parts.pop()
  loadPath(parts.join('/'))
}

function openEntry(entry) {
  if (entry.isDir) loadPath(entry.path)
  else openEditor(entry)
}

async function downloadEntry(entry) {
  try {
    const res = await fileApi.download(entry.path)
    const url = URL.createObjectURL(res.data)
    const link = document.createElement('a')
    link.href = url
    link.download = entry.name
    document.body.appendChild(link)
    link.click()
    link.remove()
    setTimeout(() => URL.revokeObjectURL(url), 1000)
  } catch (e) {
    toastError('下载失败', errText(e))
  }
}

function triggerUpload() {
  uploadOpen.value = true
  uploadTarget.value = null
  uploadError.value = ''
}

function resetUpload() {
  uploadTarget.value = null
  uploadError.value = ''
}

function closeUpload() {
  if (uploading.value) return
  uploadOpen.value = false
  resetUpload()
  clearActionQuery()
}

function setUploadFile(file) {
  if (!file) return
  uploadTarget.value = file
  uploadError.value = ''
}

function onPickUpload(event) {
  const file = event.target.files && event.target.files[0]
  setUploadFile(file)
  event.target.value = ''
}

function onDropUpload(event) {
  const file = event.dataTransfer && event.dataTransfer.files && event.dataTransfer.files[0]
  setUploadFile(file)
}

async function submitUpload() {
  if (uploading.value || !uploadTarget.value) return
  uploading.value = true
  uploadError.value = ''
  try {
    await fileApi.upload(currentPath.value, uploadTarget.value)
    toastSuccess('上传成功', uploadTarget.value.name)
    uploadOpen.value = false
    resetUpload()
    clearActionQuery()
    await loadPath(currentPath.value)
  } catch (e) {
    uploadError.value = errText(e, '上传失败')
  } finally {
    uploading.value = false
  }
}

function clearActionQuery() {
  if (route.query.action === 'upload') {
    const query = { ...route.query }
    delete query.action
    router.replace({ path: route.path, query })
  }
}

// create
const createOpen = ref(false)
const createType = ref('folder')
const createName = ref('')
const modalError = ref('')
const submitting = ref(false)

function openCreate(type) {
  createType.value = type
  createName.value = ''
  modalError.value = ''
  createOpen.value = true
}

async function submitCreate() {
  if (submitting.value) return
  modalError.value = ''
  if (!createName.value.trim()) {
    modalError.value = '请输入名称'
    return
  }
  submitting.value = true
  try {
    if (createType.value === 'folder') await fileApi.folder(currentPath.value, createName.value.trim())
    else await fileApi.file(currentPath.value, createName.value.trim())
    toastSuccess('创建成功', createName.value)
    createOpen.value = false
    await loadPath(currentPath.value)
  } catch (e) {
    modalError.value = errText(e, '创建失败')
  } finally {
    submitting.value = false
  }
}

// rename
const renameOpen = ref(false)
const renameTarget = ref(null)
const renameName = ref('')

function openRename(entry) {
  renameTarget.value = entry
  renameName.value = entry.name
  modalError.value = ''
  renameOpen.value = true
}

async function submitRename() {
  if (submitting.value || !renameTarget.value) return
  modalError.value = ''
  if (!renameName.value.trim()) {
    modalError.value = '请输入新名称'
    return
  }
  submitting.value = true
  try {
    await fileApi.rename(renameTarget.value.path, renameName.value.trim())
    toastSuccess('重命名成功')
    renameOpen.value = false
    await loadPath(currentPath.value)
  } catch (e) {
    modalError.value = errText(e, '重命名失败')
  } finally {
    submitting.value = false
  }
}

// delete
const deleteOpen = ref(false)
const deleteTarget = ref(null)

function openDelete(entry) {
  deleteTarget.value = entry
  deleteOpen.value = true
}

async function submitDelete() {
  if (submitting.value || !deleteTarget.value) return
  submitting.value = true
  try {
    await fileApi.remove(deleteTarget.value.path)
    toastSuccess('删除成功', deleteTarget.value.name)
    deleteOpen.value = false
    await loadPath(currentPath.value)
  } catch (e) {
    toastError('删除失败', errText(e))
  } finally {
    submitting.value = false
  }
}

// editor
const editorOpen = ref(false)
const editorLoading = ref(false)
const editorEntry = ref(null)
const editorContent = ref('')

async function openEditor(entry) {
  editorEntry.value = entry
  editorContent.value = ''
  modalError.value = ''
  editorOpen.value = true
  editorLoading.value = true
  try {
    const res = await fileApi.content(entry.path)
    editorContent.value = res.content || ''
  } catch (e) {
    modalError.value = errText(e, '读取文件失败')
  } finally {
    editorLoading.value = false
  }
}

async function submitEditor() {
  if (submitting.value || !editorEntry.value) return
  modalError.value = ''
  submitting.value = true
  try {
    await fileApi.save(editorEntry.value.path, editorContent.value)
    toastSuccess('文件已保存', editorEntry.value.name)
    editorOpen.value = false
    await loadPath(currentPath.value)
  } catch (e) {
    modalError.value = errText(e, '保存失败')
  } finally {
    submitting.value = false
  }
}

watch(
  () => route.query.action,
  (action) => {
    if (action === 'upload') setTimeout(triggerUpload, 100)
  }
)

onMounted(async () => {
  try {
    const res = await fileApi.root()
    rootPath.value = res.root || '/'
  } catch (e) {
    /* 忽略 */
  }
  await loadPath('')
  if (route.query.action === 'upload') setTimeout(triggerUpload, 150)
})
</script>

<style scoped>
.file-open {
  display: flex;
  align-items: center;
  gap: 8px;
  max-width: 360px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--color-text);
  font-size: var(--text-md);
  text-align: left;
}

.file-open:hover {
  color: var(--color-primary);
}

.file-open__icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: var(--radius-xs);
  background: var(--color-surface-3);
  color: var(--color-text-muted);
  flex: 0 0 auto;
}

.file-open__icon--dir {
  background: var(--color-primary-soft);
  color: var(--color-primary);
}

a.btn {
  text-decoration: none;
}
</style>
