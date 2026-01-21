<script setup>
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import {
  GetKiroAccounts, AddKiroAccount, RemoveKiroAccount, UpdateKiroAccount,
  SwitchKiroAccount, GetActiveKiroAccount, StartKiroOAuth, ValidateKiroToken,
  RefreshKiroToken, GetKiroQuota, RefreshKiroQuota, BatchRefreshKiroTokens,
  BatchDeleteKiroAccounts, BatchAddKiroTags, ExportKiroAccounts, ImportKiroAccounts,
  GetAccountSettings, UpdateAccountSettings,
  GetTags, AddTag, DeleteTag, LogToTerminal,
  CompleteKiroOAuthWithURL
} from '../../wailsjs/go/main/App'

const { t } = useI18n()
const emit = defineEmits(['close'])

// 响应式状态
const state = reactive({
  accounts: [],
  tags: [],
  selectedAccounts: [],
  loading: false,
  saving: false,
  refreshing: false,
  filterTag: '',
  sortBy: 'lastUsed',
  searchQuery: '',
  errorMessage: '' // 添加错误消息状态
})

const settings = reactive({
  autoChangeMachineID: false
})

// 对话框状态
const dialogs = reactive({
  showAddDialog: false,
  showEditDialog: false,
  showDeleteDialog: false,
  showBatchDialog: false,
  showExportDialog: false,
  showImportDialog: false,
  showSettingsDialog: false,
  showTagManager: false
})

// 标签表单
const tagForm = reactive({
  name: '',
  color: '#3B82F6',
  description: ''
})

// 表单数据
const accountForm = reactive({
  id: '',
  displayName: '',
  notes: '',
  loginMethod: 'token',
  provider: 'google',
  refreshToken: '',
  email: '',
  password: '',
  tags: []
})

// OAuth 流程状态
const oauthState = reactive({
  isWaitingForCallback: false,
  callbackUrl: '',
  authUrl: '',
  magicCopied: false,
});

const magicCopied = ref(false);

const copyMagicSnippet = () => {
  const snippet = `fetch('http://127.0.0.1:54321/oauth/callback?fullUrl=' + encodeURIComponent(window.location.href)).then(() => alert('捕获成功！认证已在应用中完成。')).catch(() => alert('捕获失败。请确保应用正在运行。'));`;
  navigator.clipboard.writeText(snippet);
  magicCopied.value = true;
  setTimeout(() => magicCopied.value = false, 2000);
};

// 表单验证错误
const formErrors = reactive({
  email: '',
  password: '',
  refreshToken: ''
})

// 其他状态
const batchOperation = ref('')
const exportPassword = ref('')
const importFile = ref(null)
const importPassword = ref('')
const deleteTarget = ref(null)
const editingAccount = ref(null)
const switchingId = ref(null)
const refreshingId = ref(null)

// 计算属性
const filteredAccounts = computed(() => {
  let filtered = [...state.accounts]
  
  if (state.searchQuery) {
    const query = state.searchQuery.toLowerCase()
    filtered = filtered.filter(account => 
      account.email.toLowerCase().includes(query) ||
      account.displayName.toLowerCase().includes(query)
    )
  }
  
  filtered.sort((a, b) => {
    switch (state.sortBy) {
      case 'name':
        return a.displayName.localeCompare(b.displayName)
      case 'lastUsed':
        return new Date(b.lastUsed) - new Date(a.lastUsed)
      case 'quota':
        const aUsage = a.quota.main.used / a.quota.main.total
        const bUsage = b.quota.main.used / b.quota.main.total
        return bUsage - aUsage
      case 'subscription':
        const subOrder = { 'pro_plus': 3, 'pro': 2, 'free': 1 }
        return (subOrder[b.subscriptionType] || 0) - (subOrder[a.subscriptionType] || 0)
      default:
        return 0
    }
  })
  
  return filtered
})

const activeAccount = computed(() => {
  return state.accounts.find(account => account.isActive)
})

// 生命周期
onMounted(async () => {
  await LogToTerminal('=== KiroAccountManager onMounted ===')
  await LogToTerminal('→ 开始加载数据...')
  await loadAccounts()
  await LogToTerminal('→ 账号数量: ' + state.accounts.length)
  await loadTags()
  await loadSettings()
  
  EventsOn('kiro-account-added', handleAccountAdded)
  EventsOn('kiro-account-removed', handleAccountRemoved)
  EventsOn('kiro-account-switched', handleAccountSwitched)
  EventsOn('kiro-quota-updated', handleQuotaUpdated)
  await LogToTerminal('=== KiroAccountManager 初始化完成 ===')
})

onUnmounted(() => {
  EventsOff('kiro-account-added')
  EventsOff('kiro-account-removed')
  EventsOff('kiro-account-switched')
  EventsOff('kiro-quota-updated')
})

// 数据加载
async function loadAccounts() {
  await LogToTerminal('=== loadAccounts 开始 ===')
  state.loading = true
  try {
    await LogToTerminal('→ 调用 GetKiroAccounts...')
    const accounts = await GetKiroAccounts()
    await LogToTerminal('✓ 获取到账号数据，数量: ' + (accounts ? accounts.length : 0))
    state.accounts = accounts || []
    await LogToTerminal('✓ state.accounts 已更新')
  } catch (error) {
    await LogToTerminal('✗ 加载账号失败: ' + error)
    console.error('✗ 加载账号失败:', error)
    state.accounts = []
  } finally {
    state.loading = false
    await LogToTerminal('=== loadAccounts 完成 ===')
  }
}

async function loadTags() {
  try {
    const tags = await GetTags()
    state.tags = tags || []
  } catch (error) {
    console.error('Failed to load tags:', error)
    state.tags = []
  }
}

async function saveTag() {
  if (!tagForm.name) return
  
  try {
    await AddTag({
      name: tagForm.name,
      color: tagForm.color,
      description: tagForm.description
    })
    
    // Reset form
    tagForm.name = ''
    tagForm.color = '#3B82F6'
    tagForm.description = ''
    
    await loadTags()
  } catch (error) {
    console.error('Failed to save tag:', error)
    alert('保存标签失败: ' + error.message)
  }
}

async function removeTag(tagName) {
  if (!confirm(`确定要删除标签 "${tagName}" 吗？`)) return
  
  try {
    await DeleteTag(tagName)
    await loadTags()
    
    // Refresh accounts as tags might have been removed from them
    await loadAccounts()
  } catch (error) {
    console.error('Failed to delete tag:', error)
    alert('删除标签失败: ' + error.message)
  }
}

async function loadSettings() {
  try {
    const s = await GetAccountSettings()
    settings.autoChangeMachineID = s.autoChangeMachineID
  } catch (error) {
    console.error('Failed to load settings:', error)
  }
}

async function updateSettings() {
  try {
    await UpdateAccountSettings(settings)
  } catch (error) {
    console.error('Failed to update settings:', error)
    alert('保存设置失败: ' + error.message)
    await loadSettings() // Revert on error
  }
}

function openSettingsDialog() {
  dialogs.showSettingsDialog = true
}

// 事件处理
function handleAccountAdded(account) {
  state.accounts.push(account)
}

function handleAccountRemoved(accountId) {
  const index = state.accounts.findIndex(acc => acc.id === accountId)
  if (index >= 0) {
    state.accounts.splice(index, 1)
  }
  const selectedIndex = state.selectedAccounts.indexOf(accountId)
  if (selectedIndex >= 0) {
    state.selectedAccounts.splice(selectedIndex, 1)
  }
}

function handleAccountSwitched(data) {
  console.log('收到账号切换事件:', data)
  
  // 更新账号状态
  state.accounts.forEach(account => {
    account.isActive = account.id === data.newAccountId
  })
  
  // 显示切换成功的通知
  state.errorMessage = `✅ ${data.message || '账号切换成功！请重启 OpenCode 使新账号生效。'}`
  
  // 5秒后清除消息
  setTimeout(() => {
    state.errorMessage = ''
  }, 5000)
}

function handleQuotaUpdated(accountId, quota) {
  const account = state.accounts.find(acc => acc.id === accountId)
  if (account) {
    account.quota = quota
  }
}

