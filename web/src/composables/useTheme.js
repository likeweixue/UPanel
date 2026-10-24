import { reactive } from 'vue'

const THEME_KEY = 'upanel_theme'
const DENSITY_KEY = 'upanel_density'
const RADIUS_KEY = 'upanel_radius'

const media = typeof window !== 'undefined' && window.matchMedia
  ? window.matchMedia('(prefers-color-scheme: dark)')
  : null

export const themeState = reactive({
  mode: localStorage.getItem(THEME_KEY) || 'light',
  density: localStorage.getItem(DENSITY_KEY) || 'comfortable',
  systemDark: media ? media.matches : false,
  cornerRadius: Math.min(16, Math.max(0, Number(localStorage.getItem(RADIUS_KEY) ?? 4) || 0)),
})

function apply() {
  const effective =
    themeState.mode === 'system' ? (themeState.systemDark ? 'dark' : 'light') : themeState.mode
  document.documentElement.setAttribute('data-theme', effective === 'dark' ? 'dark' : 'light')
  document.documentElement.setAttribute('data-density', themeState.density)
  document.documentElement.style.setProperty('--radius-ui', `${themeState.cornerRadius}px`)
  const meta = document.querySelector('meta[name="theme-color"]')
  if (meta) meta.setAttribute('content', effective === 'dark' ? '#171a21' : '#ffffff')
}

export function setTheme(mode) {
  themeState.mode = ['light', 'dark', 'system'].includes(mode) ? mode : 'light'
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
  const current =
    themeState.mode === 'system' ? (themeState.systemDark ? 'dark' : 'light') : themeState.mode
  setTheme(current === 'dark' ? 'light' : 'dark')
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
  return { themeState, setTheme, setDensity, setCornerRadius, toggleTheme, initTheme }
}
