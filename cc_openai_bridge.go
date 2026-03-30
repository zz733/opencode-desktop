package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type CCBridgeManager struct {
	app     *App
	mu      sync.Mutex
	servers map[string]*CCOpenAIBridge
}

type CCOpenAIBridge struct {
	app           *App
	platform      string
	port          int
	apiKey        string
	selectedModel string
	server        *http.Server
}

type bridgeSubscription struct {
	ready  chan error
	events chan bridgeEvent
	errs   chan error
}

type bridgeEvent struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
}

type bridgeChatCompletionRequest struct {
	Model    string              `json:"model"`
	Messages []bridgeChatMessage `json:"messages"`
	Stream   bool                `json:"stream"`
	User     string              `json:"user"`
}

type bridgeChatMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
	Name    string      `json:"name"`
}

type bridgeChatCompletionResponse struct {
	ID      string                        `json:"id"`
	Object  string                        `json:"object"`
	Created int64                         `json:"created"`
	Model   string                        `json:"model"`
	Choices []bridgeChatCompletionChoice  `json:"choices"`
	Usage   bridgeChatCompletionUsageInfo `json:"usage"`
}

type bridgeChatCompletionChoice struct {
	Index        int               `json:"index"`
	Message      bridgeChatMessage `json:"message"`
	FinishReason string            `json:"finish_reason"`
}

type bridgeChatCompletionUsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type bridgeModelListResponse struct {
	Object string              `json:"object"`
	Data   []bridgeModelEntity `json:"data"`
}

type bridgeModelEntity struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	OwnedBy string `json:"owned_by"`
}

type bridgeStreamChunk struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Choices []bridgeStreamChoice `json:"choices"`
}

type bridgeStreamChoice struct {
	Index        int               `json:"index"`
	Delta        bridgeStreamDelta `json:"delta"`
	FinishReason *string           `json:"finish_reason"`
}

type bridgeStreamDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

func NewCCBridgeManager(app *App) *CCBridgeManager {
	return &CCBridgeManager{
		app:     app,
		servers: make(map[string]*CCOpenAIBridge),
	}
}

func (m *CCBridgeManager) Start(platform string, pConfig CCConnectPlatformConfig, selectedModel string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.servers[platform]; ok {
		_ = existing.stop()
		delete(m.servers, platform)
	}

	bridge, err := newCCOpenAIBridge(m.app, platform, pConfig, selectedModel)
	if err != nil {
		return err
	}

	if err := bridge.start(); err != nil {
		return err
	}

	m.servers[platform] = bridge
	return nil
}

func (m *CCBridgeManager) Stop(platform string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	bridge, ok := m.servers[platform]
	if !ok {
		return nil
	}

	delete(m.servers, platform)
	return bridge.stop()
}

func newCCOpenAIBridge(app *App, platform string, pConfig CCConnectPlatformConfig, selectedModel string) (*CCOpenAIBridge, error) {
	port := 8081
	if rawPort := strings.TrimSpace(pConfig.Config["port"]); rawPort != "" {
		parsedPort, err := strconv.Atoi(rawPort)
		if err != nil || parsedPort <= 0 || parsedPort > 65535 {
			return nil, fmt.Errorf("%s 桥接端口无效: %s", platform, rawPort)
		}
		port = parsedPort
	}

	modelID := strings.TrimSpace(selectedModel)
	if modelID == "" {
		defaultModel, err := app.GetDefaultModel()
		if err == nil {
			modelID = strings.TrimSpace(defaultModel)
		}
	}
	if modelID == "" {
		modelID = "opencode/gpt-5.1-codex"
	}

	return &CCOpenAIBridge{
		app:           app,
		platform:      platform,
		port:          port,
		apiKey:        strings.TrimSpace(pConfig.Config["api_key"]),
		selectedModel: modelID,
	}, nil
}

