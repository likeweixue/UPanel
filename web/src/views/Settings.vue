<template>
  <div>
    <h1 class="text-2xl font-bold text-gray-800 mb-6">面板设置</h1>
    
    <el-tabs v-model="activeTab">
      <!-- 个人设置 -->
      <el-tab-pane label="个人设置" name="profile">
        <div class="bg-white rounded-lg shadow-sm p-6">
          <h2 class="text-lg font-semibold mb-4">修改密码</h2>
          <el-form :model="passwordForm" label-width="100px" style="max-width: 400px">
            <el-form-item label="当前密码">
              <el-input v-model="passwordForm.old_password" type="password" />
            </el-form-item>
            <el-form-item label="新密码">
              <el-input v-model="passwordForm.new_password" type="password" />
            </el-form-item>
            <el-form-item label="确认新密码">
              <el-input v-model="passwordForm.confirm_password" type="password" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="changePassword" :loading="changingPwd">
                修改密码
              </el-button>
            </el-form-item>
          </el-form>
        </div>
      </el-tab-pane>
      
      <!-- 系统设置 -->
      <el-tab-pane label="系统设置" name="system">
        <div class="bg-white rounded-lg shadow-sm p-6">
          <h2 class="text-lg font-semibold mb-4">Docker 镜像加速</h2>
          <el-form label-width="120px">
            <el-form-item label="镜像加速器">
              <el-input 
                v-model="settings.registry_mirrors_str" 
                placeholder="多个地址用逗号分隔"
              />
              <div class="text-xs text-gray-400 mt-1">
                例如: https://docker.mirrors.ustc.edu.cn,https://hub-mirror.c.163.com
              </div>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="saveRegistryMirrors">
                保存并重启 Docker
              </el-button>
            </el-form-item>
          </el-form>
          
          <el-divider />
          
          <h2 class="text-lg font-semibold mb-4 mt-6">路径设置</h2>
          <el-form label-width="120px">
            <el-form-item label="网站目录">
              <el-input v-model="settings.website_path" />
            </el-form-item>
            <el-form-item label="数据库目录">
              <el-input v-model="settings.database_path" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="savePaths">保存</el-button>
            </el-form-item>
          </el-form>
        </div>
      </el-tab-pane>
      
      <!-- 关于 -->
      <el-tab-pane label="关于" name="about">
        <div class="bg-white rounded-lg shadow-sm p-6 text-center">
          <h1 class="text-3xl font-bold text-[#477779]">UPanel</h1>
          <p class="text-gray-500 mt-2">轻量级容器管理面板</p>
          <p class="text-gray-400 text-sm mt-4">版本: v0.1.0</p>
          <p class="text-gray-400 text-sm">基于 Go + Vue 3 + Docker</p>
          <el-divider />
          <p class="text-gray-400 text-sm">
            © 2024 UPanel. All rights reserved.
          </p>
        </div>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'

const activeTab = ref('profile')
const changingPwd = ref(false)

const passwordForm = reactive({
  old_password: '',
  new_password: '',
  confirm_password: ''
})

const settings = reactive({
  registry_mirrors_str: '',
  website_path: '',
  database_path: ''
})

const fetchSettings = async () => {
  try {
    const res = await axios.get('/api/settings')
    const data = res.data.data
    settings.registry_mirrors_str = (data.registry_mirrors || []).join(',')
    settings.website_path = data.website_path
    settings.database_path = data.database_path
  } catch (err) {
    console.error('获取设置失败:', err)
  }
}

const changePassword = async () => {
  if (passwordForm.new_password !== passwordForm.confirm_password) {
    ElMessage.warning('两次输入的密码不一致')
    return
  }
  
  changingPwd.value = true
  try {
    await axios.post('/api/auth/change-password', {
      old_password: passwordForm.old_password,
      new_password: passwordForm.new_password
    })
    ElMessage.success('密码修改成功')
    passwordForm.old_password = ''
    passwordForm.new_password = ''
    passwordForm.confirm_password = ''
  } catch (err) {
    ElMessage.error(err.response?.data?.error || '修改失败')
  } finally {
    changingPwd.value = false
  }
}

const saveRegistryMirrors = async () => {
  const mirrors = settings.registry_mirrors_str.split(',').filter(m => m.trim())
  try {
    await axios.put('/api/settings', { registry_mirrors: mirrors })
    ElMessage.success('设置已保存，请手动重启 Docker 生效')
  } catch (err) {
    ElMessage.error('保存失败')
  }
}

const savePaths = async () => {
  try {
    await axios.put('/api/settings', {
      website_path: settings.website_path,
      database_path: settings.database_path
    })
    ElMessage.success('设置已保存')
  } catch (err) {
    ElMessage.error('保存失败')
  }
}

onMounted(() => {
  fetchSettings()
})
</script>
