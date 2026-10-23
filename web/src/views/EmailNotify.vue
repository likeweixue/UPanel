<template>
  <div class="page">
    <header class="page__header">
      <div>
        <div class="page__title-row">
          <h1 class="page__title">邮件通知</h1>
          <span :class="['badge', form.enabled ? 'badge--success' : '']">{{ form.enabled ? '已启用' : '未启用' }}</span>
        </div>
        <p class="page__subtitle">配置 SMTP，用于发送测试邮件与后续的通知消息</p>
      </div>
      <div class="page__actions">
        <UButton icon="refresh" label="刷新" :loading="loading" @click="load" />
        <UButton icon="mail" label="发送测试邮件" :loading="testing" @click="testEmail" />
        <UButton variant="primary" icon="save" label="保存配置" :loading="saving" @click="save" />
      </div>
    </header>

    <div v-if="error" class="alert alert--danger" role="alert">
      <UIcon name="alert" :size="16" class="alert__icon" />
      <div class="alert__body">{{ error }}</div>
    </div>

    <div v-if="message" class="alert" :class="messageType === 'success' ? 'alert--success' : 'alert--danger'" role="status">
      <UIcon :name="messageType === 'success' ? 'check-circle' : 'alert'" :size="16" class="alert__icon" />
      <div class="alert__body">{{ message }}</div>
    </div>

    <div v-if="loading && !loaded" class="loading-block"><span class="spinner" /> 正在读取邮件配置…</div>

    <div v-else class="widget-grid">
      <UWidget title="SMTP 服务器" subtitle="邮件发送通道" icon="mail" :span="7">
        <form id="email-form" class="stack" @submit.prevent="save">
          <USwitch v-model="form.enabled" label="启用邮件通知" />
          <div class="form-grid">
            <UField label="SMTP 主机" for-id="smtp-host" required hint="例如 smtp.qq.com、smtp.gmail.com">
              <UInput id="smtp-host" v-model="form.host" placeholder="smtp.example.com" autocomplete="off" />
            </UField>
            <UField label="端口" for-id="smtp-port" required hint="STARTTLS 常用 587，SSL 常用 465">
              <UInput id="smtp-port" v-model.number="form.port" type="number" min="1" max="65535" />
            </UField>
            <UField label="加密方式">
              <UDropdownSelect
                v-model="form.encryption"
                :options="encryptionOptions"
                :searchable="false"
                aria-label="SMTP 加密方式"
              />
            </UField>
            <UField label="用户名" for-id="smtp-username" hint="通常为完整邮箱地址">
              <UInput id="smtp-username" v-model="form.username" autocomplete="username" placeholder="user@example.com" />
            </UField>
            <UField
              class="field--full"
              label="密码 / 授权码"
              for-id="smtp-password"
              :hint="hasPassword ? '已保存密码，留空表示不修改' : 'QQ 邮箱等需要使用授权码，不是登录密码'"
            >
              <UInput
                id="smtp-password"
                v-model="form.password"
                type="password"
                revealable
                autocomplete="new-password"
                :placeholder="hasPassword ? '已保存，留空不修改' : '请输入 SMTP 密码或授权码'"
              />
            </UField>
          </div>
        </form>
      </UWidget>

      <UWidget title="收件人" subtitle="测试邮件与通知的接收地址" icon="user" :span="5">
        <div class="stack">
          <UField label="发件人地址" for-id="smtp-from" required hint="需要与 SMTP 账号匹配，否则可能被拒收">
            <UInput id="smtp-from" v-model="form.from" placeholder="UPanel <noreply@example.com>" autocomplete="off" />
          </UField>
          <UField label="收件人列表" for-id="smtp-to" required hint="多个地址用逗号或换行分隔，最多 10 个">
            <UTextarea id="smtp-to" v-model="form.toText" :rows="4" placeholder="admin@example.com" />
          </UField>
          <div class="alert alert--info">
            <UIcon name="info" :size="16" class="alert__icon" />
            <div class="alert__body">
              SMTP 密码保存在 <code class="code--inline">data/email.json</code>（权限 0600），不会通过接口返回明文。
            </div>
          </div>
          <div class="alert alert--warning">
            <UIcon name="alert" :size="16" class="alert__icon" />
            <div class="alert__body">部分服务商要求先开启 SMTP 服务并使用独立授权码，请先在邮箱后台确认。</div>
          </div>
        </div>
      </UWidget>

      <UWidget title="配置说明" subtitle="常见邮件服务商参数" icon="info" :span="12">
        <div class="table-wrap" style="border: 0">
          <table class="table table--compact">
            <caption class="sr-only">常见 SMTP 服务商参数</caption>
            <thead>
              <tr>
                <th scope="col">服务商</th>
                <th scope="col">主机</th>
                <th scope="col">端口</th>
                <th scope="col">加密</th>
              </tr>
            </thead>
            <tbody>
              <tr><td>QQ 邮箱</td><td class="mono">smtp.qq.com</td><td class="mono">465 / 587</td><td>SSL / STARTTLS</td></tr>
              <tr><td>网易 163</td><td class="mono">smtp.163.com</td><td class="mono">465</td><td>SSL</td></tr>
              <tr><td>阿里云邮件推送</td><td class="mono">smtpdm.aliyun.com</td><td class="mono">465</td><td>SSL</td></tr>
              <tr><td>Gmail</td><td class="mono">smtp.gmail.com</td><td class="mono">587</td><td>STARTTLS</td></tr>
            </tbody>
          </table>
        </div>
      </UWidget>
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import UButton from '@/components/ui/UButton.vue'
import UDropdownSelect from '@/components/ui/UDropdownSelect.vue'
import UField from '@/components/ui/UField.vue'
import UIcon from '@/components/ui/UIcon.vue'
import UInput from '@/components/ui/UInput.vue'
import USwitch from '@/components/ui/USwitch.vue'
import UTextarea from '@/components/ui/UTextarea.vue'
import UWidget from '@/components/ui/UWidget.vue'
import { emailApi, errText } from '@/api'
import { toastError, toastSuccess } from '@/composables/useToast'

