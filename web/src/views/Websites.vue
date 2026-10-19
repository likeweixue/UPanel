<template>
  <div class="websites-container">
    <!-- 网关状态条 -->
    <div class="gateway-bar">
      <div class="gw-left">
        <span class="gw-title">网站网关</span>
        <span class="status-badge" :class="gateway.state === 'running' ? 'running' : 'stopped'">
          {{ gatewayStateText }}
        </span>
        <span class="gw-meta">容器 {{ gateway.container }} · HTTP {{ gateway.httpPort }}</span>
      </div>
      <div class="gw-right">
        <el-button size="small" @click="refreshAll">
          <el-icon><Refresh /></el-icon> 刷新
        </el-button>
        <el-button size="small" type="primary" plain @click="fixGateway" :loading="fixingGateway">
          修复网关
        </el-button>
      </div>
    </div>

    <el-alert
      v-if="gateway.state !== 'running'"
      type="warning"
      :closable="false"
      show-icon
      title="网关未运行"
      description="网站通过统一网关对外提供 80/443 服务。请点击「修复网关」，它会拉取镜像并启动容器；若端口被占用，请在「面板设置」中更换网关端口。"
      style="margin-bottom: 4px"
    />

    <!-- 顶部分类面板 -->
    <div class="top-panel">
      <div class="top-left">
        <el-button type="primary" size="small" @click="openCreateDrawer">
          <el-icon><Plus /></el-icon> 创建网站
        </el-button>
        <el-button
          v-if="selectedRows.length > 0"
          type="danger"
          size="small"
          plain
          @click="batchDelete"
        >
          批量删除 ({{ selectedRows.length }})
        </el-button>
        <el-button
          v-if="selectedRows.length > 0"
          type="success"
          size="small"
          plain
          @click="batchStart"
        >
          批量启动
        </el-button>
        <el-button
          v-if="selectedRows.length > 0"
          type="warning"
          size="small"
          plain
          @click="batchStop"
        >
          批量停止
        </el-button>
      </div>
      <div class="top-right">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索域名"
          prefix-icon="Search"
          clearable
          style="width: 200px"
          size="small"
        />
      </div>
    </div>

    <!-- 网站表格 -->
    <div class="websites-panel">
      <el-table
        v-loading="loading"
        :data="filteredWebsites"
        style="width: 100%"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" />
        <el-table-column prop="domain" label="域名" min-width="180">
          <template #default="{ row }">
            <div class="domain-cell">
              <span>{{ row.domain }}</span>
              <el-tag v-if="row.aliases && row.aliases.length" size="small" type="info">
                +{{ row.aliases.length }}
              </el-tag>
              <el-tag v-if="row.sslEnabled" size="small" type="success">SSL</el-tag>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="类型" width="130">
          <template #default="{ row }">
            <el-tag size="small" :type="typeTagType(row.type)">
              {{ typeLabel(row) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="根目录" min-width="200">
          <template #default="{ row }">
            <div class="path-cell">
              <span class="path-text">{{ row.rootDir }}</span>
              <el-icon class="copy-icon" @click="copyText(row.rootDir)"><DocumentCopy /></el-icon>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <span class="status-badge" :class="row.status">
              {{ row.status === 'running' ? '运行中' : '已停止' }}
            </span>
          </template>
        </el-table-column>

        <el-table-column prop="remark" label="备注" min-width="140">
          <template #default="{ row }">
            <span class="remark-cell" :class="{ 'remark-placeholder': !row.remark }" @click="openEditDrawer(row)">
              {{ row.remark || '点击添加备注' }}
            </span>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="260" fixed="right">
          <template #default="{ row }">
            <el-button
              size="small"
              :type="row.status === 'running' ? 'warning' : 'success'"
              @click="toggleWebsite(row)"
            >
              {{ row.status === 'running' ? '停止' : '启动' }}
            </el-button>
            <el-button size="small" @click="openWebsite(row)">
              <el-icon><Link /></el-icon> 访问
            </el-button>
            <el-dropdown @command="(cmd) => handleCommand(cmd, row)">
              <el-button size="small">
                更多 <el-icon class="el-icon--right"><ArrowDown /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="edit">编辑</el-dropdown-item>
                  <el-dropdown-item command="ssl">SSL 证书</el-dropdown-item>
                  <el-dropdown-item command="restart">重启</el-dropdown-item>
                  <el-dropdown-item command="delete" divided>删除</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
        </el-table-column>

        <template #empty>
          <div class="empty-tip">暂无网站，点击「创建网站」开始部署</div>
        </template>
      </el-table>
    </div>

    <!-- 创建网站抽屉 -->
    <el-drawer v-model="createDrawerVisible" title="创建网站" direction="rtl" size="460px">
      <div class="drawer-content">
        <el-alert
          type="info"
          :closable="false"
          show-icon
          title="域名请先解析到本机 IP"
          description="创建后把网页文件上传到站点根目录即可访问。"
          style="margin-bottom: 16px"
        />
        <el-form :model="createForm" label-width="100px" label-position="left">
          <el-form-item label="域名" required>
            <el-input v-model="createForm.domain" placeholder="例如: example.com" />
          </el-form-item>

          <el-form-item label="其他域名">
            <el-select
              v-model="createForm.aliases"
              multiple
              filterable
              allow-create
              default-first-option
              placeholder="回车添加，如 www.example.com"
              style="width: 100%"
            />
          </el-form-item>

          <el-form-item label="网站类型" required>
            <el-radio-group v-model="createForm.type">
              <el-radio-button value="static">静态</el-radio-button>
              <el-radio-button value="php">PHP</el-radio-button>
              <el-radio-button value="proxy">反向代理</el-radio-button>
            </el-radio-group>
            <div class="form-tip">{{ typeTip }}</div>
          </el-form-item>

          <el-form-item v-if="createForm.type === 'php'" label="PHP 版本">
            <el-select v-model="createForm.phpVersion" style="width: 100%">
              <el-option label="PHP 8.3" value="8.3" />
              <el-option label="PHP 8.2" value="8.2" />
              <el-option label="PHP 8.1" value="8.1" />
              <el-option label="PHP 8.0" value="8.0" />
              <el-option label="PHP 7.4" value="7.4" />
            </el-select>
          </el-form-item>

          <template v-if="createForm.type === 'proxy'">
            <el-form-item label="镜像" required>
              <el-input v-model="createForm.image" placeholder="例如: nginx:alpine" />
            </el-form-item>
            <el-form-item label="容器端口" required>
              <el-input-number v-model="createForm.port" :min="1" :max="65535" />
            </el-form-item>
          </template>

          <el-form-item label="备注">
            <el-input v-model="createForm.remark" type="textarea" :rows="2" placeholder="可选" />
          </el-form-item>
        </el-form>

        <div class="drawer-footer">
          <el-button size="small" @click="createDrawerVisible = false">取消</el-button>
          <el-button size="small" type="primary" @click="confirmCreate" :loading="creating">
            创建
          </el-button>
        </div>
      </div>
    </el-drawer>

    <!-- 编辑网站抽屉 -->
    <el-drawer
      v-model="editDrawerVisible"
      :title="`编辑网站 - ${currentEditSite ? currentEditSite.domain : ''}`"
      direction="rtl"
      size="460px"
    >
      <div class="drawer-content">
        <el-form :model="editForm" label-width="110px" label-position="left">
          <el-form-item label="主域名">
            <el-input v-model="editForm.domain" disabled />
          </el-form-item>

          <el-form-item label="其他域名">
            <el-select
              v-model="editForm.aliases"
              multiple
              filterable
              allow-create
              default-first-option
              placeholder="回车添加"
              style="width: 100%"
            />
          </el-form-item>

          <el-form-item v-if="editForm.type === 'php'" label="PHP 版本">
            <el-select v-model="editForm.phpVersion" style="width: 100%">
              <el-option label="PHP 8.3" value="8.3" />
              <el-option label="PHP 8.2" value="8.2" />
              <el-option label="PHP 8.1" value="8.1" />
              <el-option label="PHP 8.0" value="8.0" />
              <el-option label="PHP 7.4" value="7.4" />
            </el-select>
            <div class="form-tip">切换版本会重建 PHP 容器，站点文件不受影响</div>
          </el-form-item>

          <el-form-item v-if="editForm.type === 'proxy'" label="容器端口">
            <el-input-number v-model="editForm.port" :min="1" :max="65535" />
          </el-form-item>

          <el-form-item label="根目录">
            <el-input v-model="editForm.rootDir" disabled />
          </el-form-item>

          <el-form-item label="备注">
            <el-input v-model="editForm.remark" type="textarea" :rows="2" />
          </el-form-item>
        </el-form>

        <div class="drawer-footer">
          <el-button size="small" @click="editDrawerVisible = false">取消</el-button>
          <el-button size="small" type="primary" @click="confirmEdit" :loading="saving">保存</el-button>
        </div>
      </div>
    </el-drawer>

    <!-- SSL 抽屉 -->
    <el-drawer
      v-model="sslDrawerVisible"
      :title="`SSL 证书 - ${currentSslSite ? currentSslSite.domain : ''}`"
      direction="rtl"
      size="520px"
    >
      <div class="drawer-content">
        <el-form label-width="120px" label-position="left">
          <el-form-item label="证书 (PEM)">
            <el-input
              v-model="sslForm.fullchain"
              type="textarea"
              :rows="8"
              placeholder="-----BEGIN CERTIFICATE----- ..."
            />
          </el-form-item>
          <el-form-item label="私钥 (KEY)">
            <el-input
              v-model="sslForm.privkey"
              type="textarea"
              :rows="8"
              placeholder="-----BEGIN PRIVATE KEY----- ..."
            />
          </el-form-item>
          <el-form-item label="强制 HTTPS">
            <el-switch v-model="sslForm.forceSSL" />
          </el-form-item>
        </el-form>

        <div class="drawer-footer">
          <el-button
            v-if="currentSslSite && currentSslSite.sslEnabled"
            size="small"
            type="danger"
            plain
            @click="disableSSL"
          >关闭 SSL</el-button>
          <el-button size="small" @click="sslDrawerVisible = false">取消</el-button>
          <el-button size="small" type="primary" @click="confirmSSL" :loading="sslSaving">部署</el-button>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const websites = ref([])
const loading = ref(false)
const searchKeyword = ref('')
const selectedRows = ref([])

const gateway = reactive({ state: 'unknown', httpPort: '80', container: 'upanel-gateway' })
const fixingGateway = ref(false)

// ---------- 创建 ----------
const createDrawerVisible = ref(false)
const creating = ref(false)
const createForm = ref({
  domain: '',
  aliases: [],
  type: 'static',
  phpVersion: '8.2',
  image: '',
  port: 80,
  remark: ''
})

const typeTip = computed(() => {
  switch (createForm.value.type) {
    case 'php':
      return '启动 php-fpm 容器，网关通过 fastcgi 转发，适合 WordPress / Laravel 等'
    case 'proxy':
      return '启动指定镜像容器，网关反向代理到容器端口，适合 Node / Go / Java 应用'
    default:
      return '网关直接托管静态文件，不额外启动容器，最快最省资源'
  }
})

const gatewayStateText = computed(() => {
  switch (gateway.state) {
    case 'running': return '运行中'
    case 'not_installed': return '未安装'
    case 'unknown': return '未知'
    default: return gateway.state
  }
})

// ---------- 编辑 ----------
const editDrawerVisible = ref(false)
const saving = ref(false)
const currentEditSite = ref(null)
const editForm = ref({ domain: '', aliases: [], phpVersion: '', port: 0, rootDir: '', remark: '', type: '' })

// ---------- SSL ----------
const sslDrawerVisible = ref(false)
const sslSaving = ref(false)
const currentSslSite = ref(null)
const sslForm = ref({ fullchain: '', privkey: '', forceSSL: true })

const filteredWebsites = computed(() => {
  if (!searchKeyword.value) return websites.value
  const kw = searchKeyword.value.toLowerCase()
  return websites.value.filter(s => s.domain.toLowerCase().includes(kw))
})

const typeLabel = (row) => {
  if (row.type === 'php') return 'PHP ' + (row.phpVersion || '')
  if (row.type === 'proxy') return '反向代理'
  return '静态'
}

const typeTagType = (t) => {
  if (t === 'php') return 'warning'
  if (t === 'proxy') return 'success'
  return 'info'
}

// 统一提取后端错误信息
const errMsg = (err) => {
  return (err && err.response && err.response.data && err.response.data.error) || err.message || '请求失败'
}

// ---------- 数据加载 ----------
const loadWebsites = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/websites/')
    websites.value = (res.data && res.data.data) || []
  } catch (err) {
    ElMessage.error('加载网站列表失败: ' + errMsg(err))
  } finally {
    loading.value = false
  }
}

