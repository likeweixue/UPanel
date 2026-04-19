<template>
  <div class="docker-config">
    <div class="settings-section">
      <h3 class="section-title">Docker 镜像加速</h3>
      <div class="form-tip">配置镜像加速器可以显著提升 Docker 镜像拉取速度</div>
      
      <el-form label-width="120px" class="settings-form">
        <el-form-item label="镜像加速器">
          <div class="mirror-list">
            <div 
              v-for="(mirror, idx) in config.registry_mirrors" 
              :key="idx"
              class="mirror-item"
            >
              <el-input v-model="config.registry_mirrors[idx]" placeholder="https://example.com" />
              <el-button type="danger" @click="removeMirror(idx)">删除</el-button>
            </div>
            <div class="add-mirror">
              <el-input v-model="newMirror" placeholder="添加新的镜像加速地址" style="width: 400px" />
              <el-button type="primary" @click="addMirror">添加</el-button>
            </div>
          </div>
        </el-form-item>
        
        <el-form-item>
          <el-button type="primary" @click="saveConfig" :loading="saving">保存配置</el-button>
          <el-button @click="restartDocker" :loading="restarting">重启 Docker</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="settings-section">
      <h3 class="section-title">常用镜像加速源</h3>
      <div class="mirror-recommendations">
        <div 
          v-for="mirror in mirrors" 
          :key="mirror.name"
          class="recommend-item"
          @click="useMirror(mirror)"
        >
          <div class="recommend-name">{{ mirror.name }}</div>
          <div class="recommend-url">{{ mirror.url }}</div>
          <div class="recommend-desc">{{ mirror.description }}</div>
        </div>
      </div>
    </div>

    <div class="settings-section">
      <h3 class="section-title">当前配置预览</h3>
      <pre class="config-preview">{{ JSON.stringify(config, null, 2) }}</pre>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import axios from 'axios'

const config = ref({
  registry_mirrors: []
})
const mirrors = ref([])
const newMirror = ref('')
const saving = ref(false)
const restarting = ref(false)

const loadConfig = async () => {
  try {
    const res = await axios.get('/api/docker/config')
    config.value = res.data.data || { registry_mirrors: [] }
    mirrors.value = res.data.mirrors || []
  } catch (err) {
    ElMessage.error('加载配置失败: ' + err.message)
  }
}

const addMirror = () => {
  if (!newMirror.value) {
    ElMessage.warning('请输入镜像加速地址')
    return
  }
  if (!config.value.registry_mirrors) {
    config.value.registry_mirrors = []
  }
  config.value.registry_mirrors.push(newMirror.value)
  newMirror.value = ''
}

const removeMirror = (idx) => {
  config.value.registry_mirrors.splice(idx, 1)
}

const useMirror = (mirror) => {
  if (!config.value.registry_mirrors) {
    config.value.registry_mirrors = []
  }
  if (!config.value.registry_mirrors.includes(mirror.url)) {
    config.value.registry_mirrors.push(mirror.url)
    ElMessage.success(`已添加 ${mirror.name} 镜像加速`)
  } else {
    ElMessage.warning('该镜像加速已存在')
  }
}

const saveConfig = async () => {
  saving.value = true
  try {
    await axios.put('/api/docker/config', config.value)
    ElMessage.success('配置已保存，请点击"重启 Docker"使配置生效')
  } catch (err) {
    ElMessage.error('保存失败: ' + err.message)
  } finally {
    saving.value = false
  }
}

const restartDocker = async () => {
  await ElMessageBox.confirm('重启 Docker 会导致所有容器暂时停止，确定继续吗？', '警告', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  })
  
  restarting.value = true
  try {
    await axios.post('/api/docker/restart')
    ElMessage.success('Docker 正在重启，请稍后...')
    setTimeout(() => {
      ElMessage.info('如果长时间未恢复，请手动检查 Docker 状态')
    }, 10000)
  } catch (err) {
    ElMessage.error('重启失败: ' + err.message)
  } finally {
    restarting.value = false
  }
}

onMounted(() => {
  loadConfig()
})
</script>

<style scoped>
.docker-config {
  padding: 0;
}

.settings-section {
  margin-bottom: 32px;
  padding-bottom: 24px;
  border-bottom: 1px solid #e5e7eb;
}

.settings-section:last-child {
  border-bottom: none;
  margin-bottom: 0;
  padding-bottom: 0;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: #1f2937;
  margin-bottom: 16px;
  padding-left: 8px;
  border-left: 3px solid #477779;
}

.form-tip {
  font-size: 12px;
  color: #9ca3af;
  margin-bottom: 16px;
}

.mirror-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.mirror-item {
  display: flex;
  gap: 12px;
  align-items: center;
}

.add-mirror {
  display: flex;
  gap: 12px;
  margin-top: 8px;
}

.mirror-recommendations {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.recommend-item {
  padding: 12px;
  border: 1px solid #e5e7eb;
  cursor: pointer;
  transition: all 0.2s;
}

.recommend-item:hover {
  border-color: #477779;
  background-color: #f9fafb;
}

.recommend-name {
  font-size: 14px;
  font-weight: 600;
  color: #1f2937;
}

.recommend-url {
  font-size: 12px;
  color: #477779;
  font-family: monospace;
  margin-top: 4px;
}

.recommend-desc {
  font-size: 11px;
  color: #9ca3af;
  margin-top: 4px;
}

.config-preview {
  background-color: #f5f5f5;
  padding: 12px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 12px;
  overflow-x: auto;
}
</style>
