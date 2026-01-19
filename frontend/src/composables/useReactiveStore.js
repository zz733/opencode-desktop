import { ref, reactive, computed, readonly, watch, nextTick } from 'vue'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

/**
 * 响应式存储基类
 * 提供通用的响应式状态管理模式
 */
export function createReactiveStore(initialState = {}) {
  // 内部状态
  const state = reactive({
    ...initialState,
    loading: false,
    error: null,
    lastUpdated: null
  })

  // 操作状态
  const operations = reactive({
    pending: new Set(),
    results: new Map()
  })

  // 事件监听器管理
  const eventListeners = new Map()

  /**
   * 注册事件监听器
   * @param {string} eventName - 事件名称
   * @param {Function} handler - 事件处理函数
   */
  function registerEventListener(eventName, handler) {
    if (eventListeners.has(eventName)) {
      EventsOff(eventName)
    }
    
    EventsOn(eventName, handler)
    eventListeners.set(eventName, handler)
  }

  /**
   * 清理所有事件监听器
   */
  function cleanupEventListeners() {
    for (const [eventName] of eventListeners) {
      EventsOff(eventName)
    }
    eventListeners.clear()
  }

  /**
   * 执行异步操作
   * @param {string} operationId - 操作标识
   * @param {Function} operation - 异步操作函数
   * @param {Object} options - 选项
   */
  async function executeOperation(operationId, operation, options = {}) {
    const { 
      loadingState = true, 
      errorHandling = true,
      updateTimestamp = true 
    } = options

    // 防止重复操作
    if (operations.pending.has(operationId)) {
      return operations.results.get(operationId)
    }

    operations.pending.add(operationId)
    
    if (loadingState) {
      state.loading = true
    }
    
    if (errorHandling) {
      state.error = null
    }

    try {
      const result = await operation()
      
      operations.results.set(operationId, result)
      
      if (updateTimestamp) {
        state.lastUpdated = Date.now()
      }
      
      return result
    } catch (error) {
      console.error(`Operation ${operationId} failed:`, error)
      
      if (errorHandling) {
        state.error = error.message || `Operation ${operationId} failed`
      }
      
      operations.results.set(operationId, { error })
      throw error
    } finally {
      operations.pending.delete(operationId)
      
      if (loadingState) {
        state.loading = false
      }
    }
  }

  /**
   * 批量执行操作
   * @param {Array} operationConfigs - 操作配置数组
   */
  async function executeBatchOperations(operationConfigs) {
    const batchId = `batch_${Date.now()}`
    
    return executeOperation(batchId, async () => {
      const results = await Promise.allSettled(
        operationConfigs.map(config => 
          executeOperation(config.id, config.operation, {
            loadingState: false,
            errorHandling: false,
            ...config.options
          })
        )
      )
      
      const successful = results.filter(r => r.status === 'fulfilled')
      const failed = results.filter(r => r.status === 'rejected')
      
      return {
        successful: successful.length,
        failed: failed.length,
        total: results.length,
        results,
        errors: failed.map(r => r.reason)
      }
    })
  }

  /**
   * 更新状态
   * @param {Object|Function} updates - 更新对象或更新函数
   */
  function updateState(updates) {
    if (typeof updates === 'function') {
      updates(state)
    } else {
      Object.assign(state, updates)
    }
    
    state.lastUpdated = Date.now()
  }

  /**
   * 重置状态
   */
  function resetState() {
    Object.assign(state, initialState, {
      loading: false,
      error: null,
      lastUpdated: null
    })
    
    operations.pending.clear()
    operations.results.clear()
  }

  /**
   * 清除错误状态
   */
  function clearError() {
    state.error = null
  }

  /**
   * 检查操作是否正在进行
   * @param {string} operationId - 操作标识
   */
  function isOperationPending(operationId) {
    return operations.pending.has(operationId)
  }

  /**
   * 获取操作结果
   * @param {string} operationId - 操作标识
   */
  function getOperationResult(operationId) {
    return operations.results.get(operationId)
  }

  // 计算属性
  const isLoading = computed(() => state.loading)
  const hasError = computed(() => !!state.error)
  const hasPendingOperations = computed(() => operations.pending.size > 0)
  const lastUpdateTime = computed(() => 
    state.lastUpdated ? new Date(state.lastUpdated) : null
  )

  // 监听器：自动清理过期的操作结果
  watch(() => operations.results.size, (newSize) => {
    if (newSize > 100) { // 限制缓存大小
      const entries = Array.from(operations.results.entries())
      const toDelete = entries.slice(0, newSize - 50)
      toDelete.forEach(([key]) => operations.results.delete(key))
    }
  })

  return {
    // 状态
    state: readonly(state),
    operations: readonly(operations),
    
    // 计算属性
    isLoading,
    hasError,
    hasPendingOperations,
    lastUpdateTime,
    
    // 方法
    registerEventListener,
    cleanupEventListeners,
    executeOperation,
    executeBatchOperations,
    updateState,
    resetState,
    clearError,
    isOperationPending,
    getOperationResult
  }
}

/**
 * 创建响应式集合管理器
 * @param {Object} options - 配置选项
 */
