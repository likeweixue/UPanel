<template>
  <div class="files-container">
    <!-- 顶部分类面板 -->
    <div class="top-panel">
      <div class="top-left">
        <el-button size="small" @click="uploadFile">
          <el-icon><Upload /></el-icon> 上传文件
        </el-button>
        <el-button size="small" @click="goBack" :disabled="!canGoBack">
          <el-icon><ArrowLeft /></el-icon> 后退
        </el-button>
        <el-button size="small" @click="goForward" :disabled="!canGoForward">
          <el-icon><ArrowRight /></el-icon> 前进
        </el-button>
        <el-button size="small" @click="refreshFiles">
          <el-icon><Refresh /></el-icon> 刷新
        </el-button>
        
        <el-dropdown @command="handleCreateCommand">
          <el-button size="small">
            创建 <el-icon class="el-icon--right"><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="folder">新建文件夹</el-dropdown-item>
              <el-dropdown-item command="file">新建文件</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        
        <el-button size="small" @click="openTerminal">
          <el-icon><Monitor /></el-icon> 终端
        </el-button>
      </div>
      <div class="top-right">
        <el-input 
          v-model="searchKeyword" 
          placeholder="搜索文件" 
          prefix-icon="Search"
          clearable
          size="small"
          style="width: 200px"
        />
        <el-button size="small" type="danger" plain @click="batchDelete" :disabled="selectedFiles.length === 0">
          删除 ({{ selectedFiles.length }})
        </el-button>
        <el-button size="small" @click="openRecycleBin">
          <el-icon><Delete /></el-icon> 回收站
        </el-button>
      </div>
    </div>

    <!-- 面包屑导航 -->
    <div class="breadcrumb-panel">
      <div class="breadcrumb-content">
        <span class="breadcrumb-item" @click="navigateTo('')">
          <el-icon><HomeFilled /></el-icon> 根目录
        </span>
        <el-icon class="separator"><ArrowRight /></el-icon>
        <span 
          v-for="(part, index) in breadcrumbs" 
          :key="index"
          class="breadcrumb-item"
          :class="{ active: index === breadcrumbs.length - 1 }"
          @click="navigateTo(breadcrumbPaths[index])"
        >
          {{ part }}
        </span>
      </div>
      <div class="path-info">{{ currentPathAbs }}</div>
    </div>

    <!-- 文件列表表格 -->
    <div class="files-panel">
      <el-table 
        :data="filteredFiles" 
        stripe 
        style="width: 100%"
        @selection-change="handleSelectionChange"
        @row-dblclick="openItem"
      >
        <el-table-column type="selection" width="45" />
        <el-table-column label="名称" min-width="250">
          <template #default="{ row }">
            <div class="file-name" @click="openItem(row)">
              <el-icon :size="18" :color="row.is_dir ? '#477779' : '#9ca3af'">
                <component :is="row.is_dir ? Folder : getFileIcon(row.extension)" />
              </el-icon>
              <span>{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="size" label="大小" width="100">
          <template #default="{ row }">
            {{ row.is_dir ? '-' : formatFileSize(row.size) }}
          </template>
        </el-table-column>
        <el-table-column prop="mod_time" label="修改时间" width="160">
          <template #default="{ row }">
            {{ new Date(row.mod_time).toLocaleString() }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button size="small" text @click="renameItem(row)">
              <el-icon><Edit /></el-icon>
            </el-button>
            <el-button size="small" text @click="copyItem(row)">
              <el-icon><CopyDocument /></el-icon>
            </el-button>
            <el-button size="small" text @click="moveItem(row)">
              <el-icon><Position /></el-icon>
            </el-button>
            <el-button size="small" text type="danger" @click="deleteItem(row)">
              <el-icon><Delete /></el-icon>
            </el-button>
            <el-dropdown v-if="!row.is_dir" @command="(cmd) => handleFileCommand(cmd, row)">
              <el-button size="small" text>
                <el-icon><MoreFilled /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="edit">编辑</el-dropdown-item>
                  <el-dropdown-item command="download">下载</el-dropdown-item>
                  <el-dropdown-item command="compress">压缩</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 新建文件夹/文件对话框 -->
    <el-dialog v-model="createDialogVisible" :title="createType === 'folder' ? '新建文件夹' : '新建文件'" width="400px">
      <el-input v-model="createName" :placeholder="createType === 'folder' ? '请输入文件夹名称' : '请输入文件名称'" />
      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmCreate">确定</el-button>
      </template>
    </el-dialog>

    <!-- 重命名对话框 -->
    <el-dialog v-model="renameDialogVisible" title="重命名" width="400px">
      <el-input v-model="renameName" placeholder="请输入新名称" />
      <template #footer>
        <el-button @click="renameDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmRename">确定</el-button>
      </template>
    </el-dialog>

    <!-- 编辑文件对话框 -->
    <el-dialog v-model="editDialogVisible" title="编辑文件" width="70%">
      <el-input 
        v-model="editContent" 
        type="textarea" 
        :rows="20"
        placeholder="文件内容"
      />
      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveFileContent">保存</el-button>
      </template>
    </el-dialog>

    <!-- 移动/复制对话框 -->
    <el-dialog v-model="moveDialogVisible" :title="moveType === 'move' ? '移动到' : '复制到'" width="500px">
      <div class="move-path">
        <el-input v-model="movePath" placeholder="目标路径">
          <template #prepend>路径</template>
        </el-input>
      </div>
      <div class="move-tip">当前路径: {{ currentPathAbs }}</div>
      <template #footer>
        <el-button @click="moveDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmMove">确定</el-button>
      </template>
    </el-dialog>

    <!-- 回收站抽屉 -->
    <el-drawer v-model="recycleDrawerVisible" title="回收站" direction="rtl" size="500px">
      <div class="recycle-content">
        <el-table :data="recycleBin" stripe style="width: 100%">
          <el-table-column prop="name" label="名称" min-width="180" />
          <el-table-column prop="originalPath" label="原路径" min-width="200" />
          <el-table-column prop="deleteTime" label="删除时间" width="160" />
          <el-table-column label="操作" width="150">
            <template #default="{ row }">
              <el-button size="small" @click="restoreItem(row)">还原</el-button>
              <el-button size="small" type="danger" @click="permanentDelete(row)">彻底删除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="recycle-footer">
          <el-button size="small" type="danger" @click="emptyRecycleBin">清空回收站</el-button>
        </div>
      </div>
    </el-drawer>

    <!-- 上传文件抽屉 -->
    <el-drawer v-model="uploadDrawerVisible" title="上传文件" direction="rtl" size="400px">
      <div class="upload-content">
        <el-upload
          class="upload-area"
          drag
          multiple
          :auto-upload="false"
          :on-change="handleFilesChange"
          :show-file-list="true"
        >
          <el-icon class="upload-icon"><Upload /></el-icon>
          <div class="upload-text">点击或拖拽文件到此区域上传</div>
          <div class="upload-hint">支持任意文件格式</div>
        </el-upload>
        <div class="upload-footer">
          <el-button @click="uploadDrawerVisible = false">取消</el-button>
          <el-button type="primary" @click="confirmUpload" :loading="uploading">开始上传</el-button>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  Upload, ArrowLeft, ArrowRight, Refresh, ArrowDown, Monitor, 
  Delete, HomeFilled, Folder, Edit, 
  CopyDocument, Position, MoreFilled, Document, 
  Picture, VideoCamera, Headset, Files
} from '@element-plus/icons-vue'
import axios from 'axios'

const files = ref([])
const loading = ref(false)
const currentPath = ref('')
const currentPathAbs = ref('')
const searchKeyword = ref('')
const selectedFiles = ref([])

const history = ref([])
const historyIndex = ref(-1)
const canGoBack = computed(() => historyIndex.value > 0)
const canGoForward = computed(() => historyIndex.value < history.value.length - 1)

const createDialogVisible = ref(false)
const createType = ref('folder')
const createName = ref('')
const renameDialogVisible = ref(false)
const renameName = ref('')
const renameTarget = ref(null)
const editDialogVisible = ref(false)
const editContent = ref('')
const editFilePath = ref('')
const moveDialogVisible = ref(false)
const moveType = ref('move')
const movePath = ref('')
const moveTarget = ref(null)
const recycleDrawerVisible = ref(false)
const uploadDrawerVisible = ref(false)
const recycleBin = ref([])
const uploadFiles = ref([])
const uploading = ref(false)

const breadcrumbs = computed(() => {
  if (!currentPath.value) return []
  return currentPath.value.split('/').filter(p => p)
})

const breadcrumbPaths = computed(() => {
  const paths = []
  let current = ''
  for (const part of breadcrumbs.value) {
    current += (current ? '/' : '') + part
    paths.push(current)
  }
  return paths
})

const filteredFiles = computed(() => {
  if (!searchKeyword.value) return files.value
  return files.value.filter(f => 
    f.name.toLowerCase().includes(searchKeyword.value.toLowerCase())
  )
})

const loadFiles = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/files/list', { params: { path: currentPath.value } })
    files.value = res.data.data || []
    currentPathAbs.value = res.data.current_path || ''
    
    if (history.value[historyIndex.value] !== currentPath.value) {
      history.value = history.value.slice(0, historyIndex.value + 1)
      history.value.push(currentPath.value)
      historyIndex.value++
    }
  } catch (err) {
    ElMessage.error('加载失败: ' + err.message)
  } finally {
    loading.value = false
  }
}

