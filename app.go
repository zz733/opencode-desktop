package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx           context.Context
	serverURL     string
	httpClient    *http.Client
	termMgr       *TerminalManager
	openCode      *OpenCodeManager
	fileMgr       *FileManager
	sseCancel     context.CancelFunc // 用于取消 SSE 订阅
	sseSubscribed bool
}

// NewApp creates a new App application struct
func NewApp() *App {
	app := &App{
		serverURL: "http://localhost:4096",
		httpClient: &http.Client{
			Timeout: 0, // no timeout for SSE
		},
	}
	app.termMgr = NewTerminalManager(app)
	app.openCode = NewOpenCodeManager(app)
	app.fileMgr = NewFileManager(app)
	return app
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// SetServerURL 设置服务器地址
func (a *App) SetServerURL(url string) {
	a.serverURL = strings.TrimSuffix(url, "/")
}

// GetServerURL 获取服务器地址
func (a *App) GetServerURL() string {
	return a.serverURL
}

// Session 会话信息
type Session struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// Message 消息
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// GetSessions 获取会话列表
func (a *App) GetSessions() ([]Session, error) {
	resp, err := a.httpClient.Get(a.serverURL + "/session")
	if err != nil {
		return nil, fmt.Errorf("连接失败: %v", err)
	}
	defer resp.Body.Close()

	var sessions []Session
	if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

// CreateSession 创建新会话
func (a *App) CreateSession() (*Session, error) {
	resp, err := a.httpClient.Post(a.serverURL+"/session", "application/json", bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return nil, fmt.Errorf("创建会话失败: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Info Session `json:"info"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result.Info, nil
}

// SendMessage 发送消息（异步，不等待响应）
func (a *App) SendMessage(sessionID, content string) error {
	payload := map[string]interface{}{
		"parts": []map[string]interface{}{
			{"type": "text", "text": content},
		},
	}
	body, _ := json.Marshal(payload)

	// 使用异步接口，立即返回
	url := fmt.Sprintf("%s/session/%s/prompt_async", a.serverURL, sessionID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 && resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("请求失败 %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// SubscribeEvents 订阅服务器事件
func (a *App) SubscribeEvents() error {
	// 取消之前的订阅
	if a.sseCancel != nil {
		runtime.EventsEmit(a.ctx, "output-log", "取消旧的事件订阅...")
		a.sseCancel()
		a.sseCancel = nil
		time.Sleep(100 * time.Millisecond) // 等待旧连接关闭
	}

	// 创建新的取消上下文
	ctx, cancel := context.WithCancel(context.Background())
	a.sseCancel = cancel
	a.sseSubscribed = true

	// 保存当前的 serverURL，避免在循环中被修改
	serverURL := a.serverURL

	go func() {
		for {
			select {
			case <-ctx.Done():
				runtime.EventsEmit(a.ctx, "output-log", "事件订阅已取消")
				return
			default:
			}

			url := serverURL + "/event"
			runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("订阅事件: %s", url))

			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("创建请求失败: %v", err))
				time.Sleep(3 * time.Second)
				continue
			}

			resp, err := a.httpClient.Do(req)
			if err != nil {
				if ctx.Err() != nil {
					return // 上下文已取消
				}
				runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("订阅失败: %v", err))
				runtime.EventsEmit(a.ctx, "connection-error", err.Error())
				time.Sleep(3 * time.Second)
				continue
			}

			runtime.EventsEmit(a.ctx, "output-log", "事件订阅成功，等待事件...")
			reader := bufio.NewReader(resp.Body)
			var dataBuffer bytes.Buffer

			for {
				select {
				case <-ctx.Done():
					resp.Body.Close()
					return
				default:
				}

				line, err := reader.ReadString('\n')
				if err != nil {
					resp.Body.Close()
					runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("读取事件流中断: %v", err))
					break
				}
				
				// 处理 SSE 协议
				// 1. 空行表示事件结束，发送累积的数据
				if strings.TrimSpace(line) == "" {
					if dataBuffer.Len() > 0 {
						data := dataBuffer.String()
						// runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("收到完整事件: %s", data[:min(100, len(data))]))
						runtime.EventsEmit(a.ctx, "server-event", data)
						dataBuffer.Reset()
					}
					continue
				}

				// 2. data 行累积数据
				if strings.HasPrefix(line, "data:") {
					data := strings.TrimPrefix(line, "data:")
					// 如果有多行 data，用换行符连接
					if dataBuffer.Len() > 0 {
						dataBuffer.WriteString("\n")
					}
					dataBuffer.WriteString(strings.TrimSpace(data))
				}
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Second):
			}
		}
	}()
	return nil
}

// CheckConnection 检查连接状态
func (a *App) CheckConnection() (bool, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(a.serverURL + "/session")
	if err != nil {
		return false, fmt.Errorf("无法连接到 %s", a.serverURL)
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200, nil
}

// Provider 信息
type Provider struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Model 信息
type Model struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
}

// ProviderInfo 完整的 provider 信息
type ProviderInfo struct {
	All       []Provider        `json:"all"`
	Connected []string          `json:"connected"`
	Default   map[string]string `json:"default"`
}

// GetProviders 获取所有 provider 和模型信息
func (a *App) GetProviders() (*ProviderInfo, error) {
	resp, err := a.httpClient.Get(a.serverURL + "/provider")
	if err != nil {
		return nil, fmt.Errorf("获取 provider 失败: %v", err)
	}
	defer resp.Body.Close()

	var info ProviderInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

// ConfigInfo 配置信息
type ConfigInfo struct {
	Model string `json:"model"`
}

// GetConfig 获取当前配置
func (a *App) GetConfig() (*ConfigInfo, error) {
	resp, err := a.httpClient.Get(a.serverURL + "/config")
	if err != nil {
		return nil, fmt.Errorf("获取配置失败: %v", err)
	}
	defer resp.Body.Close()

	var config ConfigInfo
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

// ImageData 图片数据
type ImageData struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Data string `json:"data"` // base64 编码的图片数据
}

// SaveImageToWorkDir 将图片保存到工作目录
func (a *App) SaveImageToWorkDir(img ImageData) (string, error) {
	workDir := a.openCode.GetWorkDir()
	if workDir == "" {
		// 如果工作目录未设置，使用用户主目录
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("获取主目录失败: %v", err)
		}
		workDir = homeDir
	}

	// 创建 .opencode-images 目录
	imgDir := filepath.Join(workDir, ".opencode-images")
	if err := os.MkdirAll(imgDir, 0755); err != nil {
		return "", fmt.Errorf("创建图片目录失败: %v", err)
	}

	// 从 base64 data URL 中提取数据
	dataURL := img.Data
	if !strings.HasPrefix(dataURL, "data:") {
		return "", fmt.Errorf("无效的图片数据格式")
	}

	urlParts := strings.SplitN(dataURL, ",", 2)
	if len(urlParts) != 2 {
		return "", fmt.Errorf("无效的 base64 数据")
	}

	// 解码 base64
	imgData, err := base64.StdEncoding.DecodeString(urlParts[1])
	if err != nil {
		return "", fmt.Errorf("解码图片失败: %v", err)
	}

	// 生成文件名
	ext := ".png"
	if strings.Contains(img.Type, "jpeg") || strings.Contains(img.Type, "jpg") {
		ext = ".jpg"
	} else if strings.Contains(img.Type, "gif") {
		ext = ".gif"
	} else if strings.Contains(img.Type, "webp") {
		ext = ".webp"
	}
	
	filename := fmt.Sprintf("img_%d%s", time.Now().UnixNano(), ext)
	filePath := filepath.Join(imgDir, filename)

	// 写入文件
	if err := os.WriteFile(filePath, imgData, 0644); err != nil {
		return "", fmt.Errorf("保存图片失败: %v", err)
	}

	// 返回相对路径
	relPath := filepath.Join(".opencode-images", filename)
	return relPath, nil
}

// SendMessageWithModel 发送消息并指定模型（支持图片）
func (a *App) SendMessageWithModel(sessionID, content, model string, images []ImageData) error {
	// 构建消息内容
	messageText := content
	
	// 如果有图片，保存到工作目录并在消息中引用
	if len(images) > 0 {
		var imagePaths []string
		for _, img := range images {
			relPath, err := a.SaveImageToWorkDir(img)
			if err != nil {
				runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("保存图片失败: %v", err))
				continue
			}
			imagePaths = append(imagePaths, relPath)
			runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("图片已保存: %s", relPath))
		}
		
		if len(imagePaths) > 0 {
			messageText += "\n\n[Attached images - please read and analyze these image files:]"
			for _, p := range imagePaths {
				messageText += fmt.Sprintf("\n- %s", p)
			}
		}
	}
	
	parts := []map[string]interface{}{
		{"type": "text", "text": messageText},
	}
	
	payload := map[string]interface{}{
		"parts": parts,
	}
	
	if model != "" {
		// model 格式: provider/modelID，需要拆分成对象
		modelParts := strings.SplitN(model, "/", 2)
		if len(modelParts) == 2 {
			payload["model"] = map[string]string{
				"providerID": modelParts[0],
				"modelID":    modelParts[1],
			}
		}
	}
	body, _ := json.Marshal(payload)

	url := fmt.Sprintf("%s/session/%s/prompt_async", a.serverURL, sessionID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 && resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("请求失败 %d: %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

// CancelSession 取消会话中正在进行的请求
func (a *App) CancelSession(sessionID string) error {
	url := fmt.Sprintf("%s/session/%s/cancel", a.serverURL, sessionID)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("取消失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 && resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("取消失败 %d: %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

// SetActiveFile 设置当前活动文件（注入上下文）
func (a *App) SetActiveFile(sessionID, filePath string) error {
	if sessionID == "" || filePath == "" {
		return nil
	}
	
	// 使用 noReply: true 注入上下文，不触发 AI 响应
	payload := map[string]interface{}{
		"noReply": true,
		"parts": []map[string]interface{}{
			{"type": "text", "text": fmt.Sprintf("[Current active file: %s]", filePath)},
		},
	}
	body, _ := json.Marshal(payload)

	url := fmt.Sprintf("%s/session/%s/prompt", a.serverURL, sessionID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("设置活动文件失败: %v", err)
	}
	defer resp.Body.Close()

	return nil
}

// OpenCodeMessage OpenCode 返回的消息格式
type OpenCodeMessage struct {
	Info struct {
		ID        string `json:"id"`
		SessionID string `json:"sessionID"`
		Role      string `json:"role"`
	} `json:"info"`
	Parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"parts"`
}

// GetSessionMessages 获取会话的历史消息
func (a *App) GetSessionMessages(sessionID string) ([]Message, error) {
	url := fmt.Sprintf("%s/session/%s/message", a.serverURL, sessionID)
	runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("获取历史消息: %s", url))
	
	resp, err := a.httpClient.Get(url)
	if err != nil {
		runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("获取消息失败: %v", err))
		return nil, fmt.Errorf("获取消息失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, _ := io.ReadAll(resp.Body)
	
	// 解析 OpenCode 格式的消息
	var ocMessages []OpenCodeMessage
	if err := json.Unmarshal(body, &ocMessages); err != nil {
		runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("解析消息失败: %v", err))
		return nil, err
	}
	
	// 转换为简单格式
	var messages []Message
	for _, ocMsg := range ocMessages {
		// 提取文本内容
		var content string
		for _, part := range ocMsg.Parts {
			if part.Type == "text" && part.Text != "" {
				content = part.Text
				break
			}
		}
		
		if ocMsg.Info.Role != "" && content != "" {
			messages = append(messages, Message{
				Role:    ocMsg.Info.Role,
				Content: content,
			})
		}
	}
	
	runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("解析到 %d 条消息，转换后 %d 条", len(ocMessages), len(messages)))
	return messages, nil
}

// CodeCompletion 代码补全请求
func (a *App) CodeCompletion(sessionID, code, language, filename string) (string, error) {
	// 使用更简洁的 prompt，让 AI 只返回补全内容
	prompt := fmt.Sprintf(`You are a code completion assistant. Complete the following %s code.
IMPORTANT: Only output the completion text that should be inserted at the cursor position. No explanation, no markdown, no code blocks.
If no completion is needed, output nothing.

File: %s
Code before cursor:
%s

Completion:`, language, filename, code)

	payload := map[string]interface{}{
		"parts": []map[string]interface{}{
			{"type": "text", "text": prompt},
		},
	}
	body, _ := json.Marshal(payload)

	// 使用同步 prompt 接口，添加 Accept header
	url := fmt.Sprintf("%s/session/%s/prompt", a.serverURL, sessionID)
	runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("请求代码补全: %s", url))
	
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("创建补全请求失败: %v", err))
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("补全请求失败: %v", err))
		return "", err
	}
	defer resp.Body.Close()

	// 读取响应体
	respBody, _ := io.ReadAll(resp.Body)
	
	// 检查是否返回了 HTML（说明 API 不支持这个端点）
	if strings.HasPrefix(string(respBody), "<!") || strings.HasPrefix(string(respBody), "<html") {
		runtime.EventsEmit(a.ctx, "output-log", "补全 API 返回了 HTML，可能不支持同步 prompt 接口")
		return "", fmt.Errorf("API 返回了 HTML 而不是 JSON")
	}
	
	runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("补全响应状态: %d, 内容: %s", resp.StatusCode, string(respBody[:min(200, len(respBody))])))

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("请求失败: %d", resp.StatusCode)
	}

	// 解析响应
	var result struct {
		Parts []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"parts"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("解析补全响应失败: %v", err))
		return "", err
	}

	// 提取文本内容
	for _, part := range result.Parts {
		if part.Type == "text" && part.Text != "" {
			// 清理返回的代码
			text := strings.TrimSpace(part.Text)
			// 移除可能的 markdown 代码块标记
			if strings.HasPrefix(text, "```") {
				lines := strings.Split(text, "\n")
				if len(lines) > 2 {
					// 移除首尾的 ``` 行
					text = strings.Join(lines[1:len(lines)-1], "\n")
				}
			}
			text = strings.TrimSpace(text)
			runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("补全结果: %s", text[:min(100, len(text))]))
			return text, nil
		}
	}

	return "", nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// CreateTerminal 创建新终端
func (a *App) CreateTerminal() (int, error) {
	return a.termMgr.CreateTerminal()
}

// WriteTerminal 写入终端
func (a *App) WriteTerminal(id int, data string) error {
	return a.termMgr.WriteTerminal(id, data)
}

// ResizeTerminal 调整终端大小
func (a *App) ResizeTerminal(id int, cols, rows int) error {
	return a.termMgr.ResizeTerminal(id, cols, rows)
}

// CloseTerminal 关闭终端
func (a *App) CloseTerminal(id int) {
	a.termMgr.CloseTerminal(id)
}

// GetTerminals 获取所有终端
func (a *App) GetTerminals() []int {
	return a.termMgr.GetTerminals()
}

// --- OpenCode 管理 ---

// GetOpenCodeStatus 获取 OpenCode 状态
func (a *App) GetOpenCodeStatus() *OpenCodeStatus {
	return a.openCode.GetStatus()
}

// InstallOpenCode 安装 OpenCode
func (a *App) InstallOpenCode() error {
	return a.openCode.Install()
}

// StartOpenCode 启动 OpenCode
func (a *App) StartOpenCode() error {
	return a.openCode.Start()
}

// StopOpenCode 停止 OpenCode
func (a *App) StopOpenCode() {
	a.openCode.Stop()
}

// AutoStartOpenCode 自动检测并启动 OpenCode
func (a *App) AutoStartOpenCode() error {
	return a.openCode.AutoStart()
}

// SetOpenCodeWorkDir 设置 OpenCode 工作目录
func (a *App) SetOpenCodeWorkDir(dir string) error {
	a.openCode.SetWorkDir(dir)
	// 同时更新文件管理器的根目录
	a.fileMgr.SetRootDir(dir)
	// 不自动重启，让 autoConnect 处理连接
	return nil
}

// --- Oh My OpenCode ---

// OhMyOpenCodeStatus oh-my-opencode 状态
type OhMyOpenCodeStatus struct {
	Installed bool   `json:"installed"`
	Version   string `json:"version"`
}

// GetOhMyOpenCodeStatus 获取 oh-my-opencode 状态
func (a *App) GetOhMyOpenCodeStatus() *OhMyOpenCodeStatus {
	status := &OhMyOpenCodeStatus{Installed: false}
	status.Installed, status.Version = a.checkPluginInstalled("oh-my-opencode")
	return status
}

// AntigravityAuthStatus antigravity-auth 状态
type AntigravityAuthStatus struct {
	Installed bool   `json:"installed"`
	Version   string `json:"version"`
}

// GetAntigravityAuthStatus 获取 antigravity-auth 状态
func (a *App) GetAntigravityAuthStatus() *AntigravityAuthStatus {
	status := &AntigravityAuthStatus{Installed: false}
	status.Installed, status.Version = a.checkPluginInstalled("opencode-antigravity-auth")
	return status
}

// checkPluginInstalled 检查插件是否安装
func (a *App) checkPluginInstalled(pluginName string) (bool, string) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false, ""
	}

	configPath := filepath.Join(homeDir, ".config", "opencode", "opencode.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false, ""
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return false, ""
	}

	// 检查 plugin 数组
	if plugins, ok := config["plugin"].([]interface{}); ok {
		for _, p := range plugins {
			if ps, ok := p.(string); ok && strings.Contains(ps, pluginName) {
				version := ""
				if strings.Contains(ps, "@") {
					parts := strings.Split(ps, "@")
					if len(parts) > 1 {
						version = parts[len(parts)-1]
					}
				}
				return true, version
			}
		}
	}
	
	// 也检查 plugins 字段（复数形式，兼容旧配置）
	if plugins, ok := config["plugins"].([]interface{}); ok {
		for _, p := range plugins {
			if ps, ok := p.(string); ok && strings.Contains(ps, pluginName) {
				version := ""
				if strings.Contains(ps, "@") {
					parts := strings.Split(ps, "@")
					if len(parts) > 1 {
						version = parts[len(parts)-1]
					}
				}
				return true, version
			}
		}
	}

	return false, ""
}