const loadGateway = async () => {
  try {
    const res = await axios.get('/api/gateway/status')
    Object.assign(gateway, (res.data && res.data.data) || {})
  } catch (err) {
    gateway.state = 'unknown'
  }
}

const refreshAll = async () => {
  await Promise.all([loadWebsites(), loadGateway()])
}

const fixGateway = async () => {
  fixingGateway.value = true
  try {
    await axios.post('/api/gateway/ensure')
    ElMessage.success('网关已就绪')
  } catch (err) {
    ElMessage.error('网关启动失败: ' + errMsg(err))
  } finally {
    fixingGateway.value = false
    await loadGateway()
  }
}

// ---------- 创建 ----------
const openCreateDrawer = () => {
  createForm.value = { domain: '', aliases: [], type: 'static', phpVersion: '8.2', image: '', port: 80, remark: '' }
  createDrawerVisible.value = true
}

const confirmCreate = async () => {
  const f = createForm.value
  if (!f.domain) {
    ElMessage.warning('请输入域名')
    return
  }
  if (f.type === 'proxy' && !f.image) {
    ElMessage.warning('反代类型必须填写镜像名称')
    return
  }

  creating.value = true
  try {
    await axios.post('/api/websites/', {
      domain: f.domain.trim(),
      aliases: f.aliases || [],
      type: f.type,
      phpVersion: f.type === 'php' ? f.phpVersion : '',
      image: f.type === 'proxy' ? f.image.trim() : '',
      port: f.type === 'proxy' ? f.port : 0,
      remark: f.remark
    })
    ElMessage.success('网站创建成功')
    createDrawerVisible.value = false
    await refreshAll()
  } catch (err) {
    ElMessage.error('创建失败: ' + errMsg(err))
  } finally {
    creating.value = false
  }
}

