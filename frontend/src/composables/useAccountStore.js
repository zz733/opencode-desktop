import { computed, watch, onUnmounted } from 'vue'
import { createReactiveCollection } from './useReactiveStore.js'
import {
  GetKiroAccounts, AddKiroAccount, RemoveKiroAccount, UpdateKiroAccount,
  SwitchKiroAccount, GetActiveKiroAccount, RefreshKiroToken, GetKiroQuota,
  RefreshKiroQuota, BatchRefreshKiroTokens, BatchDeleteKiroAccounts,
  BatchAddKiroTags, ExportKiroAccounts, ImportKiroAccounts
} from '../../wailsjs/go/main/App'

/**
 * Kiro 账号存储管理
 * 基于响应式集合管理器的账号状态管理
 */
export function useAccountStore() {
  // 创建响应式集合
  const accountCollection = createReactiveCollection({
    keyField: 'id',
    initialItems: [],
    sortBy: 'lastUsed'
  })

  // 扩展状态
  accountCollection.updateState(state => {
    state.activeAccountId = null
    state.quotaRefreshInterval = 5 * 60 * 1000 // 5分钟
    state.autoRefreshEnabled = true
    state.lastQuotaRefresh = null
  })

  // 定时器管理
  let quotaRefreshTimer = null

  /**
   * 初始化事件监听器
   */
  function initializeEventListeners() {
    // 账号添加事件
    accountCollection.registerEventListener('kiro-account-added', (account) => {
      console.log('Account added:', account)
      accountCollection.addItems(account)
      
      // 如果是第一个账号，自动激活
      if (accountCollection.state.items.length === 1) {
        accountCollection.updateState(state => {
          state.activeAccountId = account.id
        })
      }
    })

    // 账号删除事件
    accountCollection.registerEventListener('kiro-account-removed', (accountId) => {
      console.log('Account removed:', accountId)
      accountCollection.removeItems(accountId)
      
      // 如果删除的是当前激活账号，清除激活状态
      if (accountCollection.state.activeAccountId === accountId) {
        accountCollection.updateState(state => {
          state.activeAccountId = null
        })
      }
    })

    // 账号切换事件
    accountCollection.registerEventListener('kiro-account-switched', (newAccountId, oldAccountId) => {
      console.log('Account switched:', newAccountId, oldAccountId)
      
      accountCollection.updateState(state => {
        state.activeAccountId = newAccountId
        
        // 更新账号的激活状态和最后使用时间
        state.items.forEach(account => {
          if (account.id === newAccountId) {
            account.isActive = true
            account.lastUsed = new Date().toISOString()
          } else if (account.id === oldAccountId) {
            account.isActive = false
          }
        })
      })
    })

    // 配额更新事件
    accountCollection.registerEventListener('kiro-quota-updated', (accountId, quota) => {
      console.log('Quota updated:', accountId, quota)
      
      accountCollection.updateItem(accountId, {
        quota,
        lastQuotaUpdate: new Date().toISOString()
      })
      
      accountCollection.updateState(state => {
        state.lastQuotaRefresh = Date.now()
      })
    })

    // 账号更新事件
    accountCollection.registerEventListener('kiro-account-updated', (accountId, updates) => {
      console.log('Account updated:', accountId, updates)
      accountCollection.updateItem(accountId, updates)
    })

    // Token 刷新事件
    accountCollection.registerEventListener('kiro-token-refreshed', (accountId, tokenInfo) => {
      console.log('Token refreshed:', accountId, tokenInfo)
      
      accountCollection.updateItem(accountId, {
        tokenExpiry: tokenInfo.expiresAt,
        lastTokenRefresh: new Date().toISOString()
      })
    })
  }

  /**
   * 加载账号列表
   */
  async function loadAccounts() {
    return accountCollection.executeOperation('load-accounts', async () => {
      const accounts = await GetKiroAccounts()
      
      accountCollection.updateState(state => {
        state.items = accounts || []
        
        // 找到当前激活的账号
        const activeAccount = state.items.find(acc => acc.isActive)
        state.activeAccountId = activeAccount?.id || null
      })
      
      console.log(`Loaded ${accounts?.length || 0} accounts`)
      return accounts
    })
  }

  /**
   * 添加账号
   * @param {string} loginMethod - 登录方式
   * @param {Object} data - 账号数据
   */
  async function addAccount(loginMethod, data) {
    return accountCollection.executeOperation('add-account', async () => {
      await AddKiroAccount(loginMethod, data)
      // 事件监听器会自动更新状态
      return true
    })
  }

  /**
   * 删除账号
   * @param {string} accountId - 账号ID
   */
  async function removeAccount(accountId) {
    return accountCollection.executeOperation(`remove-account-${accountId}`, async () => {
      await RemoveKiroAccount(accountId)
      // 事件监听器会自动更新状态
      return true
    })
  }

  /**
   * 更新账号
   * @param {string} accountId - 账号ID
   * @param {Object} updates - 更新数据
   */
  async function updateAccount(accountId, updates) {
    return accountCollection.executeOperation(`update-account-${accountId}`, async () => {
      await UpdateKiroAccount(accountId, updates)
      
      // 立即更新本地状态
      accountCollection.updateItem(accountId, updates)
      
      return true
    })
  }

  /**
   * 切换账号
   * @param {string} accountId - 账号ID
   */
  async function switchAccount(accountId) {
    if (accountCollection.state.activeAccountId === accountId) {
      return false // 已经是当前账号
    }
    
    return accountCollection.executeOperation(`switch-account-${accountId}`, async () => {
      await SwitchKiroAccount(accountId)
      // 事件监听器会自动更新状态
      return true
    })
  }

  /**
   * 刷新账号Token
   * @param {string} accountId - 账号ID
   */
  async function refreshAccountToken(accountId) {
    return accountCollection.executeOperation(`refresh-token-${accountId}`, async () => {
      await RefreshKiroToken(accountId)
      
      // 更新本地状态
      accountCollection.updateItem(accountId, {
        lastTokenRefresh: new Date().toISOString()
      })
      
      return true
    })
  }

  /**
   * 刷新账号配额
   * @param {string} accountId - 账号ID
   */
  async function refreshAccountQuota(accountId) {
    return accountCollection.executeOperation(`refresh-quota-${accountId}`, async () => {
      await RefreshKiroQuota(accountId)
      // 事件监听器会自动更新配额
      return true
    })
  }

  /**
   * 批量刷新Token
   * @param {Array} accountIds - 账号ID数组
   */
  async function batchRefreshTokens(accountIds) {
    return accountCollection.executeOperation('batch-refresh-tokens', async () => {
      await BatchRefreshKiroTokens(accountIds)
      
      // 更新本地状态
      const now = new Date().toISOString()
      accountIds.forEach(id => {
        accountCollection.updateItem(id, {
          lastTokenRefresh: now
        })
      })
      
      return true
    })
  }

  /**
   * 批量删除账号
   * @param {Array} accountIds - 账号ID数组
   */
  async function batchDeleteAccounts(accountIds) {
    return accountCollection.executeOperation('batch-delete-accounts', async () => {
      await BatchDeleteKiroAccounts(accountIds)
      // 事件监听器会自动更新状态
      return true
    })
  }

  /**
   * 批量添加标签
   * @param {Array} accountIds - 账号ID数组
   * @param {Array} tags - 标签数组
   */
  async function batchAddTags(accountIds, tags) {
    return accountCollection.executeOperation('batch-add-tags', async () => {
      await BatchAddKiroTags(accountIds, tags)
      
      // 立即更新本地状态
      accountIds.forEach(id => {
        const account = accountCollection.getItem(id)
        if (account) {
          const existingTags = new Set(account.tags || [])
          tags.forEach(tag => existingTags.add(tag))
          
          accountCollection.updateItem(id, {
            tags: Array.from(existingTags)
          })
        }
      })
      
      return true
    })
  }

  /**
   * 导出账号
   * @param {string} password - 加密密码
   */
  async function exportAccounts(password = '') {
    return accountCollection.executeOperation('export-accounts', async () => {
      const data = await ExportKiroAccounts(password)
      return data
    })
  }

  /**
   * 导入账号
   * @param {string} filePath - 文件路径
   * @param {string} password - 解密密码
   */
  async function importAccounts(filePath, password = '') {
    return accountCollection.executeOperation('import-accounts', async () => {
      await ImportKiroAccounts(filePath, password)
      // 重新加载账号列表
      await loadAccounts()
      return true
    })
  }

  /**
   * 启动配额自动刷新
   */
  function startQuotaAutoRefresh() {
    if (quotaRefreshTimer) {
      clearInterval(quotaRefreshTimer)
    }
    
    if (!accountCollection.state.autoRefreshEnabled) {
      return
    }
    
    quotaRefreshTimer = setInterval(async () => {
      const accounts = accountCollection.state.items
      if (accounts.length === 0) return
      
      try {
        // 刷新所有账号的配额
        const refreshPromises = accounts.map(account => 
          refreshAccountQuota(account.id).catch(error => {
            console.warn(`Failed to refresh quota for account ${account.id}:`, error)
            return null
          })
        )
        
        await Promise.allSettled(refreshPromises)
        console.log('Auto quota refresh completed')
      } catch (error) {
        console.error('Auto quota refresh failed:', error)
      }
    }, accountCollection.state.quotaRefreshInterval)
  }

  /**
   * 停止配额自动刷新
   */
  function stopQuotaAutoRefresh() {
    if (quotaRefreshTimer) {
      clearInterval(quotaRefreshTimer)
      quotaRefreshTimer = null
    }
  }

  /**
   * 设置自动刷新配置
   * @param {boolean} enabled - 是否启用
   * @param {number} interval - 刷新间隔（毫秒）
   */
  function setAutoRefreshConfig(enabled, interval) {
    accountCollection.updateState(state => {
      state.autoRefreshEnabled = enabled
      if (interval) {
        state.quotaRefreshInterval = interval
      }
    })
    
    if (enabled) {
      startQuotaAutoRefresh()
    } else {
      stopQuotaAutoRefresh()
    }
  }

  // 计算属性：当前激活账号
  const activeAccount = computed(() => {
    return accountCollection.state.items.find(
      acc => acc.id === accountCollection.state.activeAccountId
    ) || null
  })

  // 计算属性：有效账号数（Token未过期）
  const validAccountCount = computed(() => {
    const now = new Date()
    return accountCollection.state.items.filter(acc => {
      if (!acc.tokenExpiry) return true
      return new Date(acc.tokenExpiry) > now
    }).length
  })

  // 计算属性：所有标签
  const allTags = computed(() => {
    const tags = new Set()
    accountCollection.state.items.forEach(account => {
      if (account.tags) {
        account.tags.forEach(tag => tags.add(tag))
      }
    })
    return Array.from(tags).sort()
  })

  // 计算属性：订阅类型统计
  const subscriptionStats = computed(() => {
    const stats = { free: 0, pro: 0, pro_plus: 0 }
    accountCollection.state.items.forEach(account => {
      if (stats.hasOwnProperty(account.subscriptionType)) {
        stats[account.subscriptionType]++
      }
    })
    return stats
  })

  // 计算属性：配额警告
  const quotaAlerts = computed(() => {
    const alerts = []
    const threshold = 0.9 // 90%
    
    accountCollection.state.items.forEach(account => {
      if (!account.quota) return
      
      Object.entries(account.quota).forEach(([type, quota]) => {
        if (quota.total > 0) {
          const usage = quota.used / quota.total
          if (usage >= threshold) {
            alerts.push({
              accountId: account.id,
              accountName: account.displayName || account.email,
              quotaType: type,
              usage: Math.round(usage * 100),
              message: `${account.displayName || account.email} 的 ${type} 配额已用 ${quota.used} / ${quota.total}`
            })
          }
        }
      })
    })
    
    return alerts
  })

  // 监听器：自动启动配额刷新
  watch(() => accountCollection.state.items.length, (newLength, oldLength) => {
    if (newLength > 0 && oldLength === 0) {
      // 从无账号到有账号，启动自动刷新
      startQuotaAutoRefresh()
    } else if (newLength === 0 && oldLength > 0) {
      // 从有账号到无账号，停止自动刷新
      stopQuotaAutoRefresh()
    }
  })

  // 监听器：自动刷新配置变化
  watch(() => [
    accountCollection.state.autoRefreshEnabled,
    accountCollection.state.quotaRefreshInterval
  ], ([enabled, interval]) => {
    if (enabled) {
      startQuotaAutoRefresh()
    } else {
      stopQuotaAutoRefresh()
    }
  })

  // 清理函数
  function cleanup() {
    stopQuotaAutoRefresh()
    accountCollection.cleanupEventListeners()
  }

  // 组件卸载时清理
  onUnmounted(cleanup)

  // 初始化
  initializeEventListeners()

  return {
    // 继承集合管理功能
    ...accountCollection,
    
    // 账号特有的计算属性
    activeAccount,
    validAccountCount,
    allTags,
    subscriptionStats,
    quotaAlerts,
    
    // 账号操作方法
    loadAccounts,
    addAccount,
    removeAccount,
    updateAccount,
    switchAccount,
    refreshAccountToken,
    refreshAccountQuota,
    batchRefreshTokens,
    batchDeleteAccounts,
    batchAddTags,
    exportAccounts,
    importAccounts,
    
    // 配额管理
    startQuotaAutoRefresh,
    stopQuotaAutoRefresh,
    setAutoRefreshConfig,
    
    // 清理方法
    cleanup
  }
}

// 全局账号存储实例（单例模式）
let globalAccountStore = null

/**
 * 获取全局账号存储实例
 */
export function useGlobalAccountStore() {
  if (!globalAccountStore) {
    globalAccountStore = useAccountStore()
  }
  return globalAccountStore
}
