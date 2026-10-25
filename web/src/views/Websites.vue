<template>
  <div class="page">
    <div class="page-shell">
    <header class="page__header">
      <div>
        <div class="page__title-row">
          <h1 class="page__title">网站</h1>
          <span class="badge badge--primary">{{ sites.length }}</span>
        </div>
        <p class="page__subtitle">统一网关按域名转发到静态目录、PHP-FPM 或反代容器</p>
      </div>
      <div class="page__actions">
        <UButton icon="refresh" label="刷新" :loading="loading" @click="loadSites" />
        <UButton variant="primary" icon="plus" label="创建网站" @click="openCreate" />
      </div>
    </header>

    <div v-if="error" class="alert alert--danger" role="alert">
      <UIcon name="alert" :size="16" class="alert__icon" />
      <div class="alert__body">{{ error }}</div>
    </div>

    <UTabs v-model="activeTab" :tabs="tabs" aria-label="网站管理标签页">
      <!-- 站点列表 -->
      <div v-if="activeTab === 'sites'" class="stack">
        <div class="toolbar">
          <div class="toolbar__left">
            <div class="input-affix" style="width: 260px">
              <UIcon name="search" :size="15" class="subtle" style="position: absolute; left: 10px" />
              <input v-model="search" class="input" placeholder="搜索域名、别名或备注" style="padding-left: 32px" />
            </div>
          </div>
          <div class="toolbar__right">
            <span class="subtle" style="font-size: 12px">{{ filteredSites.length }} / {{ sites.length }} 个站点</span>
          </div>
        </div>

        <div v-if="loading && sites.length === 0" class="loading-block"><span class="spinner" /> 正在加载站点…</div>

        <div v-else-if="filteredSites.length === 0" class="table-wrap">
          <UEmpty
            icon="globe"
            title="还没有网站"
            description="创建站点后，把域名解析到服务器 IP，再上传你的网站文件即可访问。"
          >
            <UButton variant="primary" icon="plus" label="创建第一个网站" @click="openCreate" />
          </UEmpty>
        </div>

        <div v-else class="table-wrap">
          <table class="table">
            <caption class="sr-only">网站列表</caption>
            <thead>
              <tr>
                <th scope="col">域名</th>
                <th scope="col">类型</th>
                <th scope="col">状态</th>
                <th scope="col">HTTPS</th>
                <th scope="col">网站根目录</th>
                <th scope="col" style="text-align: right">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="site in filteredSites" :key="site.domain">
                <td>
                  <div class="stack stack--sm" style="gap: 2px">
                    <a
                      v-if="!site.domain.startsWith('*.')"
                      class="table__primary site-url"
                      :href="siteUrl(site)"
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      {{ site.domain }}
                      <UIcon name="external" :size="12" />
                    </a>
                    <span v-else class="table__primary">{{ site.domain }}</span>
                    <span v-if="site.remark" class="subtle" style="font-size: 11px">{{ site.remark }}</span>
                    <span v-if="site.aliases && site.aliases.length" class="subtle" style="font-size: 11px">
                      别名：{{ site.aliases.join(', ') }}
                    </span>
                  </div>
                </td>
                <td><span class="badge" :class="typeBadge(site.type)">{{ typeLabel(site.type) }}</span></td>
                <td><UStatusDot :status="site.status" :text="statusLabel(site.status)" /></td>
                <td>
                  <span :class="['badge', site.sslEnabled ? 'badge--success' : 'badge--warning']">
                    {{ site.sslEnabled ? (site.forceSSL ? '强制 HTTPS' : '已启用') : '未启用' }}
                  </span>
                </td>
                <td>
                  <div class="row" style="gap: 4px; flex-wrap: nowrap">
                    <code class="code--inline truncate" style="max-width: 240px">{{ site.rootDir }}</code>
                    <button class="copy-btn" type="button" aria-label="复制网站根目录" @click="copy(site.rootDir)">
                      <UIcon name="copy" :size="13" />
                    </button>
                  </div>
                </td>
                <td>
                  <div class="table__actions">
                    <UButton
                      variant="ghost"
                      icon-only
                      size="sm"
                      icon="logs"
                      aria-label="查看站点日志"
                      title="站点日志"
                      @click="openLogs(site)"
                    />
                    <UButton
                      v-if="site.status === 'running'"
                      variant="ghost"
                      icon-only
                      size="sm"
                      icon="stop"
                      aria-label="停止网站"
                      title="停止"
                      :loading="busyDomain === site.domain"
                      @click="toggleSite(site, 'stop')"
                    />
                    <UButton
                      v-else
                      variant="ghost"
                      icon-only
                      size="sm"
                      icon="play"
                      aria-label="启动网站"
                      title="启动"
                      :loading="busyDomain === site.domain"
                      @click="toggleSite(site, 'start')"
                    />
                    <UButton
                      variant="ghost"
                      icon-only
                      size="sm"
                      icon="edit"
                      aria-label="编辑网站"
                      title="编辑"
                      @click="openEdit(site)"
                    />
                    <UButton
                      variant="ghost"
                      icon-only
                      size="sm"
                      icon="trash"
                      aria-label="删除网站"
                      title="删除"
                      @click="openDelete(site)"
                    />
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- 域名与别名 -->
      <div v-else-if="activeTab === 'domains'" class="stack">
        <div class="alert alert--info">
          <UIcon name="info" :size="16" class="alert__icon" />
          <div class="alert__body">
            域名的 A 记录需要指向本机公网 IP。修改别名后网关会自动执行 <code class="code--inline">nginx -t</code> 并热重载。
          </div>
        </div>
        <div v-if="sites.length === 0" class="table-wrap">
          <UEmpty icon="link" title="还没有可管理的域名" description="先创建一个网站，再为其添加别名域名。">
            <UButton variant="primary" icon="plus" label="创建网站" @click="openCreate" />
          </UEmpty>
        </div>
        <div v-else class="table-wrap">
          <table class="table">
            <caption class="sr-only">站点主域名与别名</caption>
            <thead>
              <tr>
                <th scope="col">主域名</th>
                <th scope="col">别名域名</th>
                <th scope="col">解析提示</th>
                <th scope="col" style="text-align: right">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="site in sites" :key="site.domain">
                <td class="table__primary">{{ site.domain }}</td>
                <td>
                  <div v-if="site.aliases && site.aliases.length" class="row" style="gap: 4px">
                    <span v-for="alias in site.aliases" :key="alias" class="chip">{{ alias }}</span>
                  </div>
                  <span v-else class="subtle">未设置别名</span>
                </td>
                <td class="muted" style="font-size: 12px">
                  {{ site.domain.startsWith('*.') ? '通配解析（*.域名）' : `A 记录 → 服务器 IP` }}
                </td>
                <td>
                  <div class="table__actions">
                    <UButton variant="ghost" size="sm" icon="edit" label="编辑" @click="openEdit(site)" />
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- SSL -->
      <div v-else-if="activeTab === 'ssl'" class="stack">
        <div class="alert alert--info">
          <UIcon name="info" :size="16" class="alert__icon" />
          <div class="alert__body">
            当前版本支持手动粘贴证书（fullchain + privkey）。证书写入前会校验私钥匹配与到期时间，配置错误会自动回滚。
          </div>
        </div>
        <div v-if="sites.length === 0" class="table-wrap">
          <UEmpty icon="ssl" title="还没有网站" description="创建网站后即可部署 SSL 证书。" />
        </div>
        <div v-else class="table-wrap">
          <table class="table">
            <caption class="sr-only">网站 SSL 证书</caption>
            <thead>
              <tr>
                <th scope="col">域名</th>
                <th scope="col">证书状态</th>
                <th scope="col">HTTPS 跳转</th>
                <th scope="col" style="text-align: right">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="site in sites" :key="site.domain">
                <td class="table__primary">{{ site.domain }}</td>
                <td>
                  <span :class="['badge', site.sslEnabled ? 'badge--success' : 'badge--warning']">
                    {{ site.sslEnabled ? '已部署' : '未部署' }}
                  </span>
                </td>
                <td>{{ site.forceSSL ? '强制跳转 HTTPS' : '否' }}</td>
                <td>
                  <div class="table__actions">
                    <UButton
                      variant="secondary"
                      size="sm"
                      icon="ssl"
                      :label="site.sslEnabled ? '更新证书' : '部署证书'"
                      @click="openSSL(site)"
                    />
                    <UButton
                      v-if="site.sslEnabled"
                      variant="danger-soft"
                      size="sm"
                      label="关闭 SSL"
                      @click="disableSSL(site)"
                    />
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- 日志 -->
      <div v-else-if="activeTab === 'logs'" class="stack">
        <div class="log-toolbar">
          <UDropdownSelect
            v-model="logDomain"
            :options="logOptions"
            placeholder="选择站点"
            aria-label="选择要查看日志的站点"
            style="width: 240px"
            @change="loadLogs"
          />
          <UDropdownSelect
            v-model="logKind"
            :options="logKindOptions"
            :searchable="false"
            aria-label="日志类型"
            style="width: 150px"
            @change="loadLogs"
          />
          <UButton icon="refresh" label="刷新日志" :loading="logLoading" @click="loadLogs" />
          <span class="subtle" style="font-size: 12px">显示最后 {{ logLines }} 行</span>
        </div>
        <pre v-if="logContent" class="log" aria-label="站点日志内容">{{ logContent }}</pre>
        <div v-else class="table-wrap">
          <UEmpty
            icon="logs"
            :title="logLoading ? '正在读取日志…' : '暂无日志'"
            description="站点产生访问记录后，这里会显示 Nginx 的访问/错误日志。"
          />
        </div>
      </div>

      <!-- 网关 -->
      <div v-else class="stack">
        <div class="widget-grid">
          <UWidget title="网关状态" subtitle="upanel-gateway 容器" icon="network" :span="7">
            <template #actions>
              <UButton
                variant="primary"
                size="sm"
                icon="wrench"
                label="修复 / 重载"
                :loading="repairing"
                @click="repairGateway"
              />
            </template>
            <div class="stack">
              <div class="row">
                <UStatusDot :status="gatewayTone" :text="gatewayText" />
                <span class="badge">{{ gateway.container || 'upanel-gateway' }}</span>
              </div>
              <dl class="info-list">
                <div class="info-list__row">
                  <dt class="info-list__label">HTTP 端口</dt>
                  <dd class="info-list__value">{{ gateway.httpPort || '—' }}</dd>
                </div>
                <div class="info-list__row">
                  <dt class="info-list__label">HTTPS 端口</dt>
                  <dd class="info-list__value">{{ gateway.httpsPort || '—' }}</dd>
                </div>
                <div class="info-list__row">
                  <dt class="info-list__label">镜像</dt>
                  <dd class="info-list__value mono">{{ gateway.image || 'nginx:alpine' }}</dd>
                </div>
                <div class="info-list__row">
                  <dt class="info-list__label">站点配置</dt>
                  <dd class="info-list__value mono">data/gateway/conf.d/*.conf</dd>
                </div>
              </dl>
              <div v-if="gateway.error" class="alert alert--warning">
                <UIcon name="alert" :size="15" class="alert__icon" />
                <div class="alert__body">{{ gateway.error }}</div>
              </div>
            </div>
          </UWidget>

          <UWidget title="部署检查" subtitle="网站无法访问时按顺序排查" icon="check-circle" :span="5">
            <ol class="checklist" style="counter-reset: step">
              <li class="checklist__item">
                <span class="checklist__icon checklist__icon--warn"><UIcon name="globe" :size="13" /></span>
                <div class="checklist__body">
                  <p class="checklist__title">域名解析</p>
                  <p class="checklist__desc">确认 A 记录已指向服务器公网 IP</p>
                </div>
              </li>
              <li class="checklist__item">
                <span class="checklist__icon checklist__icon--warn"><UIcon name="network" :size="13" /></span>
                <div class="checklist__body">
                  <p class="checklist__title">80 / 443 放行</p>
                  <p class="checklist__desc">检查云安全组与服务器防火墙</p>
                </div>
              </li>
              <li class="checklist__item">
                <span class="checklist__icon checklist__icon--warn"><UIcon name="folder" :size="13" /></span>
                <div class="checklist__body">
                  <p class="checklist__title">站点文件</p>
                  <p class="checklist__desc">静态站点需要 index.html，PHP 需要 index.php</p>
                </div>
              </li>
              <li class="checklist__item">
                <span class="checklist__icon checklist__icon--warn"><UIcon name="logs" :size="13" /></span>
                <div class="checklist__body">
                  <p class="checklist__title">查看日志</p>
                  <p class="checklist__desc">在「站点日志」标签页查看 error.log</p>
                </div>
              </li>
            </ol>
          </UWidget>
        </div>
      </div>
    </UTabs>
    </div>

    <!-- 创建网站 -->
    <UDrawer v-model="createOpen" title="创建网站" description="站点创建后会自动生成默认首页并接入网关" size="lg" :busy="submitting" @close="clearActionQuery">
      <form id="create-site-form" class="stack" @submit.prevent="submitCreate">
        <div v-if="formError" class="alert alert--danger" role="alert">
          <UIcon name="alert" :size="16" class="alert__icon" />
          <div class="alert__body">{{ formError }}</div>
        </div>
        <div class="form-grid">
          <UField label="主域名" for-id="site-domain" required hint="例如 example.com 或 *.example.com">
            <UInput
              id="site-domain"
              v-model="createForm.domain"
              autofocus
              placeholder="example.com"
              autocomplete="off"
              :invalid="!!fieldError.domain"
            />
          </UField>
          <UField label="备注" for-id="site-remark" hint="可选，便于区分站点用途">
            <UInput id="site-remark" v-model="createForm.remark" placeholder="例如：官网 / 博客" />
          </UField>
          <UField class="field--full" label="别名域名" hint="多个域名用逗号或换行分隔，最多 20 个">
            <UTextarea v-model="createForm.aliasesText" :rows="2" placeholder="www.example.com, example.cn" />
          </UField>
        </div>

        <div class="field">
          <span class="field__label">站点类型</span>
          <div class="type-grid" role="group" aria-label="站点类型">
            <button
              v-for="type in typeOptions"
              :key="type.value"
              type="button"
              class="type-card"
              :aria-pressed="createForm.type === type.value ? 'true' : 'false'"
              @click="createForm.type = type.value"
            >
              <span class="type-card__title"><UIcon :name="type.icon" :size="15" /> {{ type.label }}</span>
              <span class="type-card__desc">{{ type.desc }}</span>
            </button>
          </div>
        </div>

        <div v-if="createForm.type === 'php'" class="form-grid">
          <UField label="PHP 版本" for-id="site-php">
            <UDropdownSelect
              id="site-php"
              v-model="createForm.phpVersion"
              :options="phpOptions"
              :searchable="false"
              aria-label="PHP 版本"
            />
          </UField>
        </div>

        <div v-if="createForm.type === 'proxy'" class="form-grid">
          <UField label="容器镜像" for-id="site-image" required hint="例如 node:20-alpine、nginx:alpine">
            <UInput id="site-image" v-model="createForm.image" placeholder="node:20-alpine" />
          </UField>
          <UField label="容器内端口" for-id="site-port" required hint="应用在容器内监听的端口">
            <UInput id="site-port" v-model.number="createForm.port" type="number" min="1" max="65535" placeholder="3000" />
          </UField>
        </div>

        <div class="alert alert--info">
          <UIcon name="info" :size="16" class="alert__icon" />
          <div class="alert__body">
            静态站点由网关直接托管文件；PHP 站点会启动 php-fpm 容器；反代站点会启动你指定的镜像并转发全部请求。
          </div>
        </div>
      </form>
      <template #footer>
        <UButton variant="secondary" @click="createOpen = false">取消</UButton>
        <UButton variant="primary" :loading="submitting" icon="plus" label="创建网站" @click="submitCreate" />
      </template>
    </UDrawer>

    <!-- 编辑网站 -->
    <UDrawer v-model="editOpen" :title="`编辑 ${editSite?.domain || ''}`" description="修改别名、备注与运行时参数" size="lg" :busy="submitting">
      <form id="edit-site-form" class="stack" @submit.prevent="submitEdit">
        <div v-if="formError" class="alert alert--danger" role="alert">
          <UIcon name="alert" :size="16" class="alert__icon" />
          <div class="alert__body">{{ formError }}</div>
        </div>
        <div class="form-grid">
          <UField class="field--full" label="别名域名" hint="多个域名用逗号或换行分隔">
            <UTextarea v-model="editForm.aliasesText" :rows="2" />
          </UField>
          <UField label="备注" for-id="edit-remark">
            <UInput id="edit-remark" v-model="editForm.remark" placeholder="可选" />
          </UField>
          <UField v-if="editSite?.type === 'php'" label="PHP 版本">
            <UDropdownSelect v-model="editForm.phpVersion" :options="phpOptions" :searchable="false" aria-label="PHP 版本" />
          </UField>
          <UField v-if="editSite?.type === 'proxy'" label="容器内端口">
            <UInput v-model.number="editForm.port" type="number" min="1" max="65535" />
          </UField>
        </div>
        <div class="alert alert--info">
          <UIcon name="info" :size="16" class="alert__icon" />
          <div class="alert__body">保存后网关会重新校验配置并热重载，已有连接不会中断。</div>
        </div>
      </form>
      <template #footer>
        <UButton variant="secondary" @click="editOpen = false">取消</UButton>
        <UButton variant="primary" :loading="submitting" label="保存修改" @click="submitEdit" />
      </template>
    </UDrawer>

    <!-- SSL -->
    <UDrawer v-model="sslOpen" :title="`${sslSite?.domain || ''} · SSL 证书`" size="lg" :busy="submitting">
      <form id="ssl-form" class="stack" @submit.prevent="submitSSL">
        <div v-if="formError" class="alert alert--danger" role="alert">
          <UIcon name="alert" :size="16" class="alert__icon" />
          <div class="alert__body">{{ formError }}</div>
        </div>
        <UField label="证书链（fullchain.pem）" hint="包含站点证书与中间证书" required>
          <UTextarea v-model="sslForm.fullchain" code :rows="7" placeholder="-----BEGIN CERTIFICATE-----" />
        </UField>
        <UField label="私钥（privkey.pem）" hint="仅保存在服务器 data/gateway/certs 下，权限 0600" required>
          <UTextarea v-model="sslForm.privkey" code :rows="7" placeholder="-----BEGIN PRIVATE KEY-----" />
        </UField>
        <USwitch v-model="sslForm.forceSSL" label="强制 HTTP 跳转到 HTTPS" />
      </form>
      <template #footer>
        <UButton variant="secondary" @click="sslOpen = false">取消</UButton>
        <UButton variant="primary" :loading="submitting" icon="ssl" label="部署证书" @click="submitSSL" />
      </template>
    </UDrawer>

    <!-- 删除网站 -->
    <UModal v-model="deleteOpen" title="删除网站" size="sm" :busy="submitting">
      <div class="stack">
        <p>即将删除站点 <strong>{{ deleteSite?.domain }}</strong> 的网关配置与运行容器。</p>
        <USwitch v-model="deleteKeepFiles" label="保留网站文件（推荐）" />
        <div class="alert" :class="deleteKeepFiles ? 'alert--info' : 'alert--danger'">
          <UIcon name="alert" :size="16" class="alert__icon" />
          <div class="alert__body">
            {{
              deleteKeepFiles
                ? '站点目录会保留，之后可以用同名域名重新创建。'
                : '网站目录、数据库文件等都会被永久删除，无法恢复。'
            }}
          </div>
        </div>
      </div>
      <template #footer>
        <UButton variant="secondary" @click="deleteOpen = false">取消</UButton>
        <UButton variant="danger" :loading="submitting" icon="trash" label="确认删除" @click="submitDelete" />
      </template>
    </UModal>
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
import UDrawer from '@/components/ui/UDrawer.vue'
import UModal from '@/components/ui/UModal.vue'
import UStatusDot from '@/components/ui/UStatusDot.vue'
import UTabs from '@/components/ui/UTabs.vue'
import UTextarea from '@/components/ui/UTextarea.vue'
import UWidget from '@/components/ui/UWidget.vue'
import USwitch from '@/components/ui/USwitch.vue'
import UEmpty from '@/components/ui/UEmpty.vue'
import { errText, gatewayApi, siteApi } from '@/api'
import { copyText, statusLabel } from '@/composables/useFormat'
import { toastError, toastSuccess, toastWarning } from '@/composables/useToast'

const route = useRoute()
const router = useRouter()

const sites = ref([])
const loading = ref(false)
const error = ref('')
const search = ref('')
const busyDomain = ref('')
const repairing = ref(false)
const gateway = reactive({ state: 'unknown', httpPort: '', httpsPort: '', image: '', container: '', error: '' })

const tabIds = ['sites', 'domains', 'ssl', 'logs', 'gateway']
const activeTab = ref(tabIds.includes(route.query.tab) ? route.query.tab : 'sites')

const tabs = computed(() => [
  { id: 'sites', label: '站点列表', count: sites.value.length },
  { id: 'domains', label: '域名与别名', count: sites.value.length },
  { id: 'ssl', label: 'SSL 证书', count: sites.value.filter((s) => s.sslEnabled).length },
  { id: 'logs', label: '站点日志', icon: 'logs' },
  { id: 'gateway', label: '网关状态', icon: 'network' },
])

const filteredSites = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return sites.value
  return sites.value.filter((site) => {
    const haystack = [site.domain, site.remark, ...(site.aliases || [])].join(' ').toLowerCase()
    return haystack.includes(q)
  })
})

const gatewayTone = computed(() => {
  if (gateway.state === 'running') return 'running'
  if (gateway.state === 'docker_unavailable') return 'error'
  return 'warning'
})
const gatewayText = computed(() => {
  if (gateway.state === 'running') return '网关运行中'
  if (gateway.state === 'not_installed') return '网关尚未创建'
  if (gateway.state === 'docker_unavailable') return 'Docker 未连接'
  return statusLabel(gateway.state)
})

// create
const createOpen = ref(false)
const createForm = reactive({ domain: '', aliasesText: '', type: 'static', phpVersion: '8.2', image: '', port: 80, remark: '' })
const fieldError = reactive({ domain: '' })
const formError = ref('')
const submitting = ref(false)

const typeOptions = [
  { value: 'static', label: '静态站点', icon: 'file', desc: '网关直接托管 HTML/CSS/JS' },
  { value: 'php', label: 'PHP 站点', icon: 'code', desc: 'php-fpm + fastcgi 转发' },
  { value: 'proxy', label: '反向代理', icon: 'network', desc: '任意镜像容器 + proxy_pass' },
]
const phpOptions = ['8.4', '8.3', '8.2', '8.1', '8.0', '7.4'].map((v) => ({ value: v, label: `PHP ${v}` }))

// edit
const editOpen = ref(false)
const editSite = ref(null)
const editForm = reactive({ aliasesText: '', remark: '', phpVersion: '', port: 80 })

// ssl
const sslOpen = ref(false)
const sslSite = ref(null)
const sslForm = reactive({ fullchain: '', privkey: '', forceSSL: false })

// delete
const deleteOpen = ref(false)
const deleteSite = ref(null)
const deleteKeepFiles = ref(true)

// logs
const logDomain = ref('')
const logKind = ref('access')
const logContent = ref('')
const logLoading = ref(false)
const logLines = 200
const logOptions = computed(() => sites.value.map((s) => ({ value: s.domain, label: s.domain })))
const logKindOptions = [
  { value: 'access', label: '访问日志' },
  { value: 'error', label: '错误日志' },
]

function typeLabel(type) {
  return { static: '静态', php: 'PHP', proxy: '反代' }[type] || type
}
function typeBadge(type) {
  return type === 'php' ? 'badge--info' : type === 'proxy' ? 'badge--primary' : ''
}
function siteUrl(site) {
  return `${site.sslEnabled ? 'https' : 'http'}://${site.domain}`
}

async function loadSites() {
  loading.value = true
  error.value = ''
  try {
    const res = await siteApi.list()
    sites.value = res.data || []
    if (!logDomain.value && sites.value.length > 0) logDomain.value = sites.value[0].domain
  } catch (e) {
    error.value = errText(e, '加载网站列表失败')
  } finally {
    loading.value = false
  }
}

async function loadGateway() {
  try {
    const res = await gatewayApi.status()
    Object.assign(gateway, { error: '' }, res.data || {})
  } catch (e) {
    gateway.state = 'docker_unavailable'
    gateway.error = errText(e)
  }
}

async function repairGateway() {
  repairing.value = true
  try {
    await gatewayApi.ensure()
    toastSuccess('网关已就绪')
    await loadGateway()
    await loadSites()
  } catch (e) {
    toastError('修复网关失败', errText(e))
  } finally {
    repairing.value = false
  }
}

function resetCreateForm() {
  createForm.domain = ''
  createForm.aliasesText = ''
  createForm.type = 'static'
  createForm.phpVersion = '8.2'
  createForm.image = ''
  createForm.port = 80
  createForm.remark = ''
  fieldError.domain = ''
  formError.value = ''
}

function openCreate() {
  resetCreateForm()
  createOpen.value = true
  if (route.query.action !== 'create') {
    router.replace({ path: route.path, query: { ...route.query, action: 'create' } })
  }
}

function clearActionQuery() {
  if (route.query.action === 'create') {
    const query = { ...route.query }
    delete query.action
    router.replace({ path: route.path, query })
  }
}

function parseAliases(text) {
  return text
    .split(/[\n,，\s]+/)
    .map((item) => item.trim())
    .filter(Boolean)
}

async function submitCreate() {
  if (submitting.value) return
  formError.value = ''
  fieldError.domain = ''
  if (!createForm.domain.trim()) {
    fieldError.domain = '请输入主域名'
    return
  }
  const payload = {
    domain: createForm.domain.trim(),
    aliases: parseAliases(createForm.aliasesText),
    type: createForm.type,
    remark: createForm.remark.trim(),
  }
  if (createForm.type === 'php') payload.phpVersion = createForm.phpVersion
  if (createForm.type === 'proxy') {
    payload.image = createForm.image.trim()
    payload.port = Number(createForm.port) || 80
  }
  submitting.value = true
  try {
    await siteApi.create(payload)
    toastSuccess('网站创建成功', `${payload.domain} 已接入网关`)
    createOpen.value = false
    clearActionQuery()
    await loadSites()
    activeTab.value = 'sites'
  } catch (e) {
    formError.value = errText(e, '创建网站失败')
  } finally {
    submitting.value = false
  }
}

function openEdit(site) {
  editSite.value = site
  editForm.aliasesText = (site.aliases || []).join('\n')
  editForm.remark = site.remark || ''
  editForm.phpVersion = site.phpVersion || '8.2'
  editForm.port = site.port || 80
  formError.value = ''
  editOpen.value = true
}

async function submitEdit() {
  if (submitting.value || !editSite.value) return
  formError.value = ''
  const payload = {
    aliases: parseAliases(editForm.aliasesText),
    remark: editForm.remark.trim(),
  }
  if (editSite.value.type === 'php') payload.phpVersion = editForm.phpVersion
  if (editSite.value.type === 'proxy') payload.port = Number(editForm.port) || 80
  submitting.value = true
  try {
    await siteApi.update(editSite.value.domain, payload)
    toastSuccess('网站已更新', '网关配置已热重载')
    editOpen.value = false
    await loadSites()
  } catch (e) {
    formError.value = errText(e, '更新网站失败')
  } finally {
    submitting.value = false
  }
}

function openSSL(site) {
  sslSite.value = site
  sslForm.fullchain = ''
  sslForm.privkey = ''
  sslForm.forceSSL = !!site.forceSSL
  formError.value = ''
  sslOpen.value = true
}

async function submitSSL() {
  if (submitting.value || !sslSite.value) return
  formError.value = ''
  if (!sslForm.fullchain.trim() || !sslForm.privkey.trim()) {
    formError.value = '证书和私钥都不能为空'
    return
  }
  submitting.value = true
  try {
    await siteApi.applySSL(sslSite.value.domain, {
      fullchain: sslForm.fullchain,
      privkey: sslForm.privkey,
      forceSSL: sslForm.forceSSL,
    })
    toastSuccess('证书已部署', sslSite.value.domain)
    sslOpen.value = false
    await loadSites()
  } catch (e) {
    formError.value = errText(e, '部署证书失败')
  } finally {
    submitting.value = false
  }
}

async function disableSSL(site) {
  try {
    await siteApi.disableSSL(site.domain)
    toastSuccess('SSL 已关闭', site.domain)
    await loadSites()
  } catch (e) {
    toastError('关闭 SSL 失败', errText(e))
  }
}

function openDelete(site) {
  deleteSite.value = site
  deleteKeepFiles.value = true
  deleteOpen.value = true
}

async function submitDelete() {
  if (submitting.value || !deleteSite.value) return
  submitting.value = true
  try {
    await siteApi.remove(deleteSite.value.domain, deleteKeepFiles.value)
    toastSuccess('网站已删除', deleteKeepFiles.value ? '网站文件已保留' : '站点目录已删除')
    deleteOpen.value = false
    await loadSites()
  } catch (e) {
    toastError('删除网站失败', errText(e))
  } finally {
    submitting.value = false
  }
}

async function toggleSite(site, action) {
  busyDomain.value = site.domain
  try {
    if (action === 'start') await siteApi.start(site.domain)
    else await siteApi.stop(site.domain)
    toastSuccess(action === 'start' ? '网站已启动' : '网站已停止', site.domain)
    await loadSites()
  } catch (e) {
    toastError('操作失败', errText(e))
  } finally {
    busyDomain.value = ''
  }
}

function openLogs(site) {
  logDomain.value = site.domain
  activeTab.value = 'logs'
  loadLogs()
}

async function loadLogs() {
  if (!logDomain.value) return
  logLoading.value = true
  logContent.value = ''
  try {
    const res = await siteApi.logs(logDomain.value, logKind.value, logLines)
    logContent.value = res.data || ''
  } catch (e) {
    toastError('读取日志失败', errText(e))
  } finally {
    logLoading.value = false
  }
}

async function copy(text) {
  try {
    await copyText(text)
    toastSuccess('已复制')
  } catch (e) {
    toastWarning('复制失败', '请手动选择文本复制')
  }
}

watch(activeTab, (tab) => {
  if (route.query.tab !== tab) {
    router.replace({ path: route.path, query: { ...route.query, tab } })
  }
  if (tab === 'logs' && logDomain.value) loadLogs()
  if (tab === 'gateway') loadGateway()
})

watch(
  () => route.query.tab,
  (tab) => {
    if (!tab) {
      activeTab.value = 'sites'
      return
    }
    if (tabIds.includes(tab) && tab !== activeTab.value) activeTab.value = tab
  }
)

watch(
  () => route.query.action,
  (action) => {
    if (action === 'create' && !createOpen.value) openCreate()
  }
)

onMounted(async () => {
  await Promise.all([loadSites(), loadGateway()])
  if (route.query.action === 'create') openCreate()
  if (activeTab.value === 'logs' && logDomain.value) loadLogs()
})
</script>
