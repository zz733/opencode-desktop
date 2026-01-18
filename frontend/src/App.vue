<script setup>
import { ref, onMounted, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Environment, EventsOn } from '../wailsjs/runtime/runtime'
import TitleBar from './components/TitleBar.vue'
import ActivityBar from './components/ActivityBar.vue'
import Sidebar from './components/Sidebar.vue'
import EditorArea from './components/EditorArea.vue'
import ChatPanel from './components/ChatPanel.vue'
import TerminalPanel from './components/TerminalPanel.vue'
import SettingsPanel from './components/SettingsPanel.vue'
import StatusBar from './components/StatusBar.vue'
import { useOpenCode } from './composables/useOpenCode'
import { useTheme } from './composables/useTheme'

const { t } = useI18n()
const { initTheme } = useTheme()

// 平台检测
const platform = ref('windows')
const isMac = computed(() => platform.value === 'darwin')

// 布局状态
const activeTab = ref('files')
const showSidebar = ref(true)
const showSettings = ref(false)
const showTerminal = ref(true)
const showKiroSettings = ref(false) // 新增：标记是否显示 Kiro 账号设置

// 从 localStorage 读取上次的工作目录
const workDir = ref(localStorage.getItem('lastWorkDir') || '')

// 监听工作目录变化
const handleWorkDirChange = (dir) => {
  workDir.value = dir
  // 保存到 localStorage
  localStorage.setItem('lastWorkDir', dir)
  switchWorkDir(dir)
}

// 面板尺寸
const sidebarWidth = ref(260)
const savedSidebarWidth = ref(260) // 保存设置打开前的宽度
const chatWidth = ref(420)
const terminalHeight = ref(200)

// 设置面板宽度
const settingsWidth = 550

// 编辑器最大化状态
const editorMaximized = ref(false)

// 拖动状态
const isDragging = ref(false)

// 组件引用
const editorAreaRef = ref(null)

// OpenCode
const {
  connected, connecting, sessions, currentSession, messages,
  sending, currentModel, models, dynamicModels, getAllModels, fetchModels, autoConnect,
  selectSession, createSession, sendMessage, setModel, cancelMessage,
  switchWorkDir, setActiveFile
} = useOpenCode()

// 动态模型列表（响应动态模型更新）
// 直接依赖 dynamicModels.value 以确保响应式更新
const allModels = computed(() => {
  const customModels = JSON.parse(localStorage.getItem('customModels') || '[]')
  return [...dynamicModels.value, ...models, ...customModels]
})

onMounted(async () => {
  initTheme()
  
  // 如果有保存的工作目录，先通过后端设置它（不重启，只设置目录）
  if (workDir.value) {
    try {
      const { SetWorkDir } = await import('../wailsjs/go/main/App')
      await SetWorkDir(workDir.value)
      // 同时设置 OpenCode 的工作目录
      const { SetOpenCodeWorkDir } = await import('../wailsjs/go/main/App')
      await SetOpenCodeWorkDir(workDir.value)
    } catch (e) {
      console.log('设置工作目录失败:', e)
    }
  }
  
  autoConnect()
  try {
    const env = await Environment()
    platform.value = env.platform
  } catch (e) {}
})

// 会话操作
const handleSelectSession = async (session) => {
  session ? selectSession(session) : await createSession()
}

// 活动栏切换
const handleTabChange = (tab) => {
  if (tab === 'settings') {
    if (showSettings.value) {
      // 关闭设置，恢复原来的宽度
      showSettings.value = false
      showSidebar.value = false
      sidebarWidth.value = savedSidebarWidth.value
    } else {
      // 打开设置，保存当前宽度并加宽
      savedSidebarWidth.value = sidebarWidth.value
      sidebarWidth.value = settingsWidth
      showSettings.value = true
      showSidebar.value = true
    }
  } else {
    // 切换到其他 tab 时，如果之前是设置，恢复宽度
    if (showSettings.value) {
      sidebarWidth.value = savedSidebarWidth.value
    }
    if (activeTab.value === tab && showSidebar.value && !showSettings.value) {
      showSidebar.value = false
    } else {
      activeTab.value = tab
      showSettings.value = false
      showSidebar.value = true
    }
  }
}

