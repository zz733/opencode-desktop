<script setup>
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

defineProps({
  activeTab: String
})

const files = ref([
  { name: 'src', type: 'folder', expanded: true, children: [
    { name: 'main.go', type: 'file' },
    { name: 'app.go', type: 'file' },
  ]},
  { name: 'frontend', type: 'folder', expanded: false, children: [] },
  { name: 'go.mod', type: 'file' },
  { name: 'README.md', type: 'file' },
])

const toggleFolder = (item) => {
  if (item.type === 'folder') {
    item.expanded = !item.expanded
  }
}
</script>

<template>
  <aside class="sidebar">
    <div class="sidebar-header">
      <span>{{ activeTab === 'files' ? t('sidebar.explorer').toUpperCase() : t('sidebar.' + activeTab).toUpperCase() }}</span>
    </div>
    
    <div v-if="activeTab === 'files'" class="file-tree">
      <div class="section-header">
        <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M6 9l6 6 6-6"/>
        </svg>
        <span>OPENCODE-DESKTOP</span>
      </div>
      
      <div class="tree-items">
        <template v-for="item in files" :key="item.name">
          <div class="tree-item" @click="toggleFolder(item)">
            <svg v-if="item.type === 'folder'" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" :class="{ rotated: item.expanded }">
              <path d="M9 18l6-6-6-6"/>
            </svg>
            <span v-else class="spacer"></span>
            
            <svg v-if="item.type === 'folder'" width="16" height="16" viewBox="0 0 24 24" fill="#e8a838" stroke="none">
              <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
            </svg>
            <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#75beff" stroke-width="1.5">
              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 2v6h6"/>
            </svg>
            
            <span class="name">{{ item.name }}</span>
          </div>
          
          <template v-if="item.type === 'folder' && item.expanded && item.children">
            <div v-for="child in item.children" :key="child.name" class="tree-item nested">
              <span class="spacer"></span>
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#75beff" stroke-width="1.5">
                <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 2v6h6"/>
              </svg>
              <span class="name">{{ child.name }}</span>
            </div>
          </template>
        </template>
      </div>
    </div>
    
    <div v-else class="placeholder">
      <p>{{ activeTab }}</p>
    </div>
  </aside>
</template>

<style scoped>
.sidebar {
  flex: 1;
  background: var(--bg-surface);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.sidebar-header {
  padding: 10px 20px;
  font-size: 11px;
  font-weight: 400;
  letter-spacing: 1.2px;
  color: var(--text-secondary);
  text-transform: uppercase;
}

.file-tree {
  flex: 1;
  overflow-y: auto;
}

.section-header {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  font-size: 11px;
  font-weight: 700;
  color: var(--text-primary);
  cursor: pointer;
}

.section-header:hover {
  background: var(--bg-hover);
}

.tree-items {
  padding-left: 8px;
}

.tree-item {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px 2px 16px;
  cursor: pointer;
}

.tree-item:hover {
  background: var(--bg-hover);
}

.tree-item.nested {
  padding-left: 32px;
}

.tree-item svg.rotated {
  transform: rotate(90deg);
}

.spacer {
  width: 16px;
}

.name {
  font-size: 13px;
  color: var(--text-primary);
}

.placeholder {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
  font-size: 13px;
}
</style>