// InstallOhMyOpenCode 安装 oh-my-opencode
func (a *App) InstallOhMyOpenCode() error {
	runtime.EventsEmit(a.ctx, "output-log", "正在安装 oh-my-opencode...")

	// 使用 npx 运行安装程序
	var cmd *exec.Cmd
	if goruntime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "npx", "oh-my-opencode", "--non-interactive")
	} else {
		cmd = exec.Command("npx", "oh-my-opencode", "--non-interactive")
	}
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		// 如果 npx 失败，尝试直接添加到配置
		runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("npx 安装失败，尝试直接配置: %s", string(output)))
		if addErr := a.addPlugin("oh-my-opencode"); addErr != nil {
			return fmt.Errorf("安装失败: %v", addErr)
		}
	}

	runtime.EventsEmit(a.ctx, "output-log", "oh-my-opencode 安装成功")
	return nil
}

// UninstallOhMyOpenCode 卸载 oh-my-opencode
func (a *App) UninstallOhMyOpenCode() error {
	if err := a.removePlugin("oh-my-opencode"); err != nil {
		return err
	}

	// 删除配置文件
	homeDir, _ := os.UserHomeDir()
	ohMyConfigPath := filepath.Join(homeDir, ".config", "opencode", "oh-my-opencode.json")
	os.Remove(ohMyConfigPath)

	runtime.EventsEmit(a.ctx, "output-log", "oh-my-opencode 已卸载")
	return nil
}