func (b *CCOpenAIBridge) start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", b.handleHealth)
	mux.HandleFunc("/v1/models", b.handleModels)
	mux.HandleFunc("/v1/chat/completions", b.handleChatCompletions)

	b.server = &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", b.port),
		Handler: mux,
	}

	listener, err := net.Listen("tcp", b.server.Addr)
	if err != nil {
		return fmt.Errorf("%s Go 桥接启动失败: %v", b.platform, err)
	}

	go func() {
		if err := b.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			b.emitLog(fmt.Sprintf("⚠️ %s Go 桥接异常退出: %v", b.platform, err))
		}
	}()

	b.emitLog(fmt.Sprintf("🚀 已启动 %s Go 桥接: http://127.0.0.1:%d/v1", b.platform, b.port))
	return nil
}

func (b *CCOpenAIBridge) stop() error {
	if b.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := b.server.Shutdown(ctx)
	b.emitLog(fmt.Sprintf("🛑 已停止 %s Go 桥接", b.platform))
	return err
}

func (b *CCOpenAIBridge) emitLog(message string) {
	if b.app != nil && b.app.ctx != nil {
		runtime.EventsEmit(b.app.ctx, "output-log", message)
	}
	fmt.Println(message)
}

func (b *CCOpenAIBridge) requireAuth(w http.ResponseWriter, r *http.Request) bool {
	if b.apiKey == "" {
		return true
	}

	token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer"))
	if token == b.apiKey {
		return true
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"message": "Unauthorized",
			"type":    "invalid_request_error",
		},
	})
	return false
}

func (b *CCOpenAIBridge) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":       true,
		"platform": b.platform,
		"model":    b.selectedModel,
		"port":     b.port,
	})
}

func (b *CCOpenAIBridge) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !b.requireAuth(w, r) {
		return
	}

	modelIDs := []string{"qclaw-project", "wechat-bridge"}
	if strings.TrimSpace(b.selectedModel) != "" {
		modelIDs = append(modelIDs, b.selectedModel)
	}

	seen := make(map[string]struct{})
	models := make([]bridgeModelEntity, 0, len(modelIDs))
	for _, modelID := range modelIDs {
		modelID = strings.TrimSpace(modelID)
		if modelID == "" {
			continue
		}
		if _, ok := seen[modelID]; ok {
			continue
		}
		seen[modelID] = struct{}{}
		models = append(models, bridgeModelEntity{
			ID:      modelID,
			Object:  "model",
			OwnedBy: "opencode-desktop",
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(bridgeModelListResponse{
		Object: "list",
		Data:   models,
	})
}

func (b *CCOpenAIBridge) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !b.requireAuth(w, r) {
		return
	}
	if !b.app.openCode.CheckConnection() {
		b.writeBridgeError(w, http.StatusServiceUnavailable, "OpenCode 未连接，请先启动桌面端 OpenCode")
		return
	}

	var req bridgeChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		b.writeBridgeError(w, http.StatusBadRequest, fmt.Sprintf("请求体解析失败: %v", err))
		return
	}

	b.emitLog(fmt.Sprintf("💬 收到 ChatCompletion 请求: model=%s", req.Model))

	prompt := b.buildPrompt(req.Messages)
	if prompt == "" {
		b.emitLog("⚠️ messages 不能为空")
		b.writeBridgeError(w, http.StatusBadRequest, "messages 不能为空")
		return
	}

	modelID := b.resolveModel(req.Model)
	workDir := b.app.openCode.GetWorkDir()
	b.emitLog(fmt.Sprintf("🔄 准备创建会话 (模型: %s, 工作目录: %s)", modelID, workDir))
	session, err := b.app.CreateSession()
	if err != nil {
		b.emitLog(fmt.Sprintf("❌ 创建会话失败: %v", err))
		b.writeBridgeError(w, http.StatusInternalServerError, fmt.Sprintf("创建 OpenCode 会话失败: %v", err))
		return
	}
	b.emitLog(fmt.Sprintf("✅ 会话创建成功: %s", session.ID))

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()

	b.emitLog("🔄 开始订阅事件...")
	sub := b.subscribeSessionEvents(ctx, session.ID)
	if err := <-sub.ready; err != nil {
		b.emitLog(fmt.Sprintf("❌ 订阅事件失败: %v", err))
		b.writeBridgeError(w, http.StatusInternalServerError, fmt.Sprintf("订阅 OpenCode 事件失败: %v", err))
		return
	}
	b.emitLog("✅ 事件订阅成功")

	b.emitLog("🔄 准备发送 Prompt...")
	if err := b.sendPrompt(ctx, session.ID, prompt, modelID); err != nil {
		b.emitLog(fmt.Sprintf("❌ 发送 Prompt 失败: %v", err))
		b.writeBridgeError(w, http.StatusInternalServerError, fmt.Sprintf("发送消息到 OpenCode 失败: %v", err))
		return
	}
	b.emitLog("✅ Prompt 发送成功，等待响应...")

	requestID := b.generateID("chatcmpl")
	if req.Stream {
		if err := b.writeStreamingResponse(w, requestID, req.Model, session.ID, sub); err != nil {
			b.emitLog(fmt.Sprintf("⚠️ %s 流式响应失败: %v", b.platform, err))
		}
		return
	}

	answer, err := b.collectResponse(ctx, session.ID, sub)
	if err != nil {
		b.writeBridgeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	responseModel := strings.TrimSpace(req.Model)
	if responseModel == "" {
		responseModel = modelID
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(bridgeChatCompletionResponse{
		ID:      requestID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   responseModel,
		Choices: []bridgeChatCompletionChoice{
			{
				Index: 0,
				Message: bridgeChatMessage{
					Role:    "assistant",
					Content: answer,
				},
				FinishReason: "stop",
			},
		},
		Usage: bridgeChatCompletionUsageInfo{},
	})
}

func (b *CCOpenAIBridge) writeBridgeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
			"type":    "invalid_request_error",
		},
	})
}

