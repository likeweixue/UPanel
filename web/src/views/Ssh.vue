<template>
  <div class="page">
    <header class="page__header">
      <div>
        <div class="page__title-row">
          <h1 class="page__title">SSH 服务</h1>
          <UStatusDot
            :status="ssh?.serviceActive ? 'running' : ssh?.installed ? 'warning' : 'error'"
            :text="ssh?.serviceActive ? 'sshd 运行中' : ssh?.installed ? 'sshd 未运行' : '未检测到'"
          />
        </div>
        <p class="page__subtitle">只读查看 sshd 配置与安全建议，不在此处修改系统配置</p>
      </div>
      <div class="page__actions">
        <UButton icon="refresh" label="刷新" :loading="loading" @click="load" />
      </div>
    </header>

    <div v-if="error" class="alert alert--danger" role="alert">
      <UIcon name="alert" :size="16" class="alert__icon" />
      <div class="alert__body">{{ error }}</div>
    </div>

    <div v-if="loading && !ssh" class="loading-block"><span class="spinner" /> 正在读取 sshd 配置…</div>

    <div v-else-if="ssh && ssh.supported === false" class="table-wrap">
      <UEmpty
        icon="terminal"
        title="当前系统不支持该功能"
        :description="ssh.error || 'SSH 状态仅在 Linux 服务器上可用。'"
      />
    </div>

    <div v-else-if="ssh" class="widget-grid">
      <UWidget title="服务状态" subtitle="systemctl 检测结果" icon="terminal" :span="7">
        <div class="stack">
          <div class="row">
            <UStatusDot
              :status="ssh.serviceActive ? 'running' : ssh.installed ? 'warning' : 'error'"
              :text="ssh.serviceActive ? 'SSH 服务运行中' : ssh.installed ? 'SSH 服务未运行' : '未安装 / 无法检测'"
            />
            <span class="badge">端口 {{ ssh.port || '22' }}</span>
            <span v-if="ssh.serviceName" class="badge badge--primary">{{ ssh.serviceName }}</span>
          </div>
          <dl class="info-list">
            <div class="info-list__row">
              <dt class="info-list__label">配置文件</dt>
              <dd class="info-list__value mono">{{ ssh.configPath || '/etc/ssh/sshd_config' }}</dd>
            </div>
            <div class="info-list__row">
              <dt class="info-list__label">密码登录</dt>
              <dd class="info-list__value">
                <span :class="['badge', ssh.passwordAuthentication === 'yes' ? 'badge--warning' : 'badge--success']">
                  {{ ssh.passwordAuthentication || '未设置' }}
                </span>
              </dd>
            </div>
            <div class="info-list__row">
              <dt class="info-list__label">root 登录</dt>
              <dd class="info-list__value">
                <span :class="['badge', ssh.permitRootLogin === 'yes' ? 'badge--danger' : 'badge--success']">
                  {{ ssh.permitRootLogin || '未设置' }}
                </span>
              </dd>
            </div>
            <div class="info-list__row">
              <dt class="info-list__label">公钥登录</dt>
              <dd class="info-list__value">{{ ssh.pubkeyAuthentication || '未设置' }}</dd>
            </div>
          </dl>
          <div v-if="ssh.error" class="alert alert--warning">
            <UIcon name="alert" :size="16" class="alert__icon" />
            <div class="alert__body">{{ ssh.error }}</div>
          </div>
          <div class="field">
            <span class="field__label">连接命令</span>
            <div class="row" style="align-items: flex-start">
              <code class="code--inline grow" style="white-space: normal">{{ sshCommand }}</code>
              <UButton variant="secondary" size="sm" icon="copy" label="复制" @click="copy(sshCommand)" />
            </div>
            <p class="field__hint">请在本地终端执行，面板不会保存你的 SSH 密钥。</p>
          </div>
        </div>
      </UWidget>

      <UWidget title="安全建议" subtitle="来自当前 sshd 配置" icon="shield" :span="5">
        <div v-if="!ssh.warnings || ssh.warnings.length === 0" class="alert alert--success">
          <UIcon name="check-circle" :size="16" class="alert__icon" />
          <div class="alert__body">当前 SSH 配置未发现明显风险</div>
        </div>
        <div v-else class="warning-list">
          <div v-for="(warning, index) in ssh.warnings" :key="index" class="warning-item">
            <UIcon name="alert" :size="15" />
            <span>{{ warning }}</span>
          </div>
        </div>
        <div class="alert alert--info" style="margin-top: 12px">
          <UIcon name="info" :size="16" class="alert__icon" />
          <div class="alert__body">
            修改 <code class="code--inline">/etc/ssh/sshd_config</code> 后执行
            <code class="code--inline">sshd -t</code> 校验，再 <code class="code--inline">systemctl reload sshd</code>。
          </div>
        </div>
      </UWidget>

      <UWidget title="加固清单" subtitle="适合生产服务器的通用建议" icon="check-circle" :span="12">
        <div class="checklist">
          <div v-for="tip in tips" :key="tip.title" class="checklist__item">
            <span class="checklist__icon checklist__icon--ok"><UIcon name="check" :size="13" /></span>
            <div class="checklist__body">
              <p class="checklist__title">{{ tip.title }}</p>
              <p class="checklist__desc">{{ tip.desc }}</p>
            </div>
          </div>
        </div>
      </UWidget>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import UButton from '@/components/ui/UButton.vue'
import UEmpty from '@/components/ui/UEmpty.vue'
import UIcon from '@/components/ui/UIcon.vue'
import UStatusDot from '@/components/ui/UStatusDot.vue'
import UWidget from '@/components/ui/UWidget.vue'
import { errText, systemApi } from '@/api'
import { copyText } from '@/composables/useFormat'
import { toastError, toastSuccess } from '@/composables/useToast'

const ssh = ref(null)
const loading = ref(false)
const error = ref('')

const sshCommand = computed(() => {
  const host = window.location.hostname || '服务器IP'
  return `ssh root@${host} -p ${ssh.value?.port || 22}`
})

const tips = [
  { title: '使用密钥登录', desc: '禁用密码登录（PasswordAuthentication no），只保留公钥认证。' },
  { title: '限制 root 登录', desc: '设置 PermitRootLogin prohibit-password，日常使用普通用户 + sudo。' },
  { title: '修改默认端口', desc: '把 22 端口改为高位端口，并同步更新防火墙与云安全组规则。' },
  { title: '限制来源 IP', desc: '如果固定办公网络，可在安全组中只允许指定 IP 访问 SSH。' },
  { title: '安装 fail2ban', desc: '自动封禁暴力破解来源 IP，降低公网暴露风险。' },
]

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await systemApi.ssh()
    ssh.value = res.data || null
  } catch (e) {
    error.value = errText(e, '读取 SSH 状态失败')
  } finally {
    loading.value = false
  }
}

async function copy(text) {
  try {
    await copyText(text)
    toastSuccess('已复制')
  } catch (e) {
    toastError('复制失败', '请手动选择文本复制')
  }
}

onMounted(load)
</script>
