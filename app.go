package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	accountMgr    *AccountManager // Kiro Account Manager
	configMgr     *ConfigManager  // Configuration Manager
	httpServer    *HTTPServer     // Remote Control HTTP Server
}

// NewApp creates a new App application struct
func NewApp() *App {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DisableKeepAlives = true
	transport.MaxIdleConns = 0
	transport.MaxIdleConnsPerHost = 0

	app := &App{
		serverURL: "http://localhost:4096",
		httpClient: &http.Client{
			Timeout:   0, // no timeout for SSE
			Transport: transport,
		},
	}
	app.termMgr = NewTerminalManager(app)
	app.openCode = NewOpenCodeManager(app)
	app.fileMgr = NewFileManager(app)

	// Initialize Kiro Account Manager
	app.initAccountManager()

	return app
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Set context for account manager events
	if a.accountMgr != nil {
		a.accountMgr.SetContext(ctx)
	}

	// 监听 OpenCode 的 server-event 并转发到远程控制客户端
	runtime.EventsOn(ctx, "server-event", func(data ...interface{}) {
		if a.httpServer != nil && len(data) > 0 {
			fmt.Printf("📨 收到 OpenCode 事件，转发到手机端: %v\n", data[0])
			
			// 解析事件，更新当前会话
			if dataStr, ok := data[0].(string); ok {
				var event map[string]interface{}
				if err := json.Unmarshal([]byte(dataStr), &event); err == nil {
					eventType, _ := event["type"].(string)
					// 当会话创建或更新时，更新当前会话 ID
					if eventType == "session.created" || eventType == "session.updated" {
						if props, ok := event["properties"].(map[string]interface{}); ok {
							if info, ok := props["info"].(map[string]interface{}); ok {
								if sessionID, ok := info["id"].(string); ok {
									a.httpServer.SetCurrentSession(sessionID)
									fmt.Printf("📍 当前会话已更新: %s\n", sessionID)
								}
							}
						}
					}
				}
			}
			
			// 转发事件到所有连接的手机端
			a.httpServer.BroadcastEvent("server-event", data[0])
		}
	})

	// 自动启动远程控制服务
	go func() {
		time.Sleep(2 * time.Second) // 等待应用完全启动
		info, err := a.StartRemoteControl(8080)
		if err != nil {
			fmt.Printf("⚠️  远程控制启动失败: %v\n", err)
		} else {
			fmt.Println("========================================")
			fmt.Println("📱 OpenCode Mobile 远程控制已启动")
			fmt.Println("========================================")
			fmt.Printf("连接码: %s\n", info["token"])
			fmt.Printf("端口: %v\n", info["port"])
			fmt.Println("")
			fmt.Println("手机端访问步骤：")
			fmt.Println("1. 手机浏览器打开: http://[你的IP]:5173")
			fmt.Println("2. 输入连接码")
			fmt.Println("3. 开始使用")
			fmt.Println("========================================")
			
			// 发送事件到前端
			runtime.EventsEmit(ctx, "remote-control-started", info)
		}
	}()
}

func (a *App) SetServerURL(url string) {
	a.serverURL = strings.TrimSuffix(url, "/")
}

// GetServerURL 获取服务器地址
func (a *App) GetServerURL() string {
	return a.serverURL
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

// 获取 Java 主类名 (package.ClassName)
func (a *App) getJavaMainClass(filePath string) string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}

	code := string(content)

	// 查找 package 声明
	packageName := ""
	lines := strings.Split(code, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package ") && strings.HasSuffix(line, ";") {
			packageName = strings.TrimSuffix(strings.TrimPrefix(line, "package "), ";")
			break
		}
	}

	// 获取文件名作为类名 (Java 规范)
	className := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))

	if packageName != "" {
		return packageName + "." + className
	}
	return className
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
	return ""
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

// --- Kiro Account Manager Initialization ---

// initAccountManager initializes the Kiro Account Manager
func (a *App) initAccountManager() {
	fmt.Println("=== 初始化 Kiro 账号管理器 ===")

	// Initialize crypto service with a default master key
	// TODO: In production, this should be derived from user credentials or system keychain
	crypto := NewCryptoService("opencode-kiro-master-key-v1")
	fmt.Println("✓ 加密服务初始化完成")

	// Initialize configuration manager
	configMgr, err := NewConfigManager(crypto)
	if err != nil {
		fmt.Printf("✗ 创建配置管理器失败: %v\n", err)
		return
	}
	fmt.Println("✓ 配置管理器创建完成")

	// Initialize configuration and directory structure
	if err := configMgr.Initialize(); err != nil {
		fmt.Printf("✗ 初始化配置失败: %v\n", err)
		return
	}
	fmt.Println("✓ 配置初始化完成")

	// Get data directory from config manager
	dataDir := configMgr.GetDataDirectory()
	fmt.Printf("✓ 数据目录: %s\n", dataDir)

	// Initialize storage service with config-managed directory
	storage := NewStorageService(dataDir, crypto)
	fmt.Println("✓ 存储服务初始化完成")

	// Initialize account manager
	a.accountMgr = NewAccountManager(storage, crypto)
	fmt.Println("✓ 账号管理器初始化完成")

	// 加载现有账号
	accounts := a.accountMgr.ListAccounts()
	fmt.Printf("✓ 已加载 %d 个账号\n", len(accounts))

	// Store config manager reference for later use
	a.configMgr = configMgr

	fmt.Println("=== Kiro 账号管理器初始化完成 ===")
}