func (b *CCOpenAIBridge) resolveModel(requested string) string {
	// Always use the selected model from desktop app, ignoring the requested model from QClaw/WeChat
	// This ensures cc-connect always uses the model selected in the desktop UI
	if strings.TrimSpace(b.selectedModel) != "" {
		return b.selectedModel
	}
	return requested
}

func (b *CCOpenAIBridge) buildPrompt(messages []bridgeChatMessage) string {
	parts := make([]string, 0, len(messages))
	for _, msg := range messages {
		content := strings.TrimSpace(b.extractMessageText(msg.Content))
		if content == "" {
			continue
		}

		role := strings.ToLower(strings.TrimSpace(msg.Role))
		switch role {
		case "system":
			parts = append(parts, "系统指令:\n"+content)
		case "assistant":
			parts = append(parts, "助手:\n"+content)
		case "tool":
			parts = append(parts, "工具结果:\n"+content)
		default:
			parts = append(parts, "用户:\n"+content)
		}
	}

	if len(parts) == 0 {
		return "继续" // Return a default non-empty string to avoid "messages 不能为空" or token length errors if all parts are empty after trimming
	}
	if len(parts) == 1 && strings.HasPrefix(parts[0], "用户:\n") {
		// Truncate overly long prompts to avoid OpenCode "Range of input length" errors
		prompt := strings.TrimPrefix(parts[0], "用户:\n")
		if len(prompt) > 200000 {
			prompt = prompt[len(prompt)-200000:] // Keep the last 200k chars (typically includes latest conversation)
		}
		return prompt
	}

	combined := strings.Join(parts, "\n\n") + "\n\n请基于以上对话继续回复最后一条用户消息，保持上下文一致。"
	if len(combined) > 200000 {
		combined = combined[len(combined)-200000:]
	}
	return combined
}

func (b *CCOpenAIBridge) extractMessageText(content interface{}) string {
	switch value := content.(type) {
	case string:
		return value
	case []interface{}:
		parts := make([]string, 0, len(value))
		for _, item := range value {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			itemType, _ := itemMap["type"].(string)
			if itemType != "" && itemType != "text" && itemType != "input_text" {
				continue
			}
			switch textValue := itemMap["text"].(type) {
			case string:
				parts = append(parts, textValue)
			case map[string]interface{}:
				if nested, ok := textValue["value"].(string); ok && nested != "" {
					parts = append(parts, nested)
				}
			}
		}
		return strings.Join(parts, "\n")
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return ""
		}
		return string(data)
	}
}