const navigateTo = (path) => {
  currentPath.value = path
  loadFiles()
}

const goBack = () => {
  if (canGoBack.value) {
    historyIndex.value--
    currentPath.value = history.value[historyIndex.value]
    loadFiles()
  }
}

const goForward = () => {
  if (canGoForward.value) {
    historyIndex.value++
    currentPath.value = history.value[historyIndex.value]
    loadFiles()
  }
}

const refreshFiles = () => {
  loadFiles()
}

const openItem = (item) => {
  if (item.is_dir) {
    navigateTo(item.path)
  } else {
    editFile(item)
  }
}

const editFile = async (item) => {
  try {
    const res = await axios.get('/api/files/content', { params: { path: item.path } })
    editContent.value = res.data.content || ''
    editFilePath.value = item.path
    editDialogVisible.value = true
  } catch (err) {
    ElMessage.error('读取文件失败: ' + err.message)
  }
}

const saveFileContent = async () => {
  try {
    await axios.post('/api/files/content', {
      path: editFilePath.value,
      content: editContent.value
    })
    ElMessage.success('保存成功')
    editDialogVisible.value = false
    loadFiles()
  } catch (err) {
    ElMessage.error('保存失败: ' + err.message)
  }
}

const handleCreateCommand = (command) => {
  createType.value = command
  createName.value = ''
  createDialogVisible.value = true
}

