<template>
  <div>
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-bold text-gray-800">网站管理</h1>
      <el-button type="primary" @click="showCreateDialog = true">
        <el-icon><Plus /></el-icon> 创建网站
      </el-button>
    </div>

    <!-- 网站列表 -->
    <div class="bg-white rounded-lg shadow-sm">
      <el-table :data="websites" stripe v-loading="loading" class="w-full">
        <el-table-column prop="name" label="域名" min-width="200" />
        <el-table-column prop="php_version" label="PHP版本" width="100" />
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
        <el-table-column label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="openWebsite(row)">
              <el-icon><Link /></el-icon> 访问
            </el-button>
            <el-button 
              v-if="row.status !== 'running'" 
              size="small" 
              type="success" 
              @click="startWebsite(row)"
            >
              启动
            </el-button>
            <el-button 
              v-if="row.status === 'running'" 
              size="small" 
              type="warning" 
              @click="stopWebsite(row)"
            >
              停止
            </el-button>
            <el-button size="small" type="danger" @click="deleteWebsite(row)">
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 创建网站对话框 -->
    <el-dialog v-model="showCreateDialog" title="创建网站" width="500px">
      <el-form :model="createForm" label-width="100px">
        <el-form-item label="域名" required>
          <el-input v-model="createForm.domain" placeholder="例如: example.com" />
          <div class="text-xs text-gray-400 mt-1">支持本地测试域名，如 test.local</div>
        </el-form-item>
        <el-form-item label="PHP版本">
          <el-select v-model="createForm.php_version">
            <el-option label="PHP 8.2" value="8.2" />
            <el-option label="PHP 8.1" value="8.1" />
            <el-option label="PHP 8.0" value="8.0" />
            <el-option label="PHP 7.4" value="7.4" />
          </el-select>
        </el-form-item>
        <el-form-item label="创建数据库">
          <el-switch v-model="createForm.create_db" />
        </el-form-item>
        <el-form-item v-if="createForm.create_db" label="数据库名">
          <el-input v-model="createForm.db_name" placeholder="留空则使用域名" />
        </el-form-item>
        <el-form-item v-if="createForm.create_db" label="数据库密码">
          <el-input v-model="createForm.db_password" type="password" placeholder="留空则使用默认密码" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">取消</el-button>
        <el-button type="primary" @click="createWebsite" :loading="creating">
          创建
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Link } from '@element-plus/icons-vue'

const websites = ref([])
const loading = ref(false)
const creating = ref(false)
const showCreateDialog = ref(false)

const createForm = ref({
  domain: '',
  php_version: '8.2',
  create_db: false,
  db_name: '',
  db_password: ''
})

const fetchWebsites = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/websites/')
    websites.value = res.data.data || []
  } catch (err) {
    ElMessage.error('获取网站列表失败: ' + err.message)
  } finally {
    loading.value = false
  }
}

const createWebsite = async () => {
  if (!createForm.value.domain) {
    ElMessage.warning('请输入域名')
    return
  }
  
  creating.value = true
  try {
    await axios.post('/api/websites/', createForm.value)
    ElMessage.success('网站创建成功，正在启动容器...')
    showCreateDialog.value = false
    createForm.value = { domain: '', php_version: '8.2', create_db: false, db_name: '', db_password: '' }
    setTimeout(() => fetchWebsites(), 2000)
  } catch (err) {
    ElMessage.error('创建失败: ' + err.message)
  } finally {
    creating.value = false
  }
}

const startWebsite = async (website) => {
  try {
    await axios.post(`/api/websites/${website.name}/start`)
    ElMessage.success('网站已启动')
    fetchWebsites()
  } catch (err) {
    ElMessage.error('启动失败: ' + err.message)
  }
}

const stopWebsite = async (website) => {
  try {
    await axios.post(`/api/websites/${website.name}/stop`)
    ElMessage.success('网站已停止')
    fetchWebsites()
  } catch (err) {
    ElMessage.error('停止失败: ' + err.message)
  }
}

const deleteWebsite = async (website) => {
  await ElMessageBox.confirm(`确定删除网站 ${website.name} 吗？删除后无法恢复！`, '警告', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  })
  
  try {
    await axios.delete(`/api/websites/${website.name}`)
    ElMessage.success('网站已删除')
    fetchWebsites()
  } catch (err) {
    ElMessage.error('删除失败: ' + err.message)
  }
}

const openWebsite = async (website) => {
  try {
    const res = await axios.get(`/api/websites/${website.name}/url`)
window.open(res.data, '_blank')
  } catch (err) {
    ElMessage.error('获取访问地址失败: ' + err.message)
  }
}

onMounted(() => {
  fetchWebsites()
})
</script>
