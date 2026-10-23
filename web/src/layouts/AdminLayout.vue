<template>
  <div class="app-shell">
    <a class="skip-link" href="#main-content">跳到主要内容</a>

    <header class="app-header">
      <UButton
        class="app-header__toggle"
        variant="ghost"
        icon-only
        icon="menu"
        aria-label="打开导航抽屉"
        :aria-expanded="drawerOpen ? 'true' : 'false'"
        aria-controls="app-drawer"
        @click="toggleDrawer"
      />

      <router-link to="/dashboard" class="app-header__brand" aria-label="UPanel 管理后台，返回总览">
        <span class="app-header__mark" aria-hidden="true"><UIcon name="zap" :size="16" /></span>
        <span class="app-header__wordmark">
          UPanel <span class="app-header__wordmark-muted">Admin</span>
        </span>
      </router-link>

      <span class="app-header__divider" aria-hidden="true" />

      <div class="app-header__nav">
        <AdminNavDropdown ref="navRef" :current-label="currentTitle" />
      </div>

      <div class="app-header__spacer" />

      <div class="app-header__actions">
        <span class="app-header__status" :aria-label="`网关状态：${gatewayLabel}`">
          <UStatusDot :status="gatewayStatus" :aria-label="`网关状态：${gatewayLabel}`" />
          <span>{{ gatewayShortLabel }}</span>
        </span>

        <UButton
          variant="ghost"
          icon-only
          :icon="isDark ? 'sun' : 'moon'"
          :aria-label="isDark ? '切换到浅色主题' : '切换到深色主题'"
          :title="isDark ? '切换到浅色主题' : '切换到深色主题'"
          @click="toggleTheme"
        />

        <button
          ref="userButtonRef"
          type="button"
          class="app-header__user"
          :aria-expanded="userMenuOpen ? 'true' : 'false'"
          aria-haspopup="menu"
          aria-controls="user-menu"
          @click="toggleUserMenu"
          @keydown.down.prevent="openUserMenu"
        >
          <span class="app-header__avatar" aria-hidden="true">{{ userInitial }}</span>
          <span class="app-header__user-name">{{ username }}</span>
          <UIcon name="chevron-down" :size="14" class="subtle" />
        </button>
      </div>
    </header>

    <button
      v-if="drawerOpen"
      type="button"
      class="drawer-backdrop"
      aria-label="关闭导航"
      @click="drawerOpen = false"
    />

    <aside
      id="app-drawer"
      class="app-drawer"
      aria-label="侧边导航"
      :inert="isMobile && !drawerOpen"
    >
      <div class="app-drawer__scroll">
        <nav class="drawer-nav" aria-label="主导航">
          <div class="drawer-nav__group">
            <p class="drawer-nav__label">导航</p>
            <router-link
              v-for="item in primaryNav"
              :key="item.id"
              :to="item.to"
              class="drawer-nav__link"
              :class="{ 'is-active': isNavActive(item.to) }"
              :aria-current="isNavActive(item.to) ? 'page' : undefined"
            >
              <UIcon :name="item.icon" :size="17" class="drawer-nav__icon" />
              <span>{{ item.label }}</span>
            </router-link>
          </div>

          <div class="drawer-nav__group">
            <p class="drawer-nav__label">工具</p>
            <router-link
              v-for="item in secondaryNav"
              :key="item.id"
              :to="item.to"
              class="drawer-nav__link"
              :class="{ 'is-active': isNavActive(item.to) }"
              :aria-current="isNavActive(item.to) ? 'page' : undefined"
            >
              <UIcon :name="item.icon" :size="17" class="drawer-nav__icon" />
              <span>{{ item.label }}</span>
            </router-link>
          </div>

          <button type="button" class="drawer-nav__all" @click="openNav">
            <UIcon name="grid" :size="16" />
            <span>全部功能</span>
            <span class="drawer-nav__badge">{{ navCount }}</span>
            <kbd class="drawer-nav__all-key" aria-hidden="true">/</kbd>
          </button>
        </nav>
      </div>

      <div class="app-drawer__footer">
        <div class="app-drawer__footer-row">
          <span class="row" style="gap: 6px">
            <UStatusDot :status="gatewayStatus" :aria-label="`网关状态：${gatewayLabel}`" />
            <span>网关</span>
          </span>
          <span class="app-drawer__version">{{ version }}</span>
        </div>
        <div v-if="dockerStatus === 'unavailable'" class="text-warning" style="margin-top: 4px">
          Docker 未连接，部分功能不可用
        </div>
      </div>
    </aside>

    <div class="app-main" :inert="isMobile && drawerOpen">
      <main id="main-content" class="app-content" tabindex="-1">
        <router-view />
      </main>
      <footer class="app-footer">
        <span>UPanel 轻量级 Linux 管理面板</span>
        <span>Docker · Nginx · MySQL · Redis</span>
      </footer>
    </div>

    <Teleport to="body">
      <div
        v-if="userMenuOpen"
        id="user-menu"
        ref="userMenuRef"
        class="menu"
        :style="userMenuStyle"
        role="menu"
        aria-label="账号菜单"
        @keydown.esc.prevent="closeUserMenu(true)"
      >
        <div class="menu__item" role="presentation" style="cursor: default">
          <span class="app-header__avatar" aria-hidden="true">{{ userInitial }}</span>
          <span class="stack stack--sm" style="gap: 0">
            <strong>{{ username }}</strong>
            <span class="subtle" style="font-size: 11px">管理员</span>
          </span>
        </div>
        <div class="menu__divider" role="separator" />
        <button type="button" class="menu__item" role="menuitem" @click="go('/settings?tab=account')">
          <UIcon name="user" :size="15" /> 账号与密码
        </button>
        <button type="button" class="menu__item" role="menuitem" @click="go('/settings')">
          <UIcon name="settings" :size="15" /> 面板设置
        </button>
        <button type="button" class="menu__item" role="menuitem" @click="toggleTheme">
          <UIcon :name="isDark ? 'sun' : 'moon'" :size="15" />
          {{ isDark ? '切换到浅色主题' : '切换到深色主题' }}
        </button>
        <div class="menu__divider" role="separator" />
        <button type="button" class="menu__item menu__item--danger" role="menuitem" @click="logout">
          <UIcon name="logout" :size="15" /> 退出登录
        </button>
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AdminNavDropdown from '@/components/AdminNavDropdown.vue'
import UButton from '@/components/ui/UButton.vue'
import UIcon from '@/components/ui/UIcon.vue'
import UStatusDot from '@/components/ui/UStatusDot.vue'
import { gatewayApi, panelApi } from '@/api'
import { authState, useAuth } from '@/composables/useAuth'
import { toggleTheme, useTheme } from '@/composables/useTheme'
import { navCount, navItems, primaryNav, secondaryNav } from '@/nav/catalog'

