import { createRouter, createWebHistory } from 'vue-router'
import { baseHref } from '@/api'
import { authState, useAuth } from '@/composables/useAuth'

const layout = () => import('@/layouts/AdminLayout.vue')

const routes = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/Login.vue'),
    meta: { public: true, title: '登录' },
  },
  {
    path: '/',
    component: layout,
    meta: { requiresAuth: true },
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'dashboard',
        component: () => import('@/views/Dashboard.vue'),
        meta: { title: '总览' },
      },
      {
        path: 'websites',
        name: 'websites',
        component: () => import('@/views/Websites.vue'),
        meta: { title: '网站' },
      },
      {
        path: 'databases',
        name: 'databases',
        component: () => import('@/views/Databases.vue'),
        meta: { title: '数据库' },
      },
      {
        path: 'apps',
        name: 'apps',
        component: () => import('@/views/Apps.vue'),
        meta: { title: '应用商店' },
      },
      {
        path: 'containers',
        name: 'containers',
        component: () => import('@/views/Containers.vue'),
        meta: { title: '容器' },
      },
      {
        path: 'files',
        name: 'files',
        component: () => import('@/views/Files.vue'),
        meta: { title: '文件管理' },
      },
      {
        path: 'monitor',
        name: 'monitor',
        component: () => import('@/views/Monitor.vue'),
        meta: { title: '资源监控' },
      },
      {
        path: 'security',
        name: 'security',
        component: () => import('@/views/Security.vue'),
        meta: { title: '安全中心' },
      },
      {
        path: 'cron',
        name: 'cron',
        component: () => import('@/views/Cron.vue'),
        meta: { title: '计划任务' },
      },
      {
        path: 'ssh',
        name: 'ssh',
        component: () => import('@/views/Ssh.vue'),
        meta: { title: 'SSH 服务' },
      },
      {
        path: 'email-notify',
        name: 'email',
        component: () => import('@/views/EmailNotify.vue'),
        meta: { title: '邮件通知' },
      },
      {
        path: 'settings',
        name: 'settings',
        component: () => import('@/views/Settings.vue'),
        meta: { title: '面板设置' },
      },
      {
        path: 'logout',
        name: 'logout',
        component: () => import('@/views/Logout.vue'),
        meta: { title: '退出登录' },
      },
    ],
  },
  { path: '/:pathMatch(.*)*', redirect: '/dashboard' },
]

const router = createRouter({
  history: createWebHistory(baseHref),
  routes,
  scrollBehavior(to, from, savedPosition) {
    if (savedPosition) return savedPosition
    if (to.hash) return { el: to.hash, top: 80, behavior: 'smooth' }
    return { top: 0 }
  },
})

router.beforeEach(async (to) => {
  const authed = !!authState.token
  if (to.meta.public) {
    if (to.path === '/login' && authed) {
      const { fetchMe, clear } = useAuth()
      if (!authState.user) {
        try {
          await fetchMe()
        } catch (error) {
          clear()
          return true
        }
      }
      return { path: '/dashboard' }
    }
    return true
  }
  if (!authed) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  if (!authState.user) {
    const { fetchMe, clear } = useAuth()
    try {
      await fetchMe()
    } catch (error) {
      clear()
      return { path: '/login', query: { redirect: to.fullPath } }
    }
  }
  return true
})

router.afterEach((to) => {
  const title = (to.meta && to.meta.title) || 'UPanel'
  document.title = `${title} · UPanel`
})

export default router