func (b *CCOpenAIBridge) sendPrompt(ctx context.Context, sessionID, prompt, modelID string) error {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		modelID = b.selectedModel
	}

	payload := map[string]interface{}{
		"parts": []map[string]interface{}{
			{
				"type": "text",
				"text": prompt,
			},
		},
	}

	if modelID != "" {
		modelParts := strings.SplitN(modelID, "/", 2)
		if len(modelParts) == 2 {
			payload["model"] = map[string]string{
				"providerID": modelParts[0],
				"modelID":    modelParts[1],
			}
		} else {
			payload["model"] = map[string]string{
				"modelID": modelID,
			}
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/session/%s/prompt_async", b.app.serverURL, sessionID), bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.app.apiClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("OpenCode 返回 %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (b *CCOpenAIBridge) subscribeSessionEvents(ctx context.Context, sessionID string) *bridgeSubscription {
	sub := &bridgeSubscription{
		ready:  make(chan error, 1),
		events: make(chan bridgeEvent, 32),
		errs:   make(chan error, 1),
	}

	go func() {
		defer close(sub.events)
		defer close(sub.errs)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.app.serverURL+"/event", nil)
		if err != nil {
			sub.ready <- err
			return
		}

		resp, err := b.app.sseClient.Do(req)
		if err != nil {
			sub.ready <- err
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			sub.ready <- fmt.Errorf("订阅状态码 %d: %s", resp.StatusCode, string(body))
			return
		}

		sub.ready <- nil

		reader := bufio.NewReader(resp.Body)
		var dataBuffer bytes.Buffer

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			line, err := reader.ReadString('\n')
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				sub.errs <- err
				return
			}

			if strings.TrimSpace(line) == "" {
				if dataBuffer.Len() > 0 {
					var event bridgeEvent
					if err := json.Unmarshal(dataBuffer.Bytes(), &event); err == nil {
						if b.matchSessionEvent(sessionID, event) {
							select {
							case sub.events <- event:
							case <-ctx.Done():
								return
							}
						}
					}
					dataBuffer.Reset()
				}
				continue
			}

			if strings.HasPrefix(line, "data:") {
				data := strings.TrimPrefix(line, "data:")
				data = strings.TrimPrefix(data, " ")
				data = strings.TrimSuffix(data, "\n")
				data = strings.TrimSuffix(data, "\r")
				if dataBuffer.Len() > 0 {
					dataBuffer.WriteByte('\n')
				}
				dataBuffer.WriteString(data)
			}
		}
	}()

	return sub
}

func (b *CCOpenAIBridge) matchSessionEvent(sessionID string, event bridgeEvent) bool {
	if sessionID == "" {
		return false
	}

	properties := event.Properties
	if properties == nil {
		return false
	}

	if value, ok := properties["sessionID"].(string); ok && value == sessionID {
		return true
	}
	if info, ok := properties["info"].(map[string]interface{}); ok {
		if value, ok := info["sessionID"].(string); ok && value == sessionID {
			return true
		}
	}
	if part, ok := properties["part"].(map[string]interface{}); ok {
		if value, ok := part["sessionID"].(string); ok && value == sessionID {
			return true
		}
	}
	if message, ok := properties["message"].(map[string]interface{}); ok {
		if value, ok := message["sessionID"].(string); ok && value == sessionID {
			return true
		}
	}
	if errorInfo, ok := properties["error"].(map[string]interface{}); ok {
		if value, ok := errorInfo["sessionID"].(string); ok && value == sessionID {
			return true
		}
	}
	return false
}

