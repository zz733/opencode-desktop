<script setup>
import { ref, onMounted, computed, onUnmounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import KiroAccountManager from './KiroAccountManager.vue'
import KiroAccountDialog from './KiroAccountDialog.vue'
import SkillsManager from './SkillsManager.vue'
import { languages, setLocale } from '../i18n'
import { useTheme } from '../composables/useTheme'
import { EventsEmit } from '../../wailsjs/runtime/runtime'
import { 
  GetMCPConfig, SaveMCPConfig, GetMCPMarket, AddMCPServer, RemoveMCPServer, 
  ToggleMCPServer, OpenMCPConfigFile, GetMCPStatus, ConnectMCPServer, 
  DisconnectMCPServer, GetMCPTools,
  GetOhMyOpenCodeStatus, InstallOhMyOpenCode, UninstallOhMyOpenCode, FixOhMyOpenCode,
  GetAntigravityAuthStatus, InstallAntigravityAuth, UninstallAntigravityAuth, UpdateAntigravityAuth,
  GetKiroAuthStatus, InstallKiroAuth, UninstallKiroAuth, UpdateKiroAuth,
  GetUIUXProMaxStatus, InstallUIUXProMax, UninstallUIUXProMax, UpdateUIUXProMax,
  RestartOpenCode,
  GetRemoteControlInfo
} from '../../wailsjs/go/main/App'
import { BrowserOpenURL, EventsOn } from '../../wailsjs/runtime/runtime'

const { t, locale } = useI18n()
const { currentTheme, themes, setTheme } = useTheme()
const emit = defineEmits(['close', 'open-file', 'runCommand', 'kiro-settings-active'])

const activeCategory = ref('theme')
const mcpConfig = ref({ mcp: {} })
const mcpMarket = ref([])
const mcpStatus = ref({})
const mcpTools = ref([])
const mcpLoading = ref(false)
const showAddDialog = ref(false)
const showToolsDialog = ref(false)
const showConfirmDialog = ref(false)
const confirmTarget = ref(null)
const editingServer = ref(null)
const selectedServerTools = ref(null)
let statusInterval = null

const serverForm = ref({
  name: '', type: 'local', command: '', url: '', enabled: true, environment: {}
})
const envVars = ref([])

// ========== 模型管理 ==========
const defaultModels = [
  { id: 'opencode/big-pickle', name: 'Big Pickle', free: true, builtin: true },
  { id: 'opencode/grok-code', name: 'Grok Code Fast', free: true, builtin: true },
  { id: 'opencode/minimax-m2.1-free', name: 'MiniMax M2.1', free: true, builtin: true },
  { id: 'opencode/glm-4.7-free', name: 'GLM 4.7', free: true, builtin: true },
  { id: 'opencode/gpt-5-nano', name: 'GPT 5 Nano', free: true, builtin: true },
  { id: 'opencode/kimi-k2', name: 'Kimi K2', free: false, builtin: true },
  { id: 'opencode/claude-opus-4-5', name: 'Claude Opus 4.5', free: false, builtin: true },
  { id: 'opencode/claude-sonnet-4-5', name: 'Claude Sonnet 4.5', free: false, builtin: true },
  { id: 'opencode/gpt-5.1-codex', name: 'GPT 5.1 Codex', free: false, builtin: true },
]

const customModels = ref(JSON.parse(localStorage.getItem('customModels') || '[]'))
const showModelDialog = ref(false)
const showModelConfirmDialog = ref(false)
const modelConfirmTarget = ref(null)
const editingModel = ref(null)
const modelForm = ref({ id: '', name: '', free: true, baseUrl: '', apiKey: '', supportsImage: false })

function saveCustomModels() {
  localStorage.setItem('customModels', JSON.stringify(customModels.value))
  // 通知其他组件模型列表已更新
  EventsEmit('models-updated')
}

function openModelDialog(model = null) {
  if (model) {
    editingModel.value = model.id
    modelForm.value = { 
      id: model.id, 
      name: model.name, 
      free: model.free,
      baseUrl: model.baseUrl || '',
      apiKey: model.apiKey || '',
      supportsImage: model.supportsImage || false
    }
  } else {
    editingModel.value = null
    modelForm.value = { id: '', name: '', free: true, baseUrl: '', apiKey: '', supportsImage: false }
  }
  showModelDialog.value = true
}

function saveModel() {
  if (!modelForm.value.id || !modelForm.value.name) return
  
  const model = {
    id: modelForm.value.id,
    name: modelForm.value.name,
    free: modelForm.value.free,
    baseUrl: modelForm.value.baseUrl,
    apiKey: modelForm.value.apiKey,
    supportsImage: modelForm.value.supportsImage,
    builtin: false
  }
  
  if (editingModel.value) {
    // 编辑现有模型
    const index = customModels.value.findIndex(m => m.id === editingModel.value)
    if (index >= 0) {
      customModels.value[index] = model
    }
  } else {
    // 添加新模型
    customModels.value.push(model)
  }
  
  saveCustomModels()
  showModelDialog.value = false
}

function askRemoveModel(model) {
  modelConfirmTarget.value = model
  showModelConfirmDialog.value = true
}

function confirmRemoveModel() {
  const model = modelConfirmTarget.value
  showModelConfirmDialog.value = false
  modelConfirmTarget.value = null
  
  const index = customModels.value.findIndex(m => m.id === model.id)
  if (index >= 0) {
    customModels.value.splice(index, 1)
    saveCustomModels()
  }
}

const allModels = computed(() => [...defaultModels, ...customModels.value])

// ========== 插件管理 ==========
const ohMyOpenCodeStatus = ref({ installed: false, version: '' })
const antigravityAuthStatus = ref({ installed: false, version: '' })
const kiroAuthStatus = ref({ installed: false, version: '' })
const uiuxProMaxStatus = ref({ installed: false, version: '' })
const pluginLoading = ref(false)
const pluginLoadingName = ref('')
const showKiroAccountManager = ref(false) // 控制 Kiro 账号管理器的显示

// ========== 远程控制 ==========
const remoteControlInfo = ref({ active: false, port: 0, token: '', url: '' })
const remoteControlLoading = ref(false)

async function loadRemoteControlInfo() {
  try {
    const info = await GetRemoteControlInfo()
    remoteControlInfo.value = info || { active: false, port: 0, token: '', url: '' }
  } catch (e) {
    console.error('获取远程控制信息失败:', e)
  }
}

async function loadPluginStatus() {
  try {
    ohMyOpenCodeStatus.value = await GetOhMyOpenCodeStatus() || { installed: false, version: '' }
    antigravityAuthStatus.value = await GetAntigravityAuthStatus() || { installed: false, version: '' }
    kiroAuthStatus.value = await GetKiroAuthStatus() || { installed: false, version: '' }
    uiuxProMaxStatus.value = await GetUIUXProMaxStatus() || { installed: false, version: '' }
  } catch (e) {
    console.error('获取插件状态失败:', e)
  }
}

async function installOhMyOpenCode() {
  pluginLoading.value = true
  pluginLoadingName.value = 'oh-my-opencode'
  try {
    await InstallOhMyOpenCode()
    await loadPluginStatus()
  } catch (e) {
    console.error('安装失败:', e)
  } finally {
    pluginLoading.value = false
    pluginLoadingName.value = ''
  }
}

async function uninstallOhMyOpenCode() {
  pluginLoading.value = true
  pluginLoadingName.value = 'oh-my-opencode'
  try {
    await UninstallOhMyOpenCode()
    await loadPluginStatus()
  } catch (e) {
    console.error('卸载失败:', e)
  } finally {
    pluginLoading.value = false
    pluginLoadingName.value = ''
  }
}

async function fixOhMyOpenCode() {
  pluginLoading.value = true
  pluginLoadingName.value = 'oh-my-opencode-fix'
  try {
    await FixOhMyOpenCode()
  } catch (e) {
    console.error('修复失败:', e)
  } finally {
    pluginLoading.value = false
    pluginLoadingName.value = ''
  }
}

async function installAntigravityAuth() {
  pluginLoading.value = true
  pluginLoadingName.value = 'antigravity-auth'
  try {
    await InstallAntigravityAuth()
    await loadPluginStatus()
    // 通知重新加载模型列表
    EventsEmit('antigravity-models-changed', true)
  } catch (e) {
    console.error('安装失败:', e)
  } finally {
    pluginLoading.value = false
    pluginLoadingName.value = ''
  }
}

async function uninstallAntigravityAuth() {
  pluginLoading.value = true
  pluginLoadingName.value = 'antigravity-auth'
  try {
    await UninstallAntigravityAuth()
    await loadPluginStatus()
    // 通知清空模型列表
    EventsEmit('antigravity-models-changed', false)
  } catch (e) {
    console.error('卸载失败:', e)
  } finally {
    pluginLoading.value = false
    pluginLoadingName.value = ''
  }
}

function runAntigravityAuth() {
  // 发送命令到终端执行（不带参数，会显示交互式选择菜单）
  emit('runCommand', 'opencode auth login')
}

async function installKiroAuth() {
  pluginLoading.value = true
  pluginLoadingName.value = 'kiro-auth'
  try {
    await InstallKiroAuth()
    await loadPluginStatus()
    // 通知重新加载模型列表
    EventsEmit('kiro-models-changed', true)
    // 提示用户重启 OpenCode 以确保插件生效
    setTimeout(() => {
      if (confirm('Kiro Auth 插件安装成功！建议重启 OpenCode 以确保插件完全生效。是否现在重启？')) {
        restartOpenCode()
      }
    }, 1000)
  } catch (e) {
    console.error('安装失败:', e)
  } finally {
    pluginLoading.value = false
    pluginLoadingName.value = ''
  }
}

async function uninstallKiroAuth() {
  pluginLoading.value = true
  pluginLoadingName.value = 'kiro-auth'
  try {
    await UninstallKiroAuth()
    await loadPluginStatus()
    // 通知清空模型列表
    EventsEmit('kiro-models-changed', false)
  } catch (e) {
    console.error('卸载失败:', e)
  } finally {
    pluginLoading.value = false
    pluginLoadingName.value = ''
  }
}

function runKiroAuth() {
  // 直接执行认证命令，让用户在终端中手动选择
  emit('runCommand', 'opencode auth login')
}

async function installUIUXProMax() {
  pluginLoading.value = true
  pluginLoadingName.value = 'uiux-pro-max'
  try {
    await InstallUIUXProMax()
    await loadPluginStatus()
  } catch (e) {
    console.error('安装失败:', e)
  } finally {
    pluginLoading.value = false
    pluginLoadingName.value = ''
  }
}

async function updateAntigravityAuth() {
  pluginLoading.value = true
  pluginLoadingName.value = 'antigravity-auth-update'
  try {
    await UpdateAntigravityAuth()
    await loadPluginStatus()
    // 通知重新加载模型列表
    EventsEmit('antigravity-models-changed', true)
  } catch (e) {
    console.error('升级失败:', e)
  } finally {
    pluginLoading.value = false
    pluginLoadingName.value = ''
  }
}

async function updateKiroAuth() {
  pluginLoading.value = true
  pluginLoadingName.value = 'kiro-auth-update'
  try {
    await UpdateKiroAuth()
    await loadPluginStatus()
    // 通知重新加载模型列表
    EventsEmit('kiro-models-changed', true)
  } catch (e) {
    console.error('升级失败:', e)
  } finally {
    pluginLoading.value = false
    pluginLoadingName.value = ''
  }
}

async function updateUIUXProMax() {
  pluginLoading.value = true
  pluginLoadingName.value = 'uiux-pro-max-update'
  try {
    await UpdateUIUXProMax()
    await loadPluginStatus()
  } catch (e) {
    console.error('升级失败:', e)
  } finally {
    pluginLoading.value = false
    pluginLoadingName.value = ''
  }
}

async function uninstallUIUXProMax() {
  pluginLoading.value = true
  pluginLoadingName.value = 'uiux-pro-max'
  try {
    await UninstallUIUXProMax()
    await loadPluginStatus()
  } catch (e) {
    console.error('卸载失败:', e)
  } finally {
    pluginLoading.value = false
    pluginLoadingName.value = ''
  }
}

async function restartOpenCode() {
  pluginLoading.value = true
  pluginLoadingName.value = 'restart'
  try {
    await RestartOpenCode()
    // 重启成功后，等待一段时间让用户看到状态变化
    setTimeout(() => {
      pluginLoading.value = false
      pluginLoadingName.value = ''
    }, 2000)
  } catch (e) {
    console.error('重启失败:', e)
    pluginLoading.value = false
    pluginLoadingName.value = ''
  }
}

const changeLanguage = (code) => setLocale(code)
const changeTheme = (themeId) => setTheme(themeId)

async function loadMCPConfig() {
  mcpLoading.value = true
  try {
    const [config, market] = await Promise.all([
      GetMCPConfig(), GetMCPMarket()
    ])
    mcpConfig.value = config || { mcp: {} }
    mcpMarket.value = market || []
    
    // 获取状态（会自动同步配置到 OpenCode）
    const [status, tools] = await Promise.all([
      GetMCPStatus().catch(() => ({})), GetMCPTools().catch(() => [])
    ])
    mcpStatus.value = status || {}
    mcpTools.value = tools || []
  } catch (e) {
    console.error('加载 MCP 配置失败:', e)
  } finally {
    mcpLoading.value = false
  }
}

async function refreshStatus() {
  try {
    const [status, tools] = await Promise.all([
      GetMCPStatus().catch(() => ({})), GetMCPTools().catch(() => [])
    ])
    mcpStatus.value = status || {}
    mcpTools.value = tools || []
  } catch (e) {}
}

const installedServers = computed(() => {
  return Object.entries(mcpConfig.value.mcp || {}).map(([name, config]) => {
    const apiStatus = mcpStatus.value[name]
    let status = 'unknown'
    let error = ''
    if (apiStatus) {
      status = apiStatus.status || 'unknown'
      error = apiStatus.error || ''
    } else if (config.enabled === false) {
      status = 'disabled'
    }
    return { name, ...config, status, error }
  })
})

const availableServers = computed(() => {
  const installed = new Set(Object.keys(mcpConfig.value.mcp || {}))
  return mcpMarket.value.filter(item => !installed.has(item.name))
})

const groupedMarket = computed(() => {
  const groups = {}
  availableServers.value.forEach(item => {
    const cat = item.category || 'other'
    if (!groups[cat]) groups[cat] = []
    groups[cat].push(item)
  })
  return groups
})

const categoryNames = {
  filesystem: '文件系统', development: '开发工具', database: '数据库',
  automation: '自动化', search: '搜索', network: '网络', memory: '记忆',
  reasoning: '推理', monitoring: '监控', communication: '通讯',
  maps: '地图', testing: '测试', other: '其他'
}

function getServerTools(serverName) {
  return mcpTools.value.filter(t => t.id.startsWith(`mcp_${serverName}_`))
}

function showServerTools(serverName) {
  selectedServerTools.value = { name: serverName, tools: getServerTools(serverName) }
  showToolsDialog.value = true
}

async function installFromMarket(item) {
  try {
    const server = { type: 'local', command: item.command, enabled: true, environment: {} }
    // 如果有环境变量要求，预填空值
    if (item.envVars?.length) {
      item.envVars.forEach(v => { server.environment[v] = '' })
    }
    const status = await AddMCPServer(item.name, server)
    // 更新状态
    if (status) {
      mcpStatus.value = status
    }
    await loadMCPConfig()
    // 安装后自动打开编辑对话框让用户配置
    const installed = installedServers.value.find(s => s.name === item.name)
    if (installed) {
      // 附加配置提示
      installed.configTips = item.configTips
      installed.docsUrl = item.docsUrl
      openAddDialog(installed)
    }
  } catch (e) { console.error('安装失败:', e) }
}

async function toggleServer(name, enabled) {
  try {
    await ToggleMCPServer(name, enabled)
    // ToggleMCPServer 内部已经处理了连接/断开
    await loadMCPConfig()
  } catch (e) { console.error('切换失败:', e) }
}

function askRemoveServer(name) {
  confirmTarget.value = name
  showConfirmDialog.value = true
}

async function confirmRemoveServer() {
  const name = confirmTarget.value
  showConfirmDialog.value = false
  confirmTarget.value = null
  try {
    await DisconnectMCPServer(name).catch(() => {})
    await RemoveMCPServer(name)
    await loadMCPConfig()
  } catch (e) { console.error('删除失败:', e) }
}

function openAddDialog(server = null) {
  if (server) {
    editingServer.value = server.name
    serverForm.value = {
      name: server.name, type: server.type || 'local',
      command: Array.isArray(server.command) ? server.command.join(' ') : '',
      url: server.url || '', enabled: server.enabled !== false,
      environment: server.environment || {}
    }
    envVars.value = Object.entries(server.environment || {}).map(([k, v]) => ({ key: k, value: v }))
    // 查找市场中的配置提示
    const marketItem = mcpMarket.value.find(m => m.name === server.name)
    serverForm.value.configTips = marketItem?.configTips || ''
    serverForm.value.docsUrl = marketItem?.docsUrl || ''
  } else {
    editingServer.value = null
    serverForm.value = { name: '', type: 'local', command: '', url: '', enabled: true, environment: {}, configTips: '', docsUrl: '' }
    envVars.value = []
  }
  showAddDialog.value = true
}

function addEnvVar() { envVars.value.push({ key: '', value: '' }) }
function removeEnvVar(index) { envVars.value.splice(index, 1) }

async function saveServer() {
  if (!serverForm.value.name) return
  const env = {}
  envVars.value.forEach(v => { if (v.key) env[v.key] = v.value })
  const server = { type: serverForm.value.type, enabled: serverForm.value.enabled, environment: env }
  if (serverForm.value.type === 'local') {
    server.command = serverForm.value.command.split(/\s+/).filter(Boolean)
  } else { server.url = serverForm.value.url }
  try {
    if (editingServer.value && editingServer.value !== serverForm.value.name) {
      await RemoveMCPServer(editingServer.value)
    }
    const status = await AddMCPServer(serverForm.value.name, server)
    // 更新状态
    if (status) {
      mcpStatus.value = status
    }
    await loadMCPConfig()
    showAddDialog.value = false
  } catch (e) { console.error('保存失败:', e) }
}

async function openConfigFile() {
  try {
    const path = await OpenMCPConfigFile()
    emit('open-file', path)
  } catch (e) { console.error('打开配置文件失败:', e) }
}

function openDocs(url) {
  if (url) BrowserOpenURL(url)
}

onMounted(() => {
  loadMCPConfig()
  loadPluginStatus()
  loadRemoteControlInfo()
  statusInterval = setInterval(() => {
    refreshStatus()
    loadRemoteControlInfo()
  }, 5000)
})
onUnmounted(() => { if (statusInterval) clearInterval(statusInterval) })

// 监听远程控制启动事件
EventsOn('remote-control-started', (info) => {
  remoteControlInfo.value = info
})

// 复制到剪贴板
function copyToClipboard(text) {
  navigator.clipboard.writeText(text).then(() => {
    // 可以添加一个提示
    console.log('已复制:', text)
  }).catch(err => {
    console.error('复制失败:', err)
  })
}

// 监听 activeCategory 变化，通知父组件是否显示 Kiro 设置
watch(activeCategory, (newValue) => {
  emit('kiro-settings-active', newValue === 'kiro')
})
</script>

<template>
  <aside class="settings-panel">
    <div class="settings-header"><span>{{ t('settings.title') }}</span></div>
    
    <div class="settings-body">
      <!-- 左侧导航 -->
      <div class="settings-nav">
        <div :class="['nav-item', { active: activeCategory === 'theme' }]" @click="activeCategory = 'theme'">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="5"/><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/>
          </svg>
          <span>{{ t('settings.theme') }}</span>
        </div>
        <div :class="['nav-item', { active: activeCategory === 'language' }]" @click="activeCategory = 'language'">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/><path d="M2 12h20M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
          </svg>
          <span>{{ t('settings.language') }}</span>
        </div>
        <div :class="['nav-item', { active: activeCategory === 'models' }]" @click="activeCategory = 'models'">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 2L2 7l10 5 10-5-10-5z"/><path d="M2 17l10 5 10-5"/><path d="M2 12l10 5 10-5"/>
          </svg>
          <span>{{ t('settings.models.title') }}</span>
        </div>
        <div :class="['nav-item', { active: activeCategory === 'mcp' }]" @click="activeCategory = 'mcp'">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="2" y="3" width="20" height="14" rx="2"/><path d="M8 21h8M12 17v4"/>
          </svg>
          <span>MCP</span>
        </div>
        <div :class="['nav-item', { active: activeCategory === 'plugins' }]" @click="activeCategory = 'plugins'">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83"/>
          </svg>
          <span>{{ t('settings.plugins.title') }}</span>
        </div>
        <div :class="['nav-item', { active: activeCategory === 'skills' }]" @click="activeCategory = 'skills'">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 2L2 7l10 5 10-5-10-5z"/><path d="M2 17l10 5 10-5"/><path d="M2 12l10 5 10-5"/>
          </svg>
          <span>技能</span>
        </div>
        <div :class="['nav-item', { active: activeCategory === 'kiro' }]" @click="activeCategory = 'kiro'">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
            <circle cx="12" cy="7" r="4"/>
          </svg>
          <span>Kiro 账号</span>
        </div>
        <div :class="['nav-item', { active: activeCategory === 'remote' }]" @click="activeCategory = 'remote'">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="2" y="7" width="20" height="14" rx="2" ry="2"/>
            <path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16"/>
          </svg>
          <span>远程控制</span>
        </div>
      </div>
      
      <!-- 右侧内容 -->
      <div class="settings-content">
      <!-- 主题设置 -->
      <div v-if="activeCategory === 'theme'" class="settings-section">
        <div class="setting-item">
          <div class="setting-label">{{ t('settings.theme') }}</div>
          <div class="setting-control">
            <select :value="currentTheme" @change="changeTheme($event.target.value)">
              <option v-for="theme in themes" :key="theme.id" :value="theme.id">{{ theme.name }}</option>
            </select>
          </div>
        </div>
      </div>
      
      <!-- 语言设置 -->
      <div v-if="activeCategory === 'language'" class="settings-section">
        <div class="setting-item">
          <div class="setting-label">{{ t('settings.language') }}</div>
          <div class="setting-control">
            <select :value="locale" @change="changeLanguage($event.target.value)">
              <option v-for="lang in languages" :key="lang.code" :value="lang.code">{{ lang.name }}</option>
            </select>
          </div>
        </div>
      </div>
      
      <!-- 模型管理 -->
      <div v-if="activeCategory === 'models'" class="settings-section models-section">
        <div class="section-header">
          <span class="section-title">{{ t('settings.models.custom') }}</span>
          <div class="section-actions">
            <button class="btn-icon" @click="openModelDialog()" :title="t('settings.models.add')">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 5v14M5 12h14"/></svg>
            </button>
          </div>
        </div>
        
        <div v-if="customModels.length === 0" class="empty-state">{{ t('settings.models.noCustom') }}</div>
        
        <div v-else class="model-list">
          <div v-for="model in customModels" :key="model.id" class="model-item">
            <div class="model-info">
              <div class="model-name">
                <span :class="['model-badge', model.free ? 'free' : 'premium']">{{ model.free ? '🆓' : '⭐' }}</span>
                {{ model.name }}
                <span v-if="model.supportsImage" class="model-feature" :title="t('settings.models.supportsImage')">🖼️</span>
              </div>
              <div class="model-id">{{ model.id }}</div>
              <div v-if="model.baseUrl" class="model-url">{{ model.baseUrl }}</div>
            </div>
            <div class="model-actions">
              <button class="btn-icon" @click="openModelDialog(model)" :title="t('common.edit')">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
              </button>
              <button class="btn-icon danger" @click="askRemoveModel(model)" :title="t('common.delete')">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
              </button>
            </div>
          </div>
        </div>
        
        <div class="section-header builtin-header">
          <span class="section-title">{{ t('settings.models.builtin') }}</span>
        </div>
        
        <div class="model-list builtin-list">
          <div v-for="model in defaultModels" :key="model.id" class="model-item builtin">
            <div class="model-info">
              <div class="model-name">
                <span :class="['model-badge', model.free ? 'free' : 'premium']">{{ model.free ? '🆓' : '⭐' }}</span>
                {{ model.name }}
              </div>
              <div class="model-id">{{ model.id }}</div>
            </div>
          </div>
        </div>
      </div>
      
      <div v-if="activeCategory === 'mcp'" class="settings-section mcp-section">
        <div class="section-header">
          <span class="section-title">{{ t('settings.mcp.installed') }}</span>
          <div class="section-actions">
            <button class="btn-icon" @click="openAddDialog()" :title="t('settings.mcp.addManual')">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 5v14M5 12h14"/></svg>
            </button>
            <button class="btn-icon" @click="openConfigFile" :title="t('settings.mcp.editFile')">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 2v6h6M16 13H8M16 17H8M10 9H8"/></svg>
            </button>
          </div>
        </div>
        
        <div v-if="mcpLoading" class="loading">{{ t('common.loading') }}...</div>
        <div v-else-if="installedServers.length === 0" class="empty-state">{{ t('settings.mcp.noInstalled') }}</div>
        
        <div v-else class="server-list">
          <div v-for="server in installedServers" :key="server.name" class="server-item">
            <div class="server-info">
              <div class="server-name">
                <span :class="['status-dot', server.status]" :title="server.error || server.status"></span>
                {{ server.name }}
              </div>
              <div class="server-meta">
                <span class="server-type">{{ server.type === 'remote' ? 'Remote' : 'Local' }}</span>
                <span v-if="server.error" class="server-error" :title="server.error">{{ server.error.substring(0, 30) }}{{ server.error.length > 30 ? '...' : '' }}</span>
                <span v-else-if="getServerTools(server.name).length" class="server-tools" @click="showServerTools(server.name)">
                  {{ getServerTools(server.name).length }} {{ t('settings.mcp.tools') }}
                </span>
              </div>
            </div>
            <div class="server-actions">
              <label class="switch">
                <input type="checkbox" :checked="server.enabled !== false" @change="toggleServer(server.name, $event.target.checked)">
                <span class="slider"></span>
              </label>
              <button class="btn-icon" @click="openAddDialog(server)" :title="t('common.edit')">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
              </button>
              <button class="btn-icon danger" @click="askRemoveServer(server.name)" :title="t('common.delete')">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
              </button>
            </div>
          </div>
        </div>
        
        <div class="section-header market-header">
          <span class="section-title">{{ t('settings.mcp.market') }}</span>
        </div>
        
        <div class="market-list">
          <template v-for="(items, category) in groupedMarket" :key="category">
            <div class="market-category">{{ categoryNames[category] || category }}</div>
            <div v-for="item in items" :key="item.name" class="market-item">
              <div class="market-info">
                <div class="market-name">
                  {{ item.name }}
                  <span v-if="item.docsUrl" class="docs-link" @click="openDocs(item.docsUrl)" :title="t('settings.mcp.viewDocs')">
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>
                      <polyline points="15 3 21 3 21 9"/>
                      <line x1="10" y1="14" x2="21" y2="3"/>
                    </svg>
                  </span>
                </div>
                <div class="market-desc">{{ item.description }}</div>
                <div v-if="item.envVars?.length" class="market-env">{{ t('settings.mcp.requiresEnv') }}: {{ item.envVars.join(', ') }}</div>
              </div>
              <button class="btn-install" @click="installFromMarket(item)">{{ t('settings.mcp.install') }}</button>
            </div>
          </template>
        </div>
      </div>
      
      <!-- 技能管理 -->
      <div v-if="activeCategory === 'skills'" class="settings-section skills-section">
        <SkillsManager />
      </div>
      
      <!-- 远程控制 -->
      <div v-if="activeCategory === 'remote'" class="settings-section remote-section">
        <div class="remote-card">
          <div class="remote-header">
            <div class="remote-icon">📱</div>
            <div class="remote-info">
              <div class="remote-title">OpenCode Mobile 远程控制</div>
              <div class="remote-desc">通过手机浏览器远程控制你的 AI 编程助手</div>
            </div>
          </div>
          
          <div v-if="remoteControlInfo.active" class="remote-body active">
            <div class="connection-info">
              <div class="info-row">
                <span class="info-label">状态</span>
                <span class="status-badge active">🟢 运行中</span>
              </div>
              <div class="info-row">
                <span class="info-label">连接码</span>
                <div class="connection-code">
                  <span class="code-display">{{ remoteControlInfo.token }}</span>
                  <button class="btn-copy" @click="copyToClipboard(remoteControlInfo.token)" title="复制连接码">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
                      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
                    </svg>
                  </button>
                </div>
              </div>
              <div class="info-row">
                <span class="info-label">端口</span>
                <span class="info-value">{{ remoteControlInfo.port }}</span>
              </div>
            </div>
            
            <div class="usage-steps">
              <div class="steps-title">📖 使用步骤</div>
              <ol class="steps-list">
                <li>确保手机和电脑在同一 WiFi 网络</li>
                <li>在手机浏览器打开 OpenCode Mobile</li>
                <li>输入上面显示的 6 位连接码</li>
                <li>开始远程控制</li>
              </ol>
            </div>
          </div>
          
          <div v-else class="remote-body inactive">
            <div class="inactive-message">
              <div class="message-icon">⚠️</div>
              <div class="message-text">远程控制服务未运行</div>
              <div class="message-hint">应用启动时会自动启动远程控制服务</div>
            </div>
          </div>
        </div>
        
        <div class="remote-features">
          <div class="feature-card">
            <div class="feature-icon">💬</div>
            <div class="feature-name">AI 对话</div>
            <div class="feature-desc">在手机上与 AI 助手对话</div>
          </div>
          <div class="feature-card">
            <div class="feature-icon">📁</div>
            <div class="feature-name">文件浏览</div>
            <div class="feature-desc">查看和管理项目文件</div>
          </div>
          <div class="feature-card">
            <div class="feature-icon">💻</div>
            <div class="feature-name">终端查看</div>
            <div class="feature-desc">实时查看终端输出</div>
          </div>
        </div>
      </div>
      
      <!-- 插件管理 -->
      <div v-if="activeCategory === 'plugins'" class="settings-section plugins-section">
        <!-- Oh My OpenCode -->
        <div class="plugin-card">
          <div class="plugin-header">
            <div class="plugin-icon">🚀</div>
            <div class="plugin-info">
              <div class="plugin-name">Oh My OpenCode</div>
              <div class="plugin-desc">{{ t('settings.plugins.ohMyOpenCodeDesc') }}</div>
            </div>
          </div>
          <div class="plugin-features">
            <div class="feature-item">✨ Sisyphus Agent - {{ t('settings.plugins.sisyphusDesc') }}</div>
            <div class="feature-item">🔧 {{ t('settings.plugins.multiAgent') }}</div>
            <div class="feature-item">⚡ {{ t('settings.plugins.ultrawork') }}</div>
            <div class="feature-item">🔌 {{ t('settings.plugins.claudeCompat') }}</div>
          </div>
          <div class="plugin-footer">
            <div v-if="ohMyOpenCodeStatus.installed" class="plugin-status installed">
              <span class="status-badge">✓ {{ t('settings.plugins.installed') }}</span>
              <span v-if="ohMyOpenCodeStatus.version" class="version">v{{ ohMyOpenCodeStatus.version }}</span>
            </div>
            <div v-else class="plugin-status">
              <span class="status-badge not-installed">{{ t('settings.plugins.notInstalled') }}</span>
            </div>
            <div class="plugin-actions">
              <button v-if="!ohMyOpenCodeStatus.installed" class="btn-install" @click="installOhMyOpenCode" :disabled="pluginLoading">
                {{ pluginLoadingName === 'oh-my-opencode' ? t('common.loading') + '...' : t('settings.mcp.install') }}
              </button>
              <template v-else>
                <button class="btn-fix" @click="fixOhMyOpenCode" :disabled="pluginLoading">
                  {{ pluginLoadingName === 'oh-my-opencode-fix' ? t('common.loading') + '...' : t('settings.plugins.fix') }}
                </button>
                <button class="btn-uninstall" @click="uninstallOhMyOpenCode" :disabled="pluginLoading">
                  {{ pluginLoadingName === 'oh-my-opencode' ? t('common.loading') + '...' : t('settings.plugins.uninstall') }}
                </button>
              </template>
              <a class="btn-docs" href="https://github.com/code-yeongyu/oh-my-opencode" target="_blank" @click.prevent="openDocs('https://github.com/code-yeongyu/oh-my-opencode')">
                {{ t('settings.mcp.viewDocs') }}
              </a>
            </div>
          </div>
          <!-- Oh My OpenCode 使用提示 -->
          <div class="plugin-tip-inline">
            <div class="tip-icon">💡</div>
            <div class="tip-content">
              <div class="tip-title">{{ t('settings.plugins.tipTitle') }}</div>
              <div class="tip-text">{{ t('settings.plugins.tipText') }}</div>
            </div>
          </div>
        </div>
        
        <!-- Antigravity Auth -->
        <div class="plugin-card">
          <div class="plugin-header">
            <div class="plugin-icon">🔐</div>
            <div class="plugin-info">
              <div class="plugin-name">Antigravity Auth</div>
              <div class="plugin-desc">{{ t('settings.plugins.antigravityDesc') }}</div>
            </div>
          </div>
          <div class="plugin-features">
            <div class="feature-item">🌐 {{ t('settings.plugins.googleOAuth') }}</div>
            <div class="feature-item">💎 {{ t('settings.plugins.geminiModels') }}</div>
            <div class="feature-item">🤖 {{ t('settings.plugins.claudeModels') }}</div>
            <div class="feature-item">♾️ {{ t('settings.plugins.multiAccount') }}</div>
          </div>
          <div class="plugin-footer">
            <div v-if="antigravityAuthStatus.installed" class="plugin-status installed">
              <span class="status-badge">✓ {{ t('settings.plugins.installed') }}</span>
              <span v-if="antigravityAuthStatus.version" class="version">v{{ antigravityAuthStatus.version }}</span>
            </div>
            <div v-else class="plugin-status">
              <span class="status-badge not-installed">{{ t('settings.plugins.notInstalled') }}</span>
            </div>
            <div class="plugin-actions">
              <button v-if="!antigravityAuthStatus.installed" class="btn-install" @click="installAntigravityAuth" :disabled="pluginLoading">
                {{ pluginLoadingName === 'antigravity-auth' ? t('common.loading') + '...' : t('settings.mcp.install') }}
              </button>
              <template v-else>
                <button class="btn-auth" @click="runAntigravityAuth">
                  {{ t('settings.plugins.authenticate') }}
                </button>
                <button v-if="antigravityAuthStatus.updateAvailable" class="btn-update" @click="updateAntigravityAuth" :disabled="pluginLoading">
                  {{ pluginLoadingName === 'antigravity-auth-update' ? t('settings.plugins.updating') : t('settings.plugins.update') }}
                </button>
                <button class="btn-uninstall" @click="uninstallAntigravityAuth" :disabled="pluginLoading">
                  {{ pluginLoadingName === 'antigravity-auth' ? t('common.loading') + '...' : t('settings.plugins.uninstall') }}
                </button>
              </template>
              <a class="btn-docs" href="https://github.com/NoeFabris/opencode-antigravity-auth" target="_blank" @click.prevent="openDocs('https://github.com/NoeFabris/opencode-antigravity-auth')">
                {{ t('settings.mcp.viewDocs') }}
              </a>
            </div>
          </div>
          <!-- Antigravity Auth 认证提示 -->
          <div class="plugin-tip-inline">
            <div class="tip-icon">🔑</div>
            <div class="tip-content">
              <div class="tip-title">{{ t('settings.plugins.authTipTitle') }}</div>
              <div class="tip-text">{{ t('settings.plugins.authTipText') }}</div>
            </div>
          </div>
        </div>
        
        <!-- Kiro Auth -->
        <div class="plugin-card">
          <div class="plugin-header">
            <div class="plugin-icon">🚀</div>
            <div class="plugin-info">
              <div class="plugin-name">Kiro Auth</div>
              <div class="plugin-desc">{{ t('settings.plugins.kiroDesc') }}</div>
            </div>
          </div>
          <div class="plugin-body">
            <div class="plugin-features">
              <span class="feature-tag">AWS Kiro</span>
              <span class="feature-tag">Claude 4.5</span>
              <span class="feature-tag">550+ Free</span>
            </div>
          </div>
          <div class="plugin-footer">
            <div v-if="kiroAuthStatus.installed" class="plugin-status installed">
              <span class="status-badge">✓ {{ t('settings.plugins.installed') }}</span>
              <span v-if="kiroAuthStatus.version" class="version">v{{ kiroAuthStatus.version }}</span>
            </div>
            <div v-else class="plugin-status">
              <span class="status-badge">{{ t('settings.plugins.notInstalled') }}</span>
            </div>
            <div class="plugin-actions">
              <button v-if="!kiroAuthStatus.installed" class="btn-install" @click="installKiroAuth" :disabled="pluginLoading">
                {{ pluginLoadingName === 'kiro-auth' ? t('common.loading') + '...' : t('settings.mcp.install') }}
              </button>
              <template v-else>
                <button class="btn-auth" @click="showKiroAccountManager = true">
                  账号管理
                </button>
                <button v-if="kiroAuthStatus.updateAvailable" class="btn-update" @click="updateKiroAuth" :disabled="pluginLoading">
                  {{ pluginLoadingName === 'kiro-auth-update' ? t('settings.plugins.updating') : t('settings.plugins.update') }}
                </button>
                <button class="btn-uninstall" @click="uninstallKiroAuth" :disabled="pluginLoading">
                  {{ pluginLoadingName === 'kiro-auth' ? t('common.loading') + '...' : t('settings.plugins.uninstall') }}
                </button>
              </template>
              <a class="btn-docs" href="https://github.com/tickernelz/opencode-kiro-auth" target="_blank" @click.prevent="openDocs('https://github.com/tickernelz/opencode-kiro-auth')">
                {{ t('settings.mcp.viewDocs') }}
              </a>
            </div>
          </div>
        </div>
        
        <!-- UI/UX Pro Max Skill -->
        <div class="plugin-card">
          <div class="plugin-header">
            <div class="plugin-icon">🎨</div>
            <div class="plugin-info">
              <div class="plugin-name">UI/UX Pro Max Skill</div>
              <div class="plugin-desc">{{ t('settings.plugins.uiuxDesc') }}</div>
            </div>
          </div>
          <div class="plugin-features">
            <div class="feature-item">🎨 {{ t('settings.plugins.uiuxStyles') }}</div>
            <div class="feature-item">🎯 {{ t('settings.plugins.uiuxSystem') }}</div>
            <div class="feature-item">📱 {{ t('settings.plugins.uiuxPlatforms') }}</div>
            <div class="feature-item">🏭 {{ t('settings.plugins.uiuxRules') }}</div>
          </div>
          <div class="plugin-footer">
            <div v-if="uiuxProMaxStatus.installed" class="plugin-status installed">
              <span class="status-badge">✓ {{ t('settings.plugins.installed') }}</span>
              <span v-if="uiuxProMaxStatus.version" class="version">v{{ uiuxProMaxStatus.version }}</span>
            </div>
            <div v-else class="plugin-status">
              <span class="status-badge not-installed">{{ t('settings.plugins.notInstalled') }}</span>
            </div>
            <div class="plugin-actions">
              <button v-if="!uiuxProMaxStatus.installed" class="btn-install" @click="installUIUXProMax" :disabled="pluginLoading">
                {{ pluginLoadingName === 'uiux-pro-max' ? t('common.loading') + '...' : t('settings.mcp.install') }}
              </button>
              <template v-else>
                <button v-if="uiuxProMaxStatus.updateAvailable" class="btn-update" @click="updateUIUXProMax" :disabled="pluginLoading">
                  {{ pluginLoadingName === 'uiux-pro-max-update' ? t('settings.plugins.updating') : t('settings.plugins.update') }}
                </button>
                <button class="btn-uninstall" @click="uninstallUIUXProMax" :disabled="pluginLoading">
                  {{ pluginLoadingName === 'uiux-pro-max' ? t('common.loading') + '...' : t('settings.plugins.uninstall') }}
                </button>
              </template>
              <a class="btn-docs" href="https://github.com/nextlevelbuilder/ui-ux-pro-max-skill" target="_blank" @click.prevent="openDocs('https://github.com/nextlevelbuilder/ui-ux-pro-max-skill')">
                {{ t('settings.mcp.viewDocs') }}
              </a>
            </div>
          </div>
          <!-- UI/UX Pro Max 使用提示 -->
          <div class="plugin-tip-inline">
            <div class="tip-icon">💡</div>
            <div class="tip-content">
              <div class="tip-title">{{ t('settings.plugins.uiuxTipTitle') }}</div>
              <div class="tip-text">{{ t('settings.plugins.uiuxTipText') }}</div>
            </div>
          </div>
        </div>
        
        <!-- 重启 OpenCode -->
        <div class="restart-section">
          <button class="btn-restart" @click="restartOpenCode" :disabled="pluginLoading">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M23 4v6h-6"/><path d="M1 20v-6h6"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
            </svg>
            {{ pluginLoadingName === 'restart' ? t('settings.plugins.restarting') : t('settings.plugins.restartOpenCode') }}
          </button>
          <div class="restart-hint">{{ t('settings.plugins.restartHint') }}</div>
        </div>
      </div>
      </div>
    </div>

    <!-- 添加/编辑对话框 -->
    <div v-if="showAddDialog" class="dialog-overlay" @click.self="showAddDialog = false">
      <div class="dialog">
        <div class="dialog-header">
          {{ editingServer ? t('settings.mcp.editServer') : t('settings.mcp.addServer') }}
          <span v-if="serverForm.docsUrl" class="header-docs" @click="openDocs(serverForm.docsUrl)">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>
              <polyline points="15 3 21 3 21 9"/>
              <line x1="10" y1="14" x2="21" y2="3"/>
            </svg>
            {{ t('settings.mcp.viewDocs') }}
          </span>
        </div>
        
        <!-- 配置说明 -->
        <div v-if="serverForm.configTips" class="config-tips">
          <div class="tips-title">{{ t('settings.mcp.configTips') }}</div>
          <pre class="tips-content">{{ serverForm.configTips }}</pre>
        </div>
        
        <div class="dialog-content">
          <div class="form-group">
            <label>{{ t('settings.mcp.serverName') }}</label>
            <input v-model="serverForm.name" type="text" :placeholder="t('settings.mcp.serverNamePlaceholder')">
          </div>
          <div class="form-group">
            <label>{{ t('settings.mcp.serverType') }}</label>
            <select v-model="serverForm.type"><option value="local">Local</option><option value="remote">Remote</option></select>
          </div>
          <div v-if="serverForm.type === 'local'" class="form-group">
            <label>{{ t('settings.mcp.command') }}</label>
            <input v-model="serverForm.command" type="text" placeholder="npx -y @modelcontextprotocol/server-xxx">
          </div>
          <div v-else class="form-group">
            <label>URL</label>
            <input v-model="serverForm.url" type="text" placeholder="https://...">
          </div>
          <div class="form-group">
            <label>{{ t('settings.mcp.envVars') }} <button class="btn-add-env" @click="addEnvVar">+</button></label>
            <div v-for="(env, index) in envVars" :key="index" class="env-row">
              <input v-model="env.key" type="text" :placeholder="t('settings.mcp.envKey')" autocapitalize="off" autocomplete="off" spellcheck="false">
              <input v-model="env.value" type="text" :placeholder="t('settings.mcp.envValue')" autocapitalize="off" autocomplete="off" spellcheck="false">
              <button class="btn-remove-env" @click="removeEnvVar(index)">×</button>
            </div>
          </div>
          <div class="form-group checkbox-group">
            <label><input v-model="serverForm.enabled" type="checkbox"> {{ t('settings.mcp.enabled') }}</label>
          </div>
        </div>
        <div class="dialog-footer">
          <button class="btn-cancel" @click="showAddDialog = false">{{ t('common.cancel') }}</button>
          <button class="btn-save" @click="saveServer">{{ t('common.save') }}</button>
        </div>
      </div>
    </div>
    
    <!-- 工具列表对话框 -->
    <div v-if="showToolsDialog" class="dialog-overlay" @click.self="showToolsDialog = false">
      <div class="dialog tools-dialog">
        <div class="dialog-header">{{ selectedServerTools?.name }} - {{ t('settings.mcp.tools') }}</div>
        <div class="dialog-content">
          <div v-if="!selectedServerTools?.tools?.length" class="empty-state">{{ t('settings.mcp.noTools') }}</div>
          <div v-else class="tools-list">
            <div v-for="tool in selectedServerTools.tools" :key="tool.id" class="tool-item">
              <div class="tool-name">{{ tool.id.replace(`mcp_${selectedServerTools.name}_`, '') }}</div>
              <div class="tool-desc">{{ tool.description }}</div>
            </div>
          </div>
        </div>
        <div class="dialog-footer">
          <button class="btn-cancel" @click="showToolsDialog = false">{{ t('common.close') }}</button>
        </div>
      </div>
    </div>
    
    <!-- 删除确认对话框 -->
    <div v-if="showConfirmDialog" class="dialog-overlay" @click.self="showConfirmDialog = false">
      <div class="dialog confirm-dialog">
        <div class="dialog-header">{{ t('common.confirm') }}</div>
        <div class="dialog-content">
          <p class="confirm-message">{{ t('settings.mcp.confirmDelete', { name: confirmTarget }) }}</p>
        </div>
        <div class="dialog-footer">
          <button class="btn-cancel" @click="showConfirmDialog = false">{{ t('common.cancel') }}</button>
          <button class="btn-danger" @click="confirmRemoveServer">{{ t('common.delete') }}</button>
        </div>
      </div>
    </div>
    
    <!-- 模型添加/编辑对话框 -->
    <div v-if="showModelDialog" class="dialog-overlay" @click.self="showModelDialog = false">
      <div class="dialog model-dialog">
        <div class="dialog-header">
          {{ editingModel ? t('settings.models.edit') : t('settings.models.add') }}
        </div>
        <div class="dialog-content">
          <div class="form-group">
            <label>{{ t('settings.models.modelId') }} <span class="required">*</span></label>
            <input v-model="modelForm.id" type="text" :placeholder="t('settings.models.modelIdPlaceholder')" autocapitalize="off" autocomplete="off" spellcheck="false">
          </div>
          <div class="form-group">
            <label>{{ t('settings.models.modelName') }} <span class="required">*</span></label>
            <input v-model="modelForm.name" type="text" :placeholder="t('settings.models.modelNamePlaceholder')" autocapitalize="off" autocomplete="off" spellcheck="false">
          </div>
          <div class="form-group">
            <label>{{ t('settings.models.baseUrl') }}</label>
            <input v-model="modelForm.baseUrl" type="text" :placeholder="t('settings.models.baseUrlPlaceholder')" autocapitalize="off" autocomplete="off" spellcheck="false">
          </div>
          <div class="form-group">
            <label>{{ t('settings.models.apiKey') }}</label>
            <input v-model="modelForm.apiKey" type="password" :placeholder="t('settings.models.apiKeyPlaceholder')" autocapitalize="off" autocomplete="off" spellcheck="false">
          </div>
          <div class="form-group checkbox-group">
            <label><input v-model="modelForm.supportsImage" type="checkbox"> {{ t('settings.models.supportsImage') }}</label>
          </div>
        </div>
        <div class="dialog-footer">
          <button class="btn-cancel" @click="showModelDialog = false">{{ t('common.cancel') }}</button>
          <button class="btn-save" @click="saveModel">{{ t('common.save') }}</button>
        </div>
      </div>
    </div>
    
    <!-- 模型删除确认对话框 -->
    <div v-if="showModelConfirmDialog" class="dialog-overlay" @click.self="showModelConfirmDialog = false">
      <div class="dialog confirm-dialog">
        <div class="dialog-header">{{ t('common.confirm') }}</div>
        <div class="dialog-content">
          <p class="confirm-message">{{ t('settings.models.confirmDelete', { name: modelConfirmTarget?.name }) }}</p>
        </div>
        <div class="dialog-footer">
          <button class="btn-cancel" @click="showModelConfirmDialog = false">{{ t('common.cancel') }}</button>
          <button class="btn-danger" @click="confirmRemoveModel">{{ t('common.delete') }}</button>
        </div>
      </div>
    </div>
  </aside>
  
  <!-- Kiro 账号管理弹窗 -->
  <KiroAccountDialog v-if="showKiroAccountManager" @close="showKiroAccountManager = false" />
</template>

<style scoped>
.settings-panel { flex: 1; background: var(--bg-surface); display: flex; flex-direction: column; overflow: hidden; }
.settings-header { padding: 12px 16px; font-size: 11px; font-weight: 500; letter-spacing: 0.5px; color: var(--text-secondary); text-transform: uppercase; border-bottom: 1px solid var(--border-subtle); flex-shrink: 0; }

/* 左右布局 */
.settings-body { flex: 1; display: flex; overflow: hidden; }

/* 左侧导航 */
.settings-nav { width: 140px; flex-shrink: 0; display: flex; flex-direction: column; padding: 8px; gap: 2px; border-right: 1px solid var(--border-subtle); overflow-y: auto; }
.nav-item { display: flex; align-items: center; gap: 8px; padding: 8px 10px; border-radius: 4px; cursor: pointer; font-size: 12px; color: var(--text-secondary); transition: all 0.15s; }
.nav-item:hover { background: var(--bg-hover); color: var(--text-primary); }
.nav-item.active { background: var(--accent-primary); color: white; }
.nav-item svg { opacity: 0.7; flex-shrink: 0; }
.nav-item.active svg { opacity: 1; }

/* 右侧内容 */
.settings-content { flex: 1; overflow-y: auto; padding: 12px; max-width: none; min-width: 0; }
.settings-content:has(.kiro-section) { padding: 0; overflow-y: auto; overflow-x: hidden; }
.settings-section { display: flex; flex-direction: column; gap: 8px; }
.section-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.section-title { font-size: 12px; font-weight: 600; color: var(--text-primary); }
.section-actions { display: flex; gap: 4px; }
.setting-item { display: flex; justify-content: space-between; align-items: center; padding: 10px 12px; background: var(--bg-elevated); border-radius: 6px; }
.setting-label { font-size: 13px; color: var(--text-primary); }
.setting-control select { padding: 6px 10px; background: var(--bg-elevated); border: 1px solid var(--border-default); border-radius: 4px; color: var(--text-primary); font-size: 12px; cursor: pointer; outline: none; }
.setting-control select:hover { border-color: var(--text-muted); }
.setting-control select:focus { border-color: var(--accent-primary); }
.mcp-section { gap: 0; }
.btn-icon { display: flex; align-items: center; justify-content: center; width: 28px; height: 28px; background: var(--bg-elevated); border: 1px solid var(--border-subtle); border-radius: 4px; color: var(--text-secondary); cursor: pointer; transition: all 0.15s; }
.btn-icon:hover { background: var(--bg-hover); color: var(--text-primary); }
.btn-icon.danger:hover { background: var(--red); color: white; border-color: var(--red); }
.loading, .empty-state { padding: 20px; text-align: center; color: var(--text-muted); font-size: 12px; }
.server-list { display: flex; flex-direction: column; gap: 6px; margin-bottom: 16px; }
.server-item { display: flex; justify-content: space-between; align-items: center; padding: 10px 12px; background: var(--bg-elevated); border-radius: 6px; border: 1px solid var(--border-subtle); }
.server-info { flex: 1; min-width: 0; }
.server-name { font-size: 13px; font-weight: 500; color: var(--text-primary); display: flex; align-items: center; gap: 8px; }
.status-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--text-muted); flex-shrink: 0; }
.status-dot.connected { background: var(--green); }
.status-dot.disabled, .status-dot.unknown { background: var(--text-muted); }
.status-dot.failed { background: var(--red); }
.status-dot.needs_auth { background: var(--yellow); }
.server-meta { display: flex; gap: 8px; margin-top: 2px; }
.server-type { font-size: 11px; color: var(--text-muted); }
.server-error { font-size: 11px; color: var(--red); cursor: help; }
.server-tools { font-size: 11px; color: var(--accent-primary); cursor: pointer; }
.server-tools:hover { text-decoration: underline; }
.server-actions { display: flex; align-items: center; gap: 8px; }