// --- Kiro Account Management API ---

// GetKiroAccounts returns all Kiro accounts
func (a *App) GetKiroAccounts() ([]*KiroAccount, error) {
	fmt.Println("=== API 调用: GetKiroAccounts ===")
	if a.accountMgr == nil {
		fmt.Println("✗ 错误: account manager not initialized")
		return nil, fmt.Errorf("account manager not initialized")
	}
	accounts := a.accountMgr.ListAccounts()
	fmt.Printf("→ 账号总数: %d\n", len(accounts))

	// 打印每个账号的详细信息
	for i, acc := range accounts {
		fmt.Printf("  账号 %d:\n", i+1)
		fmt.Printf("    ID: %s\n", acc.ID)
		fmt.Printf("    Email: %s\n", acc.Email)
		fmt.Printf("    DisplayName: %s\n", acc.DisplayName)
		fmt.Printf("    IsActive: %v\n", acc.IsActive)
		fmt.Printf("    RefreshToken 长度: %d\n", len(acc.RefreshToken))
		fmt.Printf("    BearerToken 长度: %d\n", len(acc.BearerToken))
	}

	// 同时检查 OpenCode 配置文件
	fmt.Println("\n→ 检查 OpenCode 配置文件:")
	openCodeSystem := NewOpenCodeKiroSystem()
	if activeAcc, err := openCodeSystem.GetActiveOpenCodeAccount(); err == nil {
		fmt.Printf("    OpenCode Email: %s\n", activeAcc.Email)
		fmt.Printf("    OpenCode ID: %s\n", activeAcc.ID)
	} else {
		fmt.Printf("    读取失败: %v\n", err)
	}

	fmt.Println("=== GetKiroAccounts 完成 ===")
	return accounts, nil
}

// AddKiroAccount adds a new Kiro account
func (a *App) AddKiroAccount(method string, data map[string]interface{}) error {
	fmt.Printf("API 调用: AddKiroAccount (method=%s)\n", method)
	if a.accountMgr == nil {
		fmt.Println("✗ 错误: account manager not initialized")
		return fmt.Errorf("account manager not initialized")
	}

	switch method {
	case "token":
		return a.addAccountByToken(data)
	case "oauth":
		return a.addAccountByOAuth(data)
	case "password":
		return a.addAccountByPassword(data)
	default:
		fmt.Printf("✗ 错误: unsupported login method: %s\n", method)
		return fmt.Errorf("unsupported login method: %s", method)
	}
}

// addAccountByToken adds an account using refresh token
func (a *App) addAccountByToken(data map[string]interface{}) error {
	fmt.Println("→ addAccountByToken 开始")
	refreshToken, ok := data["refreshToken"].(string)
	if !ok || refreshToken == "" {
		fmt.Println("✗ 错误: refresh token is required")
		return fmt.Errorf("refresh token is required")
	}
	fmt.Printf("  Refresh Token 长度: %d\n", len(refreshToken))

	// 使用新的 Kiro API 客户端
	fmt.Println("  创建 Kiro API 客户端...")
	kiroClient := NewKiroAPIClient()

	// Step 1: 刷新 Token 获取 Access Token
	fmt.Println("  调用 RefreshKiroToken...")
	tokenResp, err := kiroClient.RefreshKiroToken(refreshToken)
	if err != nil {
		fmt.Printf("✗ 刷新 Token 失败: %v\n", err)
		return fmt.Errorf("刷新 Token 失败: %w", err)
	}
	fmt.Println("  ✓ Token 刷新成功")

	// Step 2: 获取用户信息和配额
	fmt.Println("  调用 GetKiroUsageLimits...")
	usageResp, err := kiroClient.GetKiroUsageLimits(tokenResp.AccessToken)
	if err != nil {
		fmt.Printf("✗ 获取配额信息失败: %v\n", err)
		return fmt.Errorf("获取配额信息失败: %w", err)
	}
	fmt.Println("  ✓ 配额信息获取成功")

	// Step 3: 转换为账号对象
	fmt.Println("  转换为账号对象...")
	account := ConvertKiroResponseToAccount(tokenResp, usageResp, a.accountMgr)
	fmt.Printf("  账号邮箱: %s\n", account.Email)
	fmt.Printf("  订阅类型: %s\n", account.SubscriptionType)
	fmt.Printf("  主配额: %d/%d\n", account.Quota.Main.Used, account.Quota.Main.Total)

	// Add custom fields if provided
	if displayName, ok := data["displayName"].(string); ok && displayName != "" {
		account.DisplayName = displayName
	}
	if notes, ok := data["notes"].(string); ok {
		account.Notes = notes
	}
	if tags, ok := data["tags"].([]interface{}); ok {
		stringTags := make([]string, 0, len(tags))
		for _, tag := range tags {
			if strTag, ok := tag.(string); ok {
				stringTags = append(stringTags, strTag)
			}
		}
		account.Tags = stringTags
	}

	fmt.Println("  调用 AddAccount...")
	err = a.accountMgr.AddAccount(account)
	if err != nil {
		fmt.Printf("✗ 添加账号失败: %v\n", err)
		return err
	}
	fmt.Println("✓ 账号添加成功")
	return nil
}

