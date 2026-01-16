<script setup>
import { ref, watch, onMounted, onUnmounted, shallowRef, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import * as monaco from 'monaco-editor'
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker'
import jsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker'
import cssWorker from 'monaco-editor/esm/vs/language/css/css.worker?worker'
import htmlWorker from 'monaco-editor/esm/vs/language/html/html.worker?worker'
import tsWorker from 'monaco-editor/esm/vs/language/typescript/ts.worker?worker'
import { ReadFileContent, WriteFileContent, WatchFile, UnwatchFile, CodeCompletion, RunFile, WriteTerminal, CreateTerminal } from '../../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { useFileEdits } from '../composables/useFileEdits'

const { t } = useI18n()

// 配置 Monaco workers
self.MonacoEnvironment = {
  getWorker(_, label) {
    if (label === 'json') return new jsonWorker()
    if (label === 'css' || label === 'scss' || label === 'less') return new cssWorker()
    if (label === 'html' || label === 'handlebars' || label === 'razor') return new htmlWorker()
    if (label === 'typescript' || label === 'javascript') return new tsWorker()
    return new editorWorker()
  }
}

const props = defineProps({ 
  file: Object,
  sessionId: String
})
const emit = defineEmits(['close', 'save', 'run'])

const { addEdit } = useFileEdits()

const editorContainer = ref(null)
const editor = shallowRef(null)
const content = ref('')
const originalContent = ref('')
const loading = ref(false)
const saving = ref(false)
const modified = ref(false)
const running = ref(false)

// AI 补全状态
const completing = ref(false)
const ghostText = ref('')
const ghostDecoration = ref([])

// 右键菜单状态
const contextMenu = ref({
  visible: false,
  x: 0,
  y: 0
})

// 判断是否是 Mac
const isMac = navigator.platform.toUpperCase().indexOf('MAC') >= 0

// 获取快捷键显示文本
const getShortcut = (key) => {
  const mod = isMac ? '⌘' : 'Ctrl+'
  const shortcuts = {
    cut: `${mod}X`,
    copy: `${mod}C`,
    paste: `${mod}V`,
    delete: 'Del',
    selectAll: `${mod}A`,
    undo: `${mod}Z`,
    redo: isMac ? '⇧⌘Z' : 'Ctrl+Y',
    find: `${mod}F`,
    replace: isMac ? '⌥⌘F' : 'Ctrl+H',
    goToLine: `${mod}G`,
    format: isMac ? '⇧⌥F' : 'Shift+Alt+F',
    comment: `${mod}/`,
  }
  return shortcuts[key] || ''
}

// 右键菜单操作
const menuActions = {
  undo: () => editor.value?.trigger('contextmenu', 'undo', null),
  redo: () => editor.value?.trigger('contextmenu', 'redo', null),
  cut: () => {
    editor.value?.focus()
    document.execCommand('cut')
  },
  copy: () => {
    editor.value?.focus()
    document.execCommand('copy')
  },
  paste: async () => {
    editor.value?.focus()
    try {
      const text = await navigator.clipboard.readText()
      editor.value?.trigger('contextmenu', 'paste', { text })
    } catch (e) {
      document.execCommand('paste')
    }
  },
  delete: () => editor.value?.trigger('contextmenu', 'deleteRight', null),
  selectAll: () => editor.value?.trigger('contextmenu', 'selectAll', null),
  find: () => editor.value?.trigger('contextmenu', 'actions.find', null),
  replace: () => editor.value?.trigger('contextmenu', 'editor.action.startFindReplaceAction', null),
  goToLine: () => editor.value?.trigger('contextmenu', 'editor.action.gotoLine', null),
  format: () => editor.value?.trigger('contextmenu', 'editor.action.formatDocument', null),
  comment: () => editor.value?.trigger('contextmenu', 'editor.action.commentLine', null),
}

// 显示右键菜单
const showContextMenu = (e) => {
  e.preventDefault()
  contextMenu.value = {
    visible: true,
    x: e.clientX,
    y: e.clientY
  }
}

// 隐藏右键菜单
const hideContextMenu = () => {
  contextMenu.value.visible = false
}

// 执行菜单操作
const executeMenuAction = (action) => {
  hideContextMenu()
  if (menuActions[action]) {
    menuActions[action]()
  }
}

const getLanguage = (filename) => {
  const ext = filename?.split('.').pop()?.toLowerCase()
  const langMap = {
    'go': 'go', 'js': 'javascript', 'jsx': 'javascript', 'ts': 'typescript', 'tsx': 'typescript',
    'vue': 'vue', 'html': 'html', 'css': 'css', 'scss': 'scss', 'less': 'less', 'json': 'json',
    'md': 'markdown', 'py': 'python', 'rs': 'rust', 'sh': 'shell', 'yaml': 'yaml', 'yml': 'yaml',
    'xml': 'xml', 'sql': 'sql', 'java': 'java', 'c': 'c', 'cpp': 'cpp', 'swift': 'swift',
    'kt': 'kotlin', 'rb': 'ruby', 'php': 'php', 'lua': 'lua', 'toml': 'toml', 'ini': 'ini',
  }
  return langMap[ext] || 'plaintext'
}

const loadFile = async () => {
  if (!props.file?.path) return
  loading.value = true
  try {
    const text = await ReadFileContent(props.file.path)
    content.value = text
    originalContent.value = text
    modified.value = false
    if (editor.value) {
      monaco.editor.setModelLanguage(editor.value.getModel(), getLanguage(props.file.name))
      const position = editor.value.getPosition()
      editor.value.setValue(text)
      if (position) editor.value.setPosition(position)
    }
    await WatchFile(props.file.path).catch(() => {})
  } catch (e) {
    content.value = `// 读取文件失败: ${e}`
    if (editor.value) editor.value.setValue(content.value)
  } finally {
    loading.value = false
  }
}

// 文件变化时记录编辑并刷新
const handleFileChanged = async (changedPath) => {
  if (changedPath !== props.file?.path) return
  
  const oldContent = editor.value?.getValue() || content.value
  
  try {
    const newContent = await ReadFileContent(props.file.path)
    
    if (oldContent !== newContent) {
      addEdit(props.file.path, oldContent, newContent)
      content.value = newContent
      originalContent.value = newContent
      modified.value = false
      
      if (editor.value) {
        const position = editor.value.getPosition()
        editor.value.setValue(newContent)
        if (position) editor.value.setPosition(position)
      }
    }
  } catch (e) {
    console.error('读取文件失败:', e)
  }
}

const saveFile = async () => {
  if (!props.file?.path || saving.value) return
  saving.value = true
  try {
    const currentContent = editor.value?.getValue() || content.value
    await WriteFileContent(props.file.path, currentContent)
    originalContent.value = currentContent
    modified.value = false
    emit('save', props.file)
  } catch (e) {
    alert('保存失败: ' + e)
  } finally {
    saving.value = false
  }
}

// 可运行的文件类型
const runnableExtensions = ['py', 'go', 'js', 'ts', 'java', 'rs', 'rb', 'php', 'sh', 'html', 'htm']

const canRun = computed(() => {
  if (!props.file?.name) return false
  const ext = props.file.name.split('.').pop()?.toLowerCase()
  return runnableExtensions.includes(ext)
})

// 运行文件
const runFile = async () => {
  if (!props.file?.path || running.value) return
  
  // 先保存
  if (modified.value) {
    await saveFile()
  }
  
  running.value = true
  try {
    const result = await RunFile(props.file.path)
    
    if (result.startsWith('OPEN_BROWSER:')) {
      // HTML 文件，打开浏览器
      const filePath = result.replace('OPEN_BROWSER:', '')
      // 使用简单的 HTTP 服务器或直接打开文件
      window.open('file://' + filePath)
    } else {
      // 在终端执行命令
      emit('run', result)
    }
  } catch (e) {
    alert('运行失败: ' + e)
  } finally {
    running.value = false
  }
}

// AI 代码补全
let completionTimeout = null
let currentGhostPosition = null
// 暂时禁用 AI 补全，因为 OpenCode 的同步 API 不可用
let completionEnabled = ref(false)

const requestCompletion = async () => {
  // AI 补全暂时禁用
  if (!completionEnabled.value) {
    return
  }
  
  console.log('[Completion] requestCompletion called')
  console.log('[Completion] sessionId:', props.sessionId)
  console.log('[Completion] editor:', !!editor.value)
  console.log('[Completion] completing:', completing.value)
  console.log('[Completion] enabled:', completionEnabled.value)
  
  if (!props.sessionId || !editor.value || completing.value) {
    console.log('[Completion] Skipped - conditions not met')
    return
  }
  
  const position = editor.value.getPosition()
  const model = editor.value.getModel()
  
  if (!position || !model) {
    console.log('[Completion] No position or model')
    return
  }
  
  // 获取光标前的代码（最多 30 行，减少请求大小）
  const startLine = Math.max(1, position.lineNumber - 30)
  const textBeforeCursor = model.getValueInRange({
    startLineNumber: startLine,
    startColumn: 1,
    endLineNumber: position.lineNumber,
    endColumn: position.column
  })
  
  // 如果光标前没有内容或只有空白，不补全
  if (!textBeforeCursor.trim()) {
    console.log('[Completion] No text before cursor')
    return
  }
  
  completing.value = true
  console.log('[Completion] Requesting for:', textBeforeCursor.slice(-200))
  
  try {
    const language = getLanguage(props.file?.name)
    console.log('[Completion] Language:', language, 'File:', props.file?.name)
    
    const completion = await CodeCompletion(props.sessionId, textBeforeCursor, language, props.file?.name || '')
    
    console.log('[Completion] Result:', completion)
    
    // 检查光标位置是否还在原来的位置
    const currentPos = editor.value.getPosition()
    if (currentPos.lineNumber !== position.lineNumber || currentPos.column !== position.column) {
      console.log('[Completion] Cursor moved, discarding result')
      return
    }
    
    if (completion && completion.trim()) {
      // 显示幽灵文本
      showGhostText(completion.trim(), position)
    } else {
      console.log('[Completion] Empty result')
    }
  } catch (e) {
    console.error('[Completion] Error:', e)
  } finally {
    completing.value = false
  }
}

// 显示幽灵文本（灰色预览）
const showGhostText = (text, position) => {
  if (!editor.value) {
    console.log('[GhostText] No editor')
    return
  }
  
  console.log('[GhostText] Showing:', text.substring(0, 50))
  
  ghostText.value = text
  currentGhostPosition = position
  
  // 只取第一行作为内联显示
  const firstLine = text.split('\n')[0]
  
  // 使用 inline decoration 显示幽灵文本
  const decorations = [{
    range: new monaco.Range(position.lineNumber, position.column, position.lineNumber, position.column),
    options: {
      after: {
        content: firstLine,
        inlineClassName: 'ghost-text-inline',
        cursorStops: monaco.editor.InjectedTextCursorStops.None
      }
    }
  }]
  
  console.log('[GhostText] Creating decoration at line', position.lineNumber, 'col', position.column)
  ghostDecoration.value = editor.value.deltaDecorations(ghostDecoration.value, decorations)
  console.log('[GhostText] Decoration IDs:', ghostDecoration.value)
}

// 清除幽灵文本
const clearGhostText = () => {
  if (editor.value && ghostDecoration.value.length) {
    ghostDecoration.value = editor.value.deltaDecorations(ghostDecoration.value, [])
  }
  ghostText.value = ''
  currentGhostPosition = null
}

// 接受补全
const acceptCompletion = () => {
  if (!ghostText.value || !currentGhostPosition || !editor.value) return false
  
  const text = ghostText.value
  clearGhostText()
  
  // 插入补全文本
  editor.value.executeEdits('ai-completion', [{
    range: new monaco.Range(
      currentGhostPosition.lineNumber,
      currentGhostPosition.column,
      currentGhostPosition.lineNumber,
      currentGhostPosition.column
    ),
    text: text
  }])
  
  // 移动光标到插入文本末尾
  const lines = text.split('\n')
  const newLine = currentGhostPosition.lineNumber + lines.length - 1
  const newColumn = lines.length === 1 
    ? currentGhostPosition.column + text.length 
    : lines[lines.length - 1].length + 1
  editor.value.setPosition({ lineNumber: newLine, column: newColumn })
  
  return true
}

const initEditor = () => {
  if (!editorContainer.value) return
  editor.value = monaco.editor.create(editorContainer.value, {
    value: content.value,
    language: getLanguage(props.file?.name),
    theme: 'vs-dark',
    fontSize: 13,
    fontFamily: "'Menlo', 'Monaco', 'Courier New', monospace",
    lineNumbers: 'on',
    minimap: { enabled: false },
    scrollBeyondLastLine: false,
    automaticLayout: true,
    tabSize: 2,
    wordWrap: 'off',
    renderWhitespace: 'selection',
    cursorBlinking: 'smooth',
    smoothScrolling: true,
    padding: { top: 8 },
    // 自动补全配置
    quickSuggestions: true,
    suggestOnTriggerCharacters: true,
    acceptSuggestionOnEnter: 'on',
    tabCompletion: 'on',
    wordBasedSuggestions: 'currentDocument',
    snippetSuggestions: 'inline',
    // 禁用默认右键菜单
    contextmenu: false,
  })
  
  // 使用 Monaco 的鼠标事件检测右键
  editor.value.onMouseDown((e) => {
    if (e.event.rightButton) {
      e.event.preventDefault()
      e.event.stopPropagation()
      contextMenu.value = {
        visible: true,
        x: e.event.posx,
        y: e.event.posy
      }
    }
  })
  
  // 阻止编辑器区域的默认右键菜单
  editor.value.getDomNode()?.addEventListener('contextmenu', (e) => {
    e.preventDefault()
    e.stopPropagation()
    contextMenu.value = {
      visible: true,
      x: e.clientX,
      y: e.clientY
    }
  }, true)
  
  // 自动保存：内容变化后延迟保存
  let saveTimeout = null
  editor.value.onDidChangeModelContent(() => {
    modified.value = editor.value.getValue() !== originalContent.value
    
    // 清除幽灵文本
    clearGhostText()
    
    // 清除之前的定时器
    if (saveTimeout) clearTimeout(saveTimeout)
    if (completionTimeout) clearTimeout(completionTimeout)
    
    // 如果有修改，延迟 1 秒后自动保存
    if (modified.value) {
      saveTimeout = setTimeout(() => {
        saveFile()
      }, 1000)
    }
    
    // 延迟 500ms 后请求 AI 补全
    if (props.sessionId) {
      completionTimeout = setTimeout(() => {
        requestCompletion()
      }, 800)
    }
  })
  
  // Tab 键接受补全
  editor.value.addCommand(monaco.KeyCode.Tab, () => {
    if (ghostText.value) {
      acceptCompletion()
    } else {
      // 默认 Tab 行为
      editor.value.trigger('keyboard', 'tab', {})
    }
  })
  
  // Escape 键取消补全
  editor.value.addCommand(monaco.KeyCode.Escape, () => {
    if (ghostText.value) {
      clearGhostText()
    }
  })
  
  // 光标移动时清除幽灵文本
  editor.value.onDidChangeCursorPosition(() => {
    if (ghostText.value) {
      clearGhostText()
    }
  })
  
  // 手动保存快捷键
  editor.value.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, saveFile)
}

