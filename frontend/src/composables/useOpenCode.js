import { ref } from 'vue'
import { 
  GetServerURL, SetServerURL, CheckConnection,
  GetSessions, CreateSession, SendMessage, SendMessageWithModel, 
  SubscribeEvents, GetOpenCodeStatus, AutoStartOpenCode, InstallOpenCode
} from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'

const connected = ref(false)
const connecting = ref(false)
const sessions = ref([])
const currentSession = ref(null)
const messages = ref([])
const sending = ref(false)
const currentModel = ref('opencode/claude-opus-4-5')
const openCodeStatus = ref(null) // 'not-installed', 'installing', 'starting', 'connected'

const models = [
  { id: 'opencode/big-pickle', name: 'Big Pickle', free: true },
  { id: 'opencode/grok-code', name: 'Grok Code Fast', free: true },
  { id: 'opencode/minimax-m2.1-free', name: 'MiniMax M2.1', free: true },
  { id: 'opencode/glm-4.7-free', name: 'GLM 4.7', free: true },
  { id: 'opencode/gpt-5-nano', name: 'GPT 5 Nano', free: true },
  { id: 'opencode/kimi-k2', name: 'Kimi K2', free: false },
  { id: 'opencode/claude-opus-4-5', name: 'Claude Opus 4.5', free: false },
  { id: 'opencode/claude-sonnet-4-5', name: 'Claude Sonnet 4.5', free: false },
  { id: 'opencode/gpt-5.1-codex', name: 'GPT 5.1 Codex', free: false },
]

// 自动连接（包含检测、安装、启动）
async function autoConnect() {
  if (connected.value || connecting.value) return
  connecting.value = true
  
  try {
    // 先检查 OpenCode 状态
    const status = await GetOpenCodeStatus()
    console.log('OpenCode status:', status)
    
    if (!status.installed) {
      // 未安装，提示用户
      openCodeStatus.value = 'not-installed'
      connecting.value = false
      return
    }
    
    if (status.connected) {
      // 已连接，直接使用
      await onConnected()
      return
    }
    
    // 已安装但未运行，自动启动
    openCodeStatus.value = 'starting'
    try {
      await AutoStartOpenCode()
    } catch (e) {
      console.log('AutoStart error (may already be starting):', e)
    }
    
    // 轮询等待连接
    waitForConnection()
  } catch (e) {
    console.error('连接失败:', e)
    openCodeStatus.value = 'error'
    connecting.value = false
    setTimeout(autoConnect, 3000)
  }
}

// 等待连接成功
async function waitForConnection() {
  let retries = 0
  const maxRetries = 30
  
  const check = async () => {
    try {
      const status = await GetOpenCodeStatus()
      if (status.connected) {
        await onConnected()
        return
      }
    } catch (e) {}
    
    retries++
    if (retries < maxRetries) {
      setTimeout(check, 1000)
    } else {
      openCodeStatus.value = 'timeout'
      connecting.value = false
    }
  }
  
  setTimeout(check, 1000)
}

// 安装 OpenCode
async function installOpenCode() {
  openCodeStatus.value = 'installing'
  try {
    await InstallOpenCode()
    // 安装完成后自动启动
    await autoConnect()
  } catch (e) {
    console.error('安装失败:', e)
    openCodeStatus.value = 'install-failed'
  }
}

// 连接成功后的处理
async function onConnected() {
  connected.value = true
  connecting.value = false
  openCodeStatus.value = 'connected'
  await loadSessions()
  setupEventListeners()
  await SubscribeEvents()
}

// 监听 OpenCode 状态事件
function setupOpenCodeEvents() {
  EventsOn('opencode-status', (status) => {
    openCodeStatus.value = status
    if (status === 'connected') {
      onConnected()
    }
  })
  
  EventsOn('opencode-installed', () => {
    autoConnect()
  })
}

async function loadSessions() {
  try {
    sessions.value = await GetSessions()
    // 自动选择最近的会话或创建新会话
    if (sessions.value.length > 0) {
      selectSession(sessions.value[0])
    }
  } catch (e) {
    console.error('加载会话失败:', e)
  }
}

function selectSession(session) {
  currentSession.value = session
  messages.value = []
}

async function createSession() {
  try {
    const session = await CreateSession()
    sessions.value.unshift(session)
    selectSession(session)
    return session
  } catch (e) {
    console.error('创建会话失败:', e)
  }
}

async function sendMessage(text) {
  if (!text.trim() || sending.value) return
  
  // 如果没有会话，自动创建
  if (!currentSession.value) {
    await createSession()
  }
  
  sending.value = true
  messages.value.push({ role: 'user', content: text })
  messages.value.push({ role: 'assistant', content: '', reasoning: '', tools: {} })
  
  try {
    await SendMessageWithModel(currentSession.value.id, text, currentModel.value)
    setTimeout(() => { if (sending.value) sending.value = false }, 60000)
  } catch (e) {
    messages.value[messages.value.length - 1].content = '错误: ' + e
    sending.value = false
  }
}

function setupEventListeners() {
  EventsOn('server-event', (data) => {
    try {
      const event = JSON.parse(data)
      handleEvent(event)
    } catch (e) {}
  })
}

function handleEvent(event) {
  if (event.type === 'message.part.updated') {
    const part = event.properties?.part
    if (!part) return
    const last = messages.value[messages.value.length - 1]
    if (!last || last.role !== 'assistant') return
    
    if (part.type === 'text' && part.text) {
      last.content = part.text.replace(/^\n+/, '')
    } else if (part.type === 'reasoning' && part.text) {
      last.reasoning = part.text
    } else if (part.type === 'tool') {
      if (!last.tools) last.tools = {}
      last.tools[part.id] = {
        id: part.id,
        name: part.tool,
        status: part.state?.status || 'pending'
      }
    }
  }
  
  if (event.type === 'message.updated' || event.type === 'session.status') {
    const info = event.properties?.info
    const status = event.properties?.status
    if (info?.time?.completed || info?.finish || status === 'idle') {
      sending.value = false
    }
  }
}

export function useOpenCode() {
  // 初始化时设置事件监听
  setupOpenCodeEvents()
  
  return {
    connected,
    connecting,
    sessions,
    currentSession,
    messages,
    sending,
    currentModel,
    models,
    openCodeStatus,
    autoConnect,
    installOpenCode,
    selectSession,
    createSession,
    sendMessage
  }
}
