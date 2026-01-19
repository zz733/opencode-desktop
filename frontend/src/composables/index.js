/**
 * 响应式数据管理模块导出
 * 
 * 这个文件提供了 Kiro 多账号管理器的所有响应式数据管理功能
 * 包括状态管理、表单验证、UI状态等
 */

// 核心响应式存储
export { 
  createReactiveStore, 
  createReactiveCollection 
} from './useReactiveStore.js'

// 账号数据管理
export { 
  useAccountStore, 
  useGlobalAccountStore 
} from './useAccountStore.js'

// 认证管理
export { useKiroAuth } from './useKiroAuth.js'

// UI状态管理
export { useUIState } from './useUIState.js'

// 表单验证
export { 
  useFormValidation,
  accountFormSchema,
  batchOperationSchema,
  exportFormSchema,
  importFormSchema
} from './useFormValidation.js'

// 兼容性包装器
export { useKiroAccounts } from './useKiroAccounts.js'

// 其他现有的 composables
export { useOpenCode } from './useOpenCode.js'
export { useTheme } from './useTheme.js'
export { useFileEdits } from './useFileEdits.js'

/**
 * 创建完整的账号管理器实例
 * 包含所有必要的响应式状态和方法
 */
export function createKiroAccountManager() {
  // 获取全局账号存储
  const accountStore = useGlobalAccountStore()
  
  // 创建UI状态管理
  const uiState = useUIState()
  
  // 创建表单验证
  const accountFormValidation = useFormValidation(accountFormSchema)
  const batchFormValidation = useFormValidation(batchOperationSchema)
  const exportFormValidation = useFormValidation(exportFormSchema)
  const importFormValidation = useFormValidation(importFormSchema)
  
  /**
   * 初始化管理器
   */
  async function initialize() {
    try {
      // 加载账号数据
      await accountStore.loadAccounts()
      
      // 启动配额自动刷新
      accountStore.startQuotaAutoRefresh()
      
      uiState.notifications.success('账号管理器初始化成功')
    } catch (error) {
      console.error('Failed to initialize account manager:', error)
      uiState.notifications.error('账号管理器初始化失败: ' + error.message)
    }
  }
  
  /**
   * 清理资源
   */
  function cleanup() {
    accountStore.cleanup()
    // UI状态会在组件卸载时自动清理
  }
  
  /**
   * 添加账号的完整流程
   */
  async function addAccountWithValidation(formData) {
    // 验证表单
    const isValid = await accountFormValidation.validateAll(formData)
    if (!isValid) {
      uiState.notifications.error('请检查表单输入')
      return false
    }
    
    try {
      uiState.loading.setLoading('add-account', true)
      
      await accountStore.addAccount(formData.loginMethod, {
        displayName: formData.displayName,
        notes: formData.notes,
        tags: formData.tags,
        token: formData.bearerToken,
        email: formData.email,
        password: formData.password
      })
      
      uiState.notifications.success('账号添加成功')
      uiState.dialogs.close('addAccount')
      accountFormValidation.resetValidation()
      
      return true
    } catch (error) {
      console.error('Failed to add account:', error)
      uiState.notifications.error('添加账号失败: ' + error.message)
      return false
    } finally {
      uiState.loading.setLoading('add-account', false)
    }
  }
  
  /**
   * 更新账号的完整流程
   */
  async function updateAccountWithValidation(accountId, formData) {
    // 验证表单（只验证可编辑字段）
    const editableSchema = {
      displayName: accountFormSchema.displayName,
      notes: accountFormSchema.notes,
      tags: accountFormSchema.tags
    }
    
    const editFormValidation = useFormValidation(editableSchema)
    const isValid = await editFormValidation.validateAll(formData)
    
    if (!isValid) {
      uiState.notifications.error('请检查表单输入')
      return false
    }
    
    try {
      uiState.loading.setLoading('update-account', true)
      
      await accountStore.updateAccount(accountId, {
        displayName: formData.displayName,
        notes: formData.notes,
        tags: formData.tags
      })
      
      uiState.notifications.success('账号更新成功')
      uiState.dialogs.close('editAccount')
      
      return true
    } catch (error) {
      console.error('Failed to update account:', error)
      uiState.notifications.error('更新账号失败: ' + error.message)
      return false
    } finally {
      uiState.loading.setLoading('update-account', false)
    }
  }
  
  /**
   * 批量操作的完整流程
   */
  async function executeBatchOperationWithValidation(operationType, formData) {
    // 验证表单
    const isValid = await batchFormValidation.validateAll(formData)
    if (!isValid) {
      uiState.notifications.error('请检查操作参数')
      return false
    }
    
    try {
      uiState.loading.setLoading('batch-operation', true)
      
      let result
      switch (operationType) {
        case 'refreshTokens':
          result = await accountStore.batchRefreshTokens(formData.selectedIds)
          break
        case 'deleteAccounts':
          result = await accountStore.batchDeleteAccounts(formData.selectedIds)
          break
        case 'addTags':
          const tags = formData.tags.split(',').map(tag => tag.trim()).filter(Boolean)
          result = await accountStore.batchAddTags(formData.selectedIds, tags)
          break
        default:
          throw new Error('未知的批量操作类型')
      }
      
      if (result) {
        uiState.notifications.success('批量操作执行成功')
        uiState.dialogs.close('batchOperations')
        uiState.selection.clear()
        batchFormValidation.resetValidation()
      }
      
      return result
    } catch (error) {
      console.error('Batch operation failed:', error)
      uiState.notifications.error('批量操作失败: ' + error.message)
      return false
    } finally {
      uiState.loading.setLoading('batch-operation', false)
    }
  }
  
  return {
    // 存储实例
    accountStore,
    uiState,
    
    // 表单验证
    accountFormValidation,
    batchFormValidation,
    exportFormValidation,
    importFormValidation,
    
    // 生命周期方法
    initialize,
    cleanup,
    
    // 业务方法
    addAccountWithValidation,
    updateAccountWithValidation,
    executeBatchOperationWithValidation
  }
}

/**
 * 全局账号管理器实例（单例）
 */
let globalAccountManager = null

/**
 * 获取全局账号管理器实例
 */
export function useGlobalAccountManager() {
  if (!globalAccountManager) {
    globalAccountManager = createKiroAccountManager()
  }
  return globalAccountManager
}

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