<template>
  <div class="settings-container">
    <!-- 顶部分类面板 -->
    <div class="top-panel">
      <div class="top-left">
        <el-button type="primary" @click="saveAllSettings" :loading="saving">
          <el-icon><Check /></el-icon> 保存设置
        </el-button>
        <el-button @click="refreshSettings">
          <el-icon><Refresh /></el-icon> 刷新
        </el-button>
      </div>
      <div class="top-right">
        <el-tabs v-model="activeTab" class="settings-tabs">
          <el-tab-pane name="profile">
            <template #label>
              <span><el-icon><User /></el-icon> 个人设置</span>
            </template>
          </el-tab-pane>
          <el-tab-pane name="system">
            <template #label>
              <span><el-icon><Setting /></el-icon> 系统设置</span>
            </template>
          </el-tab-pane>
          <el-tab-pane name="security">
            <template #label>
              <span><el-icon><Lock /></el-icon> 安全设置</span>
            </template>
          </el-tab-pane>
          <el-tab-pane name="network">
            <template #label>
              <span><el-icon><Connection /></el-icon> 网络设置</span>
            </template>
          </el-tab-pane>
          <el-tab-pane name="backup">
            <template #label>
              <span><el-icon><FolderOpened /></el-icon> 备份还原</span>
            </template>
          </el-tab-pane>
          <el-tab-pane name="about">
            <template #label>
              <span><el-icon><InfoFilled /></el-icon> 关于</span>
            </template>
          </el-tab-pane>
        </el-tabs>
      </div>
    </div>

    <!-- 个人设置 -->
    <div class="content-panel" v-if="activeTab === 'profile'">
      <div class="settings-section">
        <h3 class="section-title">账号信息</h3>
        <el-form :model="profileForm" label-width="120px" class="settings-form">
          <el-form-item label="用户名">
            <el-input v-model="profileForm.username" disabled />
          </el-form-item>
          <el-form-item label="邮箱">
            <el-input v-model="profileForm.email" placeholder="admin@example.com" />
          </el-form-item>
          <el-form-item label="手机号">
            <el-input v-model="profileForm.phone" placeholder="13800138000" />
          </el-form-item>
        </el-form>
      </div>

      <div class="settings-section">
        <h3 class="section-title">修改密码</h3>
        <el-form :model="passwordForm" :rules="passwordRules" ref="passwordFormRef" label-width="120px" class="settings-form">
          <el-form-item label="当前密码" prop="oldPassword">
            <el-input v-model="passwordForm.oldPassword" type="password" placeholder="请输入当前密码" show-password />
          </el-form-item>
          <el-form-item label="新密码" prop="newPassword">
            <el-input v-model="passwordForm.newPassword" type="password" placeholder="请输入新密码" show-password />
          </el-form-item>
          <el-form-item label="确认新密码" prop="confirmPassword">
            <el-input v-model="passwordForm.confirmPassword" type="password" placeholder="请再次输入新密码" show-password />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="changePassword" :loading="changingPassword">修改密码</el-button>
          </el-form-item>
        </el-form>
      </div>

      <div class="settings-section">
        <h3 class="section-title">界面设置</h3>
        <el-form label-width="120px" class="settings-form">
          <el-form-item label="主题颜色">
            <el-select v-model="uiSettings.theme">
              <el-option label="浅色" value="light" />
              <el-option label="深色" value="dark" />
              <el-option label="跟随系统" value="auto" />
            </el-select>
          </el-form-item>
          <el-form-item label="语言">
            <el-select v-model="uiSettings.language">
              <el-option label="简体中文" value="zh-CN" />
              <el-option label="English" value="en-US" />
            </el-select>
          </el-form-item>
        </el-form>
      </div>
    </div>

    <!-- 系统设置 -->
    <div class="content-panel" v-if="activeTab === 'system'">
      <div class="settings-section">
        <h3 class="section-title">基础设置</h3>
        <el-form :model="systemForm" label-width="150px" class="settings-form">
          <el-form-item label="面板名称">
            <el-input v-model="systemForm.panelName" placeholder="UPanel" />
          </el-form-item>
          <el-form-item label="面板端口">
            <el-input-number v-model="systemForm.port" :min="1" :max="65535" />
            <div class="form-tip">修改后需要重启面板生效</div>
          </el-form-item>
          <el-form-item label="网站根目录">
            <el-input v-model="systemForm.websitePath" />
          </el-form-item>
          <el-form-item label="备份目录">
            <el-input v-model="systemForm.backupPath" />
          </el-form-item>
          <el-form-item label="日志保留天数">
            <el-input-number v-model="systemForm.logRetention" :min="1" :max="365" /> 天
          </el-form-item>
        </el-form>
      </div>

      <div class="settings-section">
        <h3 class="section-title">Docker 设置</h3>
        <el-form label-width="150px" class="settings-form">
          <el-form-item label="Docker 镜像加速">
            <el-input v-model="dockerSettings.registryMirror" placeholder="https://docker.mirrors.ustc.edu.cn" />
          </el-form-item>
          <el-form-item label="Docker 数据目录">
            <el-input v-model="dockerSettings.dataRoot" placeholder="/var/lib/docker" />
          </el-form-item>
        </el-form>
      </div>
    </div>

    <!-- 安全设置 -->
    <div class="content-panel" v-if="activeTab === 'security'">
      <div class="settings-section">
        <h3 class="section-title">登录安全</h3>
        <el-form label-width="150px" class="settings-form">
          <el-form-item label="登录失败限制">
            <el-switch v-model="securityForm.loginLimit" />
          </el-form-item>
          <el-form-item label="最大失败次数" v-if="securityForm.loginLimit">
            <el-input-number v-model="securityForm.maxAttempts" :min="1" :max="20" /> 次
          </el-form-item>
          <el-form-item label="锁定时长" v-if="securityForm.loginLimit">
            <el-input-number v-model="securityForm.lockMinutes" :min="1" :max="60" /> 分钟
          </el-form-item>
          <el-form-item label="会话超时">
            <el-input-number v-model="securityForm.sessionTimeout" :min="5" :max="1440" /> 分钟
          </el-form-item>
        </el-form>
      </div>

      <div class="settings-section">
        <h3 class="section-title">API 安全</h3>
        <el-form label-width="150px" class="settings-form">
          <el-form-item label="JWT 有效期">
            <el-input-number v-model="securityForm.jwtExpiry" :min="1" :max="720" /> 小时
          </el-form-item>
        </el-form>
      </div>
    </div>

    <!-- 网络设置 -->
    <div class="content-panel" v-if="activeTab === 'network'">
      <div class="settings-section">
        <h3 class="section-title">代理设置</h3>
        <el-form label-width="120px" class="settings-form">
          <el-form-item label="HTTP 代理">
            <el-input v-model="networkForm.httpProxy" placeholder="http://127.0.0.1:7890" />
          </el-form-item>
          <el-form-item label="HTTPS 代理">
            <el-input v-model="networkForm.httpsProxy" placeholder="http://127.0.0.1:7890" />
          </el-form-item>
        </el-form>
      </div>

      <div class="settings-section">
        <h3 class="section-title">DNS 设置</h3>
        <el-form label-width="120px" class="settings-form">
          <el-form-item label="首选 DNS">
            <el-input v-model="networkForm.dns1" placeholder="8.8.8.8" />
          </el-form-item>
          <el-form-item label="备用 DNS">
            <el-input v-model="networkForm.dns2" placeholder="114.114.114.114" />
          </el-form-item>
        </el-form>
      </div>
    </div>

    <!-- 备份还原 -->
    <div class="content-panel" v-if="activeTab === 'backup'">
      <div class="settings-section">
        <h3 class="section-title">数据备份</h3>
        <el-form label-width="120px" class="settings-form">
          <el-form-item label="自动备份">
            <el-switch v-model="backupForm.autoBackup" />
          </el-form-item>
          <el-form-item label="备份周期" v-if="backupForm.autoBackup">
            <el-select v-model="backupForm.backupSchedule">
              <el-option label="每天" value="daily" />
              <el-option label="每周" value="weekly" />
              <el-option label="每月" value="monthly" />
            </el-select>
          </el-form-item>
          <el-form-item label="保留份数">
            <el-input-number v-model="backupForm.retention" :min="1" :max="30" /> 份
          </el-form-item>
          <el-form-item>
            <el-button @click="manualBackup" :loading="backingUp">立即备份</el-button>
          </el-form-item>
        </el-form>
      </div>

      <div class="settings-section">
        <h3 class="section-title">备份列表</h3>
        <el-table :data="backupList" stripe style="width: 100%">
          <el-table-column prop="name" label="备份名称" min-width="200" />
          <el-table-column prop="size" label="大小" width="100" />
          <el-table-column prop="time" label="创建时间" width="180" />
          <el-table-column label="操作" width="150">
            <template #default="{ row }">
              <el-button size="small" @click="downloadBackup(row)">下载</el-button>
              <el-button size="small" type="danger" @click="deleteBackup(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <!-- 关于 -->
    <div class="content-panel" v-if="activeTab === 'about'">
      <div class="about-section">
        <div class="about-logo">
          <h1 class="logo">UPanel</h1>
          <p class="slogan">轻量级容器管理面板</p>
        </div>
        
        <div class="about-info">
          <div class="info-row">
            <span class="label">版本</span>
            <span class="value">v0.1.0</span>
          </div>
          <div class="info-row">
            <span class="label">Go 版本</span>
            <span class="value">1.21</span>
          </div>
          <div class="info-row">
            <span class="label">Vue 版本</span>
            <span class="value">3.4</span>
          </div>
        </div>

        <div class="about-links">
          <a href="https://github.com/likeweixue/UPanel" target="_blank">
            GitHub
          </a>
          <a href="#" @click.prevent="checkUpdate">
            检查更新
          </a>
        </div>

        <div class="about-copyright">
          <p>© 2024 UPanel. 开源容器面板</p>
          <p>基于 MIT 协议开源</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Check, Refresh, User, Setting, Lock, Connection, FolderOpened, InfoFilled } from '@element-plus/icons-vue'

