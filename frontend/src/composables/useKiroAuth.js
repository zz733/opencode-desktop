import { ref, reactive, computed } from 'vue'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import {
  StartKiroOAuth, HandleKiroOAuthCallback, ValidateKiroToken
} from '../../wailsjs/go/main/App'

// OAuth 提供商配置
const oauthProviders = {
  google: {
    id: 'google',
    name: 'Google',
    icon: '🔍',
    description: '使用 Google 账号登录'
  },
  github: {
    id: 'github',
    name: 'GitHub',
    icon: '🐙',
    description: '使用 GitHub 账号登录'
  },
  builderid: {
    id: 'builderid',
    name: 'AWS Builder ID',
    icon: '☁️',
    description: '使用 AWS Builder ID 登录'
  }
}

// 认证状态
const authState = reactive({
  isAuthenticating: false,
  currentProvider: null,
  oauthWindow: null,
  error: null,
  lastAuthAttempt: null
})

// Token 验证状态
const tokenValidation = reactive({
  validating: false,
  lastValidation: null,
  validationResults: new Map() // token -> { valid, error, timestamp }
})

// 事件监听器状态
let authEventListenersInitialized = false

/**
 * Kiro 认证管理 Composable
 * 处理 OAuth 流程、Token 验证等认证相关功能
 */
