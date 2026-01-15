<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps({
  message: Object,
  isLoading: Boolean
})

// 工具图标映射 - 根据状态返回不同图标
const getToolIcon = (tool) => {
  if (tool.status === 'running' || tool.status === 'pending') {
    return 'spinner' // 转圈
  }
  if (tool.status === 'completed') {
    return 'check' // 勾选
  }
  if (tool.status === 'error') {
    return 'error' // 错误
  }
  return 'default'
}

// 工具类型图标
const getToolTypeIcon = (name) => {
  const n = name?.toLowerCase() || ''
  if (n.includes('bash') || n.includes('command') || n.includes('shell')) {
    return 'terminal'
  }
  if (n.includes('edit') || n.includes('write') || n.includes('create')) {
    return 'edit'
  }
  if (n.includes('read') || n.includes('file')) {
    return 'file'
  }
  if (n.includes('search') || n.includes('grep') || n.includes('find')) {
    return 'search'
  }
  if (n.includes('list') || n.includes('dir')) {
    return 'folder'
  }
  return 'tool'
}

// 工具显示名称
const getToolDisplayName = (name) => {
  const n = name?.toLowerCase() || ''
  if (n.includes('bash') || n.includes('command') || n.includes('shell')) {
    return 'Command'
  }
  if (n.includes('edit') || n.includes('write') || n.includes('str_replace')) {
    return 'Editing'
  }
  if (n.includes('read')) {
    return 'Read file(s)'
  }
  if (n.includes('search') || n.includes('grep')) {
    return 'Search'
  }
  if (n.includes('list')) {
    return 'List directory'
  }
  return name
}

// 展开/折叠工具详情
const expandedTools = ref({})

const toggleTool = (id) => {
  expandedTools.value[id] = !expandedTools.value[id]
}

// 格式化工具输入显示
const formatToolInput = (tool) => {
  if (!tool.input) return ''
  return JSON.stringify(tool.input, null, 2)
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
            <!-- 状态图标 -->
            <span :class="['status-icon', tool.status]">
              <!-- 转圈 -->
              <svg v-if="getToolIcon(tool) === 'spinner'" class="spinner" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
              </svg>
              <!-- 勾选 -->
              <svg v-else-if="getToolIcon(tool) === 'check'" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M20 6L9 17l-5-5"/>
              </svg>
              <!-- 错误 -->
              <svg v-else-if="getToolIcon(tool) === 'error'" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/><path d="M15 9l-6 6M9 9l6 6"/>
              </svg>
              <!-- 默认 -->
              <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="3"/>
              </svg>
            </span>
            
            <!-- 工具类型图标 -->
            <span class="type-icon">
              <svg v-if="getToolTypeIcon(tool.name) === 'terminal'" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/>
              </svg>
              <svg v-else-if="getToolTypeIcon(tool.name) === 'edit'" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
              </svg>
              <svg v-else-if="getToolTypeIcon(tool.name) === 'file'" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 2v6h6"/>
              </svg>
              <svg v-else-if="getToolTypeIcon(tool.name) === 'search'" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/>
              </svg>
              <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"/>
              </svg>
            </span>
            
            <span class="tool-name">{{ getToolDisplayName(tool.name) }}</span>
            
            <svg 
              :class="['expand-icon', { expanded: expandedTools[tool.id] }]"
              width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
            >
              <path d="M6 9l6 6 6-6"/>
            </svg>
          </div>
          
          <!-- 工具内容 - 默认展开显示命令 -->
          <div class="tool-content" v-if="tool.input">
            <!-- 命令显示 -->
            <div v-if="tool.input.command" class="command-line">
              <code>{{ tool.input.command }}</code>
            </div>
            <!-- 文件路径 -->
            <div v-else-if="tool.input.path" class="file-path">
              <code>{{ tool.input.path }}</code>
            </div>
            <!-- 搜索查询 -->
            <div v-else-if="tool.input.query" class="search-query">
              <code>{{ tool.input.query }}</code>
            </div>
          </div>
          
          <!-- 展开的详细输出 -->
          <div v-if="expandedTools[tool.id] && tool.output" class="tool-output">
            <pre>{{ tool.output }}</pre>
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
  gap: 10px;
  padding: 10px 16px;
}

.avatar {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 500;
  flex-shrink: 0;
}

.avatar.user {
  background: var(--blue);
  color: var(--bg-base);
}

.avatar.assistant {
  background: var(--accent-primary);
  color: var(--bg-base);
}

.content {
  flex: 1;
  min-width: 0;
}

.role-name {
  font-size: 12px;
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: 4px;
}

/* 工具卡片 */
.tools {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 8px;
}

.tool-card {
  background: var(--bg-elevated);
  border-radius: 6px;
  overflow: hidden;
}

.tool-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 10px;
  cursor: pointer;
}

.tool-header:hover {
  background: var(--bg-hover);
}

.status-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
}

.status-icon.running svg,
.status-icon.pending svg {
  color: var(--blue);
}

.status-icon.completed svg {
  color: var(--green);
}

.status-icon.error svg {
  color: var(--red);
}

.spinner {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.type-icon {
  display: flex;
  align-items: center;
  color: var(--text-muted);
}

.tool-name {
  font-size: 12px;
  color: var(--text-secondary);
  flex: 1;
}

.expand-icon {
  color: var(--text-muted);
  transition: transform 0.15s;
}

.expand-icon.expanded {
  transform: rotate(180deg);
}

/* 工具内容 */
.tool-content {
  padding: 0 10px 8px;
}

.command-line,
.file-path,
.search-query {
  background: rgba(0,0,0,0.2);
  border-radius: 4px;
  padding: 6px 8px;
  overflow-x: auto;
}

.command-line code,
.file-path code,
.search-query code {
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 11px;
  color: var(--text-secondary);
  white-space: pre-wrap;
  word-break: break-all;
}

/* 工具输出 */
.tool-output {
  border-top: 1px solid var(--border-default);
  padding: 8px 10px;
  background: rgba(0,0,0,0.15);
  max-height: 120px;
  overflow-y: auto;
}

.tool-output pre {
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 10px;
  color: var(--text-muted);
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
}

/* 思考过程 */
.reasoning {
  background: rgba(176, 128, 255, 0.1);
  border-radius: 6px;
  margin-bottom: 8px;
  overflow: hidden;
}

.reasoning-header {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 10px;
  font-size: 11px;
  color: var(--accent-primary);
  font-weight: 500;
}

.reasoning-content {
  padding: 0 10px 8px;
  font-size: 12px;
  color: var(--text-secondary);
  line-height: 1.4;
}

/* 正文 */
.text pre {
  white-space: pre-wrap;
  word-break: break-word;
  font-family: inherit;
  font-size: 13px;
  line-height: 1.5;
  color: var(--text-primary);
  margin: 0;
}

/* working 动画 */
.working-dots {
  color: var(--text-muted);
  font-size: 12px;
}

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
