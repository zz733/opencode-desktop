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
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

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

// ImageData 图片数据
type ImageData struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Data string `json:"data"` // base64 编码的图片数据
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