// 打开文件
const handleOpenFile = (file) => {
  editorAreaRef.value?.openFile(file)
}

// 活动文件变化
const handleActiveFileChange = (path) => {
  setActiveFile(path)
}

// 比较文件差异
const handleCompare = (edit) => {
  // 打开文件并显示 diff
  editorAreaRef.value?.openFileWithDiff(edit)
}

// 撤销编辑后刷新编辑器
const handleRevertEdit = (editId) => {
  editorAreaRef.value?.reloadCurrentFile()
}

// 编辑器最大化切换
const handleEditorMaximize = (maximized) => {
  editorMaximized.value = maximized
}

// 终端引用
const terminalRef = ref(null)

// 运行命令
const handleRunCommand = async (command) => {
  // 确保终端可见
  showTerminal.value = true
  // 如果编辑器最大化，退出最大化
  if (editorMaximized.value) {
    editorMaximized.value = false
  }
  // 发送命令到终端
  // 通过事件通知终端执行命令
  setTimeout(() => {
    if (terminalRef.value) {
      terminalRef.value.executeCommand(command)
    }
  }, 100)
}

// 拖动处理
const startDrag = (type, e) => {
  e.preventDefault()
  isDragging.value = true
  
  const startX = e.clientX
  const startY = e.clientY
  
  const startSidebarWidth = sidebarWidth.value
  const startChatWidth = chatWidth.value
  const startTerminalHeight = terminalHeight.value
  
  const onMove = (e) => {
    e.preventDefault()
    if (type === 'sidebar') {
      const deltaX = e.clientX - startX
      const w = startSidebarWidth + deltaX
      if (w >= 150 && w <= 800) sidebarWidth.value = w
    } else if (type === 'chat') {
      const deltaX = startX - e.clientX // 右侧拖动方向相反
      const w = startChatWidth + deltaX
      if (w >= 200 && w <= 1200) chatWidth.value = w
    } else if (type === 'terminal') {
      const deltaY = startY - e.clientY // 底部拖动方向相反（向上拖动高度增加）
      const h = startTerminalHeight + deltaY
      if (h >= 50 && h <= 800) terminalHeight.value = h
    }
  }
  const onUp = () => {
    isDragging.value = false
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup', onUp)
  }
  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup', onUp)
}
</script>

<template>
  <div class="app" :class="{ dragging: isDragging, mac: isMac }">
    <!-- Mac 标题栏区域（用于拖动窗口） -->
    <div v-if="isMac" class="mac-titlebar" style="--wails-draggable:drag"></div>
    
    <TitleBar v-if="!isMac" />
    
    <div class="main">
      <!-- 活动栏 -->
      <ActivityBar 
        v-show="!editorMaximized"
        :activeTab="activeTab" 
        :showSidebar="showSidebar"
        :showSettings="showSettings"
        @change="handleTabChange" 
      />
      
      <!-- 侧边栏 -->
      <div v-if="showSidebar && !editorMaximized" class="sidebar-container" :style="{ width: sidebarWidth + 'px' }">
        <SettingsPanel 
          v-if="showSettings" 
          @close="showSettings = false; showSidebar = false; showKiroSettings = false" 
          @open-file="handleOpenFile" 
          @runCommand="handleRunCommand"
          @kiro-settings-active="showKiroSettings = $event"
        />
        <Sidebar v-else :activeTab="activeTab" :workDir="workDir" @openFile="handleOpenFile" @update:workDir="handleWorkDirChange" @runCommand="handleRunCommand" />
        <div class="resize-handle" @mousedown="startDrag('sidebar', $event)"></div>
      </div>
      
      <!-- 编辑器 + 终端 -->
      <div class="editor-wrapper">
        <EditorArea 
          ref="editorAreaRef" 
          :currentSessionId="currentSession?.id"
          @update:workDir="handleWorkDirChange"
          @activeFileChange="handleActiveFileChange"
          @toggleMaximize="handleEditorMaximize"
          @runCommand="handleRunCommand"
        />
        
        <template v-if="!editorMaximized">
          <div v-if="showTerminal" class="resize-handle-h" @mousedown="startDrag('terminal', $event)"></div>
          <div v-if="showTerminal" class="terminal-wrapper" :style="{ height: terminalHeight + 'px' }">
            <TerminalPanel ref="terminalRef" :visible="showTerminal" />
          </div>
        </template>
      </div>
      
      <!-- 聊天面板 -->
      <template v-if="!editorMaximized">
        <div class="resize-handle-chat" @mousedown="startDrag('chat', $event)"></div>
        <div class="chat-container" :style="{ width: chatWidth + 'px' }">
          <ChatPanel
            :sessions="sessions"
            :currentSession="currentSession"
            :messages="messages"
            :sending="sending"
            :currentModel="currentModel"
            :models="allModels"
            :connected="connected"
            :connecting="connecting"
            @selectSession="handleSelectSession"
            @send="sendMessage"
            @cancel="cancelMessage"
            @update:currentModel="setModel"
            @compare="handleCompare"
            @revertEdit="handleRevertEdit"
          />
        </div>
      </template>
    </div>
    
    <StatusBar 
      :connected="connected"
      :connecting="connecting"
      :currentModel="currentModel"
      :sessionTitle="currentSession?.title"
    />
  </div>
