<template>
  <div class="kiro-account-manager-enhanced">
    <!-- 头部工具栏 -->
    <div class="manager-header">
      <div class="header-title">
        <h2>Kiro 账号管理 (Enhanced)</h2>
        <span class="account-count">{{ accountStore.stats.value.total }} 个账号</span>
        <span v-if="accountStore.quotaAlerts.value.length > 0" class="alert-badge">
          {{ accountStore.quotaAlerts.value.length }} 个警告
        </span>
      </div>
      <div class="header-actions">
        <button 
          class="btn-primary" 
          @click="uiState.dialogs.open('addAccount')"
          :disabled="accountStore.isLoading.value"
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 5v14M5 12h14"/>
          </svg>
          添加账号
        </button>
        <button 
          v-if="uiState.hasSelectedItems.value" 
          class="btn-secondary"
          @click="uiState.dialogs.open('batchOperations', { selectedIds: uiState.state.selection.selectedIds })"
        >
          批量操作 ({{ uiState.selectedCount.value }})
        </button>
      </div>
    </div>

    <!-- 筛选和搜索 -->
    <div class="manager-filters">
      <div class="filter-group">
        <input 
          :value="uiState.state.filters.searchQuery"
          @input="uiState.filters.setSearch($event.target.value)"
          type="text" 
          placeholder="搜索账号..." 
          class="search-input"
        >
        <select 
          :value="uiState.state.filters.tagFilter"
          @change="uiState.filters.setTagFilter($event.target.value)"
          class="filter-select"
        >
          <option value="">所有标签</option>
          <option v-for="tag in accountStore.allTags.value" :key="tag" :value="tag">
            {{ tag }}
          </option>
        </select>
        <select 
          :value="uiState.state.filters.subscriptionFilter"
          @change="uiState.filters.setSubscriptionFilter($event.target.value)"
          class="filter-select"
        >
          <option value="">所有订阅</option>
          <option value="free">Free</option>
          <option value="pro">Pro</option>
          <option value="pro_plus">Pro+</option>
        </select>
        <select 
          :value="uiState.state.filters.sortBy"
          @change="uiState.filters.setSorting($event.target.value, uiState.state.filters.sortDirection)"
          class="filter-select"
        >
          <option value="lastUsed">最近使用</option>
          <option value="displayName">名称</option>
          <option value="subscriptionType">订阅类型</option>
        </select>
      </div>
    </div>

    <!-- 配额警告 -->
    <div v-if="accountStore.quotaAlerts.value.length > 0" class="quota-alerts">
      <div class="alert-header">
        <h4>配额警告</h4>
        <button @click="showQuotaAlerts = !showQuotaAlerts" class="btn-toggle">
          {{ showQuotaAlerts ? '隐藏' : '显示' }}
        </button>
      </div>
      <div v-if="showQuotaAlerts" class="alerts-list">
        <div v-for="alert in accountStore.quotaAlerts.value" :key="alert.accountId" class="alert-item">
          <i class="warning-icon">⚠️</i>
          <span>{{ alert.message }}</span>
        </div>
      </div>
    </div>

    <!-- 账号列表 -->
    <div class="accounts-container">
      <div v-if="accountStore.isLoading.value" class="loading-state">
        <div class="loading-spinner"></div>
        <span>加载账号中...</span>
      </div>
      
      <div v-else-if="filteredAccounts.length === 0" class="empty-state">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
          <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
          <circle cx="12" cy="7" r="4"/>
        </svg>
        <h3>{{ uiState.hasActiveFilters.value ? '未找到匹配的账号' : '暂无账号' }}</h3>
        <p>{{ uiState.hasActiveFilters.value ? '尝试调整搜索条件' : '点击"添加账号"开始管理您的 Kiro 账号' }}</p>
      </div>

      <div v-else class="accounts-list">
        <!-- 批量选择工具栏 -->
        <div v-if="uiState.state.selection.selectMode" class="batch-toolbar">
          <label class="select-all">
            <input 
              type="checkbox" 
              :checked="isAllSelected"
              @change="toggleSelectAll"
            >
            全选
          </label>
          <span class="selected-count">已选择 {{ uiState.selectedCount.value }} 个账号</span>
          <button @click="uiState.selection.setSelectMode(false)" class="btn-cancel">
            取消选择
          </button>
        </div>

        <div 
          v-for="account in filteredAccounts" 
          :key="account.id"
          :class="['account-card-enhanced', { 
            active: account.isActive, 
            selected: uiState.state.selection.selectedIds.includes(account.id) 
          }]"
        >
          <!-- 选择框 -->
          <div v-if="uiState.state.selection.selectMode" class="selection-checkbox">
            <input 
              type="checkbox" 
              :checked="uiState.state.selection.selectedIds.includes(account.id)"
              @change="uiState.selection.toggle(account.id)"
            >
          </div>

          <!-- 账号信息 -->
          <div class="card-content" @click="handleAccountClick(account)">
            <div class="account-header">
              <div class="account-avatar">
                <img v-if="account.avatar" :src="account.avatar" :alt="account.displayName">
                <div v-else class="avatar-placeholder">
                  {{ account.displayName.charAt(0).toUpperCase() }}
                </div>
              </div>
              <div class="account-info">
                <div class="account-name">
                  {{ account.displayName }}
                  <span v-if="account.isActive" class="active-badge">当前</span>
                </div>
                <div class="account-email">{{ account.email }}</div>
                <div class="account-meta">
                  <span class="subscription-type">{{ getSubscriptionLabel(account.subscriptionType) }}</span>
                  <span class="last-used">{{ formatDate(account.lastUsed) }}</span>
                </div>
              </div>
            </div>

            <!-- 配额信息 -->
            <div class="quota-section" v-if="account.quota">
              <div class="quota-item" v-for="(quota, type) in account.quota" :key="type">
                <div class="quota-label">{{ getQuotaLabel(type) }}</div>
                <div class="quota-text">
                  已用 {{ quota.used }} / 总量 {{ quota.total }}
                  <span v-if="quota.total > 0"> / 剩余 {{ Math.max(0, quota.total - quota.used) }}</span>
                </div>
              </div>
            </div>

            <!-- 标签 -->
            <div v-if="account.tags && account.tags.length > 0" class="tags-section">
              <span v-for="tag in account.tags" :key="tag" class="tag">{{ tag }}</span>
            </div>
          </div>

          <!-- 操作按钮 -->
          <div class="card-actions">
            <button 
              v-if="!account.isActive" 
              class="btn-switch" 
              @click="switchAccount(account)"
              :disabled="accountStore.isOperationPending(`switch-account-${account.id}`)"
            >
              {{ accountStore.isOperationPending(`switch-account-${account.id}`) ? '切换中...' : '切换' }}
            </button>
            <button 
              class="btn-icon" 
              @click="editAccount(account)"
              title="编辑账号"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
              </svg>
            </button>
            <button 
              class="btn-icon danger" 
              @click="deleteAccount(account)"
              title="删除账号"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2 2h4a2 2 0 0 1 2 2v2"/>
              </svg>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 状态栏 -->
    <div class="status-bar">
      <div class="status-left">
        <span>总计: {{ accountStore.stats.value.total }}</span>
        <span>有效: {{ accountStore.validAccountCount.value }}</span>
        <span v-if="uiState.hasActiveFilters.value">
          筛选: {{ accountStore.stats.value.filtered }}
        </span>
      </div>
      <div class="status-right">
        <button 
          @click="uiState.selection.setSelectMode(!uiState.state.selection.selectMode)"
          class="btn-select-mode"
        >
          {{ uiState.state.selection.selectMode ? '退出选择' : '批量选择' }}
        </button>
        <button 
          @click="refreshAllQuotas"
          :disabled="accountStore.hasPendingOperations.value"
          class="btn-refresh"
        >
          {{ accountStore.hasPendingOperations.value ? '刷新中...' : '刷新配额' }}
        </button>
      </div>
    </div>

    <!-- 通知系统 -->
    <div v-if="uiState.hasNotifications.value" class="notifications">
      <div 
        v-for="notification in uiState.state.notifications" 
        :key="notification.id"
        :class="['notification', notification.type]"
      >
        <div class="notification-content">
          <strong v-if="notification.title">{{ notification.title }}</strong>
          <p>{{ notification.message }}</p>
        </div>
        <button @click="uiState.notifications.remove(notification.id)" class="btn-close-notification">
          ×
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useGlobalAccountStore } from '../composables/useAccountStore.js'
import { useUIState } from '../composables/useUIState.js'

