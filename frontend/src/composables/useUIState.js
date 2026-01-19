import { ref, reactive, computed, watch, nextTick } from 'vue'
import { createReactiveStore } from './useReactiveStore.js'

/**
 * UI 状态管理 Composable
 * 管理界面状态、对话框、表单等UI相关的响应式状态
 */
export function useUIState() {
  // 创建响应式存储
  const uiStore = createReactiveStore({
    // 对话框状态
    dialogs: {
      addAccount: false,
      editAccount: false,
      deleteAccount: false,
      batchOperations: false,
      exportAccounts: false,
      importAccounts: false,
      quotaDetails: false,
      settings: false
    },
    
    // 表单状态
    forms: {
      account: {
        id: '',
        displayName: '',
        notes: '',
        tags: [],
        loginMethod: 'token',
        provider: 'google',
        bearerToken: '',
        email: '',
        password: ''
      },
      batchOperation: {
        type: '',
        selectedIds: [],
        tags: '',
        password: ''
      },
      export: {
        password: '',
        selectedIds: []
      },
      import: {
        file: null,
        password: ''
      }
    },
    
    // 视图状态
    view: {
      layout: 'grid', // grid | list
      sidebarCollapsed: false,
      theme: 'auto', // light | dark | auto
      density: 'comfortable' // compact | comfortable | spacious
    },
    
    // 筛选和搜索状态
    filters: {
      searchQuery: '',
      tagFilter: '',
      subscriptionFilter: '',
      statusFilter: '',
      sortBy: 'lastUsed',
      sortDirection: 'desc'
    },
    
    // 选择状态
    selection: {
      selectedIds: [],
      selectMode: false,
      lastSelectedId: null
    },
    
    // 通知状态
    notifications: [],
    
    // 加载状态
    loadingStates: new Map(),
    
    // 临时状态
    temp: {
      editingAccount: null,
      deleteTarget: null,
      draggedItem: null,
      contextMenu: null
    }
  })

  /**
   * 对话框管理
   */
  const dialogManager = {
    /**
     * 打开对话框
     * @param {string} dialogName - 对话框名称
     * @param {Object} data - 初始数据
     */
    open(dialogName, data = {}) {
      uiStore.updateState(state => {
        state.dialogs[dialogName] = true
        
        // 根据对话框类型设置初始数据
        switch (dialogName) {
          case 'editAccount':
            if (data.account) {
              state.temp.editingAccount = data.account
              Object.assign(state.forms.account, {
                id: data.account.id,
                displayName: data.account.displayName || '',
                notes: data.account.notes || '',
                tags: [...(data.account.tags || [])]
              })
            }
            break
            
          case 'deleteAccount':
            state.temp.deleteTarget = data.account
            break
            
          case 'batchOperations':
            state.forms.batchOperation.selectedIds = [...data.selectedIds]
            break
            
          case 'exportAccounts':
            state.forms.export.selectedIds = data.selectedIds || []
            break
        }
      })
    },

    /**
     * 关闭对话框
     * @param {string} dialogName - 对话框名称
     */
    close(dialogName) {
      uiStore.updateState(state => {
        state.dialogs[dialogName] = false
        
        // 清理临时数据
        switch (dialogName) {
          case 'addAccount':
          case 'editAccount':
            this.resetAccountForm()
            state.temp.editingAccount = null
            break
            
          case 'deleteAccount':
            state.temp.deleteTarget = null
            break
            
          case 'batchOperations':
            this.resetBatchForm()
            break
            
          case 'exportAccounts':
            this.resetExportForm()
            break
            
          case 'importAccounts':
            this.resetImportForm()
            break
        }
      })
    },

    /**
     * 关闭所有对话框
     */
    closeAll() {
      uiStore.updateState(state => {
        Object.keys(state.dialogs).forEach(key => {
          state.dialogs[key] = false
        })
        
        // 清理所有表单
        this.resetAllForms()
      })
    },

    /**
     * 重置账号表单
     */
    resetAccountForm() {
      uiStore.updateState(state => {
        Object.assign(state.forms.account, {
          id: '',
          displayName: '',
          notes: '',
          tags: [],
          loginMethod: 'token',
          provider: 'google',
          bearerToken: '',
          email: '',
          password: ''
        })
      })
    },

    /**
     * 重置批量操作表单
     */
    resetBatchForm() {
      uiStore.updateState(state => {
        Object.assign(state.forms.batchOperation, {
          type: '',
          selectedIds: [],
          tags: '',
          password: ''
        })
      })
    },

    /**
     * 重置导出表单
     */
    resetExportForm() {
      uiStore.updateState(state => {
        Object.assign(state.forms.export, {
          password: '',
          selectedIds: []
        })
      })
    },

    /**
     * 重置导入表单
     */
    resetImportForm() {
      uiStore.updateState(state => {
        Object.assign(state.forms.import, {
          file: null,
          password: ''
        })
      })
    },

    /**
     * 重置所有表单
     */
    resetAllForms() {
      this.resetAccountForm()
      this.resetBatchForm()
      this.resetExportForm()
      this.resetImportForm()
    }
  }

  /**
   * 选择管理
   */
  const selectionManager = {
    /**
     * 选择项目
     * @param {string|Array} ids - 项目ID
     * @param {boolean} selected - 是否选中
     */
    select(ids, selected = true) {
      const idsArray = Array.isArray(ids) ? ids : [ids]
      
      uiStore.updateState(state => {
        if (selected) {
          idsArray.forEach(id => {
            if (!state.selection.selectedIds.includes(id)) {
              state.selection.selectedIds.push(id)
            }
          })
          state.selection.lastSelectedId = idsArray[idsArray.length - 1]
        } else {
          state.selection.selectedIds = state.selection.selectedIds.filter(
            id => !idsArray.includes(id)
          )
        }
      })
    },

    /**
     * 切换选择状态
     * @param {string} id - 项目ID
     */
    toggle(id) {
      const isSelected = uiStore.state.selection.selectedIds.includes(id)
      this.select(id, !isSelected)
    },

    /**
     * 全选/取消全选
     * @param {Array} allIds - 所有可选项目的ID
     */
    toggleAll(allIds) {
      const allSelected = allIds.every(id => 
        uiStore.state.selection.selectedIds.includes(id)
      )
      
      if (allSelected) {
        this.clear()
      } else {
        this.select(allIds, true)
      }
    },

    /**
     * 清除所有选择
     */
    clear() {
      uiStore.updateState(state => {
        state.selection.selectedIds = []
        state.selection.lastSelectedId = null
      })
    },

    /**
     * 范围选择
     * @param {string} startId - 起始ID
     * @param {string} endId - 结束ID
     * @param {Array} allIds - 所有项目ID（按顺序）
     */
    selectRange(startId, endId, allIds) {
      const startIndex = allIds.indexOf(startId)
      const endIndex = allIds.indexOf(endId)
      
      if (startIndex === -1 || endIndex === -1) return
      
      const minIndex = Math.min(startIndex, endIndex)
      const maxIndex = Math.max(startIndex, endIndex)
      const rangeIds = allIds.slice(minIndex, maxIndex + 1)
      
      this.select(rangeIds, true)
    },

    /**
     * 设置选择模式
     * @param {boolean} enabled - 是否启用选择模式
     */
    setSelectMode(enabled) {
      uiStore.updateState(state => {
        state.selection.selectMode = enabled
        if (!enabled) {
          state.selection.selectedIds = []
          state.selection.lastSelectedId = null
        }
      })
    }
  }

  /**
   * 筛选管理
   */
  const filterManager = {
    /**
     * 设置搜索查询
     * @param {string} query - 搜索查询
     */
    setSearch(query) {
      uiStore.updateState(state => {
        state.filters.searchQuery = query
      })
    },

    /**
     * 设置标签筛选
     * @param {string} tag - 标签
     */
    setTagFilter(tag) {
      uiStore.updateState(state => {
        state.filters.tagFilter = tag
      })
    },

    /**
     * 设置订阅类型筛选
     * @param {string} subscription - 订阅类型
     */
    setSubscriptionFilter(subscription) {
      uiStore.updateState(state => {
        state.filters.subscriptionFilter = subscription
      })
    },

    /**
     * 设置状态筛选
     * @param {string} status - 状态
     */
    setStatusFilter(status) {
      uiStore.updateState(state => {
        state.filters.statusFilter = status
      })
    },

    /**
     * 设置排序
     * @param {string} field - 排序字段
     * @param {string} direction - 排序方向
     */
    setSorting(field, direction = 'asc') {
      uiStore.updateState(state => {
        state.filters.sortBy = field
        state.filters.sortDirection = direction
      })
    },

    /**
     * 切换排序方向
     * @param {string} field - 排序字段
     */
    toggleSorting(field) {
      uiStore.updateState(state => {
        if (state.filters.sortBy === field) {
          state.filters.sortDirection = 
            state.filters.sortDirection === 'asc' ? 'desc' : 'asc'
        } else {
          state.filters.sortBy = field
          state.filters.sortDirection = 'asc'
        }
      })
    },

    /**
     * 重置所有筛选
     */
    reset() {
      uiStore.updateState(state => {
        Object.assign(state.filters, {
          searchQuery: '',
          tagFilter: '',
          subscriptionFilter: '',
          statusFilter: '',
          sortBy: 'lastUsed',
          sortDirection: 'desc'
        })
      })
    }
  }

  /**
   * 通知管理
   */
  const notificationManager = {
    /**
     * 添加通知
     * @param {Object} notification - 通知对象
     */
    add(notification) {
      const id = Date.now() + Math.random()
      const fullNotification = {
        id,
        type: 'info',
        title: '',
        message: '',
        duration: 5000,
        actions: [],
        ...notification,
        timestamp: Date.now()
      }
      
      uiStore.updateState(state => {
        state.notifications.push(fullNotification)
      })
      
      // 自动移除
      if (fullNotification.duration > 0) {
        setTimeout(() => {
          this.remove(id)
        }, fullNotification.duration)
      }
      
      return id
    },

    /**
     * 移除通知
     * @param {string|number} id - 通知ID
     */
    remove(id) {
      uiStore.updateState(state => {
        const index = state.notifications.findIndex(n => n.id === id)
        if (index >= 0) {
          state.notifications.splice(index, 1)
        }
      })
    },

    /**
     * 清除所有通知
     */
    clear() {
      uiStore.updateState(state => {
        state.notifications = []
      })
    },

    /**
     * 快捷方法：成功通知
     */
    success(message, options = {}) {
      return this.add({
        type: 'success',
        message,
        ...options
      })
    },

    /**
     * 快捷方法：错误通知
     */
    error(message, options = {}) {
      return this.add({
        type: 'error',
        message,
        duration: 0, // 错误通知不自动消失
        ...options
      })
    },

    /**
     * 快捷方法：警告通知
     */
    warning(message, options = {}) {
      return this.add({
        type: 'warning',
        message,
        ...options
      })
    },

    /**
     * 快捷方法：信息通知
     */
    info(message, options = {}) {
      return this.add({
        type: 'info',
        message,
        ...options
      })
    }
  }

  /**
   * 视图管理
   */
  const viewManager = {
    /**
     * 设置布局
     * @param {string} layout - 布局类型
     */
    setLayout(layout) {
      uiStore.updateState(state => {
        state.view.layout = layout
      })
    },

    /**
     * 切换侧边栏
     */
    toggleSidebar() {
      uiStore.updateState(state => {
        state.view.sidebarCollapsed = !state.view.sidebarCollapsed
      })
    },

    /**
     * 设置主题
     * @param {string} theme - 主题
     */
    setTheme(theme) {
      uiStore.updateState(state => {
        state.view.theme = theme
      })
    },

    /**
     * 设置密度
     * @param {string} density - 密度
     */
    setDensity(density) {
      uiStore.updateState(state => {
        state.view.density = density
      })
    }
  }

  /**
   * 加载状态管理
   */
  const loadingManager = {
    /**
     * 设置加载状态
     * @param {string} key - 加载键
     * @param {boolean} loading - 是否加载中
     */
    setLoading(key, loading) {
      uiStore.updateState(state => {
        if (loading) {
          state.loadingStates.set(key, true)
        } else {
          state.loadingStates.delete(key)
        }
      })
    },

    /**
     * 检查是否加载中
     * @param {string} key - 加载键
     */
    isLoading(key) {
      return uiStore.state.loadingStates.has(key)
    },

    /**
     * 清除所有加载状态
     */
    clearAll() {
      uiStore.updateState(state => {
        state.loadingStates.clear()
      })
    }
  }

  // 计算属性
  const hasSelectedItems = computed(() => 
    uiStore.state.selection.selectedIds.length > 0
  )

  const selectedCount = computed(() => 
    uiStore.state.selection.selectedIds.length
  )

  const hasActiveFilters = computed(() => {
    const filters = uiStore.state.filters
    return !!(
      filters.searchQuery ||
      filters.tagFilter ||
      filters.subscriptionFilter ||
      filters.statusFilter
    )
  })

  const hasNotifications = computed(() => 
    uiStore.state.notifications.length > 0
  )

  const isAnyDialogOpen = computed(() => 
    Object.values(uiStore.state.dialogs).some(open => open)
  )

  // 监听器：键盘快捷键
  function handleKeyboardShortcuts(event) {
    // Escape 键关闭对话框
    if (event.key === 'Escape' && isAnyDialogOpen.value) {
      const openDialog = Object.entries(uiStore.state.dialogs)
        .find(([_, open]) => open)?.[0]
      
      if (openDialog) {
        dialogManager.close(openDialog)
      }
    }
    
    // Ctrl/Cmd + A 全选
    if ((event.ctrlKey || event.metaKey) && event.key === 'a') {
      event.preventDefault()
      // 这里需要从外部传入所有项目ID
      // selectionManager.toggleAll(allItemIds)
    }
    
    // Delete 键删除选中项
    if (event.key === 'Delete' && hasSelectedItems.value) {
      // 触发删除操作
      // 这里需要从外部处理删除逻辑
    }
  }

  // 监听器：自动保存筛选状态到本地存储
  watch(() => uiStore.state.filters, (filters) => {
    try {
      localStorage.setItem('kiro-account-filters', JSON.stringify(filters))
    } catch (error) {
      console.warn('Failed to save filters to localStorage:', error)
    }
  }, { deep: true })

  // 监听器：自动保存视图状态到本地存储
  watch(() => uiStore.state.view, (view) => {
    try {
      localStorage.setItem('kiro-account-view', JSON.stringify(view))
    } catch (error) {
      console.warn('Failed to save view to localStorage:', error)
    }
  }, { deep: true })

  /**
   * 从本地存储恢复状态
   */
  function restoreFromLocalStorage() {
    try {
      // 恢复筛选状态
      const savedFilters = localStorage.getItem('kiro-account-filters')
      if (savedFilters) {
        const filters = JSON.parse(savedFilters)
        uiStore.updateState(state => {
          Object.assign(state.filters, filters)
        })
      }
      
      // 恢复视图状态
      const savedView = localStorage.getItem('kiro-account-view')
      if (savedView) {
        const view = JSON.parse(savedView)
        uiStore.updateState(state => {
          Object.assign(state.view, view)
        })
      }
    } catch (error) {
      console.warn('Failed to restore state from localStorage:', error)
    }
  }

  // 初始化
  restoreFromLocalStorage()

  return {
    // 基础存储功能
    ...uiStore,
    
    // 计算属性
    hasSelectedItems,
    selectedCount,
    hasActiveFilters,
    hasNotifications,
    isAnyDialogOpen,
    
    // 管理器
    dialogs: dialogManager,
    selection: selectionManager,
    filters: filterManager,
    notifications: notificationManager,
    view: viewManager,
    loading: loadingManager,
    
    // 工具方法
    handleKeyboardShortcuts,
    restoreFromLocalStorage
  }
}