func (b *CCOpenAIBridge) collectResponse(ctx context.Context, sessionID string, sub *bridgeSubscription) (string, error) {
	var totalEmittedText strings.Builder
	lastTextPerPart := make(map[string]string)
	assistantMessageID := ""

	for {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("等待 OpenCode 响应超时")
		case err := <-sub.errs:
			if err != nil {
				return "", err
			}
		case event, ok := <-sub.events:
			if !ok {
				answer := totalEmittedText.String()
				if strings.TrimSpace(answer) != "" {
					return answer, nil
				}
				return b.fetchLatestAssistantMessage(sessionID)
			}
			switch event.Type {
			case "message.created", "message.updated":
				if msg, ok := event.Properties["message"].(map[string]interface{}); ok {
					if role, _ := msg["role"].(string); role == "assistant" {
						if id, _ := msg["id"].(string); id != "" {
							assistantMessageID = id
							b.emitLog(fmt.Sprintf("[SSE Stream] Found assistant message ID: %s", id))
						}
					}
				}
			case "message.part.updated":
				current, partID, msgID := extractEventTextAndID(event)
				eventJson, _ := json.Marshal(event)
				b.emitLog(fmt.Sprintf("[SSE] %s (partID: %s, msgID: %s, text: %q)", string(eventJson), partID, msgID, current))

				if assistantMessageID == "" || msgID != assistantMessageID {
					continue
				}

				if current == "" && partID == "" {
					continue
				}
				lastPartText := lastTextPerPart[partID]
				delta := diffText(lastPartText, current)
				if delta != "" {
					totalEmittedText.WriteString(delta)
				}
				lastTextPerPart[partID] = current
			case "session.error":
				return "", fmt.Errorf("%s", extractEventError(event))
			case "session.idle":
				return b.finalizeAssistantText(sessionID, totalEmittedText.String())
			case "session.status":
				if extractEventStatus(event) == "idle" {
					return b.finalizeAssistantText(sessionID, totalEmittedText.String())
				}
			}
		}
	}
}

func (b *CCOpenAIBridge) finalizeAssistantText(sessionID, current string) (string, error) {
	finalText, err := b.fetchLatestAssistantMessage(sessionID)
	if err == nil && strings.TrimSpace(finalText) != "" {
		return finalText, nil
	}
	if strings.TrimSpace(current) != "" {
		return current, nil
	}
	if err != nil {
		return "", err
	}
	return "", fmt.Errorf("未收到 OpenCode 响应内容")
}

func (b *CCOpenAIBridge) fetchLatestAssistantMessage(sessionID string) (string, error) {
	messages, err := b.app.GetSessionMessages(sessionID)
	if err != nil {
		return "", err
	}
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "assistant" && strings.TrimSpace(messages[i].Content) != "" {
			return messages[i].Content, nil
		}
	}
	return "", fmt.Errorf("未找到 assistant 消息")
}

func (b *CCOpenAIBridge) writeStreamingResponse(w http.ResponseWriter, requestID, requestedModel, sessionID string, sub *bridgeSubscription) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("当前响应不支持流式输出")
	}

	responseModel := strings.TrimSpace(requestedModel)
	if responseModel == "" {
		responseModel = b.selectedModel
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	initial := bridgeStreamChunk{
		ID:      requestID,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   responseModel,
		Choices: []bridgeStreamChoice{
			{
				Index: 0,
				Delta: bridgeStreamDelta{Role: "assistant"},
			},
		},
	}
	if err := writeSSEChunk(w, initial); err != nil {
		return err
	}
	flusher.Flush()

	lastTextPerPart := make(map[string]string)
	var totalEmittedText strings.Builder
	assistantMessageID := ""
	for {
		select {
		case err := <-sub.errs:
			if err != nil {
				return err
			}
		case event, ok := <-sub.events:
			if !ok {
				return writeSSEDone(w, flusher)
			}
			switch event.Type {
			case "message.created", "message.updated":
				if msg, ok := event.Properties["message"].(map[string]interface{}); ok {
					if role, _ := msg["role"].(string); role == "assistant" {
						if id, _ := msg["id"].(string); id != "" {
							assistantMessageID = id
							b.emitLog(fmt.Sprintf("[SSE Stream] Found assistant message ID: %s", id))
						}
					}
				}
			case "message.part.updated":
				current, partID, msgID := extractEventTextAndID(event)
				eventJson, _ := json.Marshal(event)
				b.emitLog(fmt.Sprintf("[SSE Stream] %s (partID: %s, msgID: %s, text: %q)", string(eventJson), partID, msgID, current))

				if assistantMessageID == "" || msgID != assistantMessageID {
					continue
				}

				if current == "" && partID == "" {
					continue
				}
				lastPartText := lastTextPerPart[partID]
				delta := diffText(lastPartText, current)
				if delta != "" {
					chunk := bridgeStreamChunk{
						ID:      requestID,
						Object:  "chat.completion.chunk",
						Created: time.Now().Unix(),
						Model:   responseModel,
						Choices: []bridgeStreamChoice{
							{
								Index: 0,
								Delta: bridgeStreamDelta{Content: delta},
							},
						},
					}
					if err := writeSSEChunk(w, chunk); err != nil {
						return err
					}
					flusher.Flush()
					totalEmittedText.WriteString(delta)
				}
				lastTextPerPart[partID] = current
			case "session.error":
				message := extractEventError(event)
				if strings.TrimSpace(message) != "" {
					chunk := bridgeStreamChunk{
						ID:      requestID,
						Object:  "chat.completion.chunk",
						Created: time.Now().Unix(),
						Model:   responseModel,
						Choices: []bridgeStreamChoice{
							{
								Index: 0,
								Delta: bridgeStreamDelta{Content: "\n" + message},
							},
						},
					}
					if err := writeSSEChunk(w, chunk); err != nil {
						return err
					}
					flusher.Flush()
				}
				return writeSSEDone(w, flusher)
			case "session.idle":
				return b.finishStream(w, flusher, requestID, responseModel, sessionID, totalEmittedText.String())
			case "session.status":
				if extractEventStatus(event) == "idle" {
					return b.finishStream(w, flusher, requestID, responseModel, sessionID, totalEmittedText.String())
				}
			}
		}
	}
}

