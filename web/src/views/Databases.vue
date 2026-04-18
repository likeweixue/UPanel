<template>
  <div class="databases-container">
    <!-- 顶部分类面板 -->
    <div class="top-panel">
      <div class="top-left">
        <el-button type="primary" @click="openCreateDrawer">
          <el-icon><Plus /></el-icon> 创建数据库
        </el-button>
        <el-button 
          v-if="selectedRows.length > 0" 
          type="danger" 
          plain
          @click="batchDelete"
        >
          批量删除 ({{ selectedRows.length }})
        </el-button>
      </div>
      <div class="top-right">
        <el-tabs v-model="activeTab" class="db-tabs">
          <el-tab-pane label="MySQL" name="mysql" />
          <el-tab-pane label="PostgreSQL" name="postgres" />
          <el-tab-pane label="Redis" name="redis" />
          <el-tab-pane label="MongoDB" name="mongodb" />
        </el-tabs>
        <el-input 
          v-model="searchKeyword" 
          placeholder="搜索数据库" 
          prefix-icon="Search"
          clearable
          style="width: 200px; margin-left: 16px;"
        />
      </div>
    </div>

    <!-- 数据库表格 -->
    <div class="databases-panel">
      <el-table 
        :data="filteredDatabases" 
        
        style="width: 100%"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" />
        <el-table-column prop="name" label="数据库名" min-width="180" />
        <el-table-column prop="type" label="类型" width="100">
          <template #default="{ row }">
            <el-tag :type="getTypeTag(row.type)">{{ row.type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="version" label="版本" width="100" />
        <el-table-column prop="port" label="端口" width="80" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <span class="status-badge" :class="row.status">
              {{ row.status === 'running' ? '运行中' : '已停止' }}
            </span>
          </template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="150">
          <template #default="{ row }">
            <span 
              class="remark-cell" 
              :class="{ 'remark-placeholder': !row.remark }"
              @click="editRemark(row)"
            >
              {{ row.remark || '点击添加备注' }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="260" fixed="right">
          <template #default="{ row }">
            <el-button 
              size="small" 
              :type="row.status === 'running' ? 'warning' : 'success'"
              @click="toggleDatabase(row)"
            >
              {{ row.status === 'running' ? '停止' : '启动' }}
            </el-button>
            <el-button size="small" @click="openConnectionInfo(row)">
              <el-icon><Connection /></el-icon> 连接
            </el-button>
            <el-dropdown @command="(cmd) => handleCommand(cmd, row)">
              <el-button size="small">
                更多 <el-icon class="el-icon--right"><ArrowDown /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="edit">编辑</el-dropdown-item>
                  <el-dropdown-item command="password">修改密码</el-dropdown-item>
                  <el-dropdown-item command="backup">备份</el-dropdown-item>
                  <el-dropdown-item command="phpmyadmin">phpMyAdmin</el-dropdown-item>
                  <el-dropdown-item command="delete" divided>删除</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 创建数据库抽屉 -->
    <el-drawer
      v-model="createDrawerVisible"
      title="创建数据库"
      direction="rtl"
      size="450px"
    >
      <div class="drawer-content">
        <el-form :model="createForm" label-width="100px" label-position="left">
          <el-form-item label="数据库名" required>
            <el-input v-model="createForm.name" placeholder="例如: mydb" />
            <div class="form-tip">只支持字母、数字、下划线</div>
          </el-form-item>

          <el-form-item label="类型" required>
            <el-select v-model="createForm.type" placeholder="请选择类型">
              <el-option label="MySQL" value="mysql" />
              <el-option label="PostgreSQL" value="postgres" />
              <el-option label="Redis" value="redis" />
              <el-option label="MongoDB" value="mongodb" />
            </el-select>
          </el-form-item>

          <el-form-item label="版本">
            <el-select v-model="createForm.version" placeholder="请选择版本">
              <template v-if="createForm.type === 'mysql'">
                <el-option label="MySQL 8.0" value="8.0" />
                <el-option label="MySQL 5.7" value="5.7" />
              </template>
              <template v-else-if="createForm.type === 'postgres'">
                <el-option label="PostgreSQL 15" value="15" />
                <el-option label="PostgreSQL 14" value="14" />
              </template>
              <template v-else-if="createForm.type === 'redis'">
                <el-option label="Redis 7.2" value="7.2" />
                <el-option label="Redis 6.2" value="6.2" />
              </template>
              <template v-else>
                <el-option label="MongoDB 7.0" value="7.0" />
                <el-option label="MongoDB 6.0" value="6.0" />
              </template>
            </el-select>
          </el-form-item>

          <el-form-item label="端口">
            <el-input-number v-model="createForm.port" :min="1024" :max="65535" placeholder="自动分配" />
            <div class="form-tip">留空则使用默认端口</div>
          </el-form-item>

          <el-form-item label="密码">
            <el-input v-model="createForm.password" type="password" placeholder="留空则自动生成" />
          </el-form-item>

          <el-form-item label="备注">
            <el-input 
              v-model="createForm.remark" 
              type="textarea" 
              :rows="3"
              placeholder="可选，添加数据库备注信息"
            />
          </el-form-item>
        </el-form>

        <div class="drawer-footer">
          <el-button @click="createDrawerVisible = false">取消</el-button>
          <el-button type="primary" @click="confirmCreate" :loading="creating">
            创建
          </el-button>
        </div>
      </div>
    </el-drawer>

    <!-- 连接信息抽屉 -->
    <el-drawer
      v-model="connectionDrawerVisible"
      title="连接信息"
      direction="rtl"
      size="400px"
    >
      <div class="drawer-content">
        <div class="connection-info">
          <div class="info-item">
            <span class="info-label">主机</span>
            <span class="info-value">{{ currentConnection?.host }}</span>
            <el-button size="small" @click="copyToClipboard(currentConnection?.host)">复制</el-button>
          </div>
          <div class="info-item">
            <span class="info-label">端口</span>
            <span class="info-value">{{ currentConnection?.port }}</span>
            <el-button size="small" @click="copyToClipboard(currentConnection?.port)">复制</el-button>
          </div>
          <div class="info-item">
            <span class="info-label">用户名</span>
            <span class="info-value">{{ currentConnection?.username }}</span>
            <el-button size="small" @click="copyToClipboard(currentConnection?.username)">复制</el-button>
          </div>
          <div class="info-item">
            <span class="info-label">密码</span>
            <span class="info-value">********</span>
            <el-button size="small" @click="showPassword = !showPassword">
              {{ showPassword ? '隐藏' : '显示' }}
            </el-button>
            <el-button size="small" @click="copyToClipboard(currentConnection?.password)">复制</el-button>
          </div>
          <div v-if="showPassword" class="password-value">
            密码: {{ currentConnection?.password }}
          </div>
          <div class="info-item">
            <span class="info-label">连接串</span>
            <span class="info-value connection-string">{{ currentConnection?.connectionString }}</span>
            <el-button size="small" @click="copyToClipboard(currentConnection?.connectionString)">复制</el-button>
          </div>
        </div>
      </div>
    </el-drawer>

    <!-- 修改密码抽屉 -->
    <el-drawer
      v-model="passwordDrawerVisible"
      title="修改密码"
      direction="rtl"
      size="400px"
    >
      <div class="drawer-content">
        <el-form :model="passwordForm" label-width="100px">
          <el-form-item label="数据库名">
            <span>{{ currentDatabase?.name }}</span>
          </el-form-item>
          <el-form-item label="新密码" required>
            <el-input v-model="passwordForm.newPassword" type="password" />
          </el-form-item>
          <el-form-item label="确认密码" required>
            <el-input v-model="passwordForm.confirmPassword" type="password" />
          </el-form-item>
        </el-form>

        <div class="drawer-footer">
          <el-button @click="passwordDrawerVisible = false">取消</el-button>
          <el-button type="primary" @click="confirmChangePassword" :loading="changingPassword">
            确认修改
          </el-button>
        </div>
      </div>
    </el-drawer>


    <!-- 编辑数据库抽屉 -->
    <el-drawer
      v-model="editDrawerVisible"
      :title="`编辑数据库 - ${currentEditDb?.name}`"
      direction="rtl"
      size="550px"
    >
      <div class="drawer-content">
        <el-tabs v-model="editActiveTab">
          <el-tab-pane label="基础设置" name="basic">
            <el-form :model="editForm" label-width="100px" label-position="left">
              <el-form-item label="数据库名">
                <el-input v-model="editForm.name" disabled />
              </el-form-item>
              <el-form-item label="类型">
                <el-tag :type="getTypeTag(editForm.type)">{{ editForm.type }}</el-tag>
              </el-form-item>
              <el-form-item label="版本">
                <el-select v-model="editForm.version">
                  <template v-if="editForm.type === mysql">
                    <el-option label="MySQL 8.0" value="8.0" />
                    <el-option label="MySQL 5.7" value="5.7" />
                  </template>
                  <template v-else-if="editForm.type === postgres">
                    <el-option label="PostgreSQL 15" value="15" />
                    <el-option label="PostgreSQL 14" value="14" />
                  </template>
                  <template v-else-if="editForm.type === redis">
                    <el-option label="Redis 7.2" value="7.2" />
                    <el-option label="Redis 6.2" value="6.2" />
                  </template>
                  <template v-else>
                    <el-option label="MongoDB 7.0" value="7.0" />
                    <el-option label="MongoDB 6.0" value="6.0" />
                  </template>
                </el-select>
              </el-form-item>
              <el-form-item label="端口">
                <el-input-number v-model="editForm.port" :min="1024" :max="65535" />
              </el-form-item>
              <el-form-item label="字符集" v-if="editForm.type === mysql">
                <el-select v-model="editForm.charset">
                  <el-option label="utf8mb4 (推荐)" value="utf8mb4" />
                  <el-option label="utf8" value="utf8" />
                  <el-option label="latin1" value="latin1" />
                </el-select>
              </el-form-item>
              <el-form-item label="排序规则" v-if="editForm.type === mysql">
                <el-select v-model="editForm.collation">
                  <el-option label="utf8mb4_unicode_ci" value="utf8mb4_unicode_ci" />
                  <el-option label="utf8mb4_general_ci" value="utf8mb4_general_ci" />
                </el-select>
              </el-form-item>
            </el-form>
          </el-tab-pane>

          <el-tab-pane label="性能设置" name="performance">
            <el-form :model="editForm" label-width="120px">
              <el-form-item label="最大连接数">
                <el-input-number v-model="editForm.maxConnections" :min="10" :max="10000" />
                <div class="form-tip">默认 151，建议根据服务器配置调整</div>
              </el-form-item>
              <el-form-item label="查询缓存" v-if="editForm.type === mysql">
                <el-switch v-model="editForm.queryCache" />
              </el-form-item>
              <el-form-item label="缓存大小" v-if="editForm.type === mysql && editForm.queryCache">
                <el-input-number v-model="editForm.queryCacheSize" :min="1" :max="1024" /> MB
              </el-form-item>
              <el-form-item label="慢查询日志">
                <el-switch v-model="editForm.slowLog" />
              </el-form-item>
              <el-form-item label="慢查询阈值" v-if="editForm.slowLog">
                <el-input-number v-model="editForm.slowLogTime" :min="0" :max="60" /> 秒
              </el-form-item>
            </el-form>
          </el-tab-pane>

          <el-tab-pane label="安全设置" name="security">
            <el-form :model="editForm" label-width="120px">
              <el-form-item label="绑定IP">
                <el-input v-model="editForm.bindIp" placeholder="0.0.0.0 表示所有IP" />
                <div class="form-tip">多个IP用逗号分隔</div>
              </el-form-item>
              <el-form-item label="SSL加密">
                <el-switch v-model="editForm.sslEnabled" />
              </el-form-item>
              <el-form-item label="允许远程访问">
                <el-switch v-model="editForm.remoteAccess" />
              </el-form-item>
              <el-form-item label="最大包大小" v-if="editForm.type === mysql">
                <el-input-number v-model="editForm.maxPacketSize" :min="1" :max="1024" /> MB
              </el-form-item>
            </el-form>
          </el-tab-pane>

          <el-tab-pane label="备份策略" name="backup">
            <el-form :model="editForm" label-width="120px">
              <el-form-item label="自动备份">
                <el-switch v-model="editForm.autoBackup" />
              </el-form-item>
              <el-form-item label="备份周期" v-if="editForm.autoBackup">
                <el-select v-model="editForm.backupSchedule">
                  <el-option label="每天" value="daily" />
                  <el-option label="每周" value="weekly" />
                  <el-option label="每月" value="monthly" />
                </el-select>
              </el-form-item>
              <el-form-item label="保留份数" v-if="editForm.autoBackup">
                <el-input-number v-model="editForm.backupRetention" :min="1" :max="30" />
              </el-form-item>
              <el-form-item label="备份时间" v-if="editForm.autoBackup">
                <el-time-picker v-model="editForm.backupTime" format="HH:mm" />
              </el-form-item>
            </el-form>
          </el-tab-pane>

          <el-tab-pane label="高级设置" name="advanced">
            <el-form :model="editForm" label-width="120px">
              <el-form-item label="时区">
                <el-select v-model="editForm.timezone">
                  <el-option label="Asia/Shanghai" value="Asia/Shanghai" />
                  <el-option label="UTC" value="UTC" />
                  <el-option label="America/New_York" value="America/New_York" />
                </el-select>
              </el-form-item>
              <el-form-item label="日志级别">
                <el-select v-model="editForm.logLevel">
                  <el-option label="ERROR" value="error" />
                  <el-option label="WARNING" value="warning" />
                  <el-option label="INFO" value="info" />
                  <el-option label="DEBUG" value="debug" />
                </el-select>
              </el-form-item>
              <el-form-item label="内存限制" v-if="editForm.type === redis">
                <el-input-number v-model="editForm.maxMemory" :min="64" :max="16384" /> MB
                <div class="form-tip">Redis 最大可用内存</div>
              </el-form-item>
              <el-form-item label="淘汰策略" v-if="editForm.type === redis">
                <el-select v-model="editForm.evictionPolicy">
                  <el-option label="noeviction" value="noeviction" />
                  <el-option label="allkeys-lru" value="allkeys-lru" />
                  <el-option label="volatile-lru" value="volatile-lru" />
                  <el-option label="allkeys-random" value="allkeys-random" />
                </el-select>
              </el-form-item>
            </el-form>
          </el-tab-pane>
        </el-tabs>

        <div class="drawer-footer">
          <el-button @click="editDrawerVisible = false">取消</el-button>
          <el-button type="primary" @click="saveEdit" :loading="savingEdit">
            保存设置
          </el-button>
        </div>
      </div>
    </el-drawer>
    <!-- 备份抽屉 -->
    <el-drawer
      v-model="backupDrawerVisible"
      title="数据库备份"
      direction="rtl"
      size="400px"
    >
      <div class="drawer-content">
        <div class="backup-info">
          <div class="backup-site">{{ currentBackupDb?.name }}</div>
          <div class="backup-path">{{ currentBackupDb?.type }} 数据库</div>
        </div>
        
        <el-form label-width="100px" label-position="left">
          <el-form-item label="备份名称">
            <el-input v-model="backupName" placeholder="留空则自动生成" />
          </el-form-item>
          <el-form-item label="备份内容">
            <el-radio-group v-model="backupType">
              <el-radio value="full">完整备份</el-radio>
              <el-radio value="structure">仅结构</el-radio>
              <el-radio value="data">仅数据</el-radio>
            </el-radio-group>
          </el-form-item>
        </el-form>

        <div class="backup-list" v-if="backupList.length">
          <div class="backup-list-title">已有备份</div>
          <div class="backup-items">
            <div v-for="item in backupList" :key="item.id" class="backup-item">
              <div class="backup-item-info">
                <div class="backup-item-name">{{ item.name }}</div>
                <div class="backup-item-time">{{ item.time }}</div>
                <div class="backup-item-size">{{ item.size }}</div>
              </div>
              <div class="backup-item-actions">
                <el-button size="small" @click="downloadBackup(item)">下载</el-button>
                <el-button size="small" type="danger" @click="deleteBackup(item)">删除</el-button>
                <el-button size="small" @click="restoreBackup(item)">恢复</el-button>
              </div>
            </div>
          </div>
        </div>

        <div class="drawer-footer">
          <el-button @click="backupDrawerVisible = false">取消</el-button>
          <el-button type="primary" @click="confirmBackup" :loading="backingUp">
            开始备份
          </el-button>
        </div>
      </div>
    </el-drawer>

    <!-- 备注编辑对话框 -->
    <el-dialog v-model="remarkDialogVisible" title="编辑备注" width="400px">
      <el-input 
        v-model="editingRemark" 
        type="textarea" 
        :rows="4"
        placeholder="请输入备注信息"
      />
      <template #footer>
        <el-button @click="remarkDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveRemark">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, ArrowDown, Connection } from '@element-plus/icons-vue'

const searchKeyword = ref('')
const activeTab = ref('mysql')
const createDrawerVisible = ref(false)
const connectionDrawerVisible = ref(false)
const passwordDrawerVisible = ref(false)
const backupDrawerVisible = ref(false)
const creating = ref(false)
const changingPassword = ref(false)
const backingUp = ref(false)
const remarkDialogVisible = ref(false)
const editingRemark = ref('')
const showPassword = ref(false)
const currentDatabase = ref(null)
const currentConnection = ref(null)
const currentBackupDb = ref(null)
const selectedRows = ref([])
const editDrawerVisible = ref(false)
const savingEdit = ref(false)
const currentEditDb = ref(null)
const editActiveTab = ref('basic')

const editForm = ref({
  name: '',
  type: '',
  version: '',
  port: null,
  charset: 'utf8mb4',
  collation: 'utf8mb4_unicode_ci',
  maxConnections: 151,
  queryCache: true,
  queryCacheSize: 16,
  slowLog: false,
  slowLogTime: 10,
  bindIp: '0.0.0.0',
  sslEnabled: false,
  remoteAccess: false,
  maxPacketSize: 16,
  autoBackup: false,
  backupSchedule: 'daily',
  backupRetention: 7,
  backupTime: new Date(2024, 0, 1, 2, 0),
  timezone: 'Asia/Shanghai',
  logLevel: 'info',
  maxMemory: 1024,
  evictionPolicy: 'noeviction'
})

// 备份相关
const backupType = ref('full')
const backupName = ref('')
const backupList = ref([])

// 密码表单
const passwordForm = ref({
  newPassword: '',
  confirmPassword: ''
})

// 创建表单
const createForm = ref({
  name: '',
  type: 'mysql',
  version: '8.0',
  port: null,
  password: '',
  remark: ''
})

// 数据库列表
const databases = ref([
  {
    id: 1,
    name: 'wordpress_db',
    type: 'mysql',
    version: '8.0',
    port: 3306,
    status: 'running',
    remark: 'WordPress 数据库',
    username: 'root',
    password: 'password123',
    createdAt: '2024-01-15'
  },
  {
    id: 2,
    name: 'app_db',
    type: 'postgres',
    version: '15',
    port: 5432,
    status: 'running',
    remark: '',
    username: 'postgres',
    password: 'password456',
    createdAt: '2024-01-20'
  },
  {
    id: 3,
    name: 'cache_redis',
    type: 'redis',
    version: '7.2',
    port: 6379,
    status: 'stopped',
    remark: '缓存服务',
    username: '',
    password: '',
    createdAt: '2024-02-01'
  },
  {
    id: 4,
    name: 'logs_mongo',
    type: 'mongodb',
    version: '7.0',
    port: 27017,
    status: 'running',
    remark: '日志存储',
    username: 'root',
    password: 'password789',
    createdAt: '2024-02-10'
  }
])

// 过滤数据库
const filteredDatabases = computed(() => {
  let result = databases.value.filter(db => db.type === activeTab.value)
  
  if (searchKeyword.value) {
    result = result.filter(db => 
      db.name.toLowerCase().includes(searchKeyword.value.toLowerCase())
    )
  }
  
  return result
})

// 获取类型标签样式
const getTypeTag = (type) => {
  const types = {
    mysql: 'primary',
    postgres: 'success',
    redis: 'warning',
    mongodb: 'info'
  }
  return types[type] || 'primary'
}

const handleSelectionChange = (selection) => {
  selectedRows.value = selection
}

const batchDelete = async () => {
  await ElMessageBox.confirm(`确定删除选中的 ${selectedRows.value.length} 个数据库吗？`, '警告', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  })
  
  selectedRows.value.forEach(row => {
    const index = databases.value.findIndex(db => db.id === row.id)
    if (index !== -1) {
      databases.value.splice(index, 1)
    }
  })
  ElMessage.success('批量删除成功')
}

const openCreateDrawer = () => {
  createForm.value = {
    name: '',
    type: activeTab.value,
    version: activeTab.value === 'mysql' ? '8.0' : 
             activeTab.value === 'postgres' ? '15' :
             activeTab.value === 'redis' ? '7.2' : '7.0',
    port: null,
    password: '',
    remark: ''
  }
  createDrawerVisible.value = true
}

const confirmCreate = async () => {
  if (!createForm.value.name) {
    ElMessage.warning('请输入数据库名')
    return
  }
  
  creating.value = true
  try {
    await new Promise(resolve => setTimeout(resolve, 1000))
    const newDb = {
      id: Date.now(),
      name: createForm.value.name,
      type: createForm.value.type,
      version: createForm.value.version,
      port: createForm.value.port || getDefaultPort(createForm.value.type),
      status: 'running',
      remark: createForm.value.remark || '',
      username: createForm.value.type === 'redis' ? '' : 'root',
      password: createForm.value.password || generatePassword(),
      createdAt: new Date().toISOString().split('T')[0]
    }
    databases.value.unshift(newDb)
    ElMessage.success('数据库创建成功')
    createDrawerVisible.value = false
  } catch (err) {
    ElMessage.error('创建失败: ' + err.message)
  } finally {
    creating.value = false
  }
}

const getDefaultPort = (type) => {
  const ports = {
    mysql: 3306,
    postgres: 5432,
    redis: 6379,
    mongodb: 27017
  }
  return ports[type] || 3306
}

const generatePassword = () => {
  return Math.random().toString(36).slice(-12)
}

const toggleDatabase = async (db) => {
  try {
    await new Promise(resolve => setTimeout(resolve, 500))
    db.status = db.status === 'running' ? 'stopped' : 'running'
    ElMessage.success(`数据库已${db.status === 'running' ? '启动' : '停止'}`)
  } catch (err) {
    ElMessage.error('操作失败: ' + err.message)
  }
}

const openConnectionInfo = (db) => {
  const connectionString = getConnectionString(db)
  currentConnection.value = {
    host: 'localhost',
    port: db.port,
    username: db.username || 'root',
    password: db.password || '',
    connectionString: connectionString
  }
  showPassword.value = false
  connectionDrawerVisible.value = true
}

const getConnectionString = (db) => {
  switch (db.type) {
    case 'mysql':
      return `mysql -h localhost -P ${db.port} -u ${db.username || 'root'} -p`
    case 'postgres':
      return `psql -h localhost -p ${db.port} -U ${db.username || 'postgres'} -d ${db.name}`
    case 'redis':
      return `redis-cli -h localhost -p ${db.port}`
    case 'mongodb':
      return `mongosh --host localhost --port ${db.port} -u ${db.username || 'root'} -p`
    default:
      return ''
  }
}

const copyToClipboard = (text) => {
  if (!text) return
  navigator.clipboard.writeText(text)
  ElMessage.success('已复制到剪贴板')
}

const openChangePassword = (db) => {
  currentDatabase.value = db
  passwordForm.value = { newPassword: '', confirmPassword: '' }
  passwordDrawerVisible.value = true
}

const confirmChangePassword = async () => {
  if (!passwordForm.value.newPassword) {
    ElMessage.warning('请输入新密码')
    return
  }
  if (passwordForm.value.newPassword !== passwordForm.value.confirmPassword) {
    ElMessage.warning('两次输入的密码不一致')
    return
  }
  
  changingPassword.value = true
  try {
    await new Promise(resolve => setTimeout(resolve, 1000))
    if (currentDatabase.value) {
      currentDatabase.value.password = passwordForm.value.newPassword
    }
    ElMessage.success('密码修改成功')
    passwordDrawerVisible.value = false
  } catch (err) {
    ElMessage.error('修改失败: ' + err.message)
  } finally {
    changingPassword.value = false
  }
}

const openBackupDrawer = (db) => {
  currentBackupDb.value = db
  backupType.value = 'full'
  backupName.value = ''
  backupList.value = [
    { id: 1, name: 'backup_20240115.sql.gz', time: '2024-01-15 10:30', size: '2.3 MB' },
    { id: 2, name: 'backup_20240120.sql.gz', time: '2024-01-20 14:20', size: '2.1 MB' }
  ]
  backupDrawerVisible.value = true
}

const confirmBackup = async () => {
  backingUp.value = true
  try {
    await new Promise(resolve => setTimeout(resolve, 2000))
    ElMessage.success('备份创建成功')
    backupDrawerVisible.value = false
  } catch (err) {
    ElMessage.error('备份失败: ' + err.message)
  } finally {
    backingUp.value = false
  }
}

const downloadBackup = (item) => {
  ElMessage.info(`下载 ${item.name}`)
}

const deleteBackup = async (item) => {
  await ElMessageBox.confirm(`确定删除备份 ${item.name} 吗？`, '警告', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  })
  const index = backupList.value.findIndex(b => b.id === item.id)
  if (index !== -1) {
    backupList.value.splice(index, 1)
    ElMessage.success('备份已删除')
  }
}

