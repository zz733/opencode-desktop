import { ref } from 'vue'
import { 
  GetServerURL, SetServerURL, CheckConnection,
  GetSessions, CreateSession, SendMessage, SendMessageWithModel, 
  SubscribeEvents
} from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'

const connected = ref(false)
const connecting = ref(false)
const sessions = ref([])
const currentSession = ref(null)
const messages = ref([])
const sending = ref(false)
const currentModel = ref('opencode/claude-opus-4-5')

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

// 自动连接
async function autoConnect() {
  if (connected.value || connecting.value) return
  connecting.value = true
  
  try {
    const url = await GetServerURL()
    await SetServerURL(url)
    const ok = await CheckConnection()
    if (ok) {
      connected.value = true
      await loadSessions()
      setupEventListeners()
      await SubscribeEvents()
    }
  } catch (e) {
    console.error('连接失败:', e)
    // 3秒后重试
    setTimeout(autoConnect, 3000)
  }
  connecting.value = false
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
  return {
    connected,
    connecting,
    sessions,
    currentSession,
    messages,
    sending,
    currentModel,
    models,
    autoConnect,
    selectSession,
    createSession,
    sendMessage
  }
}
