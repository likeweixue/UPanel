<template>
  <div class="cron-container">
    <!-- 顶部分类面板 -->
    <div class="top-panel">
      <div class="top-left">
        <el-button type="primary" @click="openCreateDrawer">
          <el-icon><Plus /></el-icon> 添加任务
        </el-button>
        <el-button @click="refreshTasks">
          <el-icon><Refresh /></el-icon> 刷新
        </el-button>
        <el-button @click="openLogDrawer">
          <el-icon><Document /></el-icon> 执行日志
        </el-button>
      </div>
      <div class="top-right">
        <el-input 
          v-model="searchKeyword" 
          placeholder="搜索任务" 
          prefix-icon="Search"
          clearable
          style="width: 200px"
        />
      </div>
    </div>

    <!-- 任务表格 -->
    <div class="content-panel">
      <el-table :data="filteredTasks" stripe style="width: 100%" @selection-change="handleSelectionChange">
        <el-table-column type="selection" width="55" />
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <span class="status-badge" :class="row.enabled ? 'enabled' : 'disabled'">
              {{ row.enabled ? '启用' : '禁用' }}
            </span>
          </template>
        </el-table-column>
        <el-table-column prop="name" label="任务名称" min-width="150" />
        <el-table-column prop="expression" label="执行规则" min-width="150">
          <template #default="{ row }">
            <span class="cron-expression">{{ row.expression }}</span>
            <el-tooltip :content="getCronDescription(row.expression)" placement="top">
              <el-icon class="help-icon"><QuestionFilled /></el-icon>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column prop="command" label="执行命令" min-width="250">
          <template #default="{ row }">
            <span class="command-text">{{ row.command }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="lastRun" label="上次执行" width="160" />
        <el-table-column prop="nextRun" label="下次执行" width="160" />
        <el-table-column label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <el-button size="small" :type="row.enabled ? 'warning' : 'success'" @click="toggleTask(row)">
              {{ row.enabled ? '禁用' : '启用' }}
            </el-button>
            <el-button size="small" @click="runTaskNow(row)">
              <el-icon><VideoPlay /></el-icon> 执行
            </el-button>
            <el-dropdown @command="(cmd) => handleTaskCommand(cmd, row)">
              <el-button size="small">
                更多 <el-icon class="el-icon--right"><ArrowDown /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="edit">编辑</el-dropdown-item>
                  <el-dropdown-item command="logs">执行日志</el-dropdown-item>
                  <el-dropdown-item command="delete" divided>删除</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 创建/编辑任务抽屉 -->
    <el-drawer
      v-model="taskDrawerVisible"
      :title="editMode ? '编辑任务' : '添加任务'"
      direction="rtl"
      size="500px"
    >
      <div class="drawer-content">
        <el-form :model="taskForm" :rules="taskRules" ref="taskFormRef" label-width="100px">
          <el-form-item label="任务名称" prop="name">
            <el-input v-model="taskForm.name" placeholder="例如: 备份数据库" />
          </el-form-item>
          
          <el-form-item label="执行命令" prop="command">
            <el-input 
              v-model="taskForm.command" 
              type="textarea" 
              :rows="3"
              placeholder="例如: /usr/bin/php /www/wwwroot/backup.php"
            />
            <div class="form-tip">支持 Shell 命令、PHP 脚本、Python 脚本等</div>
          </el-form-item>
          
          <el-form-item label="执行周期" prop="expression">
            <el-select v-model="taskForm.expression" placeholder="选择预设周期" @change="onPresetChange">
              <el-option label="每分钟" value="* * * * *" />
              <el-option label="每小时" value="0 * * * *" />
              <el-option label="每天" value="0 0 * * *" />
              <el-option label="每周" value="0 0 * * 0" />
              <el-option label="每月" value="0 0 1 * *" />
              <el-option label="自定义" value="custom" />
            </el-select>
            <div class="form-tip" v-if="taskForm.expression === 'custom'">
              <el-input 
                v-model="taskForm.customExpression" 
                placeholder="* * * * * (分 时 日 月 周)"
              />
              <div class="cron-help">
                <a @click="showCronHelp = !showCronHelp">Cron 表达式帮助</a>
                <div v-if="showCronHelp" class="cron-help-content">
                  <p>格式：分 时 日 月 周</p>
                  <p>示例：</p>
                  <p>- <code>0 2 * * *</code> 每天凌晨2点执行</p>
                  <p>- <code>*/5 * * * *</code> 每5分钟执行一次</p>
                  <p>- <code>0 9-17 * * *</code> 每天9-17点整点执行</p>
                </div>
              </div>
            </div>
          </el-form-item>
          
          <el-form-item label="超时时间">
            <el-input-number v-model="taskForm.timeout" :min="0" :max="3600" /> 秒
            <div class="form-tip">0 表示不限制</div>
          </el-form-item>
          
          <el-form-item label="最大输出">
            <el-input-number v-model="taskForm.maxOutput" :min="0" :max="100" /> KB
            <div class="form-tip">日志最大保留大小，0 表示不限制</div>
          </el-form-item>
          
          <el-form-item label="通知邮箱">
            <el-input v-model="taskForm.notifyEmail" placeholder="admin@example.com" />
            <div class="form-tip">任务失败时发送邮件通知</div>
          </el-form-item>
          
          <el-form-item label="备注">
            <el-input 
              v-model="taskForm.remark" 
              type="textarea" 
              :rows="2"
              placeholder="可选，添加任务备注"
            />
          </el-form-item>
        </el-form>
        
        <div class="drawer-footer">
          <el-button @click="taskDrawerVisible = false">取消</el-button>
          <el-button type="primary" @click="saveTask" :loading="saving">
            {{ editMode ? '保存' : '添加' }}
          </el-button>
        </div>
      </div>
    </el-drawer>

    <!-- 执行日志抽屉 -->
    <el-drawer
      v-model="logDrawerVisible"
      title="执行日志"
      direction="rtl"
      size="600px"
    >
      <div class="logs-container">
        <div class="log-filters">
          <el-select v-model="logFilter" placeholder="全部" clearable style="width: 120px">
            <el-option label="全部" value="" />
            <el-option label="成功" value="success" />
            <el-option label="失败" value="failed" />
          </el-select>
          <el-input 
            v-model="logSearch" 
            placeholder="搜索日志" 
            prefix-icon="Search"
            clearable
            style="width: 200px"
          />
          <el-button size="small" @click="clearLogs">清空日志</el-button>
        </div>
        <div class="log-list">
          <div v-for="log in filteredLogs" :key="log.id" class="log-item" :class="log.status">
            <div class="log-header">
              <span class="log-time">{{ log.time }}</span>
              <span class="log-task">{{ log.taskName }}</span>
              <span class="log-status" :class="log.status">
                {{ log.status === 'success' ? '成功' : '失败' }}
              </span>
            </div>
            <div class="log-output" v-if="log.output">{{ log.output }}</div>
            <div class="log-error" v-if="log.error">{{ log.error }}</div>
          </div>
          <div v-if="filteredLogs.length === 0" class="empty">暂无日志</div>
        </div>
      </div>
    </el-drawer>

    <!-- 任务日志抽屉 -->
    <el-drawer
      v-model="taskLogDrawerVisible"
      :title="`${currentTask?.name} - 执行日志`"
      direction="rtl"
      size="500px"
    >
      <div class="task-logs">
        <div v-for="log in taskLogs" :key="log.id" class="log-item" :class="log.status">
          <div class="log-time">{{ log.time }}</div>
          <div class="log-output" v-if="log.output">{{ log.output }}</div>
          <div class="log-error" v-if="log.error">{{ log.error }}</div>
        </div>
        <div v-if="taskLogs.length === 0" class="empty">暂无执行记录</div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh, Document, VideoPlay, ArrowDown, QuestionFilled } from '@element-plus/icons-vue'