// 响应式存储
const accountStore = useGlobalAccountStore()
const uiState = useUIState()

// 本地状态
const showQuotaAlerts = ref(true)

// 计算属性
const filteredAccounts = computed(() => {
  let accounts = [...accountStore.state.items]
  
  // 搜索筛选
  if (uiState.state.filters.searchQuery) {
    const query = uiState.state.filters.searchQuery.toLowerCase()
    accounts = accounts.filter(account => 
      account.email.toLowerCase().includes(query) ||
      account.displayName.toLowerCase().includes(query) ||
      account.tags.some(tag => tag.toLowerCase().includes(query))
    )
  }
  
  // 标签筛选
  if (uiState.state.filters.tagFilter) {
    accounts = accounts.filter(account => 
      account.tags.includes(uiState.state.filters.tagFilter)
    )
  }
  
  // 订阅类型筛选
  if (uiState.state.filters.subscriptionFilter) {
    accounts = accounts.filter(account => 
      account.subscriptionType === uiState.state.filters.subscriptionFilter
    )
  }
  
  // 排序
  accounts.sort((a, b) => {
    const field = uiState.state.filters.sortBy
    const direction = uiState.state.filters.sortDirection
    
    let aValue = a[field]
    let bValue = b[field]
    
    if (field === 'lastUsed') {
      aValue = new Date(aValue)
      bValue = new Date(bValue)
    }
    
    let comparison = 0
    if (aValue < bValue) comparison = -1
    if (aValue > bValue) comparison = 1
    
    return direction === 'desc' ? -comparison : comparison
  })
  
  return accounts
})