func (b *CCOpenAIBridge) finishStream(w http.ResponseWriter, flusher http.Flusher, requestID, responseModel, sessionID, lastText string) error {
	finalText, err := b.fetchLatestAssistantMessage(sessionID)
	if err == nil {
		delta := diffText(lastText, finalText)
		if delta != "" {
			chunk := bridgeStreamChunk{
				ID:      requestID,
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Model:   responseModel,
				Choices: []bridgeStreamChoice{
					{
						Index: 0,
						Delta: bridgeStreamDelta{Content: delta},
					},
				},
			}
			if err := writeSSEChunk(w, chunk); err != nil {
				return err
			}
			flusher.Flush()
		}
	}

	finishReason := "stop"
	chunk := bridgeStreamChunk{
		ID:      requestID,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   responseModel,
		Choices: []bridgeStreamChoice{
			{
				Index:        0,
				Delta:        bridgeStreamDelta{},
				FinishReason: &finishReason,
			},
		},
	}
	if err := writeSSEChunk(w, chunk); err != nil {
		return err
	}
	flusher.Flush()

	return writeSSEDone(w, flusher)
}

func writeSSEChunk(w http.ResponseWriter, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", string(data))
	return err
}

func writeSSEDone(w http.ResponseWriter, flusher http.Flusher) error {
	if _, err := fmt.Fprint(w, "data: [DONE]\n\n"); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}

func extractEventText(event bridgeEvent) string {
	text, _, _ := extractEventTextAndID(event)
	return text
}

func extractEventTextAndID(event bridgeEvent) (string, string, string) {
	part, ok := event.Properties["part"].(map[string]interface{})
	if !ok {
		return "", "", ""
	}
	partType, _ := part["type"].(string)
	if partType != "text" {
		return "", "", ""
	}
	text, _ := part["text"].(string)
	id, _ := part["id"].(string)
	msgID, _ := part["messageID"].(string)
	return text, id, msgID
}

func extractEventStatus(event bridgeEvent) string {
	status, ok := event.Properties["status"].(map[string]interface{})
	if ok {
		if statusType, ok := status["type"].(string); ok {
			return statusType
		}
	}
	statusType, _ := event.Properties["status"].(string)
	return statusType
}

func extractEventError(event bridgeEvent) string {
	errorValue, ok := event.Properties["error"]
	if !ok {
		if message, ok := event.Properties["message"].(string); ok {
			return message
		}
		return "OpenCode 返回未知错误"
	}

	switch value := errorValue.(type) {
	case string:
		return value
	case map[string]interface{}:
		if data, ok := value["data"].(map[string]interface{}); ok {
			if message, ok := data["message"].(string); ok && message != "" {
				return message
			}
		}
		if message, ok := value["message"].(string); ok && message != "" {
			return message
		}
		if name, ok := value["name"].(string); ok && name != "" {
			return name
		}
	}

	return "OpenCode 返回未知错误"
}

func diffText(previous, current string) string {
	if current == "" {
		return ""
	}
	if strings.HasPrefix(current, previous) {
		return current[len(previous):]
	}
	return current
}

func (b *CCOpenAIBridge) generateID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}
