<script setup>
import { ref, nextTick, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import MessageItem from './MessageItem.vue'
import ModelSelector from './ModelSelector.vue'
import FileEditCard from './FileEditCard.vue'
import { useFileEdits } from '../composables/useFileEdits'

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

const emit = defineEmits(['selectSession', 'send', 'update:currentModel', 'cancel', 'compare', 'revertEdit'])

const { fileEdits, revertEdit } = useFileEdits()

const inputText = ref('')
const messagesContainer = ref(null)
const showSessionList = ref(false)

// 合并消息和编辑记录，按时间排序
const combinedItems = computed(() => {
  const items = []
  
  // 添加消息，带时间戳
  props.messages.forEach((msg, index) => {
    items.push({
      type: 'message',
      data: msg,
      index,
      // 消息没有时间戳，用索引作为排序依据
      order: index * 1000
    })
  })
  
  // 添加编辑记录
  fileEdits.value.forEach(edit => {
    // 找到编辑记录应该插入的位置（在最后一条 assistant 消息之后）
    const lastAssistantIndex = props.messages.length - 1
    items.push({
      type: 'edit',
      data: edit,
      // 编辑记录放在消息之后，用时间戳区分多个编辑
      order: (lastAssistantIndex + 1) * 1000 + (edit.timestamp % 1000)
    })
  })
  
  // 按 order 排序
  return items.sort((a, b) => a.order - b.order)
})

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

const handleCompare = (edit) => {
  emit('compare', edit)
}

const handleRevert = async (editId) => {
  const success = await revertEdit(editId)
  if (success) {
    emit('revertEdit', editId)
  }
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

// 监听文件编辑变化，自动滚动
watch(() => fileEdits.value.length, () => {
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
    </header>
    
    <!-- 消息 -->
    <div class="messages" ref="messagesContainer">
      <div v-if="!messages.length && !fileEdits.length" class="empty-state">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
          <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
        </svg>
        <h3>{{ t('chat.howCanIHelp') }}</h3>
        <p>{{ t('chat.askAnything') }}</p>
        <p v-if="!connected" class="status-hint">{{ connecting ? '正在连接...' : '未连接到 OpenCode 服务' }}</p>
      </div>
      
      <template v-for="item in combinedItems" :key="item.type + '-' + (item.data.id || item.index)">
        <MessageItem 
          v-if="item.type === 'message'"
          :message="item.data"
          :isLoading="sending && item.index === messages.length - 1 && item.data.role === 'assistant'"
        />
        <div v-else-if="item.type === 'edit'" class="edit-card-wrapper">
          <FileEditCard
            :edit="item.data"
            @compare="handleCompare"
            @revert="handleRevert"
          />
        </div>
      </template>
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
  background: var(--bg-base);
  overflow: hidden;
  border-left: 1px solid var(--border-default);
}

.chat-header {
  height: 35px;
  padding: 0 12px;
  display: flex;
  align-items: center;
  border-bottom: 1px solid var(--border-default);
}

.session-selector {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
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
  width: 240px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-subtle);
  border-radius: 4px;
  box-shadow: 0 4px 12px rgba(0,0,0,0.3);
  z-index: 1000;
}

.dropdown-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  border-bottom: 1px solid var(--border-default);
  font-size: 12px;
  color: var(--text-secondary);
}

.btn-new {
  padding: 4px 8px;
  background: var(--accent-button);
  border: none;
  border-radius: 3px;
  color: white;
  font-size: 11px;
  cursor: pointer;
}

.btn-new:hover {
  background: var(--accent-primary);
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
  color: var(--text-secondary);
}

.session-item:hover {
  background: var(--bg-hover);
}

.session-item.active {
  background: var(--accent-button);
  color: white;
}

.empty {
  padding: 12px;
  text-align: center;
  color: var(--text-muted);
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
  color: var(--text-muted);
}

.empty-state svg {
  opacity: 0.3;
  margin-bottom: 16px;
}

.empty-state h3 {
  font-size: 14px;
  font-weight: 400;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.empty-state p {
  font-size: 12px;
  color: var(--text-muted);
}

.status-hint {
  margin-top: 12px;
  padding: 6px 12px;
  background: var(--bg-elevated);
  border-radius: 6px;
  color: var(--yellow) !important;
}

.edit-card-wrapper {
  padding: 0 16px 8px;
}

.input-area {
  padding: 12px;
}

.input-box {
  display: flex;
  flex-direction: column;
  background: var(--bg-elevated);
  border: 1px solid var(--bg-active);
  border-radius: 12px;
  padding: 12px;
  transition: border-color 0.15s;
}

.input-box:hover {
  border-color: var(--accent-button);
}

.input-box:focus-within {
  border-color: var(--accent-button);
}

.input-box textarea {
  flex: 1;
  background: transparent;
  border: none;
  color: var(--text-primary);
  font-size: 14px;
  resize: none;
  outline: none;
  font-family: inherit;
  line-height: 1.4;
  min-height: 24px;
  max-height: 120px;
}

.input-box textarea::placeholder {
  color: var(--text-muted);
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
  background: var(--text-muted);
  color: var(--bg-base);
}

.btn-send:disabled {
  background: var(--bg-hover);
  color: var(--text-muted);
  cursor: not-allowed;
}

.btn-send:hover:not(:disabled) {
  background: var(--text-secondary);
}

.btn-send.active {
  background: var(--accent-button);
  color: var(--text-primary);
}

.btn-send.active:hover {
  background: var(--accent-primary);
}

.btn-cancel {
  background: var(--red);
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
  color: var(--text-muted);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
}

.toolbar-btn:hover {
  color: var(--text-primary);
  background: var(--bg-hover);
}

.backdrop {
  position: fixed;
  inset: 0;
  z-index: 999;
}
</style>
