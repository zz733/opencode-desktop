import { ref } from 'vue'
import { 
  GetServerURL, SetServerURL, CheckConnection,
  GetSessions, CreateSession, SendMessage, SendMessageWithModel, 
  SubscribeEvents, GetOpenCodeStatus, AutoStartOpenCode, InstallOpenCode,
  CancelSession
} from '../../wailsjs/go/main/App'
import { EventsOn, EventsEmit } from '../../wailsjs/runtime/runtime'

// 输出日志到 OUTPUT 面板
function log(message) {
  console.log(message)
  EventsEmit('output-log', message)
}

const connected = ref(false)
const connecting = ref(false)
const sessions = ref([])
const currentSession = ref(null)
const messages = ref([])
const sending = ref(false)
// 从 localStorage 读取上次选择的模型
const currentModel = ref(localStorage.getItem('selectedModel') || 'opencode/claude-opus-4-5')
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
  log('开始检查 OpenCode 状态...')
  
  try {
    // 先检查 OpenCode 状态
    const status = await GetOpenCodeStatus()
    log(`OpenCode 状态: 已安装=${status.installed}, 运行中=${status.running}, 已连接=${status.connected}`)
    if (status.path) log(`OpenCode 路径: ${status.path}`)
    if (status.version) log(`OpenCode 版本: ${status.version}`)
    
    if (!status.installed) {
      // 未安装，自动安装
      log('OpenCode 未安装，开始自动安装...')
      openCodeStatus.value = 'installing'
      try {
        await InstallOpenCode()
        log('安装完成，继续启动...')
      } catch (e) {
        log(`安装失败: ${e}`)
        openCodeStatus.value = 'install-failed'
        connecting.value = false
        return
      }
    }
    
    // 重新检查状态
    const newStatus = await GetOpenCodeStatus()
    
    if (newStatus.connected) {
      // 已连接，直接使用
      log('OpenCode 服务已在运行，直接连接')
      await onConnected()
      return
    }
    
    // 已安装但未运行，自动启动
    log('正在启动 OpenCode 服务...')
    openCodeStatus.value = 'starting'
    try {
      await AutoStartOpenCode()
    } catch (e) {
      log(`启动出错 (可能已在启动中): ${e}`)
    }
    
    // 轮询等待连接
    waitForConnection()
  } catch (e) {
    log(`连接失败: ${e}`)
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
  if (connected.value) return // 防止重复调用
  connected.value = true
  connecting.value = false
  openCodeStatus.value = 'connected'
  log('正在加载会话列表...')
  await loadSessions()
  setupEventListeners()
  await SubscribeEvents()
  log('OpenCode 连接成功，已订阅事件')
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
    log(`已加载 ${sessions.value.length} 个会话`)
    // 自动选择最近的会话或创建新会话
    if (sessions.value.length > 0) {
      selectSession(sessions.value[0])
    }
  } catch (e) {
    log(`加载会话失败: ${e}`)
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
  console.log('sendMessage called:', text, 'sending:', sending.value, 'connected:', connected.value)
  if (!text.trim() || sending.value) return
  
  // 如果没有会话，自动创建
  let session = currentSession.value
  console.log('current session:', session)
  if (!session) {
    session = await createSession()
    if (!session) {
      console.error('创建会话失败')
      return
    }
  }
  
  sending.value = true
  messages.value.push({ role: 'user', content: text })
  messages.value.push({ role: 'assistant', content: '', reasoning: '', tools: {} })
  
  try {
    console.log('calling SendMessageWithModel:', session.id, text, currentModel.value)
    await SendMessageWithModel(session.id, text, currentModel.value)
    setTimeout(() => { if (sending.value) sending.value = false }, 60000)
  } catch (e) {
    console.error('SendMessageWithModel error:', e)
    messages.value[messages.value.length - 1].content = '错误: ' + e
    sending.value = false
  }
}

function setupEventListeners() {
  EventsOn('server-event', (data) => {
    console.log('收到 server-event:', data)
    try {
      const event = JSON.parse(data)
      handleEvent(event)
    } catch (e) {
      console.error('解析事件失败:', e, data)
    }
  })
}

function handleEvent(event) {
  console.log('处理事件:', event.type, event)
  
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
      // 提取工具信息，包括输入参数
      const toolInfo = {
        id: part.id,
        name: part.tool,
        status: part.state?.status || 'pending',
        input: part.state?.input || part.input || null,
        output: part.state?.output || null
      }
      last.tools[part.id] = toolInfo
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

// 设置模型并保存到 localStorage
function setModel(modelId) {
  currentModel.value = modelId
  localStorage.setItem('selectedModel', modelId)
}

// 取消当前请求
async function cancelMessage() {
  if (!sending.value || !currentSession.value) return
  
  try {
    await CancelSession(currentSession.value.id)
    sending.value = false
  } catch (e) {
    console.error('取消失败:', e)
    // 即使取消失败也停止 sending 状态
    sending.value = false
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
    sendMessage,
    setModel,
    cancelMessage
  }
}
