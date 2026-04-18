<template>
  <div class="email-notify-container">
    <div class="top-panel">
      <div class="top-left">
        <el-button type="primary" @click="saveSettings" :loading="saving">
          <el-icon><Check /></el-icon> 保存设置
        </el-button>
        <el-button @click="testSend">
          <el-icon><Message /></el-icon> 测试发送
        </el-button>
      </div>
      <div class="top-right">
        <el-switch v-model="settings.enabled" active-text="启用邮件通知" />
      </div>
    </div>

    <div class="content-panel">
      <div class="settings-section">
        <h3 class="section-title">SMTP 服务器配置</h3>
        <el-form :model="settings" label-width="120px" class="settings-form">
          <el-form-item label="SMTP 服务器" required>
            <el-input v-model="settings.smtpHost" placeholder="smtp.qq.com" />
          </el-form-item>
          <el-form-item label="端口" required>
            <el-input-number v-model="settings.smtpPort" :min="1" :max="65535" />
            <span class="form-hint"> 465 (SSL) 或 587 (TLS)</span>
          </el-form-item>
          <el-form-item label="加密方式">
            <el-select v-model="settings.encryption">
              <el-option label="SSL/TLS" value="ssl" />
              <el-option label="STARTTLS" value="tls" />
              <el-option label="无" value="none" />
            </el-select>
          </el-form-item>
          <el-form-item label="发件人邮箱" required>
            <el-input v-model="settings.fromEmail" placeholder="your-email@example.com" />
          </el-form-item>
          <el-form-item label="授权码/密码" required>
            <el-input v-model="settings.password" type="password" show-password placeholder="邮箱授权码或密码" />
          </el-form-item>
        </el-form>
      </div>

      <div class="settings-section">
        <h3 class="section-title">收件人配置</h3>
        <el-form label-width="120px" class="settings-form">
          <el-form-item label="接收邮箱">
            <el-input v-model="settings.toEmail" placeholder="admin@example.com" />
            <div class="form-tip">多个邮箱用逗号分隔</div>
          </el-form-item>
        </el-form>
      </div>

      <div class="settings-section">
        <h3 class="section-title">通知设置</h3>
        <el-form label-width="140px" class="settings-form">
          <el-form-item label="系统异常通知">
            <el-switch v-model="settings.notifyError" />
          </el-form-item>
          <el-form-item label="容器状态变更">
            <el-switch v-model="settings.notifyContainer" />
          </el-form-item>
          <el-form-item label="网站状态变更">
            <el-switch v-model="settings.notifyWebsite" />
          </el-form-item>
          <el-form-item label="数据库状态变更">
            <el-switch v-model="settings.notifyDatabase" />
          </el-form-item>
          <el-form-item label="SSL 证书过期">
            <el-switch v-model="settings.notifySSLCert" />
            <span class="form-hint" v-if="settings.notifySSLCert"> 提前30天通知</span>
          </el-form-item>
          <el-form-item label="备份完成通知">
            <el-switch v-model="settings.notifyBackup" />
          </el-form-item>
        </el-form>
      </div>

      <div class="settings-section">
        <h3 class="section-title">通知记录</h3>
        <el-table :data="notifyLogs" stripe style="width: 100%">
          <el-table-column prop="time" label="时间" width="180" />
          <el-table-column prop="type" label="类型" width="100" />
          <el-table-column prop="title" label="标题" min-width="200" />
          <el-table-column prop="status" label="状态" width="80">
            <template #default="{ row }">
              <el-tag :type="row.status === '成功' ? 'success' : 'danger'" size="small">
                {{ row.status }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="80">
            <template #default="{ row }">
              <el-button size="small" text @click="viewLogDetail(row)">详情</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <!-- 测试发送对话框 -->
    <el-dialog v-model="testDialogVisible" title="测试邮件" width="400px">
      <el-form label-width="80px">
        <el-form-item label="收件人">
          <el-input v-model="testEmail" placeholder="test@example.com" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="testDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="sendTest" :loading="sending">发送测试</el-button>
      </template>
    </el-dialog>

    <!-- 日志详情对话框 -->
    <el-dialog v-model="logDetailVisible" title="通知详情" width="500px">
      <pre class="log-detail">{{ currentLogDetail }}</pre>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Check, Message } from '@element-plus/icons-vue'

const saving = ref(false)
const sending = ref(false)
const testDialogVisible = ref(false)
const logDetailVisible = ref(false)
const testEmail = ref('')
const currentLogDetail = ref('')

const settings = reactive({
  enabled: false,
  smtpHost: 'smtp.qq.com',
  smtpPort: 465,
  encryption: 'ssl',
  fromEmail: '',
  password: '',
  toEmail: '',
  notifyError: true,
  notifyContainer: true,
  notifyWebsite: true,
  notifyDatabase: true,
  notifySSLCert: true,
  notifyBackup: true
})

const notifyLogs = ref([
  { time: '2024-01-15 10:30:00', type: '系统', title: '面板启动成功', status: '成功' },
  { time: '2024-01-15 09:00:00', type: '容器', title: 'Nginx 容器已启动', status: '成功' }
])

const saveSettings = async () => {
  if (!settings.smtpHost || !settings.fromEmail) {
    ElMessage.warning('请填写完整的 SMTP 配置')
    return
  }
  saving.value = true
  try {
    await new Promise(resolve => setTimeout(resolve, 1000))
    localStorage.setItem('emailSettings', JSON.stringify(settings))
    ElMessage.success('设置已保存')
  } catch (err) {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

const testSend = () => {
  testEmail.value = settings.toEmail || ''
  testDialogVisible.value = true
}

const sendTest = async () => {
  if (!testEmail.value) {
    ElMessage.warning('请输入收件人邮箱')
    return
  }
  sending.value = true
  try {
    await new Promise(resolve => setTimeout(resolve, 1500))
    ElMessage.success('测试邮件已发送，请查收')
    testDialogVisible.value = false
    notifyLogs.value.unshift({
      time: new Date().toLocaleString(),
      type: '测试',
      title: `测试邮件发送至 ${testEmail.value}`,
      status: '成功'
    })
  } catch (err) {
    ElMessage.error('发送失败: ' + err.message)
  } finally {
    sending.value = false
  }
}

const viewLogDetail = (log) => {
  currentLogDetail.value = JSON.stringify(log, null, 2)
  logDetailVisible.value = true
}

onMounted(() => {
  const saved = localStorage.getItem('emailSettings')
  if (saved) {
    try {
      const data = JSON.parse(saved)
      Object.assign(settings, data)
    } catch (e) {}
  }
})
</script>

<style scoped>
.email-notify-container {
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
  flex-wrap: wrap;
}

.top-left {
  display: flex;
  gap: 8px;
}

.top-right {
  flex-shrink: 0;
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

.form-hint {
  font-size: 12px;
  color: #9ca3af;
  margin-left: 8px;
}

.log-detail {
  background-color: #f5f5f5;
  padding: 12px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 12px;
  white-space: pre-wrap;
}
</style>
