<template>
  <div class="page">
    <div class="page-shell">
    <header class="page__header">
      <div>
        <div class="page__title-row">
          <h1 class="page__title">面板设置</h1>
          <span class="badge">{{ panelInfo.version || '—' }}</span>
        </div>
        <p class="page__subtitle">账号、端口、安全入口、网关、Docker 与外观</p>
      </div>
      <div class="page__actions">
        <UButton icon="refresh" label="重新加载" :loading="loading" @click="loadAll" />
        <UButton
          v-if="['panel', 'gateway', 'appearance'].includes(activeTab)"
          variant="primary"
          icon="save"
          label="保存设置"
          :loading="saving"
          @click="saveSettings"
        />
        <UButton
          v-else-if="activeTab === 'docker'"
          variant="primary"
          icon="save"
          label="保存 Docker 配置"
          :loading="savingDocker"
          @click="saveDocker"
        />
      </div>
    </header>

    <div v-if="error" class="alert alert--danger" role="alert">
      <UIcon name="alert" :size="16" class="alert__icon" />
      <div class="alert__body">{{ error }}</div>
    </div>
    <div v-if="message" class="alert alert--success" role="status">
      <UIcon name="check-circle" :size="16" class="alert__icon" />
      <div class="alert__body">{{ message }}</div>
    </div>
    <div v-if="restartRequired" class="alert alert--warning" role="alert">
      <UIcon name="alert" :size="16" class="alert__icon" />
      <div class="alert__body">
        端口 / 监听地址 / 安全入口已保存，但需要重启面板进程后生效。请在服务器执行
        <code class="code--inline">up 3</code> 或 <code class="code--inline">systemctl restart upanel</code>。
      </div>
    </div>

    <UTabs v-model="activeTab" :tabs="tabs" aria-label="面板设置标签页">
      <!-- 面板 -->
      <div v-if="activeTab === 'panel'" class="widget-grid">
        <UWidget title="面板访问" subtitle="监听地址与安全入口" icon="settings" :span="7">
          <div class="form-grid">
            <UField label="面板名称" for-id="panel-name">
              <UInput id="panel-name" v-model="settings.panelName" placeholder="UPanel" />
            </UField>
            <UField label="面板端口" for-id="panel-port" hint="1-65535，修改后需要重启面板">
              <UInput id="panel-port" v-model="settings.panelPort" inputmode="numeric" />
            </UField>
            <UField label="监听地址" for-id="panel-bind" hint="0.0.0.0 表示所有网卡；127.0.0.1 仅本机">
              <UInput id="panel-bind" v-model="settings.bindIP" placeholder="0.0.0.0" />
            </UField>
            <UField label="安全入口" for-id="panel-entry" hint="留空为不启用；例如 mypanel 表示通过 /mypanel 访问">
              <UInput id="panel-entry" v-model="settings.securityEntry" placeholder="mypanel" />
            </UField>
          </div>
          <div class="alert alert--warning" style="margin-top: 14px">
            <UIcon name="alert" :size="16" class="alert__icon" />
            <div class="alert__body">
              启用安全入口后，除登录页和 API 外的所有路径都必须带该前缀；旧的访问地址会返回 404。
            </div>
          </div>
        </UWidget>

        <UWidget title="数据目录" subtitle="必须位于安装目录之内" icon="folder" :span="5">
          <div class="stack">
            <UField label="网站目录" for-id="path-website">
              <UInput id="path-website" v-model="settings.websitePath" class="mono" />
            </UField>
            <UField label="数据库目录" for-id="path-database">
              <UInput id="path-database" v-model="settings.databasePath" class="mono" />
            </UField>
            <UField label="应用目录" for-id="path-app">
              <UInput id="path-app" v-model="settings.appPath" class="mono" />
            </UField>
            <div class="alert alert--info">
              <UIcon name="info" :size="16" class="alert__icon" />
              <div class="alert__body">
                修改目录不会自动迁移已有文件，且会触发网关重建以更新挂载。请谨慎操作。
              </div>
            </div>
          </div>
        </UWidget>
      </div>

      <!-- 账号 -->
      <div v-else-if="activeTab === 'account'" class="widget-grid">
        <UWidget title="当前账号" subtitle="单管理员模式" icon="user" :span="5">
          <dl class="info-list">
            <div class="info-list__row">
              <dt class="info-list__label">用户名</dt>
              <dd class="info-list__value mono">{{ username }}</dd>
            </div>
            <div class="info-list__row">
              <dt class="info-list__label">角色</dt>
              <dd class="info-list__value">管理员（全部权限）</dd>
            </div>
            <div class="info-list__row">
              <dt class="info-list__label">登录状态</dt>
              <dd class="info-list__value"><UStatusDot status="running" text="已登录" /></dd>
            </div>
          </dl>
          <div class="alert alert--info" style="margin-top: 14px">
            <UIcon name="info" :size="16" class="alert__icon" />
            <div class="alert__body">
              忘记密码时，可在服务器执行 <code class="code--inline">up 9 password 新密码</code> 重置。
            </div>
          </div>
        </UWidget>

        <UWidget title="修改密码" subtitle="修改成功后所有会话立即失效" icon="key" :span="7">
          <form class="stack" @submit.prevent="changePassword">
            <div v-if="passwordError" class="alert alert--danger" role="alert">
              <UIcon name="alert" :size="16" class="alert__icon" />
              <div class="alert__body">{{ passwordError }}</div>
            </div>
            <UField label="当前密码" for-id="set-old-password" required>
              <UInput id="set-old-password" v-model="passwordForm.oldPassword" type="password" revealable autocomplete="current-password" />
            </UField>
            <div class="form-grid">
              <UField label="新密码" for-id="set-new-password" required hint="至少 8 位，包含字母和数字">
                <UInput id="set-new-password" v-model="passwordForm.newPassword" type="password" revealable autocomplete="new-password" />
              </UField>
              <UField label="确认新密码" for-id="set-confirm-password" required>
                <UInput id="set-confirm-password" v-model="passwordForm.confirmPassword" type="password" revealable autocomplete="new-password" />
              </UField>
            </div>
            <div class="row">
              <UButton type="submit" variant="primary" icon="key" label="修改密码" :loading="changingPassword" />
            </div>
          </form>
        </UWidget>
      </div>

      <!-- 网关 -->
      <div v-else-if="activeTab === 'gateway'" class="widget-grid">
        <UWidget title="网站网关" subtitle="所有站点共用的 Nginx 容器" icon="network" :span="7">
          <div class="form-grid">
            <UField label="网关镜像" for-id="gateway-image" hint="默认 nginx:alpine，可换成私有镜像">
              <UInput id="gateway-image" v-model="settings.gatewayImage" class="mono" />
            </UField>
            <div />
            <UField label="HTTP 端口" for-id="gateway-http" hint="网站对外 HTTP 端口，默认 80">
              <UInput id="gateway-http" v-model="settings.gatewayHttpPort" inputmode="numeric" />
            </UField>
            <UField label="HTTPS 端口" for-id="gateway-https" hint="网站对外 HTTPS 端口，默认 443">
              <UInput id="gateway-https" v-model="settings.gatewayHttpsPort" inputmode="numeric" />
            </UField>
          </div>
          <div class="alert alert--warning" style="margin-top: 14px">
            <UIcon name="alert" :size="16" class="alert__icon" />
            <div class="alert__body">
              修改镜像或端口会重建网关容器（短暂中断）。若宿主机已有 nginx/apache 占用 80/443，请先停止它们或更换端口。
            </div>
          </div>
        </UWidget>

        <UWidget title="网关状态" subtitle="实时容器状态" icon="activity" :span="5">
          <div class="stack">
            <div class="row">
              <UStatusDot :status="gateway.state === 'running' ? 'running' : 'warning'" :text="gateway.state === 'running' ? '运行中' : '未运行'" />
              <span class="badge">{{ gateway.httpPort || settings.gatewayHttpPort }}</span>
            </div>
            <dl class="info-list">
              <div class="info-list__row">
                <dt class="info-list__label">容器</dt>
                <dd class="info-list__value mono">{{ gateway.container || 'upanel-gateway' }}</dd>
              </div>
              <div class="info-list__row">
                <dt class="info-list__label">HTTPS</dt>
                <dd class="info-list__value">{{ gateway.httpsPort || settings.gatewayHttpsPort }}</dd>
              </div>
            </dl>
            <UButton variant="secondary" icon="wrench" label="修复 / 重载网关" :loading="repairing" @click="repairGateway" />
          </div>
        </UWidget>

        <UWidget title="镜像加速" subtitle="写入面板配置，供手动同步到 Docker" icon="cloud" :span="12">
          <div class="stack">
            <UField label="加速地址" hint="每行一个，例如 https://docker.1panel.live；也应在 Docker 设置中写入 daemon.json">
              <UTextarea v-model="mirrorsText" :rows="4" placeholder="https://docker.1panel.live" />
            </UField>
            <div class="row">
              <UButton variant="secondary" icon="save" label="保存加速地址" :loading="saving" @click="saveSettings" />
              <UButton variant="ghost" icon="server" label="打开 Docker 设置" @click="activeTab = 'docker'" />
            </div>
          </div>
        </UWidget>
      </div>

      <!-- Docker -->
      <div v-else-if="activeTab === 'docker'" class="widget-grid">
        <UWidget title="Docker 守护进程" subtitle="合并写入 /etc/docker/daemon.json，保留其它字段" icon="server" :span="7">
          <div class="stack">
            <div class="row">
              <UStatusDot :status="dockerVersion ? 'running' : 'error'" :text="dockerVersion ? `Docker ${dockerVersion}` : '未连接'" />
              <UButton variant="secondary" size="sm" icon="refresh" label="检测" :loading="checkingDocker" @click="loadDocker" />
            </div>
            <UField label="镜像加速地址" hint="每行一个，仅支持 http(s) URL">
              <UTextarea v-model="dockerForm.mirrorsText" :rows="4" placeholder="https://docker.1panel.live" />
            </UField>
            <UField label="非安全仓库" hint="可选，每行一个 host:port">
              <UTextarea v-model="dockerForm.insecureText" :rows="2" placeholder="registry.local:5000" />
            </UField>
            <div class="form-grid">
              <UField label="日志驱动" for-id="docker-log-driver">
                <UDropdownSelect
                  v-model="dockerForm.logDriver"
                  :options="logDriverOptions"
                  :searchable="false"
                  aria-label="Docker 日志驱动"
                />
              </UField>
              <UField label="最大并发下载数" for-id="docker-max-download" hint="0 表示使用 Docker 默认值">
                <UInput id="docker-max-download" v-model.number="dockerForm.maxConcurrentDownloads" type="number" min="0" max="64" />
              </UField>
            </div>
            <div class="alert alert--warning">
              <UIcon name="alert" :size="16" class="alert__icon" />
              <div class="alert__body">
                保存只会修改 daemon.json 中对应字段，原文件会备份为 <code class="code--inline">daemon.json.upanel.bak</code>。需要重启 Docker 才生效。
              </div>
            </div>
            <div class="row">
              <UButton variant="primary" icon="save" label="保存 Docker 配置" :loading="savingDocker" @click="saveDocker" />
              <UButton variant="danger-soft" icon="restart" label="重启 Docker" :loading="restartingDocker" @click="restartDocker" />
            </div>
          </div>
        </UWidget>

        <UWidget title="运行信息" subtitle="来自 docker info" icon="info" :span="5">
          <dl class="info-list">
            <div class="info-list__row">
              <dt class="info-list__label">版本</dt>
              <dd class="info-list__value mono">{{ dockerVersion || '—' }}</dd>
            </div>
            <div class="info-list__row">
              <dt class="info-list__label">配置文件</dt>
              <dd class="info-list__value mono">/etc/docker/daemon.json</dd>
            </div>
            <div class="info-list__row">
              <dt class="info-list__label">当前镜像加速</dt>
              <dd class="info-list__value">{{ dockerForm.mirrorsText ? dockerForm.mirrorsText.split('\n').filter(Boolean).length + ' 个' : '未配置' }}</dd>
            </div>
          </dl>
          <div class="alert alert--info" style="margin-top: 12px">
            <UIcon name="info" :size="16" class="alert__icon" />
            <div class="alert__body">
              如果拉取镜像缓慢，优先配置镜像加速；国内网络下建议使用可用的代理或镜像站。
            </div>
          </div>
        </UWidget>
      </div>

      <!-- 外观 -->
      <div v-else-if="activeTab === 'appearance'" class="widget-grid">
        <UWidget title="主题" subtitle="立即生效，保存在浏览器" icon="sun" :span="12">
          <div class="stack">
            <div class="type-grid theme-grid">
              <button
                v-for="option in themeOptions"
                :key="option.value"
                type="button"
                class="type-card"
                :aria-pressed="themeState.mode === option.value ? 'true' : 'false'"
                @click="chooseTheme(option.value)"
              >
                <span class="theme-swatch" aria-hidden="true">
                  <i :style="{ background: option.swatch[0] }" />
                  <i :style="{ background: option.swatch[1] }" />
                </span>
                <span class="type-card__title"><UIcon :name="option.icon" :size="15" /> {{ option.label }}</span>
                <span class="type-card__desc">{{ option.desc }}</span>
              </button>
            </div>
            <UField label="界面密度">
              <UDropdownSelect
                :model-value="themeState.density"
                :options="densityOptions"
                :searchable="false"
                aria-label="界面密度"
                @update:model-value="chooseDensity"
              />
            </UField>
            <div class="row">
              <UButton variant="secondary" icon="save" label="保存到服务器" :loading="saving" @click="saveSettings" />
            </div>
          </div>
        </UWidget>

        <UWidget title="界面圆角" subtitle="0 为直角，默认 4px" icon="grid" :span="6">
          <div class="stack">
            <div class="radius-preview" :style="{ borderRadius: `${settings.cornerRadius}px` }">
              <span class="radius-preview__bar" :style="{ borderRadius: `${settings.cornerRadius}px` }" />
              <span class="radius-preview__panel" :style="{ borderRadius: `${settings.cornerRadius}px` }">
                <span class="radius-preview__line" />
                <span class="radius-preview__line radius-preview__line--short" />
              </span>
              <span class="radius-preview__value">{{ settings.cornerRadius }} px</span>
            </div>
            <input
              class="range"
              type="range"
              min="0"
              max="16"
              step="1"
              :value="settings.cornerRadius"
              aria-label="界面圆角"
              @input="chooseRadius($event.target.value)"
            />
            <div class="radius-presets">
              <button
                v-for="value in radiusOptions"
                :key="value"
                type="button"
                class="radius-chip"
                :class="{ 'is-active': settings.cornerRadius === value }"
                @click="chooseRadius(value)"
              >
                {{ value }}px
              </button>
            </div>
            <p class="field__hint">同时影响卡片、按钮、输入框、标签页和弹窗，保存后同步到服务器。</p>
          </div>
        </UWidget>

        <UWidget title="底部栏" subtitle="版本号与项目地址" icon="list" :span="6">
          <div class="stack">
            <div class="setting-row">
              <div>
                <strong>显示底部栏</strong>
                <p class="field__hint">关闭后页面底部不再显示信息栏</p>
              </div>
              <USwitch v-model="settings.footerVisible" aria-label="显示底部栏" />
            </div>
            <UField label="左侧文字" for-id="footer-text" hint="留空时显示 UPanel">
              <UInput id="footer-text" v-model="settings.footerText" placeholder="UPanel" maxlength="80" />
            </UField>
            <div class="setting-row">
              <div><strong>显示版本号</strong></div>
              <USwitch v-model="settings.footerShowVersion" aria-label="显示版本号" />
            </div>
            <div class="setting-row">
              <div>
                <strong>显示 GitHub 链接</strong>
                <p class="field__hint">https://github.com/likeweixue/UPanel</p>
              </div>
              <USwitch v-model="settings.footerShowGithub" aria-label="显示 GitHub 链接" />
            </div>
            <div v-if="settings.footerVisible" class="footer-preview" aria-hidden="true">
              <span>{{ settings.footerText || 'UPanel' }}</span>
              <span v-if="settings.footerShowVersion" class="subtle">{{ panelInfo.version || 'v0.1.7' }}</span>
              <span class="grow" />
              <span v-if="settings.footerShowGithub" class="row" style="gap: 4px">
                <UIcon name="github" :size="13" /> GitHub
              </span>
            </div>
          </div>
        </UWidget>
      </div>

      <!-- 关于 -->
      <div v-else class="widget-grid">
        <UWidget title="关于 UPanel" subtitle="轻量级 Linux 服务器管理面板" icon="info" :span="7">
          <div class="stack">
            <dl class="info-list">
              <div class="info-list__row">
                <dt class="info-list__label">版本</dt>
                <dd class="info-list__value mono">{{ panelInfo.version || '—' }}</dd>
              </div>
              <div class="info-list__row">
                <dt class="info-list__label">安装目录</dt>
                <dd class="info-list__value mono">{{ settings.websitePath ? settings.websitePath.replace(/\/data\/websites$/, '') : '—' }}</dd>
              </div>
              <div class="info-list__row">
                <dt class="info-list__label">网站目录</dt>
                <dd class="info-list__value mono">{{ settings.websitePath }}</dd>
              </div>
              <div class="info-list__row">
                <dt class="info-list__label">数据库目录</dt>
                <dd class="info-list__value mono">{{ settings.databasePath }}</dd>
              </div>
              <div class="info-list__row">
                <dt class="info-list__label">网关镜像</dt>
                <dd class="info-list__value mono">{{ settings.gatewayImage }}</dd>
              </div>
            </dl>
            <div class="alert alert--info">
              <UIcon name="info" :size="16" class="alert__icon" />
              <div class="alert__body">
                UPanel 使用 Go + Vue 3 构建，网站部署基于统一 Nginx 网关，所有数据保存在安装目录的
                <code class="code--inline">data/</code> 下。
              </div>
            </div>
          </div>
        </UWidget>

        <UWidget title="快捷命令" subtitle="服务器端常用操作" icon="terminal" :span="5">
          <div class="stack">
            <div class="field">
              <span class="field__label">查看面板地址</span>
              <code class="code--inline">up 6</code>
            </div>
            <div class="field">
              <span class="field__label">修改密码</span>
              <code class="code--inline">up 9 password 新密码</code>
            </div>
            <div class="field">
              <span class="field__label">重启面板</span>
              <code class="code--inline">up 3</code>
            </div>
            <div class="field">
              <span class="field__label">查看网关状态</span>
              <code class="code--inline">up gateway</code>
            </div>
            <div class="field">
              <span class="field__label">查看日志</span>
              <code class="code--inline">up logs</code>
            </div>
          </div>
        </UWidget>
      </div>
    </UTabs>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import UButton from '@/components/ui/UButton.vue'
