package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
					break
				}
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "data:") {
					data := strings.TrimPrefix(line, "data:")
					data = strings.TrimSpace(data)
					runtime.EventsEmit(a.ctx, "server-event", data)
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

// SendMessageWithModel 发送消息并指定模型
func (a *App) SendMessageWithModel(sessionID, content, model string) error {
	payload := map[string]interface{}{
		"parts": []map[string]interface{}{
			{"type": "text", "text": content},
		},
	}
	if model != "" {
		// model 格式: provider/modelID，需要拆分成对象
		parts := strings.SplitN(model, "/", 2)
		if len(parts) == 2 {
			payload["model"] = map[string]string{
				"providerID": parts[0],
				"modelID":    parts[1],
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

// SetOpenCodeWorkDir 设置 OpenCode 工作目录并重启
func (a *App) SetOpenCodeWorkDir(dir string) error {
	a.openCode.SetWorkDir(dir)
	// 同时更新文件管理器的根目录
	a.fileMgr.SetRootDir(dir)
	// 重启 OpenCode 以应用新目录
	return a.openCode.Restart()
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
