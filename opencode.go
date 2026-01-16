package main

import (
	"bufio"
	"fmt"
	"io"
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

// OpenCodeInstance 单个 OpenCode 实例
type OpenCodeInstance struct {
	cmd     *exec.Cmd
	workDir string
	port    int
	running bool
}

// OpenCodeManager 管理多个 OpenCode 实例
type OpenCodeManager struct {
	app        *App
	instances  map[string]*OpenCodeInstance
	mu         sync.Mutex
	currentDir string
}

func NewOpenCodeManager(app *App) *OpenCodeManager {
	return &OpenCodeManager{
		app:       app,
		instances: make(map[string]*OpenCodeInstance),
	}
}

func (m *OpenCodeManager) getPortForDir(dir string) int {
	// 使用固定端口，简化逻辑
	return 4096
}

func (m *OpenCodeManager) SetWorkDir(dir string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.currentDir = dir
	port := m.getPortForDir(dir)
	m.app.serverURL = fmt.Sprintf("http://localhost:%d", port)
}

func (m *OpenCodeManager) GetWorkDir() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.currentDir
}

func (m *OpenCodeManager) GetCurrentPort() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.currentDir == "" {
		return 4096
	}
	return m.getPortForDir(m.currentDir)
}

type OpenCodeStatus struct {
	Installed bool   `json:"installed"`
	Running   bool   `json:"running"`
	Connected bool   `json:"connected"`
	Path      string `json:"path"`
	Version   string `json:"version"`
	Port      int    `json:"port"`
	WorkDir   string `json:"workDir"`
}

func (m *OpenCodeManager) CheckInstalled() (bool, string) {
	path, err := exec.LookPath("opencode")
	if err == nil {
		return true, path
	}
	homeDir, _ := os.UserHomeDir()
	var paths []string
	if runtime.GOOS == "windows" {
		paths = []string{filepath.Join(homeDir, ".opencode", "bin", "opencode.exe")}
	} else {
		paths = []string{
			filepath.Join(homeDir, ".opencode", "bin", "opencode"),
			filepath.Join(homeDir, ".local", "bin", "opencode"),
			"/usr/local/bin/opencode",
		}
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return true, p
		}
	}
	return false, ""
}

func (m *OpenCodeManager) GetVersion(path string) string {
	if path == "" {
		return ""
	}
	cmd := exec.Command(path, "--version")
	out, _ := cmd.Output()
	return strings.TrimSpace(string(out))
}

func (m *OpenCodeManager) CheckConnectionForPort(port int) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/session", port))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func (m *OpenCodeManager) CheckConnection() bool {
	return m.CheckConnectionForPort(m.GetCurrentPort())
}

func (m *OpenCodeManager) GetStatus() *OpenCodeStatus {
	installed, path := m.CheckInstalled()
	version := ""
	if installed {
		version = m.GetVersion(path)
	}
	port := m.GetCurrentPort()
	connected := m.CheckConnectionForPort(port)
	m.mu.Lock()
	workDir := m.currentDir
	running := false
	if inst, ok := m.instances[workDir]; ok {
		running = inst.running
	}
	m.mu.Unlock()
	return &OpenCodeStatus{Installed: installed, Running: running, Connected: connected, Path: path, Version: version, Port: port, WorkDir: workDir}
}

func (m *OpenCodeManager) Install() error {
	wailsRuntime.EventsEmit(m.app.ctx, "output-log", "正在安装 OpenCode...")
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell", "-Command", "irm https://opencode.ai/install.ps1 | iex")
	} else {
		cmd = exec.Command("bash", "-c", "curl -fsSL https://opencode.ai/install | bash")
	}
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		wailsRuntime.EventsEmit(m.app.ctx, "output-log", fmt.Sprintf("安装失败: %s", string(out)))
		return err
	}
	wailsRuntime.EventsEmit(m.app.ctx, "output-log", "OpenCode 安装完成")
	wailsRuntime.EventsEmit(m.app.ctx, "opencode-installed", true)
	return nil
}

