import { ref } from 'vue'
import { 
  GetServerURL, SetServerURL, CheckConnection,
  GetSessions, CreateSession, SendMessage, SendMessageWithModel, 
  SubscribeEvents, GetOpenCodeStatus, AutoStartOpenCode, InstallOpenCode,
  CancelSession, GetSessionMessages
} from '../../wailsjs/go/main/App'
import { EventsOn, EventsEmit } from '../../wailsjs/runtime/runtime'
import { i18n } from '../i18n'

// 语言名称映射
const languageNames = {
  'zh-CN': '中文',
  'en': 'English',
  'ja': '日本語'
}

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
const currentWorkDir = ref('') // 当前工作目录

// 目录 -> 会话ID 的映射
const dirSessionMap = ref(JSON.parse(localStorage.getItem('dirSessionMap') || '{}'))

// 保存映射到 localStorage
function saveDirSessionMap() {
  localStorage.setItem('dirSessionMap', JSON.stringify(dirSessionMap.value))
}

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
  
  // 只在首次连接时订阅事件
  log('订阅服务器事件...')
  await SubscribeEvents()
  log('OpenCode 连接成功')
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

async function selectSession(session) {
  currentSession.value = session
  messages.value = []
  
  // 加载历史消息
  if (session?.id) {
    try {
      log(`加载会话 ${session.id} 的历史消息...`)
      const history = await GetSessionMessages(session.id)
      if (history && history.length > 0) {
        // 过滤并转换消息格式
        let processedMessages = history
          .filter(msg => msg.role === 'user' || msg.role === 'assistant')
          .map(msg => {
            let content = msg.content || ''
            // 过滤掉语言提示和文件上下文前缀
            content = content.replace(/^\[Please respond in [^\]]+\]\n*/g, '')
            content = content.replace(/^\[Current active file: [^\]]+\]\n*/g, '')
            content = content.trim()
            return {
              role: msg.role,
              content: content,
              reasoning: '',
              tools: {}
            }
          })
          .filter(msg => msg.content) // 过滤空消息
        
        // 如果消息是倒序的（最新在前），需要反转
        // 检查：如果第一条是 assistant 而最后一条是 user，可能是倒序
        if (processedMessages.length >= 2) {
          const first = processedMessages[0]
          const last = processedMessages[processedMessages.length - 1]
          // 正常对话应该是 user 开始，如果第一条是 assistant，说明是倒序
          if (first.role === 'assistant' && last.role === 'user') {
            processedMessages = processedMessages.reverse()
          }
        }
        
        messages.value = processedMessages
        log(`已加载 ${messages.value.length} 条历史消息`)
      }
    } catch (e) {
      log(`加载历史消息失败: ${e}`)
    }
  }
}

async function createSession() {
  try {
    const session = await CreateSession()
    sessions.value.unshift(session)
    selectSession(session)
    
    // 绑定当前目录到新会话
    if (currentWorkDir.value) {
      dirSessionMap.value[currentWorkDir.value] = session.id
      saveDirSessionMap()
    }
    
    return session
  } catch (e) {
    console.error('创建会话失败:', e)
  }
}

// 当前活动文件路径
const activeFilePath = ref('')

// 设置活动文件
function setActiveFile(path) {
  activeFilePath.value = path
  log(`当前活动文件: ${path}`)
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
  // 界面显示原始消息
  messages.value.push({ role: 'user', content: text })
  messages.value.push({ role: 'assistant', content: '', reasoning: '', tools: {} })
  
  // 根据当前语言添加提示（只发送给 AI，不显示在界面）
  const currentLocale = i18n.global.locale.value
  const langName = languageNames[currentLocale] || 'English'
  
  // 构建消息，包含当前文件上下文
  let messageToSend = `[Please respond in ${langName}]`
  if (activeFilePath.value) {
    messageToSend += `\n[Current active file: ${activeFilePath.value}]`
  }
  messageToSend += `\n\n${text}`
  
  try {
    console.log('calling SendMessageWithModel:', session.id, currentModel.value)
    await SendMessageWithModel(session.id, messageToSend, currentModel.value)
    setTimeout(() => { if (sending.value) sending.value = false }, 60000)
  } catch (e) {
    console.error('SendMessageWithModel error:', e)
    messages.value[messages.value.length - 1].content = '错误: ' + e
    sending.value = false
  }
}