const isAllSelected = computed(() => {
  const allIds = filteredAccounts.value.map(acc => acc.id)
  return allIds.length > 0 && allIds.every(id => 
    uiState.state.selection.selectedIds.includes(id)
  )
})

// 生命周期
onMounted(async () => {
  try {
    await accountStore.loadAccounts()
    if (accountStore.state.items.length > 0) {
      await refreshAllQuotas(true)
    }
  } catch (error) {
    uiState.notifications.error('加载账号数据失败: ' + error.message)
  }
})

onUnmounted(() => {
  accountStore.cleanup()
})

// 方法
async function switchAccount(account) {
  try {
    await accountStore.switchAccount(account.id)
    uiState.notifications.success(`已切换到账号: ${account.displayName}`)
  } catch (error) {
    uiState.notifications.error('切换账号失败: ' + error.message)
  }
}

function editAccount(account) {
  uiState.dialogs.open('editAccount', { account })
}

function deleteAccount(account) {
  uiState.dialogs.open('deleteAccount', { account })
}

function handleAccountClick(account) {
  if (uiState.state.selection.selectMode) {
    uiState.selection.toggle(account.id)
  } else if (!account.isActive) {
    switchAccount(account)
  }
}

function toggleSelectAll() {
  const allIds = filteredAccounts.value.map(acc => acc.id)
  uiState.selection.toggleAll(allIds)
}

async function refreshAllQuotas(silent = false) {
  try {
    const accountIds = accountStore.state.items.map(acc => acc.id)
    const results = await Promise.allSettled(
      accountIds.map(id => accountStore.refreshAccountQuota(id))
    )
    
    const successful = results.filter(r => r.status === 'fulfilled').length
    const failed = results.filter(r => r.status === 'rejected').length
    
    if (!silent) {
      if (failed === 0) {
        uiState.notifications.success('所有账号配额刷新成功')
      } else {
        uiState.notifications.warning(`配额刷新完成: ${successful} 成功, ${failed} 失败`)
      }
    }
  } catch (error) {
    if (!silent) {
      uiState.notifications.error('刷新配额失败: ' + error.message)
    }
  }
}