const route = useRoute()
const router = useRouter()
const { logout: doLogout } = useAuth()
const { themeState } = useTheme()

const drawerOpen = ref(false)
const isMobile = ref(false)
let mobileMedia = null
const navRef = ref(null)
const userMenuOpen = ref(false)
const userButtonRef = ref(null)
const userMenuRef = ref(null)
const userMenuStyle = ref({})
const version = ref('')
const gatewayStatus = ref('unknown')
const gatewayPort = ref('')
const dockerStatus = ref('')

const username = computed(() => (authState.user && authState.user.username) || 'admin')
const userInitial = computed(() => username.value.slice(0, 1).toUpperCase())
const currentNavItem = computed(() => {
  const candidates = navItems.filter((item) => item.to === route.path)
  const matching = candidates.filter((item) =>
    Object.entries(item.query || {}).every(
      ([key, value]) => String(route.query[key] ?? '') === String(value)
    )
  )
  matching.sort((a, b) => Object.keys(b.query || {}).length - Object.keys(a.query || {}).length)
  return matching[0] || null
})
const currentTitle = computed(
  () => (currentNavItem.value && currentNavItem.value.label) || (route.meta && route.meta.title) || '总览'
)
const isDark = computed(
  () =>
    themeState.mode === 'dark' ||
    (themeState.mode === 'system' && themeState.systemDark)
)
const gatewayLabel = computed(() => {
  if (gatewayStatus.value === 'running') return `运行中${gatewayPort.value ? `，端口 ${gatewayPort.value}` : ''}`
  if (gatewayStatus.value === 'docker_unavailable') return 'Docker 未连接'
  if (gatewayStatus.value === 'not_installed') return '尚未创建'
  return gatewayStatus.value === 'stopped' ? '已停止' : '未知'
})
const gatewayShortLabel = computed(() =>
  gatewayStatus.value === 'running' ? '网关在线' : gatewayStatus.value === 'docker_unavailable' ? 'Docker 离线' : '网关离线'
)

