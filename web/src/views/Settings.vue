<template>
  <div class="settings-container">
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
            <template #label><span><el-icon><User /></el-icon> 个人设置</span></template>
          </el-tab-pane>
          <el-tab-pane name="system">
            <template #label><span><el-icon><Setting /></el-icon> 系统设置</span></template>
          </el-tab-pane>
          <el-tab-pane name="docker">
            <template #label><span><el-icon><Box /></el-icon> Docker设置</span></template>
          </el-tab-pane>
          <el-tab-pane name="security">
            <template #label><span><el-icon><Lock /></el-icon> 安全设置</span></template>
          </el-tab-pane>
          <el-tab-pane name="network">
            <template #label><span><el-icon><Connection /></el-icon> 网络设置</span></template>
          </el-tab-pane>
          <el-tab-pane name="backup">
            <template #label><span><el-icon><FolderOpened /></el-icon> 备份还原</span></template>
          </el-tab-pane>
          <el-tab-pane name="about">
            <template #label><span><el-icon><InfoFilled /></el-icon> 关于</span></template>
          </el-tab-pane>
        </el-tabs>
      </div>
    </div>

    <!-- 个人设置 -->
    <div class="content-panel" v-if="activeTab === 'profile'">
      <div class="settings-section">
        <h3 class="section-title">账号信息</h3>
        <el-form :model="profileForm" label-width="120px" class="settings-form">
          <el-form-item label="用户名"><el-input v-model="profileForm.username" disabled /></el-form-item>
          <el-form-item label="邮箱"><el-input v-model="profileForm.email" placeholder="admin@example.com" /></el-form-item>
          <el-form-item label="手机号"><el-input v-model="profileForm.phone" placeholder="13800138000" /></el-form-item>
        </el-form>
      </div>

      <div class="settings-section">
        <h3 class="section-title">修改密码</h3>
        <el-form :model="passwordForm" :rules="passwordRules" ref="passwordFormRef" label-width="120px" class="settings-form">
          <el-form-item label="当前密码" prop="oldPassword"><el-input v-model="passwordForm.oldPassword" type="password" show-password /></el-form-item>
          <el-form-item label="新密码" prop="newPassword"><el-input v-model="passwordForm.newPassword" type="password" show-password /></el-form-item>
          <el-form-item label="确认新密码" prop="confirmPassword"><el-input v-model="passwordForm.confirmPassword" type="password" show-password /></el-form-item>
          <el-form-item><el-button type="primary" @click="changePassword" :loading="changingPassword">修改密码</el-button></el-form-item>
        </el-form>
      </div>

      <div class="settings-section">
        <h3 class="section-title">界面设置</h3>
        <el-form label-width="120px" class="settings-form">
          <el-form-item label="主题配色">
            <el-select v-model="uiSettings.theme" @change="changeTheme">
              <el-option label="浅色" value="light" />
              <el-option label="暗色" value="dark" />
              <el-option label="黑金" value="gold" />
              <el-option label="深海蓝" value="blue" />
              <el-option label="森林绿" value="green" />
            </el-select>
          </el-form-item>
          <el-form-item label="语言">
            <el-select v-model="uiSettings.language">
              <el-option label="简体中文" value="zh-CN" />
              <el-option label="English" value="en-US" />
            </el-select>
          </el-form-item>
          <el-form-item label="全局透明度">
            <el-slider v-model="uiSettings.opacity" :min="0" :max="100" :format-tooltip="formatOpacity" @change="changeOpacity" />
            <div class="form-tip">调整面板整体透明度，数值越大越不透明（100%为完全不透明）</div>
          </el-form-item>
        </el-form>
      </div>

      <div class="settings-section">
        <h3 class="section-title">全局 CSS</h3>
        <el-form label-width="120px" class="settings-form">
          <el-form-item label="自定义样式">
            <el-input v-model="uiSettings.customCSS" type="textarea" :rows="12" placeholder="/* 自定义 CSS 样式 */" />
            <div class="form-tip">在这里编写自定义 CSS 样式，会实时生效</div>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="applyCustomCSS" :loading="applyingCSS">应用样式</el-button>
            <el-button @click="resetCustomCSS">重置</el-button>
          </el-form-item>
        </el-form>
      </div>
    </div>

    <!-- 系统设置 -->
    <div class="content-panel" v-if="activeTab === 'system'">
      <div class="settings-section">
        <h3 class="section-title">基础设置</h3>
        <el-form :model="systemForm" label-width="150px" class="settings-form">
          <el-form-item label="面板名称"><el-input v-model="systemForm.panelName" /></el-form-item>
          <el-form-item label="面板端口"><el-input-number v-model="systemForm.port" :min="1" :max="65535" /></el-form-item>
        </el-form>
      </div>
    </div>

    <!-- Docker 设置 -->
    <div class="content-panel" v-if="activeTab === 'docker'">
      <DockerConfig />
    </div>

    <!-- 安全设置 -->
    <div class="content-panel" v-if="activeTab === 'security'">
      <div class="settings-section">
        <h3 class="section-title">登录安全</h3>
        <el-form label-width="150px" class="settings-form">
          <el-form-item label="登录失败限制"><el-switch v-model="securityForm.loginLimit" /></el-form-item>
          <el-form-item label="最大失败次数" v-if="securityForm.loginLimit"><el-input-number v-model="securityForm.maxAttempts" :min="1" :max="20" /></el-form-item>
        </el-form>
      </div>
    </div>

    <!-- 网络设置 -->
    <div class="content-panel" v-if="activeTab === 'network'">
      <div class="settings-section">
        <h3 class="section-title">代理设置</h3>
        <el-form label-width="120px" class="settings-form">
          <el-form-item label="HTTP 代理"><el-input v-model="networkForm.httpProxy" placeholder="http://127.0.0.1:7890" /></el-form-item>
        </el-form>
      </div>
    </div>

    <!-- 备份还原 -->
    <div class="content-panel" v-if="activeTab === 'backup'">
      <div class="settings-section">
        <h3 class="section-title">数据备份</h3>
        <el-form label-width="120px" class="settings-form">
          <el-form-item><el-button @click="manualBackup" :loading="backingUp">立即备份</el-button></el-form-item>
        </el-form>
      </div>
    </div>

    <!-- 关于 -->
    <div class="content-panel" v-if="activeTab === 'about'">
      <div class="about-section">
        <div class="about-logo"><h1 class="logo">UPanel</h1><p class="slogan">轻量级容器管理面板</p></div>
        <div class="about-info"><div class="info-row"><span class="label">版本</span><span class="value">v0.1.0</span></div></div>
        <div class="about-links"><a href="https://github.com/likeweixue/UPanel" target="_blank">GitHub</a></div>
        <div class="about-copyright"><p>© 2024 UPanel. 开源容器面板</p></div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Check, Refresh, User, Setting, Lock, Connection, FolderOpened, InfoFilled, Box } from '@element-plus/icons-vue'