// 账号操作
async function testSwitch(event) {
  if (event) {
    event.preventDefault()
    event.stopPropagation()
  }
  
  try {
    await LogToTerminal('=== 前端测试按钮被点击 ===')
    console.log('=== 测试按钮被点击 ===')
    alert('测试按钮工作正常！即将调用后端...')
    
    if (state.accounts.length > 0) {
      await LogToTerminal('→ 尝试切换第一个账号: ' + state.accounts[0].email)
      console.log('→ 尝试切换第一个账号:', state.accounts[0].email)
      await switchAccount(state.accounts[0])
    } else {
      await LogToTerminal('✗ 没有可切换的账号')
      alert('没有可切换的账号')
    }
  } catch (error) {
    await LogToTerminal('✗ 测试失败: ' + error)
    console.error('✗ 测试失败:', error)
    alert('测试失败: ' + error.message)
  }
}

async function switchAccount(account) {
  console.log('=== 前端: switchAccount 开始 ===')
  console.log('→ 账号 ID:', account.id)
  console.log('→ 账号邮箱:', account.email)
  
  switchingId.value = account.id
  try {
    console.log('→ 调用后端 SwitchKiroAccount...')
    
    await SwitchKiroAccount(account.id)
    
    console.log('✓ 后端调用成功')
    console.log('→ 重新加载账号列表...')
    await loadAccounts()
    
    console.log('✓ 账号列表已重新加载')
  } catch (error) {
    console.error('✗ 切换账号失败:', error)
    state.errorMessage = '❌ 切换账号失败: ' + error.message
    setTimeout(() => {
      state.errorMessage = ''
    }, 5000)
  } finally {
    switchingId.value = null
    console.log('=== 前端: switchAccount 完成 ===')
  }
}

async function refreshAccountQuota(accountId) {
  refreshingId.value = accountId
  
  try {
    await RefreshKiroQuota(accountId)
    await loadAccounts()
  } catch (error) {
    console.error('✗ 刷新失败:', error)
  } finally {
    refreshingId.value = null
  }
}

function openAddDialog() {
  resetAccountForm()
  dialogs.showAddDialog = true
}

function openEditDialog(account) {
  editingAccount.value = account
  accountForm.id = account.id
  accountForm.displayName = account.displayName
  accountForm.notes = account.notes || ''
  accountForm.tags = account.tags ? [...account.tags] : []
  dialogs.showEditDialog = true
}

function resetAccountForm() {
  accountForm.id = ''
  accountForm.displayName = ''
  accountForm.notes = ''
  accountForm.loginMethod = 'token'
  accountForm.provider = 'google'
  accountForm.refreshToken = ''
  accountForm.email = ''
  accountForm.password = ''
  accountForm.tags = []
  editingAccount.value = null
  
  // Clear validation errors
  formErrors.email = ''
  formErrors.password = ''
  formErrors.refreshToken = ''
}

async function saveAccount() {
  // 清除之前的错误消息
  state.errorMessage = ''
  
  if (state.saving) {
    return
  }
  
  state.saving = true
  
  try {
    if (editingAccount.value) {
      // Editing existing account - no validation needed for basic info
      const updates = {
        displayName: accountForm.displayName,
        notes: accountForm.notes,
        tags: accountForm.tags
      }
      await UpdateKiroAccount(accountForm.id, updates)
      
      const account = state.accounts.find(acc => acc.id === accountForm.id)
      if (account) {
        Object.assign(account, updates)
      }
      
      dialogs.showEditDialog = false
    } else {
      // Adding new account - validate form
      if (!validateForm()) {
        // 显示验证错误
        if (formErrors.refreshToken) {
          state.errorMessage = formErrors.refreshToken
        } else if (formErrors.email) {
          state.errorMessage = formErrors.email
        } else if (formErrors.password) {
          state.errorMessage = formErrors.password
        } else {
          state.errorMessage = '请填写必填项'
        }
        return
      }
      
      const data = {
        displayName: accountForm.displayName,
        notes: accountForm.notes,
        tags: accountForm.tags
      }
      
      if (accountForm.loginMethod === 'token') {
        data.refreshToken = accountForm.refreshToken.trim()
      } else if (accountForm.loginMethod === 'oauth') {
        await startOAuthFlow()
        return
      } else if (accountForm.loginMethod === 'password') {
        data.email = accountForm.email.trim()
        data.password = accountForm.password
        
        // Check for duplicate account
        if (checkDuplicateAccount(data.email)) {
          state.errorMessage = '该邮箱账号已存在'
          formErrors.email = '该邮箱账号已存在'
          return
        }
      }
      
      // 显示正在添加的提示
      state.errorMessage = '正在添加账号，请稍候...'
      
      await AddKiroAccount(accountForm.loginMethod, data)
      
      // 成功后关闭对话框
      dialogs.showAddDialog = false
      state.errorMessage = ''
      await loadAccounts()
    }
    
    resetAccountForm()
  } catch (error) {
    // Provide user-friendly error messages
    let errorMessage = '❌ 保存账号失败'
    
    // 获取完整的错误信息
    const fullError = error?.message || error?.toString() || String(error)
    
    if (fullError) {
      if (fullError.includes('临时封禁') || fullError.includes('SUSPENDED') || fullError.includes('suspended')) {
        errorMessage = '❌ 账号已被临时封禁：AWS 检测到异常活动并锁定了您的账号。请联系 AWS 支持团队恢复访问：https://support.aws.amazon.com/#/contacts/kiro'
      } else if (fullError.includes('invalid') || fullError.includes('unauthorized') || fullError.includes('刷新 Token 失败') || fullError.includes('Token 失败') || fullError.includes('Bad credentials')) {
        errorMessage = '❌ 认证失败：Refresh Token 无效或已过期，请重新获取'
      } else if (fullError.includes('network') || fullError.includes('timeout')) {
        errorMessage = '❌ 网络错误：请检查网络连接'
      } else if (fullError.includes('duplicate') || fullError.includes('已存在')) {
        errorMessage = '❌ 该账号已存在'
      } else {
        // 显示完整的错误信息
        errorMessage = '❌ ' + fullError
      }
    }
    
    // 显示错误消息在界面上
    state.errorMessage = errorMessage
  } finally {
    state.saving = false
  }
}

async function startOAuthFlow() {
  try {
    state.errorMessage = '正在打开授权页面...'
    const authUrl = await StartKiroOAuth(accountForm.provider)
    
    // 设置 OAuth 等待状态
    oauthState.isWaitingForCallback = true
    oauthState.authUrl = authUrl
    oauthState.callbackUrl = ''
    state.errorMessage = ''
    
    console.log('OAuth flow started, waiting for callback URL...')
  } catch (error) {
    console.error('Failed to start OAuth flow:', error)
    state.errorMessage = '启动 OAuth 认证失败: ' + error.message
  }
}

// 完成 OAuth 流程
async function completeOAuthFlow() {
  if (!oauthState.callbackUrl) {
    state.errorMessage = '请粘贴授权完成后的回调 URL'
    return
  }
  
  // 验证 URL 格式
  if (!oauthState.callbackUrl.includes('code=') || !oauthState.callbackUrl.includes('state=')) {
    state.errorMessage = 'URL 格式无效，请确保复制完整的回调地址（包含 code 和 state 参数）'
    return
  }
  
  state.saving = true
  state.errorMessage = '正在验证并添加账号...'
  
  try {
    await CompleteKiroOAuthWithURL(oauthState.callbackUrl)
    
    // 成功后重置状态
    oauthState.isWaitingForCallback = false
    oauthState.callbackUrl = ''
    oauthState.authUrl = ''
    dialogs.showAddDialog = false
    state.errorMessage = ''
    
    await loadAccounts()
    console.log('OAuth flow completed successfully')
  } catch (error) {
    console.error('Failed to complete OAuth flow:', error)
    state.errorMessage = '验证失败: ' + (error.message || error)
  } finally {
    state.saving = false
  }
}

// 取消 OAuth 流程
function cancelOAuthFlow() {
  oauthState.isWaitingForCallback = false
  oauthState.callbackUrl = ''
  oauthState.authUrl = ''
  state.errorMessage = ''
}

function askDeleteAccount(account) {
  deleteTarget.value = account
  dialogs.showDeleteDialog = true
}

async function confirmDeleteAccount() {
  if (!deleteTarget.value) return
  
  try {
    await RemoveKiroAccount(deleteTarget.value.id)
    dialogs.showDeleteDialog = false
    deleteTarget.value = null
  } catch (error) {
    console.error('Failed to delete account:', error)
    alert('删除账号失败: ' + error.message)
  }
}

function openTagManager() {
  dialogs.showTagManager = true
}

function toggleAccountTag(tagName) {
  const index = accountForm.tags.indexOf(tagName)
  if (index >= 0) {
    accountForm.tags.splice(index, 1)
  } else {
    accountForm.tags.push(tagName)
  }
}

