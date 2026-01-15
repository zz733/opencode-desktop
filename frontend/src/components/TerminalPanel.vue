<script setup>
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import { StartTerminal, WriteTerminal, ResizeTerminal } from '../../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

const { t } = useI18n()

const props = defineProps({
  visible: Boolean
})

const terminalRef = ref(null)
let term = null
let fitAddon = null

onMounted(async () => {
  // 创建终端
  term = new Terminal({
    theme: {
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
    },
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    fontSize: 13,
    lineHeight: 1.2,
    cursorBlink: true,
    cursorStyle: 'bar',
  })

  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)

  // 挂载到 DOM
  if (terminalRef.value) {
    term.open(terminalRef.value)
    // 延迟 fit 确保 DOM 已渲染
    setTimeout(() => {
      fitAddon.fit()
    }, 50)
  }

  // 监听输入
  term.onData((data) => {
    WriteTerminal(data)
  })

  // 监听后端输出
  EventsOn('terminal-output', (data) => {
    term.write(data)
  })

  EventsOn('terminal-error', (err) => {
    term.write(`\r\n\x1b[31mError: ${err}\x1b[0m\r\n`)
  })

  // 启动终端
  await StartTerminal()

  // 发送初始大小
  const { cols, rows } = term
  await ResizeTerminal(cols, rows)

  // 监听窗口大小变化
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  EventsOff('terminal-output')
  EventsOff('terminal-error')
  window.removeEventListener('resize', handleResize)
  if (term) {
    term.dispose()
  }
})

const handleResize = () => {
  if (fitAddon && props.visible) {
    fitAddon.fit()
    if (term) {
      ResizeTerminal(term.cols, term.rows)
    }
  }
}

// 当面板显示时重新 fit
watch(() => props.visible, (visible) => {
  if (visible) {
    setTimeout(() => {
      handleResize()
      term?.focus()
    }, 100)
  }
})
</script>

<template>
  <div class="terminal-panel" v-show="visible">
    <div class="terminal-header">
      <span class="title">{{ t('terminal.title') }}</span>
      <div class="actions">
        <button class="btn-icon" :title="t('terminal.new')">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 5v14M5 12h14"/>
          </svg>
        </button>
      </div>
    </div>
    <div class="terminal-container" ref="terminalRef"></div>
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

.terminal-header {
  height: 32px;
  min-height: 32px;
  padding: 0 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: var(--bg-surface);
  border-bottom: 1px solid var(--border-default);
}

.title {
  font-size: 12px;
  color: var(--text-secondary);
}

.actions {
  display: flex;
  gap: 4px;
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

.terminal-container {
  flex: 1;
  overflow: hidden;
  padding: 4px 0 4px 8px;
}

/* 强制 xterm 左对齐 */
.terminal-container :deep(.xterm) {
  height: 100%;
  width: 100%;
  text-align: left !important;
}

.terminal-container :deep(.xterm-screen) {
  margin: 0 !important;
  padding: 0 !important;
  width: 100% !important;
}

.terminal-container :deep(.xterm-screen canvas) {
  display: block !important;
}

.terminal-container :deep(.xterm-rows) {
  padding: 0 !important;
  margin: 0 !important;
}

.terminal-container :deep(.xterm-helpers) {
  position: absolute !important;
  top: 0 !important;
  left: 0 !important;
}

.terminal-container :deep(.xterm-viewport) {
  overflow-y: auto !important;
  width: 100% !important;
}

.terminal-container :deep(.xterm-viewport::-webkit-scrollbar) {
  width: 8px;
}

.terminal-container :deep(.xterm-viewport::-webkit-scrollbar-thumb) {
  background: #333;
  border-radius: 4px;
}
</style>
