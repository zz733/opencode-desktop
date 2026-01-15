<script setup>
import { ref, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import { CreateTerminal, WriteTerminal, ResizeTerminal, CloseTerminal } from '../../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

const { t } = useI18n()

const props = defineProps({
  visible: Boolean
})

// 面板类型: terminal, problems, output
const activePanel = ref('terminal')

// 多终端支持
const terminals = ref([]) // { id, name, term, fitAddon, containerRef }
const activeTerminalId = ref(null)
const terminalContainerRef = ref(null)

// 问题和输出数据
const problems = ref([])
const outputs = ref([])

const createTerminalTheme = () => ({
  background: '#1a1a1a',
  foreground: '#e4e4e4',
  cursor: '#e4e4e4',
  cursorAccent: '#1a1a1a',
  selectionBackground: '#4d9cf650',
  black: '#1a1a1a',
  red: '#f87171',
  green: '#4ade80',
  yellow: '#facc15',
  blue: '#60a5fa',
  magenta: '#c084fc',
  cyan: '#22d3ee',
  white: '#e4e4e4',
  brightBlack: '#6b6b6b',
  brightRed: '#fca5a5',
  brightGreen: '#86efac',
  brightYellow: '#fde047',
  brightBlue: '#93c5fd',
  brightMagenta: '#d8b4fe',
  brightCyan: '#67e8f9',
  brightWhite: '#ffffff',
})

// 创建新终端
const addTerminal = async () => {
  try {
    const id = await CreateTerminal()
    const termIndex = terminals.value.length + 1
    
    const termData = {
      id,
      name: `${t('terminal.title')} ${termIndex}`,
      term: null,
      fitAddon: null,
    }
    
    terminals.value.push(termData)
    activeTerminalId.value = id
    
    // 等待 DOM 更新后初始化 xterm
    await nextTick()
    initTerminalInstance(termData)
  } catch (err) {
    console.error('创建终端失败:', err)
  }
}

// 初始化终端实例
const initTerminalInstance = (termData) => {
  const container = document.getElementById(`terminal-${termData.id}`)
  if (!container) return
  
  const term = new Terminal({
    theme: createTerminalTheme(),
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    fontSize: 13,
    lineHeight: 1.2,
    cursorBlink: true,
    cursorStyle: 'bar',
  })
  
  const fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  term.open(container)
  
  termData.term = term
  termData.fitAddon = fitAddon
  
  // 延迟 fit
  setTimeout(() => {
    fitAddon.fit()
    ResizeTerminal(termData.id, term.cols, term.rows)
  }, 50)
  
  // 监听输入
  term.onData((data) => {
    WriteTerminal(termData.id, data)
  })
  
  // 监听后端输出
  EventsOn(`terminal-output-${termData.id}`, (data) => {
    term.write(data)
  })
  
  EventsOn(`terminal-error-${termData.id}`, (err) => {
    term.write(`\r\n\x1b[31mError: ${err}\x1b[0m\r\n`)
  })
  
  term.focus()
}

// 关闭终端
const closeTerminal = (id) => {
  const index = terminals.value.findIndex(t => t.id === id)
  if (index === -1) return
  
  const termData = terminals.value[index]
  
  // 清理事件监听
  EventsOff(`terminal-output-${id}`)
  EventsOff(`terminal-error-${id}`)
  
  // 销毁 xterm
  if (termData.term) {
    termData.term.dispose()
  }
  
  // 调用后端关闭
  CloseTerminal(id)
  
  // 从列表移除
  terminals.value.splice(index, 1)
  
  // 切换到其他终端
  if (activeTerminalId.value === id) {
    activeTerminalId.value = terminals.value.length > 0 ? terminals.value[0].id : null
  }
}

// 切换终端
const switchTerminal = (id) => {
  activeTerminalId.value = id
  nextTick(() => {
    const termData = terminals.value.find(t => t.id === id)
    if (termData?.term) {
      termData.fitAddon?.fit()
      termData.term.focus()
    }
  })
}

// 处理窗口大小变化
const handleResize = () => {
  if (!props.visible || activePanel.value !== 'terminal') return
  
  const termData = terminals.value.find(t => t.id === activeTerminalId.value)
  if (termData?.fitAddon && termData?.term) {
    termData.fitAddon.fit()
    ResizeTerminal(termData.id, termData.term.cols, termData.term.rows)
  }
}

onMounted(async () => {
  window.addEventListener('resize', handleResize)
  // 自动创建第一个终端
  if (props.visible) {
    await addTerminal()
  }
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  // 清理所有终端
  terminals.value.forEach(termData => {
    EventsOff(`terminal-output-${termData.id}`)
    EventsOff(`terminal-error-${termData.id}`)
    if (termData.term) {
      termData.term.dispose()
    }
    CloseTerminal(termData.id)
  })
})

// 当面板显示时
watch(() => props.visible, async (visible) => {
  if (visible) {
    if (terminals.value.length === 0) {
      await addTerminal()
    } else {
      setTimeout(() => {
        handleResize()
        const termData = terminals.value.find(t => t.id === activeTerminalId.value)
        termData?.term?.focus()
      }, 100)
    }
  }
})

// 切换面板类型时
watch(activePanel, (panel) => {
  if (panel === 'terminal') {
    nextTick(() => {
      handleResize()
      const termData = terminals.value.find(t => t.id === activeTerminalId.value)
      termData?.term?.focus()
    })
  }
})
</script>

<template>
  <div class="terminal-panel" v-show="visible">
    <!-- 标签栏 -->
    <div class="panel-tabs">
      <div class="tabs-left">
        <button 
          :class="['tab', { active: activePanel === 'problems' }]"
          @click="activePanel = 'problems'"
        >
          {{ t('panel.problems') }}
          <span v-if="problems.length" class="badge">{{ problems.length }}</span>
        </button>
        <button 
          :class="['tab', { active: activePanel === 'output' }]"
          @click="activePanel = 'output'"
        >
          {{ t('panel.output') }}
        </button>
        <button 
          :class="['tab', { active: activePanel === 'terminal' }]"
          @click="activePanel = 'terminal'"
        >
          {{ t('terminal.title') }}
        </button>
      </div>
      <div class="tabs-right">
        <!-- 终端标签页 -->
        <template v-if="activePanel === 'terminal'">
          <div class="terminal-tabs">
            <button
              v-for="term in terminals"
              :key="term.id"
              :class="['terminal-tab', { active: activeTerminalId === term.id }]"
              @click="switchTerminal(term.id)"
            >
              <span class="tab-name">{{ term.name }}</span>
              <span class="tab-close" @click.stop="closeTerminal(term.id)">×</span>
            </button>
          </div>
          <button class="btn-icon" :title="t('terminal.new')" @click="addTerminal">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 5v14M5 12h14"/>
            </svg>
          </button>
        </template>
      </div>
    </div>
    
    <!-- 内容区域 -->
    <div class="panel-content">
      <!-- 问题面板 -->
      <div v-show="activePanel === 'problems'" class="problems-panel">
        <div v-if="problems.length === 0" class="empty-state">
          {{ t('panel.noProblems') }}
        </div>
        <div v-else class="problems-list">
          <div v-for="(problem, index) in problems" :key="index" class="problem-item">
            <span :class="['icon', problem.severity]"></span>
            <span class="message">{{ problem.message }}</span>
            <span class="location">{{ problem.file }}:{{ problem.line }}</span>
          </div>
        </div>
      </div>
      
      <!-- 输出面板 -->
      <div v-show="activePanel === 'output'" class="output-panel">
        <div v-if="outputs.length === 0" class="empty-state">
          {{ t('panel.noOutput') }}
        </div>
        <div v-else class="output-content">
          <div v-for="(line, index) in outputs" :key="index" class="output-line">
            {{ line }}
          </div>
        </div>
      </div>
      
      <!-- 终端面板 -->
      <div v-show="activePanel === 'terminal'" class="terminals-container">
        <div
          v-for="term in terminals"
          :key="term.id"
          :id="`terminal-${term.id}`"
          :class="['terminal-instance', { active: activeTerminalId === term.id }]"
        ></div>
      </div>
    </div>
  </div>
</template>


<style scoped>
.terminal-panel {
  display: flex;
  flex-direction: column;
  background: #1a1a1a;
  border-top: 1px solid var(--border-default);
  height: 100%;
}

.panel-tabs {
  height: 35px;
  min-height: 35px;
  padding: 0 8px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: var(--bg-surface);
  border-bottom: 1px solid var(--border-default);
}

.tabs-left {
  display: flex;
  gap: 2px;
}

.tabs-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.tab {
  padding: 4px 12px;
  background: transparent;
  border: none;
  border-radius: 4px;
  color: var(--text-muted);
  font-size: 12px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
}

.tab:hover {
  color: var(--text-secondary);
}

.tab.active {
  color: var(--text-primary);
  background: var(--bg-hover);
}

.badge {
  background: var(--accent-primary);
  color: white;
  font-size: 10px;
  padding: 1px 5px;
  border-radius: 8px;
  min-width: 16px;
  text-align: center;
}

.terminal-tabs {
  display: flex;
  gap: 2px;
}

.terminal-tab {
  padding: 3px 8px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-default);
  border-radius: 4px;
  color: var(--text-muted);
  font-size: 11px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
}