const activeTab = ref('profile')
const saving = ref(false)
const changingPassword = ref(false)
const backingUp = ref(false)
const passwordFormRef = ref(null)

const profileForm = reactive({
  username: 'admin',
  email: '',
  phone: ''
})

const passwordForm = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: ''
})

const passwordRules = {
  oldPassword: [{ required: true, message: '请输入当前密码', trigger: 'blur' }],
  newPassword: [{ required: true, message: '请输入新密码', trigger: 'blur', min: 6 }],
  confirmPassword: [
    { required: true, message: '请确认新密码', trigger: 'blur' },
    {
      validator: (rule, value, callback) => {
        if (value !== passwordForm.newPassword) {
          callback(new Error('两次输入的密码不一致'))
        } else {
          callback()
        }
      },
      trigger: 'blur'
    }
  ]
}

const uiSettings = reactive({
  theme: 'light',
  language: 'zh-CN'
})

const systemForm = reactive({
  panelName: 'UPanel',
  port: 8080,
  websitePath: '/www/wwwroot',
  backupPath: '/backup',
  logRetention: 30
})

const dockerSettings = reactive({
  registryMirror: '',
  dataRoot: '/var/lib/docker'
})

const securityForm = reactive({
  loginLimit: true,
  maxAttempts: 5,
  lockMinutes: 15,
  sessionTimeout: 60,
  jwtExpiry: 24
})

