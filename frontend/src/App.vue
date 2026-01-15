<script setup>
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Environment } from '../wailsjs/runtime/runtime'
import TitleBar from './components/TitleBar.vue'
import ActivityBar from './components/ActivityBar.vue'
import Sidebar from './components/Sidebar.vue'
import ChatPanel from './components/ChatPanel.vue'
import TerminalPanel from './components/TerminalPanel.vue'
import SettingsPanel from './components/SettingsPanel.vue'
import StatusBar from './components/StatusBar.vue'
import { useOpenCode } from './composables/useOpenCode'
import { useTheme } from './composables/useTheme'

const { t } = useI18n()
const { initTheme } = useTheme()

// 检测平台
const platform = ref('windows')
const isMac = computed(() => platform.value === 'darwin')

const activeTab = ref('files')
const sidebarWidth = ref(260)
const chatWidth = ref(420)
const terminalHeight = ref(200)
const showTerminal = ref(true)
const showSettings = ref(false)
const showSidebar = ref(true)
const isDraggingSidebar = ref(false)
const isDraggingChat = ref(false)
const isDraggingTerminal = ref(false)

const {
  connected,
  connecting,
  sessions,
  currentSession,
  messages,
  sending,
  currentModel,
  models,
  autoConnect,
  selectSession,
  createSession,
  sendMessage,
  setModel,
  cancelMessage
} = useOpenCode()

onMounted(async () => {
  initTheme()
  autoConnect()
  // 获取平台信息
  try {
    const env = await Environment()
    platform.value = env.platform
  } catch (e) {
    console.log('获取平台信息失败:', e)
  }
})

const handleSelectSession = async (session) => {
  if (session) {
    selectSession(session)
  } else {
    await createSession()
  }
}

const handleSend = (text) => {
  sendMessage(text)
}

const handleCancel = () => {
  cancelMessage()
}

const handleModelChange = (modelId) => {
  setModel(modelId)
}

// 拖动侧边栏
const startDragSidebar = (e) => {
  isDraggingSidebar.value = true
  document.addEventListener('mousemove', onDragSidebar)
  document.addEventListener('mouseup', stopDragSidebar)
}

const onDragSidebar = (e) => {
  if (!isDraggingSidebar.value) return
  const newWidth = e.clientX - 48 // 减去活动栏宽度
  if (newWidth >= 180 && newWidth <= 500) {
    sidebarWidth.value = newWidth
  }
}

const stopDragSidebar = () => {
  isDraggingSidebar.value = false
  document.removeEventListener('mousemove', onDragSidebar)
  document.removeEventListener('mouseup', stopDragSidebar)
}

// 拖动聊天面板
const startDragChat = (e) => {
  isDraggingChat.value = true
  document.addEventListener('mousemove', onDragChat)
  document.addEventListener('mouseup', stopDragChat)
}

const onDragChat = (e) => {
  if (!isDraggingChat.value) return
  const newWidth = window.innerWidth - e.clientX
  if (newWidth >= 320 && newWidth <= 800) {
    chatWidth.value = newWidth
  }
}

const stopDragChat = () => {
  isDraggingChat.value = false
  document.removeEventListener('mousemove', onDragChat)
  document.removeEventListener('mouseup', stopDragChat)
}

// 拖动终端面板
const startDragTerminal = (e) => {
  isDraggingTerminal.value = true
  document.addEventListener('mousemove', onDragTerminal)
  document.addEventListener('mouseup', stopDragTerminal)
}

const onDragTerminal = (e) => {
  if (!isDraggingTerminal.value) return
  const containerRect = document.querySelector('.editor-area').getBoundingClientRect()
  const newHeight = containerRect.bottom - e.clientY
  if (newHeight >= 100 && newHeight <= 500) {
    terminalHeight.value = newHeight
  }
}

const stopDragTerminal = () => {
  isDraggingTerminal.value = false
  document.removeEventListener('mousemove', onDragTerminal)
  document.removeEventListener('mouseup', stopDragTerminal)
}

const toggleTerminal = () => {
  showTerminal.value = !showTerminal.value
}

const handleTabChange = (tab) => {
  if (tab === 'settings') {
    // 设置面板：点击切换显示/隐藏
    if (showSettings.value) {
      showSettings.value = false
      showSidebar.value = false
    } else {
      showSettings.value = true
      showSidebar.value = true
    }
  } else {
    // 普通标签：点击同一个隐藏，点击不同的切换
    if (activeTab.value === tab && showSidebar.value && !showSettings.value) {
      showSidebar.value = false
    } else {
      activeTab.value = tab
      showSettings.value = false
      showSidebar.value = true
    }
  }
}
</script>