.terminal-tab:hover {
  background: var(--bg-hover);
  color: var(--text-secondary);
}

.terminal-tab.active {
  background: var(--bg-active);
  color: var(--text-primary);
  border-color: var(--accent-primary);
}

.tab-close {
  font-size: 14px;
  line-height: 1;
  opacity: 0.5;
}

.tab-close:hover {
  opacity: 1;
}

.btn-icon {
  width: 24px;
  height: 24px;
  background: transparent;
  border: none;
  border-radius: 4px;
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

.panel-content {
  flex: 1;
  overflow: hidden;
  position: relative;
}

/* 问题面板 */
.problems-panel,
.output-panel {
  height: 100%;
  overflow-y: auto;
  padding: 8px;
}

.empty-state {
  color: var(--text-muted);
  font-size: 12px;
  text-align: center;
  padding: 20px;
}

.problems-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.problem-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
}

.problem-item:hover {
  background: var(--bg-hover);
}

.problem-item .icon {
  width: 14px;
  height: 14px;
  border-radius: 50%;
}

.problem-item .icon.error {
  background: #f87171;
}

.problem-item .icon.warning {
  background: #facc15;
}

.problem-item .icon.info {
  background: #60a5fa;
}

.problem-item .message {
  flex: 1;
  color: var(--text-primary);
}

