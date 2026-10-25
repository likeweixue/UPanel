<template>
  <div class="app-shell" :class="{ 'is-sidebar-collapsed': sidebarCollapsed && !isMobile, 'has-fixed-footer': footer.visible }">
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
        <ULogo size="sm" />
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
          <div v-if="visiblePrimary.length" class="drawer-nav__group">
            <p class="drawer-nav__label">导航</p>
            <router-link
              v-for="item in visiblePrimary"
              :key="item.id"
              :to="item.to"
              class="drawer-nav__link"
              :class="{ 'is-active': isNavActive(item.to) }"
              :aria-current="isNavActive(item.to) ? 'page' : undefined"
              :title="item.label"
              :aria-label="item.label"
            >
              <UIcon :name="item.icon" :size="17" class="drawer-nav__icon" />
              <span>{{ item.label }}</span>
            </router-link>
          </div>

          <div v-if="visibleSecondary.length" class="drawer-nav__group">
            <p class="drawer-nav__label">工具</p>
            <router-link
              v-for="item in visibleSecondary"
              :key="item.id"
              :to="item.to"
              class="drawer-nav__link"
              :class="{ 'is-active': isNavActive(item.to) }"
              :aria-current="isNavActive(item.to) ? 'page' : undefined"
              :title="item.label"
              :aria-label="item.label"
            >
              <UIcon :name="item.icon" :size="17" class="drawer-nav__icon" />
              <span>{{ item.label }}</span>
            </router-link>
          </div>

          <button type="button" class="drawer-nav__all" title="全部功能" @click="openNav">
            <UIcon name="grid" :size="16" />
            <span>全部功能</span>
            <span class="drawer-nav__badge">{{ navCount }}</span>
            <kbd class="drawer-nav__all-key" aria-hidden="true">/</kbd>
          </button>
        </nav>
      </div>

      <div class="app-drawer__actions">
        <button
          type="button"
          class="drawer-action"
          :title="sidebarToggleLabel"
          :aria-label="sidebarToggleLabel"
          @click="toggleSidebar"
        >
          <UIcon :name="sidebarToggleIcon" :size="16" />
          <span v-if="!sidebarCollapsed || isMobile">{{ sidebarToggleLabel }}</span>
        </button>
        <button
          type="button"
          class="drawer-action"
          title="菜单设置"
          aria-label="菜单设置"
          @click="menuConfigOpen = true"
        >
          <UIcon name="settings" :size="16" />
          <span v-if="!sidebarCollapsed || isMobile">菜单设置</span>
        </button>
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
      <footer v-if="footer.visible" class="app-footer">
        <div class="app-footer__inner">
          <div class="app-footer__meta">
            <span class="app-footer__brand">{{ footer.text || 'UPanel' }}</span>
            <span v-if="footer.showVersion && version" class="app-footer__version">{{ version }}</span>
            <span class="app-footer__license">MIT License</span>
          </div>
          <nav class="app-footer__links" aria-label="项目链接">
            <a
              v-if="footer.showGithub"
              href="https://github.com/likeweixue/UPanel"
              target="_blank"
              rel="noopener noreferrer"
            >
              <UIcon name="github" :size="14" />
              GitHub
            </a>
          </nav>
        </div>
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

    <UModal
      v-model="menuConfigOpen"
      title="侧边栏菜单"
      description="显示或隐藏菜单，并调整它们的上下顺序"
      size="lg"
    >
      <div class="menu-config">
        <section v-for="group in menuConfigGroups" :key="group.key" class="menu-config__group">
          <h3 class="menu-config__title">{{ group.title }}</h3>
          <div class="menu-config__list">
            <div
              v-for="(item, index) in group.items"
              :key="item.id"
              class="menu-config__item"
              :class="{ 'is-hidden': !isMenuItemVisible(item.id) }"
            >
              <USwitch
                :model-value="isMenuItemVisible(item.id)"
                :aria-label="`显示 ${item.label}`"
                @update:model-value="toggleMenuItem(item.id)"
              />
              <UIcon :name="item.icon" :size="16" class="menu-config__icon" />
              <span class="menu-config__label">{{ item.label }}</span>
              <div class="menu-config__move">
                <UButton
                  variant="ghost"
                  icon-only
                  size="sm"
                  icon="arrow-up"
                  :disabled="index === 0"
                  :aria-label="`上移 ${item.label}`"
                  @click="moveMenuItem(group.key, index, -1)"
                />
                <UButton
                  variant="ghost"
                  icon-only
                  size="sm"
                  icon="arrow-down"
                  :disabled="index === group.items.length - 1"
                  :aria-label="`下移 ${item.label}`"
                  @click="moveMenuItem(group.key, index, 1)"
                />
              </div>
            </div>
          </div>
        </section>
        <p class="field__hint">
          隐藏的菜单仍可通过顶部「全部功能」搜索进入。此配置保存在当前浏览器。
        </p>
      </div>
      <template #footer>
        <div class="row" style="margin-left: auto; gap: 8px">
          <UButton variant="secondary" icon="refresh" label="恢复默认" @click="resetMenuConfig" />
          <UButton variant="primary" label="完成" @click="menuConfigOpen = false" />
        </div>
      </template>
    </UModal>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AdminNavDropdown from '@/components/AdminNavDropdown.vue'
