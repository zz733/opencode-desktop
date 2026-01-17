package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

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
	EnvVars     []string `json:"envVars,omitempty"`    // 需要的环境变量
	Category    string   `json:"category"`
	DocsURL     string   `json:"docsUrl,omitempty"`    // 官方文档链接
	ConfigTips  string   `json:"configTips,omitempty"` // 配置说明
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