// FixOhMyOpenCode 修复 oh-my-opencode 配置（禁用 Google 认证）
func (a *App) FixOhMyOpenCode() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, ".config", "opencode", "oh-my-opencode.json")
	
	// 创建配置：禁用 Google 认证
	config := map[string]interface{}{
		"google_auth": false,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return err
	}

	runtime.EventsEmit(a.ctx, "output-log", "oh-my-opencode 已修复：禁用 Google 认证")
	return nil
}

// InstallAntigravityAuth 安装 opencode-antigravity-auth
func (a *App) InstallAntigravityAuth() error {
	runtime.EventsEmit(a.ctx, "output-log", "正在安装 opencode-antigravity-auth...")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, ".config", "opencode", "opencode.json")
	
	// 确保目录存在
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	var config map[string]interface{}
	
	// 读取现有配置
	data, err := os.ReadFile(configPath)
	if err != nil {
		// 文件不存在，创建新配置
		config = map[string]interface{}{
			"$schema": "https://opencode.ai/config.json",
		}
	} else {
		if err := json.Unmarshal(data, &config); err != nil {
			return err
		}
	}

	// 添加插件
	pluginName := "opencode-antigravity-auth@beta"
	if plugins, ok := config["plugin"].([]interface{}); ok {
		found := false
		for _, p := range plugins {
			if ps, ok := p.(string); ok && strings.Contains(ps, "opencode-antigravity-auth") {
				found = true
				break
			}
		}
		if !found {
			config["plugin"] = append(plugins, pluginName)
		}
	} else {
		config["plugin"] = []interface{}{pluginName}
	}

	// 添加 provider 配置（这是认证所必需的）
	googleModels := map[string]interface{}{
		"antigravity-gemini-3-pro": map[string]interface{}{
			"name":  "Gemini 3 Pro (Antigravity)",
			"limit": map[string]interface{}{"context": 1048576, "output": 65535},
			"modalities": map[string]interface{}{
				"input":  []string{"text", "image", "pdf"},
				"output": []string{"text"},
			},
			"variants": map[string]interface{}{
				"low":  map[string]interface{}{"thinkingLevel": "low"},
				"high": map[string]interface{}{"thinkingLevel": "high"},
			},
		},
		"antigravity-gemini-3-flash": map[string]interface{}{
			"name":  "Gemini 3 Flash (Antigravity)",
			"limit": map[string]interface{}{"context": 1048576, "output": 65536},
			"modalities": map[string]interface{}{
				"input":  []string{"text", "image", "pdf"},
				"output": []string{"text"},
			},
			"variants": map[string]interface{}{
				"minimal": map[string]interface{}{"thinkingLevel": "minimal"},
				"low":     map[string]interface{}{"thinkingLevel": "low"},
				"medium":  map[string]interface{}{"thinkingLevel": "medium"},
				"high":    map[string]interface{}{"thinkingLevel": "high"},
			},
		},
		"antigravity-claude-sonnet-4-5": map[string]interface{}{
			"name":  "Claude Sonnet 4.5 (Antigravity)",
			"limit": map[string]interface{}{"context": 200000, "output": 64000},
			"modalities": map[string]interface{}{
				"input":  []string{"text", "image", "pdf"},
				"output": []string{"text"},
			},
		},
		"antigravity-claude-sonnet-4-5-thinking": map[string]interface{}{
			"name":  "Claude Sonnet 4.5 Thinking (Antigravity)",
			"limit": map[string]interface{}{"context": 200000, "output": 64000},
			"modalities": map[string]interface{}{
				"input":  []string{"text", "image", "pdf"},
				"output": []string{"text"},
			},
			"variants": map[string]interface{}{
				"low": map[string]interface{}{"thinkingConfig": map[string]interface{}{"thinkingBudget": 8192}},
				"max": map[string]interface{}{"thinkingConfig": map[string]interface{}{"thinkingBudget": 32768}},
			},
		},
		"antigravity-claude-opus-4-5-thinking": map[string]interface{}{
			"name":  "Claude Opus 4.5 Thinking (Antigravity)",
			"limit": map[string]interface{}{"context": 200000, "output": 64000},
			"modalities": map[string]interface{}{
				"input":  []string{"text", "image", "pdf"},
				"output": []string{"text"},
			},
			"variants": map[string]interface{}{
				"low": map[string]interface{}{"thinkingConfig": map[string]interface{}{"thinkingBudget": 8192}},
				"max": map[string]interface{}{"thinkingConfig": map[string]interface{}{"thinkingBudget": 32768}},
			},
		},
		"gemini-2.5-flash": map[string]interface{}{
			"name":  "Gemini 2.5 Flash (Gemini CLI)",
			"limit": map[string]interface{}{"context": 1048576, "output": 65536},
			"modalities": map[string]interface{}{
				"input":  []string{"text", "image", "pdf"},
				"output": []string{"text"},
			},
		},
		"gemini-2.5-pro": map[string]interface{}{
			"name":  "Gemini 2.5 Pro (Gemini CLI)",
			"limit": map[string]interface{}{"context": 1048576, "output": 65536},
			"modalities": map[string]interface{}{
				"input":  []string{"text", "image", "pdf"},
				"output": []string{"text"},
			},
		},
	}

	// 设置 provider 配置
	provider, ok := config["provider"].(map[string]interface{})
	if !ok {
		provider = make(map[string]interface{})
	}
	
	// 设置 google provider
	provider["google"] = map[string]interface{}{
		"models": googleModels,
	}
	config["provider"] = provider

	// 保存配置
	newData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, newData, 0644); err != nil {
		return err
	}

	runtime.EventsEmit(a.ctx, "output-log", "opencode-antigravity-auth 安装成功，配置已写入")
	runtime.EventsEmit(a.ctx, "output-log", "请点击'认证'按钮进行 Google 账号认证")
	return nil
}

