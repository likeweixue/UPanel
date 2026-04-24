// 主题定义
export const themes = {
  light: {
    name: '浅色',
    description: '明亮清爽的默认主题',
    colors: {
      primary: '#477779',
      primaryLight: '#5e8a8c',
      primaryDark: '#2d5a5c',
      bg: '#f1f1f1',
      cardBg: '#ffffff',
      text: '#1f2937',
      textSecondary: '#6b7280',
      border: '#e5e7eb',
      success: '#2e7d32',
      warning: '#ed6c02',
      danger: '#c62828',
      info: '#477779'
    }
  },
  davinci: {
    name: 'DaVinci Resolve',
    description: '专业视频编辑软件配色风格',
    colors: {
      primary: '#e67e22',
      primaryLight: '#f39c12',
      primaryDark: '#d35400',
      bg: '#1a1a1a',
      cardBg: '#2a2a2a',
      text: '#e0e0e0',
      textSecondary: '#a0a0a0',
      border: '#3a3a3a',
      success: '#5a8a5a',
      warning: '#f1c40f',
      danger: '#e74c3c',
      info: '#e67e22'
    }
  }
}

// 当前主题
let currentTheme = 'light'

// 获取当前主题
export const getCurrentTheme = () => {
  return currentTheme
}

// 应用主题
export const applyTheme = (themeName) => {
  const theme = themes[themeName]
  if (!theme) return
  
  currentTheme = themeName
  localStorage.setItem('theme', themeName)
  
  // 设置 CSS 变量
  const root = document.documentElement
  Object.entries(theme.colors).forEach(([key, value]) => {
    root.style.setProperty(`--theme-${key}`, value)
  })
  
  // 添加主题类名
  document.body.classList.remove('theme-light', 'theme-davinci')
  document.body.classList.add(`theme-${themeName}`)
  
  // 触发主题变更事件
  window.dispatchEvent(new CustomEvent('theme-change', { detail: themeName }))
}

// 初始化主题
export const initTheme = () => {
  const saved = localStorage.getItem('theme')
  const themeName = saved && themes[saved] ? saved : 'light'
  applyTheme(themeName)
}
