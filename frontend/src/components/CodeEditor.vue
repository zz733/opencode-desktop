<script setup>
import { ref, watch, onMounted, onUnmounted, shallowRef } from 'vue'
import * as monaco from 'monaco-editor'
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker'
import jsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker'
import cssWorker from 'monaco-editor/esm/vs/language/css/css.worker?worker'
import htmlWorker from 'monaco-editor/esm/vs/language/html/html.worker?worker'
import tsWorker from 'monaco-editor/esm/vs/language/typescript/ts.worker?worker'
import { ReadFileContent, WriteFileContent, WatchFile, UnwatchFile } from '../../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { useFileEdits } from '../composables/useFileEdits'

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

const props = defineProps({ file: Object })
const emit = defineEmits(['close', 'save'])

const { addEdit } = useFileEdits()

const editorContainer = ref(null)
const editor = shallowRef(null)
const content = ref('')
const originalContent = ref('')
const loading = ref(false)
const saving = ref(false)
const modified = ref(false)

const getLanguage = (filename) => {
  const ext = filename?.split('.').pop()?.toLowerCase()
  const langMap = {
    'go': 'go', 'js': 'javascript', 'jsx': 'javascript', 'ts': 'typescript', 'tsx': 'typescript',
    'vue': 'html', 'html': 'html', 'css': 'css', 'scss': 'scss', 'less': 'less', 'json': 'json',
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
    
    // 如果内容不同，记录编辑
    if (oldContent !== newContent) {
      addEdit(props.file.path, oldContent, newContent)
      
      // 更新编辑器
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
  })
  editor.value.onDidChangeModelContent(() => {
    modified.value = editor.value.getValue() !== originalContent.value
  })
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
})

onUnmounted(() => {
  editor.value?.dispose()
  if (props.file?.path) UnwatchFile(props.file.path)
  EventsOff('file-changed')
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
      <span class="shortcut">⌘S 保存</span>
    </div>
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
.shortcut { margin-left: auto; }
</style>
