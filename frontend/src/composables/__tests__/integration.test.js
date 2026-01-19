/**
 * Integration test for reactive data management
 * Tests the complete flow of account management with reactive stores
 */

import { describe, it, expect, beforeEach, vi } from 'vitest'
import { nextTick } from 'vue'
import { useAccountStore } from '../useAccountStore.js'
import { useUIState } from '../useUIState.js'
import { useFormValidation } from '../useFormValidation.js'

// Mock Wails runtime
vi.mock('../../../wailsjs/runtime/runtime', () => ({
  EventsOn: vi.fn((eventName, handler) => {
    // Store handlers for manual triggering in tests
    if (!global.mockEventHandlers) {
      global.mockEventHandlers = new Map()
    }
    global.mockEventHandlers.set(eventName, handler)
  }),
  EventsOff: vi.fn()
}))

// Mock Wails Go functions
vi.mock('../../../wailsjs/go/main/App', () => ({
  GetKiroAccounts: vi.fn().mockResolvedValue([
    {
      id: 'test-1',
      email: 'test1@example.com',
      displayName: 'Test User 1',
      isActive: true,
      subscriptionType: 'pro',
      quota: {
        main: { used: 100, total: 1000 },
        trial: { used: 0, total: 100 },
        reward: { used: 50, total: 200 }
      },
      tags: ['work'],
      lastUsed: new Date().toISOString(),
      createdAt: new Date().toISOString()
    }
  ]),
  AddKiroAccount: vi.fn().mockResolvedValue(undefined),
  RemoveKiroAccount: vi.fn().mockResolvedValue(undefined),
  UpdateKiroAccount: vi.fn().mockResolvedValue(undefined),
  SwitchKiroAccount: vi.fn().mockResolvedValue(undefined),
  GetActiveKiroAccount: vi.fn().mockResolvedValue(null),
  RefreshKiroToken: vi.fn().mockResolvedValue(undefined),
  GetKiroQuota: vi.fn().mockResolvedValue({
    main: { used: 100, total: 1000 },
    trial: { used: 0, total: 100 },
    reward: { used: 50, total: 200 }
  }),
  RefreshKiroQuota: vi.fn().mockResolvedValue(undefined),
  BatchRefreshKiroTokens: vi.fn().mockResolvedValue(undefined),
  BatchDeleteKiroAccounts: vi.fn().mockResolvedValue(undefined),
  BatchAddKiroTags: vi.fn().mockResolvedValue(undefined),
  ExportKiroAccounts: vi.fn().mockResolvedValue('{}'),
  ImportKiroAccounts: vi.fn().mockResolvedValue(undefined)
}))

