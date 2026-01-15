<script setup>
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ActivityBar from './components/ActivityBar.vue'
import Sidebar from './components/Sidebar.vue'
import ChatPanel from './components/ChatPanel.vue'
import TerminalPanel from './components/TerminalPanel.vue'
import SettingsPanel from './components/SettingsPanel.vue'
import { useOpenCode } from './composables/useOpenCode'

const { t } = useI18n()

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

onMounted(() => {
  autoConnect()
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
  <div class="app" :class="{ dragging: isDraggingSidebar || isDraggingChat || isDraggingTerminal }">
    <!-- 标题栏 -->
    <header class="titlebar">
      <div class="titlebar-drag"></div>
      <div class="title">{{ t('app.title') }}</div>
      <div class="status">
        <span :class="['dot', { connected }]"></span>
      </div>
    </header>
    
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
          @selectSession="handleSelectSession"
          @send="handleSend"
          @cancel="handleCancel"
          @update:currentModel="handleModelChange"
        />
      </div>
    </div>
  </div>
</template>

<style>
/* CSS Variables - Kiro 风格 */
:root {
  --bg-base: #181818;
  --bg-surface: #1f1f1f;
  --bg-elevated: #262626;
  --bg-hover: #2c2c2c;
  --bg-active: #333333;
  --bg-input: #2a2a2a;
  --text-primary: #e4e4e4;
  --text-secondary: #a0a0a0;
  --text-muted: #6b6b6b;
  --border-default: #333333;
  --border-subtle: #2a2a2a;
  --accent-primary: #4d9cf6;
  --accent-hover: #5aa8ff;
  --green: #4ade80;
  --purple: #a78bfa;
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'SF Pro Text', 'Segoe UI', Roboto, sans-serif;
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

/* Titlebar */
.titlebar {
  height: 38px;
  background: var(--bg-surface);
  border-bottom: 1px solid var(--border-default);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
}

.titlebar-drag {
  position: absolute;
  inset: 0;
  -webkit-app-region: drag;
}

.title {
  font-size: 13px;
  color: var(--text-secondary);
  z-index: 1;
  pointer-events: none;
}

.status {
  position: absolute;
  right: 16px;
  display: flex;
  align-items: center;
  z-index: 1;
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--text-muted);
  transition: background 0.3s;
}

.dot.connected {
  background: var(--green);
}

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
  padding: 8px 16px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-default);
  border-radius: 6px;
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
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
  min-width: 320px;
  max-width: 800px;
  display: flex;
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
  width: 4px;
  cursor: col-resize;
  background: var(--border-default);
  transition: background 0.15s;
}

.resize-handle-chat:hover,
.resize-handle-chat:active {
  background: var(--accent-primary);
}

/* Scrollbar */
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track {
  background: transparent;
}

::-webkit-scrollbar-thumb {
  background: var(--bg-active);
  border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
  background: #444;
}
</style>
