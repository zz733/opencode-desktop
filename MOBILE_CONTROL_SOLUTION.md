# 手机端控制编程方案

## 项目背景

参考 Happy 项目 (https://github.com/slopus/happy)，实现从手机端控制桌面 AI 编程助手的功能。

## Happy 项目分析

### 核心特性
- **移动端和 Web 客户端**：支持从任何设备访问
- **实时语音**：支持语音输入和输出
- **端到端加密**：保证数据安全
- **完整功能**：与桌面端功能一致
- **多实例支持**：可以并行运行多个 AI 编程会话

### 工作原理
1. 在电脑上运行 `happy` 命令（替代 `claude` 或 `codex`）
2. Happy 作为包装器启动 AI 编程助手
3. 当需要从手机控制时，会话切换到远程模式
4. 手机端通过 Happy 应用连接到桌面端
5. 在键盘上按任意键可切换回桌面模式

## 我们的实现方案

### 方案概述

基于当前项目的 Wails + Go + Vue.js 架构，实现一个轻量级的远程控制系统。

### 架构设计

```
┌─────────────────┐         ┌─────────────────┐         ┌─────────────────┐
│   手机端 App    │◄───────►│   中继服务器    │◄───────►│   桌面端应用    │
│  (PWA/原生)     │  HTTPS  │  (WebSocket)    │  WS     │   (Wails)       │
└─────────────────┘         └─────────────────┘         └─────────────────┘
                                                                │
                                                                ▼
                                                         ┌─────────────────┐
                                                         │   OpenCode      │
                                                         │   引擎          │
                                                         └─────────────────┘
```

### 技术方案

#### 1. 桌面端改造（最小改动）

**新增 API 端点**：
- `StartRemoteSession()` - 启动远程会话
- `StopRemoteSession()` - 停止远程会话
- `GetRemoteSessionInfo()` - 获取会话信息（连接码、状态）
- `SendRemoteMessage(message)` - 接收来自手机的消息
- `GetRemoteMessages()` - 获取待发送到手机的消息

**新增 Go 模块** (`myapp/remote_control.go`):
```go
type RemoteControlManager struct {
    app           *App
    wsClient      *websocket.Conn
    sessionID     string
    connectionCode string
    active        bool
    messageQueue  chan RemoteMessage
}

type RemoteMessage struct {
    Type      string      // "chat", "file", "terminal", "status"
    Content   interface{}
    Timestamp time.Time
}
```

#### 2. 中继服务器（独立部署）

**技术栈**：
- Go + Gorilla WebSocket
- 轻量级，可部署在任何云服务器
- 支持端到端加密（消息在客户端加密）

**核心功能**：
- 会话配对（通过 6 位连接码）
- 消息转发（桌面端 ↔ 手机端）
- 会话管理（超时、断线重连）
- 不存储任何敏感数据

**部署方式**：
- 选项 1：自建服务器（推荐，完全控制）
- 选项 2：使用 Cloudflare Workers（免费额度）
- 选项 3：使用 Railway/Render 等 PaaS

#### 3. 手机端实现

**方案 A：PWA（推荐，快速实现）**
- 基于 Vue.js 开发
- 响应式设计，适配手机屏幕
- 支持离线缓存
- 无需应用商店审核
- 可添加到主屏幕

**方案 B：原生应用（长期方案）**
- React Native / Flutter
- 更好的性能和体验
- 支持推送通知
- 需要应用商店发布

**核心功能**：
- 连接码输入界面
- 聊天界面（发送消息、查看回复）
- 文件浏览器（查看项目文件）
- 终端输出查看
- 会话状态监控

### 数据流设计

#### 连接建立流程
```
1. 桌面端：点击"启动远程控制" → 生成 6 位连接码 → 连接中继服务器
2. 手机端：输入连接码 → 连接中继服务器 → 配对成功
3. 双方建立加密通道
```

#### 消息同步流程
```
手机端输入 → 加密 → 中继服务器 → 桌面端 → 解密 → OpenCode 处理
OpenCode 响应 → 桌面端 → 加密 → 中继服务器 → 手机端 → 解密 → 显示
```

### 安全设计

1. **端到端加密**：
   - 使用 AES-256-GCM 加密所有消息
   - 密钥通过 ECDH 密钥交换生成
   - 中继服务器无法解密消息内容

2. **会话认证**：
   - 6 位随机连接码（有效期 5 分钟）
   - 一次性使用，配对后失效
   - 支持会话 Token 续期

3. **访问控制**：
   - 桌面端可随时断开连接
   - 支持白名单/黑名单
   - 会话超时自动断开（30 分钟无活动）

### 实现步骤

#### 阶段 1：基础架构（1-2 周）
- [ ] 实现中继服务器（Go + WebSocket）
- [ ] 桌面端添加 RemoteControlManager
- [ ] 实现连接码生成和配对逻辑
- [ ] 基础消息转发功能

#### 阶段 2：桌面端集成（1 周）
- [ ] 在设置面板添加"远程控制"标签
- [ ] 实现启动/停止远程会话 UI
- [ ] 显示连接码和二维码
- [ ] 会话状态监控

#### 阶段 3：手机端 PWA（2 周）
- [ ] 创建 Vue.js PWA 项目
- [ ] 实现连接码输入界面
- [ ] 实现聊天界面
- [ ] 实现文件浏览功能
- [ ] 实现终端输出查看

#### 阶段 4：加密和安全（1 周）
- [ ] 实现端到端加密
- [ ] 添加会话认证
- [ ] 实现访问控制

#### 阶段 5：优化和测试（1 周）
- [ ] 断线重连机制
- [ ] 性能优化
- [ ] 跨平台测试
- [ ] 用户体验优化

### 技术细节

#### 1. 连接码生成
```go
func GenerateConnectionCode() string {
    // 生成 6 位数字码（100000-999999）
    code := rand.Intn(900000) + 100000
    return fmt.Sprintf("%06d", code)
}
```

#### 2. WebSocket 消息格式
```json
{
  "type": "message",
  "sessionId": "abc123",
  "encrypted": true,
  "payload": "base64_encrypted_data",
  "timestamp": 1234567890
}
```

#### 3. 加密实现
```go
// 使用 Go 标准库 crypto/aes
func EncryptMessage(key []byte, plaintext string) (string, error) {
    block, _ := aes.NewCipher(key)
    gcm, _ := cipher.NewGCM(block)
    nonce := make([]byte, gcm.NonceSize())
    rand.Read(nonce)
    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}
```

### 用户体验设计

#### 桌面端
```
┌─────────────────────────────────────┐
│ 设置 > 远程控制                      │
├─────────────────────────────────────┤
│                                     │
│  ○ 远程控制已关闭                    │
│                                     │
│  [启动远程会话]                      │
│                                     │
│  说明：启动后可从手机控制此应用       │
│                                     │
└─────────────────────────────────────┘

启动后：
┌─────────────────────────────────────┐
│ 设置 > 远程控制                      │
├─────────────────────────────────────┤
│                                     │
│  ● 远程控制已启动                    │
│                                     │
│  连接码：  1 2 3 4 5 6              │
│                                     │
│  [显示二维码]  [停止会话]            │
│                                     │
│  已连接设备：iPhone (192.168.1.100) │
│  会话时长：00:15:32                  │
│                                     │
└─────────────────────────────────────┘
```

#### 手机端
```
┌─────────────────────┐
│  Kiro Remote        │
├─────────────────────┤
│                     │
│  输入连接码：        │
│  ┌─┬─┬─┬─┬─┬─┐      │
│  │1│2│3│4│5│6│      │
│  └─┴─┴─┴─┴─┴─┘      │
│                     │
│  [连接]  [扫码]      │
│                     │
└─────────────────────┘

连接后：
┌─────────────────────┐
│  Kiro Remote   [⚙]  │
├─────────────────────┤
│ 💬 Chat             │
│ 🖥️ Desktop          │
│ 📁 Files            │
│ 💻 Terminal         │
├─────────────────────┤
│                     │
│  [当前标签内容]      │
│                     │
└─────────────────────┘

桌面查看界面：
┌─────────────────────┐
│ 桌面查看  [高清][快速]│
├─────────────────────┤
│                     │
│   [屏幕截图显示]     │
│                     │
│   点击可全屏查看     │
│                     │
├─────────────────────┤
│ 更新: 14:23:15      │
│ 大小: 125 KB        │
└─────────────────────┘
```

### 成本估算

#### 开发成本
- 中继服务器：3-5 天
- 桌面端集成：5-7 天
- 手机端 PWA：10-14 天
- 加密和安全：3-5 天
- 测试和优化：5-7 天
- **总计：约 4-6 周**

#### 运营成本
- 服务器（1 核 1G）：$5-10/月
- 域名：$10-15/年
- SSL 证书：免费（Let's Encrypt）
- **总计：约 $60-130/年**

### 替代方案

#### 方案 1：使用现有服务
- **Ngrok/Cloudflare Tunnel**：快速实现，但依赖第三方
- **优点**：无需自建服务器，快速上线
- **缺点**：免费版有限制，数据经过第三方

#### 方案 2：P2P 直连
- **WebRTC**：点对点连接，无需中继服务器
- **优点**：低延迟，无服务器成本
- **缺点**：NAT 穿透复杂，需要 STUN/TURN 服务器

#### 方案 3：简化版（仅查看）
- **只读模式**：手机端只能查看，不能操作
- **优点**：实现简单，安全性高
- **缺点**：功能受限

### 推荐方案

**第一阶段（MVP）**：
1. 使用 Ngrok 快速实现原型（1 周）
2. 手机端 PWA 只实现聊天功能
3. 不实现加密（仅用于测试）

**第二阶段（生产版）**：
1. 自建中继服务器（Go + WebSocket）
2. 实现端到端加密
3. 完善手机端功能（文件、终端）

**第三阶段（增强版）**：
1. 开发原生移动应用
2. 添加语音输入
3. 支持多设备同时连接

### 风险评估

#### 技术风险
- **网络延迟**：移动网络可能不稳定
  - 缓解：实现消息队列和重试机制
- **加密性能**：加密可能影响性能
  - 缓解：使用硬件加速，优化算法

#### 安全风险
- **中间人攻击**：中继服务器可能被攻击
  - 缓解：端到端加密，证书固定
- **连接码泄露**：6 位码可能被猜测
  - 缓解：限制尝试次数，短有效期

#### 用户体验风险
- **连接复杂**：用户可能不会使用
  - 缓解：提供详细教程，二维码扫描
- **功能受限**：手机屏幕小，操作不便
  - 缓解：优化 UI，提供语音输入

## 远程桌面查看功能

### 需求分析

用户需要从手机端查看电脑桌面，监控 AI 编程进度。

### 技术方案对比

#### 方案 1：屏幕截图流（推荐）

**原理**：
- 桌面端定期截取屏幕
- 压缩后通过 WebSocket 发送到手机
- 手机端显示最新截图

**优点**：
- ✅ 实现简单
- ✅ 带宽占用可控
- ✅ 跨平台兼容性好
- ✅ 不需要额外的编解码库

**缺点**：
- ❌ 不是实时视频（有延迟）
- ❌ 帧率较低（1-5 FPS）

**适用场景**：
- 监控编程进度（不需要实时）
- 查看代码变化
- 检查终端输出

#### 方案 2：视频流（H.264）

**原理**：
- 使用 FFmpeg 捕获屏幕
- H.264 编码压缩
- WebRTC 或 WebSocket 传输
- 手机端解码播放

**优点**：
- ✅ 流畅的视频体验
- ✅ 高帧率（15-30 FPS）
- ✅ 压缩率高

**缺点**：
- ❌ 实现复杂
- ❌ 需要 FFmpeg 依赖
- ❌ CPU 占用较高
- ❌ 移动网络带宽要求高

**适用场景**：
- 需要实时查看
- 观看动画或视频
- 远程演示

#### 方案 3：VNC/RDP 协议

**原理**：
- 使用现有的远程桌面协议
- 集成 VNC 服务器
- 手机端使用 VNC 客户端

**优点**：
- ✅ 成熟的协议
- ✅ 支持鼠标键盘控制
- ✅ 有现成的库

**缺点**：
- ❌ 需要额外的服务器
- ❌ 配置复杂
- ❌ 安全性需要额外处理

### 推荐实现：屏幕截图流

基于我们的使用场景（监控编程进度），推荐使用**屏幕截图流**方案。

### 技术实现

#### 1. 桌面端截图（Go）

**跨平台截图库**：
- Windows: `github.com/kbinani/screenshot`
- macOS: `github.com/kbinani/screenshot`
- Linux: `github.com/kbinani/screenshot`

**代码实现**：
```go
// myapp/screen_capture.go
package main

import (
    "bytes"
    "image"
    "image/jpeg"
    "time"
    
    "github.com/kbinani/screenshot"
)

type ScreenCaptureManager struct {
    active       bool
    interval     time.Duration
    quality      int
    displayIndex int
    lastCapture  []byte
}

func NewScreenCaptureManager() *ScreenCaptureManager {
    return &ScreenCaptureManager{
        interval:     time.Second * 2, // 每 2 秒截图一次
        quality:      60,               // JPEG 质量 60%
        displayIndex: 0,                // 主显示器
    }
}

// CaptureScreen 截取屏幕
func (s *ScreenCaptureManager) CaptureScreen() ([]byte, error) {
    // 获取显示器数量
    n := screenshot.NumActiveDisplays()
    if s.displayIndex >= n {
        s.displayIndex = 0
    }
    
    // 获取显示器边界
    bounds := screenshot.GetDisplayBounds(s.displayIndex)
    
    // 截图
    img, err := screenshot.CaptureRect(bounds)
    if err != nil {
        return nil, err
    }
    
    // 压缩为 JPEG
    return s.compressImage(img)
}

// compressImage 压缩图片
func (s *ScreenCaptureManager) compressImage(img *image.RGBA) ([]byte, error) {
    var buf bytes.Buffer
    
    // 可选：缩小分辨率以减少带宽
    // resized := resize.Resize(1280, 0, img, resize.Lanczos3)
    
    // JPEG 压缩
    err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: s.quality})
    if err != nil {
        return nil, err
    }
    
    return buf.Bytes(), nil
}

// StartCapture 开始定期截图
func (s *ScreenCaptureManager) StartCapture(callback func([]byte)) {
    s.active = true
    go func() {
        ticker := time.NewTicker(s.interval)
        defer ticker.Stop()
        
        for s.active {
            select {
            case <-ticker.C:
                if data, err := s.CaptureScreen(); err == nil {
                    s.lastCapture = data
                    callback(data)
                }
            }
        }
    }()
}

// StopCapture 停止截图
func (s *ScreenCaptureManager) StopCapture() {
    s.active = false
}

// GetLastCapture 获取最后一次截图
func (s *ScreenCaptureManager) GetLastCapture() []byte {
    return s.lastCapture
}

// SetQuality 设置 JPEG 质量 (1-100)
func (s *ScreenCaptureManager) SetQuality(quality int) {
    if quality >= 1 && quality <= 100 {
        s.quality = quality
    }
}

// SetInterval 设置截图间隔
func (s *ScreenCaptureManager) SetInterval(interval time.Duration) {
    s.interval = interval
}

// SetDisplay 设置要截取的显示器
func (s *ScreenCaptureManager) SetDisplay(index int) {
    s.displayIndex = index
}

// GetDisplayCount 获取显示器数量
func (s *ScreenCaptureManager) GetDisplayCount() int {
    return screenshot.NumActiveDisplays()
}
```

#### 2. 集成到 RemoteControlManager

```go
// myapp/remote_control.go
type RemoteControlManager struct {
    app            *App
    wsClient       *websocket.Conn
    sessionID      string
    connectionCode string
    active         bool
    messageQueue   chan RemoteMessage
    screenCapture  *ScreenCaptureManager  // 新增
    screenEnabled  bool                    // 新增
}

// EnableScreenSharing 启用屏幕共享
func (r *RemoteControlManager) EnableScreenSharing() {
    r.screenEnabled = true
    r.screenCapture.StartCapture(func(data []byte) {
        // 发送截图到手机端
        r.SendMessage(RemoteMessage{
            Type:    "screen",
            Content: base64.StdEncoding.EncodeToString(data),
        })
    })
}

// DisableScreenSharing 禁用屏幕共享
func (r *RemoteControlManager) DisableScreenSharing() {
    r.screenEnabled = false
    r.screenCapture.StopCapture()
}

// SetScreenQuality 设置屏幕质量
func (r *RemoteControlManager) SetScreenQuality(quality int) {
    r.screenCapture.SetQuality(quality)
}

// SetScreenInterval 设置截图间隔（秒）
func (r *RemoteControlManager) SetScreenInterval(seconds int) {
    r.screenCapture.SetInterval(time.Duration(seconds) * time.Second)
}
```

#### 3. App API 接口

```go
// myapp/app.go

// EnableRemoteScreenSharing 启用远程屏幕共享
func (a *App) EnableRemoteScreenSharing() error {
    if a.remoteMgr == nil || !a.remoteMgr.active {
        return fmt.Errorf("remote session not active")
    }
    a.remoteMgr.EnableScreenSharing()
    return nil
}

// DisableRemoteScreenSharing 禁用远程屏幕共享
func (a *App) DisableRemoteScreenSharing() error {
    if a.remoteMgr == nil {
        return fmt.Errorf("remote manager not initialized")
    }
    a.remoteMgr.DisableScreenSharing()
    return nil
}

// SetRemoteScreenQuality 设置屏幕质量 (1-100)
func (a *App) SetRemoteScreenQuality(quality int) error {
    if a.remoteMgr == nil {
        return fmt.Errorf("remote manager not initialized")
    }
    a.remoteMgr.SetScreenQuality(quality)
    return nil
}

// SetRemoteScreenInterval 设置截图间隔（秒）
func (a *App) SetRemoteScreenInterval(seconds int) error {
    if a.remoteMgr == nil {
        return fmt.Errorf("remote manager not initialized")
    }
    a.remoteMgr.SetScreenInterval(seconds)
    return nil
}

// GetRemoteScreenshot 获取当前屏幕截图
func (a *App) GetRemoteScreenshot() (string, error) {
    if a.remoteMgr == nil || a.remoteMgr.screenCapture == nil {
        return "", fmt.Errorf("screen capture not initialized")
    }
    data := a.remoteMgr.screenCapture.GetLastCapture()
    if data == nil {
        return "", fmt.Errorf("no screenshot available")
    }
    return base64.StdEncoding.EncodeToString(data), nil
}
```

#### 4. 手机端显示

**Vue 组件** (`ScreenViewer.vue`):
```vue
<template>
  <div class="screen-viewer">
    <div class="screen-header">
      <h3>桌面查看</h3>
      <div class="controls">
        <button @click="toggleQuality">
          {{ quality === 'high' ? '高清' : quality === 'medium' ? '标清' : '省流' }}
        </button>
        <button @click="toggleInterval">
          {{ interval === 1 ? '快速' : interval === 2 ? '正常' : '慢速' }}
        </button>
        <button @click="refresh">刷新</button>
      </div>
    </div>
    
    <div class="screen-container" @click="toggleFullscreen">
      <img 
        v-if="screenData" 
        :src="'data:image/jpeg;base64,' + screenData" 
        alt="Desktop Screen"
        class="screen-image"
      />
      <div v-else class="screen-placeholder">
        <p>等待屏幕数据...</p>
      </div>
    </div>
    
    <div class="screen-info">
      <span>最后更新: {{ lastUpdate }}</span>
      <span>大小: {{ screenSize }}</span>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'

const screenData = ref(null)
const lastUpdate = ref('--:--:--')
const screenSize = ref('0 KB')
const quality = ref('medium') // low, medium, high
const interval = ref(2) // 1, 2, 5 秒

// 质量映射
const qualityMap = {
  low: 40,
  medium: 60,
  high: 85
}

// 接收屏幕数据
const handleScreenData = (data) => {
  screenData.value = data
  lastUpdate.value = new Date().toLocaleTimeString()
  
  // 计算大小
  const bytes = atob(data).length
  screenSize.value = (bytes / 1024).toFixed(1) + ' KB'
}

// 切换质量
const toggleQuality = () => {
  const qualities = ['low', 'medium', 'high']
  const currentIndex = qualities.indexOf(quality.value)
  quality.value = qualities[(currentIndex + 1) % qualities.length]
  
  // 通知后端
  window.SetRemoteScreenQuality(qualityMap[quality.value])
}

// 切换间隔
const toggleInterval = () => {
  const intervals = [1, 2, 5]
  const currentIndex = intervals.indexOf(interval.value)
  interval.value = intervals[(currentIndex + 1) % intervals.length]
  
  // 通知后端
  window.SetRemoteScreenInterval(interval.value)
}

// 刷新
const refresh = async () => {
  try {
    const data = await window.GetRemoteScreenshot()
    handleScreenData(data)
  } catch (e) {
    console.error('刷新失败:', e)
  }
}

// 全屏切换
const toggleFullscreen = () => {
  const elem = document.querySelector('.screen-container')
  if (!document.fullscreenElement) {
    elem.requestFullscreen()
  } else {
    document.exitFullscreen()
  }
}

onMounted(() => {
  // 监听屏幕数据
  window.addEventListener('screen-data', (e) => {
    handleScreenData(e.detail)
  })
})
</script>

<style scoped>
.screen-viewer {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #000;
}

.screen-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  background: rgba(255, 255, 255, 0.1);
}

.screen-header h3 {
  margin: 0;
  color: #fff;
  font-size: 16px;
}

.controls {
  display: flex;
  gap: 8px;
}

.controls button {
  padding: 6px 12px;
  background: rgba(255, 255, 255, 0.2);
  border: none;
  border-radius: 4px;
  color: #fff;
  font-size: 12px;
  cursor: pointer;
}

.controls button:hover {
  background: rgba(255, 255, 255, 0.3);
}

.screen-container {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  cursor: pointer;
}

.screen-image {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
}

.screen-placeholder {
  color: rgba(255, 255, 255, 0.5);
  text-align: center;
}

.screen-info {
  display: flex;
  justify-content: space-between;
  padding: 8px 12px;
  background: rgba(255, 255, 255, 0.1);
  color: rgba(255, 255, 255, 0.7);
  font-size: 11px;
}

/* 全屏样式 */
.screen-container:fullscreen {
  background: #000;
}

.screen-container:fullscreen .screen-image {
  width: 100%;
  height: 100%;
}
</style>
```

#### 5. 手机端主界面集成

```vue
<!-- 手机端 App.vue -->
<template>
  <div class="mobile-app">
    <div class="tabs">
      <button @click="activeTab = 'chat'" :class="{ active: activeTab === 'chat' }">
        💬 聊天
      </button>
      <button @click="activeTab = 'screen'" :class="{ active: activeTab === 'screen' }">
        🖥️ 桌面
      </button>
      <button @click="activeTab = 'files'" :class="{ active: activeTab === 'files' }">
        📁 文件
      </button>
      <button @click="activeTab = 'terminal'" :class="{ active: activeTab === 'terminal' }">
        💻 终端
      </button>
    </div>
    
    <div class="content">
      <ChatPanel v-if="activeTab === 'chat'" />
      <ScreenViewer v-if="activeTab === 'screen'" />
      <FileExplorer v-if="activeTab === 'files'" />
      <TerminalViewer v-if="activeTab === 'terminal'" />
    </div>
  </div>
</template>
```

### 性能优化

#### 1. 自适应质量

根据网络状况自动调整质量：

```go
func (s *ScreenCaptureManager) AutoAdjustQuality(latency time.Duration) {
    if latency > 2*time.Second {
        s.quality = 40 // 低质量
        s.interval = 5 * time.Second
    } else if latency > 1*time.Second {
        s.quality = 60 // 中等质量
        s.interval = 2 * time.Second
    } else {
        s.quality = 80 // 高质量
        s.interval = 1 * time.Second
    }
}
```

#### 2. 增量更新

只发送变化的区域：

```go
func (s *ScreenCaptureManager) CaptureChanges(previous *image.RGBA) ([]byte, error) {
    current, _ := s.CaptureScreen()
    
    // 比较两张图片，只编码变化的区域
    changes := s.detectChanges(previous, current)
    
    return s.compressChanges(changes)
}
```

#### 3. 压缩优化

- 使用 WebP 格式（比 JPEG 小 25-35%）
- 降低分辨率（1920x1080 → 1280x720）
- 只截取应用窗口而非整个屏幕

### 带宽估算

**JPEG 压缩（质量 60%）**：
- 1920x1080 全屏：约 150-300 KB/帧
- 1280x720 缩小：约 80-150 KB/帧

**不同更新频率的带宽**：
- 1 FPS (每秒 1 帧)：80-300 KB/s = 0.6-2.4 Mbps
- 0.5 FPS (每 2 秒)：40-150 KB/s = 0.3-1.2 Mbps
- 0.2 FPS (每 5 秒)：16-60 KB/s = 0.1-0.5 Mbps

**推荐配置**：
- WiFi：高质量 + 1 FPS
- 4G：中等质量 + 0.5 FPS
- 3G：低质量 + 0.2 FPS

### 桌面端 UI 设置

```vue
<!-- SettingsPanel.vue 中的远程控制设置 -->
<div class="remote-control-settings">
  <h3>远程控制</h3>
  
  <div class="setting-group">
    <label>
      <input type="checkbox" v-model="screenSharingEnabled" />
      启用屏幕共享
    </label>
  </div>
  
  <div v-if="screenSharingEnabled" class="screen-settings">
    <div class="setting-item">
      <label>屏幕质量</label>
      <select v-model="screenQuality">
        <option value="40">省流模式 (40%)</option>
        <option value="60">标准模式 (60%)</option>
        <option value="80">高清模式 (80%)</option>
      </select>
    </div>
    
    <div class="setting-item">
      <label>更新频率</label>
      <select v-model="screenInterval">
        <option value="1">快速 (1 秒)</option>
        <option value="2">正常 (2 秒)</option>
        <option value="5">慢速 (5 秒)</option>
      </select>
    </div>
    
    <div class="setting-item">
      <label>显示器</label>
      <select v-model="displayIndex">
        <option v-for="i in displayCount" :key="i" :value="i-1">
          显示器 {{ i }}
        </option>
      </select>
    </div>
  </div>
</div>
```

### 总结

这个方案是可行的，参考 Happy 项目的设计理念，结合我们现有的架构，可以实现一个功能完善的手机端控制系统。

**关键优势**：
- ✅ 最小化桌面端改动
- ✅ 独立的中继服务器，易于维护
- ✅ PWA 方案快速上线
- ✅ 端到端加密保证安全
- ✅ 成本可控

**建议**：
1. 先实现 MVP 版本验证可行性
2. 收集用户反馈后再完善功能
3. 考虑开源中继服务器代码
