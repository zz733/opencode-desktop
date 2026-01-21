# 手机端远程控制 MVP 计划

## MVP 范围（不包括屏幕共享）

**核心功能**：
- ✅ 手机端远程控制
- ✅ 实时聊天（发送消息、查看回复）
- ✅ 文件浏览（查看项目文件）
- ✅ 终端输出查看
- ❌ 桌面屏幕查看（后期实现）

## 简化的实现方案

### 阶段 1：快速原型（使用 Ngrok）- 1 周

#### TASK-MVP-1.1：Ngrok 方案验证
**状态**：pending  
**优先级**：高  
**预计时间**：1 天

**目标**：使用 Ngrok 快速实现远程访问

**实现步骤**：
1. 在桌面端添加一个简单的 HTTP API
2. 使用 Ngrok 暴露本地服务
3. 手机浏览器直接访问

**优点**：
- 无需自建服务器
- 快速验证可行性
- 开发简单

**API 设计**：
```
GET  /api/sessions          - 获取会话列表
GET  /api/messages/:id      - 获取消息
POST /api/messages          - 发送消息
GET  /api/files             - 获取文件列表
GET  /api/files/content     - 获取文件内容
GET  /api/terminal/output   - 获取终端输出
```

---

#### TASK-MVP-1.2：桌面端 HTTP API
**状态**：pending  
**优先级**：高  
**预计时间**：2 天

**目标**：在桌面端添加 HTTP API 服务器

**实现**：
- 创建 `myapp/http_server.go`
- 实现 RESTful API
- 添加 CORS 支持
- 添加简单的 Token 认证

**代码结构**：
```go
type HTTPServer struct {
    app    *App
    server *http.Server
    token  string
}

func (s *HTTPServer) Start(port int) error
func (s *HTTPServer) Stop() error
func (s *HTTPServer) handleSessions(w, r)
func (s *HTTPServer) handleMessages(w, r)
func (s *HTTPServer) handleFiles(w, r)
func (s *HTTPServer) handleTerminal(w, r)
```

---

#### TASK-MVP-1.3：手机端 PWA（基础版）
**状态**：pending  
**优先级**：高  
**预计时间**：3 天

**目标**：创建简单的手机端界面

**功能**：
- 输入桌面端 URL 和 Token
- 聊天界面
- 文件浏览
- 终端查看

**技术栈**：
- Vue 3 + Vite
- Tailwind CSS（快速样式）
- Axios（HTTP 请求）

---

### 阶段 2：完善功能 - 1 周

#### TASK-MVP-2.1：实时更新（SSE）
**状态**：pending  
**优先级**：中  
**预计时间**：2 天

**目标**：使用 Server-Sent Events 实现实时更新

**实现**：
- 桌面端添加 SSE 端点
- 手机端监听 SSE 事件
- 自动更新消息和终端输出

---

#### TASK-MVP-2.2：UI 优化
**状态**：pending  
**优先级**：中  
**预计时间**：2 天

**目标**：优化手机端界面

**改进**：
- 响应式设计
- 加载状态
- 错误提示
- 离线提示

---

#### TASK-MVP-2.3：安全增强
**状态**：pending  
**优先级**：中  
**预计时间**：1 天

**目标**：增强安全性

**实现**：
- Token 过期机制
- HTTPS（Ngrok 自带）
- 请求频率限制

---

## 实现细节

### 1. 桌面端 HTTP Server

```go
// myapp/http_server.go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type HTTPServer struct {
    app    *App
    server *http.Server
    token  string
    port   int
}

func NewHTTPServer(app *App) *HTTPServer {
    return &HTTPServer{
        app:   app,
        token: generateToken(),
    }
}

func (s *HTTPServer) Start(port int) error {
    s.port = port
    
    mux := http.NewServeMux()
    
    // API 路由
    mux.HandleFunc("/api/sessions", s.authMiddleware(s.handleSessions))
    mux.HandleFunc("/api/messages", s.authMiddleware(s.handleMessages))
    mux.HandleFunc("/api/files", s.authMiddleware(s.handleFiles))
    mux.HandleFunc("/api/terminal", s.authMiddleware(s.handleTerminal))
    mux.HandleFunc("/api/status", s.authMiddleware(s.handleStatus))
    
    // SSE 端点
    mux.HandleFunc("/api/events", s.authMiddleware(s.handleEvents))
    
    s.server = &http.Server{
        Addr:    fmt.Sprintf(":%d", port),
        Handler: s.corsMiddleware(mux),
    }
    
    go s.server.ListenAndServe()
    return nil
}

func (s *HTTPServer) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        if token != "Bearer "+s.token {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        next(w, r)
    }
}

func (s *HTTPServer) corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
        
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}

func generateToken() string {
    // 生成随机 Token
    return fmt.Sprintf("%d", time.Now().Unix())
}
```