import UButton from '@/components/ui/UButton.vue'
import UIcon from '@/components/ui/UIcon.vue'
import ULogo from '@/components/ui/ULogo.vue'
import UModal from '@/components/ui/UModal.vue'
import UStatusDot from '@/components/ui/UStatusDot.vue'
import USwitch from '@/components/ui/USwitch.vue'
import { gatewayApi, panelApi, settingsApi } from '@/api'
import { authState, useAuth } from '@/composables/useAuth'
import { isDarkTheme, setCornerRadius, toggleTheme, useTheme } from '@/composables/useTheme'
import { navCount, navItems, primaryNav, secondaryNav } from '@/nav/catalog'

const route = useRoute()
const router = useRouter()
const { logout: doLogout } = useAuth()
const { themeState } = useTheme()

const drawerOpen = ref(false)
const isMobile = ref(false)
const MENU_KEY = 'upanel_sidebar_menu'
const COLLAPSE_KEY = 'upanel_sidebar_collapsed'
const drawerItemMap = Object.fromEntries(
  [...primaryNav, ...secondaryNav].map((item) => [item.id, item])
)
const defaultMenuConfig = {
  primary: primaryNav.map((item) => item.id),
  secondary: secondaryNav.map((item) => item.id),
  hidden: [],
}
const menuConfig = reactive(loadMenuConfig())
const menuConfigOpen = ref(false)
const sidebarCollapsed = ref(localStorage.getItem(COLLAPSE_KEY) === 'true')
const footer = reactive({
  visible: true,
  text: 'UPanel',
  showVersion: true,
  showGithub: true,
})

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
const isDark = computed(() => isDarkTheme(themeState.effective))
const gatewayLabel = computed(() => {
  if (gatewayStatus.value === 'running') return `运行中${gatewayPort.value ? `，端口 ${gatewayPort.value}` : ''}`
  if (gatewayStatus.value === 'docker_unavailable') return 'Docker 未连接'
  if (gatewayStatus.value === 'not_installed') return '尚未创建'
  return gatewayStatus.value === 'stopped' ? '已停止' : '未知'
})
const gatewayShortLabel = computed(() =>
  gatewayStatus.value === 'running' ? '网关在线' : gatewayStatus.value === 'docker_unavailable' ? 'Docker 离线' : '网关离线'
)

function normalizeMenuConfig(raw) {
  const valid = new Set(Object.keys(drawerItemMap))
  const order = (value, fallback) => {
    const seen = new Set()
    const out = []
    if (Array.isArray(value)) {
      for (const id of value) {
        if (valid.has(id) && !seen.has(id)) {
          seen.add(id)
          out.push(id)
        }
      }
    }
    for (const id of fallback) {
      if (!seen.has(id)) out.push(id)
    }
    return out
  }
  return {
    primary: order(raw && raw.primary, defaultMenuConfig.primary),
    secondary: order(raw && raw.secondary, defaultMenuConfig.secondary),
    hidden: Array.isArray(raw && raw.hidden)
      ? [...new Set(raw.hidden.filter((id) => valid.has(id)))]
      : [],
  }
}

