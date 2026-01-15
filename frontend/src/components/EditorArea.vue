<script setup>
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import CodeEditor from './CodeEditor.vue'
import { OpenFolder } from '../../wailsjs/go/main/App'

const { t } = useI18n()

const props = defineProps({
  currentSessionId: String
})

const emit = defineEmits(['openFolder', 'update:workDir', 'activeFileChange'])

// 打开的文件
const openFiles = ref([])
const activeFile = ref(null)

// 监听活动文件变化
watch(activeFile, (file) => {
  if (file && props.currentSessionId) {
    emit('activeFileChange', file.path)
  }
})

const openFile = (file) => {
  const existing = openFiles.value.find(f => f.path === file.path)
  if (existing) {
    activeFile.value = existing
  } else {
    openFiles.value.push(file)
    activeFile.value = file
  }
}

const closeFileTab = (file) => {
  const index = openFiles.value.findIndex(f => f.path === file.path)
  if (index > -1) {
    openFiles.value.splice(index, 1)
    if (activeFile.value?.path === file.path) {
      activeFile.value = openFiles.value.length > 0 
        ? openFiles.value[Math.min(index, openFiles.value.length - 1)] 
        : null
    }
  }
}

const handleOpenFolder = async () => {
  try {
    const dir = await OpenFolder()
    if (dir) emit('update:workDir', dir)
  } catch (e) {
    console.error('打开目录失败:', e)
  }
}

defineExpose({ openFile, activeFile })
</script>

<template>
  <div class="editor-area">
    <template v-if="openFiles.length > 0">
      <div class="editor-tabs">
        <div 
          v-for="file in openFiles" 
          :key="file.path"
          :class="['tab', { active: activeFile?.path === file.path }]"
          @click="activeFile = file"
        >
          <span class="tab-name">{{ file.name }}</span>
          <button class="tab-close" @click.stop="closeFileTab(file)">×</button>
        </div>
      </div>
      <div class="editor-content">
        <CodeEditor 
          v-for="file in openFiles"
          :key="file.path"
          v-show="activeFile?.path === file.path"
          :file="file" 
        />
      </div>
    </template>
    
    <div v-else class="editor-placeholder">
      <div class="placeholder-content">
        <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
          <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
        </svg>
        <p>{{ t('editor.openFile') }}</p>
        <button class="btn-open-folder" @click="handleOpenFolder">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
            <line x1="12" y1="11" x2="12" y2="17"/><line x1="9" y1="14" x2="15" y2="14"/>
          </svg>
          {{ t('editor.openFolder') }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.editor-area { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
.editor-tabs { display: flex; background: var(--bg-surface); border-bottom: 1px solid var(--border-default); height: 35px; overflow-x: auto; flex-shrink: 0; }
.editor-tabs::-webkit-scrollbar { height: 0; }
.tab { display: flex; align-items: center; gap: 6px; padding: 0 12px; height: 100%; background: var(--bg-surface); border-right: 1px solid var(--border-default); font-size: 12px; color: var(--text-muted); cursor: pointer; white-space: nowrap; }
.tab:hover { color: var(--text-primary); }
.tab.active { background: var(--bg-base); color: var(--text-primary); }
.tab-close { background: transparent; border: none; color: var(--text-muted); cursor: pointer; font-size: 14px; padding: 0 2px; border-radius: 2px; opacity: 0; }
.tab:hover .tab-close, .tab.active .tab-close { opacity: 1; }
.tab-close:hover { background: var(--bg-hover); color: var(--text-primary); }
.editor-content { flex: 1; display: flex; flex-direction: column; overflow: hidden; position: relative; }
.editor-placeholder { flex: 1; display: flex; align-items: center; justify-content: center; text-align: center; color: var(--text-muted); }
.placeholder-content svg { opacity: 0.3; margin-bottom: 16px; }
.placeholder-content p { font-size: 14px; margin-bottom: 16px; }
.btn-open-folder { display: flex; align-items: center; gap: 8px; padding: 8px 16px; background: var(--accent-button); border: none; border-radius: 6px; color: white; font-size: 13px; cursor: pointer; transition: all 0.15s; }
.btn-open-folder:hover { background: var(--accent-primary); }
</style>
