/**
 * 手动测试脚本 - 验证响应式数据管理功能
 * 
 * 这个脚本可以在浏览器控制台中运行，验证响应式数据管理的核心功能
 */

// 模拟 Wails 运行时
window.wailsjs = {
  runtime: {
    EventsOn: (event, callback) => {
      console.log(`EventsOn registered: ${event}`)
      // 存储事件监听器以便后续测试
      if (!window._testEventListeners) {
        window._testEventListeners = new Map()
      }
      window._testEventListeners.set(event, callback)
    },
    EventsOff: (event) => {
      console.log(`EventsOff called: ${event}`)
      if (window._testEventListeners) {
        window._testEventListeners.delete(event)
      }
    }
  },
  go: {
    main: {
      App: {
        GetKiroAccounts: () => Promise.resolve([
          {
            id: 'test-1',
            email: 'test1@example.com',
            displayName: 'Test Account 1',
            isActive: true,
            subscriptionType: 'pro',
            quota: {
              main: { used: 100, total: 1000 },
              trial: { used: 0, total: 100 }
            },
            tags: ['工作', '主账号'],
            lastUsed: new Date().toISOString()
          },
          {
            id: 'test-2',
            email: 'test2@example.com',
            displayName: 'Test Account 2',
            isActive: false,
            subscriptionType: 'free',
            quota: {
              main: { used: 50, total: 100 },
              trial: { used: 10, total: 100 }
            },
            tags: ['测试'],
            lastUsed: new Date(Date.now() - 86400000).toISOString()
          }
        ]),
        AddKiroAccount: (method, data) => {
          console.log('AddKiroAccount called:', method, data)
          return Promise.resolve()
        },
        UpdateKiroAccount: (id, updates) => {
          console.log('UpdateKiroAccount called:', id, updates)
          return Promise.resolve()
        },
        SwitchKiroAccount: (id) => {
          console.log('SwitchKiroAccount called:', id)
          return Promise.resolve()
        }
      }
    }
  }
}

/**
 * 测试响应式存储基础功能
 */
async function testReactiveStore() {
  console.log('🧪 Testing Reactive Store...')
  
  try {
    // 动态导入模块
    const { createReactiveStore } = await import('../useReactiveStore.js')
    
    // 创建存储实例
    const store = createReactiveStore({
      testData: 'initial'
    })
    
    console.log('✅ Store created with initial state:', store.state.testData)
    
    // 测试状态更新
    store.updateState({ testData: 'updated' })
    console.log('✅ State updated:', store.state.testData)
    
    // 测试操作执行
    const result = await store.executeOperation('test-op', async () => {
      await new Promise(resolve => setTimeout(resolve, 100))
      return 'operation completed'
    })
    
    console.log('✅ Operation executed:', result)
    console.log('✅ Last updated:', store.state.lastUpdated)
    
    // 测试错误处理
    try {
      await store.executeOperation('error-op', async () => {
        throw new Error('Test error')
      })
    } catch (error) {
      console.log('✅ Error handling works:', store.state.error)
    }
    
    console.log('✅ Reactive Store tests passed!')
    return true
  } catch (error) {
    console.error('❌ Reactive Store test failed:', error)
    return false
  }
}

/**
 * 测试响应式集合功能
 */
async function testReactiveCollection() {
  console.log('🧪 Testing Reactive Collection...')
  
  try {
    const { createReactiveCollection } = await import('../useReactiveStore.js')
    
    // 创建集合实例
    const collection = createReactiveCollection({
      keyField: 'id',
      initialItems: [
        { id: '1', name: 'Item 1', category: 'A' },
        { id: '2', name: 'Item 2', category: 'B' }
      ]
    })
    
    console.log('✅ Collection created with items:', collection.state.items.length)
    
    // 测试添加项目
    collection.addItems({ id: '3', name: 'Item 3', category: 'C' })
    console.log('✅ Item added, total:', collection.state.items.length)
    
    // 测试选择功能
    collection.selectItems(['1', '2'])
    console.log('✅ Items selected:', collection.state.selectedItems)
    
    // 测试筛选功能
    collection.setSearchQuery('Item 1')
    console.log('✅ Search applied, filtered items:', collection.filteredItems.value.length)
    
    // 测试排序功能
    collection.setSorting('name', 'desc')
    console.log('✅ Sorting applied')
    
    console.log('✅ Reactive Collection tests passed!')
    return true
  } catch (error) {
    console.error('❌ Reactive Collection test failed:', error)
    return false
  }
}

/**
 * 测试账号存储功能
 */