const searchKeyword = ref('')
const taskDrawerVisible = ref(false)
const logDrawerVisible = ref(false)
const taskLogDrawerVisible = ref(false)
const editMode = ref(false)
const saving = ref(false)
const showCronHelp = ref(false)
const logFilter = ref('')
const logSearch = ref('')
const selectedRows = ref([])
const currentTask = ref(null)
const taskFormRef = ref(null)

// 任务表单
const taskForm = ref({
  name: '',
  command: '',
  expression: '',
  customExpression: '',
  timeout: 0,
  maxOutput: 10,
  notifyEmail: '',
  remark: ''
})

const taskRules = {
  name: [{ required: true, message: '请输入任务名称', trigger: 'blur' }],
  command: [{ required: true, message: '请输入执行命令', trigger: 'blur' }],
  expression: [{ required: true, message: '请选择执行周期', trigger: 'change' }]
}

// 任务列表
const tasks = ref([
  {
    id: 1,
    name: '数据库备份',
    expression: '0 2 * * *',
    command: '/usr/bin/mysqldump -u root -p123456 --all-databases > /backup/db.sql',
    enabled: true,
    lastRun: '2024-01-15 02:00:01',
    nextRun: '2024-01-16 02:00:00',
    remark: '每天凌晨2点备份数据库'
  },
  {
    id: 2,
    name: '清理日志',
    expression: '0 3 * * 0',
    command: 'find /var/log -name "*.log" -mtime +7 -delete',
    enabled: true,
    lastRun: '2024-01-14 03:00:00',
    nextRun: '2024-01-21 03:00:00',
    remark: '每周日凌晨3点清理7天前的日志'
  },
  {
    id: 3,
    name: '网站健康检查',
    expression: '*/5 * * * *',
    command: 'curl -s https://example.com/health > /dev/null',
    enabled: false,
    lastRun: '2024-01-10 10:35:00',
    nextRun: '2024-01-15 10:40:00',
    remark: '每5分钟检查网站健康状态'
  }
])

