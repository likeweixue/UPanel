import { ref, watch } from 'vue'

const isDark = ref(localStorage.getItem('theme') === 'dark')

// 监听系统主题变化
const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
mediaQuery.addEventListener('change', (e) => {
  if (localStorage.getItem('theme') === 'auto') {
    isDark.value = e.matches
  }
})

// 应用主题到 DOM
const applyTheme = (dark) => {
  if (dark) {
    document.documentElement.classList.add('dark')
    document.body.classList.add('dark-mode')
  } else {
    document.documentElement.classList.remove('dark')
    document.body.classList.remove('dark-mode')
  }
  localStorage.setItem('theme', dark ? 'dark' : 'light')
}

// 切换主题
const toggleTheme = () => {
  isDark.value = !isDark.value
  applyTheme(isDark.value)
}

// 初始化主题
const initTheme = () => {
  const saved = localStorage.getItem('theme')
  if (saved === 'dark') {
    isDark.value = true
    applyTheme(true)
  } else if (saved === 'light') {
    isDark.value = false
    applyTheme(false)
  } else {
    // 跟随系统
    isDark.value = mediaQuery.matches
    applyTheme(isDark.value)
  }
}

export function useTheme() {
  return {
    isDark,
    toggleTheme,
    initTheme
  }
}
