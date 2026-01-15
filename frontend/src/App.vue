<script setup>
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Environment } from '../wailsjs/runtime/runtime'
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
const chatWidth = ref(420)
const terminalHeight = ref(200)

// 拖动状态
const isDragging = ref(false)

// 组件引用
const editorAreaRef = ref(null)

// OpenCode
const {
  connected, connecting, sessions, currentSession, messages,
  sending, currentModel, models, autoConnect,
  selectSession, createSession, sendMessage, setModel, cancelMessage,
  switchWorkDir, setActiveFile
} = useOpenCode()

onMounted(async () => {
  initTheme()
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
      showSettings.value = false
      showSidebar.value = false
    } else {
      showSettings.value = true
      showSidebar.value = true
    }
  } else {
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

// 拖动处理
const startDrag = (type) => (e) => {
  isDragging.value = true
  const onMove = (e) => {
    if (type === 'sidebar') {
      const w = e.clientX - 48
      if (w >= 180 && w <= 500) sidebarWidth.value = w
    } else if (type === 'chat') {
      const w = window.innerWidth - e.clientX
      if (w >= 320 && w <= 800) chatWidth.value = w
    } else if (type === 'terminal') {
      const rect = document.querySelector('.editor-wrapper').getBoundingClientRect()
      const h = rect.bottom - e.clientY
      if (h >= 100 && h <= 500) terminalHeight.value = h
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
    <TitleBar v-if="!isMac" />
    
    <div class="main">
      <!-- 活动栏 -->
      <ActivityBar 
        :activeTab="activeTab" 
        :showSidebar="showSidebar"
        :showSettings="showSettings"
        @change="handleTabChange" 
      />
      
      <!-- 侧边栏 -->
      <div v-if="showSidebar" class="sidebar-container" :style="{ width: sidebarWidth + 'px' }">
        <SettingsPanel v-if="showSettings" @close="showSettings = false; showSidebar = false" />
        <Sidebar v-else :activeTab="activeTab" :workDir="workDir" @openFile="handleOpenFile" @update:workDir="handleWorkDirChange" />
        <div class="resize-handle" @mousedown="startDrag('sidebar')"></div>
      </div>
      
      <!-- 编辑器 + 终端 -->
      <div class="editor-wrapper">
        <EditorArea 
          ref="editorAreaRef" 
          :currentSessionId="currentSession?.id"
          @update:workDir="handleWorkDirChange"
          @activeFileChange="handleActiveFileChange"
        />
        
        <div v-if="showTerminal" class="resize-handle-h" @mousedown="startDrag('terminal')"></div>
        <div v-if="showTerminal" class="terminal-wrapper" :style="{ height: terminalHeight + 'px' }">
          <TerminalPanel :visible="showTerminal" />
        </div>
      </div>
      
      <!-- 聊天面板 -->
      <div class="resize-handle-chat" @mousedown="startDrag('chat')"></div>
      <div class="chat-container" :style="{ width: chatWidth + 'px' }">
        <ChatPanel
          :sessions="sessions"
          :currentSession="currentSession"
          :messages="messages"
          :sending="sending"
          :currentModel="currentModel"
          :models="models"
          :connected="connected"
          :connecting="connecting"
          @selectSession="handleSelectSession"
          @send="sendMessage"
          @cancel="cancelMessage"
          @update:currentModel="setModel"
        />
      </div>
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
.app.dragging { cursor: col-resize; }
.app.dragging * { pointer-events: none; }
.app.mac .main { padding-top: 28px; }

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
  min-height: 100px;
  max-height: 500px;
  display: flex;
  flex-direction: column;
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
}

.resize-handle-h:hover { background: var(--accent-primary); }

.resize-handle-chat {
  width: 1px;
  cursor: col-resize;
  background: var(--border-default);
}

::-webkit-scrollbar { width: 6px; height: 6px; }
::-webkit-scrollbar-track { background: transparent; }
::-webkit-scrollbar-thumb { background: rgba(255,255,255,0.1); border-radius: 3px; }
::-webkit-scrollbar-thumb:hover { background: rgba(255,255,255,0.2); }
</style>