// 重新加载文件（供外部调用，如撤销后）
const reloadFile = async () => {
  await loadFile()
}

watch(() => props.file?.path, (newPath, oldPath) => {
  if (oldPath) UnwatchFile(oldPath)
  loadFile()
})

onMounted(() => {
  initEditor()
  if (props.file) loadFile()
  EventsOn('file-changed', handleFileChanged)
  
  // 点击外部关闭右键菜单
  document.addEventListener('click', hideContextMenu)
  document.addEventListener('contextmenu', hideContextMenu)
  
  // 添加幽灵文本样式（全局样式，因为 Monaco 在 shadow DOM 外）
  if (!document.getElementById('ghost-text-style')) {
    const style = document.createElement('style')
    style.id = 'ghost-text-style'
    style.textContent = `
      .ghost-text-inline {
        color: #6b6b6b !important;
        font-style: italic !important;
        opacity: 0.7 !important;
      }
    `
    document.head.appendChild(style)
  }
})

onUnmounted(() => {
  editor.value?.dispose()
  if (props.file?.path) UnwatchFile(props.file.path)
  EventsOff('file-changed')
  if (completionTimeout) clearTimeout(completionTimeout)
  document.removeEventListener('click', hideContextMenu)
  document.removeEventListener('contextmenu', hideContextMenu)
})