.problem-item .location {
  color: var(--text-muted);
}

/* 输出面板 */
.output-content {
  font-family: Menlo, Monaco, "Courier New", monospace;
  font-size: 12px;
  line-height: 1.4;
}

.output-line {
  padding: 2px 0;
  color: var(--text-secondary);
}

/* 终端容器 */
.terminals-container {
  height: 100%;
  position: relative;
}

.terminal-instance {
  position: absolute;
  inset: 0;
  display: none;
  padding: 4px 0 4px 8px;
}

.terminal-instance.active {
  display: block;
}

/* xterm 样式覆盖 */
.terminal-instance :deep(.xterm) {
  height: 100%;
  width: 100%;
  text-align: left !important;
}

.terminal-instance :deep(.xterm-screen) {
  margin: 0 !important;
  padding: 0 !important;
  width: 100% !important;
}

.terminal-instance :deep(.xterm-screen canvas) {
  display: block !important;
}

.terminal-instance :deep(.xterm-rows) {
  padding: 0 !important;
  margin: 0 !important;
}

.terminal-instance :deep(.xterm-helpers) {
  position: absolute !important;
  top: 0 !important;
  left: 0 !important;
}

.terminal-instance :deep(.xterm-viewport) {
  overflow-y: auto !important;
  width: 100% !important;
}

.terminal-instance :deep(.xterm-viewport::-webkit-scrollbar) {
  width: 8px;
}

.terminal-instance :deep(.xterm-viewport::-webkit-scrollbar-thumb) {
  background: #333;
  border-radius: 4px;
}
</style>