</template>

<style>
:root {
  --bg-base: #19161d;
  --bg-surface: #211d25;
  --bg-elevated: #28242e;
  --bg-hover: #322e3a;
  --bg-active: #3c3846;
  --bg-input: #28242e;
  --text-primary: #ffffff;
  --text-secondary: #938f9b;
  --text-muted: #6b6773;
  --border-default: #28242e;
  --border-subtle: #322e3a;
  --accent-primary: #b080ff;
  --accent-hover: #c4a0ff;
  --accent-button: #7138cc;
  --green: #80ffb5;
  --blue: #8dc8fb;
  --yellow: #ffcf99;
  --red: #ff8080;
  --pink: #ff80b5;
  --cyan: #80f4ff;
  --purple: #e2d3fe;
}

* { margin: 0; padding: 0; box-sizing: border-box; }

body {
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'SF Pro Text', 'Segoe UI', Roboto, sans-serif;
  background: var(--bg-base);
  color: var(--text-primary);
  font-size: 13px;
  -webkit-font-smoothing: antialiased;
  overflow: hidden;
  user-select: none;
}

.app { height: 100vh; display: flex; flex-direction: column; }
.app.dragging { user-select: none; }
.app.dragging .sidebar-container > *:not(.resize-handle),
.app.dragging .editor-wrapper > *:not(.resize-handle-h),
.app.dragging .chat-container { pointer-events: none; }

/* Mac 标题栏 */
.mac-titlebar {
  height: 38px;
  background: var(--bg-surface);
  flex-shrink: 0;
  border-bottom: 1px solid var(--border-default);
}

.main { flex: 1; display: flex; overflow: hidden; }

.sidebar-container {
  position: relative;
  display: flex;
  min-width: 180px;
  max-width: 500px;
  background: var(--bg-surface);
  border-right: 1px solid var(--border-default);
}

.editor-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 200px;
  background: var(--bg-base);
}

.terminal-wrapper {
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  min-height: 50px;
}

.chat-container {
  min-width: 280px;
  max-width: 500px;
  display: flex;
  background: var(--bg-base);
}

.resize-handle {
  position: absolute;
  right: 0; top: 0; bottom: 0;
  width: 4px;
  cursor: col-resize;
  z-index: 10;
}
.resize-handle:hover { background: var(--accent-primary); }

.resize-handle-h {
  height: 4px;
  cursor: row-resize;
  background: var(--border-default);
  z-index: 10;
  position: relative;
}

.resize-handle-h:hover { background: var(--accent-primary); }

.resize-handle-chat {
  width: 4px;
  cursor: col-resize;
  background: var(--border-default);
  z-index: 10;
  position: relative;
}
.resize-handle-chat:hover { background: var(--accent-primary); }

::-webkit-scrollbar { width: 6px; height: 6px; }
::-webkit-scrollbar-track { background: transparent; }
::-webkit-scrollbar-thumb { background: rgba(255,255,255,0.1); border-radius: 3px; }
::-webkit-scrollbar-thumb:hover { background: rgba(255,255,255,0.2); }
</style>
