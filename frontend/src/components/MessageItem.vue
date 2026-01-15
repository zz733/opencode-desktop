<script setup>
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

defineProps({
  message: Object,
  isLoading: Boolean
})
</script>

<template>
  <div :class="['message', message.role]">
    <div class="avatar" :class="message.role">
      <span v-if="message.role === 'user'">U</span>
      <span v-else>K</span>
    </div>
    
    <div class="content">
      <div class="role-name">{{ message.role === 'user' ? t('chat.you') : t('chat.assistant') }}</div>
      
      <!-- 思考 -->
      <div v-if="message.reasoning" class="reasoning">
        <div class="label">💭 {{ t('chat.thinking') }}</div>
        <pre>{{ message.reasoning }}</pre>
      </div>
      
      <!-- 工具 -->
      <div v-if="message.tools && Object.keys(message.tools).length" class="tools">
        <div v-for="tool in message.tools" :key="tool.id" class="tool-item">
          <span class="tool-icon">🔧</span>
          <span class="tool-name">{{ tool.name }}</span>
          <span :class="['tool-status', tool.status]">{{ tool.status }}</span>
        </div>
      </div>
      
      <!-- 内容 -->
      <div class="text">
        <pre>{{ message.content || (isLoading ? t('chat.thinking') : '') }}</pre>
      </div>
    </div>
  </div>
</template>

<style scoped>
.message {
  display: flex;
  gap: 12px;
  padding: 14px 16px;
}

.message:hover {
  background: var(--bg-hover);
}

.avatar {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  flex-shrink: 0;
}

.avatar.user {
  background: var(--accent-primary);
  color: white;
}

.avatar.assistant {
  background: var(--purple);
  color: white;
}

.content {
  flex: 1;
  min-width: 0;
}

.role-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 6px;
}

.text pre {
  white-space: pre-wrap;
  word-break: break-word;
  font-family: inherit;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text-primary);
  margin: 0;
}

.reasoning {
  background: rgba(167, 139, 250, 0.1);
  border-left: 3px solid var(--purple);
  padding: 10px 14px;
  border-radius: 0 6px 6px 0;
  margin-bottom: 10px;
}

.reasoning .label {
  font-size: 12px;
  color: var(--purple);
  margin-bottom: 6px;
}

.reasoning pre {
  font-size: 12px;
  color: #c4b5fd;
  white-space: pre-wrap;
  font-family: inherit;
  margin: 0;
}

.tools {
  margin-bottom: 10px;
}

.tool-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: var(--bg-elevated);
  border-radius: 6px;
  margin-bottom: 4px;
  font-size: 12px;
}

.tool-name {
  color: var(--accent-primary);
}

.tool-status {
  margin-left: auto;
  font-size: 10px;
  padding: 2px 8px;
  border-radius: 4px;
}

.tool-status.pending { background: #5c4813; color: #fcd34d; }
.tool-status.running { background: #1e3a5f; color: #93c5fd; }
.tool-status.completed { background: #1e4620; color: #86efac; }
</style>