import UDropdownSelect from '@/components/ui/UDropdownSelect.vue'
import UField from '@/components/ui/UField.vue'
import UIcon from '@/components/ui/UIcon.vue'
import UInput from '@/components/ui/UInput.vue'
import UStatusDot from '@/components/ui/UStatusDot.vue'
import USwitch from '@/components/ui/USwitch.vue'
import UTabs from '@/components/ui/UTabs.vue'
import UTextarea from '@/components/ui/UTextarea.vue'
import UWidget from '@/components/ui/UWidget.vue'
import { authApi, dockerApi, errText, gatewayApi, panelApi, settingsApi } from '@/api'
import { useAuth } from '@/composables/useAuth'
import { setCornerRadius, useTheme } from '@/composables/useTheme'
import { toastError, toastSuccess, toastWarning } from '@/composables/useToast'

const route = useRoute()
const router = useRouter()
const { authState, clear } = useAuth()
const { themeState, setTheme, setDensity } = useTheme()

const tabIds = ['panel', 'account', 'gateway', 'docker', 'appearance', 'about']
const activeTab = ref(tabIds.includes(route.query.tab) ? route.query.tab : 'panel')
const tabs = [
  { id: 'panel', label: '面板', icon: 'settings' },
  { id: 'account', label: '账号', icon: 'user' },
  { id: 'gateway', label: '网关', icon: 'network' },
  { id: 'docker', label: 'Docker', icon: 'server' },
  { id: 'appearance', label: '外观', icon: 'sun' },
  { id: 'about', label: '关于', icon: 'info' },
]

