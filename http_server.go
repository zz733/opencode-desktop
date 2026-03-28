package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// HTTPServer 远程控制 HTTP 服务器
type HTTPServer struct {
	app            *App
	server         *http.Server
	token          string
	port           int
	active         bool
	mu             sync.RWMutex
	sseConns       map[string]chan []byte
	currentSession string // 当前会话 ID
}

// NewHTTPServer 创建 HTTP 服务器
func NewHTTPServer(app *App) *HTTPServer {
	return &HTTPServer{
		app:      app,
		token:    generateConnectionCode(), // 使用 6 位连接码
		sseConns: make(map[string]chan []byte),
	}
}

// Start 启动服务器
func (s *HTTPServer) Start(port int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.active {
		return fmt.Errorf("server already running")
	}

	s.port = port

	mux := http.NewServeMux()

	// API 路由
	mux.HandleFunc("/api/status", s.corsMiddleware(s.authMiddleware(s.handleStatus)))
	mux.HandleFunc("/api/models", s.corsMiddleware(s.authMiddleware(s.handleModels)))
	mux.HandleFunc("/api/sessions", s.corsMiddleware(s.authMiddleware(s.handleSessions)))
	mux.HandleFunc("/api/messages", s.corsMiddleware(s.authMiddleware(s.handleMessages)))
	mux.HandleFunc("/api/history", s.corsMiddleware(s.authMiddleware(s.handleHistory)))
	mux.HandleFunc("/api/files", s.corsMiddleware(s.authMiddleware(s.handleFiles)))
	mux.HandleFunc("/api/terminal", s.corsMiddleware(s.authMiddleware(s.handleTerminal)))
	mux.HandleFunc("/api/events", s.corsMiddleware(s.authMiddleware(s.handleEvents)))

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("HTTP server error: %v\n", err)
		}
	}()

	s.active = true
	fmt.Printf("Remote control server started on port %d\n", port)
	fmt.Printf("Access token: %s\n", s.token)

	// 订阅 OpenCode 事件，转发给手机端
	go s.forwardOpenCodeEvents()

	return nil
}

// Stop 停止服务器
func (s *HTTPServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		return err
	}

	// 关闭所有 SSE 连接
	for _, ch := range s.sseConns {
		close(ch)
	}
	s.sseConns = make(map[string]chan []byte)

	s.active = false
	fmt.Println("Remote control server stopped")

	return nil
}

// GetToken 获取访问令牌
func (s *HTTPServer) GetToken() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.token
}

// GetPort 获取端口
func (s *HTTPServer) GetPort() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.port
}

// IsActive 是否激活
func (s *HTTPServer) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.active
}

// SetCurrentSession 设置当前会话 ID
func (s *HTTPServer) SetCurrentSession(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentSession = sessionID
}

// GetCurrentSession 获取当前会话 ID
func (s *HTTPServer) GetCurrentSession() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentSession
}

