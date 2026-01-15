<script setup>
import { ref } from 'vue'

const props = defineProps({
  edit: Object
})

const emit = defineEmits(['compare', 'revert', 'dismiss'])

const reverting = ref(false)

const handleRevert = async () => {
  reverting.value = true
  emit('revert', props.edit.id)
}
</script>

<template>
  <div :class="['file-edit-card', { reverted: edit.reverted }]">
    <div class="edit-status">
      <svg v-if="edit.reverted" class="icon-reverted" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/>
        <path d="M3 3v5h5"/>
      </svg>
      <svg v-else class="icon-check" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M20 6L9 17l-5-5"/>
      </svg>
      <span class="edit-text">
        {{ edit.reverted ? 'Reverted edits to' : 'Accepted edits to' }}
      </span>
      <span class="edit-filename">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
          <path d="M14 2v6h6"/>
        </svg>
        {{ edit.filename }}
      </span>
    </div>
    <div class="edit-actions" v-if="!edit.reverted">
      <button class="btn-compare" @click="emit('compare', edit)" title="比较差异">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="3" y="3" width="7" height="7"/>
          <rect x="14" y="3" width="7" height="7"/>
          <rect x="14" y="14" width="7" height="7"/>
          <rect x="3" y="14" width="7" height="7"/>
        </svg>
      </button>
      <button class="btn-revert" @click="handleRevert" :disabled="reverting" title="撤销修改">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/>
          <path d="M3 3v5h5"/>
        </svg>
      </button>
    </div>
  </div>
</template>

<style scoped>
.file-edit-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: var(--bg-elevated);
  border-radius: 6px;
  margin: 8px 0;
}

.file-edit-card.reverted {
  opacity: 0.6;
}

.edit-status {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--text-secondary);
}

.icon-check {
  color: var(--green);
}

.icon-reverted {
  color: var(--text-muted);
}

.edit-text {
  color: var(--text-secondary);
}

.edit-filename {
  display: flex;
  align-items: center;
  gap: 4px;
  color: var(--blue);
  font-weight: 500;
}

.edit-filename svg {
  color: var(--text-muted);
}

.edit-actions {
  display: flex;
  gap: 4px;
}

.edit-actions button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  background: transparent;
  border: none;
  border-radius: 4px;
  color: var(--text-muted);
  cursor: pointer;
  transition: all 0.15s;
}

.edit-actions button:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.edit-actions button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-revert:hover {
  color: var(--yellow);
}
</style>