const loading = ref(false)
const saving = ref(false)
const savingDocker = ref(false)
const restartingDocker = ref(false)
const checkingDocker = ref(false)
const repairing = ref(false)
const error = ref('')
const message = ref('')
const restartRequired = ref(false)
const panelInfo = ref({})
const gateway = reactive({ state: 'unknown', httpPort: '', httpsPort: '', container: '' })
const dockerVersion = ref('')

const username = computed(() => (authState.user && authState.user.username) || 'admin')

const settings = reactive({
  panelName: 'UPanel',
  panelPort: '8080',
  bindIP: '0.0.0.0',
  securityEntry: '',
  websitePath: '',
  databasePath: '',
  appPath: '',
  gatewayImage: 'nginx:alpine',
  gatewayHttpPort: '80',
  gatewayHttpsPort: '443',
  registryMirrors: [],
  theme: 'light',
  density: 'comfortable',
  cornerRadius: 4,
  footerVisible: true,
  footerText: '',
  footerShowVersion: true,
  footerShowGithub: true,
})
const mirrorsText = ref('')

const dockerForm = reactive({
  mirrorsText: '',
  insecureText: '',
  logDriver: '',
  maxConcurrentDownloads: 0,
})

const logDriverOptions = [
  { value: '', label: 'Docker 默认' },
  { value: 'json-file', label: 'json-file' },
  { value: 'local', label: 'local' },
  { value: 'journald', label: 'journald' },
  { value: 'syslog', label: 'syslog' },
  { value: 'none', label: 'none' },
]