const networkForm = reactive({
  httpProxy: '',
  httpsProxy: '',
  dns1: '8.8.8.8',
  dns2: '114.114.114.114'
})

const backupForm = reactive({
  autoBackup: false,
  backupSchedule: 'daily',
  retention: 7
})

const backupList = ref([
  { name: 'upanel_backup_20240115.tar.gz', size: '156 MB', time: '2024-01-15 02:00:01' },
  { name: 'upanel_backup_20240108.tar.gz', size: '152 MB', time: '2024-01-08 02:00:01' }
])

const saveAllSettings = async () => {
  saving.value = true
  try {
    await new Promise(resolve => setTimeout(resolve, 1000))
    ElMessage.success('设置已保存')
  } catch (err) {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

const refreshSettings = () => {
  ElMessage.success('刷新成功')
}

const changePassword = async () => {
  if (!passwordFormRef.value) return
  
  await passwordFormRef.value.validate(async (valid) => {
    if (!valid) return
    
    changingPassword.value = true
    try {
      await new Promise(resolve => setTimeout(resolve, 1000))
      ElMessage.success('密码修改成功，请重新登录')
      passwordForm.oldPassword = ''
      passwordForm.newPassword = ''
      passwordForm.confirmPassword = ''
    } catch (err) {
      ElMessage.error('修改失败')
    } finally {
      changingPassword.value = false
    }
  })
}

const manualBackup = async () => {
  backingUp.value = true
  try {
    await new Promise(resolve => setTimeout(resolve, 2000))
    backupList.value.unshift({
      name: `upanel_backup_${new Date().toISOString().slice(0, 10)}.tar.gz`,
      size: '158 MB',
      time: new Date().toLocaleString()
    })
    ElMessage.success('备份创建成功')
  } catch (err) {
    ElMessage.error('备份失败')
  } finally {
    backingUp.value = false
  }
}

const downloadBackup = (row) => {
  ElMessage.info(`下载 ${row.name}`)
}

const deleteBackup = async (row) => {
  await ElMessageBox.confirm(`确定删除备份 ${row.name} 吗？`, '警告', { type: 'warning' })
  const index = backupList.value.findIndex(b => b.name === row.name)
  if (index !== -1) {
    backupList.value.splice(index, 1)
    ElMessage.success('备份已删除')
  }
}

const checkUpdate = () => {
  ElMessage.info('当前已是最新版本')
}
</script>

<style scoped>
.settings-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.top-panel {
  background: #ffffff;
  border-radius: 4px;
  padding: 12px 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.top-left {
  display: flex;
  gap: 8px;
}

.top-right {
  flex-shrink: 0;
}

.settings-tabs {
  margin-right: 16px;
}

.content-panel {
  background: #ffffff;
  border-radius: 4px;
  padding: 20px;
  flex: 1;
  overflow-y: auto;
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
  margin-bottom: 20px;
  padding-left: 8px;
  border-left: 3px solid #477779;
}

.settings-form {
  max-width: 500px;
}

.form-tip {
  font-size: 11px;
  color: #9ca3af;
  margin-top: 4px;
}

/* 关于页面 */
.about-section {
  text-align: center;
  padding: 40px 20px;
}

.about-logo {
  margin-bottom: 32px;
}

.logo {
  font-size: 48px;
  font-weight: bold;
  color: #477779;
  margin: 0;
}

.slogan {
  font-size: 14px;
  color: #9ca3af;
  margin-top: 8px;
}

.about-info {
  display: inline-block;
  text-align: left;
  background-color: #f9fafb;
  border-radius: 8px;
  padding: 20px 40px;
  margin-bottom: 24px;
}

.info-row {
  display: flex;
  gap: 40px;
  padding: 8px 0;
}

.info-row .label {
  color: #9ca3af;
  width: 80px;
}

.info-row .value {
  color: #1f2937;
  font-weight: 500;
}

.about-links {
  display: flex;
  justify-content: center;
  gap: 24px;
  margin-bottom: 32px;
}

.about-links a {
  color: #477779;
  text-decoration: none;
  font-size: 14px;
}

.about-links a:hover {
  text-decoration: underline;
}

.about-copyright {
  font-size: 12px;
  color: #9ca3af;
}
</style>
