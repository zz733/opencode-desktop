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
  models: Array
})

const emit = defineEmits(['selectSession', 'send', 'update:currentModel'])

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
  if (!inputText.value.trim() || props.sending) return
  emit('send', inputText.value)
  inputText.value = ''
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
      
      <ModelSelector 
        :modelValue="currentModel"
        :models="models"
        @update:modelValue="emit('update:currentModel', $event)"
      />
    </header>
    
    <!-- 消息 -->
    <div class="messages" ref="messagesContainer">
      <div v-if="!messages.length" class="empty-state">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
          <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
        </svg>
        <h3>{{ t('chat.howCanIHelp') }}</h3>
        <p>{{ t('chat.askAnything') }}</p>
      </div>
      
      <MessageItem 
        v-for="(msg, i) in messages" 
        :key="i" 
        :message="msg"
        :isLoading="sending && i === messages.length - 1 && msg.role === 'assistant'"
      />
    </div>
    
    <!-- 输入 -->
    <div class="input-area">
      <div class="input-box">
        <textarea 
          v-model="inputText"
          :placeholder="t('chat.placeholder')"
          @keydown="handleKeydown"
          :disabled="sending"
        ></textarea>
        <div class="input-actions">
          <button class="btn-icon" title="Attach">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21.44 11.05l-9.19 9.19a6 6 0 0 1-8.49-8.49l9.19-9.19a4 4 0 0 1 5.66 5.66l-9.2 9.19a2 2 0 0 1-2.83-2.83l8.49-8.48"/>
            </svg>
          </button>
          <button @click="send" :disabled="sending || !inputText.trim()" class="btn-send">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M22 2L11 13M22 2l-7 20-4-9-9-4 20-7z"/>
            </svg>
          </button>
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
  background: var(--bg-base);
  border-left: 1px solid var(--border-default);
}

.chat-header {
  height: 44px;
  padding: 0 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: var(--bg-surface);
  border-bottom: 1px solid var(--border-default);
}

.session-selector {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border-radius: 6px;
  cursor: pointer;
  color: var(--text-primary);
  font-size: 13px;
  position: relative;
}

.session-selector:hover {
  background: var(--bg-hover);
}

.session-dropdown {
  position: absolute;
  top: 100%;
  left: 0;
  margin-top: 4px;
  width: 260px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-default);
  border-radius: 8px;
  box-shadow: 0 8px 24px rgba(0,0,0,0.3);
  z-index: 1000;
  overflow: hidden;
}

.dropdown-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 12px;
  border-bottom: 1px solid var(--border-default);
  font-size: 12px;
  color: var(--text-secondary);
}

.btn-new {
  padding: 4px 10px;
  background: var(--accent-primary);
  border: none;
  border-radius: 4px;
  color: white;
  font-size: 12px;
  cursor: pointer;
}

.btn-new:hover {
  background: var(--accent-hover);
}

.session-list {
  max-height: 200px;
  overflow-y: auto;
  padding: 6px;
}

.session-item {
  padding: 8px 10px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
  color: var(--text-secondary);
}

.session-item:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.session-item.active {
  background: var(--accent-primary);
  color: white;
}

.empty {
  padding: 16px;
  text-align: center;
  color: var(--text-muted);
  font-size: 13px;
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
  color: var(--text-muted);
  padding: 40px;
}

.empty-state svg {
  opacity: 0.3;
  margin-bottom: 16px;
}

.empty-state h3 {
  font-size: 16px;
  font-weight: 500;
  color: var(--text-secondary);
  margin-bottom: 8px;
}

.input-area {
  padding: 12px;
  background: var(--bg-surface);
  border-top: 1px solid var(--border-default);
}

.input-box {
  background: var(--bg-input);
  border: 1px solid var(--border-default);
  border-radius: 10px;
  padding: 10px 12px;
  transition: border-color 0.15s;
}

.input-box:focus-within {
  border-color: var(--accent-primary);
}

.input-box textarea {
  width: 100%;
  background: transparent;
  border: none;
  color: var(--text-primary);
  font-size: 13px;
  resize: none;
  outline: none;
  font-family: inherit;
  line-height: 1.5;
  min-height: 20px;
  max-height: 100px;
}

.input-box textarea::placeholder {
  color: var(--text-muted);
}

.input-actions {
  display: flex;
  justify-content: flex-end;
  gap: 6px;
  margin-top: 8px;
}

.btn-icon {
  width: 32px;
  height: 32px;
  background: transparent;
  border: none;
  border-radius: 6px;
  color: var(--text-muted);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}

.btn-icon:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.btn-send {
  width: 32px;
  height: 32px;
  background: var(--accent-primary);
  border: none;
  border-radius: 6px;
  color: white;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}

.btn-send:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.btn-send:hover:not(:disabled) {
  background: var(--accent-hover);
}

.backdrop {
  position: fixed;
  inset: 0;
  z-index: 999;
}
</style>