// corsMiddleware CORS 中间件
func (s *HTTPServer) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// authMiddleware 认证中间件
func (s *HTTPServer) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 从 Header 或 URL 参数获取 token
		var token string

		// 优先从 Authorization Header 获取
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimPrefix(auth, "Bearer ")
		}

		// 如果 Header 中没有，尝试从 URL 参数获取（用于 SSE）
		if token == "" {
			token = r.URL.Query().Get("token")
		}

		expectedAuth := "Bearer " + s.token
		actualAuth := "Bearer " + token

		if actualAuth != expectedAuth {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

// handleStatus 处理状态请求
func (s *HTTPServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := map[string]interface{}{
		"active":    s.active,
		"port":      s.port,
		"timestamp": time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleModels 处理模型列表请求
func (s *HTTPServer) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 动态模型列表（从 OpenCode API 获取，完全依赖官方接口）
	var dynamicModels []map[string]interface{}

	resp, err := s.app.apiClient.Get(s.app.serverURL + "/provider")
	if err == nil {
		defer resp.Body.Close()

		var providerResp struct {
			All []struct {
				ID     string                 `json:"id"`
				Name   string                 `json:"name"`
				Models map[string]interface{} `json:"models"`
			} `json:"all"`
			Connected []string `json:"connected"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&providerResp); err == nil {
			// 建立 connected 集合以便快速查找，只有配置了 key 的才算 connected
			connectedMap := make(map[string]bool)
			for _, c := range providerResp.Connected {
				connectedMap[c] = true
			}

			// 遍历每个 provider，只添加已连接的提供商的模型
			for _, provider := range providerResp.All {
				if provider.Models == nil || !connectedMap[provider.ID] {
					continue
				}

				// 只保留未被弃用的模型
				for modelID, modelData := range provider.Models {
					modelMap, ok := modelData.(map[string]interface{})
					if !ok {
						continue
					}

					status, _ := modelMap["status"].(string)
					if status == "deprecated" {
						continue
					}

					modelName := modelID
					if name, ok := modelMap["name"].(string); ok && name != "" {
						modelName = name
					}
					// 清理 (latest) 后缀
					modelName = strings.TrimSpace(strings.ReplaceAll(modelName, "(latest)", ""))

					// 判断是否免费
					isFree := false
					if costObj, ok := modelMap["cost"].(map[string]interface{}); ok {
						inputCost, _ := costObj["input"].(float64)
						outputCost, _ := costObj["output"].(float64)
						if inputCost == 0 && outputCost == 0 {
							isFree = true
						}
					}

					dynamicModels = append(dynamicModels, map[string]interface{}{
						"id":       fmt.Sprintf("%s/%s", provider.ID, modelID),
						"name":     modelName,
						"provider": provider.ID,
						"category": provider.Name, // 直接使用官方的 Provider 名称作为分类
						"free":     isFree,
						"builtin":  false,
					})
				}
			}
		}
	}

	fmt.Printf("✅ 返回 %d 个模型给手机端\n", len(dynamicModels))

	// 返回模型列表
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"models":  dynamicModels,
		"count":   len(dynamicModels),
	})
}

// handleSessions 处理会话请求
func (s *HTTPServer) handleSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取 OpenCode 会话列表
	sessions := s.app.openCode.GetStatus()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

// handleHistory 处理聊天历史请求
func (s *HTTPServer) handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 如果没有当前会话，尝试获取有内容的会话
	if s.currentSession == "" {
		fmt.Println("📜 获取历史: 当前无会话，尝试获取有内容的会话...")
		sessions, err := s.app.GetSessions()
		if err != nil {
			fmt.Printf("📜 获取会话列表失败: %v\n", err)
		} else if len(sessions) > 0 {
			// 优先选择有标题的会话（不是 "New session" 开头的）
			for i := len(sessions) - 1; i >= 0; i-- {
				if !strings.HasPrefix(sessions[i].Title, "New session") {
					s.currentSession = sessions[i].ID
					fmt.Printf("📜 使用有内容的会话: %s (%s)\n", s.currentSession, sessions[i].Title)
					break
				}
			}
			// 如果没找到有标题的，使用最新的
			if s.currentSession == "" {
				s.currentSession = sessions[len(sessions)-1].ID
				fmt.Printf("📜 使用最新会话: %s\n", s.currentSession)
			}
		}
	}

	// 如果还是没有会话，返回空列表
	if s.currentSession == "" {
		fmt.Println("📜 无可用会话，返回空历史")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"messages":  []interface{}{},
			"sessionID": "",
		})
		return
	}

	fmt.Printf("📜 获取会话 %s 的历史消息...\n", s.currentSession)

	// 获取会话消息
	messages, err := s.app.GetSessionMessages(s.currentSession)
	if err != nil {
		fmt.Printf("📜 获取历史失败: %v\n", err)
		// 返回空列表而不是错误
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"messages":  []interface{}{},
			"sessionID": s.currentSession,
			"error":     err.Error(),
		})
		return
	}

	fmt.Printf("📜 获取到 %d 条消息\n", len(messages))

	// 转换为前端格式，确保不返回 null
	result := make([]map[string]interface{}, 0)
	if messages != nil {
		for i, msg := range messages {
			result = append(result, map[string]interface{}{
				"id":        fmt.Sprintf("%s-%d", s.currentSession, i),
				"role":      msg.Role,
				"content":   msg.Content,
				"timestamp": time.Now().Unix(),
			})
		}
	}

	fmt.Printf("📜 返回 %d 条消息给前端\n", len(result))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"messages":  result,
		"sessionID": s.currentSession,
	})
}

// handleMessages 处理消息请求
func (s *HTTPServer) handleMessages(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// 获取消息列表
		messages := []map[string]interface{}{}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messages)

	case http.MethodPost:
		// 检查是否是 multipart/form-data
		contentType := r.Header.Get("Content-Type")
		var content string
		var modelID string
		var hasImage bool
		var imageName string

		if strings.HasPrefix(contentType, "multipart/form-data") {
			// 解析 multipart 表单
			if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
				http.Error(w, "Failed to parse form", http.StatusBadRequest)
				return
			}

			// 获取文本内容
			content = r.FormValue("content")
			modelID = r.FormValue("model")

			// 获取图片文件
			file, header, err := r.FormFile("image")
			if err == nil {
				defer file.Close()
				hasImage = true
				imageName = header.Filename

				fmt.Printf("📷 收到图片: %s (size: %d bytes)\n", imageName, header.Size)
			}
		} else {
			// JSON 格式
			var req struct {
				Content string `json:"content"`
				Model   string `json:"model"`
			}

			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}
			content = req.Content
			modelID = req.Model
		}

		// 打印收到的消息
		fmt.Printf("📩 收到消息: %s\n", content)
		if modelID != "" {
			fmt.Printf("📋 使用模型: %s\n", modelID)
		}
		if hasImage {
			fmt.Printf("📷 附带图片: %s\n", imageName)
		}

		// 检查 OpenCode 是否连接
		if !s.app.openCode.CheckConnection() {
			fmt.Printf("❌ OpenCode 未连接! serverURL: %s\n", s.app.serverURL)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "OpenCode 未连接，请先在桌面端启动 OpenCode",
			})
			return
		}
		fmt.Printf("✓ OpenCode 已连接: %s\n", s.app.serverURL)

		// 确保有会话 - 使用桌面端当前的会话
		if s.currentSession == "" {
			fmt.Printf("🔍 获取桌面端当前会话...\n")
			// 尝试获取桌面端的当前会话
			sessions, err := s.app.GetSessions()
			if err != nil {
				fmt.Printf("⚠️  获取会话列表失败: %v\n", err)
			} else {
				fmt.Printf("📋 找到 %d 个会话\n", len(sessions))
				if len(sessions) > 0 {
					// 使用最新的会话
					s.currentSession = sessions[len(sessions)-1].ID
					fmt.Printf("✓ 使用现有会话: %s\n", s.currentSession)
				}
			}

			// 如果还是没有会话，创建新的
			if s.currentSession == "" {
				fmt.Printf("🆕 创建新会话...\n")
				session, err := s.app.CreateSession()
				if err != nil {
					fmt.Printf("❌ 创建会话失败: %v\n", err)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"success": false,
						"error":   fmt.Sprintf("创建会话失败: %v", err),
					})
					return
				}
				if session != nil {
					s.currentSession = session.ID
					fmt.Printf("✓ 新会话已创建: %s\n", s.currentSession)
				}
			}
		}

		if s.currentSession == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "无法创建或获取会话",
			})
			return
		}

		// 发送消息到 OpenCode
		var sendErr error
		if modelID != "" {
			// 使用指定模型发送
			fmt.Printf("📤 发送消息到会话 %s (模型: %s)\n", s.currentSession, modelID)
			sendErr = s.app.SendMessageWithModel(s.currentSession, content, modelID, nil)
		} else {
			// 使用默认模型
			fmt.Printf("📤 发送消息到会话 %s\n", s.currentSession)
			sendErr = s.app.SendMessage(s.currentSession, content)
		}

		if sendErr != nil {
			fmt.Printf("❌ 发送消息失败: %v\n", sendErr)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   fmt.Sprintf("发送消息失败: %v", sendErr),
			})
			return
		}

		fmt.Printf("✅ 消息已发送到 OpenCode，会话: %s\n", s.currentSession)
		fmt.Println("   AI 响应将通过 SSE 推送到手机端")

		// 返回成功，AI 响应会通过 SSE 推送
		response := map[string]interface{}{
			"success":   true,
			"message":   "消息已发送",
			"sessionID": s.currentSession,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleFiles 处理文件请求
func (s *HTTPServer) handleFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取查询参数
	path := r.URL.Query().Get("path")
	action := r.URL.Query().Get("action")

	if action == "content" {
		// 读取文件内容
		content, err := s.app.ReadFileContent(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"path":    path,
			"content": content,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	} else {
		// 列出目录
		if path == "" {
			path = s.app.GetWorkDir()
		}

		files, err := s.app.ListDir(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(files)
	}
}

// handleTerminal 处理终端请求
func (s *HTTPServer) handleTerminal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: 获取终端输出
	output := map[string]interface{}{
		"output": "Terminal output...",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(output)
}

// handleEvents 处理 SSE 事件流
func (s *HTTPServer) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 支持从查询参数获取 token（用于 EventSource）
	token := r.URL.Query().Get("token")
	if token == "" {
		// 如果查询参数没有，尝试从 Header 获取
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimPrefix(auth, "Bearer ")
		}
	}

	// 验证 token
	if token != s.token {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 设置 SSE 头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 创建事件通道
	connID := generateToken()
	eventChan := make(chan []byte, 1000)

	s.mu.Lock()
	s.sseConns[connID] = eventChan
	s.mu.Unlock()

	// 清理连接
	defer func() {
		fmt.Printf("🔌 SSE 客户端已断开: %s\n", connID)
		s.mu.Lock()
		delete(s.sseConns, connID)
		close(eventChan)
		s.mu.Unlock()
	}()

	// 发送初始连接事件
	fmt.Printf("🔌 SSE 客户端已连接: %s\n", connID)
	fmt.Fprintf(w, "data: {\"type\":\"connected\",\"id\":\"%s\"}\n\n", connID)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// 保持连接并发送事件
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			// 发送心跳
			fmt.Fprintf(w, "data: {\"type\":\"ping\"}\n\n")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case event, ok := <-eventChan:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", event)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}

// BroadcastEvent 广播事件到所有 SSE 连接
func (s *HTTPServer) BroadcastEvent(eventType string, data interface{}) {
	s.mu.RLock()
	connCount := len(s.sseConns)
	s.mu.RUnlock()

	if connCount == 0 {
		return // 没有连接，不需要广播
	}

	event := map[string]interface{}{
		"type": eventType,
		"data": data,
		"time": time.Now().Unix(),
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		fmt.Printf("❌ 序列化事件失败: %v\n", err)
		return
	}

	// 频繁打印会导致性能问题，注释掉
	// fmt.Printf("📡 广播事件到 %d 个连接: %s\n", connCount, eventType)

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, ch := range s.sseConns {
		select {
		case ch <- eventJSON:
			// 成功发送
		default:
			// 通道已满，跳过
			// fmt.Printf("⚠️  连接 %s 的通道已满，跳过事件\n", connID)
		}
	}
}

// generateConnectionCode 生成 6 位连接码
func generateConnectionCode() string {
	// 生成 6 位数字码（100000-999999）
	b := make([]byte, 4)
	rand.Read(b)
	code := int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])
	if code < 0 {
		code = -code
	}
	code = (code % 900000) + 100000
	return fmt.Sprintf("%06d", code)
}

// generateToken 生成随机令牌（用于内部）
func generateToken() string {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		// 降级到时间戳
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// forwardOpenCodeEvents 转发 OpenCode 事件到手机端
func (s *HTTPServer) forwardOpenCodeEvents() {
	fmt.Println("开始监听 OpenCode 事件...")

	// 确保 OpenCode 已连接
	for i := 0; i < 10; i++ {
		if s.app.openCode.CheckConnection() {
			break
		}
		fmt.Printf("OpenCode 未连接，等待连接... (%d/10)\n", i+1)
		time.Sleep(2 * time.Second)
	}

	if !s.app.openCode.CheckConnection() {
		fmt.Println("⚠️  OpenCode 连接超时，但继续运行")
	}

	// 订阅 OpenCode 事件
	if err := s.app.SubscribeEvents(); err != nil {
		fmt.Printf("订阅 OpenCode 事件失败: %v\n", err)
		return
	}

	fmt.Println("✓ 已订阅 OpenCode 事件")
	fmt.Println("事件转发已在 app.go startup 中配置")
}