// addAccountByOAuth adds an account using OAuth
func (a *App) addAccountByOAuth(data map[string]interface{}) error {
	// Provider is not strictly needed here as it's looked up by state,
	// but we can keep the check if desired.

	code, ok := data["code"].(string)
	if !ok || code == "" {
		return fmt.Errorf("OAuth code is required")
	}

	state, ok := data["state"].(string)
	if !ok || state == "" {
		return fmt.Errorf("OAuth state is required")
	}

	// Handle OAuth callback
	account, err := a.accountMgr.authService.HandleOAuthCallback(state, code)
	if err != nil {
		return fmt.Errorf("OAuth authentication failed: %w", err)
	}

	return a.accountMgr.AddAccount(account)
}

// addAccountByPassword adds an account using username/password
func (a *App) addAccountByPassword(data map[string]interface{}) error {
	email, ok := data["email"].(string)
	if !ok || email == "" {
		return fmt.Errorf("email is required")
	}

	password, ok := data["password"].(string)
	if !ok || password == "" {
		return fmt.Errorf("password is required")
	}

	// Authenticate with email and password
	account, err := a.accountMgr.authService.LoginWithPassword(email, password)
	if err != nil {
		return fmt.Errorf("password authentication failed: %w", err)
	}

	// Update quota information
	if err := a.accountMgr.authService.UpdateAccountQuota(account, a.accountMgr.quotaService); err != nil {
		// Log error but don't fail account creation
		fmt.Printf("Warning: failed to update quota for account %s: %v\n", account.Email, err)
	}

	// Add custom fields if provided
	if displayName, ok := data["displayName"].(string); ok && displayName != "" {
		account.DisplayName = displayName
	}
	if notes, ok := data["notes"].(string); ok {
		account.Notes = notes
	}
	if tags, ok := data["tags"].([]interface{}); ok {
		// Convert []interface{} to []string
		stringTags := make([]string, 0, len(tags))
		for _, tag := range tags {
			if strTag, ok := tag.(string); ok {
				stringTags = append(stringTags, strTag)
			}
		}
		account.Tags = stringTags
	}

	return a.accountMgr.AddAccount(account)
}

// RemoveKiroAccount removes a Kiro account
// RemoveKiroAccount removes a Kiro account
func (a *App) RemoveKiroAccount(id string) error {
	fmt.Printf("API 调用: RemoveKiroAccount (id=%s)\n", id)
	if a.accountMgr == nil {
		fmt.Println("✗ 错误: account manager not initialized")
		return fmt.Errorf("account manager not initialized")
	}
	err := a.accountMgr.RemoveAccount(id)
	if err != nil {
		fmt.Printf("✗ 删除失败: %v\n", err)
	} else {
		fmt.Println("✓ 账号删除成功")
	}
	return err
}

// UpdateKiroAccount updates a Kiro account
func (a *App) UpdateKiroAccount(id string, updates map[string]interface{}) error {
	fmt.Printf("API 调用: UpdateKiroAccount (id=%s)\n", id)
	if a.accountMgr == nil {
		fmt.Println("✗ 错误: account manager not initialized")
		return fmt.Errorf("account manager not initialized")
	}
	err := a.accountMgr.UpdateAccount(id, updates)
	if err != nil {
		fmt.Printf("✗ 更新失败: %v\n", err)
	} else {
		fmt.Println("✓ 账号更新成功")
	}
	return err
}