const lineCount = () => editor.value?.getModel()?.getLineCount() || content.value.split('\n').length

defineExpose({ reloadFile })
</script>

<template>
  <div class="code-editor">
    <div class="editor-body">
      <div v-if="loading" class="loading">加载中...</div>
      <div ref="editorContainer" class="monaco-container"></div>
    </div>
    <div class="editor-status">
      <span>{{ getLanguage(file?.name) }}</span>
      <span>{{ lineCount() }} 行</span>
      <span v-if="modified" class="status-modified">已修改</span>
      <span v-if="saving" class="status-saving">保存中...</span>
      <span v-if="running" class="status-running">运行中...</span>
      <span class="shortcut">自动保存</span>
      <button v-if="canRun" class="btn-run" @click="runFile" :disabled="running" title="运行 (F5)">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
          <path d="M8 5v14l11-7z"/>
        </svg>
        运行
      </button>
    </div>
    
    <!-- 右键菜单 -->
    <Teleport to="body">
      <div 
        v-if="contextMenu.visible" 
        class="context-menu"
        :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
        @click.stop
      >
        <div class="menu-item" @click="executeMenuAction('undo')">
          <span class="menu-label">{{ t('editor.undo') }}</span>
          <span class="menu-shortcut">{{ getShortcut('undo') }}</span>
        </div>
        <div class="menu-item" @click="executeMenuAction('redo')">
          <span class="menu-label">{{ t('editor.redo') }}</span>
          <span class="menu-shortcut">{{ getShortcut('redo') }}</span>
        </div>
        <div class="menu-divider"></div>
        <div class="menu-item" @click="executeMenuAction('cut')">
          <span class="menu-label">{{ t('editor.cut') }}</span>
          <span class="menu-shortcut">{{ getShortcut('cut') }}</span>
        </div>
        <div class="menu-item" @click="executeMenuAction('copy')">
          <span class="menu-label">{{ t('editor.copy') }}</span>
          <span class="menu-shortcut">{{ getShortcut('copy') }}</span>
        </div>
        <div class="menu-item" @click="executeMenuAction('paste')">
          <span class="menu-label">{{ t('editor.paste') }}</span>
          <span class="menu-shortcut">{{ getShortcut('paste') }}</span>
        </div>
        <div class="menu-item" @click="executeMenuAction('delete')">
          <span class="menu-label">{{ t('editor.delete') }}</span>
          <span class="menu-shortcut">{{ getShortcut('delete') }}</span>
        </div>
        <div class="menu-divider"></div>
        <div class="menu-item" @click="executeMenuAction('selectAll')">
          <span class="menu-label">{{ t('editor.selectAll') }}</span>
          <span class="menu-shortcut">{{ getShortcut('selectAll') }}</span>
        </div>
        <div class="menu-divider"></div>
        <div class="menu-item" @click="executeMenuAction('find')">
          <span class="menu-label">{{ t('editor.find') }}</span>
          <span class="menu-shortcut">{{ getShortcut('find') }}</span>
        </div>
        <div class="menu-item" @click="executeMenuAction('replace')">
          <span class="menu-label">{{ t('editor.replace') }}</span>
          <span class="menu-shortcut">{{ getShortcut('replace') }}</span>
        </div>
        <div class="menu-item" @click="executeMenuAction('goToLine')">
          <span class="menu-label">{{ t('editor.goToLine') }}</span>
          <span class="menu-shortcut">{{ getShortcut('goToLine') }}</span>
        </div>
        <div class="menu-divider"></div>
        <div class="menu-item" @click="executeMenuAction('comment')">
          <span class="menu-label">{{ t('editor.comment') }}</span>
          <span class="menu-shortcut">{{ getShortcut('comment') }}</span>
        </div>
        <div class="menu-item" @click="executeMenuAction('format')">
          <span class="menu-label">{{ t('editor.format') }}</span>
          <span class="menu-shortcut">{{ getShortcut('format') }}</span>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.code-editor { display: flex; flex-direction: column; width: 100%; height: 100%; flex: 1; background: var(--bg-base); }
