<script setup>
import { ref, watch, shallowRef, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import * as monaco from 'monaco-editor'
import CodeEditor from './CodeEditor.vue'
import { OpenFolder } from '../../wailsjs/go/main/App'

const { t } = useI18n()

const props = defineProps({
  currentSessionId: String
})

const emit = defineEmits(['openFolder', 'update:workDir', 'activeFileChange', 'toggleMaximize', 'runCommand'])

// 打开的文件
const openFiles = ref([])
const activeFile = ref(null)
const isMaximized = ref(false)

// 判断是否是图片文件
const isImageFile = (name) => {
  const ext = name?.split('.').pop()?.toLowerCase()
  return ['png', 'jpg', 'jpeg', 'gif', 'svg', 'webp', 'ico', 'bmp'].includes(ext)
}

// 判断是否是二进制文件
const isBinaryFile = (name) => {
  const ext = name?.split('.').pop()?.toLowerCase()
  const binaryExts = [
    'exe', 'dll', 'so', 'dylib', 'bin', 'dat',
    'zip', 'tar', 'gz', 'rar', '7z', 'bz2',
    'pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx',
    'mp3', 'mp4', 'avi', 'mov', 'mkv', 'wav', 'flac',
    'ttf', 'otf', 'woff', 'woff2', 'eot',
    'db', 'sqlite', 'sqlite3',
    'class', 'jar', 'war',
    'o', 'a', 'lib',
    'pyc', 'pyo',
  ]
  return binaryExts.includes(ext)
}

// 当前活动文件是否是图片
const isActiveFileImage = computed(() => {
  return activeFile.value && isImageFile(activeFile.value.name)
})

// 双击标签栏最大化/还原
const handleTabDoubleClick = () => {
  isMaximized.value = !isMaximized.value
  emit('toggleMaximize', isMaximized.value)
}

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
  
  // 检查是否是二进制文件（非图片）
  if (isBinaryFile(file.name)) {
    alert(`无法打开二进制文件: ${file.name}\n\n此类型文件不支持在编辑器中查看。`)
    return
  }
  
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
    // 如果没有打开的文件了，退出最大化
    if (openFiles.value.length === 0 && isMaximized.value) {
      isMaximized.value = false
      emit('toggleMaximize', false)
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

// 运行命令
const handleRunCommand = (command) => {
  emit('runCommand', command)
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

// 右键菜单
const contextMenu = ref({ show: false, x: 0, y: 0, file: null, index: -1 })

const showContextMenu = (e, file, index) => {
  e.preventDefault()
  contextMenu.value = {
    show: true,
    x: e.clientX,
    y: e.clientY,
    file,
    index
  }
}

const hideContextMenu = () => {
  contextMenu.value.show = false
}

// 关闭当前标签
const closeTab = () => {
  if (contextMenu.value.file) {
    closeFileTab(contextMenu.value.file)
  }
  hideContextMenu()
}

// 关闭左边的标签
const closeTabsToLeft = () => {
  const index = contextMenu.value.index
  if (index > 0) {
    openFiles.value.splice(0, index)
    // 如果当前活动文件被关闭，选择第一个
    if (!openFiles.value.find(f => f.path === activeFile.value?.path)) {
      activeFile.value = openFiles.value[0] || null
    }
  }
  hideContextMenu()
}

// 关闭右边的标签
const closeTabsToRight = () => {
  const index = contextMenu.value.index
  if (index < openFiles.value.length - 1) {
    openFiles.value.splice(index + 1)
    // 如果当前活动文件被关闭，选择最后一个
    if (!openFiles.value.find(f => f.path === activeFile.value?.path)) {
      activeFile.value = openFiles.value[openFiles.value.length - 1] || null
    }
  }
  hideContextMenu()
}

// 关闭其他标签
const closeOtherTabs = () => {
  const file = contextMenu.value.file
  if (file) {
    openFiles.value = [file]
    activeFile.value = file
  }
  hideContextMenu()
}

// 关闭全部标签
const closeAllTabs = () => {
  openFiles.value = []
  activeFile.value = null
  if (isMaximized.value) {
    isMaximized.value = false
    emit('toggleMaximize', false)
  }
  hideContextMenu()
}

// 点击其他地方关闭菜单
const handleClickOutside = () => {
  hideContextMenu()
}

defineExpose({ openFile, activeFile, openFileWithDiff, reloadCurrentFile, isImageFile })
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
      <div class="editor-tabs" @dblclick="handleTabDoubleClick">
        <div 
          v-for="(file, index) in openFiles" 
          :key="file.path"
          :class="['tab', { active: activeFile?.path === file.path }]"
          @click="activeFile = file"
          @contextmenu="showContextMenu($event, file, index)"
        >
          <span class="tab-name">{{ file.name }}</span>
          <button class="tab-close" @click.stop="closeFileTab(file)">×</button>
        </div>
        <div class="tabs-spacer"></div>
        <button class="btn-maximize" @click="handleTabDoubleClick" :title="isMaximized ? '还原' : '最大化'">
          <svg v-if="!isMaximized" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M8 3H5a2 2 0 0 0-2 2v3m18 0V5a2 2 0 0 0-2-2h-3m0 18h3a2 2 0 0 0 2-2v-3M3 16v3a2 2 0 0 0 2 2h3"/>
          </svg>
          <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M8 3v3a2 2 0 0 1-2 2H3m18 0h-3a2 2 0 0 1-2-2V3m0 18v-3a2 2 0 0 1 2-2h3M3 16h3a2 2 0 0 1 2 2v3"/>
          </svg>
        </button>
      </div>
      
      <!-- 右键菜单 -->
      <Teleport to="body">
        <div v-if="contextMenu.show" class="context-menu-overlay" @click="hideContextMenu">
          <div class="context-menu" :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }" @click.stop>
            <div class="context-menu-item" @click="closeTab">{{ t('editor.close') }}</div>
            <div 
              :class="['context-menu-item', { disabled: contextMenu.index === 0 }]" 
              @click="contextMenu.index > 0 && closeTabsToLeft()"
            >{{ t('editor.closeLeft') }}</div>
            <div 
              :class="['context-menu-item', { disabled: contextMenu.index === openFiles.length - 1 }]" 
              @click="contextMenu.index < openFiles.length - 1 && closeTabsToRight()"
            >{{ t('editor.closeRight') }}</div>
            <div class="context-menu-divider"></div>
            <div 
              :class="['context-menu-item', { disabled: openFiles.length <= 1 }]" 
              @click="openFiles.length > 1 && closeOtherTabs()"
            >{{ t('editor.closeOthers') }}</div>
            <div class="context-menu-item" @click="closeAllTabs">{{ t('editor.closeAll') }}</div>
          </div>
        </div>
      </Teleport>
      
      <div class="editor-content">
        <!-- 图片预览 -->
        <div v-if="isActiveFileImage" class="image-preview">
          <div class="image-container">
            <img :src="'file://' + activeFile.path" :alt="activeFile.name" />
          </div>
          <div class="image-info">
            <span>{{ activeFile.name }}</span>
            <span class="image-path">{{ activeFile.path }}</span>
          </div>
        </div>
        <!-- 代码编辑器 -->
        <template v-else>
          <CodeEditor 
            v-for="file in openFiles"
            :key="file.path"
            v-show="activeFile?.path === file.path && !isImageFile(file.name)"
            :ref="el => setEditorRef(file.path, el)"
            :file="file"
            :sessionId="props.currentSessionId"
            @run="handleRunCommand"
          />
        </template>
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

.tabs-spacer { flex: 1; }
.btn-maximize { background: transparent; border: none; color: var(--text-muted); cursor: pointer; padding: 4px 8px; margin-right: 4px; border-radius: 4px; display: flex; align-items: center; }
.btn-maximize:hover { background: var(--bg-hover); color: var(--text-primary); }

.editor-content { flex: 1; display: flex; flex-direction: column; overflow: hidden; position: relative; }

/* 图片预览 */
.image-preview { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; background: var(--bg-base); padding: 20px; overflow: auto; }
.image-container { max-width: 100%; max-height: calc(100% - 60px); display: flex; align-items: center; justify-content: center; background: repeating-conic-gradient(#333 0% 25%, #444 0% 50%) 50% / 20px 20px; border-radius: 8px; padding: 10px; }
.image-container img { max-width: 100%; max-height: 100%; object-fit: contain; border-radius: 4px; }
.image-info { margin-top: 16px; text-align: center; color: var(--text-secondary); font-size: 12px; }
.image-info .image-path { display: block; margin-top: 4px; color: var(--text-muted); font-size: 11px; word-break: break-all; }

.editor-placeholder { flex: 1; display: flex; align-items: center; justify-content: center; text-align: center; color: var(--text-muted); }
.placeholder-content svg { opacity: 0.3; margin-bottom: 16px; }
.placeholder-content p { font-size: 14px; margin-bottom: 16px; }
.btn-open-folder { display: flex; align-items: center; gap: 8px; padding: 8px 16px; background: var(--accent-button); border: none; border-radius: 6px; color: white; font-size: 13px; cursor: pointer; transition: all 0.15s; }
.btn-open-folder:hover { background: var(--accent-primary); }

/* 右键菜单 */
.context-menu-overlay { position: fixed; inset: 0; z-index: 1000; }
.context-menu { position: fixed; background: var(--bg-surface); border: 1px solid var(--border-default); border-radius: 6px; box-shadow: 0 4px 12px rgba(0,0,0,0.3); min-width: 160px; padding: 4px 0; z-index: 1001; }
.context-menu-item { padding: 6px 12px; font-size: 12px; color: var(--text-primary); cursor: pointer; }
.context-menu-item:hover { background: var(--bg-hover); }
.context-menu-item.disabled { color: var(--text-muted); cursor: not-allowed; }
.context-menu-item.disabled:hover { background: transparent; }
.context-menu-divider { height: 1px; background: var(--border-default); margin: 4px 0; }
</style>
