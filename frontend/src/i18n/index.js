import { createI18n } from 'vue-i18n'
import zhCN from './zh-CN'
import en from './en'
import ja from './ja'

// 获取系统语言
function getSystemLocale() {
  const lang = navigator.language || navigator.userLanguage
  if (lang.startsWith('zh')) return 'zh-CN'
  if (lang.startsWith('ja')) return 'ja'
  return 'en'
}

// 获取保存的语言设置
function getSavedLocale() {
  return localStorage.getItem('locale')
}

// 保存语言设置
export function saveLocale(locale) {
  localStorage.setItem('locale', locale)
}

// 获取当前应该使用的语言
function getCurrentLocale() {
  return getSavedLocale() || getSystemLocale()
}

export const i18n = createI18n({
  legacy: false,
  locale: getCurrentLocale(),
  fallbackLocale: 'en',
  messages: {
    'zh-CN': zhCN,
    'en': en,
    'ja': ja,
  },
})

export const languages = [
  { code: 'zh-CN', name: '简体中文' },
  { code: 'en', name: 'English' },
  { code: 'ja', name: '日本語' },
]

export function setLocale(locale) {
  i18n.global.locale.value = locale
  saveLocale(locale)
}