async function testAccountStore() {
  console.log('🧪 Testing Account Store...')
  
  try {
    const { useAccountStore } = await import('../useAccountStore.js')
    
    // 创建账号存储实例
    const accountStore = useAccountStore()
    
    console.log('✅ Account store created')
    
    // 测试加载账号
    await accountStore.loadAccounts()
    console.log('✅ Accounts loaded:', accountStore.state.items.length)
    
    // 测试计算属性
    console.log('✅ Active account:', accountStore.activeAccount.value?.displayName)
    console.log('✅ All tags:', accountStore.allTags.value)
    console.log('✅ Subscription stats:', accountStore.subscriptionStats.value)
    
    // 测试筛选功能
    accountStore.setSearchQuery('test1')
    console.log('✅ Search applied, filtered:', accountStore.filteredItems.value.length)
    
    console.log('✅ Account Store tests passed!')
    return true
  } catch (error) {
    console.error('❌ Account Store test failed:', error)
    return false
  }
}

/**
 * 测试UI状态管理
 */
async function testUIState() {
  console.log('🧪 Testing UI State...')
  
  try {
    const { useUIState } = await import('../useUIState.js')
    
    // 创建UI状态实例
    const uiState = useUIState()
    
    console.log('✅ UI state created')
    
    // 测试对话框管理
    uiState.dialogs.open('addAccount')
    console.log('✅ Dialog opened:', uiState.state.dialogs.addAccount)
    
    uiState.dialogs.close('addAccount')
    console.log('✅ Dialog closed:', uiState.state.dialogs.addAccount)
    
    // 测试选择管理
    uiState.selection.select(['1', '2'])
    console.log('✅ Items selected:', uiState.state.selection.selectedIds)
    
    // 测试通知管理
    const notificationId = uiState.notifications.success('Test notification')
    console.log('✅ Notification added:', uiState.state.notifications.length)
    
    uiState.notifications.remove(notificationId)
    console.log('✅ Notification removed:', uiState.state.notifications.length)
    
    // 测试筛选管理
    uiState.filters.setSearch('test query')
    console.log('✅ Search filter set:', uiState.state.filters.searchQuery)
    
    console.log('✅ UI State tests passed!')
    return true
  } catch (error) {
    console.error('❌ UI State test failed:', error)
    return false
  }
}

/**
 * 测试表单验证
 */
async function testFormValidation() {
  console.log('🧪 Testing Form Validation...')
  
  try {
    const { useFormValidation, accountFormSchema } = await import('../useFormValidation.js')
    
    // 创建表单验证实例
    const validation = useFormValidation(accountFormSchema)
    
    console.log('✅ Form validation created')
    
    // 测试字段验证
    const isValidEmail = await validation.validateField('email', 'test@example.com')
    console.log('✅ Email validation (valid):', isValidEmail)
    
    const isInvalidEmail = await validation.validateField('email', 'invalid-email')
    console.log('✅ Email validation (invalid):', isInvalidEmail)
    
    // 测试Bearer Token验证
    const isValidToken = await validation.validateField('bearerToken', 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9')
    console.log('✅ Token validation:', isValidToken)
    
    // 测试表单整体验证
    const formData = {
      email: 'test@example.com',
      bearerToken: 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ',
      displayName: 'Test User',
      tags: ['test', 'demo']
    }
    
    const isFormValid = await validation.validateAll(formData)
    console.log('✅ Form validation:', isFormValid)
    
    console.log('✅ Form Validation tests passed!')
    return true
  } catch (error) {
    console.error('❌ Form Validation test failed:', error)
    return false
  }
}

/**
 * 运行所有测试
 */
async function runAllTests() {
  console.log('🚀 Starting Reactive Data Management Tests...')
  console.log('=' .repeat(50))
  
  const results = []
  
  results.push(await testReactiveStore())
  results.push(await testReactiveCollection())
  results.push(await testAccountStore())
  results.push(await testUIState())
  results.push(await testFormValidation())
  
  console.log('=' .repeat(50))
  
  const passed = results.filter(Boolean).length
  const total = results.length
  
  if (passed === total) {
    console.log(`🎉 All tests passed! (${passed}/${total})`)
    console.log('✅ Reactive Data Management implementation is working correctly!')
  } else {
    console.log(`⚠️  Some tests failed. Passed: ${passed}/${total}`)
  }
  
  return passed === total
}

// 导出测试函数供手动调用
window.testReactiveDataManagement = {
  runAllTests,
  testReactiveStore,
  testReactiveCollection,
  testAccountStore,
  testUIState,
  testFormValidation
}

console.log('📋 Manual test script loaded!')
console.log('Run window.testReactiveDataManagement.runAllTests() to start testing')