### 2. App API 集成

```go
// myapp/app.go

type App struct {
    // ... 现有字段
    httpServer *HTTPServer
}

// StartRemoteControl 启动远程控制
func (a *App) StartRemoteControl(port int) (string, error) {
    if a.httpServer == nil {
        a.httpServer = NewHTTPServer(a)
    }
    
    err := a.httpServer.Start(port)
    if err != nil {
        return "", err
    }
    
    // 返回访问信息
    info := map[string]string{
        "port":  fmt.Sprintf("%d", port),
        "token": a.httpServer.token,
        "url":   fmt.Sprintf("http://localhost:%d", port),
    }
    
    return json.Marshal(info)
}

// StopRemoteControl 停止远程控制
func (a *App) StopRemoteControl() error {
    if a.httpServer != nil {
        return a.httpServer.Stop()
    }
    return nil
}

// GetRemoteControlInfo 获取远程控制信息
func (a *App) GetRemoteControlInfo() (string, error) {
    if a.httpServer == nil {
        return "", fmt.Errorf("remote control not started")
    }
    
    info := map[string]interface{}{
        "active": true,
        "port":   a.httpServer.port,
        "token":  a.httpServer.token,
    }
    
    return json.Marshal(info)
}
```

### 3. 手机端 PWA

```vue
<!-- kiro-mobile/src/App.vue -->
<template>
  <div class="app">
    <!-- 连接页面 -->
    <div v-if="!connected" class="connect-page">
      <h1>Kiro Remote</h1>
      <input v-model="serverUrl" placeholder="服务器地址" />
      <input v-model="token" placeholder="访问令牌" type="password" />
      <button @click="connect">连接</button>
    </div>
    
    <!-- 主界面 -->
    <div v-else class="main-page">
      <div class="tabs">
        <button @click="activeTab = 'chat'">💬 聊天</button>
        <button @click="activeTab = 'files'">📁 文件</button>
        <button @click="activeTab = 'terminal'">💻 终端</button>
      </div>
      
      <div class="content">
        <ChatPanel v-if="activeTab === 'chat'" />
        <FileExplorer v-if="activeTab === 'files'" />
        <TerminalViewer v-if="activeTab === 'terminal'" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import axios from 'axios'

const connected = ref(false)
const serverUrl = ref('http://localhost:8080')
const token = ref('')
const activeTab = ref('chat')

// 创建 axios 实例
let api = null

const connect = async () => {
  try {
    api = axios.create({
      baseURL: serverUrl.value,
      headers: {
        'Authorization': `Bearer ${token.value}`
      }
    })
    
    // 测试连接
    await api.get('/api/status')
    connected.value = true
  } catch (e) {
    alert('连接失败: ' + e.message)
  }
}
</script>
```

## 使用 Ngrok

### 安装 Ngrok
```bash
# macOS
brew install ngrok

# 或下载
# https://ngrok.com/download
```

### 启动 Ngrok
```bash
# 暴露本地 8080 端口
ngrok http 8080
```

### 获取公网地址
```
Forwarding  https://abc123.ngrok.io -> http://localhost:8080
```

手机端使用 `https://abc123.ngrok.io` 访问。

## 优势

1. **快速实现**：1-2 周完成 MVP
2. **无需服务器**：使用 Ngrok 免费版
3. **简单架构**：HTTP + SSE，无需 WebSocket
4. **易于调试**：标准的 REST API

## 后续扩展

完成 MVP 后，可以：
1. 添加屏幕共享功能
2. 自建中继服务器（替代 Ngrok）
3. 实现 WebSocket（更低延迟）
4. 添加端到端加密
5. 开发原生移动应用

## 下一步

开始实现 TASK-MVP-1.2：桌面端 HTTP API