// UninstallAntigravityAuth 卸载 opencode-antigravity-auth
func (a *App) UninstallAntigravityAuth() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, ".config", "opencode", "opencode.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	// 从 plugin 中移除
	if plugins, ok := config["plugin"].([]interface{}); ok {
		newPlugins := []interface{}{}
		for _, p := range plugins {
			if ps, ok := p.(string); ok && !strings.Contains(ps, "opencode-antigravity-auth") {
				newPlugins = append(newPlugins, p)
			}
		}
		config["plugin"] = newPlugins
	}

	// 移除 google provider 配置
	if provider, ok := config["provider"].(map[string]interface{}); ok {
		delete(provider, "google")
		if len(provider) == 0 {
			delete(config, "provider")
		} else {
			config["provider"] = provider
		}
	}

	// 保存配置
	newData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, newData, 0644); err != nil {
		return err
	}

	// 删除认证文件
	antigravityAccountsPath := filepath.Join(homeDir, ".config", "opencode", "antigravity-accounts.json")
	os.Remove(antigravityAccountsPath)

	runtime.EventsEmit(a.ctx, "output-log", "opencode-antigravity-auth 已卸载")
	return nil
}

// RestartOpenCode 重启 OpenCode 服务
func (a *App) RestartOpenCode() error {
	runtime.EventsEmit(a.ctx, "output-log", "正在重启 OpenCode...")
	
	// 发送连接断开事件
	runtime.EventsEmit(a.ctx, "opencode-status", "restarting")
	
	// 先停止当前目录的 OpenCode 实例
	a.openCode.Stop()
	
	// 等待进程完全退出
	time.Sleep(2 * time.Second)
	
	// 重新启动
	if err := a.openCode.Start(); err != nil {
		runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("重启失败: %v", err))
		runtime.EventsEmit(a.ctx, "opencode-status", "error")
		return err
	}
	
	runtime.EventsEmit(a.ctx, "output-log", "OpenCode 正在启动，请等待连接...")
	return nil
}

// addPlugin 添加插件到配置
func (a *App) addPlugin(pluginName string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, ".config", "opencode", "opencode.json")
	
	// 确保目录存在
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	var config map[string]interface{}
	
	// 读取现有配置
	data, err := os.ReadFile(configPath)
	if err != nil {
		// 文件不存在，创建新配置
		config = map[string]interface{}{
			"$schema": "https://opencode.ai/config.json",
			"plugin": []interface{}{pluginName},
		}
	} else {
		if err := json.Unmarshal(data, &config); err != nil {
			return err
		}
		
		// 添加到 plugin 数组
		if plugins, ok := config["plugin"].([]interface{}); ok {
			// 检查是否已存在
			for _, p := range plugins {
				if ps, ok := p.(string); ok && strings.Contains(ps, strings.Split(pluginName, "@")[0]) {
					return nil // 已安装
				}
			}
			config["plugin"] = append(plugins, pluginName)
		} else {
			config["plugin"] = []interface{}{pluginName}
		}
	}

	// 保存配置
	newData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, newData, 0644)
}

// removePlugin 从配置中移除插件
func (a *App) removePlugin(pluginName string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, ".config", "opencode", "opencode.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	// 从 plugin 中移除
	if plugins, ok := config["plugin"].([]interface{}); ok {
		newPlugins := []interface{}{}
		for _, p := range plugins {
			if ps, ok := p.(string); ok && !strings.Contains(ps, pluginName) {
				newPlugins = append(newPlugins, p)
			}
		}
		config["plugin"] = newPlugins
	}
	
	// 也检查 plugins 字段（兼容旧配置）
	if plugins, ok := config["plugins"].([]interface{}); ok {
		newPlugins := []interface{}{}
		for _, p := range plugins {
			if ps, ok := p.(string); ok && !strings.Contains(ps, pluginName) {
				newPlugins = append(newPlugins, p)
			}
		}
		config["plugins"] = newPlugins
	}

	// 保存配置
	newData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, newData, 0644)
}