const restoreBackup = (item) => {
  ElMessage.info(`恢复 ${item.name}`)
}

const openEditDrawer = (db) => {
  currentEditDb.value = db
  editForm.value = {
    name: db.name,
    type: db.type,
    version: db.version,
    port: db.port,
    charset: 'utf8mb4',
    collation: 'utf8mb4_unicode_ci',
    maxConnections: 151,
    queryCache: true,
    queryCacheSize: 16,
    slowLog: false,
    slowLogTime: 10,
    bindIp: '0.0.0.0',
    sslEnabled: false,
    remoteAccess: false,
    maxPacketSize: 16,
    autoBackup: false,
    backupSchedule: 'daily',
    backupRetention: 7,
    backupTime: new Date(2024, 0, 1, 2, 0),
    timezone: 'Asia/Shanghai',
    logLevel: 'info',
    maxMemory: 1024,
    evictionPolicy: 'noeviction'
  }
  editActiveTab.value = 'basic'
  editDrawerVisible.value = true
}

const saveEdit = async () => {
  savingEdit.value = true
  try {
    await new Promise(resolve => setTimeout(resolve, 1000))
    if (currentEditDb.value) {
      currentEditDb.value.version = editForm.value.version
      currentEditDb.value.port = editForm.value.port
    }
    ElMessage.success('保存成功')
    editDrawerVisible.value = false
  } catch (err) {
    ElMessage.error('保存失败: ' + err.message)
  } finally {
    savingEdit.value = false
  }
}
const editRemark = (db) => {
  currentDatabase.value = db
  editingRemark.value = db.remark || ''
  remarkDialogVisible.value = true
}

