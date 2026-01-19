import { describe, it, expect, beforeEach, vi } from 'vitest'
import { nextTick } from 'vue'
import { createReactiveStore, createReactiveCollection } from '../useReactiveStore.js'

// Mock Wails runtime
vi.mock('../../../wailsjs/runtime/runtime', () => ({
  EventsOn: vi.fn(),
  EventsOff: vi.fn()
}))

describe('createReactiveStore', () => {
  let store

  beforeEach(() => {
    store = createReactiveStore({
      testData: 'initial'
    })
  })

  it('should create store with initial state', () => {
    expect(store.state.testData).toBe('initial')
    expect(store.state.loading).toBe(false)
    expect(store.state.error).toBe(null)
    expect(store.state.lastUpdated).toBe(null)
  })

  it('should execute operations correctly', async () => {
    const mockOperation = vi.fn().mockResolvedValue('success')
    
    const result = await store.executeOperation('test-op', mockOperation)
    
    expect(result).toBe('success')
    expect(mockOperation).toHaveBeenCalledOnce()
    expect(store.state.lastUpdated).toBeTruthy()
  })

  it('should handle operation errors', async () => {
    const mockOperation = vi.fn().mockRejectedValue(new Error('Test error'))
    
    await expect(store.executeOperation('test-op', mockOperation)).rejects.toThrow('Test error')
    expect(store.state.error).toBe('Test error')
  })

  it('should prevent duplicate operations', async () => {
    const mockOperation = vi.fn().mockImplementation(() => 
      new Promise(resolve => setTimeout(() => resolve('success'), 100))
    )
    
    // 启动两个相同的操作
    const promise1 = store.executeOperation('test-op', mockOperation)
    const promise2 = store.executeOperation('test-op', mockOperation)
    
    const [result1, result2] = await Promise.all([promise1, promise2])
    
    // 第二个操作应该返回第一个的结果
    expect(result1).toBe('success')
    expect(result2).toBe('success')
    expect(mockOperation).toHaveBeenCalledOnce() // 只调用一次
  })

  it('should update state correctly', () => {
    store.updateState({ testData: 'updated' })
    
    expect(store.state.testData).toBe('updated')
    expect(store.state.lastUpdated).toBeTruthy()
  })

  it('should update state with function', () => {
    store.updateState(state => {
      state.testData = 'function updated'
    })
    
    expect(store.state.testData).toBe('function updated')
  })

  it('should reset state correctly', () => {
    store.updateState({ testData: 'changed' })
    store.state.error = 'some error'
    
    store.resetState()
    
    expect(store.state.testData).toBe('initial')
    expect(store.state.error).toBe(null)
    expect(store.state.loading).toBe(false)
  })

  it('should clear errors', () => {
    store.state.error = 'test error'
    store.clearError()
    
    expect(store.state.error).toBe(null)
  })

  it('should track pending operations', async () => {
    const mockOperation = vi.fn().mockImplementation(() => 
      new Promise(resolve => setTimeout(() => resolve('success'), 50))
    )
    
    expect(store.isOperationPending('test-op')).toBe(false)
    
    const promise = store.executeOperation('test-op', mockOperation)
    
    expect(store.isOperationPending('test-op')).toBe(true)
    
    await promise
    
    expect(store.isOperationPending('test-op')).toBe(false)
  })
})

describe('createReactiveCollection', () => {
  let collection

  beforeEach(() => {
    collection = createReactiveCollection({
      keyField: 'id',
      initialItems: [
        { id: '1', name: 'Item 1', category: 'A' },
        { id: '2', name: 'Item 2', category: 'B' }
      ]
    })
  })

  it('should create collection with initial items', () => {
    expect(collection.state.items).toHaveLength(2)
    expect(collection.state.items[0].name).toBe('Item 1')
  })

  it('should add items correctly', () => {
    collection.addItems({ id: '3', name: 'Item 3', category: 'C' })
    
    expect(collection.state.items).toHaveLength(3)
    expect(collection.state.items[2].name).toBe('Item 3')
  })

  it('should update existing items when adding', () => {
    collection.addItems({ id: '1', name: 'Updated Item 1', category: 'A' })
    
    expect(collection.state.items).toHaveLength(2) // 没有增加新项
    expect(collection.state.items[0].name).toBe('Updated Item 1')
  })

  it('should remove items correctly', () => {
    collection.removeItems('1')
    
    expect(collection.state.items).toHaveLength(1)
    expect(collection.state.items[0].id).toBe('2')
  })

  it('should update items correctly', () => {
    collection.updateItem('1', { name: 'Modified Item 1' })
    
    expect(collection.state.items[0].name).toBe('Modified Item 1')
    expect(collection.state.items[0].category).toBe('A') // 其他属性保持不变
  })

  it('should get items correctly', () => {
    const item = collection.getItem('1')
    
    expect(item).toBeTruthy()
    expect(item.name).toBe('Item 1')
  })

  it('should select items correctly', () => {
    collection.selectItems(['1', '2'])
    
    expect(collection.state.selectedItems).toEqual(['1', '2'])
  })

  it('should deselect items correctly', () => {
    collection.selectItems(['1', '2'])
    collection.selectItems('1', false)
    
    expect(collection.state.selectedItems).toEqual(['2'])
  })

  it('should toggle select all correctly', () => {
    // 第一次调用应该全选
    collection.toggleSelectAll()
    expect(collection.state.selectedItems).toEqual(['1', '2'])
    
    // 第二次调用应该取消全选
    collection.toggleSelectAll()
    expect(collection.state.selectedItems).toEqual([])
  })

  it('should filter items correctly', async () => {
    collection.setSearchQuery('Item 1')
    
    await nextTick()
    
    expect(collection.filteredItems.value).toHaveLength(1)
    expect(collection.filteredItems.value[0].name).toBe('Item 1')
  })

  it('should filter by field correctly', async () => {
    collection.setFilter('category', 'A')
    
    await nextTick()
    
    expect(collection.filteredItems.value).toHaveLength(1)
    expect(collection.filteredItems.value[0].category).toBe('A')
  })

  it('should sort items correctly', async () => {
    collection.setSorting('name', 'desc')
    
    await nextTick()
    
    expect(collection.filteredItems.value[0].name).toBe('Item 2')
    expect(collection.filteredItems.value[1].name).toBe('Item 1')
  })

  it('should provide correct stats', () => {
    collection.selectItems('1')
    
    expect(collection.stats.value.total).toBe(2)
    expect(collection.stats.value.filtered).toBe(2)
    expect(collection.stats.value.selected).toBe(1)
  })

  it('should handle batch operations', async () => {
    const operations = [
      {
        id: 'op1',
        operation: () => Promise.resolve('result1')
      },
      {
        id: 'op2', 
        operation: () => Promise.resolve('result2')
      }
    ]
    
    const result = await collection.executeBatchOperations(operations)
    
    expect(result.successful).toBe(2)
    expect(result.failed).toBe(0)
    expect(result.total).toBe(2)
  })

  it('should handle batch operation failures', async () => {
    const operations = [
      {
        id: 'op1',
        operation: () => Promise.resolve('result1')
      },
      {
        id: 'op2',
        operation: () => Promise.reject(new Error('Operation failed'))
      }
    ]
    
    const result = await collection.executeBatchOperations(operations)
    
    expect(result.successful).toBe(1)
    expect(result.failed).toBe(1)
    expect(result.total).toBe(2)
    expect(result.errors).toHaveLength(1)
  })
})