const themeOptions = [
  { value: 'light', label: '浅色', icon: 'sun', desc: '默认的扁平浅色界面', swatch: ['#ffffff', '#2f6fed'] },
  { value: 'retro-yellow', label: '复古黄', icon: 'sun', desc: '暖黄纸张，护眼', swatch: ['#fbf6ec', '#9a6428'] },
  { value: 'green-paper', label: '护眼纸', icon: 'sun', desc: '绿色纸张，低刺激', swatch: ['#f7faf4', '#3f7a4c'] },
  { value: 'dark', label: '深色', icon: 'moon', desc: '低亮度环境更舒适', swatch: ['#171a21', '#5b8cff'] },
  { value: 'black-gold', label: '黑金', icon: 'moon', desc: '黑底金色，沉稳', swatch: ['#12100c', '#d4af37'] },
  { value: 'ember', label: '暗夜橙红', icon: 'moon', desc: '暗色背景，橙红提示', swatch: ['#1c1311', '#ff6b3d'] },
  { value: 'system', label: '跟随系统', icon: 'settings-2', desc: '自动匹配系统主题', swatch: ['#ffffff', '#171a21'] },
]
const densityOptions = [
  { value: 'comfortable', label: '标准密度' },
  { value: 'compact', label: '紧凑密度' },
]
const radiusOptions = [0, 2, 4, 6, 8, 12, 16]

