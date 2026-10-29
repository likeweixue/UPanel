import { reactive } from 'vue'

const THEME_KEY = 'upanel_theme'
const DENSITY_KEY = 'upanel_density'
const RADIUS_KEY = 'upanel_radius'

export const LIGHT_THEMES = ['light', 'retro-yellow', 'green-paper']
export const DARK_THEMES = ['dark', 'black-gold', 'ember']
export const BUILTIN_THEMES = [...LIGHT_THEMES, ...DARK_THEMES, 'system']

const THEME_COLORS = {
  light: '#ffffff',
  'retro-yellow': '#fbf6ec',
  'green-paper': '#f2f7ef',
  dark: '#171a21',
  'black-gold': '#11100d',
  ember: '#1c1412',
}

const media = typeof window !== 'undefined' && window.matchMedia
  ? window.matchMedia('(prefers-color-scheme: dark)')
  : null

// themes/ 目录里动态发现的主题，由 /api/themes 返回
export const customThemes = reactive([])
let themesLoaded = false
let themesPromise = null

export function isBuiltinTheme(theme) {
  return BUILTIN_THEMES.includes(theme)
}

export function findCustomTheme(theme) {
  return customThemes.find((item) => item.id === theme) || null
}

export function isDarkTheme(theme) {
  if (DARK_THEMES.includes(theme)) return true
  const custom = findCustomTheme(theme)
  return !!custom?.dark
}

function isKnownTheme(theme) {
  return theme === 'system' || isBuiltinTheme(theme) || !!findCustomTheme(theme)
}

export function resolveTheme(mode, systemDark) {
  if (mode === 'system') return systemDark ? 'dark' : 'light'
  return mode || 'light'
}

const savedMode = typeof window !== 'undefined' ? localStorage.getItem(THEME_KEY) || 'light' : 'light'
const systemDark = media ? media.matches : false

export const themeState = reactive({
  mode: savedMode,
  effective: resolveTheme(savedMode, systemDark),
  density: localStorage.getItem(DENSITY_KEY) || 'comfortable',
  systemDark,
  cornerRadius: Math.min(16, Math.max(0, Number(localStorage.getItem(RADIUS_KEY) ?? 4) || 0)),
})

function apply() {
  const effective = resolveTheme(themeState.mode, themeState.systemDark)
  themeState.effective = effective
  const root = document.documentElement
  root.setAttribute('data-theme', effective)
  root.setAttribute('data-density', themeState.density)
  root.style.colorScheme = isDarkTheme(effective) ? 'dark' : 'light'
  root.style.setProperty('--radius-ui', `${themeState.cornerRadius}px`)
  const custom = findCustomTheme(effective)
  const fallback = isDarkTheme(effective) ? THEME_COLORS.dark : THEME_COLORS.light
  const meta = document.querySelector('meta[name="theme-color"]')
  if (meta) {
    meta.setAttribute('content', custom?.themeColor || THEME_COLORS[effective] || fallback)
  }
}

function normalizeCustomTheme(raw) {
  if (!raw || typeof raw.id !== 'string') return null
  const swatch = Array.isArray(raw.swatch) && raw.swatch.length === 2
    ? raw.swatch.map(String)
    : ['#ffffff', '#536f90']
  return {
    id: raw.id,
    name: raw.name || raw.id,
    description: raw.description || '本地主题',
    version: raw.version || '1.0.0',
    author: raw.author || '',
    dark: !!raw.dark,
    themeColor: raw.themeColor || '',
    radius: Number.isFinite(Number(raw.radius)) ? Number(raw.radius) : null,
    swatch,
  }
}

function ensureThemeLink(theme) {
  if (typeof document === 'undefined') return Promise.resolve()
  if (document.head.querySelector(`link[data-upanel-theme="${theme.id}"]`)) {
    return Promise.resolve()
  }
  return new Promise((resolve) => {
    const link = document.createElement('link')
    link.rel = 'stylesheet'
    link.dataset.upanelTheme = theme.id
    link.href = new URL(`api/themes/${theme.id}/theme.css`, document.baseURI).href
    link.onload = resolve
    link.onerror = resolve
    document.head.appendChild(link)
  })
}

export async function loadCustomThemes(force = false) {
  if (themesLoaded && !force) return customThemes
  if (themesPromise) return themesPromise
  themesPromise = (async () => {
    try {
      const response = await fetch(new URL('api/themes', document.baseURI).href, {
        headers: { Accept: 'application/json' },
        cache: 'no-store',
      })
      if (!response.ok) throw new Error(`HTTP ${response.status}`)
      const body = await response.json()
      const list = Array.isArray(body.data) ? body.data : []
      const normalized = list.map(normalizeCustomTheme).filter(Boolean)
      customThemes.splice(0, customThemes.length, ...normalized)
      themesLoaded = true
      await Promise.all(customThemes.map(ensureThemeLink))
      if (themeState.mode !== 'system' && !isKnownTheme(themeState.mode)) {
        themeState.mode = 'light'
        localStorage.setItem(THEME_KEY, themeState.mode)
      }
      apply()
      return customThemes
    } catch (error) {
      themesLoaded = false
      apply()
      return customThemes
    } finally {
      themesPromise = null
    }
  })()
  return themesPromise
}

export function setTheme(mode) {
  if (!isKnownTheme(mode)) return false
  themeState.mode = mode
  localStorage.setItem(THEME_KEY, themeState.mode)
  const custom = findCustomTheme(mode)
  if (custom && custom.radius !== null && custom.radius !== undefined) {
    setCornerRadius(custom.radius)
  } else {
    apply()
  }
  return true
}

export function setDensity(density) {
  themeState.density = density === 'compact' ? 'compact' : 'comfortable'
  localStorage.setItem(DENSITY_KEY, themeState.density)
  apply()
}

export function setCornerRadius(value) {
  const px = Math.min(16, Math.max(0, Math.round(Number(value) || 0)))
  themeState.cornerRadius = px
  localStorage.setItem(RADIUS_KEY, String(px))
  apply()
}

export function toggleTheme() {
  setTheme(isDarkTheme(themeState.effective) ? 'light' : 'dark')
}

export function initTheme() {
  apply()
  if (media) {
    media.addEventListener('change', (event) => {
      themeState.systemDark = event.matches
      if (themeState.mode === 'system') apply()
    })
  }
  loadCustomThemes()
}

export function useTheme() {
  return {
    themeState,
    customThemes,
    setTheme,
    setDensity,
    setCornerRadius,
    loadCustomThemes,
    toggleTheme,
    initTheme,
    isDarkTheme,
    isBuiltinTheme,
    resolveTheme,
  }
}
