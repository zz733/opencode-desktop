<script setup>
import { ref, computed } from 'vue'
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
  if (m) return m.name
  // 未在列表中也显示当前ID，避免切目录时模型名丢失造成误解
  return props.modelValue || t('model.select')
}

// 完全按照 provider 对模型进行分组，与官方保持一致
const providerGroups = computed(() => {
  const groups = {}

  props.models.forEach(m => {
    const pName = m.category || m.provider || (m.id && m.id.includes('/') ? m.id.split('/')[0] : t('model.select'))
    if (!groups[pName]) {
      groups[pName] = []
    }
    groups[pName].push(m)
  })
  
  return groups
})

const select = (id) => {
  emit('update:modelValue', id)
  show.value = false
}
</script>

<template>
  <div class="model-selector">
    <div v-if="show" class="backdrop" @click="show = false"></div>
    
    <button class="trigger" @click="show = !show">
      <span>{{ currentModelName() }}</span>
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M6 9l6 6 6-6"/>
      </svg>
    </button>
    
    <div v-if="show" class="dropdown" @click.stop>
      <div class="header">{{ t('model.select') }}</div>
      
      <div v-if="Object.keys(providerGroups).length === 0" class="group">
        <div class="option" style="color: var(--text-muted); cursor: default; pointer-events: none;">
          没有配置任何可用的模型
        </div>
      </div>
      
      <!-- 动态按提供商分组，与官方对齐 -->
      <div v-for="(models, providerName) in providerGroups" :key="providerName" class="group">
        <div class="group-label">{{ providerName }}</div>
        <div 
          v-for="m in models" 
          :key="m.id"
          :class="['option', { active: modelValue === m.id }]"
          @click="select(m.id)"
        >
          <span class="model-name">{{ m.name }}</span>
          <span v-if="m.free" class="free-badge">免费</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.model-selector {
  position: relative;
}

.trigger {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  background: var(--bg-elevated);
  border: none;
  border-radius: 4px;
  color: var(--text-secondary);
  font-size: 11px;
  cursor: pointer;
}

.trigger:hover {
  color: var(--text-primary);
}

.dropdown {
  position: absolute;
  bottom: 100%;
  right: 0;
  margin-bottom: 4px;
  width: 240px;
  max-height: 320px;
  overflow-y: auto;
  background: var(--bg-elevated);
  border: 1px solid var(--border-subtle);
  border-radius: 8px;
  box-shadow: 0 4px 16px rgba(0,0,0,0.4);
  z-index: 1000;
}

.header {
  padding: 8px 12px;
  font-size: 10px;
  font-weight: 500;
  color: var(--text-muted);
  border-bottom: 1px solid var(--border-default);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.group {
  padding: 4px;
}

.group-label {
  padding: 12px 10px 6px 10px;
  font-size: 14px;
  font-weight: bold;
  color: var(--accent-primary);
  border-bottom: 1px solid var(--border-subtle);
  margin-bottom: 4px;
}

.option {
  padding: 10px 12px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 2px;
}

.model-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.free-badge {
  font-size: 10px;
  font-weight: 600;
  padding: 2px 6px;
  background-color: rgba(128, 255, 181, 0.15);
  color: var(--green, #4ade80);
  border: 1px solid rgba(128, 255, 181, 0.3);
  border-radius: 4px;
  margin-left: 8px;
  flex-shrink: 0;
}

.option:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.option:hover .free-badge {
  background-color: var(--bg-elevated);
}

.option.active {
  background: var(--accent-button);
  color: white;
}

.option.active .free-badge {
  background-color: rgba(255, 255, 255, 0.2);
  color: white;
  border-color: transparent;
}

.backdrop {
  position: fixed;
  inset: 0;
  z-index: 999;
}
</style>
