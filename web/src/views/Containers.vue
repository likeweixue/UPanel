<template>
  <div>
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-bold text-gray-800">容器管理</h1>
      <div class="flex gap-2">
        <el-button @click="fetchContainers" :loading="loading" type="primary">
          刷新
        </el-button>
        <el-checkbox v-model="showAll" @change="fetchContainers">
          显示所有容器
        </el-checkbox>
      </div>
    </div>

    <el-table :data="containers" stripe v-loading="loading" class="w-full">
      <el-table-column prop="name" label="容器名称" width="200" />
      <el-table-column prop="image" label="镜像" />
      <el-table-column prop="status" label="状态" width="150">
        <template #default="{ row }">
          <el-tag :type="row.status.includes('Up') ? 'success' : 'info'">
            {{ row.status }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="250" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="startContainer(row.id)" :disabled="row.status.includes('Up')">
            启动
          </el-button>
          <el-button size="small" @click="stopContainer(row.id)" :disabled="!row.status.includes('Up')">
            停止
          </el-button>
          <el-button size="small" @click="restartContainer(row.id)">
            重启
          </el-button>
          <el-button size="small" type="danger" @click="removeContainer(row.id)">
            删除
          </el-button>
          <el-button size="small" @click="viewLogs(row.id)">
            日志
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 日志对话框 -->
    <el-dialog v-model="logDialogVisible" title="容器日志" width="70%">
      <pre class="bg-gray-900 text-green-400 p-4 rounded overflow-auto max-h-96 text-sm">
        {{ logContent }}
      </pre>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const containers = ref([])
const loading = ref(false)
const showAll = ref(true)
const logDialogVisible = ref(false)
const logContent = ref('')

const api = axios.create({
  baseURL: '/api'
})

const fetchContainers = async () => {
  loading.value = true
  try {
    const res = await api.get('/containers/', { params: { all: showAll.value } })
    containers.value = res.data.data
  } catch (err) {
    ElMessage.error('获取容器列表失败: ' + err.message)
  } finally {
    loading.value = false
  }
}

const startContainer = async (id) => {
  try {
    await api.post(`/containers/${id}/start`)
    ElMessage.success('容器已启动')
    fetchContainers()
  } catch (err) {
    ElMessage.error('启动失败: ' + err.message)
  }
}

const stopContainer = async (id) => {
  try {
    await api.post(`/containers/${id}/stop`)
    ElMessage.success('容器已停止')
    fetchContainers()
  } catch (err) {
    ElMessage.error('停止失败: ' + err.message)
  }
}

const restartContainer = async (id) => {
  try {
    await api.post(`/containers/${id}/restart`)
    ElMessage.success('容器已重启')
    fetchContainers()
  } catch (err) {
    ElMessage.error('重启失败: ' + err.message)
  }
}

const removeContainer = async (id) => {
  try {
    await ElMessageBox.confirm('确定要删除这个容器吗？删除后无法恢复！', '警告', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await api.delete(`/containers/${id}`, { params: { force: true } })
    ElMessage.success('容器已删除')
    fetchContainers()
  } catch (err) {
    if (err !== 'cancel') {
      ElMessage.error('删除失败: ' + err.message)
    }
  }
}

const viewLogs = async (id) => {
  try {
    const res = await api.get(`/containers/${id}/logs`, { params: { tail: 200 } })
    logContent.value = res.data.logs || '无日志'
    logDialogVisible.value = true
  } catch (err) {
    ElMessage.error('获取日志失败: ' + err.message)
  }
}

onMounted(() => {
  fetchContainers()
})
</script>