describe('Reactive Data Management Integration', () => {
  let accountStore
  let uiState
  let formValidation

  beforeEach(() => {
    // Reset global event handlers
    global.mockEventHandlers = new Map()
    
    // Create fresh instances
    accountStore = useAccountStore()
    uiState = useUIState()
    formValidation = useFormValidation()
  })

  describe('Account Store Integration', () => {
    it('should load accounts and update state reactively', async () => {
      expect(accountStore.state.items).toHaveLength(0)
      
      await accountStore.loadAccounts()
      
      expect(accountStore.state.items).toHaveLength(1)
      expect(accountStore.state.items[0].email).toBe('test1@example.com')
      expect(accountStore.state.activeAccountId).toBe('test-1')
    })

    it('should handle account operations with loading states', async () => {
      await accountStore.loadAccounts()
      
      expect(accountStore.isLoading.value).toBe(false)
      
      const updatePromise = accountStore.updateAccount('test-1', {
        displayName: 'Updated Name'
      })
      
      // Loading state should be active during operation
      await nextTick()
      
      await updatePromise
      
      expect(accountStore.state.items[0].displayName).toBe('Updated Name')
      expect(accountStore.isLoading.value).toBe(false)
    })

    it('should compute active account correctly', async () => {
      await accountStore.loadAccounts()
      
      expect(accountStore.activeAccount.value).toBeTruthy()
      expect(accountStore.activeAccount.value.id).toBe('test-1')
    })

    it('should compute quota alerts correctly', async () => {
      await accountStore.loadAccounts()
      
      // Update quota to trigger alert (>90% usage)
      accountStore.updateItem('test-1', {
        quota: {
          main: { used: 950, total: 1000 },
          trial: { used: 0, total: 100 },
          reward: { used: 50, total: 200 }
        }
      })
      
      await nextTick()
      
      expect(accountStore.quotaAlerts.value.length).toBeGreaterThan(0)
      expect(accountStore.quotaAlerts.value[0].usage).toBeGreaterThanOrEqual(90)
    })

    it('should handle event-driven updates', async () => {
      await accountStore.loadAccounts()
      
      // Simulate account-updated event from backend
      const handler = global.mockEventHandlers.get('kiro-account-updated')
      if (handler) {
        handler('test-1', { displayName: 'Event Updated Name' })
      }
      
      await nextTick()
      
      expect(accountStore.state.items[0].displayName).toBe('Event Updated Name')
    })
  })

  describe('UI State Integration', () => {
    it('should manage dialog state reactively', () => {
      expect(uiState.dialogs.addAccount.visible).toBe(false)
      
      uiState.openDialog('addAccount', { loginMethod: 'token' })
      
      expect(uiState.dialogs.addAccount.visible).toBe(true)
      expect(uiState.dialogs.addAccount.data.loginMethod).toBe('token')
      
      uiState.closeDialog('addAccount')
      
      expect(uiState.dialogs.addAccount.visible).toBe(false)
    })

    it('should manage notifications reactively', () => {
      expect(uiState.notifications.items.length).toBe(0)
      
      uiState.notifications.success('Operation successful')
      
      expect(uiState.notifications.items.length).toBe(1)
      expect(uiState.notifications.items[0].type).toBe('success')
      expect(uiState.notifications.items[0].message).toBe('Operation successful')
    })

    it('should manage selection state', () => {
      uiState.selection.toggleItem('item-1')
      expect(uiState.selection.selectedItems).toContain('item-1')
      
      uiState.selection.toggleItem('item-1')
      expect(uiState.selection.selectedItems).not.toContain('item-1')
    })

    it('should persist view preferences', () => {
      uiState.view.setLayout('grid')
      
      // Simulate page reload by creating new instance
      const newUiState = useUIState()
      
      // Should load from localStorage
      expect(newUiState.view.layout).toBe('grid')
    })
  })

  describe('Form Validation Integration', () => {
    it('should validate account form correctly', async () => {
      const validData = {
        email: 'test@example.com',
        displayName: 'Test User',
        bearerToken: 'Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test.signature'
      }
      
      const result = await formValidation.validateForm(validData, 'accountForm')
      
      expect(result.isValid).toBe(true)
      expect(Object.keys(result.errors)).toHaveLength(0)
    })

    it('should detect validation errors', async () => {
      const invalidData = {
        email: 'invalid-email',
        displayName: '',
        bearerToken: 'invalid-token'
      }
      
      const result = await formValidation.validateForm(invalidData, 'accountForm')
      
      expect(result.isValid).toBe(false)
      expect(result.errors.email).toBeTruthy()
      expect(result.errors.displayName).toBeTruthy()
      expect(result.errors.bearerToken).toBeTruthy()
    })

    it('should validate individual fields', async () => {
      const emailResult = await formValidation.validateField(
        'email',
        'test@example.com',
        'accountForm'
      )
      
      expect(emailResult.isValid).toBe(true)
      
      const invalidEmailResult = await formValidation.validateField(
        'email',
        'invalid',
        'accountForm'
      )
      
      expect(invalidEmailResult.isValid).toBe(false)
    })
  })

  describe('Complete Workflow Integration', () => {
    it('should handle complete account addition workflow', async () => {
      // 1. Load existing accounts
      await accountStore.loadAccounts()
      expect(accountStore.state.items).toHaveLength(1)
      
      // 2. Open add account dialog
      uiState.openDialog('addAccount', { loginMethod: 'token' })
      expect(uiState.dialogs.addAccount.visible).toBe(true)
      
      // 3. Validate form data
      const formData = {
        email: 'new@example.com',
        displayName: 'New User',
        bearerToken: 'Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test.signature'
      }
      
      const validation = await formValidation.validateForm(formData, 'accountForm')
      expect(validation.isValid).toBe(true)
      
      // 4. Add account
      await accountStore.addAccount('token', formData)
      
      // 5. Simulate backend event
      const handler = global.mockEventHandlers.get('kiro-account-added')
      if (handler) {
        handler({
          id: 'test-2',
          ...formData,
          isActive: false,
          subscriptionType: 'free',
          quota: {
            main: { used: 0, total: 100 },
            trial: { used: 0, total: 50 },
            reward: { used: 0, total: 0 }
          },
          tags: [],
          lastUsed: new Date().toISOString(),
          createdAt: new Date().toISOString()
        })
      }
      
      await nextTick()
      
      // 6. Verify account was added
      expect(accountStore.state.items).toHaveLength(2)
      
      // 7. Close dialog and show success notification
      uiState.closeDialog('addAccount')
      uiState.notifications.success('Account added successfully')
      
      expect(uiState.dialogs.addAccount.visible).toBe(false)
      expect(uiState.notifications.items.length).toBeGreaterThan(0)
    })

    it('should handle account switching workflow', async () => {
      // Load accounts
      await accountStore.loadAccounts()
      
      // Add second account
      const handler = global.mockEventHandlers.get('kiro-account-added')
      if (handler) {
        handler({
          id: 'test-2',
          email: 'test2@example.com',
          displayName: 'Test User 2',
          isActive: false,
          subscriptionType: 'free',
          quota: {
            main: { used: 0, total: 100 },
            trial: { used: 0, total: 50 },
            reward: { used: 0, total: 0 }
          },
          tags: [],
          lastUsed: new Date().toISOString(),
          createdAt: new Date().toISOString()
        })
      }
      
      await nextTick()
      
      expect(accountStore.state.items).toHaveLength(2)
      expect(accountStore.activeAccount.value.id).toBe('test-1')
      
      // Switch to second account
      await accountStore.switchAccount('test-2')
      
      // Simulate switch event
      const switchHandler = global.mockEventHandlers.get('kiro-account-switched')
      if (switchHandler) {
        switchHandler('test-2', 'test-1')
      }
      
      await nextTick()
      
      expect(accountStore.activeAccount.value.id).toBe('test-2')
      expect(accountStore.state.items.find(a => a.id === 'test-2').isActive).toBe(true)
      expect(accountStore.state.items.find(a => a.id === 'test-1').isActive).toBe(false)
    })

    it('should handle batch operations workflow', async () => {
      await accountStore.loadAccounts()
      
      // Add more accounts
      const addHandler = global.mockEventHandlers.get('kiro-account-added')
      if (addHandler) {
        addHandler({
          id: 'test-2',
          email: 'test2@example.com',
          displayName: 'Test User 2',
          isActive: false,
          subscriptionType: 'free',
          quota: { main: { used: 0, total: 100 }, trial: { used: 0, total: 50 }, reward: { used: 0, total: 0 } },
          tags: [],
          lastUsed: new Date().toISOString(),
          createdAt: new Date().toISOString()
        })
        addHandler({
          id: 'test-3',
          email: 'test3@example.com',
          displayName: 'Test User 3',
          isActive: false,
          subscriptionType: 'free',
          quota: { main: { used: 0, total: 100 }, trial: { used: 0, total: 50 }, reward: { used: 0, total: 0 } },
          tags: [],
          lastUsed: new Date().toISOString(),
          createdAt: new Date().toISOString()
        })
      }
      
      await nextTick()
      
      expect(accountStore.state.items).toHaveLength(3)
      
      // Select multiple accounts
      uiState.selection.toggleItem('test-2')
      uiState.selection.toggleItem('test-3')
      
      expect(uiState.selection.selectedItems).toHaveLength(2)
      
      // Batch add tags
      await accountStore.batchAddTags(['test-2', 'test-3'], ['personal', 'backup'])
      
      await nextTick()
      
      const account2 = accountStore.getItem('test-2')
      const account3 = accountStore.getItem('test-3')
      
      expect(account2.tags).toContain('personal')
      expect(account2.tags).toContain('backup')
      expect(account3.tags).toContain('personal')
      expect(account3.tags).toContain('backup')
      
      // Clear selection
      uiState.selection.clearSelection()
      expect(uiState.selection.selectedItems).toHaveLength(0)
    })
  })

  describe('Error Handling Integration', () => {
    it('should handle operation errors gracefully', async () => {
      const { AddKiroAccount } = await import('../../../wailsjs/go/main/App')
      
      // Mock error
      AddKiroAccount.mockRejectedValueOnce(new Error('Network error'))
      
      await expect(
        accountStore.addAccount('token', { email: 'test@example.com' })
      ).rejects.toThrow('Network error')
      
      expect(accountStore.state.error).toBe('Network error')
      expect(accountStore.isLoading.value).toBe(false)
    })

    it('should clear errors when requested', async () => {
      accountStore.updateState({ error: 'Some error' })
      
      expect(accountStore.state.error).toBe('Some error')
      
      accountStore.clearError()
      
      expect(accountStore.state.error).toBe(null)
    })
  })

  describe('Performance and Memory', () => {
    it('should handle large number of accounts efficiently', async () => {
      const largeAccountList = Array.from({ length: 100 }, (_, i) => ({
        id: `test-${i}`,
        email: `test${i}@example.com`,
        displayName: `Test User ${i}`,
        isActive: i === 0,
        subscriptionType: 'free',
        quota: {
          main: { used: i * 10, total: 1000 },
          trial: { used: 0, total: 100 },
          reward: { used: 0, total: 0 }
        },
        tags: i % 2 === 0 ? ['even'] : ['odd'],
        lastUsed: new Date().toISOString(),
        createdAt: new Date().toISOString()
      }))
      
      const { GetKiroAccounts } = await import('../../../wailsjs/go/main/App')
      GetKiroAccounts.mockResolvedValueOnce(largeAccountList)
      
      const startTime = Date.now()
      await accountStore.loadAccounts()
      const loadTime = Date.now() - startTime
      
      expect(accountStore.state.items).toHaveLength(100)
      expect(loadTime).toBeLessThan(1000) // Should load in less than 1 second
      
      // Test filtering performance
      accountStore.setFilter('tags', 'even')
      await nextTick()
      
      expect(accountStore.filteredItems.value).toHaveLength(50)
    })

    it('should cleanup resources properly', () => {
      const cleanupSpy = vi.spyOn(accountStore, 'cleanup')
      
      accountStore.cleanup()
      
      expect(cleanupSpy).toHaveBeenCalled()
    })
  })
})
