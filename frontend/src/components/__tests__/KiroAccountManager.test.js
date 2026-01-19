import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import KiroAccountManager from '../KiroAccountManager.vue'

// Mock Wails runtime
vi.mock('../../wailsjs/runtime/runtime', () => ({
  EventsOn: vi.fn(),
  EventsOff: vi.fn()
}))

// Mock Wails Go functions
vi.mock('../../wailsjs/go/main/App', () => ({
  GetKiroAccounts: vi.fn().mockResolvedValue([]),
  AddKiroAccount: vi.fn(),
  RemoveKiroAccount: vi.fn(),
  UpdateKiroAccount: vi.fn(),
  SwitchKiroAccount: vi.fn(),
  GetActiveKiroAccount: vi.fn(),
  StartKiroOAuth: vi.fn(),
  ValidateKiroToken: vi.fn(),
  RefreshKiroToken: vi.fn(),
  GetKiroQuota: vi.fn(),
  RefreshKiroQuota: vi.fn(),
  BatchRefreshKiroTokens: vi.fn(),
  BatchDeleteKiroAccounts: vi.fn(),
  BatchAddKiroTags: vi.fn(),
  ExportKiroAccounts: vi.fn(),
  ImportKiroAccounts: vi.fn()
}))

const i18n = createI18n({
  locale: 'zh-CN',
  messages: {
    'zh-CN': {}
  }
})

describe('KiroAccountManager', () => {
  it('renders correctly', () => {
    const wrapper = mount(KiroAccountManager, {
      global: {
        plugins: [i18n]
      }
    })
    
    expect(wrapper.find('.kiro-account-manager').exists()).toBe(true)
    expect(wrapper.find('.manager-header').exists()).toBe(true)
    expect(wrapper.find('.manager-filters').exists()).toBe(true)
    expect(wrapper.find('.accounts-container').exists()).toBe(true)
  })

  it('shows empty state when no accounts', () => {
    const wrapper = mount(KiroAccountManager, {
      global: {
        plugins: [i18n]
      }
    })
    
    expect(wrapper.find('.empty-state').exists()).toBe(true)
    expect(wrapper.text()).toContain('暂无账号')
  })

  it('opens add account dialog when button clicked', async () => {
    const wrapper = mount(KiroAccountManager, {
      global: {
        plugins: [i18n]
      }
    })
    
    await wrapper.find('.btn-primary').trigger('click')
    expect(wrapper.find('.add-account-dialog').exists()).toBe(true)
  })

  it('filters accounts by search query', async () => {
    const wrapper = mount(KiroAccountManager, {
      global: {
        plugins: [i18n]
      }
    })
    
    // Set some mock accounts
    wrapper.vm.state.accounts = [
      { id: '1', email: 'test@example.com', displayName: 'Test User', tags: [], quota: { main: { used: 0, total: 100 } } },
      { id: '2', email: 'user@test.com', displayName: 'Another User', tags: [], quota: { main: { used: 0, total: 100 } } }
    ]
    
    await wrapper.vm.$nextTick()
    
    const searchInput = wrapper.find('.search-input')
    await searchInput.setValue('test@example.com')
    
    expect(wrapper.vm.filteredAccounts).toHaveLength(1)
    expect(wrapper.vm.filteredAccounts[0].email).toBe('test@example.com')
  })
})