.switch { position: relative; display: inline-block; width: 36px; height: 20px; }
.switch input { opacity: 0; width: 0; height: 0; }
.slider { position: absolute; cursor: pointer; top: 0; left: 0; right: 0; bottom: 0; background-color: var(--bg-hover); transition: 0.2s; border-radius: 20px; }
.slider:before { position: absolute; content: ""; height: 14px; width: 14px; left: 3px; bottom: 3px; background-color: white; transition: 0.2s; border-radius: 50%; }
input:checked + .slider { background-color: var(--accent-primary); }
input:checked + .slider:before { transform: translateX(16px); }
.market-header { margin-top: 16px; padding-top: 16px; border-top: 1px solid var(--border-subtle); }
.market-list { display: flex; flex-direction: column; gap: 6px; }
.market-category { font-size: 11px; font-weight: 500; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.5px; margin-top: 12px; margin-bottom: 4px; }
.market-category:first-child { margin-top: 0; }
.market-item { display: flex; justify-content: space-between; align-items: center; padding: 10px 12px; background: var(--bg-elevated); border-radius: 6px; border: 1px solid var(--border-subtle); }
.market-info { flex: 1; min-width: 0; }
.market-name { font-size: 13px; font-weight: 500; color: var(--text-primary); display: flex; align-items: center; gap: 6px; }
.docs-link { color: var(--text-muted); cursor: pointer; display: flex; align-items: center; }
.docs-link:hover { color: var(--accent-primary); }
.market-desc { font-size: 11px; color: var(--text-secondary); margin-top: 2px; }
.market-env { font-size: 10px; color: var(--yellow); margin-top: 4px; }
.btn-install { padding: 4px 12px; background: var(--accent-primary); border: none; border-radius: 4px; color: white; font-size: 12px; cursor: pointer; transition: opacity 0.15s; }
.btn-install:hover { opacity: 0.9; }
.dialog-overlay { position: fixed; inset: 0; background: rgba(0, 0, 0, 0.5); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.dialog { width: 400px; max-height: 80vh; background: var(--bg-surface); border-radius: 8px; border: 1px solid var(--border-default); box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3); display: flex; flex-direction: column; }
.tools-dialog { width: 500px; }
.confirm-dialog { width: 320px; }
.confirm-message { font-size: 13px; color: var(--text-primary); margin: 0; text-align: center; }
.btn-danger { padding: 6px 16px; border-radius: 4px; font-size: 12px; cursor: pointer; background: var(--red); border: none; color: white; transition: opacity 0.15s; }
.btn-danger:hover { opacity: 0.9; }
.dialog-header { padding: 16px; font-size: 14px; font-weight: 600; color: var(--text-primary); border-bottom: 1px solid var(--border-subtle); display: flex; justify-content: space-between; align-items: center; }
.header-docs { font-size: 12px; font-weight: 400; color: var(--accent-primary); cursor: pointer; display: flex; align-items: center; gap: 4px; }
.header-docs:hover { text-decoration: underline; }
.config-tips { padding: 12px 16px; background: var(--bg-elevated); border-bottom: 1px solid var(--border-subtle); }
.tips-title { font-size: 11px; font-weight: 600; color: var(--yellow); margin-bottom: 6px; text-transform: uppercase; }
.tips-content { font-size: 12px; color: var(--text-secondary); white-space: pre-wrap; font-family: inherit; margin: 0; line-height: 1.5; }
.dialog-content { padding: 16px; overflow-y: auto; display: flex; flex-direction: column; gap: 12px; }
.form-group { display: flex; flex-direction: column; gap: 6px; }
.form-group label { font-size: 12px; color: var(--text-secondary); display: flex; align-items: center; gap: 8px; }
.form-group input[type="text"], .form-group select { padding: 8px 10px; background: var(--bg-elevated); border: 1px solid var(--border-default); border-radius: 4px; color: var(--text-primary); font-size: 13px; outline: none; }
.form-group input:focus, .form-group select:focus { border-color: var(--accent-primary); }
.checkbox-group label { flex-direction: row; cursor: pointer; }
.checkbox-group input[type="checkbox"] { width: 16px; height: 16px; }
.env-row { display: flex; gap: 8px; align-items: center; }
.env-row input { flex: 1; padding: 6px 8px; background: var(--bg-elevated); border: 1px solid var(--border-default); border-radius: 4px; color: var(--text-primary); font-size: 12px; }
.btn-add-env { padding: 2px 8px; background: var(--accent-primary); border: none; border-radius: 3px; color: white; font-size: 12px; cursor: pointer; }
.btn-remove-env { padding: 4px 8px; background: transparent; border: 1px solid var(--border-default); border-radius: 3px; color: var(--text-muted); cursor: pointer; }
.btn-remove-env:hover { background: var(--red); color: white; border-color: var(--red); }
.dialog-footer { padding: 12px 16px; border-top: 1px solid var(--border-subtle); display: flex; justify-content: flex-end; gap: 8px; }
.btn-cancel, .btn-save { padding: 6px 16px; border-radius: 4px; font-size: 12px; cursor: pointer; transition: all 0.15s; }
.btn-cancel { background: transparent; border: 1px solid var(--border-default); color: var(--text-secondary); }
.btn-cancel:hover { background: var(--bg-hover); }
.btn-save { background: var(--accent-primary); border: none; color: white; }
.btn-save:hover { opacity: 0.9; }
.tools-list { display: flex; flex-direction: column; gap: 8px; }
.tool-item { padding: 10px 12px; background: var(--bg-elevated); border-radius: 6px; border: 1px solid var(--border-subtle); }
.tool-name { font-size: 13px; font-weight: 500; color: var(--accent-primary); font-family: monospace; }
.tool-desc { font-size: 11px; color: var(--text-secondary); margin-top: 4px; }