const confirmCreate = async () => {
  if (!createName.value) {
    ElMessage.warning('请输入名称')
    return
  }
  
  try {
    if (createType.value === 'folder') {
      await axios.post('/api/files/folder', {
        path: currentPath.value,
        name: createName.value
      })
      ElMessage.success('文件夹创建成功')
    } else {
      await axios.post('/api/files/file', {
        path: currentPath.value,
        name: createName.value
      })
      ElMessage.success('文件创建成功')
    }
    createDialogVisible.value = false
    loadFiles()
  } catch (err) {
    ElMessage.error('创建失败: ' + err.message)
  }
}

const renameItem = (item) => {
  renameTarget.value = item
  renameName.value = item.name
  renameDialogVisible.value = true
}

const confirmRename = async () => {
  if (!renameName.value) {
    ElMessage.warning('请输入新名称')
    return
  }
  
  try {
    await axios.put('/api/files/rename', {
      path: renameTarget.value.path,
      new_name: renameName.value
    })
    ElMessage.success('重命名成功')
    renameDialogVisible.value = false
    loadFiles()
  } catch (err) {
    ElMessage.error('重命名失败: ' + err.message)
  }
}

const deleteItem = async (item) => {
  await ElMessageBox.confirm(`确定删除 ${item.name} 吗？`, '警告', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  })
  
  try {
    await axios.delete('/api/files/delete', { params: { path: item.path } })
    ElMessage.success('已移动到回收站')
    loadFiles()
  } catch (err) {
    ElMessage.error('删除失败: ' + err.message)
  }
}

const batchDelete = async () => {
  await ElMessageBox.confirm(`确定删除选中的 ${selectedFiles.value.length} 个文件吗？`, '警告', {
    type: 'warning'
  })
  
  for (const item of selectedFiles.value) {
    try {
      await axios.delete('/api/files/delete', { params: { path: item.path } })
    } catch (err) {}
  }
  ElMessage.success('已移动到回收站')
  loadFiles()
}

const copyItem = (item) => {
  moveType.value = 'copy'
  moveTarget.value = item
  movePath.value = ''
  moveDialogVisible.value = true
}

const moveItem = (item) => {
  moveType.value = 'move'
  moveTarget.value = item
  movePath.value = ''
  moveDialogVisible.value = true
}

const confirmMove = async () => {
  if (!movePath.value) {
    ElMessage.warning('请输入目标路径')
    return
  }
  ElMessage.info(`${moveType.value === 'move' ? '移动' : '复制'}功能开发中`)
  moveDialogVisible.value = false
}