// --- 文件管理 ---

// SetWorkDir 设置工作目录
func (a *App) SetWorkDir(dir string) error {
	return a.fileMgr.SetRootDir(dir)
}

// GetWorkDir 获取工作目录
func (a *App) GetWorkDir() string {
	return a.fileMgr.GetRootDir()
}

// ListDir 列出目录内容
func (a *App) ListDir(dir string) ([]*FileInfo, error) {
	return a.fileMgr.ListDir(dir)
}

// ReadFile 读取文件内容
func (a *App) ReadFileContent(path string) (string, error) {
	return a.fileMgr.ReadFile(path)
}

// WriteFile 写入文件内容
func (a *App) WriteFileContent(path, content string) error {
	return a.fileMgr.WriteFile(path, content)
}

// WatchFile 监听文件变化
func (a *App) WatchFile(path string) error {
	return a.fileMgr.WatchFile(path)
}

// UnwatchFile 取消监听文件
func (a *App) UnwatchFile(path string) {
	a.fileMgr.UnwatchFile(path)
}

// RunFile 运行文件
func (a *App) RunFile(filePath string) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("文件路径为空")
	}

	// 获取文件扩展名和目录
	ext := strings.ToLower(filepath.Ext(filePath))
	dir := filepath.Dir(filePath)
	fileName := filepath.Base(filePath)
	fileNameNoExt := strings.TrimSuffix(fileName, ext)

	var cmd string
	var args []string

	switch ext {
	case ".py":
		cmd = "python3"
		args = []string{filePath}
	case ".go":
		cmd = "go"
		args = []string{"run", filePath}
	case ".js":
		cmd = "node"
		args = []string{filePath}
	case ".ts":
		cmd = "npx"
		args = []string{"ts-node", filePath}
	case ".java":
		// 检测是否是 Maven 项目
		if a.isMavenProject(dir) {
			cmd = "mvn"
			args = []string{"-f", a.findPomXml(dir), "compile", "exec:java", "-Dexec.mainClass=" + a.getJavaMainClass(filePath)}
		} else if a.isGradleProject(dir) {
			cmd = "gradle"
			args = []string{"-p", a.findGradleProject(dir), "run"}
		} else {
			// 单文件编译运行
			cmd = "sh"
			args = []string{"-c", fmt.Sprintf("cd %s && javac %s && java %s", dir, fileName, fileNameNoExt)}
		}
	case ".rs":
		cmd = "cargo"
		args = []string{"run", "--manifest-path", a.findCargoToml(dir)}
	case ".rb":
		cmd = "ruby"
		args = []string{filePath}
	case ".php":
		cmd = "php"
		args = []string{filePath}
	case ".sh":
		cmd = "bash"
		args = []string{filePath}
	case ".html", ".htm":
		// 返回特殊标记，前端处理打开浏览器
		return "OPEN_BROWSER:" + filePath, nil
	default:
		return "", fmt.Errorf("不支持运行 %s 文件", ext)
	}

	// 返回命令让前端在终端中执行
	fullCmd := cmd + " " + strings.Join(args, " ")
	return fullCmd, nil
}

// 检测是否是 Maven 项目
func (a *App) isMavenProject(dir string) bool {
	return a.findPomXml(dir) != ""
}

// 查找 pom.xml
func (a *App) findPomXml(dir string) string {
	// 向上查找 pom.xml
	for {
		pomPath := filepath.Join(dir, "pom.xml")
		if _, err := os.Stat(pomPath); err == nil {
			return pomPath
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// 检测是否是 Gradle 项目
func (a *App) isGradleProject(dir string) bool {
	return a.findGradleProject(dir) != ""
}

// 查找 Gradle 项目目录
func (a *App) findGradleProject(dir string) string {
	for {
		if _, err := os.Stat(filepath.Join(dir, "build.gradle")); err == nil {
			return dir
		}
		if _, err := os.Stat(filepath.Join(dir, "build.gradle.kts")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// 查找 Cargo.toml
func (a *App) findCargoToml(dir string) string {
	for {
		cargoPath := filepath.Join(dir, "Cargo.toml")
		if _, err := os.Stat(cargoPath); err == nil {
			return cargoPath
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Join(dir, "Cargo.toml")
}

// 获取 Java 主类名
func (a *App) getJavaMainClass(filePath string) string {
	// 简单实现：从文件路径推断类名
	// 实际应该解析 package 声明
	fileName := filepath.Base(filePath)
	return strings.TrimSuffix(fileName, ".java")
}

// --- MCP 配置管理 ---

// MCPServer MCP 服务器配置
type MCPServer struct {
	Type        string            `json:"type"`                  // "local" 或 "remote"
	Command     []string          `json:"command,omitempty"`     // 本地服务器命令
	URL         string            `json:"url,omitempty"`         // 远程服务器 URL
	Enabled     bool              `json:"enabled"`
	Environment map[string]string `json:"environment,omitempty"`
	Timeout     int               `json:"timeout,omitempty"`
}

// MCPConfig MCP 配置
type MCPConfig struct {
	MCP map[string]MCPServer `json:"mcp"`
}

// MCPMarketItem MCP 市场项目
type MCPMarketItem struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Command     []string `json:"command"`
	EnvVars     []string `json:"envVars,omitempty"` // 需要的环境变量
	Category    string   `json:"category"`
	DocsURL     string   `json:"docsUrl,omitempty"`  // 官方文档链接
	ConfigTips  string   `json:"configTips,omitempty"` // 配置说明
}

// GetMCPConfigPath 获取 MCP 配置文件路径
func (a *App) GetMCPConfigPath() string {
	workDir := a.openCode.GetWorkDir()
	if workDir == "" {
		homeDir, _ := os.UserHomeDir()
		return filepath.Join(homeDir, ".opencode", "config.json")
	}
	return filepath.Join(workDir, ".opencode", "config.json")
}

// GetMCPConfig 获取 MCP 配置
func (a *App) GetMCPConfig() (*MCPConfig, error) {
	configPath := a.GetMCPConfigPath()
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 返回空配置
			return &MCPConfig{MCP: make(map[string]MCPServer)}, nil
		}
		return nil, fmt.Errorf("读取配置失败: %v", err)
	}
	
	var config MCPConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置失败: %v", err)
	}
	
	if config.MCP == nil {
		config.MCP = make(map[string]MCPServer)
	}
	
	return &config, nil
}

// SaveMCPConfig 保存 MCP 配置
func (a *App) SaveMCPConfig(config MCPConfig) error {
	configPath := a.GetMCPConfigPath()
	
	// 确保目录存在
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}
	
	// 读取现有配置（保留其他字段）
	var existingConfig map[string]interface{}
	if data, err := os.ReadFile(configPath); err == nil {
		json.Unmarshal(data, &existingConfig)
	}
	if existingConfig == nil {
		existingConfig = make(map[string]interface{})
	}
	
	// 更新 mcp 字段
	existingConfig["mcp"] = config.MCP
	
	// 写入文件
	data, err := json.MarshalIndent(existingConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}
	
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置失败: %v", err)
	}
	
	runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("MCP 配置已保存: %s", configPath))
	return nil
}

// AddMCPServer 添加 MCP 服务器（通过 OpenCode API）
func (a *App) AddMCPServer(name string, server MCPServer) (map[string]MCPServerStatus, error) {
	// 1. 保存到配置文件
	config, err := a.GetMCPConfig()
	if err != nil {
		return nil, err
	}
	config.MCP[name] = server
	if err := a.SaveMCPConfig(*config); err != nil {
		return nil, err
	}

	// 2. 通过 OpenCode API 动态添加
	apiConfig := map[string]interface{}{
		"type":    server.Type,
		"enabled": server.Enabled,
	}
	if server.Type == "local" {
		apiConfig["command"] = server.Command
	} else {
		apiConfig["url"] = server.URL
	}
	if len(server.Environment) > 0 {
		apiConfig["environment"] = server.Environment
	}

	payload := map[string]interface{}{
		"name":   name,
		"config": apiConfig,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", a.serverURL+"/mcp", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("添加 MCP 服务器失败: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("添加失败: %s", string(respBody))
	}

	// 解析返回的状态
	var status map[string]MCPServerStatus
	json.Unmarshal(respBody, &status)

	runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("MCP 服务器 %s 已添加", name))
	return status, nil
}

