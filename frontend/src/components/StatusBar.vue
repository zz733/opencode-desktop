<script setup>
import { computed } from 'vue'

const props = defineProps({
  connected: Boolean,
  connecting: Boolean,
  currentModel: String,
  sessionTitle: String
})

const connectionStatus = computed(() => {
  if (props.connected) return { text: 'OpenCode 已连接', class: 'connected' }
  if (props.connecting) return { text: '连接中...', class: 'connecting' }
  return { text: '未连接', class: 'disconnected' }
})
</script>

<template>
  <div class="status-bar">
    <div class="status-left">
      <div :class="['status-item', 'connection', connectionStatus.class]">
        <span class="status-dot"></span>
        <span>{{ connectionStatus.text }}</span>
      </div>
      <div v-if="sessionTitle" class="status-item">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
        </svg>
        <span>{{ sessionTitle }}</span>
      </div>
    </div>
    <div class="status-right">
      <div v-if="currentModel" class="status-item">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="3"/>
          <path d="M12 1v4M12 19v4M4.22 4.22l2.83 2.83M16.95 16.95l2.83 2.83M1 12h4M19 12h4M4.22 19.78l2.83-2.83M16.95 7.05l2.83-2.83"/>
        </svg>
        <span>{{ currentModel.split('/').pop() }}</span>
      </div>
      <div class="status-item">
        <span>UTF-8</span>
      </div>
      <div class="status-item">
        <span>Ln 1, Col 1</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.status-bar {
  height: 22px;
  background: #211d25;
  border-top: 1px solid #28242e;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 8px;
  font-size: 11px;
  color: #938f9b;
}

.status-left, .status-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.status-item {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 0 4px;
  cursor: default;
}

.status-item:hover {
  color: #ffffff;
}

.status-item svg {
  opacity: 0.7;
}

.connection {
  gap: 6px;
}

.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

.connection.connected .status-dot {
  background: #80ffb5;
}

.connection.connecting .status-dot {
  background: #ffcf99;
  animation: pulse 1s infinite;
}

.connection.disconnected .status-dot {
  background: #ff8080;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}
</style>