function isNavActive(path) {
  return route.path === path || (path !== '/dashboard' && route.path.startsWith(path + '/'))
}

function toggleDrawer() {
  drawerOpen.value = !drawerOpen.value
}

function openNav() {
  drawerOpen.value = false
  navRef.value?.open?.()
}

function go(path) {
  closeUserMenu()
  router.push(path)
}

function openUserMenu() {
  if (userMenuOpen.value) return
  const rect = userButtonRef.value?.getBoundingClientRect()
  if (rect) {
    userMenuStyle.value = {
      top: `${rect.bottom + 6}px`,
      left: `${Math.max(8, Math.min(rect.right - 210, window.innerWidth - 218))}px`,
      width: '210px',
    }
  }
  userMenuOpen.value = true
  document.addEventListener('mousedown', onDocumentMouseDown, true)
  window.addEventListener('resize', closeUserMenu)
  nextTick(() => {
    userMenuRef.value?.querySelector('button')?.focus()
  })
}

function closeUserMenu(restoreFocus = false) {
  if (!userMenuOpen.value) return
  userMenuOpen.value = false
  document.removeEventListener('mousedown', onDocumentMouseDown, true)
  window.removeEventListener('resize', closeUserMenu)
  if (restoreFocus === true) userButtonRef.value?.focus()
}

function toggleUserMenu() {
  if (userMenuOpen.value) closeUserMenu()
  else openUserMenu()
}

function onDocumentMouseDown(event) {
  const inButton = userButtonRef.value && userButtonRef.value.contains(event.target)
  const inMenu = userMenuRef.value && userMenuRef.value.contains(event.target)
  if (!inButton && !inMenu) closeUserMenu()
}

async function logout() {
  closeUserMenu()
  await doLogout()
  router.replace('/login')
}

async function loadStatus() {
  try {
    const info = await panelApi.info()
    version.value = info.version || ''
  } catch (error) {
    /* 忽略 */
  }
  try {
    const res = await gatewayApi.status()
    const data = res.data || {}
    gatewayStatus.value = data.state || 'unknown'
    gatewayPort.value = data.httpPort || ''
  } catch (error) {
    gatewayStatus.value = 'docker_unavailable'
  }
  dockerStatus.value = gatewayStatus.value === 'docker_unavailable' ? 'unavailable' : ''
}

let statusTimer = null

function onKeydown(event) {
  const target = event.target
  const tag = target && target.tagName ? target.tagName.toLowerCase() : ''
  const typing =
    tag === 'input' ||
    tag === 'textarea' ||
    tag === 'select' ||
    (target && target.isContentEditable) ||
    (target && target.closest && target.closest('[role="dialog"]'))
  if (typing) return
  if (event.key === '/' || ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k')) {
    event.preventDefault()
    openNav()
  }
}

watch(
  () => route.fullPath,
  () => {
    drawerOpen.value = false
    closeUserMenu()
  }
)

watch(drawerOpen, (value) => {
  document.body.classList.toggle('drawer-open', value)
  if (!isMobile.value) return
  if (value) {
    nextTick(() => {
      const firstLink = document.querySelector('#app-drawer .drawer-nav__link')
      firstLink?.focus()
    })
  } else {
    document.querySelector('.app-header__toggle')?.focus()
  }
})

onMounted(() => {
  mobileMedia = window.matchMedia('(max-width: 1024px)')
  isMobile.value = mobileMedia.matches
  mobileMedia.addEventListener('change', onMobileChange)
  loadStatus()
  statusTimer = setInterval(loadStatus, 30000)
  window.addEventListener('keydown', onKeydown)
})

function onMobileChange(event) {
  isMobile.value = event.matches
  if (!event.matches) drawerOpen.value = false
}

onBeforeUnmount(() => {
  if (statusTimer) clearInterval(statusTimer)
  if (mobileMedia) mobileMedia.removeEventListener('change', onMobileChange)
  window.removeEventListener('keydown', onKeydown)
  document.body.classList.remove('drawer-open')
  closeUserMenu()
})
</script>
