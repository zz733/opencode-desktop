<script setup>
import { ref } from 'vue'
import KiroAccountManager from './KiroAccountManager.vue'

const emit = defineEmits(['close'])

const handleClose = () => {
  emit('close')
}

// 阻止点击内容区域时关闭
const handleContentClick = (e) => {
  e.stopPropagation()
}
</script>

<template>
  <div class="dialog-overlay" @click="handleClose">
    <div class="dialog-container" @click="handleContentClick">
      <div class="dialog-header">
        <h2>Kiro 账号管理</h2>
        <button class="btn-close" @click="handleClose">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M18 6L6 18M6 6l12 12"/>
          </svg>
        </button>
      </div>
      <div class="dialog-content">
        <KiroAccountManager />
      </div>
    </div>
  </div>
</template>

<style scoped>
.dialog-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10000;
  backdrop-filter: blur(4px);
}

.dialog-container {
  background: var(--bg-surface);
  border-radius: 12px;
  border: 1px solid var(--border-subtle);
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
  width: 90vw;
  max-width: 1200px;
  height: 85vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.dialog-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px 24px;
  border-bottom: 1px solid var(--border-subtle);
  background: var(--bg-elevated);
}

.dialog-header h2 {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.btn-close {
  width: 32px;
  height: 32px;
  border-radius: 6px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.btn-close:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.dialog-content {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.dialog-content :deep(.kiro-account-manager) {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
}
</style>
