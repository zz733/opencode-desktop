<script setup>
import { useI18n } from 'vue-i18n'
import { languages, setLocale } from '../i18n'
import { useTheme } from '../composables/useTheme'

const { t, locale } = useI18n()
const { currentTheme, themes, setTheme } = useTheme()
const emit = defineEmits(['close'])

const changeLanguage = (code) => {
  setLocale(code)
}

const changeTheme = (themeId) => {
  setTheme(themeId)
}
</script>

<template>
  <aside class="settings-panel">
    <div class="settings-header">
      <span>{{ t('settings.title') }}</span>
    </div>
    
    <div class="settings-content">
      <div class="settings-section">
        <div class="section-title">{{ t('settings.general') }}</div>
        
        <div class="setting-item">
          <div class="setting-label">{{ t('settings.theme') }}</div>
          <div class="setting-control">
            <select :value="currentTheme" @change="changeTheme($event.target.value)">
              <option v-for="theme in themes" :key="theme.id" :value="theme.id">
                {{ theme.name }}
              </option>
            </select>
          </div>
        </div>
        
        <div class="setting-item">
          <div class="setting-label">{{ t('settings.language') }}</div>
          <div class="setting-control">
            <select :value="locale" @change="changeLanguage($event.target.value)">
              <option v-for="lang in languages" :key="lang.code" :value="lang.code">
                {{ lang.name }}
              </option>
            </select>
          </div>
        </div>
      </div>
    </div>
  </aside>
</template>

<style scoped>
.settings-panel {
  flex: 1;
  background: var(--bg-surface);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.settings-header {
  padding: 12px 16px;
  font-size: 11px;
  font-weight: 500;
  letter-spacing: 0.5px;
  color: var(--text-secondary);
  text-transform: uppercase;
}

.settings-content {
  flex: 1;
  overflow-y: auto;
  padding: 0 12px;
}

.settings-section {
  margin-bottom: 24px;
}

.section-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 12px;
  padding: 0 4px;
}

.setting-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 12px;
  background: var(--bg-elevated);
  border-radius: 6px;
  margin-bottom: 8px;
}

.setting-label {
  font-size: 13px;
  color: var(--text-primary);
}

.setting-control select {
  padding: 6px 10px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-default);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: 12px;
  cursor: pointer;
  outline: none;
}

.setting-control select:hover {
  border-color: var(--text-muted);
}

.setting-control select:focus {
  border-color: var(--accent-primary);
}

.setting-control select option {
  background: var(--bg-elevated);
  color: var(--text-primary);
}
</style>
