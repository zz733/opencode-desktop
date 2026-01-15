<script setup>
import { ref, watch, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import * as monaco from 'monaco-editor'
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

// Diff 视图
const showDiff = ref(false)
const diffEdit = ref(null)
const diffEditorContainer = ref(null)
const diffEditor = shallowRef(null)

// 编辑器引用
const editorRefs = ref({})

// 监听活动文件变化
watch(activeFile, (file) => {
  if (file && props.currentSessionId) {
    emit('activeFileChange', file.path)
  }
})

const openFile = (file) => {
  showDiff.value = false
  const existing = openFiles.value.find(f => f.path === file.path)
  if (existing) {
    activeFile.value = existing
  } else {
    openFiles.value.push(file)
    activeFile.value = file
  }
}

// 打开文件并显示 diff
const openFileWithDiff = (edit) => {
  diffEdit.value = edit
  showDiff.value = true
  
  // 延迟创建 diff editor
  setTimeout(() => {
    createDiffEditor(edit)
  }, 50)
}

// 创建 diff 编辑器
const createDiffEditor = (edit) => {
  if (!diffEditorContainer.value) return
  
  if (diffEditor.value) {
    diffEditor.value.dispose()
  }
  
  const ext = edit.filename.split('.').pop()?.toLowerCase()
  const langMap = { 'go': 'go', 'js': 'javascript', 'ts': 'typescript', 'vue': 'html', 'json': 'json', 'py': 'python' }
  const lang = langMap[ext] || 'plaintext'
  
  diffEditor.value = monaco.editor.createDiffEditor(diffEditorContainer.value, {
    theme: 'vs-dark',
    fontSize: 13,
    fontFamily: "'Menlo', 'Monaco', 'Courier New', monospace",
    readOnly: true,
    automaticLayout: true,
    renderSideBySide: true,
    minimap: { enabled: false },
    scrollBeyondLastLine: false,
  })
  
  diffEditor.value.setModel({
    original: monaco.editor.createModel(edit.oldContent, lang),
    modified: monaco.editor.createModel(edit.newContent, lang),
  })
}

// 关闭 diff 视图
const closeDiff = () => {
  showDiff.value = false
  if (diffEditor.value) {
    const model = diffEditor.value.getModel()
    model?.original?.dispose()
    model?.modified?.dispose()
    diffEditor.value.dispose()
    diffEditor.value = null
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

// 重新加载当前文件
const reloadCurrentFile = () => {
  if (activeFile.value) {
    const ref = editorRefs.value[activeFile.value.path]
    ref?.reloadFile()
  }
}

// 注册编辑器引用
const setEditorRef = (path, el) => {
  if (el) editorRefs.value[path] = el
  else delete editorRefs.value[path]
}

defineExpose({ openFile, activeFile, openFileWithDiff, reloadCurrentFile })
</script>

<template>
  <div class="editor-area">
    <!-- Diff 视图 -->
    <div v-if="showDiff" class="diff-view">
      <div class="diff-header">
        <div class="diff-title">
          <span class="diff-label original">修改前</span>
          <span class="diff-arrow">→</span>
          <span class="diff-label modified">修改后</span>
          <span class="diff-filename">{{ diffEdit?.filename }}</span>
        </div>
        <button class="btn-close-diff" @click="closeDiff">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M18 6L6 18M6 6l12 12"/>
          </svg>
        </button>
      </div>
      <div ref="diffEditorContainer" class="diff-container"></div>
    </div>
    
    <!-- 正常编辑器视图 -->
    <template v-else-if="openFiles.length > 0">
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
          :ref="el => setEditorRef(file.path, el)"
          :file="file"
          :sessionId="props.currentSessionId"
        />
      </div>
    </template>
    
    <!-- 空状态 -->
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

/* Diff 视图 */
.diff-view { flex: 1; display: flex; flex-direction: column; min-height: 0; }
.diff-header { display: flex; align-items: center; justify-content: space-between; padding: 8px 12px; background: var(--bg-surface); border-bottom: 1px solid var(--border-default); }
.diff-title { display: flex; align-items: center; gap: 8px; font-size: 12px; }
.diff-label { padding: 2px 8px; border-radius: 3px; font-size: 11px; }
.diff-label.original { background: rgba(255, 128, 128, 0.2); color: var(--red); }
.diff-label.modified { background: rgba(128, 255, 181, 0.2); color: var(--green); }
.diff-arrow { color: var(--text-muted); }
.diff-filename { color: var(--text-secondary); margin-left: 8px; }
.btn-close-diff { background: transparent; border: none; color: var(--text-muted); cursor: pointer; padding: 4px; border-radius: 4px; }
.btn-close-diff:hover { background: var(--bg-hover); color: var(--text-primary); }
.diff-container { flex: 1; min-height: 0; }

/* 标签栏 */
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