// SwitchKiroAccount switches the active Kiro account
func (a *App) SwitchKiroAccount(id string) error {
	fmt.Printf("=== API 调用: SwitchKiroAccount (id=%s) ===\n", id)
	if a.accountMgr == nil {
		fmt.Println("✗ 错误: account manager not initialized")
		return fmt.Errorf("account manager not initialized")
	}

	fmt.Println("→ 调用 accountMgr.SwitchAccount...")
	err := a.accountMgr.SwitchAccount(id)
	if err != nil {
		fmt.Printf("✗ 切换失败: %v\n", err)
		return err
	}

	fmt.Println("✓ 账号切换成功")

	// 获取切换后的账号（用于后续重新应用）
	switchedAccount, getErr := a.accountMgr.GetActiveAccount()
	if getErr != nil {
		fmt.Printf("⚠ 警告: 无法获取切换后的账号: %v\n", getErr)
	}

	// 重启 OpenCode 使新账号生效
	fmt.Println("→ 重启 OpenCode...")
	if a.openCode != nil {
		if restartErr := a.openCode.Restart(); restartErr != nil {
			fmt.Printf("⚠ 警告: OpenCode 重启失败: %v\n", restartErr)
			// 不返回错误，因为账号切换本身是成功的
		} else {
			fmt.Println("✓ OpenCode 已重启")

			// OpenCode 插件启动时可能会还原旧账号
			// 延迟 5 秒后再次应用账号，确保覆盖插件的还原操作
			if switchedAccount != nil {
				fmt.Println("→ 等待 5 秒后再次应用账号...")
				go func() {
					time.Sleep(5 * time.Second)
					fmt.Println("→ 再次应用账号到 OpenCode...")
					openCodeSystem := NewOpenCodeKiroSystem()
					if reapplyErr := openCodeSystem.ApplyAccountToOpenCode(switchedAccount); reapplyErr != nil {
						fmt.Printf("⚠ 警告: 再次应用账号失败: %v\n", reapplyErr)
					} else {
						fmt.Println("✓ 账号已再次应用，覆盖插件还原")
					}
				}()
			}
		}
	}

	fmt.Println("=== SwitchKiroAccount 完成 ===")
	return nil
}

// GetActiveKiroAccount returns the currently active Kiro account
func (a *App) GetActiveKiroAccount() (*KiroAccount, error) {
	if a.accountMgr == nil {
		return nil, fmt.Errorf("account manager not initialized")
	}
	account, err := a.accountMgr.GetActiveAccount()
	if err != nil {
		return nil, err
	}
	// Hide sensitive token information for frontend
	account.BearerToken = ""
	account.RefreshToken = ""
	return account, nil
}

// --- Tag Management ---

// GetTags returns all defined tags
func (a *App) GetTags() ([]Tag, error) {
	if a.accountMgr == nil {
		return nil, fmt.Errorf("account manager not initialized")
	}
	return a.accountMgr.GetTags(), nil
}

// AddTag adds a new tag
func (a *App) AddTag(tag Tag) error {
	if a.accountMgr == nil {
		return fmt.Errorf("account manager not initialized")
	}
	return a.accountMgr.AddTag(tag)
}

// DeleteTag deletes a tag
func (a *App) DeleteTag(tagName string) error {
	if a.accountMgr == nil {
		return fmt.Errorf("account manager not initialized")
	}
	return a.accountMgr.DeleteTag(tagName)
}

// --- Machine ID ---

// GetSystemMachineID returns the current system's machine ID
func (a *App) GetSystemMachineID() (string, error) {
	if a.accountMgr == nil {
		return "", fmt.Errorf("account manager not initialized")
	}
	return a.accountMgr.system.GenerateMachineID(), nil
}

// --- Authentication API ---

// StartKiroOAuth starts an OAuth flow for Kiro authentication
func (a *App) StartKiroOAuth(provider string) (string, error) {
	if a.accountMgr == nil {
		return "", fmt.Errorf("account manager not initialized")
	}
	url, err := a.accountMgr.authService.StartOAuthFlow(OAuthProvider(provider))
	if err != nil {
		return "", err
	}

	// Open the URL in the system browser
	runtime.BrowserOpenURL(a.ctx, url)

	return url, nil
}

// HandleKiroOAuthCallback handles OAuth callback
func (a *App) HandleKiroOAuthCallback(state string, code string) (*KiroAccount, error) {
	if a.accountMgr == nil {
		return nil, fmt.Errorf("account manager not initialized")
	}
	return a.accountMgr.authService.HandleOAuthCallback(state, code)
}