const passwordForm = reactive({ oldPassword: '', newPassword: '', confirmPassword: '' })
const changingPassword = ref(false)
const passwordError = ref('')

function applySettings(data) {
  if (!data) return
  settings.panelName = data.panelName || 'UPanel'
  settings.panelPort = String(data.panelPort || settings.panelPort)
  settings.bindIP = data.bindIP || settings.bindIP
  settings.securityEntry = data.securityEntry || ''
  settings.websitePath = data.websitePath || ''
  settings.databasePath = data.databasePath || ''
  settings.appPath = data.appPath || ''
  settings.gatewayImage = data.gatewayImage || 'nginx:alpine'
  settings.gatewayHttpPort = String(data.gatewayHttpPort || '80')
  settings.gatewayHttpsPort = String(data.gatewayHttpsPort || '443')
  settings.registryMirrors = data.registryMirrors || []
  mirrorsText.value = settings.registryMirrors.join('\n')
  settings.theme = themeState.mode
  settings.density = themeState.density
  settings.cornerRadius = Number.isFinite(Number(data.cornerRadius)) ? Number(data.cornerRadius) : 4
  setCornerRadius(settings.cornerRadius)
  settings.footerVisible = data.footerVisible !== false
  settings.footerText = data.footerText || ''
  settings.footerShowVersion = data.footerShowVersion !== false
  settings.footerShowGithub = data.footerShowGithub !== false
}