.editor-body { flex: 1; position: relative; overflow: hidden; min-height: 0; }
.loading { position: absolute; inset: 0; display: flex; align-items: center; justify-content: center; color: var(--text-muted); z-index: 10; background: var(--bg-base); }
.monaco-container { position: absolute; top: 0; left: 0; right: 0; bottom: 0; }
.editor-status { display: flex; align-items: center; gap: 16px; padding: 4px 12px; background: var(--bg-surface); border-top: 1px solid var(--border-default); font-size: 11px; color: var(--text-muted); }
.status-modified { color: var(--accent-primary); }
.status-saving { color: var(--green); }
.status-completing { color: var(--blue); }
.status-running { color: var(--yellow); }
.shortcut { margin-left: auto; }
.btn-run {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  margin-left: 8px;
  background: var(--green);
  border: none;
  border-radius: 4px;
  color: var(--bg-base);
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}
.btn-run:hover { background: #60d090; }
.btn-run:disabled { opacity: 0.5; cursor: not-allowed; }
</style>

<style>
/* 右键菜单样式（全局，因为使用 Teleport） */
.context-menu {
  position: fixed;
  min-width: 200px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-subtle);
  border-radius: 6px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.4);
  padding: 4px 0;
  z-index: 10000;
  font-size: 13px;
}

.menu-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 12px;
  cursor: pointer;
  color: var(--text-primary);
}

.menu-item:hover {
  background: var(--bg-hover);
}

.menu-label {
  flex: 1;
}

.menu-shortcut {
  margin-left: 24px;
  color: var(--text-muted);
  font-size: 12px;
}

.menu-divider {
  height: 1px;
  background: var(--border-default);
  margin: 4px 0;
}
</style>
