package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx        context.Context
	serverURL  string
	httpClient *http.Client
	termMgr    *TerminalManager
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
	go func() {
		for {
			resp, err := a.httpClient.Get(a.serverURL + "/event")
			if err != nil {
				runtime.EventsEmit(a.ctx, "connection-error", err.Error())
				time.Sleep(3 * time.Second)
				continue
			}

			reader := bufio.NewReader(resp.Body)
			for {
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
			time.Sleep(1 * time.Second)
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