async function loadAll() {
  loading.value = true
  error.value = ''
  try {
    const [settingsRes, infoRes] = await Promise.all([settingsApi.get(), panelApi.info()])
    applySettings(settingsRes.data)
    panelInfo.value = infoRes || {}
    await Promise.all([loadGateway(), loadDocker()])
  } catch (e) {
    error.value = errText(e, '加载设置失败')
  } finally {
    loading.value = false
  }
}

async function loadGateway() {
  try {
    const res = await gatewayApi.status()
    Object.assign(gateway, res.data || {})
  } catch (e) {
    gateway.state = 'docker_unavailable'
  }
}

async function loadDocker() {
  checkingDocker.value = true
  try {
    const [configRes, versionRes] = await Promise.allSettled([dockerApi.config(), dockerApi.version()])
    if (configRes.status === 'fulfilled') {
      const data = configRes.value.data || {}
      dockerForm.mirrorsText = (data.registry_mirrors || []).join('\n')
      dockerForm.insecureText = (data.insecure_registries || []).join('\n')
      dockerForm.logDriver = data['log-driver'] || data.logDriver || ''
      dockerForm.maxConcurrentDownloads = data.max_concurrent_downloads || 0
    }
    if (versionRes.status === 'fulfilled') {
      dockerVersion.value = (versionRes.value.data && versionRes.value.data.version) || ''
    } else {
      dockerVersion.value = ''
    }
  } finally {
    checkingDocker.value = false
  }
}

