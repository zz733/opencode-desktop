<script setup>
import { ref, nextTick, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import MessageItem from './MessageItem.vue'
import ModelSelector from './ModelSelector.vue'

const { t } = useI18n()

const props = defineProps({
  sessions: Array,
  currentSession: Object,
  messages: Array,
  sending: Boolean,
  currentModel: String,
  models: Array,
  connected: Boolean,
  connecting: Boolean
})

const emit = defineEmits(['selectSession', 'send', 'update:currentModel', 'cancel'])

const inputText = ref('')
const messagesContainer = ref(null)
const showSessionList = ref(false)

const handleKeydown = (e) => {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    send()
  }
}

const send = () => {
  console.log('send called, inputText:', inputText.value, 'sending:', props.sending)
  if (!inputText.value.trim() || props.sending) return
  console.log('emitting send event')
  emit('send', inputText.value)
  inputText.value = ''
}

const cancel = () => {
  emit('cancel')
}

const selectSession = (s) => {
  emit('selectSession', s)
  showSessionList.value = false
}

watch(() => props.messages.length, () => {
  nextTick(() => {
    if (messagesContainer.value) {
      messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
    }
  })
})

// 监听消息内容变化，自动滚动
watch(() => props.messages[props.messages.length - 1]?.content, () => {
  nextTick(() => {
    if (messagesContainer.value) {
      messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
    }
  })
})
</script>

<template>
  <div class="chat-panel">
    <!-- 头部 -->
    <header class="chat-header">
      <div class="session-selector" @click="showSessionList = !showSessionList">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
        </svg>
        <span>{{ currentSession?.title || t('chat.newSession') }}</span>
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M6 9l6 6 6-6"/>
        </svg>
        
        <div v-if="showSessionList" class="session-dropdown" @click.stop>
          <div class="dropdown-header">
            <span>{{ t('chat.sessions') }}</span>
            <button class="btn-new" @click="emit('selectSession', null)">+ {{ t('chat.new') }}</button>
          </div>
          <div class="session-list">
            <div 
              v-for="s in sessions" 
              :key="s.id"
              :class="['session-item', { active: currentSession?.id === s.id }]"
              @click="selectSession(s)"
            >
              {{ s.title || t('chat.newSession') }}
            </div>
            <div v-if="!sessions.length" class="empty">{{ t('chat.noSessions') }}</div>
          </div>
        </div>
      </div>
      
      <!-- 连接状态指示器 -->
      <div class="status-indicator" :title="connected ? '已连接' : (connecting ? '连接中...' : '未连接')">
        <span :class="['status-dot', { connected, connecting }]"></span>
      </div>
    </header>
    
    <!-- 消息 -->
    <div class="messages" ref="messagesContainer">
      <div v-if="!messages.length" class="empty-state">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
          <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
        </svg>
        <h3>{{ t('chat.howCanIHelp') }}</h3>
        <p>{{ t('chat.askAnything') }}</p>
        <p v-if="!connected" class="status-hint">{{ connecting ? '正在连接...' : '未连接到 OpenCode 服务' }}</p>
      </div>
      
      <MessageItem 
        v-for="(msg, i) in messages" 
        :key="i" 
        :message="msg"
        :isLoading="sending && i === messages.length - 1 && msg.role === 'assistant'"
      />
    </div>
    
    <!-- 输入区域 -->
    <div class="input-area">
      <div class="input-box">
        <textarea 
          v-model="inputText"
          :placeholder="t('chat.placeholder')"
          @keydown="handleKeydown"
          :disabled="sending"
          rows="1"
        ></textarea>
        
        <!-- 底部工具栏：在输入框内 -->
        <div class="input-toolbar">
          <div class="toolbar-left">
            <button class="toolbar-btn" title="Add context (#)">
              <span style="font-size: 16px; font-weight: 500;">#</span>
            </button>
            <button class="toolbar-btn" title="Attach image">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
                <circle cx="8.5" cy="8.5" r="1.5"/>
                <polyline points="21 15 16 10 5 21"/>
              </svg>
            </button>
            <button class="toolbar-btn" title="Voice input">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/>
              </svg>
            </button>
          </div>
          <div class="toolbar-right">
            <ModelSelector 
              :modelValue="currentModel"
              :models="models"
              @update:modelValue="emit('update:currentModel', $event)"
            />
            <button 
              v-if="sending" 
              @click="cancel" 
              class="btn-cancel"
              :title="t('chat.cancel')"
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="3" y="3" width="18" height="18" rx="2"/>
              </svg>
            </button>
            <button 
              v-else
              @click="send" 
              :disabled="!inputText.trim()" 
              :class="['btn-send', { active: inputText.trim() }]"
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                <path d="M12 19V5M5 12l7-7 7 7"/>
              </svg>
            </button>
          </div>
        </div>
      </div>
    </div>
    
    <div v-if="showSessionList" class="backdrop" @click="showSessionList = false"></div>
  </div>