// CompleteKiroOAuthWithURL completes OAuth flow by parsing the callback URL
// The callback URL should be in the format: https://app.kiro.dev/signin/oauth?code=xxx&state=yyy
func (a *App) CompleteKiroOAuthWithURL(callbackURL string) error {
	fmt.Printf("[OAuth] CompleteKiroOAuthWithURL called with: %s\n", callbackURL[:min(len(callbackURL), 100)])

	if a.accountMgr == nil {
		return fmt.Errorf("account manager not initialized")
	}

	// Parse the URL to extract code and state
	parsedURL, err := url.Parse(callbackURL)
	if err != nil {
		return fmt.Errorf("invalid callback URL: %w", err)
	}

	code := parsedURL.Query().Get("code")
	state := parsedURL.Query().Get("state")

	if code == "" {
		return fmt.Errorf("missing 'code' parameter in callback URL")
	}
	if state == "" {
		return fmt.Errorf("missing 'state' parameter in callback URL")
	}

	fmt.Printf("[OAuth] Extracted code: %s..., state: %s...\n", code[:min(len(code), 20)], state[:min(len(state), 20)])

	// Handle the OAuth callback
	account, err := a.accountMgr.authService.HandleOAuthCallback(state, code)
	if err != nil {
		return fmt.Errorf("OAuth authentication failed: %w", err)
	}

	// Add the account
	if err := a.accountMgr.AddAccount(account); err != nil {
		return fmt.Errorf("failed to save account: %w", err)
	}

	fmt.Printf("[OAuth] Account added successfully: %s\n", account.Email)
	return nil
}

// ValidateKiroToken validates a Kiro bearer token
func (a *App) ValidateKiroToken(token string) (*TokenInfo, error) {
	if a.accountMgr == nil {
		return nil, fmt.Errorf("account manager not initialized")
	}
	return a.accountMgr.authService.ValidateToken(token)
}

// RefreshKiroToken refreshes a Kiro account token
func (a *App) RefreshKiroToken(accountId string) error {
	if a.accountMgr == nil {
		return fmt.Errorf("account manager not initialized")
	}

	account, err := a.accountMgr.GetAccount(accountId)
	if err != nil {
		return err
	}

	tokenInfo, err := a.accountMgr.authService.RefreshToken(account.RefreshToken)
	if err != nil {
		return err
	}

	// Update account with new token info
	updates := map[string]interface{}{
		"bearerToken":  tokenInfo.AccessToken,
		"refreshToken": tokenInfo.RefreshToken,
		"tokenExpiry":  tokenInfo.ExpiresAt,
	}

	return a.accountMgr.UpdateAccount(accountId, updates)
}

// --- Quota API ---

// GetKiroQuota gets quota information for an account
func (a *App) GetKiroQuota(accountId string) (*QuotaInfo, error) {
	if a.accountMgr == nil {
		return nil, fmt.Errorf("account manager not initialized")
	}

	account, err := a.accountMgr.GetAccount(accountId)
	if err != nil {
		return nil, err
	}

	return a.accountMgr.quotaService.GetQuota(account.BearerToken)
}

// RefreshActiveKiroQuota refreshes quota for the currently active account in OpenCode
// This is useful when you want to check the quota of the account that OpenCode is actually using
func (a *App) RefreshActiveKiroQuota() error {
	fmt.Println("=== API 调用: RefreshActiveKiroQuota ===")

	if a.accountMgr == nil {
		fmt.Println("✗ 错误: account manager not initialized")
		return fmt.Errorf("account manager not initialized")
	}

	// 获取 OpenCode 当前激活的账号
	fmt.Println("→ 读取 OpenCode 当前激活账号...")
	openCodeSystem := NewOpenCodeKiroSystem()
	activeOpenCodeAccount, err := openCodeSystem.GetActiveOpenCodeAccount()
	if err != nil {
		fmt.Printf("✗ 获取 OpenCode 激活账号失败: %v\n", err)
		return fmt.Errorf("获取 OpenCode 激活账号失败: %w", err)
	}
	fmt.Printf("✓ OpenCode 当前使用账号: %s\n", activeOpenCodeAccount.Email)

	// 在我们的账号管理器中查找对应的账号
	fmt.Println("→ 查找对应的账号...")
	var accountId string
	accounts := a.accountMgr.ListAccounts()
	for _, acc := range accounts {
		if acc.Email == activeOpenCodeAccount.Email || acc.ID == activeOpenCodeAccount.ID {
			accountId = acc.ID
			break
		}
	}

	if accountId == "" {
		fmt.Printf("✗ 未找到对应账号: %s\n", activeOpenCodeAccount.Email)
		return fmt.Errorf("未找到 OpenCode 使用的账号: %s", activeOpenCodeAccount.Email)
	}
	fmt.Printf("✓ 找到账号 ID: %s\n", accountId)

	// 刷新该账号的配额
	fmt.Println("→ 刷新账号配额...")
	if err := a.RefreshKiroQuota(accountId); err != nil {
		fmt.Printf("✗ 刷新失败: %v\n", err)
		return err
	}

	fmt.Println("✓ RefreshActiveKiroQuota 完成")
	return nil
}