// 工具函数
function formatDate(dateString) {
  return new Date(dateString).toLocaleString()
}

function getSubscriptionLabel(type) {
  const labels = {
    'free': 'Free',
    'pro': 'Pro',
    'pro_plus': 'Pro+'
  }
  return labels[type] || type
}

function getQuotaLabel(type) {
  const labels = {
    'main': '主配额',
    'trial': '试用',
    'reward': '奖励'
  }
  return labels[type] || type
}
</script>

<style scoped>
.kiro-account-manager-enhanced {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--bg-surface);
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
}

/* 头部样式 */
.manager-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  border-bottom: 1px solid var(--border-subtle);
  background: var(--bg-surface);
}

.header-title {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-title h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--text-primary);
}

.account-count {
  font-size: 13px;
  color: var(--text-muted);
  background: var(--bg-hover);
  padding: 4px 8px;
  border-radius: 12px;
  font-weight: 500;
}

.alert-badge {
  font-size: 13px;
  color: white;
  background: #ff4757;
  padding: 4px 8px;
  border-radius: 12px;
  font-weight: 500;
}

/* 筛选区域 */
.manager-filters {
  padding: 16px 24px;
  border-bottom: 1px solid var(--border-subtle);
  background: var(--bg-surface);
}

.filter-group {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
}

.search-input, .filter-select {
  padding: 8px 12px;
  border: 1px solid var(--border-default);
  border-radius: 6px;
  background: var(--bg-elevated);
  color: var(--text-primary);
  font-size: 13px;
  transition: all 0.2s ease;
}

.search-input {
  min-width: 240px;
  flex: 1;
  max-width: 320px;
}

/* 配额警告 */
.quota-alerts {
  background: rgba(255, 71, 87, 0.1);
  border-bottom: 1px solid rgba(255, 71, 87, 0.2);
  padding: 16px 24px;
}

.alert-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.alert-header h4 {
  margin: 0;
  color: #ff4757;
  font-size: 14px;
  font-weight: 600;
}

.alerts-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.alert-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: #ff4757;
}

/* 账号列表 */
.accounts-container {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
}

.batch-toolbar {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px 16px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-default);
  border-radius: 8px;
  margin-bottom: 16px;
}

.select-all {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 500;
}

.selected-count {
  font-size: 13px;
  color: var(--text-secondary);
}

.accounts-list {
  display: grid;
  gap: 16px;
  grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
}

.account-card-enhanced {
  display: flex;
  background: var(--bg-elevated);
  border: 1px solid var(--border-subtle);
  border-radius: 12px;
  overflow: hidden;
  transition: all 0.3s ease;
}

.account-card-enhanced:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
}

.account-card-enhanced.active {
  border-color: var(--accent-primary);
  box-shadow: 0 4px 20px rgba(176, 128, 255, 0.2);
}

.account-card-enhanced.selected {
  border-color: var(--accent-primary);
  background: rgba(176, 128, 255, 0.05);
}

.selection-checkbox {
  display: flex;
  align-items: center;
  padding: 20px 16px;
  border-right: 1px solid var(--border-subtle);
}

/* 状态栏 */
.status-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 24px;
  border-top: 1px solid var(--border-subtle);
  background: var(--bg-surface);
  font-size: 13px;
}

.status-left {
  display: flex;
  gap: 16px;
  color: var(--text-secondary);
}

.status-right {
  display: flex;
  gap: 12px;
}

/* 通知系统 */
.notifications {
  position: fixed;
  top: 20px;
  right: 20px;
  z-index: 1000;
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-width: 400px;
}

.notification {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 16px;
  border-radius: 8px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
  animation: slideIn 0.3s ease;
}

.notification.success {
  background: #2ed573;
  color: white;
}

.notification.error {
  background: #ff4757;
  color: white;
}

.notification.warning {
  background: #ffa502;
  color: white;
}

.notification.info {
  background: var(--accent-primary);
  color: white;
}

.notification-content {
  flex: 1;
}

.notification-content strong {
  display: block;
  margin-bottom: 4px;
  font-size: 14px;
}

.notification-content p {
  margin: 0;
  font-size: 13px;
  opacity: 0.9;
}