/* 模型管理样式 */
.models-section { gap: 0; }
.model-list { display: flex; flex-direction: column; gap: 6px; margin-bottom: 16px; }
.model-item { display: flex; justify-content: space-between; align-items: center; padding: 10px 12px; background: var(--bg-elevated); border-radius: 6px; border: 1px solid var(--border-subtle); }
.model-item.builtin { opacity: 0.8; }
.model-info { flex: 1; min-width: 0; }
.model-name { font-size: 13px; font-weight: 500; color: var(--text-primary); display: flex; align-items: center; gap: 6px; }
.model-badge { font-size: 12px; }
.model-feature { font-size: 11px; opacity: 0.8; }
.model-id { font-size: 11px; color: var(--text-muted); font-family: monospace; margin-top: 2px; }
.model-url { font-size: 10px; color: var(--text-muted); margin-top: 2px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.model-actions { display: flex; align-items: center; gap: 8px; }
.builtin-header { margin-top: 16px; padding-top: 16px; border-top: 1px solid var(--border-subtle); }
.builtin-list { opacity: 0.7; }
.model-dialog { width: 420px; }
.required { color: var(--red); }
.form-group input[type="password"] { padding: 8px 10px; background: var(--bg-elevated); border: 1px solid var(--border-default); border-radius: 4px; color: var(--text-primary); font-size: 13px; outline: none; }

/* Kiro 账号管理样式 */
.kiro-section {
  padding: 0 !important;
  margin: 0 !important;
  height: 100%;
  width: 100%;
  max-width: none !important;
  overflow-y: auto;
  overflow-x: hidden;
  display: flex;
  flex-direction: column;
  position: relative;
}

.kiro-section :deep(.kiro-account-manager) {
  width: 100%;
  min-width: 0;
  flex: 1;
}

/* 技能管理样式 */
.skills-section {
  padding: 0 !important;
  margin: 0 !important;
  height: 100%;
  width: 100%;
  max-width: none !important;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.skills-section :deep(.skills-manager) {
  width: 100%;
  min-width: 0;
  flex: 1;
}

/* 插件管理样式 */
.plugins-section { gap: 16px; }
.plugin-card { background: var(--bg-elevated); border-radius: 8px; border: 1px solid var(--border-subtle); overflow: hidden; }
.plugin-header { display: flex; gap: 12px; padding: 16px; border-bottom: 1px solid var(--border-subtle); }
.plugin-icon { font-size: 32px; }
.plugin-info { flex: 1; }
.plugin-name { font-size: 16px; font-weight: 600; color: var(--text-primary); }
.plugin-desc { font-size: 12px; color: var(--text-secondary); margin-top: 4px; }
.plugin-features { padding: 12px 16px; display: flex; flex-direction: column; gap: 6px; }
.feature-item { font-size: 12px; color: var(--text-secondary); }
.plugin-footer { display: flex; flex-direction: column; gap: 10px; padding: 12px 16px; background: var(--bg-surface); border-top: 1px solid var(--border-subtle); }
.plugin-status { display: flex; align-items: center; gap: 8px; }
.status-badge { font-size: 11px; padding: 2px 8px; border-radius: 10px; }
.status-badge.not-installed { background: var(--bg-hover); color: var(--text-muted); }
.plugin-status.installed .status-badge { background: rgba(128, 255, 181, 0.15); color: var(--green); }
.version { font-size: 11px; color: var(--text-muted); }
.plugin-actions { display: flex; gap: 8px; }
.plugin-actions button, .plugin-actions a { flex: 1; text-align: center; }
.btn-uninstall { padding: 6px 12px; background: transparent; border: 1px solid var(--border-default); border-radius: 4px; color: var(--text-secondary); font-size: 12px; cursor: pointer; }
.btn-uninstall:hover { background: var(--red); color: white; border-color: var(--red); }
.btn-fix { padding: 6px 12px; background: var(--yellow); border: none; border-radius: 4px; color: #000; font-size: 12px; cursor: pointer; }
.btn-fix:hover { opacity: 0.9; }
.btn-restart { display: flex; align-items: center; gap: 8px; padding: 10px 20px; background: var(--accent-primary); border: none; border-radius: 6px; color: white; font-size: 13px; cursor: pointer; font-weight: 500; }
.btn-restart:hover { opacity: 0.9; }
.btn-restart:disabled { opacity: 0.5; cursor: not-allowed; }
.restart-section { margin-top: 16px; padding: 16px; background: var(--bg-elevated); border-radius: 8px; border: 1px solid var(--border-subtle); display: flex; flex-direction: column; align-items: center; gap: 8px; }
.restart-hint { font-size: 11px; color: var(--text-muted); }
.btn-auth { padding: 6px 12px; background: var(--accent-primary); border: none; border-radius: 4px; color: white; font-size: 12px; cursor: pointer; }
.btn-auth:hover { opacity: 0.9; }
.btn-update { padding: 6px 12px; background: var(--green); border: none; border-radius: 4px; color: white; font-size: 12px; cursor: pointer; }
.btn-update:hover { opacity: 0.9; }
.btn-auth-manual { padding: 6px 12px; background: var(--yellow); border: none; border-radius: 4px; color: #000; font-size: 12px; cursor: pointer; }
.btn-auth-manual:hover { opacity: 0.9; }
.btn-docs { padding: 6px 12px; background: transparent; border: 1px solid var(--border-default); border-radius: 4px; color: var(--text-secondary); font-size: 12px; cursor: pointer; text-decoration: none; display: inline-block; }
.btn-docs:hover { background: var(--bg-hover); color: var(--text-primary); }
.plugin-tip { display: flex; gap: 12px; padding: 12px 16px; background: var(--bg-elevated); border-radius: 8px; border: 1px solid var(--border-subtle); }
.plugin-tip-inline { display: flex; gap: 10px; padding: 10px 16px; background: var(--bg-surface); border-top: 1px solid var(--border-subtle); }
.plugin-tip-inline .tip-icon { font-size: 16px; }
.plugin-tip-inline .tip-title { font-size: 11px; font-weight: 600; color: var(--text-secondary); }
.plugin-tip-inline .tip-text { font-size: 11px; color: var(--text-muted); margin-top: 2px; }
.tip-icon { font-size: 20px; }
.tip-content { flex: 1; }
.tip-title { font-size: 12px; font-weight: 600; color: var(--text-primary); }
.tip-text { font-size: 11px; color: var(--text-secondary); margin-top: 4px; }

/* 远程控制样式 */
.remote-section { gap: 16px; }
.remote-card { background: var(--bg-elevated); border-radius: 8px; border: 1px solid var(--border-subtle); overflow: hidden; }
.remote-header { display: flex; gap: 12px; padding: 16px; border-bottom: 1px solid var(--border-subtle); }
.remote-icon { font-size: 32px; }
.remote-info { flex: 1; }
.remote-title { font-size: 16px; font-weight: 600; color: var(--text-primary); }
.remote-desc { font-size: 12px; color: var(--text-secondary); margin-top: 4px; }
.remote-body { padding: 16px; }
.remote-body.active { background: rgba(128, 255, 181, 0.05); }
.remote-body.inactive { background: var(--bg-surface); }
.connection-info { display: flex; flex-direction: column; gap: 12px; margin-bottom: 16px; }
.info-row { display: flex; justify-content: space-between; align-items: center; padding: 10px 12px; background: var(--bg-surface); border-radius: 6px; }
.info-label { font-size: 12px; color: var(--text-secondary); font-weight: 500; }
.info-value { font-size: 13px; color: var(--text-primary); font-family: monospace; }
.status-badge.active { background: rgba(128, 255, 181, 0.15); color: var(--green); padding: 4px 10px; border-radius: 12px; font-size: 12px; font-weight: 500; }
.connection-code { display: flex; align-items: center; gap: 8px; }
.code-display { font-size: 24px; font-weight: 700; color: var(--accent-primary); font-family: monospace; letter-spacing: 4px; }
.btn-copy { display: flex; align-items: center; justify-content: center; width: 32px; height: 32px; background: var(--bg-hover); border: 1px solid var(--border-subtle); border-radius: 6px; color: var(--text-secondary); cursor: pointer; transition: all 0.15s; }
.btn-copy:hover { background: var(--accent-primary); color: white; border-color: var(--accent-primary); }
.usage-steps { padding: 12px; background: var(--bg-surface); border-radius: 6px; border: 1px solid var(--border-subtle); }
.steps-title { font-size: 13px; font-weight: 600; color: var(--text-primary); margin-bottom: 8px; }
.steps-list { margin: 0; padding-left: 20px; }
.steps-list li { font-size: 12px; color: var(--text-secondary); margin-bottom: 6px; line-height: 1.5; }
.steps-list li:last-child { margin-bottom: 0; }
.inactive-message { text-align: center; padding: 20px; }
.message-icon { font-size: 48px; margin-bottom: 12px; }
.message-text { font-size: 14px; font-weight: 600; color: var(--text-primary); margin-bottom: 6px; }
.message-hint { font-size: 12px; color: var(--text-muted); }
.remote-features { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 12px; }
.feature-card { padding: 16px; background: var(--bg-elevated); border-radius: 8px; border: 1px solid var(--border-subtle); text-align: center; }
.feature-icon { font-size: 32px; margin-bottom: 8px; }
.feature-name { font-size: 13px; font-weight: 600; color: var(--text-primary); margin-bottom: 4px; }
.feature-desc { font-size: 11px; color: var(--text-secondary); }
</style>