// RefreshKiroQuota refreshes quota information for an account
func (a *App) RefreshKiroQuota(accountId string) error {
	fmt.Printf("=== API 调用: RefreshKiroQuota (accountId=%s) ===\n", accountId)

	if a.accountMgr == nil {
		fmt.Println("✗ 错误: account manager not initialized")
		return fmt.Errorf("account manager not initialized")
	}

	fmt.Println("→ 获取账号信息...")
	account, err := a.accountMgr.GetAccount(accountId)
	if err != nil {
		fmt.Printf("✗ 获取账号失败: %v\n", err)
		return err
	}
	fmt.Printf("✓ 账号: %s\n", account.Email)

	// 先刷新 Token（如果有 RefreshToken）
	if account.RefreshToken != "" {
		fmt.Println("→ 刷新 Bearer Token...")
		tokenInfo, err := a.accountMgr.authService.RefreshToken(account.RefreshToken)
		if err != nil {
			fmt.Printf("✗ Token 刷新失败: %v\n", err)
			return fmt.Errorf("token 刷新失败: %w", err)
		}

		// 更新账号的 Token
		updates := map[string]interface{}{
			"bearerToken":  tokenInfo.AccessToken,
			"refreshToken": tokenInfo.RefreshToken,
			"tokenExpiry":  tokenInfo.ExpiresAt,
		}
		if err := a.accountMgr.UpdateAccount(accountId, updates); err != nil {
			fmt.Printf("✗ 更新 Token 失败: %v\n", err)
			return fmt.Errorf("更新 token 失败: %w", err)
		}

		// 重新获取账号（使用新的 Token）
		account, err = a.accountMgr.GetAccount(accountId)
		if err != nil {
			fmt.Printf("✗ 重新获取账号失败: %v\n", err)
			return err
		}
		fmt.Println("✓ Bearer Token 刷新成功")
	}

	// Refresh quota
	fmt.Println("→ 调用 quotaService.RefreshQuota...")
	if err := a.accountMgr.quotaService.RefreshQuota(accountId, account.BearerToken); err != nil {
		fmt.Printf("✗ 刷新配额失败: %v\n", err)
		return err
	}
	fmt.Println("✓ 配额刷新成功")

	// Get updated quota and update account
	fmt.Println("→ 获取更新后的配额...")
	quota, err := a.accountMgr.quotaService.GetQuota(account.BearerToken)
	if err != nil {
		fmt.Printf("✗ 获取配额失败: %v\n", err)
		return err
	}
	fmt.Printf("✓ 配额: Used=%d, Total=%d\n",
		quota.Main.Used+quota.Trial.Used+quota.Reward.Used,
		quota.Main.Total+quota.Trial.Total+quota.Reward.Total)

	updates := map[string]interface{}{
		"quota": *quota,
	}

	fmt.Println("→ 更新账号配额...")
	err = a.accountMgr.UpdateAccount(accountId, updates)
	if err != nil {
		fmt.Printf("✗ 更新账号失败: %v\n", err)
	} else {
		fmt.Println("✓ 账号配额更新成功")
	}

	fmt.Println("=== RefreshKiroQuota 完成 ===")
	return err
}

// BatchRefreshKiroQuota refreshes quota for multiple accounts
func (a *App) BatchRefreshKiroQuota(accountIds []string) error {
	if a.accountMgr == nil {
		return fmt.Errorf("account manager not initialized")
	}

	var accounts []*KiroAccount
	for _, id := range accountIds {
		account, err := a.accountMgr.GetAccount(id)
		if err != nil {
			continue // Skip invalid accounts
		}
		accounts = append(accounts, account)
	}

	return a.accountMgr.quotaService.BatchRefreshQuota(accounts)
}

// GetQuotaAlerts returns quota alerts for all accounts
func (a *App) GetQuotaAlerts() ([]QuotaAlert, error) {
	if a.accountMgr == nil {
		return nil, fmt.Errorf("account manager not initialized")
	}
	return a.accountMgr.GetQuotaAlerts(0.9), nil // 90% threshold
}

// --- Batch Operations API ---

// BatchRefreshKiroTokens refreshes tokens for multiple accounts
func (a *App) BatchRefreshKiroTokens(accountIds []string) error {
	if a.accountMgr == nil {
		return fmt.Errorf("account manager not initialized")
	}
	return a.accountMgr.BatchRefreshTokens(accountIds)
}

// BatchDeleteKiroAccounts deletes multiple accounts
func (a *App) BatchDeleteKiroAccounts(accountIds []string) error {
	if a.accountMgr == nil {
		return fmt.Errorf("account manager not initialized")
	}
	return a.accountMgr.BatchDeleteAccounts(accountIds)
}

// BatchAddKiroTags adds tags to multiple accounts
func (a *App) BatchAddKiroTags(accountIds []string, tags []string) error {
	if a.accountMgr == nil {
		return fmt.Errorf("account manager not initialized")
	}
	return a.accountMgr.BatchAddTags(accountIds, tags)
}