<template>
  <div class="app" :class="{ dragging: isDraggingSidebar || isDraggingChat || isDraggingTerminal, mac: isMac }">
    <!-- 自定义标题栏 (仅 Windows) -->
    <TitleBar v-if="!isMac" />
    
    <!-- 主体 -->
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
        <Sidebar v-else :activeTab="activeTab" />
        <div class="resize-handle" @mousedown="startDragSidebar"></div>
      </div>
      
      <!-- 编辑器区域（中间） -->
      <div class="editor-area">
        <div class="editor-main">
          <div class="editor-placeholder">
            <div class="placeholder-content">
              <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
                <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
                <path d="M14 2v6h6"/>
                <path d="M16 13H8"/>
                <path d="M16 17H8"/>
                <path d="M10 9H8"/>
              </svg>
              <p>{{ t('editor.openFile') }}</p>
              <button class="btn-toggle-terminal" @click="toggleTerminal">
                {{ showTerminal ? t('editor.hideTerminal') : t('editor.showTerminal') }}
              </button>
            </div>
          </div>
        </div>
        
        <!-- 终端拖动条 -->
        <div v-if="showTerminal" class="resize-handle-terminal" @mousedown="startDragTerminal"></div>
        
        <!-- 终端面板 -->
        <div v-if="showTerminal" class="terminal-wrapper" :style="{ height: terminalHeight + 'px' }">
          <TerminalPanel :visible="showTerminal" />
        </div>
      </div>
      
      <!-- 聊天面板分隔条 -->
      <div class="resize-handle-chat" @mousedown="startDragChat"></div>
      
      <!-- 聊天面板 -->
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
          @send="handleSend"
          @cancel="handleCancel"
          @update:currentModel="handleModelChange"
        />
      </div>
    </div>
    
    <!-- 状态栏 -->
    <StatusBar 
      :connected="connected"
      :connecting="connecting"
      :currentModel="currentModel"
      :sessionTitle="currentSession?.title"
    />
  </div>
</template>

<style>
/* CSS Variables */
:root {
  /* Kiro Dark Theme */
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

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'SF Pro Text', 'Segoe UI', Roboto, sans-serif;
  background: var(--bg-base);
  color: var(--text-primary);
  font-size: 13px;
  -webkit-font-smoothing: antialiased;
  overflow: hidden;
  user-select: none;
}

.app {
  height: 100vh;
  display: flex;
  flex-direction: column;
}

.app.dragging {
  cursor: col-resize;
}

.app.dragging * {
  pointer-events: none;
}

/* Titlebar - removed */

/* Main */
.main {
  flex: 1;
  display: flex;
  overflow: hidden;
}

/* Sidebar Container */
.sidebar-container {
  position: relative;
  display: flex;
  min-width: 180px;
  max-width: 500px;
  background: var(--bg-surface);
  border-right: 1px solid var(--border-default);
}

/* Editor Area */
.editor-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: var(--bg-base);
  min-width: 200px;
}

.editor-main {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.editor-placeholder {
  text-align: center;
  color: var(--text-muted);
}

.placeholder-content svg {
  opacity: 0.3;
  margin-bottom: 16px;
}

.placeholder-content p {
  font-size: 14px;
  margin-bottom: 16px;
}

.btn-toggle-terminal {
  padding: 6px 14px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-default);
  border-radius: 6px;
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-toggle-terminal:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

/* Terminal */
.terminal-wrapper {
  min-height: 100px;
  max-height: 500px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.terminal-wrapper > * {
  flex: 1;
  height: 100%;
}

.resize-handle-terminal {
  height: 4px;
  cursor: row-resize;
  background: var(--border-default);
  transition: background 0.15s;
}

.resize-handle-terminal:hover,
.resize-handle-terminal:active {
  background: var(--accent-primary);
}

/* Chat Container */
.chat-container {
  width: 360px;
  min-width: 280px;
  max-width: 500px;
  display: flex;
  background: var(--bg-base);
}

/* Resize Handles */
.resize-handle {
  position: absolute;
  right: 0;
  top: 0;
  bottom: 0;
  width: 4px;
  cursor: col-resize;
  background: transparent;
  transition: background 0.15s;
  z-index: 10;
}

.resize-handle:hover,
.resize-handle:active {
  background: var(--accent-primary);
}

.resize-handle-chat {
  width: 1px;
  cursor: col-resize;
  background: var(--border-default);
}

/* Scrollbar */
::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}

::-webkit-scrollbar-track {
  background: transparent;
}

::-webkit-scrollbar-thumb {
  background: rgba(255,255,255,0.1);
  border-radius: 3px;
}

::-webkit-scrollbar-thumb:hover {
  background: rgba(255,255,255,0.2);
}

/* Mac 适配 - 为系统标题栏留出空间 */
.app.mac .main {
  padding-top: 28px;
}

.app.mac .sidebar-container,
.app.mac .chat-container {
  padding-top: 0;
}
</style>
