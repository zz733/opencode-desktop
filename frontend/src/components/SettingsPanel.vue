<script setup>
import { ref, onMounted, computed, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { languages, setLocale } from '../i18n'
import { useTheme } from '../composables/useTheme'
import { 
  GetMCPConfig, SaveMCPConfig, GetMCPMarket, AddMCPServer, RemoveMCPServer, 
  ToggleMCPServer, OpenMCPConfigFile, GetMCPStatus, ConnectMCPServer, 
  DisconnectMCPServer, GetMCPTools 
} from '../../wailsjs/go/main/App'
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime'

const { t, locale } = useI18n()
const { currentTheme, themes, setTheme } = useTheme()
const emit = defineEmits(['close', 'open-file'])

const activeCategory = ref('general')
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
  statusInterval = setInterval(refreshStatus, 5000)
})
onUnmounted(() => { if (statusInterval) clearInterval(statusInterval) })
</script>

<template>
  <aside class="settings-panel">
    <div class="settings-header"><span>{{ t('settings.title') }}</span></div>
    
    <div class="settings-nav">
      <div :class="['nav-item', { active: activeCategory === 'general' }]" @click="activeCategory = 'general'">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="3"/><path d="M12 1v4M12 19v4M4.22 4.22l2.83 2.83M16.95 16.95l2.83 2.83M1 12h4M19 12h4M4.22 19.78l2.83-2.83M16.95 7.05l2.83-2.83"/>
        </svg>
        <span>{{ t('settings.general') }}</span>
      </div>
      <div :class="['nav-item', { active: activeCategory === 'mcp' }]" @click="activeCategory = 'mcp'">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="2" y="3" width="20" height="14" rx="2"/><path d="M8 21h8M12 17v4"/>
        </svg>
        <span>MCP</span>
      </div>
    </div>
    
    <div class="settings-content">
      <div v-if="activeCategory === 'general'" class="settings-section">
        <div class="setting-item">
          <div class="setting-label">{{ t('settings.theme') }}</div>
          <div class="setting-control">
            <select :value="currentTheme" @change="changeTheme($event.target.value)">
              <option v-for="theme in themes" :key="theme.id" :value="theme.id">{{ theme.name }}</option>
            </select>
          </div>
        </div>
        <div class="setting-item">
          <div class="setting-label">{{ t('settings.language') }}</div>
          <div class="setting-control">
            <select :value="locale" @change="changeLanguage($event.target.value)">
              <option v-for="lang in languages" :key="lang.code" :value="lang.code">{{ lang.name }}</option>
            </select>
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
  </aside>
</template>

<style scoped>
.settings-panel { flex: 1; background: var(--bg-surface); display: flex; flex-direction: column; overflow: hidden; }
.settings-header { padding: 12px 16px; font-size: 11px; font-weight: 500; letter-spacing: 0.5px; color: var(--text-secondary); text-transform: uppercase; border-bottom: 1px solid var(--border-subtle); }
.settings-nav { display: flex; padding: 8px 12px; gap: 4px; border-bottom: 1px solid var(--border-subtle); }
.nav-item { display: flex; align-items: center; gap: 6px; padding: 6px 12px; border-radius: 4px; cursor: pointer; font-size: 12px; color: var(--text-secondary); transition: all 0.15s; }
.nav-item:hover { background: var(--bg-hover); color: var(--text-primary); }
.nav-item.active { background: var(--accent-primary); color: white; }
.nav-item svg { opacity: 0.7; }
.nav-item.active svg { opacity: 1; }
.settings-content { flex: 1; overflow-y: auto; padding: 12px; }
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
</style>
