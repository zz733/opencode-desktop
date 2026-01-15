import { ref, watch } from 'vue'

// 主题定义
const themes = {
  kiro: {
    name: 'Kiro Dark',
    colors: {
      '--bg-base': '#19161d',
      '--bg-surface': '#211d25',
      '--bg-elevated': '#28242e',
      '--bg-hover': '#322e3a',
      '--bg-active': '#3c3846',
      '--bg-input': '#28242e',
      '--text-primary': '#ffffff',
      '--text-secondary': '#938f9b',
      '--text-muted': '#6b6773',
      '--border-default': '#28242e',
      '--border-subtle': '#322e3a',
      '--accent-primary': '#b080ff',
      '--accent-hover': '#c4a0ff',
      '--accent-button': '#7138cc',
      '--green': '#80ffb5',
      '--blue': '#8dc8fb',
      '--yellow': '#ffcf99',
      '--red': '#ff8080',
      '--pink': '#ff80b5',
      '--cyan': '#80f4ff',
      '--purple': '#e2d3fe',
    }
  },
  ideaNewUI: {
    name: 'IDEA New UI',
    colors: {
      '--bg-base': '#1e1f22',
      '--bg-surface': '#2b2d30',
      '--bg-elevated': '#393b40',
      '--bg-hover': '#4e5157',
      '--bg-active': '#4e5157',
      '--bg-input': '#393b40',
      '--text-primary': '#dfe1e5',
      '--text-secondary': '#a8adbd',
      '--text-muted': '#6f737a',
      '--border-default': '#393b40',
      '--border-subtle': '#43454a',
      '--accent-primary': '#3574f0',
      '--accent-hover': '#467ff2',
      '--accent-button': '#3574f0',
      '--green': '#6aab73',
      '--blue': '#6897bb',
      '--yellow': '#c9a26d',
      '--red': '#f75464',
      '--pink': '#c77dbb',
      '--cyan': '#299999',
      '--purple': '#9876aa',
    }
  },
  ideaLight: {
    name: 'IDEA Light',
    colors: {
      '--bg-base': '#f7f8fa',
      '--bg-surface': '#ffffff',
      '--bg-elevated': '#f0f0f0',
      '--bg-hover': '#e8e8e8',
      '--bg-active': '#d4d4d4',
      '--bg-input': '#ffffff',
      '--text-primary': '#000000',
      '--text-secondary': '#6e6e6e',
      '--text-muted': '#999999',
      '--border-default': '#d1d1d1',
      '--border-subtle': '#e5e5e5',
      '--accent-primary': '#2470b3',
      '--accent-hover': '#1a5a99',
      '--accent-button': '#2470b3',
      '--green': '#067d17',
      '--blue': '#0033b3',
      '--yellow': '#9e880d',
      '--red': '#cf222e',
      '--pink': '#871094',
      '--cyan': '#00627a',
      '--purple': '#871094',
    }
  }
}

// 从 localStorage 读取主题
const currentTheme = ref(localStorage.getItem('theme') || 'kiro')

// 应用主题
function applyTheme(themeName) {
  const theme = themes[themeName]
  if (!theme) return
  
  const root = document.documentElement
  for (const [key, value] of Object.entries(theme.colors)) {
    root.style.setProperty(key, value)
  }
}

// 设置主题
function setTheme(themeName) {
  if (!themes[themeName]) return
  currentTheme.value = themeName
  localStorage.setItem('theme', themeName)
  applyTheme(themeName)
}

// 获取所有主题列表
function getThemes() {
  return Object.entries(themes).map(([id, theme]) => ({
    id,
    name: theme.name
  }))
}

// 初始化主题
function initTheme() {
  applyTheme(currentTheme.value)
}

export function useTheme() {
  return {
    currentTheme,
    themes: getThemes(),
    setTheme,
    initTheme
  }
}
