<template>
  <div>
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-bold text-gray-800">数据库管理</h1>
      <el-button type="primary" @click="showCreateDialog = true">
        <el-icon><Plus /></el-icon> 创建数据库
      </el-button>
    </div>

    <!-- 数据库列表 -->
    <div class="bg-white rounded-lg shadow-sm">
      <el-table :data="databases" stripe v-loading="loading" class="w-full">
        <el-table-column prop="name" label="名称" min-width="150" />
        <el-table-column prop="type" label="类型" width="100">
          <template #default="{ row }">
            <el-tag :type="row.type === 'mysql' ? 'primary' : 'success'">
              {{ row.type === 'mysql' ? 'MySQL' : 'PostgreSQL' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="version" label="版本" width="80" />
        <el-table-column prop="port" label="端口" width="80" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'running' ? 'success' : 'info'">
              {{ row.status === 'running' ? '运行中' : '已停止' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">
            {{ new Date(row.created_at).toLocaleString() }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="320" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="showConnectionInfo(row)">
              <el-icon><Connection /></el-icon> 连接信息
            </el-button>
            <el-button 
              v-if="row.status !== 'running'" 
              size="small" 
              type="success" 
              @click="startDatabase(row)"
            >
              启动
            </el-button>
            <el-button 
              v-if="row.status === 'running'" 
              size="small" 
              type="warning" 
              @click="stopDatabase(row)"
            >
              停止
            </el-button>
            <el-button size="small" type="danger" @click="deleteDatabase(row)">
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 创建数据库对话框 -->
    <el-dialog v-model="showCreateDialog" title="创建数据库" width="500px">
      <el-form :model="createForm" label-width="100px">
        <el-form-item label="名称" required>
          <el-input v-model="createForm.name" placeholder="例如: mydb" />
        </el-form-item>
        <el-form-item label="类型" required>
          <el-radio-group v-model="createForm.type">
            <el-radio value="mysql">MySQL</el-radio>
            <el-radio value="postgres">PostgreSQL</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="版本">
          <el-select v-model="createForm.version">
            <template v-if="createForm.type === 'mysql'">
              <el-option label="MySQL 8.0" value="8.0" />
              <el-option label="MySQL 5.7" value="5.7" />
            </template>
            <template v-else>
              <el-option label="PostgreSQL 15" value="15" />
              <el-option label="PostgreSQL 14" value="14" />
            </template>
          </el-select>
        </el-form-item>
        <el-form-item label="端口">
          <el-input-number v-model="createForm.port" :min="1024" :max="65535" />
          <div class="text-xs text-gray-400 mt-1">留空则使用默认端口</div>
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="createForm.password" type="password" placeholder="留空则自动生成" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">取消</el-button>
        <el-button type="primary" @click="createDatabase" :loading="creating">
          创建
        </el-button>
      </template>
    </el-dialog>

    <!-- 连接信息对话框 -->
    <el-dialog v-model="showConnectionDialog" title="连接信息" width="500px">
      <el-form :model="connectionInfo" label-width="100px">
        <el-form-item label="主机">
          <el-input v-model="connectionInfo.host" readonly />
        </el-form-item>
        <el-form-item label="端口">
          <el-input v-model="connectionInfo.port" readonly />
        </el-form-item>
        <el-form-item label="用户名">
          <el-input v-model="connectionInfo.username" readonly />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="connectionInfo.password" readonly />
          <el-button size="small" @click="copyToClipboard(connectionInfo.password)">复制</el-button>
        </el-form-item>
        <el-form-item label="数据库">
          <el-input v-model="connectionInfo.database" readonly />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showConnectionDialog = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Connection } from '@element-plus/icons-vue'

const databases = ref([])
const loading = ref(false)
const creating = ref(false)
const showCreateDialog = ref(false)
const showConnectionDialog = ref(false)
const connectionInfo = ref({})

const createForm = ref({
  name: '',
  type: 'mysql',
  version: '8.0',
  password: '',
  port: 0
})

const fetchDatabases = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/databases/')
    databases.value = res.data.data || []
    console.log('数据库列表:', databases.value)
  } catch (err) {
    console.error('获取失败:', err)
    ElMessage.error('获取数据库列表失败: ' + err.message)
  } finally {
    loading.value = false
  }
}

const createDatabase = async () => {
  if (!createForm.value.name) {
    ElMessage.warning('请输入数据库名称')
    return
  }
  
  creating.value = true
  try {
    const res = await axios.post('/api/databases/', createForm.value)
    console.log('创建成功:', res.data)
    ElMessage.success('数据库创建成功，正在启动容器...')
    showCreateDialog.value = false
    createForm.value = { name: '', type: 'mysql', version: '8.0', password: '', port: 0 }
    setTimeout(() => fetchDatabases(), 2000)
  } catch (err) {
    console.error('创建失败:', err)
    ElMessage.error('创建失败: ' + (err.response?.data?.error || err.message))
  } finally {
    creating.value = false
  }
}

const startDatabase = async (db) => {
  try {
    await axios.post(`/api/databases/${db.name}/start`)
    ElMessage.success('数据库已启动')
    fetchDatabases()
  } catch (err) {
    ElMessage.error('启动失败: ' + err.message)
  }
}

const stopDatabase = async (db) => {
  try {
    await axios.post(`/api/databases/${db.name}/stop`)
    ElMessage.success('数据库已停止')
    fetchDatabases()
  } catch (err) {
    ElMessage.error('停止失败: ' + err.message)
  }
}

const deleteDatabase = async (db) => {
  await ElMessageBox.confirm(`确定删除数据库 ${db.name} 吗？数据将被删除无法恢复！`, '警告', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  })
  
  try {
    await axios.delete(`/api/databases/${db.name}`)
    ElMessage.success('数据库已删除')
    fetchDatabases()
  } catch (err) {
    ElMessage.error('删除失败: ' + err.message)
  }
}

const showConnectionInfo = async (db) => {
  try {
    const res = await axios.get(`/api/databases/${db.name}/connection`)
    connectionInfo.value = res.data.data
    showConnectionDialog.value = true
  } catch (err) {
    ElMessage.error('获取连接信息失败: ' + err.message)
  }
}

const copyToClipboard = (text) => {
  navigator.clipboard.writeText(text)
  ElMessage.success('已复制到剪贴板')
}

onMounted(() => {
  fetchDatabases()
})
</script>