// ---------- 启停 ----------
const toggleWebsite = async (row) => {
  const action = row.status === 'running' ? 'stop' : 'start'
  try {
    await axios.post('/api/websites/' + encodeURIComponent(row.domain) + '/' + action)
    ElMessage.success(action === 'start' ? '网站已启动' : '网站已停止')
    await loadWebsites()
  } catch (err) {
    ElMessage.error('操作失败: ' + errMsg(err))
  }
}

const openWebsite = (row) => {
  if (row.status !== 'running') {
    ElMessage.warning('网站未运行，请先启动')
    return
  }
  window.open((row.sslEnabled ? 'https' : 'http') + '://' + row.domain, '_blank')
}

// ---------- 编辑 ----------
const openEditDrawer = (row) => {
  currentEditSite.value = row
  editForm.value = {
    domain: row.domain,
    aliases: [...(row.aliases || [])],
    phpVersion: row.phpVersion || '8.2',
    port: row.port || 80,
    rootDir: row.rootDir || row.path,
    remark: row.remark || '',
    type: row.type
  }
  editDrawerVisible.value = true
}

const confirmEdit = async () => {
  const domain = currentEditSite.value ? currentEditSite.value.domain : ''
  if (!domain) return
  saving.value = true
  try {
    await axios.put('/api/websites/' + encodeURIComponent(domain), {
      aliases: editForm.value.aliases,
      phpVersion: editForm.value.phpVersion,
      port: editForm.value.port,
      remark: editForm.value.remark,
      type: editForm.value.type
    })
    ElMessage.success('已保存')
    editDrawerVisible.value = false
    await loadWebsites()
  } catch (err) {
    ElMessage.error('保存失败: ' + errMsg(err))
  } finally {
    saving.value = false
  }
}

