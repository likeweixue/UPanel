export function formatBytes(value) {
  const n = Number(value) || 0
  if (n < 1024) return `${n} B`
  const units = ['KB', 'MB', 'GB', 'TB', 'PB']
  let v = n / 1024
  let i = 0
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v >= 100 ? v.toFixed(0) : v.toFixed(1)} ${units[i]}`
}

export function formatNumber(value) {
  return new Intl.NumberFormat('zh-CN').format(Number(value) || 0)
}

export function formatPercent(value, digits = 0) {
  const n = Number(value)
  if (!Number.isFinite(n)) return '0%'
  return `${n.toFixed(digits)}%`
}

export function clampPercent(value) {
  const n = Number(value)
  if (!Number.isFinite(n)) return 0
  return Math.max(0, Math.min(100, n))
}

export function formatDate(value) {
  if (!value) return '—'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return '—'
  return d.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

export function formatDateShort(value) {
  if (!value) return '—'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return '—'
  return d.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export function relativeTime(value) {
  if (!value) return '—'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return '—'
  const diff = (Date.now() - d.getTime()) / 1000
  if (diff < 60) return '刚刚'
  if (diff < 3600) return `${Math.floor(diff / 60)} 分钟前`
  if (diff < 86400) return `${Math.floor(diff / 3600)} 小时前`
  if (diff < 86400 * 30) return `${Math.floor(diff / 86400)} 天前`
  return formatDateShort(value)
}

export function shortId(id, length = 12) {
  const s = String(id || '')
  return s.length > length ? s.slice(0, length) : s
}

const STATUS_LABELS = {
  running: '运行中',
  stopped: '已停止',
  exited: '已停止',
  created: '已创建',
  paused: '已暂停',
  restarting: '重启中',
  missing: '容器缺失',
  not_installed: '未安装',
  not_installed_docker: 'Docker 未连接',
  docker_unavailable: 'Docker 未连接',
  error: '异常',
  unknown: '未知',
}

export function statusLabel(status) {
  return STATUS_LABELS[status] || status || '未知'
}

export function statusTone(status) {
  if (status === 'running' || status === 'active' || status === 'success') return 'running'
  if (status === 'error' || status === 'missing' || status === 'failed' || status === 'timeout') return 'error'
  if (status === 'docker_unavailable') return 'warning'
  if (status === 'warning' || status === 'restarting' || status === 'paused') return 'warning'
  return 'stopped'
}

export function formatMeterTone(percent) {
  const n = Number(percent) || 0
  if (n >= 90) return 'danger'
  if (n >= 70) return 'warning'
  return 'success'
}

export function copyText(text) {
  if (navigator.clipboard && window.isSecureContext) {
    return navigator.clipboard.writeText(String(text))
  }
  const textarea = document.createElement('textarea')
  textarea.value = String(text)
  textarea.style.position = 'fixed'
  textarea.style.opacity = '0'
  document.body.appendChild(textarea)
  textarea.select()
  const ok = document.execCommand('copy')
  document.body.removeChild(textarea)
  return ok ? Promise.resolve() : Promise.reject(new Error('复制失败'))
}