export function useKiroAuth() {
  // 初始化认证事件监听器
  function initAuthEventListeners() {
    if (authEventListenersInitialized) return
    authEventListenersInitialized = true

    // 监听 OAuth 完成事件
    EventsOn('kiro-oauth-complete', (result) => {
      console.log('OAuth complete:', result)
      handleOAuthComplete(result)
    })

    // 监听 OAuth 错误事件
    EventsOn('kiro-oauth-error', (error) => {
      console.error('OAuth error:', error)
      handleOAuthError(error)
    })

    // 监听 Token 验证结果
    EventsOn('kiro-token-validated', (token, result) => {
      console.log('Token validated:', token, result)
      handleTokenValidated(token, result)
    })

    // 监听浏览器消息（OAuth 回调）
    window.addEventListener('message', handleOAuthMessage)
  }

  // 清理认证事件监听器
  function cleanupAuthEventListeners() {
    if (!authEventListenersInitialized) return
    EventsOff('kiro-oauth-complete')
    EventsOff('kiro-oauth-error')
    EventsOff('kiro-token-validated')
    window.removeEventListener('message', handleOAuthMessage)
    authEventListenersInitialized = false
  }

  // 处理 OAuth 消息
  function handleOAuthMessage(event) {
    if (event.data?.type === 'oauth-complete') {
      handleOAuthComplete(event.data)
    } else if (event.data?.type === 'oauth-error') {
      handleOAuthError(event.data.error)
    }
  }

  // 处理 OAuth 完成
  function handleOAuthComplete(result) {
    authState.isAuthenticating = false
    authState.currentProvider = null
    authState.error = null
    
    if (authState.oauthWindow) {
      authState.oauthWindow.close()
      authState.oauthWindow = null
    }
    
    console.log('OAuth authentication completed successfully')
  }

  // 处理 OAuth 错误
  function handleOAuthError(error) {
    authState.isAuthenticating = false
    authState.currentProvider = null
    authState.error = error || 'OAuth authentication failed'
    
    if (authState.oauthWindow) {
      authState.oauthWindow.close()
      authState.oauthWindow = null
    }
    
    console.error('OAuth authentication failed:', error)
  }

  // 处理 Token 验证结果
  function handleTokenValidated(token, result) {
    tokenValidation.validationResults.set(token, {
      valid: result.valid,
      error: result.error,
      timestamp: Date.now()
    })
    tokenValidation.lastValidation = Date.now()
  }

  // 开始 OAuth 认证流程
  async function startOAuthFlow(provider) {
    if (authState.isAuthenticating) {
      throw new Error('Authentication already in progress')
    }

    if (!oauthProviders[provider]) {
      throw new Error(`Unsupported OAuth provider: ${provider}`)
    }

    authState.isAuthenticating = true
    authState.currentProvider = provider
    authState.error = null
    authState.lastAuthAttempt = Date.now()

    try {
      console.log(`Starting OAuth flow for provider: ${provider}`)
      const authUrl = await StartKiroOAuth(provider)
      
      // 打开 OAuth 窗口
      const windowFeatures = 'width=600,height=700,scrollbars=yes,resizable=yes'
      authState.oauthWindow = window.open(authUrl, 'kiro-oauth', windowFeatures)
      
      if (!authState.oauthWindow) {
        throw new Error('Failed to open OAuth window. Please allow popups for this site.')
      }

      // 监听窗口关闭
      const checkClosed = setInterval(() => {
        if (authState.oauthWindow?.closed) {
          clearInterval(checkClosed)
          if (authState.isAuthenticating) {
            // 用户手动关闭了窗口
            handleOAuthError('Authentication cancelled by user')
          }
        }
      }, 1000)

      return authUrl
    } catch (error) {
      console.error('Failed to start OAuth flow:', error)
      authState.isAuthenticating = false
      authState.currentProvider = null
      authState.error = error.message || 'Failed to start OAuth flow'
      throw error
    }
  }

  // 处理 OAuth 回调
  async function handleOAuthCallback(code, provider) {
    try {
      console.log(`Handling OAuth callback for provider: ${provider}`)
      const result = await HandleKiroOAuthCallback(code, provider)
      handleOAuthComplete(result)
      return result
    } catch (error) {
      console.error('Failed to handle OAuth callback:', error)
      handleOAuthError(error.message || 'Failed to handle OAuth callback')
      throw error
    }
  }

  // 验证 Bearer Token
  async function validateToken(token) {
    if (!token || token.trim() === '') {
      throw new Error('Token is required')
    }

    // 检查缓存的验证结果
    const cached = tokenValidation.validationResults.get(token)
    if (cached && (Date.now() - cached.timestamp) < 5 * 60 * 1000) { // 5分钟缓存
      if (cached.valid) {
        return cached
      } else {
        throw new Error(cached.error || 'Token validation failed')
      }
    }

    tokenValidation.validating = true
    
    try {
      console.log('Validating token...')
      const result = await ValidateKiroToken(token)
      
      const validationResult = {
        valid: true,
        userInfo: result,
        timestamp: Date.now()
      }
      
      tokenValidation.validationResults.set(token, validationResult)
      tokenValidation.lastValidation = Date.now()
      
      return validationResult
    } catch (error) {
      console.error('Token validation failed:', error)
      
      const validationResult = {
        valid: false,
        error: error.message || 'Token validation failed',
        timestamp: Date.now()
      }
      
      tokenValidation.validationResults.set(token, validationResult)
      tokenValidation.lastValidation = Date.now()
      
      throw error
    } finally {
      tokenValidation.validating = false
    }
  }

  // 取消当前认证流程
  function cancelAuthentication() {
    if (authState.oauthWindow) {
      authState.oauthWindow.close()
      authState.oauthWindow = null
    }
    
    authState.isAuthenticating = false
    authState.currentProvider = null
    authState.error = 'Authentication cancelled'
  }

  // 清除认证错误
  function clearAuthError() {
    authState.error = null
  }

  // 清除 Token 验证缓存
  function clearTokenValidationCache() {
    tokenValidation.validationResults.clear()
    tokenValidation.lastValidation = null
  }

  // 获取 Token 验证结果
  function getTokenValidationResult(token) {
    return tokenValidation.validationResults.get(token)
  }

  // 计算属性：可用的 OAuth 提供商
  const availableProviders = computed(() => {
    return Object.values(oauthProviders)
  })

  // 计算属性：当前是否正在认证
  const isAuthenticating = computed(() => {
    return authState.isAuthenticating
  })

  // 计算属性：当前认证提供商
  const currentProvider = computed(() => {
    return authState.currentProvider ? oauthProviders[authState.currentProvider] : null
  })

  // 计算属性：是否有认证错误
  const hasAuthError = computed(() => {
    return !!authState.error
  })

  // 计算属性：是否正在验证 Token
  const isValidatingToken = computed(() => {
    return tokenValidation.validating
  })

  // 初始化
  initAuthEventListeners()

  return {
    // 状态
    authState: readonly(authState),
    tokenValidation: readonly(tokenValidation),
    
    // 计算属性
    availableProviders,
    isAuthenticating,
    currentProvider,
    hasAuthError,
    isValidatingToken,
    
    // 方法
    startOAuthFlow,
    handleOAuthCallback,
    validateToken,
    cancelAuthentication,
    clearAuthError,
    clearTokenValidationCache,
    getTokenValidationResult,
    cleanupAuthEventListeners
  }
}