// ---------- SSL ----------
const openSslDrawer = (row) => {
  currentSslSite.value = row
  sslForm.value = { fullchain: '', privkey: '', forceSSL: true }
  sslDrawerVisible.value = true
}

const confirmSSL = async () => {
  const domain = currentSslSite.value ? currentSslSite.value.domain : ''
  if (!domain) return
  if (!sslForm.value.fullchain.trim() || !sslForm.value.privkey.trim()) {
    ElMessage.warning('证书和私钥不能为空')
    return
  }
  sslSaving.value = true
  try {
    await axios.post('/api/websites/' + encodeURIComponent(domain) + '/ssl', {
      fullchain: sslForm.value.fullchain,
      privkey: sslForm.value.privkey,
      forceSSL: sslForm.value.forceSSL
    })
    ElMessage.success('证书已部署')
    sslDrawerVisible.value = false
    await loadWebsites()
  } catch (err) {
    ElMessage.error('部署失败: ' + errMsg(err))
  } finally {
    sslSaving.value = false
  }
}

const disableSSL = async () => {
  const domain = currentSslSite.value ? currentSslSite.value.domain : ''
  if (!domain) return
  try {
    await axios.delete('/api/websites/' + encodeURIComponent(domain) + '/ssl')
    ElMessage.success('SSL 已关闭')
    sslDrawerVisible.value = false
    await loadWebsites()
  } catch (err) {
    ElMessage.error('操作失败: ' + errMsg(err))
  }
}