// RemoveMCPServer 删除 MCP 服务器
func (a *App) RemoveMCPServer(name string) error {
	config, err := a.GetMCPConfig()
	if err != nil {
		return err
	}
	
	delete(config.MCP, name)
	return a.SaveMCPConfig(*config)
}

// ToggleMCPServer 启用/禁用 MCP 服务器
func (a *App) ToggleMCPServer(name string, enabled bool) error {
	config, err := a.GetMCPConfig()
	if err != nil {
		return err
	}
	
	if server, ok := config.MCP[name]; ok {
		server.Enabled = enabled
		config.MCP[name] = server
		if err := a.SaveMCPConfig(*config); err != nil {
			return err
		}
		
		// 通过 OpenCode API 连接/断开
		if enabled {
			// 先同步配置到 OpenCode，然后连接
			a.syncMCPToOpenCode(name, server)
		} else {
			// 断开连接
			a.DisconnectMCPServer(name)
		}
		return nil
	}
	
	return fmt.Errorf("MCP 服务器不存在: %s", name)
}

// GetMCPMarket 获取 MCP 市场列表（内置热门服务器）
func (a *App) GetMCPMarket() []MCPMarketItem {
	return []MCPMarketItem{
		{
			Name:        "filesystem",
			Description: "文件系统操作 - 读写文件、目录管理",
			Command:     []string{"npx", "-y", "@modelcontextprotocol/server-filesystem", "/path/to/dir"},
			Category:    "filesystem",
			DocsURL:     "https://github.com/modelcontextprotocol/servers/tree/main/src/filesystem",
			ConfigTips:  "命令最后的 /path/to/dir 需要改成实际的目录路径，如 /Users/xxx/projects",
		},
		{
			Name:        "github",
			Description: "GitHub 集成 - 仓库、Issue、PR 管理",
			Command:     []string{"npx", "-y", "@modelcontextprotocol/server-github"},
			EnvVars:     []string{"GITHUB_TOKEN"},
			Category:    "development",
			DocsURL:     "https://github.com/modelcontextprotocol/servers/tree/main/src/github",
			ConfigTips:  "GITHUB_TOKEN: 在 GitHub Settings > Developer settings > Personal access tokens 创建，需要 repo 权限",
		},
		{
			Name:        "postgres",
			Description: "PostgreSQL 数据库操作",
			Command:     []string{"npx", "-y", "@modelcontextprotocol/server-postgres"},
			EnvVars:     []string{"POSTGRES_CONNECTION_STRING"},
			Category:    "database",
			DocsURL:     "https://github.com/modelcontextprotocol/servers/tree/main/src/postgres",
			ConfigTips:  "POSTGRES_CONNECTION_STRING: 格式为 postgresql://user:password@host:5432/database",
		},
		{
			Name:        "sqlite",
			Description: "SQLite 数据库操作",
			Command:     []string{"npx", "-y", "@modelcontextprotocol/server-sqlite", "--db-path", "/path/to/db.sqlite"},
			Category:    "database",
			DocsURL:     "https://github.com/modelcontextprotocol/servers/tree/main/src/sqlite",
			ConfigTips:  "命令中的 /path/to/db.sqlite 需要改成实际的数据库文件路径",
		},
		{
			Name:        "mysql",
			Description: "MySQL 数据库操作",
			Command:     []string{"uvx", "mysql_mcp_server"},
			EnvVars:     []string{"MYSQL_HOST", "MYSQL_PORT", "MYSQL_USER", "MYSQL_PASSWORD", "MYSQL_DATABASE"},
			Category:    "database",
			DocsURL:     "https://github.com/designcomputer/mysql_mcp_server",
			ConfigTips:  "需要先安装 uv: pip install uv 或 brew install uv\nMYSQL_HOST: 数据库地址(如 localhost 或 192.168.0.15)\nMYSQL_PORT: 端口(默认 3306)\nMYSQL_USER: 用户名(如 root)\nMYSQL_PASSWORD: 密码\nMYSQL_DATABASE: 数据库名",
		},
		{
			Name:        "redis",
			Description: "Redis 缓存数据库操作",
			Command:     []string{"npx", "-y", "@modelcontextprotocol/server-redis"},
			EnvVars:     []string{"REDIS_URL"},
			Category:    "database",
			DocsURL:     "https://github.com/modelcontextprotocol/servers/tree/main/src/redis",
			ConfigTips:  "REDIS_URL: 格式为 redis://[:password@]host:6379[/db]，如 redis://localhost:6379",
		},
		{
			Name:        "puppeteer",
			Description: "浏览器自动化 - 网页截图、爬虫",
			Command:     []string{"npx", "-y", "@modelcontextprotocol/server-puppeteer"},
			Category:    "automation",
			DocsURL:     "https://github.com/modelcontextprotocol/servers/tree/main/src/puppeteer",
			ConfigTips:  "无需额外配置，会自动下载 Chromium 浏览器",
		},
		{
			Name:        "brave-search",
			Description: "Brave 搜索引擎",
			Command:     []string{"npx", "-y", "@modelcontextprotocol/server-brave-search"},
			EnvVars:     []string{"BRAVE_API_KEY"},
			Category:    "search",
			DocsURL:     "https://github.com/modelcontextprotocol/servers/tree/main/src/brave-search",
			ConfigTips:  "BRAVE_API_KEY: 在 https://brave.com/search/api/ 注册获取 API Key",
		},
		{
			Name:        "fetch",
			Description: "HTTP 请求 - 获取网页内容",
			Command:     []string{"uvx", "mcp-server-fetch"},
			Category:    "network",
			DocsURL:     "https://github.com/modelcontextprotocol/servers/tree/main/src/fetch",
			ConfigTips:  "无需额外配置。注意：需要先安装 uv (pip install uv)",
		},
		{
			Name:        "memory",
			Description: "知识图谱记忆系统",
			Command:     []string{"npx", "-y", "@modelcontextprotocol/server-memory"},
			Category:    "memory",
			DocsURL:     "https://github.com/modelcontextprotocol/servers/tree/main/src/memory",
			ConfigTips:  "无需额外配置，数据存储在本地",
		},
		{
			Name:        "sequential-thinking",
			Description: "顺序思考 - 复杂问题分解",
			Command:     []string{"npx", "-y", "@modelcontextprotocol/server-sequential-thinking"},
			Category:    "reasoning",
			DocsURL:     "https://github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking",
			ConfigTips:  "无需额外配置",
		},
		{
			Name:        "sentry",
			Description: "Sentry 错误追踪集成",
			Command:     []string{"npx", "-y", "@modelcontextprotocol/server-sentry"},
			EnvVars:     []string{"SENTRY_AUTH_TOKEN"},
			Category:    "monitoring",
			DocsURL:     "https://github.com/modelcontextprotocol/servers/tree/main/src/sentry",
			ConfigTips:  "SENTRY_AUTH_TOKEN: 在 Sentry 设置 > Auth Tokens 创建，需要 project:read 权限",
		},
		{
			Name:        "slack",
			Description: "Slack 消息集成",
			Command:     []string{"npx", "-y", "@modelcontextprotocol/server-slack"},
			EnvVars:     []string{"SLACK_BOT_TOKEN"},
			Category:    "communication",
			DocsURL:     "https://github.com/modelcontextprotocol/servers/tree/main/src/slack",
			ConfigTips:  "SLACK_BOT_TOKEN: 在 Slack API 创建 App，获取 Bot User OAuth Token (xoxb-...)",
		},
		{
			Name:        "google-maps",
			Description: "Google 地图服务",
			Command:     []string{"npx", "-y", "@modelcontextprotocol/server-google-maps"},
			EnvVars:     []string{"GOOGLE_MAPS_API_KEY"},
			Category:    "maps",
			DocsURL:     "https://github.com/modelcontextprotocol/servers/tree/main/src/google-maps",
			ConfigTips:  "GOOGLE_MAPS_API_KEY: 在 Google Cloud Console 创建项目并启用 Maps API",
		},
		{
			Name:        "context7",
			Description: "Context7 文档搜索",
			Command:     []string{"npx", "-y", "@context7/mcp-server"},
			EnvVars:     []string{"CONTEXT7_API_KEY"},
			Category:    "search",
			DocsURL:     "https://github.com/upstash/context7",
			ConfigTips:  "CONTEXT7_API_KEY: 可选，在 context7.com 注册获取，不填也可使用（有速率限制）",
		},
		{
			Name:        "playwright",
			Description: "Playwright 浏览器自动化测试",
			Command:     []string{"npx", "-y", "@playwright/mcp-server"},
			Category:    "testing",
			DocsURL:     "https://github.com/microsoft/playwright-mcp",
			ConfigTips:  "无需额外配置，首次运行会自动安装浏览器",
		},
	}
}