export function createReactiveCollection(options = {}) {
  const {
    keyField = 'id',
    initialItems = [],
    sortBy = null,
    filterBy = null
  } = options

  const store = createReactiveStore({
    items: [...initialItems],
    selectedItems: [],
    searchQuery: '',
    sortField: sortBy,
    filterField: filterBy,
    filterValue: ''
  })

  /**
   * 添加项目
   * @param {Object|Array} items - 要添加的项目
   */
  function addItems(items) {
    const itemsArray = Array.isArray(items) ? items : [items]
    
    store.updateState(state => {
      itemsArray.forEach(item => {
        const existingIndex = state.items.findIndex(
          existing => existing[keyField] === item[keyField]
        )
        
        if (existingIndex >= 0) {
          // 更新现有项目
          state.items[existingIndex] = { ...state.items[existingIndex], ...item }
        } else {
          // 添加新项目
          state.items.push(item)
        }
      })
    })
  }

  /**
   * 移除项目
   * @param {string|Array} ids - 要移除的项目ID
   */
  function removeItems(ids) {
    const idsArray = Array.isArray(ids) ? ids : [ids]
    
    store.updateState(state => {
      state.items = state.items.filter(
        item => !idsArray.includes(item[keyField])
      )
      
      // 清理选中状态
      state.selectedItems = state.selectedItems.filter(
        id => !idsArray.includes(id)
      )
    })
  }

  /**
   * 更新项目
   * @param {string} id - 项目ID
   * @param {Object} updates - 更新数据
   */
  function updateItem(id, updates) {
    store.updateState(state => {
      const index = state.items.findIndex(item => item[keyField] === id)
      if (index >= 0) {
        state.items[index] = { ...state.items[index], ...updates }
      }
    })
  }

  /**
   * 获取项目
   * @param {string} id - 项目ID
   */
  function getItem(id) {
    return store.state.items.find(item => item[keyField] === id)
  }

  /**
   * 选择/取消选择项目
   * @param {string|Array} ids - 项目ID
   * @param {boolean} selected - 是否选中
   */
  function selectItems(ids, selected = true) {
    const idsArray = Array.isArray(ids) ? ids : [ids]
    
    store.updateState(state => {
      if (selected) {
        idsArray.forEach(id => {
          if (!state.selectedItems.includes(id)) {
            state.selectedItems.push(id)
          }
        })
      } else {
        state.selectedItems = state.selectedItems.filter(
          id => !idsArray.includes(id)
        )
      }
    })
  }

  /**
   * 全选/取消全选
   */
  function toggleSelectAll() {
    store.updateState(state => {
      const allIds = filteredItems.value.map(item => item[keyField])
      const allSelected = allIds.every(id => state.selectedItems.includes(id))
      
      if (allSelected) {
        state.selectedItems = state.selectedItems.filter(
          id => !allIds.includes(id)
        )
      } else {
        allIds.forEach(id => {
          if (!state.selectedItems.includes(id)) {
            state.selectedItems.push(id)
          }
        })
      }
    })
  }

  /**
   * 设置搜索查询
   * @param {string} query - 搜索查询
   */
  function setSearchQuery(query) {
    store.updateState(state => {
      state.searchQuery = query
    })
  }

  /**
   * 设置排序
   * @param {string} field - 排序字段
   * @param {string} direction - 排序方向 (asc/desc)
   */
  function setSorting(field, direction = 'asc') {
    store.updateState(state => {
      state.sortField = field
      state.sortDirection = direction
    })
  }

  /**
   * 设置筛选
   * @param {string} field - 筛选字段
   * @param {any} value - 筛选值
   */
  function setFilter(field, value) {
    store.updateState(state => {
      state.filterField = field
      state.filterValue = value
    })
  }

  // 计算属性：筛选和排序后的项目
  const filteredItems = computed(() => {
    let items = [...store.state.items]
    
    // 搜索筛选
    if (store.state.searchQuery) {
      const query = store.state.searchQuery.toLowerCase()
      items = items.filter(item => {
        return Object.values(item).some(value => 
          String(value).toLowerCase().includes(query)
        )
      })
    }
    
    // 字段筛选
    if (store.state.filterField && store.state.filterValue) {
      items = items.filter(item => {
        const fieldValue = item[store.state.filterField]
        if (Array.isArray(fieldValue)) {
          return fieldValue.includes(store.state.filterValue)
        }
        return fieldValue === store.state.filterValue
      })
    }
    
    // 排序
    if (store.state.sortField) {
      items.sort((a, b) => {
        const aValue = a[store.state.sortField]
        const bValue = b[store.state.sortField]
        
        let comparison = 0
        if (aValue < bValue) comparison = -1
        if (aValue > bValue) comparison = 1
        
        return store.state.sortDirection === 'desc' ? -comparison : comparison
      })
    }
    
    return items
  })

  // 计算属性：选中的项目
  const selectedItemsData = computed(() => {
    return store.state.selectedItems.map(id => getItem(id)).filter(Boolean)
  })

  // 计算属性：统计信息
  const stats = computed(() => ({
    total: store.state.items.length,
    filtered: filteredItems.value.length,
    selected: store.state.selectedItems.length
  }))

  return {
    // 继承基础存储功能
    ...store,
    
    // 集合特有的计算属性
    filteredItems,
    selectedItemsData,
    stats,
    
    // 集合操作方法
    addItems,
    removeItems,
    updateItem,
    getItem,
    selectItems,
    toggleSelectAll,
    setSearchQuery,
    setSorting,
    setFilter
  }
}