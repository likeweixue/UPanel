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
  dark: {
    name: '暗色',
    description: '护眼暗黑模式',
    colors: {
      primary: '#5e8a8c',
      primaryLight: '#7a9fa1',
      primaryDark: '#3a6e70',
      bg: '#1a1a1a',
      cardBg: '#1e1e1e',
      text: '#e5e7eb',
      textSecondary: '#9ca3af',
      border: '#2a2a2a',
      success: '#34ce57',
      warning: '#f9c74f',
      danger: '#c62828',
      info: '#5e8a8c'
    }
  },
  gold: {
    name: '黑金',
    description: '奢华黑金主题',
    colors: {
      primary: '#d4af37',
      primaryLight: '#f0c674',
      primaryDark: '#b8960c',
      bg: '#0a0a0a',
      cardBg: '#111111',
      text: '#e8e8e8',
      textSecondary: '#a0a0a0',
      border: '#2a2a2a',
      success: '#2e7d32',
      warning: '#ed6c02',
      danger: '#c62828',
      info: '#d4af37'
    }
  },
  blue: {
    name: '深海蓝',
    description: '深邃蓝色主题',
    colors: {
      primary: '#1a5276',
      primaryLight: '#2e86c1',
      primaryDark: '#0e2f44',
      bg: '#0a1620',
      cardBg: '#0d1c2a',
      text: '#e0e0e0',
      textSecondary: '#90a4ae',
      border: '#1a2a3a',
      success: '#2e7d32',
      warning: '#ed6c02',
      danger: '#c62828',
      info: '#1a5276'
    }
  },
  green: {
    name: '森林绿',
    description: '护眼绿色主题',
    colors: {
      primary: '#1b5e20',
      primaryLight: '#2e7d32',
      primaryDark: '#0d3b12',
      bg: '#0a1a0a',
      cardBg: '#0d1f0d',
      text: '#e0e0e0',
      textSecondary: '#90a47e',
      border: '#1a2a1a',
      success: '#2e7d32',
      warning: '#ed6c02',
      danger: '#c62828',
      info: '#1b5e20'
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
  document.body.classList.remove('theme-light', 'theme-dark', 'theme-gold', 'theme-blue', 'theme-green')
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