// ---------- 更多操作 ----------
const handleCommand = (command, row) => {
  switch (command) {
    case 'edit':
      openEditDrawer(row)
      break
    case 'ssl':
      openSslDrawer(row)
      break
    case 'restart':
      restartWebsite(row)
      break
    case 'delete':
      deleteWebsite(row)
      break
  }
}

const restartWebsite = async (row) => {
  try {
    await axios.post('/api/websites/' + encodeURIComponent(row.domain) + '/restart')
    ElMessage.success('网站已重启')
    await loadWebsites()
  } catch (err) {
    ElMessage.error('重启失败: ' + errMsg(err))
  }
}

const deleteWebsite = async (row) => {
  try {
    const res = await ElMessageBox.prompt(
      '删除网站 ' + row.domain + '。\\n如需保留站点文件，请输入 keep；直接确认则连同文件一起删除。',
      '删除确认',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        inputPlaceholder: '输入 keep 保留文件',
        inputValue: '',
        type: 'warning'
      }
    )
    const keepFiles = String((res && res.value) || '').trim().toLowerCase() === 'keep'
    await axios.delete('/api/websites/' + encodeURIComponent(row.domain), { params: { keepFiles } })
    ElMessage.success('网站已删除')
    await loadWebsites()
  } catch (err) {
    // 用户取消或输入框关闭时不做提示
    if (err === 'cancel') return
    ElMessage.error('删除失败: ' + errMsg(err))
  }
}

// ---------- 批量 ----------
const handleSelectionChange = (selection) => {
  selectedRows.value = selection
}

const batchStart = async () => {
  for (const row of selectedRows.value) {
    if (row.status !== 'running') {
      try {
        await axios.post('/api/websites/' + encodeURIComponent(row.domain) + '/start')
      } catch (e) { /* 继续处理下一个 */ }
    }
  }
  ElMessage.success('批量启动完成')
  await loadWebsites()
}

const batchStop = async () => {
  for (const row of selectedRows.value) {
    if (row.status === 'running') {
      try {
        await axios.post('/api/websites/' + encodeURIComponent(row.domain) + '/stop')
      } catch (e) { /* 继续处理下一个 */ }
    }
  }
  ElMessage.success('批量停止完成')
  await loadWebsites()
}

const batchDelete = async () => {
  try {
    await ElMessageBox.confirm('确定删除选中的 ' + selectedRows.value.length + ' 个网站吗？（站点文件将一并删除）', '警告', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch (err) {
    return
  }
  for (const row of selectedRows.value) {
    try {
      await axios.delete('/api/websites/' + encodeURIComponent(row.domain))
    } catch (e) { /* 继续处理下一个 */ }
  }
  ElMessage.success('批量删除完成')
  await loadWebsites()
}

const copyText = async (text) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制: ' + text)
  } catch (err) {
    ElMessage.info(text)
  }
}

onMounted(() => {
  refreshAll()
})
</script>

<style scoped>
.websites-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.gateway-bar {
  background: #ffffff;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  padding: 10px 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.gw-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.gw-title {
  font-weight: 600;
  font-size: 14px;
  color: #303133;
}

.gw-meta {
  font-size: 12px;
  color: #909399;
}

.gw-right {
  display: flex;
  gap: 8px;
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
  flex-wrap: wrap;
}

.websites-panel {
  background: #ffffff;
  border-radius: 4px;
  padding: 8px;
  flex: 1;
  overflow: auto;
}

.domain-cell {
  display: flex;
  align-items: center;
  gap: 6px;
}

.path-cell {
  display: flex;
  align-items: center;
  gap: 6px;
}

.path-text {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 12px;
  color: #606266;
}

.copy-icon {
  cursor: pointer;
  color: #909399;
}

.copy-icon:hover {
  color: #409eff;
}

.status-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 12px;
  line-height: 18px;
}

.status-badge.running {
  background: #f0f9eb;
  color: #67c23a;
}

.status-badge.stopped,
.status-badge.not_installed,
.status-badge.unknown {
  background: #f4f4f5;
  color: #909399;
}

.remark-cell {
  cursor: pointer;
}

.remark-placeholder {
  color: #c0c4cc;
}

.empty-tip {
  padding: 32px 0;
  color: #909399;
  font-size: 13px;
}

.drawer-content {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.drawer-footer {
  margin-top: auto;
  padding-top: 16px;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  border-top: 1px solid #ebeef5;
}

.form-tip {
  margin-top: 4px;
  font-size: 12px;
  color: #909399;
  line-height: 1.5;
}
</style>
