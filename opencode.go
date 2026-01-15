package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// OpenCodeManager 管理 OpenCode CLI 的检测、安装和启动
type OpenCodeManager struct {
	app     *App
	cmd     *exec.Cmd
	mu      sync.Mutex
	running bool
}

// NewOpenCodeManager 创建 OpenCode 管理器
func NewOpenCodeManager(app *App) *OpenCodeManager {
	return &OpenCodeManager{app: app}
}

// OpenCodeStatus 状态信息
type OpenCodeStatus struct {
	Installed bool   `json:"installed"`
	Running   bool   `json:"running"`
	Connected bool   `json:"connected"`
	Path      string `json:"path"`
	Version   string `json:"version"`
}

// CheckInstalled 检查 OpenCode 是否已安装
func (m *OpenCodeManager) CheckInstalled() (bool, string) {
	// 尝试查找 opencode 命令
	path, err := exec.LookPath("opencode")
	if err == nil {
		return true, path
	}

	// 检查常见安装位置
	homeDir, _ := os.UserHomeDir()
	commonPaths := []string{
		filepath.Join(homeDir, ".local", "bin", "opencode"),
		filepath.Join(homeDir, "go", "bin", "opencode"),
		"/usr/local/bin/opencode",
		"/usr/bin/opencode",
	}

	if runtime.GOOS == "windows" {
		commonPaths = []string{
			filepath.Join(homeDir, "go", "bin", "opencode.exe"),
			filepath.Join(os.Getenv("PROGRAMFILES"), "opencode", "opencode.exe"),
		}
	}

	for _, p := range commonPaths {
		if _, err := os.Stat(p); err == nil {
			return true, p
		}
	}

	return false, ""
}

// GetVersion 获取 OpenCode 版本
func (m *OpenCodeManager) GetVersion(path string) string {
	if path == "" {
		return ""
	}
	cmd := exec.Command(path, "--version")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// GetStatus 获取完整状态
func (m *OpenCodeManager) GetStatus() *OpenCodeStatus {
	installed, path := m.CheckInstalled()
	version := ""
	if installed {
		version = m.GetVersion(path)
	}

	connected := m.CheckConnection()

	m.mu.Lock()
	running := m.running
	m.mu.Unlock()

	return &OpenCodeStatus{
		Installed: installed,
		Running:   running,
		Connected: connected,
		Path:      path,
		Version:   version,
	}
}

// CheckConnection 检查是否能连接到 OpenCode 服务
func (m *OpenCodeManager) CheckConnection() bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(m.app.serverURL + "/session")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// Install 安装 OpenCode
func (m *OpenCodeManager) Install() error {
	// 使用 go install 安装
	cmd := exec.Command("go", "install", "github.com/opencode-ai/opencode@latest")
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("安装失败: %s, %v", string(output), err)
	}

	// 通知前端
	wailsRuntime.EventsEmit(m.app.ctx, "opencode-installed", true)
	return nil
}

// Start 后台启动 OpenCode
func (m *OpenCodeManager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 如果已经在运行，直接返回
	if m.running && m.cmd != nil && m.cmd.Process != nil {
		return nil
	}

	// 检查是否已经有服务在运行
	if m.CheckConnection() {
		m.running = true
		wailsRuntime.EventsEmit(m.app.ctx, "opencode-status", "connected")
		return nil
	}

	// 查找 opencode 路径
	installed, path := m.CheckInstalled()
	if !installed {
		return fmt.Errorf("OpenCode 未安装")
	}

	// 后台启动 opencode serve (headless 模式)
	m.cmd = exec.Command(path, "serve")
	m.cmd.Env = os.Environ()

	// 不显示窗口（静默运行）
	m.setupHiddenProcess(m.cmd)

	err := m.cmd.Start()
	if err != nil {
		return fmt.Errorf("启动失败: %v", err)
	}

	m.running = true

	// 等待服务就绪
	go m.waitForReady()

	return nil
}

// waitForReady 等待服务就绪
func (m *OpenCodeManager) waitForReady() {
	maxRetries := 30 // 最多等待 30 秒
	for i := 0; i < maxRetries; i++ {
		time.Sleep(1 * time.Second)
		if m.CheckConnection() {
			wailsRuntime.EventsEmit(m.app.ctx, "opencode-status", "connected")
			return
		}
	}
	wailsRuntime.EventsEmit(m.app.ctx, "opencode-status", "timeout")
}

// Stop 停止 OpenCode
func (m *OpenCodeManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd != nil && m.cmd.Process != nil {
		m.cmd.Process.Kill()
		m.cmd = nil
	}
	m.running = false
}

// AutoStart 自动检测并启动
func (m *OpenCodeManager) AutoStart() error {
	status := m.GetStatus()

	// 已连接，无需操作
	if status.Connected {
		wailsRuntime.EventsEmit(m.app.ctx, "opencode-status", "connected")
		return nil
	}

	// 未安装
	if !status.Installed {
		wailsRuntime.EventsEmit(m.app.ctx, "opencode-status", "not-installed")
		return fmt.Errorf("OpenCode 未安装")
	}

	// 已安装但未运行，启动它
	return m.Start()
}
