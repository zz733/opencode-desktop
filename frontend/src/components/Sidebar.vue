<script setup>
import { ref, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ListDir, OpenFolder } from '../../wailsjs/go/main/App'
import FileTreeItem from './FileTreeItem.vue'

const { t } = useI18n()

const props = defineProps({
  activeTab: String,
  workDir: String
})

const emit = defineEmits(['openFile', 'update:workDir'])

const localWorkDir = ref('')
const files = ref([])
const loading = ref(false)
const expandedFolders = ref(new Set())

const loadDir = async (dir = '') => {
  loading.value = true
  try {
    const targetDir = dir || localWorkDir.value || ''
    const items = await ListDir(targetDir)
    if (dir === '' || dir === localWorkDir.value) {
      files.value = items || []
    }
    return items || []
  } catch (e) {
    console.error('加载目录失败:', e)
    return []
  } finally {
    loading.value = false
  }
}

const refreshDir = async (dir) => {
  localWorkDir.value = dir
  expandedFolders.value.clear()
  await loadDir()
}

defineExpose({ refreshDir })

watch(() => props.workDir, async (newDir) => {
  if (newDir && newDir !== localWorkDir.value) {
    localWorkDir.value = newDir
    expandedFolders.value.clear()
    await loadDir()
  }
})

onMounted(async () => {
  if (props.workDir) {
    localWorkDir.value = props.workDir
    await loadDir()
  }
})

const toggleFolder = async (item) => {
  if (item.type !== 'folder') return
  const path = item.path
  if (expandedFolders.value.has(path)) {
    expandedFolders.value.delete(path)
  } else {
    expandedFolders.value.add(path)
    if (!item.children || item.children.length === 0) {
      item.children = await loadDir(path)
    }
  }
  // 触发响应式更新
  expandedFolders.value = new Set(expandedFolders.value)
}

const openFile = (item) => {
  if (item.type === 'file') {
    emit('openFile', item)
  }
}

const selectFolder = async () => {
  const dir = await OpenFolder()
  if (dir) {
    localWorkDir.value = dir
    expandedFolders.value.clear()
    emit('update:workDir', dir)
    await loadDir()
  }
}

const getDirName = () => {
  if (!localWorkDir.value) return ''
  return localWorkDir.value.split('/').pop() || localWorkDir.value.split('\\').pop() || 'ROOT'
}
</script>

<template>
  <aside class="sidebar">
    <div class="sidebar-header">
      <span>{{ activeTab === 'files' ? t('sidebar.explorer').toUpperCase() : t('sidebar.' + activeTab).toUpperCase() }}</span>
    </div>
    
    <div v-if="activeTab === 'files'" class="file-tree">
      <div v-if="!localWorkDir" class="empty-tree">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
          <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
        </svg>
        <p>{{ t('sidebar.noFolder') }}</p>
        <button @click="selectFolder">{{ t('editor.openFolder') }}</button>
      </div>
      
      <div v-else class="file-tree-content">
        <div class="section-header">
          <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M6 9l6 6 6-6"/></svg>
          <span class="dir-name">{{ getDirName().toUpperCase() }}</span>
          <button class="btn-open-folder" @click="selectFolder">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
              <line x1="12" y1="11" x2="12" y2="17"/><line x1="9" y1="14" x2="15" y2="14"/>
            </svg>
          </button>
        </div>
      
        <div class="tree-items" v-if="files.length">
          <FileTreeItem
            v-for="item in files"
            :key="item.path"
            :item="item"
            :depth="0"
            :expandedFolders="expandedFolders"
            @openFile="openFile"
            @toggleFolder="toggleFolder"
          />
        </div>
        <div v-else-if="!loading" class="empty-files"><p>{{ t('sidebar.emptyFolder') }}</p></div>
      </div>
    </div>
    <div v-else class="placeholder"><p>{{ activeTab }}</p></div>
  </aside>
</template>

<style scoped>
.sidebar { flex: 1; background: var(--bg-surface); display: flex; flex-direction: column; overflow: hidden; }
.sidebar-header { padding: 10px 20px; font-size: 11px; font-weight: 400; letter-spacing: 1.2px; color: var(--text-secondary); }
.file-tree { flex: 1; overflow-y: auto; display: flex; flex-direction: column; }
.file-tree-content { flex: 1; display: flex; flex-direction: column; }
.section-header { display: flex; align-items: center; gap: 4px; padding: 4px 8px; font-size: 11px; font-weight: 700; color: var(--text-primary); }
.dir-name { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.btn-open-folder { background: transparent; border: none; color: var(--text-muted); cursor: pointer; padding: 2px; border-radius: 4px; }
.btn-open-folder:hover { background: var(--bg-active); color: var(--text-primary); }
.tree-items { padding-left: 8px; flex: 1; overflow-y: auto; }
.empty-tree { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 40px 20px; color: var(--text-muted); font-size: 13px; }
.empty-tree svg { opacity: 0.3; margin-bottom: 12px; }
.empty-tree button { margin-top: 12px; padding: 6px 12px; background: var(--accent-primary); border: none; border-radius: 4px; color: white; font-size: 12px; cursor: pointer; }
.empty-tree button:hover { background: var(--accent-hover); }
.empty-files { padding: 20px; text-align: center; color: var(--text-muted); font-size: 12px; }
.placeholder { flex: 1; display: flex; align-items: center; justify-content: center; color: var(--text-muted); font-size: 13px; }
</style>