function buildSettingsPayload() {
  settings.registryMirrors = mirrorsText.value
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
  return {
    panelName: settings.panelName,
    panelPort: settings.panelPort,
    bindIP: settings.bindIP,
    securityEntry: settings.securityEntry,
    websitePath: settings.websitePath,
    databasePath: settings.databasePath,
    appPath: settings.appPath,
    gatewayImage: settings.gatewayImage,
    gatewayHttpPort: settings.gatewayHttpPort,
    gatewayHttpsPort: settings.gatewayHttpsPort,
    registryMirrors: settings.registryMirrors,
    theme: settings.theme,
    density: settings.density,
    cornerRadius: settings.cornerRadius,
    footerVisible: settings.footerVisible,
    footerText: settings.footerText,
    footerShowVersion: settings.footerShowVersion,
    footerShowGithub: settings.footerShowGithub,
  }
}

async function saveSettings() {
  if (saving.value) return
  saving.value = true
  error.value = ''
  message.value = ''
  restartRequired.value = false
  try {
    const res = await settingsApi.update(buildSettingsPayload())
    applySettings(res.data)
    window.dispatchEvent(new CustomEvent('upanel:settings-updated', { detail: res.data }))
    message.value = res.message || '设置已保存'
    restartRequired.value = !!res.restartRequired
    if (restartRequired.value) toastWarning('设置已保存', '需要重启面板后生效')
    else toastSuccess('设置已保存')
    await Promise.all([loadGateway(), loadDocker()])
  } catch (e) {
    error.value = errText(e, '保存设置失败')
  } finally {
    saving.value = false
  }
}