const loading = ref(false)
const loaded = ref(false)
const saving = ref(false)
const testing = ref(false)
const error = ref('')
const message = ref('')
const messageType = ref('success')
const hasPassword = ref(false)

const encryptionOptions = [
  { value: 'starttls', label: 'STARTTLS（587）' },
  { value: 'ssl', label: 'SSL/TLS（465）' },
  { value: 'none', label: '不加密（不推荐）' },
]

const form = reactive({
  enabled: false,
  host: '',
  port: 587,
  encryption: 'starttls',
  username: '',
  password: '',
  from: '',
  toText: '',
})

function parseTo(text) {
  return text
    .split(/[\n,，;；]+/)
    .map((item) => item.trim())
    .filter(Boolean)
}

function payload() {
  return {
    enabled: form.enabled,
    host: form.host.trim(),
    port: Number(form.port) || 587,
    encryption: form.encryption,
    username: form.username.trim(),
    password: form.password,
    from: form.from.trim(),
    to: parseTo(form.toText),
  }
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await emailApi.get()
    const data = res.data || {}
    form.enabled = !!data.enabled
    form.host = data.host || ''
    form.port = data.port || 587
    form.encryption = data.encryption || 'starttls'
    form.username = data.username || ''
    form.from = data.from || ''
    form.toText = (data.to || []).join('\n')
    hasPassword.value = !!data.hasPassword
    form.password = ''
    loaded.value = true
  } catch (e) {
    error.value = errText(e, '读取邮件配置失败')
  } finally {
    loading.value = false
  }
}

async function save() {
  if (saving.value) return
  message.value = ''
  const body = payload()
  if (!body.host) {
    error.value = '请填写 SMTP 主机'
    return
  }
  if (!body.from) {
    error.value = '请填写发件人地址'
    return
  }
  if (body.to.length === 0) {
    error.value = '请至少填写一个收件人'
    return
  }
  saving.value = true
  error.value = ''
  try {
    const res = await emailApi.save(body)
    hasPassword.value = !!(res.data && res.data.hasPassword)
    form.password = ''
    messageType.value = 'success'
    message.value = '邮件配置已保存'
    toastSuccess('邮件配置已保存')
  } catch (e) {
    error.value = errText(e, '保存邮件配置失败')
  } finally {
    saving.value = false
  }
}

async function testEmail() {
  if (testing.value) return
  message.value = ''
  const body = payload()
  if (!body.host || !body.from || body.to.length === 0) {
    error.value = '请先填写 SMTP 主机、发件人和收件人'
    return
  }
  testing.value = true
  error.value = ''
  try {
    await emailApi.test(body)
    messageType.value = 'success'
    message.value = '测试邮件已发送，请检查收件箱（注意垃圾邮件目录）'
    toastSuccess('测试邮件已发送')
  } catch (e) {
    messageType.value = 'danger'
    message.value = errText(e, '发送测试邮件失败')
    toastError('发送测试邮件失败', errText(e))
  } finally {
    testing.value = false
  }
}

onMounted(load)
</script>