// 选择操作
function toggleSelectAccount(accountId) {
  const index = state.selectedAccounts.indexOf(accountId)
  if (index >= 0) {
    state.selectedAccounts.splice(index, 1)
  } else {
    state.selectedAccounts.push(accountId)
  }
}

function selectAllAccounts() {
  if (state.selectedAccounts.length === filteredAccounts.value.length) {
    state.selectedAccounts = []
  } else {
    state.selectedAccounts = filteredAccounts.value.map(acc => acc.id)
  }
}

// 工具函数
function formatDate(dateString) {
  return new Date(dateString).toLocaleString()
}

function getQuotaPercentage(quota) {
  if (quota.total === 0) return 0
  return Math.round((quota.used / quota.total) * 100)
}

function getQuotaColor(percentage) {
  if (percentage >= 90) return 'var(--red)'
  if (percentage >= 70) return 'var(--yellow)'
  return 'var(--green)'
}

function getSubscriptionLabel(type) {
  const labels = {
    'free': 'Free',
    'pro': 'Pro',
    'pro_plus': 'Pro+'
  }
  return labels[type] || type
}

// 表单验证函数
function validateEmail() {
  formErrors.email = ''
  
  if (!accountForm.email) {
    formErrors.email = '邮箱地址不能为空'
    return false
  }
  
  // Email format validation
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  if (!emailRegex.test(accountForm.email)) {
    formErrors.email = '请输入有效的邮箱地址'
    return false
  }
  
  return true
}

function validatePassword() {
  formErrors.password = ''
  
  if (!accountForm.password) {
    formErrors.password = '密码不能为空'
    return false
  }
  
  if (accountForm.password.length < 6) {
    formErrors.password = '密码长度至少为 6 个字符'
    return false
  }
  
  return true
}

function validateRefreshToken() {
  formErrors.refreshToken = ''
  
  if (!accountForm.refreshToken) {
    formErrors.refreshToken = 'Refresh Token 不能为空'
    return false
  }
  
  // Basic token format validation
  const trimmedToken = accountForm.refreshToken.trim()
  if (trimmedToken.length < 20) {
    formErrors.refreshToken = 'Token 格式无效，长度过短'
    return false
  }
  
  return true
}

function validateForm() {
  let isValid = true
  
  // Clear all errors first
  formErrors.email = ''
  formErrors.password = ''
  formErrors.refreshToken = ''
  
  // Validate based on login method
  if (accountForm.loginMethod === 'token') {
    isValid = validateRefreshToken()
  } else if (accountForm.loginMethod === 'password') {
    const emailValid = validateEmail()
    const passwordValid = validatePassword()
    isValid = emailValid && passwordValid
  }
  // OAuth doesn't need validation as it's handled by the provider
  
  return isValid
}

function checkDuplicateAccount(email) {
  return state.accounts.some(account => 
    account.email.toLowerCase() === email.toLowerCase()
  )
}
</script>