function loadMenuConfig() {
  try {
    return normalizeMenuConfig(JSON.parse(localStorage.getItem(MENU_KEY) || 'null'))
  } catch (error) {
    return normalizeMenuConfig(null)
  }
}

function persistMenuConfig() {
  localStorage.setItem(
    MENU_KEY,
    JSON.stringify({
      primary: menuConfig.primary,
      secondary: menuConfig.secondary,
      hidden: menuConfig.hidden,
    })
  )
}

const visiblePrimary = computed(() =>
  menuConfig.primary
    .map((id) => drawerItemMap[id])
    .filter((item) => item && !menuConfig.hidden.includes(item.id))
)
const visibleSecondary = computed(() =>
  menuConfig.secondary
    .map((id) => drawerItemMap[id])
    .filter((item) => item && !menuConfig.hidden.includes(item.id))
)
const menuConfigGroups = computed(() => [
  {
    key: 'primary',
    title: '导航',
    items: menuConfig.primary.map((id) => drawerItemMap[id]).filter(Boolean),
  },
  {
    key: 'secondary',
    title: '工具',
    items: menuConfig.secondary.map((id) => drawerItemMap[id]).filter(Boolean),
  },
])
const sidebarToggleLabel = computed(() =>
  isMobile.value ? '关闭侧边栏' : sidebarCollapsed.value ? '展开侧边栏' : '收起侧边栏'
)
const sidebarToggleIcon = computed(() =>
  isMobile.value ? 'x' : sidebarCollapsed.value ? 'chevron-right' : 'chevron-left'
)

function isMenuItemVisible(id) {
  return !menuConfig.hidden.includes(id)
}

function toggleMenuItem(id) {
  const index = menuConfig.hidden.indexOf(id)
  if (index >= 0) menuConfig.hidden.splice(index, 1)
  else menuConfig.hidden.push(id)
  persistMenuConfig()
}

function moveMenuItem(groupKey, index, delta) {
  const list = menuConfig[groupKey]
  const next = index + delta
  if (!Array.isArray(list) || next < 0 || next >= list.length) return
  ;[list[index], list[next]] = [list[next], list[index]]
  persistMenuConfig()
}

function resetMenuConfig() {
  const defaults = normalizeMenuConfig(null)
  menuConfig.primary = defaults.primary
  menuConfig.secondary = defaults.secondary
  menuConfig.hidden = defaults.hidden
  persistMenuConfig()
}

function toggleSidebar() {
  if (isMobile.value) {
    drawerOpen.value = false
    return
  }
  sidebarCollapsed.value = !sidebarCollapsed.value
  localStorage.setItem(COLLAPSE_KEY, String(sidebarCollapsed.value))
}

function applyUiSettings(data) {
  if (!data) return
  if (data.cornerRadius !== undefined && data.cornerRadius !== null) {
    setCornerRadius(data.cornerRadius)
  }
  footer.visible = data.footerVisible !== false
  footer.text = data.footerText || 'UPanel'
  footer.showVersion = data.footerShowVersion !== false
  footer.showGithub = data.footerShowGithub !== false
}

async function loadUiSettings() {
  try {
    const res = await settingsApi.get()
    applyUiSettings(res.data)
  } catch (error) {
    /* 设置加载失败时保持默认外观 */
  }
}

function onSettingsUpdated(event) {
  applyUiSettings(event.detail)
}

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
  loadUiSettings()
  statusTimer = setInterval(loadStatus, 30000)
  window.addEventListener('keydown', onKeydown)
  window.addEventListener('upanel:settings-updated', onSettingsUpdated)
})

function onMobileChange(event) {
  isMobile.value = event.matches
  if (!event.matches) drawerOpen.value = false
}

onBeforeUnmount(() => {
  if (statusTimer) clearInterval(statusTimer)
  if (mobileMedia) mobileMedia.removeEventListener('change', onMobileChange)
  window.removeEventListener('keydown', onKeydown)
  window.removeEventListener('upanel:settings-updated', onSettingsUpdated)
  document.body.classList.remove('drawer-open')
  closeUserMenu()
})
</script>