func (m *OpenCodeManager) StartForDir(dir string) error {
	if dir == "" {
		return fmt.Errorf("目录不能为空")
	}
	port := m.getPortForDir(dir)

	// 如果已经有实例在运行，检查是否是同一个目录
	m.mu.Lock()
	for d, inst := range m.instances {
		if d != dir && inst.running {
			// 停止其他目录的实例
			wailsRuntime.EventsEmit(m.app.ctx, "output-log", fmt.Sprintf("停止目录 %s 的实例", d))
			if inst.cmd != nil && inst.cmd.Process != nil {
				inst.cmd.Process.Kill()
			}
			delete(m.instances, d)
		}
	}
	
	if inst, ok := m.instances[dir]; ok && inst.running {
		m.mu.Unlock()
		wailsRuntime.EventsEmit(m.app.ctx, "output-log", fmt.Sprintf("目录 %s 已在运行 (端口 %d)", dir, port))
		return nil
	}
	m.mu.Unlock()

	// 先杀掉可能占用端口的进程
	if runtime.GOOS != "windows" {
		exec.Command("bash", "-c", fmt.Sprintf("lsof -ti:%d | xargs kill -9 2>/dev/null", port)).Run()
		time.Sleep(500 * time.Millisecond)
	}

	installed, path := m.CheckInstalled()
	if !installed {
		return fmt.Errorf("OpenCode 未安装")
	}

	wailsRuntime.EventsEmit(m.app.ctx, "output-log", fmt.Sprintf("启动 OpenCode: %s (端口 %d)", dir, port))

	cmd := exec.Command(path, "serve", "--port", fmt.Sprintf("%d", port), "--print-logs")
	cmd.Dir = dir
	cmd.Env = os.Environ()
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	m.setupHiddenProcess(cmd)

	if err := cmd.Start(); err != nil {
		wailsRuntime.EventsEmit(m.app.ctx, "output-log", fmt.Sprintf("启动失败: %v", err))
		return err
	}

	inst := &OpenCodeInstance{cmd: cmd, workDir: dir, port: port, running: true}
	m.mu.Lock()
	m.instances[dir] = inst
	m.mu.Unlock()

	wailsRuntime.EventsEmit(m.app.ctx, "output-log", fmt.Sprintf("OpenCode 已启动 (PID %d)", cmd.Process.Pid))
	go m.readOutput(stdout)
	go m.readOutput(stderr)
	go func() {
		cmd.Wait()
		m.mu.Lock()
		if i, ok := m.instances[dir]; ok {
			i.running = false
		}
		m.mu.Unlock()
	}()
	go m.waitForReadyOnPort(port)
	return nil
}

func (m *OpenCodeManager) Start() error {
	dir := m.GetWorkDir()
	if dir == "" {
		homeDir, _ := os.UserHomeDir()
		dir = homeDir
		m.SetWorkDir(dir)
	}
	return m.StartForDir(dir)
}

func (m *OpenCodeManager) readOutput(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if line := scanner.Text(); line != "" {
			wailsRuntime.EventsEmit(m.app.ctx, "output-log", line)
		}
	}
}

func (m *OpenCodeManager) waitForReadyOnPort(port int) {
	for i := 0; i < 30; i++ {
		time.Sleep(time.Second)
		if m.CheckConnectionForPort(port) {
			wailsRuntime.EventsEmit(m.app.ctx, "output-log", fmt.Sprintf("服务就绪 (端口 %d)", port))
			wailsRuntime.EventsEmit(m.app.ctx, "opencode-status", "connected")
			return
		}
	}
	wailsRuntime.EventsEmit(m.app.ctx, "output-log", "连接超时")
	wailsRuntime.EventsEmit(m.app.ctx, "opencode-status", "timeout")
}

func (m *OpenCodeManager) Stop() {
	dir := m.GetWorkDir()
	m.StopForDir(dir)
	// 确保实例被清理
	m.mu.Lock()
	delete(m.instances, dir)
	m.mu.Unlock()
}

func (m *OpenCodeManager) StopForDir(dir string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if inst, ok := m.instances[dir]; ok {
		if inst.cmd != nil && inst.cmd.Process != nil {
			inst.cmd.Process.Kill()
			inst.cmd.Wait() // 等待进程退出
		}
		inst.running = false
		delete(m.instances, dir)
	}
}

func (m *OpenCodeManager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, inst := range m.instances {
		if inst.cmd != nil && inst.cmd.Process != nil {
			inst.cmd.Process.Kill()
		}
	}
	m.instances = make(map[string]*OpenCodeInstance)
}

func (m *OpenCodeManager) AutoStart() error {
	status := m.GetStatus()
	if status.Connected {
		wailsRuntime.EventsEmit(m.app.ctx, "opencode-status", "connected")
		return nil
	}
	if !status.Installed {
		wailsRuntime.EventsEmit(m.app.ctx, "opencode-status", "not-installed")
		return fmt.Errorf("OpenCode 未安装")
	}
	return m.Start()
}

func (m *OpenCodeManager) Restart() error {
	dir := m.GetWorkDir()
	wailsRuntime.EventsEmit(m.app.ctx, "output-log", fmt.Sprintf("切换到目录: %s", dir))
	return m.StartForDir(dir)
}
