<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps({
  message: Object,
  isLoading: Boolean
})

// 工具图标映射
const toolIcons = {
  'Read file': '📄',
  'Read file(s)': '📄',
  'Write file': '✏️',
  'Command': '⌨️',
  'Search': '🔍',
  'Grep': '🔍',
  'List': '📁',
  'default': '🔧'
}

const getToolIcon = (name) => {
  for (const key in toolIcons) {
    if (name?.toLowerCase().includes(key.toLowerCase())) {
      return toolIcons[key]
    }
  }
  return toolIcons.default
}

// 展开/折叠工具详情
const expandedTools = ref({})

const toggleTool = (id) => {
  expandedTools.value[id] = !expandedTools.value[id]
}

// 格式化工具参数显示
const formatToolArgs = (tool) => {
  if (tool.args) {
    return JSON.stringify(tool.args, null, 2)
  }
  return ''
}
</script>

<template>
  <div :class="['message', message.role]">
    <div class="avatar" :class="message.role">
      <span v-if="message.role === 'user'">U</span>
      <span v-else>K</span>
    </div>
    
    <div class="content">
      <div class="role-name">{{ message.role === 'user' ? t('chat.you') : t('chat.assistant') }}</div>
      
      <!-- 工具调用 -->
      <div v-if="message.tools && Object.keys(message.tools).length" class="tools">
        <div 
          v-for="tool in message.tools" 
          :key="tool.id" 
          class="tool-card"
        >
          <div class="tool-header" @click="toggleTool(tool.id)">
            <span class="tool-icon">{{ getToolIcon(tool.name) }}</span>
            <span class="tool-name">{{ tool.name }}</span>
            <span :class="['tool-status', tool.status]">
              <template v-if="tool.status === 'running'">
                <span class="working-dots">working</span>
              </template>
              <template v-else>
                {{ tool.status }}
              </template>
            </span>
            <svg 
              :class="['expand-icon', { expanded: expandedTools[tool.id] }]"
              width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
            >
              <path d="M6 9l6 6 6-6"/>
            </svg>
          </div>
          
          <!-- 工具详情 -->
          <div v-if="expandedTools[tool.id] && tool.args" class="tool-details">
            <pre>{{ formatToolArgs(tool) }}</pre>
          </div>
        </div>
      </div>
      
      <!-- 思考过程 -->
      <div v-if="message.reasoning" class="reasoning">
        <div class="reasoning-header">
          <span class="reasoning-icon">💭</span>
          <span>Thinking</span>
        </div>
        <div class="reasoning-content">{{ message.reasoning }}</div>
      </div>
      
      <!-- 正文内容 -->
      <div class="text" v-if="message.content || isLoading">
        <template v-if="isLoading && !message.content">
          <span class="working-dots">thinking</span>
        </template>
        <template v-else>
          <pre>{{ message.content }}</pre>
        </template>
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
  margin-bottom: 8px;
}

/* 工具卡片 */
.tools {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 12px;
}

.tool-card {
  background: var(--bg-elevated);
  border: 1px solid var(--border-default);
  border-radius: 8px;
  overflow: hidden;
}

.tool-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  cursor: pointer;
  transition: background 0.15s;
}

.tool-header:hover {
  background: var(--bg-hover);
}

.tool-icon {
  font-size: 14px;
}

.tool-name {
  flex: 1;
  font-size: 13px;
  color: var(--text-primary);
  font-weight: 500;
}

.tool-status {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 4px;
  font-weight: 500;
}

.tool-status.pending { 
  background: rgba(250, 204, 21, 0.15); 
  color: #fcd34d; 
}

.tool-status.running { 
  background: rgba(96, 165, 250, 0.15); 
  color: #93c5fd; 
}

.tool-status.completed { 
  background: rgba(74, 222, 128, 0.15); 
  color: #86efac; 
}

.tool-status.error { 
  background: rgba(248, 113, 113, 0.15); 
  color: #fca5a5; 
}

.expand-icon {
  color: var(--text-muted);
  transition: transform 0.2s;
}

.expand-icon.expanded {
  transform: rotate(180deg);
}

.tool-details {
  border-top: 1px solid var(--border-default);
  padding: 10px 12px;
  background: var(--bg-base);
}

.tool-details pre {
  font-family: 'Menlo', 'Monaco', 'Courier New', monospace;
  font-size: 11px;
  color: var(--text-secondary);
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
}

/* 思考过程 */
.reasoning {
  background: rgba(167, 139, 250, 0.08);
  border: 1px solid rgba(167, 139, 250, 0.2);
  border-radius: 8px;
  margin-bottom: 12px;
  overflow: hidden;
}

.reasoning-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  font-size: 12px;
  color: var(--purple);
  font-weight: 500;
}

.reasoning-content {
  padding: 0 12px 10px;
  font-size: 12px;
  color: #c4b5fd;
  line-height: 1.5;
}

/* 正文 */
.text pre {
  white-space: pre-wrap;
  word-break: break-word;
  font-family: inherit;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text-primary);
  margin: 0;
}

/* working 动画 */
.working-dots::after {
  content: '';
  animation: dots 1.5s infinite;
}

@keyframes dots {
  0%, 20% { content: '.'; }
  40% { content: '..'; }
  60%, 100% { content: '...'; }
}
</style>