// OpenMCPConfigFile 打开 MCP 配置文件
func (a *App) OpenMCPConfigFile() (string, error) {
	configPath := a.GetMCPConfigPath()
	
	// 确保文件存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// 创建默认配置
		dir := filepath.Dir(configPath)
		os.MkdirAll(dir, 0755)
		defaultConfig := map[string]interface{}{
			"$schema": "https://opencode.ai/config.json",
			"mcp":     map[string]interface{}{},
		}
		data, _ := json.MarshalIndent(defaultConfig, "", "  ")
		os.WriteFile(configPath, data, 0644)
	}
	
	return configPath, nil
}

// MCPServerStatus MCP 服务器状态（从 OpenCode API 获取）
type MCPServerStatus struct {
	Status string `json:"status"` // connected, disabled, failed, needs_auth
	Error  string `json:"error,omitempty"`
}

// MCPTool MCP 工具信息
type MCPTool struct {
	ID          string      `json:"id"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

// GetMCPStatus 从 OpenCode API 获取 MCP 服务器状态
func (a *App) GetMCPStatus() (map[string]MCPServerStatus, error) {
	// 从配置文件读取并同步到 OpenCode
	config, err := a.GetMCPConfig()
	if err != nil {
		return nil, err
	}
	
	allStatuses := make(map[string]MCPServerStatus)
	
	// 同步所有启用的服务器到 OpenCode
	for name, server := range config.MCP {
		if server.Enabled {
			status, err := a.syncMCPToOpenCodeWithStatus(name, server)
			if err == nil && status != nil {
				// 合并状态
				for k, v := range status {
					allStatuses[k] = v
				}
			}
		}
	}
	
	return allStatuses, nil
}

// syncMCPToOpenCodeWithStatus 同步单个 MCP 服务器到 OpenCode 并返回状态
func (a *App) syncMCPToOpenCodeWithStatus(name string, server MCPServer) (map[string]MCPServerStatus, error) {
	apiConfig := map[string]interface{}{
		"type":    server.Type,
		"enabled": server.Enabled,
	}
	if server.Type == "local" {
		apiConfig["command"] = server.Command
	} else {
		apiConfig["url"] = server.URL
	}
	if len(server.Environment) > 0 {
		apiConfig["environment"] = server.Environment
	}

	payload := map[string]interface{}{
		"name":   name,
		"config": apiConfig,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", a.serverURL+"/mcp", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	respBody, _ := io.ReadAll(resp.Body)
	
	var status map[string]MCPServerStatus
	if err := json.Unmarshal(respBody, &status); err != nil {
		// 如果解析失败，可能是因为 API 返回了错误信息或其他格式
		// 尝试作为单个状态解析
		var singleStatus MCPServerStatus
		if err2 := json.Unmarshal(respBody, &singleStatus); err2 == nil && singleStatus.Status != "" {
			return map[string]MCPServerStatus{name: singleStatus}, nil
		}
		return nil, err
	}
	
	return status, nil
}

// syncMCPToOpenCode 同步单个 MCP 服务器到 OpenCode（不返回状态）
func (a *App) syncMCPToOpenCode(name string, server MCPServer) error {
	_, err := a.syncMCPToOpenCodeWithStatus(name, server)
	return err
}

// ConnectMCPServer 连接 MCP 服务器
func (a *App) ConnectMCPServer(name string) error {
	url := fmt.Sprintf("%s/mcp/%s/connect", a.serverURL, name)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("连接 MCP 服务器失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("连接失败: %s", string(body))
	}

	runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("MCP 服务器 %s 已连接", name))
	return nil
}

// DisconnectMCPServer 断开 MCP 服务器
func (a *App) DisconnectMCPServer(name string) error {
	url := fmt.Sprintf("%s/mcp/%s/disconnect", a.serverURL, name)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("断开 MCP 服务器失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("断开失败: %s", string(body))
	}

	runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("MCP 服务器 %s 已断开", name))
	return nil
}

// GetMCPTools 获取 MCP 服务器的工具列表
func (a *App) GetMCPTools() ([]MCPTool, error) {
	// 1. 尝试从 OpenCode API 获取动态工具列表
	url := fmt.Sprintf("%s/mcp/tools", a.serverURL)
	resp, err := a.httpClient.Get(url)
	if err == nil && resp.StatusCode == 200 {
		defer resp.Body.Close()
		var tools []MCPTool
		if err := json.NewDecoder(resp.Body).Decode(&tools); err == nil && len(tools) > 0 {
			return tools, nil
		}
	}

	// 2. 如果 API 获取失败，回退到从配置文件读取并使用硬编码列表（保持兼容性）
	config, err := a.GetMCPConfig()
	if err != nil {
		return nil, err
	}
	
	var allTools []MCPTool
	
	// 遍历每个启用的 MCP 服务器，获取其工具
	for name, server := range config.MCP {
		if !server.Enabled {
			continue
		}
		
		// 构建工具 ID 前缀
		prefix := fmt.Sprintf("mcp_%s_", name)
		
		// 根据服务器名称添加已知的工具
		// 这里我们硬编码一些常见 MCP 服务器的工具
		switch name {
		case "mysql":
			allTools = append(allTools, MCPTool{
				ID:          prefix + "execute_sql",
				Description: "Execute SQL query on MySQL database. Use this to query tables, insert/update/delete data.",
			})
		case "filesystem":
			allTools = append(allTools, MCPTool{
				ID:          prefix + "read_file",
				Description: "Read file contents from filesystem",
			})
			allTools = append(allTools, MCPTool{
				ID:          prefix + "write_file",
				Description: "Write content to a file",
			})
			allTools = append(allTools, MCPTool{
				ID:          prefix + "list_directory",
				Description: "List directory contents",
			})
		case "redis":
			allTools = append(allTools, MCPTool{
				ID:          prefix + "redis_command",
				Description: "Execute Redis command",
			})
		case "postgres":
			allTools = append(allTools, MCPTool{
				ID:          prefix + "query",
				Description: "Execute SQL query on PostgreSQL database",
			})
		}
	}
	
	return allTools, nil
}

// GetMCPToolsPrompt 获取 MCP 工具的提示文本，用于注入到消息中
func (a *App) GetMCPToolsPrompt() (string, error) {
	// 获取工具列表（优先使用动态获取的）
	tools, err := a.GetMCPTools()
	if err != nil {
		return "", err
	}
	
	if len(tools) == 0 {
		return "", nil
	}
	
	// 构建 MCP 工具提示
	var sb strings.Builder
	sb.WriteString("[Available MCP Tools]\n")
	
	for _, tool := range tools {
		// 格式化参数描述
		params := ""
		if tool.Parameters != nil {
			if paramMap, ok := tool.Parameters.(map[string]interface{}); ok {
				// 简化的参数描述
				if props, ok := paramMap["properties"].(map[string]interface{}); ok {
					var paramList []string
					for key := range props {
						paramList = append(paramList, key)
					}
					if len(paramList) > 0 {
						params = fmt.Sprintf("(%s)", strings.Join(paramList, ", "))
					}
				}
			}
		}
		
		sb.WriteString(fmt.Sprintf("- %s: %s %s\n", tool.ID, tool.Description, params))
	}
	
	sb.WriteString("Use these tools when the user asks about related operations.\n")
	
	return sb.String(), nil
}

// OpenFolder 打开文件夹选择对话框并设置为工作目录
func (a *App) OpenFolder() (string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择工作目录",
	})
	if err != nil {
		return "", err
	}
	if dir != "" {
		// 设置文件管理器根目录
		a.fileMgr.SetRootDir(dir)
		// 设置 OpenCode 工作目录（会自动更新 serverURL）
		a.openCode.SetWorkDir(dir)
		// 更新 app 的 serverURL
		a.serverURL = fmt.Sprintf("http://localhost:%d", a.openCode.GetCurrentPort())
		runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("服务器地址已更新: %s", a.serverURL))
		// 启动该目录的 OpenCode 实例（如果已运行则复用）
		go a.openCode.StartForDir(dir)
	}
	return dir, nil
}

// --- 文件操作（右键菜单） ---

// DeletePath 删除文件或文件夹
func (a *App) DeletePath(path string) error {
	return a.fileMgr.DeletePath(path)
}

// RenamePath 重命名文件或文件夹
func (a *App) RenamePath(oldPath, newName string) (string, error) {
	return a.fileMgr.RenamePath(oldPath, newName)
}

// CopyPath 复制文件或文件夹
func (a *App) CopyPath(src, destDir string) (string, error) {
	return a.fileMgr.CopyPath(src, destDir)
}

// MovePath 移动文件或文件夹
func (a *App) MovePath(src, destDir string) (string, error) {
	return a.fileMgr.MovePath(src, destDir)
}

// CreateNewFile 创建新文件
func (a *App) CreateNewFile(dir, name string) (string, error) {
	return a.fileMgr.CreateFile(dir, name)
}

// CreateNewFolder 创建新文件夹
func (a *App) CreateNewFolder(dir, name string) (string, error) {
	return a.fileMgr.CreateFolder(dir, name)
}

// OpenInFinder 在访达/资源管理器中打开
func (a *App) OpenInFinder(path string) error {
	return openInFileManager(path)
}

// CopyToClipboard 复制文本到剪贴板
func (a *App) CopyToClipboard(text string) error {
	runtime.ClipboardSetText(a.ctx, text)
	return nil
}

// ConfigModel 配置文件中的模型信息
type ConfigModel struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Provider   string `json:"provider"`
	ContextLen int    `json:"contextLen,omitempty"`
	OutputLen  int    `json:"outputLen,omitempty"`
}

// GetConfigModels 从 opencode.json 配置文件读取模型列表
func (a *App) GetConfigModels() ([]ConfigModel, error) {
	var models []ConfigModel

	// 1. 先读取用户级配置 ~/.config/opencode/opencode.json
	homeDir, err := os.UserHomeDir()
	if err == nil {
		userConfigPath := filepath.Join(homeDir, ".config", "opencode", "opencode.json")
		if userModels, err := a.readModelsFromConfig(userConfigPath); err == nil {
			models = append(models, userModels...)
		}
	}

	// 2. 再读取项目级配置 {workDir}/opencode.json
	workDir := a.openCode.GetWorkDir()
	if workDir != "" {
		projectConfigPath := filepath.Join(workDir, "opencode.json")
		if projectModels, err := a.readModelsFromConfig(projectConfigPath); err == nil {
			// 项目配置优先，去重
			for _, pm := range projectModels {
				found := false
				for i, m := range models {
					if m.ID == pm.ID {
						models[i] = pm // 覆盖
						found = true
						break
					}
				}
				if !found {
					models = append(models, pm)
				}
			}
		}
	}

	runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("从配置文件读取到 %d 个模型", len(models)))
	return models, nil
}

// readModelsFromConfig 从单个配置文件读取模型
func (a *App) readModelsFromConfig(configPath string) ([]ConfigModel, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	var models []ConfigModel

	// 解析 provider 配置
	provider, ok := config["provider"].(map[string]interface{})
	if !ok {
		return models, nil
	}

	for providerID, providerConfig := range provider {
		pc, ok := providerConfig.(map[string]interface{})
		if !ok {
			continue
		}

		modelsConfig, ok := pc["models"].(map[string]interface{})
		if !ok {
			continue
		}

		for modelID, modelConfig := range modelsConfig {
			mc, ok := modelConfig.(map[string]interface{})
			if !ok {
				continue
			}

			model := ConfigModel{
				ID:       fmt.Sprintf("%s/%s", providerID, modelID),
				Provider: providerID,
			}

			if name, ok := mc["name"].(string); ok {
				model.Name = name
			} else {
				model.Name = modelID
			}

			// 解析 limit
			if limit, ok := mc["limit"].(map[string]interface{}); ok {
				if ctx, ok := limit["context"].(float64); ok {
					model.ContextLen = int(ctx)
				}
				if out, ok := limit["output"].(float64); ok {
					model.OutputLen = int(out)
				}
			}

			models = append(models, model)
		}
	}

	return models, nil
}