.btn-close-notification {
  background: none;
  border: none;
  color: currentColor;
  font-size: 18px;
  cursor: pointer;
  opacity: 0.7;
  transition: opacity 0.2s ease;
}

.btn-close-notification:hover {
  opacity: 1;
}

@keyframes slideIn {
  from {
    opacity: 0;
    transform: translateX(100%);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

/* 按钮样式 */
.btn-primary, .btn-secondary, .btn-switch, .btn-select-mode, .btn-refresh, .btn-toggle, .btn-cancel {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  border: none;
}

.btn-primary {
  background: linear-gradient(135deg, var(--accent-primary), var(--accent-hover));
  color: white;
}

.btn-secondary, .btn-select-mode, .btn-refresh, .btn-toggle {
  background: var(--bg-elevated);
  color: var(--text-secondary);
  border: 1px solid var(--border-default);
}

.btn-switch {
  background: linear-gradient(135deg, var(--accent-primary), var(--accent-hover));
  color: white;
  font-size: 12px;
  padding: 6px 12px;
}

.btn-cancel {
  background: var(--bg-hover);
  color: var(--text-secondary);
}

.btn-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  background: transparent;
  border: 1px solid var(--border-subtle);
  border-radius: 6px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-icon:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.btn-icon.danger:hover {
  background: #ff4757;
  color: white;
  border-color: #ff4757;
}

/* 其他样式继承自原组件 */
.card-content {
  flex: 1;
  padding: 20px;
  cursor: pointer;
}

.account-header {
  display: flex;
  gap: 16px;
  margin-bottom: 16px;
  align-items: flex-start;
}

.account-avatar {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  overflow: hidden;
  flex-shrink: 0;
}

.account-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar-placeholder {
  width: 100%;
  height: 100%;
  background: linear-gradient(135deg, var(--accent-primary), var(--accent-hover));
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 18px;
}

.account-info {
  flex: 1;
  min-width: 0;
}

.account-name {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 6px;
}

.active-badge {
  background: linear-gradient(135deg, var(--accent-primary), var(--accent-hover));
  color: white;
  padding: 4px 10px;
  border-radius: 12px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
}

.account-email {
  font-size: 13px;
  color: var(--text-secondary);
  margin-bottom: 8px;
}

.account-meta {
  display: flex;
  gap: 16px;
  font-size: 12px;
  color: var(--text-muted);
}

.subscription-type {
  padding: 4px 8px;
  background: var(--bg-hover);
  border-radius: 6px;
  font-weight: 500;
  text-transform: uppercase;
}

.quota-section {
  margin-bottom: 16px;
  padding: 12px;
  background: var(--bg-hover);
  border-radius: 8px;
}

.quota-item {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}

.quota-item:last-child {
  margin-bottom: 0;
}

.quota-label {
  font-size: 12px;
  color: var(--text-secondary);
  width: 50px;
  flex-shrink: 0;
}

.quota-bar {
  flex: 1;
  height: 8px;
  background: var(--bg-base);
  border-radius: 4px;
  overflow: hidden;
}

.quota-fill {
  height: 100%;
  transition: width 0.6s ease;
  border-radius: 4px;
}

.quota-text {
  font-size: 11px;
  color: var(--text-muted);
  white-space: nowrap;
  min-width: 80px;
  text-align: right;
}

.tags-section {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 12px;
}

.tag {
  background: var(--bg-hover);
  color: var(--text-secondary);
  padding: 4px 8px;
  border-radius: 6px;
  font-size: 11px;
  font-weight: 500;
  border: 1px solid var(--border-subtle);
}

.card-actions {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 20px 16px;
  border-left: 1px solid var(--border-subtle);
  background: var(--bg-surface);
  min-width: 80px;
}

.loading-state, .empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 40px;
  text-align: center;
  color: var(--text-muted);
  min-height: 300px;
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 3px solid var(--border-subtle);
  border-top: 3px solid var(--accent-primary);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 20px;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.empty-state h3 {
  margin: 0 0 12px 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-secondary);
}

.empty-state p {
  margin: 0;
  font-size: 14px;
  line-height: 1.5;
}
</style>
