/**
 * 响应式数据管理模块导出
 */

// 其他现有的 composables
export { useOpenCode } from './useOpenCode.js'
export { useTheme } from './useTheme.js'
export { useFileEdits } from './useFileEdits.js'

/**
 * 响应式数据管理工具函数
 */
export const reactiveUtils = {
  /**
   * 深度监听对象变化
   */
  deepWatch(source, callback, options = {}) {
    return watch(source, callback, { deep: true, ...options })
  },
  
  /**
   * 防抖计算属性
   */
  debouncedComputed(getter, delay = 300) {
    const debouncedRef = ref()
    let timeoutId = null
    
    watch(getter, (newValue) => {
      if (timeoutId) {
        clearTimeout(timeoutId)
      }
      
      timeoutId = setTimeout(() => {
        debouncedRef.value = newValue
      }, delay)
    }, { immediate: true })
    
    return readonly(debouncedRef)
  },
  
  /**
   * 节流计算属性
   */
  throttledComputed(getter, delay = 300) {
    const throttledRef = ref()
    let lastExecution = 0
    
    watch(getter, (newValue) => {
      const now = Date.now()
      if (now - lastExecution >= delay) {
        throttledRef.value = newValue
        lastExecution = now
      }
    }, { immediate: true })
    
    return readonly(throttledRef)
  },
  
  /**
   * 异步计算属性
   */
  asyncComputed(asyncGetter, defaultValue = null) {
    const result = ref(defaultValue)
    const loading = ref(false)
    const error = ref(null)
    
    const execute = async () => {
      loading.value = true
      error.value = null
      
      try {
        result.value = await asyncGetter()
      } catch (err) {
        error.value = err
        console.error('Async computed error:', err)
      } finally {
        loading.value = false
      }
    }
    
    // 立即执行一次
    execute()
    
    return {
      result: readonly(result),
      loading: readonly(loading),
      error: readonly(error),
      refresh: execute
    }
  }
}