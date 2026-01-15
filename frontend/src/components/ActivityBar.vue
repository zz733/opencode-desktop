<script setup>
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

defineProps({
  activeTab: String,
  showSidebar: Boolean,
  showSettings: Boolean
})

const emit = defineEmits(['change'])

const tabs = [
  { id: 'files', icon: 'folder', titleKey: 'sidebar.explorer' },
  { id: 'search', icon: 'search', titleKey: 'sidebar.search' },
  { id: 'git', icon: 'git', titleKey: 'sidebar.git' },
]
</script>

<template>
  <aside class="activity-bar">
    <div class="top-icons">
      <div 
        v-for="tab in tabs" 
        :key="tab.id"
        :class="['icon-btn', { active: activeTab === tab.id && showSidebar && !showSettings }]"
        :title="t(tab.titleKey)"
        @click="emit('change', tab.id)"
      >
        <svg v-if="tab.icon === 'folder'" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
        </svg>
        <svg v-if="tab.icon === 'search'" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/>
        </svg>
        <svg v-if="tab.icon === 'git'" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <circle cx="18" cy="18" r="3"/><circle cx="6" cy="6" r="3"/><path d="M6 21V9a9 9 0 0 0 9 9"/>
        </svg>
      </div>
    </div>
    
    <div class="bottom-icons">
      <div 
        :class="['icon-btn', { active: showSettings && showSidebar }]" 
        :title="t('sidebar.settings')" 
        @click="emit('change', 'settings')"
      >
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <circle cx="12" cy="12" r="3"/>
          <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>
        </svg>
      </div>
    </div>
  </aside>
</template>

<style scoped>
.activity-bar {
  width: 48px;
  background: var(--bg-surface);
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 4px 0;
  border-right: 1px solid var(--border-default);
}

.top-icons, .bottom-icons {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0;
}

.bottom-icons {
  margin-top: auto;
}

.icon-btn {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary);
  cursor: pointer;
  position: relative;
}

.icon-btn:hover {
  color: var(--text-primary);
}

.icon-btn.active {
  color: var(--text-primary);
}

.icon-btn.active::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 2px;
  background: var(--accent-primary);
}
</style>