import { applyTheme } from '@/themes'
import DockerConfig from './settings/DockerConfig.vue'

const activeTab = ref('profile')
const saving = ref(false)
const changingPassword = ref(false)
const backingUp = ref(false)
const applyingCSS = ref(false)
const passwordFormRef = ref(null)

const profileForm = reactive({ username: 'admin', email: '', phone: '' })
const passwordForm = reactive({ oldPassword: '', newPassword: '', confirmPassword: '' })
const uiSettings = reactive({ theme: 'light', language: 'zh-CN', customCSS: '', opacity: 85 })
const systemForm = reactive({ panelName: 'UPanel', port: 8080 })
const securityForm = reactive({ loginLimit: true, maxAttempts: 5 })
const networkForm = reactive({ httpProxy: '' })

const passwordRules = {
  oldPassword: [{ required: true, message: '请输入当前密码', trigger: 'blur' }],
  newPassword: [{ required: true, message: '请输入新密码', trigger: 'blur', min: 6 }],
  confirmPassword: [{ required: true, message: '请确认新密码', trigger: 'blur' }]
}

const formatOpacity = (value) => `${value}%`

const applyOpacity = (opacity) => {
  const value = opacity / 100
  const oldStyle = document.getElementById('opacity-style')
  if (oldStyle) oldStyle.remove()
  
  const style = document.createElement('style')
  style.id = 'opacity-style'
  style.textContent = `
    .el-card, .stats-panel, .gauges-panel, .chart-panel, .info-panel, .notification-panel,
    .apps-panel, .files-panel, .websites-panel, .databases-panel, .containers-panel,
    .content-panel, .categories-panel, .top-panel, .breadcrumb-panel,
    .process-card, .chart-card, .settings-section, .el-table__body tr,
    .app-card, .website-card {
      background-color: rgba(255, 255, 255, ${value}) !important;
    }
    body.dark-mode .el-card, body.dark-mode .stats-panel, body.dark-mode .gauges-panel,
    body.dark-mode .chart-panel, body.dark-mode .info-panel, body.dark-mode .notification-panel,
    body.dark-mode .apps-panel, body.dark-mode .files-panel, body.dark-mode .websites-panel,
    body.dark-mode .databases-panel, body.dark-mode .containers-panel,
    body.dark-mode .content-panel, body.dark-mode .categories-panel,
    body.dark-mode .top-panel, body.dark-mode .breadcrumb-panel {
      background-color: rgba(30, 30, 30, ${value}) !important;
    }
    .sidebar { background-color: rgba(255, 255, 255, ${Math.min(value + 0.05, 1)}) !important; }
    body.dark-mode .sidebar { background-color: rgba(30, 30, 30, ${Math.min(value + 0.05, 1)}) !important; }
  `
  document.head.appendChild(style)
  localStorage.setItem('panelOpacity', opacity)
}