// --- Data Management API ---

// ExportKiroAccounts exports accounts to JSON with optional encryption
func (a *App) ExportKiroAccounts(password string) (string, error) {
	if a.accountMgr == nil {
		return "", fmt.Errorf("account manager not initialized")
	}

	data, err := a.accountMgr.ExportAccounts(password)
	if err != nil {
		return "", err
	}

	// Save to temporary file and return path
	tempDir := os.TempDir()
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("kiro_accounts_export_%s.json", timestamp)
	if password != "" {
		filename += ".enc"
	}

	filePath := filepath.Join(tempDir, filename)
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return "", fmt.Errorf("failed to write export file: %w", err)
	}

	return filePath, nil
}

// ImportKiroAccounts imports accounts from JSON file
func (a *App) ImportKiroAccounts(filePath string, password string) error {
	if a.accountMgr == nil {
		return fmt.Errorf("account manager not initialized")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}

	return a.accountMgr.ImportAccounts(data, password)
}

// --- Statistics API ---

// GetKiroAccountStats returns statistics about managed accounts
func (a *App) GetKiroAccountStats() (map[string]interface{}, error) {
	if a.accountMgr == nil {
		return nil, fmt.Errorf("account manager not initialized")
	}
	return a.accountMgr.GetAccountStats(), nil
}

// --- Configuration Management API ---

// GetConfigPaths returns the configuration paths
func (a *App) GetConfigPaths() (ConfigPaths, error) {
	if a.configMgr == nil {
		return ConfigPaths{}, fmt.Errorf("configuration manager not initialized")
	}
	return a.configMgr.GetPaths(), nil
}

// GetAppConfig returns the current application configuration
func (a *App) GetAppConfig() (*AppConfig, error) {
	if a.configMgr == nil {
		return nil, fmt.Errorf("configuration manager not initialized")
	}
	return a.configMgr.LoadAppConfig()
}

// UpdateAppConfig updates the application configuration
func (a *App) UpdateAppConfig(config *AppConfig) error {
	if a.configMgr == nil {
		return fmt.Errorf("configuration manager not initialized")
	}
	return a.configMgr.SaveAppConfig(config)
}

// GetAccountSettings returns the current account settings
func (a *App) GetAccountSettings() (AccountSettings, error) {
	if a.accountMgr == nil {
		return AccountSettings{}, fmt.Errorf("account manager not initialized")
	}
	return a.accountMgr.GetSettings()
}

// UpdateAccountSettings updates the account settings
func (a *App) UpdateAccountSettings(settings AccountSettings) error {
	if a.accountMgr == nil {
		return fmt.Errorf("account manager not initialized")
	}
	return a.accountMgr.UpdateSettings(settings)
}

// GetStorageInfo returns storage usage information
func (a *App) GetStorageInfo() (map[string]interface{}, error) {
	if a.configMgr == nil {
		return nil, fmt.Errorf("configuration manager not initialized")
	}
	return a.configMgr.GetStorageInfo()
}

// CleanupTempFiles cleans up temporary files
func (a *App) CleanupTempFiles() error {
	if a.configMgr == nil {
		return fmt.Errorf("configuration manager not initialized")
	}
	return a.configMgr.CleanupTempDirectory()
}

// LogToTerminal logs a message to the terminal
func (a *App) LogToTerminal(message string) {
	fmt.Println(message)
	fmt.Fprintf(os.Stderr, "%s\n", message)
}

// ValidateConfiguration validates the current configuration
func (a *App) ValidateConfiguration() error {
	if a.configMgr == nil {
		return fmt.Errorf("configuration manager not initialized")
	}
	return a.configMgr.Validate()
}

// ResetConfiguration resets configuration to defaults (creates backup)
func (a *App) ResetConfiguration() error {
	if a.configMgr == nil {
		return fmt.Errorf("configuration manager not initialized")
	}
	return a.configMgr.Reset()
}


// --- Skills Management API ---

// skillsMgr 技能管理器（延迟初始化）
func (a *App) getSkillsManager() *SkillsManager {
	workDir := ""
	if a.fileMgr != nil {
		workDir = a.fileMgr.GetRootDir()
	}
	return NewSkillsManager(workDir)
}

// GetSkills 获取所有技能
func (a *App) GetSkills() ([]SkillInfo, error) {
	fmt.Println("=== API 调用: GetSkills ===")
	sm := a.getSkillsManager()
	skills, err := sm.ListSkills()
	if err != nil {
		fmt.Printf("✗ 获取技能失败: %v\n", err)
		return nil, err
	}
	fmt.Printf("✓ 获取到 %d 个技能\n", len(skills))
	return skills, nil
}