<template>
  <div class="kiro-account-manager">
    <!-- 成功/错误消息提示 -->
    <div v-if="state.errorMessage" :class="['message-banner', state.errorMessage.includes('✅') ? 'success' : 'error']">
      {{ state.errorMessage }}
    </div>
    
    <!-- 头部工具栏 -->
    <div class="manager-header">
      <div class="header-content">
        <div class="header-title">
          <h1>Kiro 账号管理 [MYAPP-DEV]</h1>
        </div>
        <div class="header-actions">
          <button class="btn-settings" @click="openSettingsDialog" title="设置">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/>
              <circle cx="12" cy="12" r="3"/>
            </svg>
          </button>
          <button class="btn-add" @click="openAddDialog">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <path d="M12 5v14M5 12h14"/>
            </svg>
            <span>添加账号</span>
          </button>
        </div>
      </div>
    </div>

    <!-- 搜索和筛选 -->
    <div class="search-bar">
      <div class="search-wrapper">
        <svg class="search-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8"/>
          <path d="m21 21-4.35-4.35"/>
        </svg>
        <input 
          v-model="state.searchQuery" 
          type="text" 
          placeholder="搜索邮箱或备注..." 
          class="search-input"
        >
      </div>
      <select v-model="state.sortBy" class="sort-dropdown">
        <option value="lastUsed">最近使用</option>
        <option value="name">名称排序</option>
        <option value="quota">配额排序</option>
      </select>
      <!-- 测试按钮 -->
      <button type="button" @click.prevent="testSwitch" style="margin-left: 10px; padding: 8px 16px; background: #f00; color: #fff; border: none; border-radius: 4px; cursor: pointer;">
        测试切换
      </button>
    </div>

    <!-- 账号列表 -->
    <div class="accounts-container">
      <div v-if="state.loading" class="loading-state">
        <div class="loading-spinner"></div>
        <span>加载账号中...</span>
      </div>
      
      <div v-else-if="filteredAccounts.length === 0" class="empty-state">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
          <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
          <circle cx="12" cy="7" r="4"/>
        </svg>
        <h3>{{ state.searchQuery || state.filterTag ? '未找到匹配的账号' : '暂无账号' }}</h3>
        <p>{{ state.searchQuery || state.filterTag ? '尝试调整搜索条件' : '点击"添加账号"开始管理您的 Kiro 账号' }}</p>
      </div>

      <div v-else class="accounts-grid">
        <div 
          v-for="account in filteredAccounts" 
          :key="account.id"
          :class="['account-card', { 
            active: account.isActive,
            banned: account.status === 'banned',
            expired: account.status === 'expired'
          }]"
        >
          <!-- 选择框 -->
          <div class="card-checkbox">
            <input 
              type="checkbox" 
              :checked="state.selectedAccounts.includes(account.id)"
              @change="toggleSelectAccount(account.id)"
            >
          </div>

          <!-- 状态标签 -->
          <div class="card-status">
            <span v-if="account.isActive" class="status-badge status-active">当前使用</span>
            <span v-else-if="account.status === 'banned'" class="status-badge status-banned">已封禁</span>
            <span v-else-if="account.status === 'expired'" class="status-badge status-expired">已过期</span>
            <span v-else class="status-badge status-normal">正常</span>
          </div>

          <!-- 头像和邮箱 -->
          <div class="card-header">
            <div class="account-avatar">
              <div class="avatar-placeholder">
                {{ account.email.charAt(0).toUpperCase() }}
              </div>
            </div>
            <div class="account-info">
              <div class="account-email" :title="account.email">{{ account.email }}</div>
              <div class="account-label">{{ account.displayName || '无备注' }}</div>
            </div>
          </div>

          <!-- 订阅类型 -->
          <div class="card-subscription">
            <span :class="['sub-badge', account.subscriptionType]">
              {{ getSubscriptionLabel(account.subscriptionType) }}
            </span>
            <span class="last-used">{{ formatDate(account.lastUsed) }}</span>
          </div>

          <!-- 配额信息 -->
          <div class="card-quota" v-if="account.quota && account.quota.main">
            <div class="quota-header">
              <span class="quota-label">使用量</span>
              <span class="quota-text">
                {{ account.quota.main.used + (account.quota.trial?.used || 0) + (account.quota.reward?.used || 0) }} / 
                {{ account.quota.main.total + (account.quota.trial?.total || 0) + (account.quota.reward?.total || 0) }}
              </span>
            </div>
            <div class="quota-bar">
              <div 
                class="quota-fill" 
                :style="{ 
                  width: getQuotaPercentage({
                    used: account.quota.main.used + (account.quota.trial?.used || 0) + (account.quota.reward?.used || 0),
                    total: account.quota.main.total + (account.quota.trial?.total || 0) + (account.quota.reward?.total || 0)
                  }) + '%',
                  backgroundColor: getQuotaColor(getQuotaPercentage({
                    used: account.quota.main.used + (account.quota.trial?.used || 0) + (account.quota.reward?.used || 0),
                    total: account.quota.main.total + (account.quota.trial?.total || 0) + (account.quota.reward?.total || 0)
                  }))
                }"
              ></div>
            </div>
            <div class="quota-details-text" style="font-size: 0.8em; color: #666; margin-top: 4px;">
              (Main: {{account.quota.main.used}}/{{account.quota.main.total}}, 
               Trial: {{account.quota.trial?.used || 0}}/{{account.quota.trial?.total || 0}}, 
               Reward: {{account.quota.reward?.used || 0}}/{{account.quota.reward?.total || 0}})
            </div>
          </div>

          <!-- 操作按钮 -->
          <div class="card-actions">
            <button type="button" class="btn-action btn-switch" @click="() => { console.log('按钮被点击了！'); switchAccount(account); }" :disabled="switchingId === account.id" :title="account.isActive ? '重新应用到系统' : '切换账号'">
              <svg v-if="switchingId === account.id" class="animate-spin" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 12a9 9 0 11-6.219-8.56"/>
              </svg>
              <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M17 1l4 4-4 4"/>
                <path d="M3 11V9a4 4 0 0 1 4-4h14"/>
                <path d="M7 23l-4-4 4-4"/>
                <path d="M21 13v2a4 4 0 0 1-4 4H3"/>
              </svg>
            </button>
            <button 
              type="button"
              class="btn-action btn-refresh" 
              @click.stop.prevent="refreshAccountQuota(account.id)" 
              :disabled="refreshingId === account.id" 
              title="刷新配额"
            >
              <svg v-if="refreshingId === account.id" class="animate-spin" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 12a9 9 0 11-6.219-8.56"/>
              </svg>
              <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21.5 2v6h-6M2.5 22v-6h6M2 11.5a10 10 0 0 1 18.8-4.3M22 12.5a10 10 0 0 1-18.8 4.2"/>
              </svg>
            </button>
            <button class="btn-action btn-edit" @click.stop="openEditDialog(account)" title="编辑备注">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
              </svg>
            </button>
            <button class="btn-action btn-delete" @click.stop="askDeleteAccount(account)" title="删除">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
              </svg>
            </button>
          </div>
        </div>

        <!-- 添加账号卡片 -->
        <button class="add-account-card" @click="openAddDialog">
          <div class="add-icon">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 5v14M5 12h14"/>
            </svg>
          </div>
          <span>添加账号</span>
        </button>
      </div>
    </div>

    <!-- 标签管理对话框 -->
    <div v-if="dialogs.showTagManager" class="dialog-overlay" @click.self="dialogs.showTagManager = false">
      <div class="dialog tag-manager-dialog">
        <div class="dialog-header">
          <h3>标签管理</h3>
          <button class="btn-close" @click="dialogs.showTagManager = false">×</button>
        </div>
        
        <div class="dialog-content">
          <!-- 添加新标签 -->
          <div class="tag-form">
            <div class="form-group">
              <label>新建标签</label>
              <div class="new-tag-row">
                <input 
                  type="text" 
                  v-model="tagForm.name" 
                  placeholder="标签名称"
                  class="tag-name-input"
                >
                <input 
                  type="color" 
                  v-model="tagForm.color"
                  class="tag-color-input"
                  title="选择颜色"
                >
                <button class="btn-primary" @click="saveTag" :disabled="!tagForm.name">添加</button>
              </div>
            </div>
            <div class="form-group">
              <input 
                type="text" 
                v-model="tagForm.description" 
                placeholder="描述（可选）"
              >
            </div>
          </div>

          <!-- 现有标签列表 -->
          <div class="tags-list-section">
            <h4>现有标签</h4>
            <div v-if="state.tags.length === 0" class="empty-tags">
              暂无标签
            </div>
            <div v-else class="tags-grid">
              <div v-for="tag in state.tags" :key="tag.name" class="tag-item">
                <div class="tag-preview" :style="{ borderColor: tag.color, backgroundColor: tag.color + '15', color: tag.color }">
                  {{ tag.name }}
                </div>
                <div class="tag-desc">{{ tag.description || '无描述' }}</div>
                <button class="btn-icon danger sm" @click="removeTag(tag.name)" title="删除标签">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M18 6L6 18M6 6l12 12"/>
                  </svg>
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 设置对话框 -->
    <div v-if="dialogs.showSettingsDialog" class="dialog-overlay" @click.self="dialogs.showSettingsDialog = false">
      <div class="dialog settings-dialog">
        <div class="dialog-header">
          <h3>Kiro 账号设置</h3>
          <button class="btn-close" @click="dialogs.showSettingsDialog = false">×</button>
        </div>
        
        <div class="dialog-content">
          <div class="settings-section">
            <h4>机器标识 (Machine ID)</h4>
            <div class="setting-item">
              <div class="setting-info">
                <div class="setting-label">自动切换机器码</div>
                <div class="setting-desc">开启后，切换 Kiro 账号时会自动更新系统的 machineId、sqmId 和 deviceId，实现账号间的完全隔离。</div>
              </div>
              <label class="switch">
                <input type="checkbox" v-model="settings.autoChangeMachineID" @change="updateSettings">
                <span class="slider round"></span>
              </label>
            </div>
            
            <div class="info-box">
              <div class="info-icon">ℹ️</div>
              <div class="info-text">
                <p>注意：修改机器码可能会导致其他绑定了当前机器码的软件需要重新激活。Kiro 账号通常绑定特定的机器码，切换账号时保持机器码一致可能会导致账号关联风险。</p>
              </div>
            </div>
          </div>
        </div>
        
        <div class="dialog-footer">
          <button class="btn-primary" @click="dialogs.showSettingsDialog = false">关闭</button>
        </div>
      </div>
    </div>

    <!-- 添加账号对话框 -->
    <div v-if="dialogs.showAddDialog" class="dialog-overlay" @click.self="dialogs.showAddDialog = false">
      <div class="dialog add-account-dialog">
        <div class="dialog-header">
          <h3>添加 Kiro 账号</h3>
          <button class="btn-close" @click="dialogs.showAddDialog = false">×</button>
        </div>
        
        <div class="dialog-content">
          <!-- 错误提示区域 -->
          <div v-if="state.errorMessage" class="error-banner">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/>
              <path d="M12 8v4M12 16h.01"/>
            </svg>
            <span>{{ state.errorMessage }}</span>
            <button class="btn-close-error" @click="state.errorMessage = ''">×</button>
          </div>

          <!-- 登录方式选择 -->
          <div class="login-methods">
            <label class="method-option" :class="{ active: accountForm.loginMethod === 'token' }">
              <input type="radio" v-model="accountForm.loginMethod" value="token">
              <div class="method-content">
                <div class="method-icon">🔑</div>
                <div class="method-info">
                  <div class="method-name">Refresh Token</div>
                  <div class="method-desc">输入刷新令牌，自动获取访问令牌</div>
                </div>
              </div>
            </label>
            
            <label class="method-option" :class="{ active: accountForm.loginMethod === 'oauth' }">
              <input type="radio" v-model="accountForm.loginMethod" value="oauth">
              <div class="method-content">
                <div class="method-icon">🌐</div>
                <div class="method-info">
                  <div class="method-name">OAuth 登录</div>
                  <div class="method-desc">通过第三方服务认证</div>
                </div>
              </div>
            </label>

            <label class="method-option" :class="{ active: accountForm.loginMethod === 'password' }">
              <input type="radio" v-model="accountForm.loginMethod" value="password">
              <div class="method-content">
                <div class="method-icon">🔐</div>
                <div class="method-info">
                  <div class="method-name">用户名密码</div>
                  <div class="method-desc">使用邮箱和密码登录</div>
                </div>
              </div>
            </label>
          </div>

          <!-- Token 登录表单 -->
          <div v-if="accountForm.loginMethod === 'token'" class="form-section">
            <div class="form-group">
              <label>Refresh Token *</label>
              <textarea 
                v-model="accountForm.refreshToken" 
                placeholder="粘贴您的 Refresh Token（刷新令牌）..."
                rows="4"
                required
                :class="{ 'input-error': formErrors.refreshToken }"
                @blur="validateRefreshToken"
              ></textarea>
              <span v-if="formErrors.refreshToken" class="error-message">{{ formErrors.refreshToken }}</span>
              <div class="form-hint">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10"/>
                  <path d="M12 16v-4M12 8h.01"/>
                </svg>
                <span>输入 Refresh Token 后，系统将自动获取 Bearer Token 和用户信息</span>
              </div>
            </div>
          </div>

          <!-- OAuth 登录表单 -->
          <div v-if="accountForm.loginMethod === 'oauth'" class="form-section">
            <!-- 步骤 1: 选择提供商并开始授权 -->
            <div v-if="!oauthState.isWaitingForCallback" class="form-group">
              <label>OAuth 提供商</label>
              <div class="provider-options">
                <label class="provider-option" :class="{ active: accountForm.provider === 'google' }">
                  <input type="radio" v-model="accountForm.provider" value="google">
                  <div class="provider-content">
                    <div class="provider-icon">🔍</div>
                    <span>Google</span>
                  </div>
                </label>
                <label class="provider-option" :class="{ active: accountForm.provider === 'github' }">
                  <input type="radio" v-model="accountForm.provider" value="github">
                  <div class="provider-content">
                    <div class="provider-icon">🐙</div>
                    <span>GitHub</span>
                  </div>
                </label>
                <label class="provider-option" :class="{ active: accountForm.provider === 'builderid' }">
                  <input type="radio" v-model="accountForm.provider" value="builderid">
                  <div class="provider-content">
                    <div class="provider-icon">☁️</div>
                    <span>AWS Builder ID</span>
                  </div>
                </label>
              </div>
            </div>
            
            <!-- 步骤 2: 等待回调 URL -->
            <div v-if="oauthState.isWaitingForCallback" class="oauth-callback-section">
              <div class="oauth-step-indicator">
                <div class="step completed">1. 选择提供商 ✓</div>
                <div class="step active">2. 等待授权回调</div>
              </div>
              
              <div class="info-box warning">
                <div class="info-icon">⚠️</div>
                <div class="info-text">
                  <p><strong>获取授权码的终极秘籍</strong></p>
                  <p>1. 完成授权后，Kiro 页面可能会<strong>瞬间跳转</strong>导致你看不清地址栏。</p>
                  <p>2. <strong>最简单的方法：</strong> 在跳转后的那个报错页面，按下 <code>F12</code> 打开浏览器控制台，点击下方按钮复制脚本并粘贴回控制台回车。</p>
                  <div class="magic-action-row">
                    <button class="btn-magic-tool" @click="copyMagicSnippet">
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"/>
                        <rect x="8" y="2" width="8" height="4" rx="1" ry="1"/>
                      </svg>
                      复制一键捕获脚本
                    </button>
                    <span v-if="magicCopied" class="magic-copied-hint">已复制！</span>
                  </div>
                  <p>3. 或者，完成后立即按 <code>Esc</code> 键停止页面加载，然后手动复制 URL 粘贴到下方。</p>
                </div>
              </div>
              
              <div class="form-group">
                <label>回调 URL *</label>
                <textarea 
                  v-model="oauthState.callbackUrl" 
                  placeholder="粘贴授权完成后的回调 URL..."
                  rows="3"
                  class="callback-url-input"
                ></textarea>
                <div class="form-hint">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="10"/>
                    <path d="M12 16v-4M12 8h.01"/>
                  </svg>
                  <span>从浏览器地址栏复制完整的 URL 并粘贴在这里</span>
                </div>
              </div>
              
              <div class="oauth-actions">
                <button 
                  class="btn-secondary" 
                  @click="cancelOAuthFlow"
                  type="button"
                >
                  取消
                </button>
                <button 
                  class="btn-primary" 
                  @click="completeOAuthFlow"
                  :disabled="state.saving || !oauthState.callbackUrl"
                  type="button"
                >
                  {{ state.saving ? '验证中...' : '完成认证' }}
                </button>
              </div>
            </div>
          </div>

          <!-- 密码登录表单 -->
          <div v-if="accountForm.loginMethod === 'password'" class="form-section">
            <div class="form-group">
              <label>邮箱地址 *</label>
              <input 
                v-model="accountForm.email" 
                type="email" 
                placeholder="your.email@example.com"
                required
                :class="{ 'input-error': formErrors.email }"
                @blur="validateEmail"
              >
              <span v-if="formErrors.email" class="error-message">{{ formErrors.email }}</span>
            </div>
            <div class="form-group">
              <label>密码 *</label>
              <input 
                v-model="accountForm.password" 
                type="password" 
                placeholder="输入您的密码"
                required
                :class="{ 'input-error': formErrors.password }"
                @blur="validatePassword"
              >
              <span v-if="formErrors.password" class="error-message">{{ formErrors.password }}</span>
            </div>
          </div>

          <!-- 通用信息 -->
          <div class="form-section">
            <div class="form-group">
              <label>显示名称</label>
              <input type="text" v-model="accountForm.displayName" placeholder="自定义显示名称（可选）">
            </div>
            <div class="form-group">
              <label>备注</label>
              <textarea v-model="accountForm.notes" placeholder="账号备注信息（可选）..." rows="2"></textarea>
            </div>
          </div>
        </div>
        
        <div class="dialog-footer">
          <button class="btn-secondary" @click="dialogs.showAddDialog = false">取消</button>
          <button class="btn-primary" @click="saveAccount" :disabled="state.saving">{{ state.saving ? '添加中...' : '添加账号' }}</button>
        </div>
      </div>
    </div>

    <!-- 编辑账号对话框 -->
    <div v-if="dialogs.showEditDialog" class="dialog-overlay" @click.self="dialogs.showEditDialog = false">
      <div class="dialog">
        <div class="dialog-header">
          <h3>编辑账号</h3>
          <button class="btn-close" @click="dialogs.showEditDialog = false">×</button>
        </div>
        
        <div class="dialog-content">
          <div class="form-group">
            <label>显示名称</label>
            <input type="text" v-model="accountForm.displayName" placeholder="自定义显示名称">
          </div>
          <div class="form-group">
            <label>标签</label>
            <div class="tags-input">
              <div class="selected-tags" v-if="accountForm.tags.length > 0">
                <span 
                  v-for="tagName in accountForm.tags" 
                  :key="tagName" 
                  class="tag"
                  :style="{ 
                    borderColor: state.tags.find(t => t.name === tagName)?.color || '#3B82F6',
                    backgroundColor: (state.tags.find(t => t.name === tagName)?.color || '#3B82F6') + '15',
                    color: state.tags.find(t => t.name === tagName)?.color || '#3B82F6'
                  }"
                >
                  {{ tagName }}
                  <button @click.stop="toggleAccountTag(tagName)">×</button>
                </span>
              </div>
              <select 
                @change="e => { if(e.target.value) toggleAccountTag(e.target.value); e.target.value = ''; }"
              >
                <option value="">选择添加标签...</option>
                <option 
                  v-for="tag in state.tags" 
                  :key="tag.name" 
                  :value="tag.name"
                  :disabled="accountForm.tags.includes(tag.name)"
                >
                  {{ tag.name }}
                </option>
              </select>
            </div>
          </div>
          <div class="form-group">
            <label>备注</label>
            <textarea v-model="accountForm.notes" placeholder="账号备注信息..." rows="3"></textarea>
          </div>
        </div>
        
        <div class="dialog-footer">
          <button class="btn-secondary" @click="dialogs.showEditDialog = false">取消</button>
          <button class="btn-primary" @click="saveAccount">保存</button>
        </div>
      </div>
    </div>

    <!-- 删除确认对话框 -->
    <div v-if="dialogs.showDeleteDialog" class="dialog-overlay" @click.self="dialogs.showDeleteDialog = false">
      <div class="dialog confirm-dialog">
        <div class="dialog-header">
          <h3>确认删除</h3>
        </div>
        <div class="dialog-content">
          <p>确定要删除账号 "{{ deleteTarget?.displayName }}" 吗？</p>
          <p class="warning-text">此操作不可撤销。</p>
        </div>
        <div class="dialog-footer">
          <button class="btn-secondary" @click="dialogs.showDeleteDialog = false">取消</button>
          <button class="btn-danger" @click="confirmDeleteAccount">删除</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.message-banner {
  padding: 12px 20px;
  margin-bottom: 16px;
  border-radius: 8px;
  font-size: 14px;
  line-height: 1.5;
  animation: slideDown 0.3s ease-out;
}