const changeOpacity = (value) => {
  applyOpacity(value)
  ElMessage.success(`透明度已调整为 ${value}%`)
}

const loadOpacity = () => {
  const saved = localStorage.getItem('panelOpacity')
  if (saved) {
    uiSettings.opacity = parseInt(saved)
    applyOpacity(uiSettings.opacity)
  }
}

const loadCustomCSS = () => {
  const saved = localStorage.getItem('customCSS')
  if (saved) {
    uiSettings.customCSS = saved
    applyCustomCSS(true)
  }
}

const applyCustomCSS = async (silent = false) => {
  applyingCSS.value = true
  try {
    const oldStyle = document.getElementById('custom-css-style')
    if (oldStyle) oldStyle.remove()
    if (uiSettings.customCSS && uiSettings.customCSS.trim()) {
      const style = document.createElement('style')
      style.id = 'custom-css-style'
      style.textContent = uiSettings.customCSS
      document.head.appendChild(style)
      localStorage.setItem('customCSS', uiSettings.customCSS)
    } else {
      localStorage.removeItem('customCSS')
    }
    if (!silent) ElMessage.success('自定义样式已应用')
  } catch (err) {
    ElMessage.error('应用样式失败: ' + err.message)
  } finally {
    applyingCSS.value = false
  }
}

const resetCustomCSS = () => {
  uiSettings.customCSS = ''
  const oldStyle = document.getElementById('custom-css-style')
  if (oldStyle) oldStyle.remove()
  localStorage.removeItem('customCSS')
  ElMessage.success('已重置为默认样式')
}

const changeTheme = (theme) => {
  applyTheme(theme)
  ElMessage.success(`已切换到${theme === 'light' ? '浅色' : theme === 'dark' ? '暗色' : theme === 'gold' ? '黑金' : theme === 'blue' ? '深海蓝' : '森林绿'}主题`)
}

const saveAllSettings = async () => {
  saving.value = true
  await new Promise(resolve => setTimeout(resolve, 1000))
  ElMessage.success('设置已保存')
  saving.value = false
}

const refreshSettings = () => { ElMessage.success('刷新成功') }

const changePassword = async () => {
  if (!passwordFormRef.value) return
  await passwordFormRef.value.validate(async (valid) => {
    if (!valid) return
    changingPassword.value = true
    await new Promise(resolve => setTimeout(resolve, 1000))
    ElMessage.success('密码修改成功')
    changingPassword.value = false
  })
}

const manualBackup = async () => {
  backingUp.value = true
  await new Promise(resolve => setTimeout(resolve, 2000))
  ElMessage.success('备份创建成功')
  backingUp.value = false
}

onMounted(() => {
  loadOpacity()
  loadCustomCSS()
})
</script>

<style scoped>
.settings-container { height: 100%; display: flex; flex-direction: column; gap: 16px; }
.top-panel { background: #fff; padding: 12px 16px; display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; }
.top-left { display: flex; gap: 8px; }
.top-right { flex-shrink: 0; }
.settings-tabs { margin-right: 16px; }
.content-panel { background: #fff; padding: 20px; flex: 1; overflow-y: auto; }
.settings-section { margin-bottom: 32px; padding-bottom: 24px; border-bottom: 1px solid #e5e7eb; }
.settings-section:last-child { border-bottom: none; margin-bottom: 0; padding-bottom: 0; }
.section-title { font-size: 16px; font-weight: 600; color: #1f2937; margin-bottom: 20px; padding-left: 8px; border-left: 3px solid #477779; }
.settings-form { max-width: 100%; }
.form-tip { font-size: 11px; color: #9ca3af; margin-top: 4px; }
.about-section { text-align: center; padding: 40px 20px; }
.logo { font-size: 48px; font-weight: bold; color: #477779; margin: 0; }
.slogan { font-size: 14px; color: #9ca3af; margin-top: 8px; }
.about-info { display: inline-block; text-align: left; background-color: #f9fafb; padding: 20px 40px; margin-bottom: 24px; }
.info-row { display: flex; gap: 40px; padding: 8px 0; }
.info-row .label { color: #9ca3af; width: 80px; }
.info-row .value { color: #1f2937; font-weight: 500; }
.about-links { display: flex; justify-content: center; gap: 24px; margin-bottom: 32px; }
.about-links a { color: #477779; text-decoration: none; }
.about-copyright { font-size: 12px; color: #9ca3af; }
.el-slider { width: 300px; }
</style>