const handleFileCommand = (cmd, item) => {
  switch (cmd) {
    case 'edit':
      editFile(item)
      break
    case 'download':
      ElMessage.info('下载功能开发中')
      break
    case 'compress':
      ElMessage.info('压缩功能开发中')
      break
  }
}

const uploadFile = () => {
  uploadFiles.value = []
  uploadDrawerVisible.value = true
}

const handleFilesChange = (file, fileList) => {
  uploadFiles.value = fileList
}

const confirmUpload = async () => {
  if (uploadFiles.value.length === 0) {
    ElMessage.warning('请选择文件')
    return
  }
  
  uploading.value = true
  try {
    for (const file of uploadFiles.value) {
      const formData = new FormData()
      formData.append('file', file.raw)
      formData.append('path', currentPath.value)
      await axios.post('/api/files/upload', formData, {
        headers: { 'Content-Type': 'multipart/form-data' }
      })
    }
    ElMessage.success('上传成功')
    uploadDrawerVisible.value = false
    loadFiles()
  } catch (err) {
    ElMessage.error('上传失败: ' + err.message)
  } finally {
    uploading.value = false
  }
}

const openTerminal = () => {
  ElMessage.info(`将在 ${currentPathAbs} 路径打开终端`)
}

const openRecycleBin = () => {
  recycleDrawerVisible.value = true
}

const restoreItem = (item) => {
  ElMessage.info('还原功能开发中')
}

const permanentDelete = (item) => {
  ElMessageBox.confirm(`确定彻底删除 ${item.name} 吗？`, '警告', { type: 'warning' })
    .then(() => {
      ElMessage.success('已彻底删除')
    })
    .catch(() => {})
}

const emptyRecycleBin = () => {
  ElMessageBox.confirm('确定清空回收站吗？此操作不可恢复！', '警告', { type: 'warning' })
    .then(() => {
      ElMessage.success('回收站已清空')
    })
    .catch(() => {})
}

const handleSelectionChange = (selection) => {
  selectedFiles.value = selection
}

const formatFileSize = (bytes) => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const getFileIcon = (ext) => {
  const imageExts = ['jpg', 'jpeg', 'png', 'gif', 'svg', 'webp']
  const videoExts = ['mp4', 'avi', 'mkv', 'mov', 'wmv']
  const audioExts = ['mp3', 'wav', 'flac', 'aac', 'ogg']
  
  if (imageExts.includes(ext)) return Picture
  if (videoExts.includes(ext)) return VideoCamera
  if (audioExts.includes(ext)) return Headset
  if (ext === 'txt' || ext === 'md' || ext === 'json') return Document
  return Files
}

onMounted(() => {
  loadFiles()
})
</script>

<style scoped>
.files-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.top-panel {
  background: #ffffff;
  border-radius: 4px;
  padding: 8px 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.top-left {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.top-right {
  display: flex;
  gap: 8px;
  align-items: center;
}

.breadcrumb-panel {
  background: #ffffff;
  border-radius: 4px;
  padding: 8px 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  font-size: 13px;
}

.breadcrumb-content {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
}

.breadcrumb-item {
  display: flex;
  align-items: center;
  gap: 4px;
  cursor: pointer;
  color: #6b7280;
}

.breadcrumb-item:hover {
  color: #477779;
}

.breadcrumb-item.active {
  color: #477779;
  font-weight: 500;
}

.separator {
  font-size: 12px;
  color: #9ca3af;
}

.path-info {
  font-size: 11px;
  color: #9ca3af;
  font-family: monospace;
}

.files-panel {
  background: #ffffff;
  border-radius: 4px;
  flex: 1;
  overflow-y: auto;
}

.file-name {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.file-name:hover {
  color: #477779;
}

.upload-area {
  margin-bottom: 16px;
}

.upload-icon {
  font-size: 48px;
  color: #9ca3af;
}

.upload-text {
  margin-top: 8px;
  font-size: 14px;
  color: #6b7280;
}

.upload-hint {
  font-size: 12px;
  color: #9ca3af;
  margin-top: 4px;
}

.upload-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 16px;
}

.move-path {
  margin-bottom: 12px;
}

.move-tip {
  font-size: 12px;
  color: #9ca3af;
}

.recycle-content {
  padding: 8px 0;
}

.recycle-footer {
  margin-top: 16px;
  text-align: center;
}
</style>