</template>

<style scoped>
.chat-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: #19161d;
  overflow: hidden;
  border-left: 1px solid #28242e;
}

.chat-header {
  height: 35px;
  padding: 0 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid #28242e;
}

.status-indicator {
  display: flex;
  align-items: center;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #ff8080;
}

.status-dot.connecting {
  background: #ffcf99;
  animation: pulse 1s infinite;
}

.status-dot.connected {
  background: #80ffb5;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.session-selector {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  cursor: pointer;
  color: #ffffff;
  font-size: 13px;
  position: relative;
}

.session-selector:hover {
  background: #322e3a;
}

.session-dropdown {
  position: absolute;
  top: 100%;
  left: 0;
  margin-top: 4px;
  width: 240px;
  background: #28242e;
  border: 1px solid #322e3a;
  border-radius: 4px;
  box-shadow: 0 4px 12px rgba(0,0,0,0.3);
  z-index: 1000;
}

.dropdown-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  border-bottom: 1px solid #28242e;
  font-size: 12px;
  color: #938f9b;
}

.btn-new {
  padding: 4px 8px;
  background: #7138cc;
  border: none;
  border-radius: 3px;
  color: white;
  font-size: 11px;
  cursor: pointer;
}

.btn-new:hover {
  background: #b080ff;
}

.session-list {
  max-height: 180px;
  overflow-y: auto;
  padding: 4px;
}

.session-item {
  padding: 6px 8px;
  cursor: pointer;
  font-size: 12px;
  color: #938f9b;
}

.session-item:hover {
  background: #322e3a;
}

.session-item.active {
  background: #7138cc;
  color: white;
}

.empty {
  padding: 12px;
  text-align: center;
  color: #6b6773;
  font-size: 12px;
}

.messages {
  flex: 1;
  overflow-y: auto;
}

.empty-state {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #6b6773;
}

.empty-state svg {
  opacity: 0.3;
  margin-bottom: 16px;
}

.empty-state h3 {
  font-size: 14px;
  font-weight: 400;
  color: #ffffff;
  margin-bottom: 4px;
}

.empty-state p {
  font-size: 12px;
  color: #6b6773;
}

.status-hint {
  margin-top: 12px;
  padding: 6px 12px;
  background: #28242e;
  border-radius: 6px;
  color: #ffcf99 !important;
}

.input-area {
  padding: 12px;
}

.input-box {
  display: flex;
  flex-direction: column;
  background: #28242e;
  border: 1px solid #3c3846;
  border-radius: 12px;
  padding: 12px;
  transition: border-color 0.15s;
}

.input-box:hover {
  border-color: #7138cc;
}

.input-box:focus-within {
  border-color: #7138cc;
}

.input-box textarea {
  flex: 1;
  background: transparent;
  border: none;
  color: #ffffff;
  font-size: 14px;
  resize: none;
  outline: none;
  font-family: inherit;
  line-height: 1.4;
  min-height: 24px;
  max-height: 120px;
}

.input-box textarea::placeholder {
  color: #6b6773;
}

.input-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.btn-send, .btn-cancel {
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.btn-send {
  background: #6b6773;
  color: #19161d;
}

.btn-send:disabled {
  background: #322e3a;
  color: #6b6773;
  cursor: not-allowed;
}

.btn-send:hover:not(:disabled) {
  background: #938f9b;
}

.btn-send.active {
  background: #7138cc;
  color: #ffffff;
}

.btn-send.active:hover {
  background: #b080ff;
}

.btn-cancel {
  background: #ff8080;
  color: white;
}

.input-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 10px;
  padding-top: 0;
}

.toolbar-left, .toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.toolbar-btn {
  width: 28px;
  height: 28px;
  background: transparent;
  border: none;
  color: #6b6773;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
}

.toolbar-btn:hover {
  color: #ffffff;
  background: #322e3a;
}

.backdrop {
  position: fixed;
  inset: 0;
  z-index: 999;
}
</style>
