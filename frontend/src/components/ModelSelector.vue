<script setup>
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps({
  modelValue: String,
  models: Array
})

const emit = defineEmits(['update:modelValue'])
const show = ref(false)

const currentModelName = () => {
  const m = props.models.find(m => m.id === props.modelValue)
  return m ? m.name : t('model.select')
}

const select = (id) => {
  emit('update:modelValue', id)
  show.value = false
}
</script>

<template>
  <div class="model-selector">
    <button class="trigger" @click="show = !show">
      <span>{{ currentModelName() }}</span>
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M6 9l6 6 6-6"/>
      </svg>
    </button>
    
    <div v-if="show" class="dropdown">
      <div class="header">{{ t('model.select') }}</div>
      
      <div class="group">
        <div class="group-label">🆓 {{ t('model.freeModels') }}</div>
        <div 
          v-for="m in models.filter(m => m.free)" 
          :key="m.id"
          :class="['option', { active: modelValue === m.id }]"
          @click="select(m.id)"
        >
          {{ m.name }}
        </div>
      </div>
      
      <div class="group">
        <div class="group-label">⭐ {{ t('model.premiumModels') }}</div>
        <div 
          v-for="m in models.filter(m => !m.free)" 
          :key="m.id"
          :class="['option', { active: modelValue === m.id }]"
          @click="select(m.id)"
        >
          {{ m.name }}
        </div>
      </div>
    </div>
    
    <div v-if="show" class="backdrop" @click="show = false"></div>
  </div>
</template>

<style scoped>
.model-selector {
  position: relative;
}

.trigger {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  background: var(--bg-input);
  border: 1px solid var(--border-default);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 12px;
  cursor: pointer;
}

.trigger:hover {
  background: var(--bg-hover);
  border-color: var(--text-muted);
}

.dropdown {
  position: absolute;
  bottom: 100%;
  right: 0;
  margin-bottom: 4px;
  width: 220px;
  max-height: 300px;
  overflow-y: auto;
  background: var(--bg-elevated);
  border: 1px solid var(--border-default);
  border-radius: 8px;
  box-shadow: 0 -8px 24px rgba(0,0,0,0.3);
  z-index: 1000;
}

.header {
  padding: 10px 12px;
  font-size: 11px;
  color: var(--text-secondary);
  border-bottom: 1px solid var(--border-default);
}

.group {
  padding: 6px;
}

.group-label {
  padding: 6px 8px;
  font-size: 11px;
  color: var(--text-muted);
}

.option {
  padding: 8px 10px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  color: var(--text-primary);
}

.option:hover {
  background: var(--bg-hover);
}

.option.active {
  background: var(--accent-primary);
  color: white;
}

.backdrop {
  position: fixed;
  inset: 0;
  z-index: 999;
}
</style>