const saveRemark = () => {
  if (currentDatabase.value) {
    currentDatabase.value.remark = editingRemark.value
    ElMessage.success('备注已更新')
  }
  remarkDialogVisible.value = false
}

const handleCommand = (command, db) => {
  switch (command) {
    case 'edit': openEditDrawer(db); break
      ElMessage.info('编辑功能开发中')
      break
    case 'password':
      openChangePassword(db)
      break
    case 'backup':
      openBackupDrawer(db)
      break
    case 'phpmyadmin':
      ElMessage.info('phpMyAdmin 功能开发中')
      break
    case 'delete':
      ElMessageBox.confirm(`确定删除数据库 ${db.name} 吗？数据将被删除无法恢复！`, '警告', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(async () => {
        const index = databases.value.findIndex(d => d.id === db.id)
        if (index !== -1) {
          databases.value.splice(index, 1)
          ElMessage.success('数据库已删除')
        }
      }).catch(() => {})
      break
  }
}
</script>

<style scoped>
.databases-container {
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
  flex-wrap: wrap;
}

.top-right {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

.db-tabs {
  margin-right: 16px;
}

.databases-panel {
  background: #ffffff;
  border-radius: 4px;
  padding: 0;
  flex: 1;
  overflow-y: auto;
}

.status-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
}

.status-badge.running {
  background-color: #e8f4e8;
  color: #2e7d32;
}

.status-badge.stopped {
  background-color: #f4e8e8;
  color: #c62828;
}

.remark-cell {
  cursor: pointer;
}

.remark-cell:hover {
  color: #477779;
}

.remark-placeholder {
  color: #9ca3af;
  font-style: italic;
}

.drawer-content {
  padding: 0 4px;
}

.form-tip {
  font-size: 11px;
  color: #9ca3af;
  margin-top: 4px;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 24px;
  padding-top: 16px;
  border-top: 1px solid #e5e7eb;
}

.connection-info {
  padding: 8px 0;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 0;
  border-bottom: 1px solid #e5e7eb;
}

.info-label {
  width: 80px;
  color: #9ca3af;
  font-size: 13px;
}

.info-value {
  flex: 1;
  color: #1f2937;
  font-size: 13px;
  word-break: break-all;
}

.connection-string {
  font-family: monospace;
  font-size: 12px;
}

.password-value {
  padding: 8px 12px;
  background-color: #f5f5f5;
  border-radius: 4px;
  margin-top: -8px;
  margin-bottom: 12px;
  font-family: monospace;
  font-size: 12px;
}

.backup-info {
  background-color: #f5f5f5;
  border-radius: 4px;
  padding: 12px;
  margin-bottom: 20px;
}

.backup-site {
  font-size: 14px;
  font-weight: 600;
  color: #1f2937;
}

.backup-path {
  font-size: 12px;
  color: #9ca3af;
  margin-top: 4px;
}

.backup-list {
  margin-top: 20px;
}

.backup-list-title {
  font-size: 13px;
  font-weight: 500;
  color: #1f2937;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid #e5e7eb;
}

.backup-items {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.backup-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px;
  background-color: #f9fafb;
  border-radius: 4px;
}

.backup-item-info {
  flex: 1;
}

.backup-item-name {
  font-size: 13px;
  color: #1f2937;
}

.backup-item-time {
  font-size: 11px;
  color: #9ca3af;
  margin-top: 2px;
}

.backup-item-size {
  font-size: 11px;
  color: #9ca3af;
}

.backup-item-actions {
  display: flex;
  gap: 8px;
}
</style>
<style>
.top-panel .el-button {
  height: 28px !important;
  line-height: 28px !important;
  padding: 0 12px !important;
  font-size: 12px !important;
}
.top-panel .el-input__wrapper {
  height: 28px !important;
  padding: 0 8px !important;
}
.top-panel .el-tabs__item {
  height: 32px !important;
  line-height: 32px !important;
  font-size: 13px !important;
}
</style>