let eventListenersSetup = false

function setupEventListeners() {
  if (eventListenersSetup) return
  eventListenersSetup = true
  
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

// 记录当前正在处理的消息 ID，避免重复处理
let currentAssistantMessageId = null

function handleEvent(event) {
  console.log('处理事件:', event.type, JSON.stringify(event.properties || {}).substring(0, 200))
  
  if (event.type === 'message.part.updated') {
    const part = event.properties?.part
    const messageInfo = event.properties?.message
    
    if (!part) return
    
    // 检查消息角色 - 只处理 assistant 消息
    if (messageInfo?.role && messageInfo.role !== 'assistant') {
      console.log('跳过非 assistant 消息:', messageInfo.role)
      return
    }
    
    // 如果有消息 ID，检查是否是新的 assistant 消息
    if (messageInfo?.id) {
      if (currentAssistantMessageId && messageInfo.id !== currentAssistantMessageId) {
        // 不同的消息 ID，可能是用户消息的回显
        if (messageInfo.role !== 'assistant') {
          console.log('跳过不同 ID 的非 assistant 消息')
          return
        }
      }
      currentAssistantMessageId = messageInfo.id
    }
    
    const last = messages.value[messages.value.length - 1]
    if (!last || last.role !== 'assistant') return
    
    if (part.type === 'text' && part.text !== undefined) {
      // 过滤掉可能是用户消息的内容（包含语言提示前缀）
      const text = part.text || ''
      if (text.includes('[Please respond in')) {
        console.log('跳过包含语言提示的文本（可能是用户消息回显）')
        return
      }
      last.content = text.replace(/^\n+/, '')
    } else if (part.type === 'reasoning' && part.text) {
      last.reasoning = part.text
    } else if (part.type === 'tool') {
      if (!last.tools) last.tools = {}
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
    
    // 只有当 session 状态变为 idle 时才停止 sending
    // message.updated 可能在工具执行过程中多次触发，不应该停止
    if (status === 'idle') {
      sending.value = false
      currentAssistantMessageId = null
    }
    
    // 或者消息明确标记为完成
    if (info?.time?.completed && info?.finish) {
      sending.value = false
      currentAssistantMessageId = null
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

// 切换工作目录时切换或创建会话
async function switchWorkDir(dir) {
  if (!dir) return
  
  // 即使是同一个目录，也需要确保连接正常
  const isSameDir = dir === currentWorkDir.value
  currentWorkDir.value = dir
  
  if (!isSameDir) {
    log(`工作目录已切换到: ${dir}`)
    
    // 重置连接状态，等待新目录的 OpenCode 实例就绪
    connecting.value = true
    connected.value = false
    messages.value = []
  }
  
  // 轮询等待连接
  let retries = 0
  const maxRetries = 30
  
  while (retries < maxRetries) {
    try {
      const status = await GetOpenCodeStatus()
      log(`检查连接状态: connected=${status.connected}, port=${status.port}, workDir=${status.workDir}`)
      
      if (status.connected && status.workDir === dir) {
        connected.value = true
        connecting.value = false
        log(`已连接到 ${dir} 的 OpenCode (端口 ${status.port})`)
        
        // 重新加载会话列表
        await loadSessions()
        
        // 重新订阅事件（会自动取消旧的订阅）
        log('重新订阅事件...')
        await SubscribeEvents()
        
        // 检查该目录是否有绑定的会话
        const sessionId = dirSessionMap.value[dir]
        if (sessionId) {
          const existingSession = sessions.value.find(s => s.id === sessionId)
          if (existingSession) {
            log(`切换到目录 ${dir} 的已有会话: ${existingSession.id}`)
            selectSession(existingSession)
            return
          }
        }
        
        // 没有绑定的会话，创建新会话
        log(`为目录 ${dir} 创建新会话`)
        await createSession()
        return
      }
    } catch (e) {
      log(`连接检查出错: ${e}`)
    }
    retries++
    await new Promise(r => setTimeout(r, 1000))
  }
  
  connecting.value = false
  log('连接超时')
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
    currentWorkDir,
    activeFilePath,
    autoConnect,
    installOpenCode,
    selectSession,
    createSession,
    sendMessage,
    setModel,
    cancelMessage,
    switchWorkDir,
    setActiveFile
  }
}