// GetSkill 获取单个技能详情
func (a *App) GetSkill(name string) (*SkillInfo, error) {
	fmt.Printf("=== API 调用: GetSkill (name=%s) ===\n", name)
	sm := a.getSkillsManager()
	skill, err := sm.GetSkill(name)
	if err != nil {
		fmt.Printf("✗ 获取技能失败: %v\n", err)
		return nil, err
	}
	fmt.Printf("✓ 获取技能: %s\n", skill.Name)
	return skill, nil
}

// CreateSkill 创建新技能
func (a *App) CreateSkill(name, description, content string, global bool) error {
	fmt.Printf("=== API 调用: CreateSkill (name=%s, global=%v) ===\n", name, global)
	sm := a.getSkillsManager()
	err := sm.CreateSkill(name, description, content, global)
	if err != nil {
		fmt.Printf("✗ 创建技能失败: %v\n", err)
		return err
	}
	fmt.Println("✓ 技能创建成功")
	return nil
}

// UpdateSkill 更新技能
func (a *App) UpdateSkill(name, description, content string) error {
	fmt.Printf("=== API 调用: UpdateSkill (name=%s) ===\n", name)
	sm := a.getSkillsManager()
	err := sm.UpdateSkill(name, description, content)
	if err != nil {
		fmt.Printf("✗ 更新技能失败: %v\n", err)
		return err
	}
	fmt.Println("✓ 技能更新成功")
	return nil
}

// DeleteSkill 删除技能
func (a *App) DeleteSkill(name string) error {
	fmt.Printf("=== API 调用: DeleteSkill (name=%s) ===\n", name)
	sm := a.getSkillsManager()
	err := sm.DeleteSkill(name)
	if err != nil {
		fmt.Printf("✗ 删除技能失败: %v\n", err)
		return err
	}
	fmt.Println("✓ 技能删除成功")
	return nil
}

// GetSkillTemplates 获取技能模板列表
func (a *App) GetSkillTemplates() []SkillTemplate {
	fmt.Println("=== API 调用: GetSkillTemplates ===")
	sm := a.getSkillsManager()
	templates := sm.GetSkillTemplates()
	fmt.Printf("✓ 获取到 %d 个模板\n", len(templates))
	return templates
}

// CreateSkillFromTemplate 从模板创建技能
func (a *App) CreateSkillFromTemplate(templateID, customName string, global bool) error {
	fmt.Printf("=== API 调用: CreateSkillFromTemplate (template=%s, name=%s, global=%v) ===\n", templateID, customName, global)
	sm := a.getSkillsManager()
	err := sm.CreateSkillFromTemplate(templateID, customName, global)
	if err != nil {
		fmt.Printf("✗ 从模板创建技能失败: %v\n", err)
		return err
	}
	fmt.Println("✓ 技能创建成功")
	return nil
}

// --- Remote Control API ---

// StartRemoteControl 启动远程控制服务器
func (a *App) StartRemoteControl(port int) (map[string]interface{}, error) {
	fmt.Printf("=== API 调用: StartRemoteControl (port=%d) ===\n", port)
	
	if a.httpServer == nil {
		a.httpServer = NewHTTPServer(a)
	}
	
	err := a.httpServer.Start(port)
	if err != nil {
		fmt.Printf("✗ 启动失败: %v\n", err)
		return nil, err
	}
	
	info := map[string]interface{}{
		"active": true,
		"port":   a.httpServer.GetPort(),
		"token":  a.httpServer.GetToken(),
		"url":    fmt.Sprintf("http://localhost:%d", a.httpServer.GetPort()),
	}
	
	fmt.Printf("✓ 远程控制服务器已启动\n")
	fmt.Printf("  端口: %d\n", info["port"])
	fmt.Printf("  令牌: %s\n", info["token"])
	
	return info, nil
}

// StopRemoteControl 停止远程控制服务器
func (a *App) StopRemoteControl() error {
	fmt.Println("=== API 调用: StopRemoteControl ===")
	
	if a.httpServer == nil {
		fmt.Println("✓ 服务器未运行")
		return nil
	}
	
	err := a.httpServer.Stop()
	if err != nil {
		fmt.Printf("✗ 停止失败: %v\n", err)
		return err
	}
	
	fmt.Println("✓ 远程控制服务器已停止")
	return nil
}

// GetRemoteControlInfo 获取远程控制信息
func (a *App) GetRemoteControlInfo() (map[string]interface{}, error) {
	if a.httpServer == nil || !a.httpServer.IsActive() {
		return map[string]interface{}{
			"active": false,
		}, nil
	}
	
	return map[string]interface{}{
		"active": true,
		"port":   a.httpServer.GetPort(),
		"token":  a.httpServer.GetToken(),
		"url":    fmt.Sprintf("http://localhost:%d", a.httpServer.GetPort()),
	}, nil
}
