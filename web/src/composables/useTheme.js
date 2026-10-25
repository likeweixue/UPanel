import { reactive } from 'vue'

const THEME_KEY = 'upanel_theme'
const DENSITY_KEY = 'upanel_density'
const RADIUS_KEY = 'upanel_radius'

export const LIGHT_THEMES = ['light', 'retro-yellow', 'green-paper']
export const DARK_THEMES = ['dark', 'black-gold', 'ember']
export const THEMES = [...LIGHT_THEMES, ...DARK_THEMES, 'system']

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

function normalizeTheme(mode) {
  return THEMES.includes(mode) ? mode : 'light'
}

export function isDarkTheme(theme) {
  return DARK_THEMES.includes(theme)
}

export function resolveTheme(mode, systemDark) {
  if (mode === 'system') return systemDark ? 'dark' : 'light'
  return normalizeTheme(mode)
}

const savedMode = typeof window !== 'undefined' ? normalizeTheme(localStorage.getItem(THEME_KEY)) : 'light'
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
  const meta = document.querySelector('meta[name="theme-color"]')
  if (meta) meta.setAttribute('content', THEME_COLORS[effective] || THEME_COLORS.light)
}

export function setTheme(mode) {
  themeState.mode = normalizeTheme(mode)
  localStorage.setItem(THEME_KEY, themeState.mode)
  apply()
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
  if (media) {
    media.addEventListener('change', (event) => {
      themeState.systemDark = event.matches
      if (themeState.mode === 'system') apply()
    })
  }
  apply()
}

export function useTheme() {
  return {
    themeState,
    setTheme,
    setDensity,
    setCornerRadius,
    toggleTheme,
    initTheme,
    isDarkTheme,
    resolveTheme,
  }
}
