<template>
  <div>
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-bold text-gray-800">文件管理</h1>
      <div class="flex gap-2">
        <el-button type="primary" @click="showCreateDialog = true">
          <el-icon><Plus /></el-icon> 新建
        </el-button>
        <el-upload
          :action="uploadUrl"
          :data="{ path: currentPath }"
          :headers="uploadHeaders"
          :show-file-list="false"
          :on-success="handleUploadSuccess"
          :on-error="handleUploadError"
        >
          <el-button type="success">
            <el-icon><Upload /></el-icon> 上传文件
          </el-button>
        </el-upload>
        <el-button @click="loadFiles">
          <el-icon><Refresh /></el-icon> 刷新
        </el-button>
      </div>
    </div>

    <!-- 面包屑导航 -->
    <div class="bg-white rounded-lg shadow-sm p-3 mb-4">
      <el-breadcrumb separator="/">
        <el-breadcrumb-item @click="navigateTo('')">
          <el-icon><HomeFilled /></el-icon> 根目录
        </el-breadcrumb-item>
        <el-breadcrumb-item 
          v-for="(part, index) in breadcrumbs" 
          :key="index"
          @click="navigateTo(breadcrumbPaths[index])"
        >
          {{ part }}
        </el-breadcrumb-item>
      </el-breadcrumb>
      <div class="text-xs text-gray-400 mt-2">
        当前路径: {{ currentPathAbs }}
      </div>
    </div>

    <!-- 文件列表 -->
    <div class="bg-white rounded-lg shadow-sm">
      <el-table :data="files" stripe v-loading="loading" class="w-full">
        <el-table-column label="名称" min-width="200">
          <template #default="{ row }">
            <div class="flex items-center gap-2 cursor-pointer" @click="openItem(row)">
              <el-icon :size="20">
                <component :is="row.is_dir ? Folder : getFileIcon(row.extension)" />
              </el-icon>
              <span>{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="size" label="大小" width="120">
          <template #default="{ row }">
            {{ row.is_dir ? '-' : formatFileSize(row.size) }}
          </template>
        </el-table-column>
        <el-table-column prop="mod_time" label="修改时间" width="180">
          <template #default="{ row }">
            {{ new Date(row.mod_time).toLocaleString() }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button 
              v-if="!row.is_dir" 
              size="small" 
              @click="editFile(row)"
            >
              编辑
            </el-button>
            <el-button size="small" @click="renameItem(row)">重命名</el-button>
            <el-button size="small" type="danger" @click="deleteItem(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 新建对话框 -->
    <el-dialog v-model="showCreateDialog" title="新建" width="400px">
      <el-form>
        <el-form-item label="类型">
          <el-radio-group v-model="createType">
            <el-radio value="folder">文件夹</el-radio>
            <el-radio value="file">文件</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="名称">
          <el-input v-model="createName" placeholder="请输入名称" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">取消</el-button>
        <el-button type="primary" @click="createItem">确定</el-button>
      </template>
    </el-dialog>

    <!-- 编辑对话框 -->
    <el-dialog v-model="showEditDialog" title="编辑文件" width="70%">
      <el-input
        v-model="editContent"
        type="textarea"
        :rows="20"
        placeholder="文件内容"
      />
      <template #footer>
        <el-button @click="showEditDialog = false">取消</el-button>
        <el-button type="primary" @click="saveFile">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  Plus, Upload, Refresh, HomeFilled, Folder, 
  Document, Picture, VideoCamera, 
  Headset, Files, InfoFilled 
} from '@element-plus/icons-vue'

const files = ref([])
const loading = ref(false)
const currentPath = ref('')
const currentPathAbs = ref('')
const showCreateDialog = ref(false)
const showEditDialog = ref(false)
const createType = ref('folder')
const createName = ref('')
const editContent = ref('')
const editFilePath = ref('')

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

const uploadUrl = '/api/files/upload'
const uploadHeaders = { 'Content-Type': 'multipart/form-data' }

const loadFiles = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/files/list', { params: { path: currentPath.value } })
    files.value = res.data.data
    currentPathAbs.value = res.data.current_path
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
    editContent.value = res.data.content
    editFilePath.value = item.path
    showEditDialog.value = true
  } catch (err) {
    ElMessage.error('读取文件失败: ' + err.message)
  }
}

const saveFile = async () => {
  try {
    await axios.post('/api/files/content', {
      path: editFilePath.value,
      content: editContent.value
    })
    ElMessage.success('保存成功')
    showEditDialog.value = false
    loadFiles()
  } catch (err) {
    ElMessage.error('保存失败: ' + err.message)
  }
}

const createItem = async () => {
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
    showCreateDialog.value = false
    createName.value = ''
    loadFiles()
  } catch (err) {
    ElMessage.error('创建失败: ' + err.message)
  }
}

const renameItem = async (item) => {
  const { value } = await ElMessageBox.prompt('请输入新名称', '重命名', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    inputValue: item.name
  })
  
  if (value && value !== item.name) {
    try {
      await axios.put('/api/files/rename', {
        path: item.path,
        new_name: value
      })
      ElMessage.success('重命名成功')
      loadFiles()
    } catch (err) {
      ElMessage.error('重命名失败: ' + err.message)
    }
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
    ElMessage.success('删除成功')
    loadFiles()
  } catch (err) {
    ElMessage.error('删除失败: ' + err.message)
  }
}

const handleUploadSuccess = () => {
  ElMessage.success('上传成功')
  loadFiles()
}

const handleUploadError = (err) => {
  ElMessage.error('上传失败: ' + err.message)
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
  if (ext === 'txt' || ext === 'md') return Document
  return Files
}

onMounted(() => {
  loadFiles()
})
</script>
