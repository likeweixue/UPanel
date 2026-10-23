import axios from 'axios'

const baseEl = typeof document !== 'undefined' ? document.querySelector('base') : null
export const baseHref = (() => {
  const href = (baseEl && baseEl.getAttribute('href')) || '/'
  return href.endsWith('/') ? href : href + '/'
})()
// '' (根路径) 或 '/entry'
export const basePath = baseHref.replace(/\/$/, '')

const TOKEN_KEY = 'upanel_token'

export function getToken() {
  return localStorage.getItem(TOKEN_KEY) || localStorage.getItem('token') || ''
}

export function setToken(token) {
  if (token) {
    localStorage.setItem(TOKEN_KEY, token)
    localStorage.removeItem('token')
  } else {
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem('token')
  }
}

const http = axios.create({
  baseURL: `${basePath}/api`,
  timeout: 10 * 60 * 1000,
  headers: { 'Content-Type': 'application/json' },
})

http.interceptors.request.use((config) => {
  const token = getToken()
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

http.interceptors.response.use(
  (response) => response,
  (error) => {
    const response = error.response
    const url = (error.config && error.config.url) || ''
    if (response && response.status === 401 && !url.includes('/auth/login')) {
      setToken('')
      window.dispatchEvent(new CustomEvent('upanel:unauthorized'))
    }
    const message =
      (response && response.data && (response.data.error || response.data.message)) ||
      (error.code === 'ECONNABORTED' ? '请求超时，请稍后重试' : '') ||
      error.message ||
      '请求失败'
    error.userMessage = message
    return Promise.reject(error)
  }
)

export function errText(error, fallback = '操作失败') {
  return (error && error.userMessage) || fallback
}

const get = (url, config) => http.get(url, config).then((r) => r.data)
const post = (url, body, config) => http.post(url, body, config).then((r) => r.data)
const put = (url, body, config) => http.put(url, body, config).then((r) => r.data)
const del = (url, config) => http.delete(url, config).then((r) => r.data)

export const panelApi = {
  info: () => get('/panel/info'),
  health: () => get('/health'),
}

export const authApi = {
  login: (username, password) => post('/auth/login', { username, password }),
  logout: () => post('/auth/logout', {}),
  me: () => get('/auth/userinfo'),
  changePassword: (oldPassword, newPassword) =>
    post('/auth/change-password', { old_password: oldPassword, new_password: newPassword }),
}

export const systemApi = {
  info: () => get('/system/info'),
  ssh: () => get('/system/ssh'),
  firewall: () => get('/system/firewall'),
}

export const siteApi = {
  list: () => get('/websites'),
  get: (domain) => get(`/websites/${encodeURIComponent(domain)}`),
  create: (payload) => post('/websites', payload, { timeout: 30 * 60 * 1000 }),
  update: (domain, payload) => put(`/websites/${encodeURIComponent(domain)}`, payload),
  remove: (domain, keepFiles) =>
    del(`/websites/${encodeURIComponent(domain)}?keep_files=${keepFiles ? 1 : 0}`),
  start: (domain) => post(`/websites/${encodeURIComponent(domain)}/start`, {}),
  stop: (domain) => post(`/websites/${encodeURIComponent(domain)}/stop`, {}),
  restart: (domain) => post(`/websites/${encodeURIComponent(domain)}/restart`, {}),
  logs: (domain, kind = 'access', lines = 200) =>
    get(`/websites/${encodeURIComponent(domain)}/logs?kind=${kind}&lines=${lines}`),
  applySSL: (domain, payload) => post(`/websites/${encodeURIComponent(domain)}/ssl`, payload),
  disableSSL: (domain) => del(`/websites/${encodeURIComponent(domain)}/ssl`),
}

export const gatewayApi = {
  status: () => get('/gateway/status'),
  ensure: () => post('/gateway/ensure', {}, { timeout: 20 * 60 * 1000 }),
}

export const dbApi = {
  list: () => get('/databases'),
  create: (payload) => post('/databases', payload, { timeout: 30 * 60 * 1000 }),
  connection: (name) => get(`/databases/${encodeURIComponent(name)}/connection`),
  start: (name) => post(`/databases/${encodeURIComponent(name)}/start`, {}),
  stop: (name) => post(`/databases/${encodeURIComponent(name)}/stop`, {}),
  restart: (name) => post(`/databases/${encodeURIComponent(name)}/restart`, {}),
  remove: (name, keepData) =>
    del(`/databases/${encodeURIComponent(name)}?keep_data=${keepData ? 1 : 0}`),
  logs: (name, tail = 200) => get(`/databases/${encodeURIComponent(name)}/logs?tail=${tail}`),
}

export const appApi = {
  list: () => get('/apps'),
  installed: () => get('/apps/installed'),
  install: (payload) => post('/apps/install', payload, { timeout: 30 * 60 * 1000 }),
  credentials: (name) => get(`/apps/${encodeURIComponent(name)}/credentials`),
  start: (name) => post(`/apps/${encodeURIComponent(name)}/start`, {}),
  stop: (name) => post(`/apps/${encodeURIComponent(name)}/stop`, {}),
  restart: (name) => post(`/apps/${encodeURIComponent(name)}/restart`, {}),
  remove: (name, keepData) =>
    del(`/apps/${encodeURIComponent(name)}?keep_data=${keepData ? 1 : 0}`),
}

export const containerApi = {
  list: () => get('/containers?all=true'),
  logs: (id, tail = 200) => get(`/containers/${encodeURIComponent(id)}/logs?tail=${tail}`),
  start: (id) => post(`/containers/${encodeURIComponent(id)}/start`, {}),
  stop: (id) => post(`/containers/${encodeURIComponent(id)}/stop`, {}),
  restart: (id) => post(`/containers/${encodeURIComponent(id)}/restart`, {}),
  remove: (id) => del(`/containers/${encodeURIComponent(id)}`),
  images: () => get('/docker/images'),
  volumes: () => get('/docker/volumes'),
  networks: () => get('/docker/networks'),
}

export const fileApi = {
  root: () => get('/files/root'),
  list: (path = '') => get(`/files/list?path=${encodeURIComponent(path)}`),
  content: (path) => get(`/files/content?path=${encodeURIComponent(path)}`),
  save: (path, content) => post('/files/content', { path, content }),
  folder: (path, name) => post('/files/folder', { path, name }),
  file: (path, name) => post('/files/file', { path, name }),
  rename: (path, newName) => put('/files/rename', { path, new_name: newName }),
  remove: (path) => del(`/files/delete?path=${encodeURIComponent(path)}`),
  upload: (path, file) => {
    const form = new FormData()
    form.append('path', path)
    form.append('file', file)
    return http
      .post('/files/upload', form, {
        headers: { 'Content-Type': 'multipart/form-data' },
        timeout: 30 * 60 * 1000,
      })
      .then((r) => r.data)
  },
  download: (path) =>
    http.get('/files/download', { params: { path }, responseType: 'blob' }).then((r) => r),
}

export const settingsApi = {
  get: () => get('/settings'),
  update: (payload) => put('/settings', payload, { timeout: 20 * 60 * 1000 }),
}

export const dockerApi = {
  version: () => get('/docker/version'),
  config: () => get('/docker/config'),
  saveConfig: (payload) => put('/docker/config', payload),
  restart: () => post('/docker/restart', {}),
}

export const securityApi = {
  overview: () => get('/security/overview'),
  audit: (limit = 100) => get(`/security/audit?limit=${limit}`),
}

export const cronApi = {
  list: () => get('/cron'),
  create: (payload) => post('/cron', payload),
  update: (id, payload) => put(`/cron/${encodeURIComponent(id)}`, payload),
  remove: (id) => del(`/cron/${encodeURIComponent(id)}`),
  run: (id) => post(`/cron/${encodeURIComponent(id)}/run`, {}),
}

export const emailApi = {
  get: () => get('/email/config'),
  save: (payload) => put('/email/config', payload),
  test: (payload) => post('/email/test', payload),
}

export default http