.message-banner.success {
  background: rgba(16, 185, 129, 0.1);
  border: 1px solid rgba(16, 185, 129, 0.3);
  color: #10b981;
}

.message-banner.error {
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: #ef4444;
}

@keyframes slideDown {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.kiro-account-manager {
  display: flex;
  flex-direction: column;
  height: 100%;
  width: 100%;
  min-width: 0;
  background: var(--bg-base);
  position: relative;
  overflow: hidden;
}

/* 头部 - 简洁专业 */
.manager-header {
  background: var(--bg-surface);
  border-bottom: 1px solid var(--border-subtle);
}

.header-content {
  padding: 14px 20px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-title h1 {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
  letter-spacing: -0.01em;
}

.header-actions {
  display: flex;
  gap: 10px;
  align-items: center;
}

.btn-settings {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  background: transparent;
  border: none;
  border-radius: 8px;
  color: var(--text-muted);
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-settings:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
  transform: scale(1.05);
}

.btn-add {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  background: linear-gradient(135deg, #8b5cf6, #7c3aed);
  color: white;
  border: none;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
  box-shadow: 0 2px 8px rgba(139, 92, 246, 0.25);
}

.btn-add:hover {
  background: linear-gradient(135deg, #7c3aed, #6d28d9);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(139, 92, 246, 0.35);
}

/* 搜索栏 - 精致设计 */
.search-bar {
  background: var(--bg-surface);
  border-bottom: 1px solid var(--border-subtle);
  padding: 12px 20px;
  display: flex;
  gap: 10px;
  align-items: center;
}

.search-wrapper {
  position: relative;
  flex: 1;
  max-width: 380px;
}

.search-icon {
  position: absolute;
  left: 11px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-muted);
  pointer-events: none;
}

.search-input {
  width: 100%;
  padding: 7px 12px 7px 34px;
  border: 1px solid var(--border-default);
  border-radius: 8px;
  background: var(--bg-base);
  color: var(--text-primary);
  font-size: 13px;
  transition: all 0.2s ease;
}

.search-input:focus {
  outline: none;
  border-color: #8b5cf6;
  background: var(--bg-surface);
  box-shadow: 0 0 0 3px rgba(139, 92, 246, 0.1);
}

.search-input::placeholder {
  color: var(--text-muted);
}

.sort-dropdown {
  padding: 7px 12px;
  border: 1px solid var(--border-default);
  border-radius: 8px;
  background: var(--bg-base);
  color: var(--text-primary);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s ease;
  min-width: 130px;
  font-weight: 500;
}

.sort-dropdown:focus {
  outline: none;
  border-color: #8b5cf6;
  background: var(--bg-surface);
  box-shadow: 0 0 0 3px rgba(139, 92, 246, 0.1);
}

.sort-dropdown:hover {
  border-color: var(--border-default);
  background: var(--bg-surface);
}

/* 账号列表容器 */
.accounts-container {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 20px;
  background: var(--bg-base);
  width: 100%;
  min-width: 0;
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

.empty-state svg {
  margin-bottom: 20px;
  opacity: 0.4;
  color: var(--text-muted);
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
  max-width: 400px;
}

.accounts-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-width: 1200px;
  margin: 0 auto;
  padding-bottom: 24px;
}

/* 网格布局 - 多列布局 */
.accounts-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
  padding-bottom: 24px;
  width: 100%;
  max-width: 100%;
}

/* 账号卡片 - 专业设计 */
.account-card {
  position: relative;
  background: var(--bg-elevated);
  border: 1px solid var(--border-subtle);
  border-radius: 12px;
  padding: 16px;
  transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
  display: flex;
  flex-direction: column;
  min-height: auto;
  overflow: hidden;
}

.account-card::before {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: 14px;
  padding: 1px;
  background: linear-gradient(135deg, transparent, transparent);
  -webkit-mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
  -webkit-mask-composite: xor;
  mask-composite: exclude;
  opacity: 0;
  transition: opacity 0.25s ease;
}

.account-card:hover {
  border-color: rgba(139, 92, 246, 0.4);
  box-shadow: 0 8px 24px rgba(139, 92, 246, 0.12);
  transform: translateY(-2px);
}

.account-card:hover::before {
  opacity: 1;
  background: linear-gradient(135deg, #8b5cf6, #7c3aed);
}

.account-card.active {
  border-color: rgba(16, 185, 129, 0.5);
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.04), transparent);
  box-shadow: 0 0 0 1px rgba(16, 185, 129, 0.3), 0 4px 16px rgba(16, 185, 129, 0.15);
}

.account-card.active::before {
  opacity: 1;
  background: linear-gradient(135deg, #10b981, #059669);
}

.account-card.banned {
  border-color: rgba(239, 68, 68, 0.4);
  background: linear-gradient(135deg, rgba(239, 68, 68, 0.04), transparent);
  box-shadow: 0 0 0 1px rgba(239, 68, 68, 0.2);
  opacity: 0.85;
}

.account-card.banned:hover {
  border-color: rgba(239, 68, 68, 0.5);
  box-shadow: 0 0 0 1px rgba(239, 68, 68, 0.3), 0 8px 24px rgba(239, 68, 68, 0.15);
}

.account-card.banned::before {
  opacity: 1;
  background: linear-gradient(135deg, #ef4444, #dc2626);
}

.account-card.expired {
  border-color: rgba(245, 158, 11, 0.4);
  background: linear-gradient(135deg, rgba(245, 158, 11, 0.04), transparent);
  opacity: 0.9;
}

.account-card.expired::before {
  opacity: 1;
  background: linear-gradient(135deg, #f59e0b, #d97706);
}

/* 状态标签 - 精致徽章 */
.card-status {
  position: absolute;
  top: 11px;
  right: 11px;
  z-index: 2;
}

.status-badge {
  display: inline-flex;
  align-items: center;
  padding: 3px 9px;
  border-radius: 10px;
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.6px;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.1);
}

.status-badge.status-active {
  background: linear-gradient(135deg, #10b981, #059669);
  color: white;
  box-shadow: 0 2px 10px rgba(16, 185, 129, 0.35);
  animation: pulse-green 2s ease-in-out infinite;
}

@keyframes pulse-green {
  0%, 100% { box-shadow: 0 2px 10px rgba(16, 185, 129, 0.35); }
  50% { box-shadow: 0 2px 16px rgba(16, 185, 129, 0.5); }
}

.status-badge.status-banned {
  background: linear-gradient(135deg, #ef4444, #dc2626);
  color: white;
  box-shadow: 0 2px 10px rgba(239, 68, 68, 0.35);
}

.status-badge.status-expired {
  background: linear-gradient(135deg, #f59e0b, #d97706);
  color: white;
  box-shadow: 0 2px 10px rgba(245, 158, 11, 0.35);
}

.status-badge.status-normal {
  background: rgba(107, 114, 128, 0.15);
  color: var(--text-secondary);
  font-weight: 600;
}

/* 选择框 */
.card-checkbox {
  position: absolute;
  top: 12px;
  left: 12px;
  z-index: 1;
}

.card-checkbox input[type="checkbox"] {
  width: 16px;
  height: 16px;
  cursor: pointer;
  accent-color: var(--accent-primary);
  border-radius: 4px;
}

/* 卡片头部 - 优雅布局 */
.card-header {
  display: flex;
  gap: 11px;
  margin-top: 30px;
  margin-bottom: 11px;
}

.account-avatar {
  width: 38px;
  height: 38px;
  flex-shrink: 0;
}

.avatar-placeholder {
  width: 100%;
  height: 100%;
  background: linear-gradient(135deg, #8b5cf6, #7c3aed);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  font-size: 15px;
  border-radius: 11px;
  box-shadow: 0 2px 8px rgba(139, 92, 246, 0.25);
}

.account-info {
  flex: 1;
  min-width: 0;
}

.account-email {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 3px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  letter-spacing: -0.01em;
}

.account-label {
  font-size: 11px;
  color: var(--text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 500;
}

/* 订阅类型 - 精美徽章 */
.card-subscription {
  display: flex;
  align-items: center;
  gap: 7px;
  margin-bottom: 11px;
}

.sub-badge {
  display: inline-flex;
  align-items: center;
  padding: 4px 9px;
  border-radius: 9px;
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.08);
}

.sub-badge.free {
  background: rgba(107, 114, 128, 0.12);
  color: var(--text-secondary);
  font-weight: 600;
}

.sub-badge.pro {
  background: linear-gradient(135deg, #3b82f6, #6366f1);
  color: white;
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
}

.sub-badge.pro_plus {
  background: linear-gradient(135deg, #a855f7, #ec4899);
  color: white;
  box-shadow: 0 2px 8px rgba(168, 85, 247, 0.3);
}

.last-used {
  font-size: 10px;
  color: var(--text-muted);
  font-weight: 500;
}

/* 配额区域 - 精致展示 */
.card-quota {
  flex: 1;
  padding: 11px;
  background: rgba(139, 92, 246, 0.04);
  border: 1px solid rgba(139, 92, 246, 0.08);
  border-radius: 11px;
  margin-bottom: 11px;
}

.quota-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 7px;
}

.quota-label {
  font-size: 10px;
  color: var(--text-muted);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.quota-percent {
  font-size: 11px;
  font-weight: 700;
  color: #10b981;
}

.quota-percent.high {
  color: #ef4444;
}

.quota-bar {
  height: 7px;
  background: rgba(0, 0, 0, 0.08);
  border-radius: 10px;
  overflow: hidden;
  margin-bottom: 7px;
  box-shadow: inset 0 1px 2px rgba(0, 0, 0, 0.1);
}

.quota-fill {
  height: 100%;
  transition: width 0.6s cubic-bezier(0.4, 0, 0.2, 1);
  border-radius: 10px;
  position: relative;
  overflow: hidden;
}

.quota-fill::after {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.3), transparent);
  animation: shimmer 2s infinite;
}

@keyframes shimmer {
  0% { transform: translateX(-100%); }
  100% { transform: translateX(100%); }
}

.quota-text {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 11px;
}

.quota-used {
  font-weight: 700;
  color: var(--text-primary);
}

.quota-remaining {
  color: var(--text-muted);
  font-weight: 500;
}

/* 操作按钮 - 精致交互 */
.card-actions {
  display: flex;
  gap: 6px;
  padding-top: 11px;
  border-top: 1px solid rgba(139, 92, 246, 0.1);
  margin-top: auto;
}

.btn-action {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 8px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
}

.btn-action::before {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: 8px;
  background: currentColor;
  opacity: 0;
  transition: opacity 0.2s ease;
}

.btn-action:hover::before {
  opacity: 0.1;
}

.btn-action:hover {
  color: var(--text-primary);
  transform: translateY(-1px) scale(1.05);
}

.btn-action:active {
  transform: translateY(0) scale(0.98);
}

.btn-action:disabled {
  opacity: 0.35;
  cursor: not-allowed;
  transform: none;
}

.btn-action:disabled:hover::before {
  opacity: 0;
}

.btn-switch {
  color: #3b82f6;
}

.btn-switch:hover:not(:disabled) {
  color: #2563eb;
}

.btn-refresh {
  color: #8b5cf6;
}

.btn-refresh:hover {
  color: #7c3aed;
}

.btn-edit {
  color: #f59e0b;
}

.btn-edit:hover {
  color: #d97706;
}

.btn-delete {
  color: #ef4444;
}

.btn-delete:hover {
  color: #dc2626;
}

/* 添加账号卡片 */
.add-account-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  min-height: 280px;
  border: 2px dashed var(--border-subtle);
  border-radius: 16px;
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  transition: all 0.2s ease;
}

.add-account-card:hover {
  border-color: var(--accent-primary);
  background: rgba(176, 128, 255, 0.05);
  color: var(--accent-primary);
}

.add-icon {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: var(--bg-hover);
}

.add-account-card:hover .add-icon {
  background: rgba(176, 128, 255, 0.1);
}

.add-account-card span {
  font-size: 13px;
  font-weight: 500;
}

/* 动画 */
@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.animate-spin {
  animation: spin 1s linear infinite;
}

/* 按钮样式 */
.btn-primary, .btn-secondary, .btn-danger {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
  border: none;
  position: relative;
  overflow: hidden;
}

.btn-primary {
  background: linear-gradient(135deg, var(--accent-primary), var(--accent-hover));
  color: white;
  box-shadow: 0 2px 8px rgba(176, 128, 255, 0.3);
}

.btn-primary:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(176, 128, 255, 0.4);
}

.btn-secondary {
  background: var(--bg-elevated);
  color: var(--text-secondary);
  border: 1px solid var(--border-default);
}

.btn-secondary:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
  border-color: var(--border-subtle);
}

.btn-danger {
  background: linear-gradient(135deg, var(--red), #ff6b6b);
  color: white;
  box-shadow: 0 2px 8px rgba(255, 128, 128, 0.3);
}

.btn-danger:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(255, 128, 128, 0.4);
}

/* 对话框 */
.dialog-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(8px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
  animation: fadeIn 0.2s ease;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.dialog {
  width: 520px;
  max-width: 90vw;
  max-height: 85vh;
  background: var(--bg-surface);
  border-radius: 12px;
  border: 1px solid var(--border-default);
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.4);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  animation: slideUp 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
  z-index: 10000;
}

@keyframes slideUp {
  from { 
    opacity: 0;
    transform: translateY(20px) scale(0.95);
  }
  to { 
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

.add-account-dialog {
  width: 640px;
}

.confirm-dialog {
  width: 440px;
}

.dialog-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  border-bottom: 1px solid var(--border-subtle);
  background: var(--bg-elevated);
}

.dialog-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
}

.btn-close {
  background: none;
  border: none;
  color: var(--text-muted);
  font-size: 24px;
  cursor: pointer;
  padding: 8px;
  border-radius: 6px;
  transition: all 0.2s ease;
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.btn-close:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.dialog-content {
  flex: 1;
  padding: 24px;
  overflow-y: auto;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 20px 24px;
  border-top: 1px solid var(--border-subtle);
  background: var(--bg-elevated);
}

/* 表单样式 */
.form-section {
  margin-bottom: 24px;
}

.form-group {
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
  margin-bottom: 6px;
}

.form-group input, .form-group textarea, .form-group select {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--border-default);
  border-radius: 6px;
  background: var(--bg-elevated);
  color: var(--text-primary);
  font-size: 14px;
  font-family: inherit;
  transition: all 0.2s ease;
}

.form-group input:focus, .form-group textarea:focus, .form-group select:focus {
  outline: none;
  border-color: var(--accent-primary);
  box-shadow: 0 0 0 3px rgba(176, 128, 255, 0.1);
}

.form-group textarea {
  resize: vertical;
  min-height: 80px;
}

.form-hint {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-top: 8px;
  padding: 10px 12px;
  background: rgba(176, 128, 255, 0.08);
  border: 1px solid rgba(176, 128, 255, 0.2);
  border-radius: 6px;
  font-size: 12px;
  color: var(--text-secondary);
  line-height: 1.5;
}

.form-hint svg {
  flex-shrink: 0;
  margin-top: 2px;
  color: var(--accent-primary);
}

/* 表单验证错误样式 */
.input-error {
  border-color: var(--red) !important;
  background: rgba(255, 128, 128, 0.05);
}

.input-error:focus {
  box-shadow: 0 0 0 3px rgba(255, 128, 128, 0.15) !important;
}

.error-message {
  display: block;
  margin-top: 6px;
  font-size: 12px;
  color: var(--red);
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 4px;
}

.error-message::before {
  content: '⚠';
  font-size: 14px;
}

/* 登录方式选择 */
.login-methods {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-bottom: 24px;
}

.method-option {
  display: flex;
  align-items: center;
  padding: 16px;
  border: 1px solid var(--border-subtle);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s ease;
  position: relative;
}

.method-option:hover {
  border-color: var(--accent-primary);
  background: rgba(176, 128, 255, 0.03);
}

.method-option.active {
  border-color: var(--accent-primary);
  background: rgba(176, 128, 255, 0.08);
  box-shadow: 0 0 0 3px rgba(176, 128, 255, 0.1);
}

.method-option input[type="radio"] {
  margin-right: 16px;
  accent-color: var(--accent-primary);
}

.method-content {
  display: flex;
  align-items: center;
  gap: 12px;p: 12px;
}

.method-icon {
  font-size: 24px;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-hover);
  border-radius: 8px;
}

.method-info {
  flex: 1;
}

.method-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 2px;
}

.method-desc {
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.3;
}

/* OAuth 提供商选择 */
.provider-options {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
}

.provider-option {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px 12px;
  border: 1px solid var(--border-subtle);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s ease;
  position: relative;
}

.provider-option:hover {
  border-color: var(--accent-primary);
  background: rgba(176, 128, 255, 0.03);
  transform: translateY(-1px);
}

.provider-option.active {
  border-color: var(--accent-primary);
  background: rgba(176, 128, 255, 0.08);
  box-shadow: 0 0 0 3px rgba(176, 128, 255, 0.1);
}

.provider-option input[type="radio"] {
  display: none;
}

.provider-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.provider-icon {
  font-size: 20px;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-hover);
  border-radius: 6px;
}

.provider-content span {
  font-size: 12px;
  font-weight: 500;
  color: var(--text-secondary);
}

.provider-option.active .provider-content span {
  color: var(--text-primary);
}

/* 标签输入 */
.tags-input {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.selected-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  min-height: 32px;
  padding: 8px;
  border: 1px solid var(--border-default);
  border-radius: 6px;
  background: var(--bg-elevated);
}

.tags-input select {
  margin-top: 4px;
}

/* 警告文本 */
.warning-text {
  color: var(--red);
  font-size: 13px;
  margin: 12px 0;
  padding: 8px 12px;
  background: rgba(255, 128, 128, 0.1);
  border: 1px solid rgba(255, 128, 128, 0.2);
  border-radius: 6px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.warning-text::before {
  content: '⚠️';
  font-size: 16px;
}

/* 响应式设计 */
@media (max-width: 1400px) {
  .accounts-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 900px) {
  .accounts-grid {
    grid-template-columns: 1fr;
  }
  
  .manager-filters .filter-group {
    flex-wrap: wrap;
  }
  
  .search-input {
    min-width: 200px;
  }
}

@media (max-width: 640px) {
  .manager-header {
    padding: 16px;
    flex-direction: column;
    gap: 12px;
    align-items: stretch;
  }
  
  .header-title {
    justify-content: center;
  }
  
  .manager-filters {
    padding: 12px 16px;
  }
  
  .filter-group {
    flex-direction: column;
    align-items: stretch;
  }
  
  .search-input {
    min-width: auto;
  }
  
  .accounts-container {
    padding: 16px;
  }
  
  .account-card {
    flex-direction: column;
  }
  
  .card-actions {
    flex-direction: row;
    border-left: none;
    border-top: 1px solid var(--border-subtle);
    padding: 16px 20px;
  }
  
  .dialog {
    width: 95vw;
    margin: 20px;
  }
  
  .provider-options {
    grid-template-columns: 1fr;
  }
}

/* 无障碍访问 */
@media (prefers-reduced-motion: reduce) {
  * {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}

/* 高对比度模式 */
@media (prefers-contrast: high) {
  .account-card {
    border-width: 2px;
  }
  
  .btn-primary, .btn-danger {
    border: 2px solid currentColor;
  }
}

/* 深色模式优化 */
@media (prefers-color-scheme: dark) {
  .quota-fill::after {
    opacity: 0.3;
  }
  
  .account-card:hover {
    box-shadow: 0 8px 32px rgba(176, 128, 255, 0.2);
  }
}

/* 焦点样式 */
.btn-primary:focus-visible,
.btn-secondary:focus-visible,
.btn-danger:focus-visible,
.btn-icon:focus-visible,
.btn-switch:focus-visible {
  outline: 2px solid var(--accent-primary);
  outline-offset: 2px;
}

.search-input:focus-visible,
.filter-select:focus-visible {
  outline: 2px solid var(--accent-primary);
  outline-offset: 2px;
}

/* 加载状态优化 */
.loading-state {
  background: radial-gradient(circle at center, rgba(176, 128, 255, 0.05), transparent);
}

/* 卡片悬停效果增强 */
.account-card {
  position: relative;
  overflow: visible;
}

.account-card::after {
  content: '';
  position: absolute;
  inset: -2px;
  border-radius: 14px;
  background: linear-gradient(135deg, var(--accent-primary), var(--accent-hover));
  z-index: -1;
  opacity: 0;
  transition: opacity 0.3s ease;
}

.account-card.active::after {
  opacity: 0.1;
}

/* 配额颜色主题 */
.quota-fill[style*="var(--green)"] {
  background: linear-gradient(90deg, var(--green), #66ff99);
}

.quota-fill[style*="var(--yellow)"] {
  background: linear-gradient(90deg, var(--yellow), #ffdb66);
}

.quota-fill[style*="var(--red)"] {
  background: linear-gradient(90deg, var(--red), #ff9999);
}

/* 设置对话框 */
.settings-dialog {
  width: 520px;
}

.settings-section {
  margin-bottom: 24px;
}

.settings-section h4 {
  margin: 0 0 16px;
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
}

.setting-item {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  padding: 16px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-subtle);
  border-radius: 8px;
  margin-bottom: 16px;
}

.setting-info {
  flex: 1;
}

.setting-label {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.setting-desc {
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.4;
}

/* 开关样式 */
.switch {
  position: relative;
  display: inline-block;
  width: 44px;
  height: 24px;
  flex-shrink: 0;
}

.switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: var(--bg-hover);
  transition: .3s;
  border: 1px solid var(--border-default);
}

.slider:before {
  position: absolute;
  content: "";
  height: 18px;
  width: 18px;
  left: 2px;
  bottom: 2px;
  background-color: white;
  transition: .3s;
  box-shadow: 0 1px 3px rgba(0,0,0,0.2);
}

.slider.round {
  border-radius: 24px;
}

.slider.round:before {
  border-radius: 50%;
}

input:checked + .slider {
  background-color: var(--accent-primary);
  border-color: var(--accent-primary);
}

input:focus + .slider {
  box-shadow: 0 0 1px var(--accent-primary);
}

input:checked + .slider:before {
  transform: translateX(20px);
}

/* 错误横幅 */
.error-banner {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 8px;
  margin-bottom: 16px;
  color: #dc2626;
  font-size: 14px;
  animation: slideDown 0.3s ease-out;
}

.error-banner svg {
  flex-shrink: 0;
  color: #dc2626;
}

.error-banner span {
  flex: 1;
  line-height: 1.5;
}

.btn-close-error {
  background: none;
  border: none;
  color: #dc2626;
  font-size: 20px;
  cursor: pointer;
  padding: 0;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  transition: background 0.2s;
}

.btn-close-error:hover {
  background: rgba(239, 68, 68, 0.1);
}

@keyframes slideDown {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* 信息提示框 */
.info-box {
  display: flex;
  gap: 12px;
  padding: 12px 16px;
  background: rgba(176, 128, 255, 0.05);
  border: 1px solid rgba(176, 128, 255, 0.15);
  border-radius: 8px;
}

.info-icon {
  font-size: 18px;
  flex-shrink: 0;
  margin-top: 2px;
}

.info-text p {
  margin: 0;
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.5;
}

.info-text p + p {
  margin-top: 6px;
}

.info-box.warning {
  background: rgba(251, 191, 36, 0.1);
  border: 1px solid rgba(251, 191, 36, 0.3);
}

.info-box.warning .info-text strong {
  color: var(--yellow, #fbbf24);
}

/* 标签管理样式 */
.tag-manager-dialog {
  width: 480px;
}

.new-tag-row {
  display: flex;
  gap: 12px;
  align-items: center;
}

.tag-name-input {
  flex: 1;
}

.tag-color-input {
  width: 40px;
  height: 38px;
  padding: 2px;
  border: 1px solid var(--border-default);
  border-radius: 6px;
  background: var(--bg-elevated);
  cursor: pointer;
}

.tags-list-section {
  margin-top: 24px;
  padding-top: 24px;
  border-top: 1px solid var(--border-subtle);
}

.tags-list-section h4 {
  margin: 0 0 16px;
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
}

.tags-grid {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 400px;
  overflow-y: auto;
}

.tag-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-subtle);
  border-radius: 8px;
}

.tag-preview {
  padding: 4px 8px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
  border: 1px solid;
}

.tag-desc {
  flex: 1;
  font-size: 13px;
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.btn-icon.sm {
  width: 28px;
  height: 28px;
  padding: 4px;
}

.empty-tags {
  text-align: center;
  padding: 32px;
  color: var(--text-muted);
  font-size: 14px;
  background: var(--bg-elevated);
  border-radius: 8px;
  border: 1px dashed var(--border-default);
}

/* OAuth 回调输入样式 */
.oauth-callback-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.oauth-step-indicator {
  display: flex;
  gap: 16px;
  padding: 12px;
  background: var(--bg-elevated);
  border-radius: 8px;
}

.oauth-step-indicator .step {
  font-size: 13px;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  gap: 8px;
}

.oauth-step-indicator .step.completed {
  color: var(--green, #10b981);
}

.oauth-step-indicator .step.active {
  color: var(--primary-color, #a855f7);
  font-weight: 600;
}

.callback-url-input {
  font-family: 'SF Mono', 'Menlo', 'Monaco', 'Consolas', monospace;
  font-size: 12px;
  resize: vertical;
}

.info-text .hint {
  font-size: 12px;
  color: var(--text-muted);
  font-family: 'SF Mono', 'Menlo', 'Monaco', 'Consolas', monospace;
  margin-top: 8px !important;
  padding: 8px;
  background: rgba(0, 0, 0, 0.2);
  border-radius: 4px;
  word-break: break-all;
}

/* 魔法脚本按钮 */
.magic-action-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 10px 0;
}

.btn-magic-tool {
  background: rgba(176, 128, 255, 0.2);
  border: 1px solid rgba(176, 128, 255, 0.4);
  color: var(--kiro-purple);
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  transition: all 0.2s;
}

.btn-magic-tool:hover {
  background: rgba(176, 128, 255, 0.3);
  transform: translateY(-1px);
}

.magic-copied-hint {
  font-size: 12px;
  color: var(--kiro-purple);
  animation: fadeIn 0.3s forwards;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateX(-5px); }
  to { opacity: 1; transform: translateX(0); }
}

/* 之前已有的样式保持不变 */
.oauth-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 8px;
}
</style>