// 日志列表
const executionLogs = ref([
  { id: 1, taskName: '数据库备份', time: '2024-01-15 02:00:01', status: 'success', output: '备份完成，大小 156MB' },
  { id: 2, taskName: '清理日志', time: '2024-01-14 03:00:00', status: 'success', output: '已清理 23 个日志文件' },
  { id: 3, taskName: '网站健康检查', time: '2024-01-15 10:35:00', status: 'failed', error: '连接超时' }
])

// 任务日志
const taskLogs = ref([])

// 过滤任务
const filteredTasks = computed(() => {
  if (!searchKeyword.value) return tasks.value
  return tasks.value.filter(t => 
    t.name.includes(searchKeyword.value) || 
    t.command.includes(searchKeyword.value)
  )
})

// 过滤日志
const filteredLogs = computed(() => {
  let result = executionLogs.value
  if (logFilter.value) {
    result = result.filter(l => l.status === logFilter.value)
  }
  if (logSearch.value) {
    result = result.filter(l => 
      l.taskName.includes(logSearch.value) || 
      (l.output && l.output.includes(logSearch.value))
    )
  }
  return result
})

// 获取 Cron 表达式描述
const getCronDescription = (expression) => {
  const descriptions = {
    '* * * * *': '每分钟执行一次',
    '0 * * * *': '每小时整点执行',
    '0 0 * * *': '每天凌晨执行',
    '0 0 * * 0': '每周日凌晨执行',
    '0 0 1 * *': '每月1号凌晨执行'
  }
  return descriptions[expression] || expression
}

// 预设周期变更
const onPresetChange = (val) => {
  if (val !== 'custom') {
    taskForm.value.expression = val
  }
}

// 打开创建抽屉
const openCreateDrawer = () => {
  editMode.value = false
  taskForm.value = {
    name: '',
    command: '',
    expression: '',
    customExpression: '',
    timeout: 0,
    maxOutput: 10,
    notifyEmail: '',
    remark: ''
  }
  showCronHelp.value = false
  taskDrawerVisible.value = true
}

// 打开编辑抽屉
const openEditDrawer = (task) => {
  editMode.value = true
  currentTask.value = task
  taskForm.value = {
    name: task.name,
    command: task.command,
    expression: task.expression,
    customExpression: '',
    timeout: task.timeout || 0,
    maxOutput: task.maxOutput || 10,
    notifyEmail: task.notifyEmail || '',
    remark: task.remark || ''
  }
  taskDrawerVisible.value = true
}

// 保存任务
const saveTask = async () => {
  if (!taskFormRef.value) return
  
  await taskFormRef.value.validate(async (valid) => {
    if (!valid) return
    
    saving.value = true
    try {
      const expression = taskForm.value.expression === 'custom' 
        ? taskForm.value.customExpression 
        : taskForm.value.expression
      
      if (editMode.value && currentTask.value) {
        currentTask.value.name = taskForm.value.name
        currentTask.value.command = taskForm.value.command
        currentTask.value.expression = expression
        currentTask.value.remark = taskForm.value.remark
        ElMessage.success('任务已更新')
      } else {
        const newTask = {
          id: Date.now(),
          name: taskForm.value.name,
          command: taskForm.value.command,
          expression: expression,
          enabled: true,
          lastRun: '-',
          nextRun: '-',
          remark: taskForm.value.remark
        }
        tasks.value.unshift(newTask)
        ElMessage.success('任务已添加')
      }
      taskDrawerVisible.value = false
    } catch (err) {
      ElMessage.error('保存失败')
    } finally {
      saving.value = false
    }
  })
}

// 启用/禁用任务
const toggleTask = async (task) => {
  task.enabled = !task.enabled
  ElMessage.success(`任务已${task.enabled ? '启用' : '禁用'}`)
}

// 立即执行任务
const runTaskNow = async (task) => {
  try {
    await new Promise(resolve => setTimeout(resolve, 1000))
    executionLogs.value.unshift({
      id: Date.now(),
      taskName: task.name,
      time: new Date().toLocaleString(),
      status: 'success',
      output: '手动执行成功'
    })
    ElMessage.success(`任务 ${task.name} 已执行`)
  } catch (err) {
    ElMessage.error('执行失败')
  }
}

