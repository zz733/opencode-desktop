import { computed, watch } from 'vue'
import { useGlobalAccountStore } from './useAccountStore.js'

/**
 * Kiro 账号管理 Composable
 * 基于新的响应式存储系统的兼容性包装器
 * @deprecated 建议直接使用 useGlobalAccountStore
 */
export function useKiroAccounts() {
  // 使用全局账号存储
  const accountStore = useGlobalAccountStore()

  // 兼容性映射：将新的存储接口映射到旧的接口
  const accountsState = computed(() => ({
    accounts: accountStore.state.items,
    activeAccountId: accountStore.state.activeAccountId,
    loading: accountStore.isLoading.value,
    error: accountStore.state.error,
    lastUpdated: accountStore.state.lastUpdated
  }))

  const operationState = computed(() => ({
    switching: accountStore.isOperationPending('switch-account'),
    refreshing: accountStore.hasPendingOperations.value,
    deleting: accountStore.isOperationPending('delete-account'),
    batchOperating: accountStore.isOperationPending('batch-operation')
  }))
  return {
    // 兼容性状态（只读）
    accountsState,
    operationState,
    
    // 新的计算属性（直接从存储获取）
    activeAccount: accountStore.activeAccount,
    accountCount: computed(() => accountStore.state.items.length),
    validAccountCount: accountStore.validAccountCount,
    allTags: accountStore.allTags,
    subscriptionStats: accountStore.subscriptionStats,
    quotaAlerts: accountStore.quotaAlerts,
    
    // 兼容性方法（映射到新的存储方法）
    loadAccounts: accountStore.loadAccounts,
    addAccount: accountStore.addAccount,
    removeAccount: accountStore.removeAccount,
    updateAccount: accountStore.updateAccount,
    switchAccount: accountStore.switchAccount,
    refreshAccountToken: accountStore.refreshAccountToken,
    refreshAccountQuota: accountStore.refreshAccountQuota,
    batchRefreshTokens: accountStore.batchRefreshTokens,
    batchDeleteAccounts: accountStore.batchDeleteAccounts,
    batchAddTags: accountStore.batchAddTags,
    exportAccounts: accountStore.exportAccounts,
    importAccounts: accountStore.importAccounts,
    clearError: accountStore.clearError,
    cleanupEventListeners: accountStore.cleanup,
    
    // 新增方法
    setAutoRefreshConfig: accountStore.setAutoRefreshConfig,
    startQuotaAutoRefresh: accountStore.startQuotaAutoRefresh,
    stopQuotaAutoRefresh: accountStore.stopQuotaAutoRefresh
  }
}