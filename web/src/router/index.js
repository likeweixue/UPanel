import { createRouter, createWebHistory } from 'vue-router'
import MainLayout from '@/layouts/MainLayout.vue'

const routes = [
  {
    path: '/',
    component: MainLayout,
    redirect: '/dashboard',
    children: [
      { path: 'dashboard', component: () => import('@/views/Dashboard.vue'), meta: { title: '总览' } },
      { path: 'apps', component: () => import('@/views/Apps.vue'), meta: { title: '应用商店' } },
      { path: 'websites', component: () => import('@/views/Websites.vue'), meta: { title: '网站' } },
      { path: 'databases', component: () => import('@/views/Databases.vue'), meta: { title: '数据库' } },
      { path: 'files', component: () => import('@/views/Files.vue'), meta: { title: '文件管理' } },
      { path: 'settings', component: () => import('@/views/Settings.vue'), meta: { title: '面板设置' } },
      { path: 'logout', component: () => import('@/views/Logout.vue'), meta: { title: '退出面板' } },
      { path: 'containers', component: () => import('@/views/Containers.vue'), meta: { title: '容器管理' } },
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router