async function repairGateway() {
  repairing.value = true
  try {
    await gatewayApi.ensure()
    toastSuccess('网关已就绪')
    await loadGateway()
  } catch (e) {
    toastError('修复网关失败', errText(e))
  } finally {
    repairing.value = false
  }
}

async function saveDocker() {
  if (savingDocker.value) return
  savingDocker.value = true
  error.value = ''
  message.value = ''
  try {
    await dockerApi.saveConfig({
      registry_mirrors: dockerForm.mirrorsText
        .split('\n')
        .map((line) => line.trim())
        .filter(Boolean),
      insecure_registries: dockerForm.insecureText
        .split('\n')
        .map((line) => line.trim())
        .filter(Boolean),
      'log-driver': dockerForm.logDriver,
      max_concurrent_downloads: Number(dockerForm.maxConcurrentDownloads) || 0,
    })
    message.value = 'Docker 配置已保存，重启 Docker 后生效'
    toastSuccess('Docker 配置已保存')
    await loadDocker()
  } catch (e) {
    error.value = errText(e, '保存 Docker 配置失败')
  } finally {
    savingDocker.value = false
  }
}

async function restartDocker() {
  if (restartingDocker.value) return
  restartingDocker.value = true
  try {
    await dockerApi.restart()
    toastSuccess('Docker 正在重启', '几秒后刷新页面状态')
    setTimeout(loadDocker, 5000)
  } catch (e) {
    toastError('重启 Docker 失败', errText(e))
  } finally {
    restartingDocker.value = false
  }
}

async function changePassword() {
  if (changingPassword.value) return
  passwordError.value = ''
  if (!passwordForm.oldPassword) {
    passwordError.value = '请输入当前密码'
    return
  }
  if (passwordForm.newPassword.length < 8) {
    passwordError.value = '新密码至少 8 位'
    return
  }
  if (!/[A-Za-z]/.test(passwordForm.newPassword) || !/[0-9]/.test(passwordForm.newPassword)) {
    passwordError.value = '新密码必须同时包含字母和数字'
    return
  }
  if (passwordForm.newPassword !== passwordForm.confirmPassword) {
    passwordError.value = '两次输入的新密码不一致'
    return
  }
  changingPassword.value = true
  try {
    await authApi.changePassword(passwordForm.oldPassword, passwordForm.newPassword)
    toastSuccess('密码已修改', '请使用新密码重新登录')
    clear()
    router.replace('/login')
  } catch (e) {
    passwordError.value = errText(e, '修改密码失败')
  } finally {
    changingPassword.value = false
  }
}

function chooseTheme(mode) {
  setTheme(mode)
  settings.theme = mode
  const option = themeOptions.find((item) => item.value === mode)
  toastSuccess('主题已切换', option ? option.label : mode)
}

function chooseDensity(density) {
  setDensity(density)
  settings.density = density
  toastSuccess('界面密度已更新')
}

function chooseRadius(value) {
  const px = Math.min(16, Math.max(0, Math.round(Number(value) || 0)))
  settings.cornerRadius = px
  setCornerRadius(px)
}

watch(activeTab, (tab) => {
  if (route.query.tab !== tab) router.replace({ path: route.path, query: { ...route.query, tab } })
})

watch(
  () => route.query.tab,
  (tab) => {
    if (!tab) {
      activeTab.value = 'panel'
      return
    }
    if (tabIds.includes(tab) && tab !== activeTab.value) activeTab.value = tab
  }
)

onMounted(loadAll)
</script>