// 打开日志抽屉
const openLogDrawer = () => {
  logFilter.value = ''
  logSearch.value = ''
  logDrawerVisible.value = true
}

// 查看任务日志
const viewTaskLogs = (task) => {
  currentTask.value = task
  taskLogs.value = executionLogs.value.filter(l => l.taskName === task.name)
  taskLogDrawerVisible.value = true
}

// 清空日志
const clearLogs = async () => {
  await ElMessageBox.confirm('确定清空所有日志吗？', '警告', { type: 'warning' })
  executionLogs.value = []
  ElMessage.success('日志已清空')
}

// 删除任务
const deleteTask = async (task) => {
  await ElMessageBox.confirm(`确定删除任务 ${task.name} 吗？`, '警告', { type: 'warning' })
  const index = tasks.value.findIndex(t => t.id === task.id)
  if (index !== -1) {
    tasks.value.splice(index, 1)
    ElMessage.success('任务已删除')
  }
}

// 刷新任务
const refreshTasks = () => {
  ElMessage.success('刷新成功')
}

// 选择变化
const handleSelectionChange = (selection) => {
  selectedRows.value = selection
}

// 任务操作命令
const handleTaskCommand = (cmd, task) => {
  switch (cmd) {
    case 'edit':
      openEditDrawer(task)
      break
    case 'logs':
      viewTaskLogs(task)
      break
    case 'delete':
      deleteTask(task)
      break
  }
}
</script>

<style scoped>
.cron-container {
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
  flex-shrink: 0;
}

.content-panel {
  background: #ffffff;
  border-radius: 4px;
  padding: 16px;
  flex: 1;
  overflow-y: auto;
}

.status-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
}

.status-badge.enabled {
  background-color: #e8f4e8;
  color: #2e7d32;
}

.status-badge.disabled {
  background-color: #f4e8e8;
  color: #c62828;
}

.cron-expression {
  font-family: monospace;
  font-size: 12px;
}

.help-icon {
  margin-left: 6px;
  cursor: pointer;
  color: #9ca3af;
}

.help-icon:hover {
  color: #477779;
}

.command-text {
  font-family: monospace;
  font-size: 12px;
  background-color: #f5f5f5;
  padding: 2px 6px;
  border-radius: 4px;
}

.drawer-content {
  padding: 0 4px;
}

.form-tip {
  font-size: 11px;
  color: #9ca3af;
  margin-top: 4px;
}

.cron-help {
  margin-top: 8px;
}

.cron-help a {
  color: #477779;
  cursor: pointer;
  font-size: 12px;
}

.cron-help-content {
  margin-top: 8px;
  padding: 8px 12px;
  background-color: #f5f5f5;
  border-radius: 4px;
  font-size: 12px;
}

.cron-help-content p {
  margin: 4px 0;
}

.cron-help-content code {
  background-color: #e5e7eb;
  padding: 2px 4px;
  border-radius: 4px;
  font-family: monospace;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 24px;
  padding-top: 16px;
  border-top: 1px solid #e5e7eb;
}

/* 日志样式 */
.logs-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.log-filters {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  padding-bottom: 12px;
  border-bottom: 1px solid #e5e7eb;
}

.log-list {
  flex: 1;
  overflow-y: auto;
}

.log-item {
  padding: 12px;
  margin-bottom: 12px;
  border-radius: 4px;
  background-color: #f9fafb;
}

.log-item.success {
  border-left: 3px solid #2e7d32;
}

.log-item.failed {
  border-left: 3px solid #c62828;
}

.log-header {
  display: flex;
  gap: 12px;
  align-items: center;
  margin-bottom: 8px;
  flex-wrap: wrap;
}

.log-time {
  font-size: 12px;
  color: #9ca3af;
  font-family: monospace;
}

.log-task {
  font-size: 13px;
  font-weight: 500;
  color: #1f2937;
}

.log-status {
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 4px;
}

.log-status.success {
  background-color: #e8f4e8;
  color: #2e7d32;
}

.log-status.failed {
  background-color: #f4e8e8;
  color: #c62828;
}

.log-output {
  font-size: 12px;
  color: #4b5563;
  font-family: monospace;
  background-color: #f5f5f5;
  padding: 8px;
  border-radius: 4px;
  margin-top: 8px;
  white-space: pre-wrap;
  word-break: break-all;
}

.log-error {
  font-size: 12px;
  color: #c62828;
  font-family: monospace;
  background-color: #fef3f3;
  padding: 8px;
  border-radius: 4px;
  margin-top: 8px;
}

.empty {
  text-align: center;
  color: #9ca3af;
  padding: 40px;
  font-size: 13px;
}

.task-logs {
  padding: 8px 0;
}
</style>
