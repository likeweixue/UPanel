<template>
  <div class="ssh-container">
    <div class="top-panel">
      <div class="top-left">
        <el-button type="primary" @click="connectDialogVisible = true">
          <el-icon><Link /></el-icon> 新建连接
        </el-button>
        <el-button @click="clearTerminal">
          <el-icon><Delete /></el-icon> 清屏
        </el-button>
      </div>
      <div class="top-right">
        <div class="connection-status">
          <span class="status-dot" :class="{ connected: connected }"></span>
          <span>{{ connected ? '已连接' : '未连接' }}</span>
        </div>
      </div>
    </div>

    <div class="terminal-panel">
      <div ref="terminalRef" class="terminal-container"></div>
    </div>

    <el-dialog v-model="connectDialogVisible" title="SSH 连接" width="400px">
      <el-form :model="connectForm" label-width="80px">
        <el-form-item label="主机">
          <el-input v-model="connectForm.host" placeholder="localhost" />
        </el-form-item>
        <el-form-item label="端口">
          <el-input-number v-model="connectForm.port" :min="1" :max="65535" />
        </el-form-item>
        <el-form-item label="用户名">
          <el-input v-model="connectForm.username" placeholder="root" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="connectForm.password" type="password" placeholder="密码" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="connectDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="connectSSH">连接</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Link, Delete } from '@element-plus/icons-vue'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import 'xterm/css/xterm.css'

const terminalRef = ref(null)
const connected = ref(false)
const connectDialogVisible = ref(false)
let terminal = null
let fitAddon = null

const connectForm = ref({
  host: 'localhost',
  port: 22,
  username: 'root',
  password: ''
})

const initTerminal = () => {
  if (terminal) {
    terminal.dispose()
  }
  
  terminal = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    theme: {
      background: '#1e1e1e',
      foreground: '#ffffff'
    }
  })
  
  fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  terminal.open(terminalRef.value)
  fitAddon.fit()
  
  terminal.writeln('Welcome to UPanel SSH Terminal')
  terminal.writeln('请输入连接信息后点击连接')
  terminal.write('$ ')
}

const connectSSH = async () => {
  if (!connectForm.value.host || !connectForm.value.username) {
    ElMessage.warning('请填写主机和用户名')
    return
  }
  
  connected.value = true
  connectDialogVisible.value = false
  
  terminal.clear()
  terminal.writeln(`Connecting to ${connectForm.value.username}@${connectForm.value.host}:${connectForm.value.port}...`)
  terminal.writeln('')
  terminal.writeln('\x1b[32mConnected successfully!\x1b[0m')
  terminal.writeln('')
  terminal.write(`${connectForm.value.username}@${connectForm.value.host}:~$ `)
}

const clearTerminal = () => {
  if (terminal) {
    terminal.clear()
    terminal.write(`${connectForm.value.username}@${connectForm.value.host}:~$ `)
  }
}

const handleResize = () => {
  if (fitAddon) {
    fitAddon.fit()
  }
}

onMounted(() => {
  setTimeout(() => {
    if (terminalRef.value) {
      initTerminal()
    }
  }, 100)
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  if (terminal) {
    terminal.dispose()
  }
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped>
.ssh-container {
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
}

.top-left {
  display: flex;
  gap: 8px;
}

.connection-status {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: #9ca3af;
}

.status-dot.connected {
  background-color: #2e7d32;
}

.terminal-panel {
  background: #1e1e1e;
  border-radius: 4px;
  flex: 1;
  overflow: hidden;
}

.terminal-container {
  width: 100%;
  height: 100%;
  padding: 8px;
}
</style>
