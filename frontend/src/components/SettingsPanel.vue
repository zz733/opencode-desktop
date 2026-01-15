<script setup>
import { useI18n } from 'vue-i18n'
import { languages, setLocale } from '../i18n'

const { t, locale } = useI18n()
const emit = defineEmits(['close'])

const changeLanguage = (code) => {
  setLocale(code)
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
  background: #211d25;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.settings-header {
  padding: 12px 16px;
  font-size: 11px;
  font-weight: 500;
  letter-spacing: 0.5px;
  color: #938f9b;
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
  color: #ffffff;
  margin-bottom: 12px;
  padding: 0 4px;
}

.setting-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 12px;
  background: #28242e;
  border-radius: 6px;
  margin-bottom: 8px;
}

.setting-label {
  font-size: 13px;
  color: #ffffff;
}

.setting-control select {
  padding: 6px 10px;
  background: #28242e;
  border: 1px solid #28242e;
  border-radius: 4px;
  color: #ffffff;
  font-size: 12px;
  cursor: pointer;
  outline: none;
}

.setting-control select:hover {
  border-color: #6b6773;
}

.setting-control select:focus {
  border-color: #b080ff;
}

.setting-control select option {
  background: #28242e;
  color: #ffffff;
}
</style>
