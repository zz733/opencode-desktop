package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	goruntime "runtime"
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
	m.mu.Lock()
	defer m.mu.Unlock()
	if inst, ok := m.instances[dir]; ok && inst.port > 0 {
		return inst.port
	}
	// 使用固定初始端口，如果被占用会动态分配
	return 4096
}

func (m *OpenCodeManager) isPortUsedByUs(port int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, inst := range m.instances {
		if inst.running && inst.port == port {
			return true
		}
	}
	return false
}

func (m *OpenCodeManager) getAvailablePort(preferred int) int {
	check := func(port int) bool {
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			return false
		}
		_ = ln.Close()
		return true
	}

	for p := preferred; p <= preferred+50; p++ {
		if check(p) {
			return p
		}
		// 端口被占用，检查是否是我们自己的实例
		if !m.isPortUsedByUs(p) {
			// 不是我们当前管理的实例占用的（可能是僵尸进程），清理掉并复用
			// 只有在真正清理了进程后，才重新检查端口
			if m.cleanupPortProcesses(p) {
				time.Sleep(200 * time.Millisecond) // 清理后稍微等待一下
				if check(p) {
					return p
				}
			}
		} else {
			// 如果是我们自己记录的实例，但端口仍然被占用，说明实例可能已经崩溃但没清理
			// 这种情况下也尝试清理并复用
			if m.cleanupPortProcesses(p) {
				time.Sleep(200 * time.Millisecond) // 清理后稍微等待一下
				if check(p) {
					return p
				}
			}
		}
	}
	return preferred
}

func (m *OpenCodeManager) SetWorkDir(dir string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.currentDir = dir
	port := 4096
	if inst, ok := m.instances[dir]; ok && inst.port > 0 {
		port = inst.port
	}
	m.app.serverURL = fmt.Sprintf("http://127.0.0.1:%d", port)
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
	if inst, ok := m.instances[m.currentDir]; ok && inst.port > 0 {
		return inst.port
	}
	return 4096
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
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	healthURL := fmt.Sprintf("http://127.0.0.1:%d/global/health", port)
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return false
	}

	if resp, err := m.app.apiClient.Do(req); err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var result struct {
				Healthy bool `json:"healthy"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return true
			}
			if result.Healthy {
				return true
			}
		}
		// 如果能连上但不是 200，说明这不是 opencode，直接返回 false，不要再试其他 endpoint
		return false
	} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		// 如果第一个请求超时，说明该端口响应迟缓（可能是其他服务），直接返回 false
		return false
	} else if err != nil {
		// 如果是 Connection Refused 之类的错误，说明端口没有被监听
		return false
	}

	eventURL := fmt.Sprintf("http://127.0.0.1:%d/event", port)
	reqEvent, _ := http.NewRequestWithContext(ctx, "GET", eventURL, nil)
	if resp, err := m.app.apiClient.Do(reqEvent); err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return true
		}
		return false
	}

	sessionURL := fmt.Sprintf("http://127.0.0.1:%d/session", port)
	reqSession, _ := http.NewRequestWithContext(ctx, "GET", sessionURL, nil)
	if resp, err := m.app.apiClient.Do(reqSession); err == nil {
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}
	return false
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

	// 0. 在启动新实例前，停止其他所有实例，确保只有一个 opencode 进程在运行
	// 这样可以解决多进程占用内存过高的问题
	m.mu.Lock()
	for d, inst := range m.instances {
		if d != dir && inst.running {
			wailsRuntime.EventsEmit(m.app.ctx, "output-log", fmt.Sprintf("正在停止非当前目录的实例: %s", d))
			if inst.cmd != nil && inst.cmd.Process != nil {
				inst.cmd.Process.Kill()
				inst.cmd.Wait()
			}
			inst.running = false
			delete(m.instances, d)
		}
	}
	m.mu.Unlock()

	preferredPort := m.getPortForDir(dir)
	port := preferredPort

	// 1. 检查内部实例状态：如果该目录实例已在运行，复用之
	m.mu.Lock()
	if inst, ok := m.instances[dir]; ok && inst.running {
		m.mu.Unlock()

		// 即使记录显示正在运行，也要去检查一下端口是否真的还活着
		if m.CheckConnectionForPort(port) {
			wailsRuntime.EventsEmit(m.app.ctx, "output-log", fmt.Sprintf("目录 %s 已在运行 (内部记录复用，端口 %d)", dir, port))
			return nil
		}
		// 如果端口已经不通了，说明进程已经意外退出了，需要清理记录并重新启动
		wailsRuntime.EventsEmit(m.app.ctx, "output-log", fmt.Sprintf("目录 %s 的内部记录虽然是运行中，但端口 %d 无法连接，准备重启", dir, port))
		m.mu.Lock()
		delete(m.instances, dir)
		m.mu.Unlock()
	} else {
		m.mu.Unlock()
	}

	// 2. 检查系统真实进程状态：如果指定端口已经被 OpenCode 进程占用，并且是正确的目录，也应该复用而不是杀掉重建
	// 如果检查端口连接成功，说明端口被占用
	if m.CheckConnectionForPort(port) {
		// 这里进一步确认占用该端口的是否是 opencode 进程
		if m.isPortUsedByOpenCode(port) {
			wailsRuntime.EventsEmit(m.app.ctx, "output-log", fmt.Sprintf("发现端口 %d 已有 OpenCode 进程运行，直接复用", port))

			// 补充内部状态（因为可能应用重启了，但底层进程还在）
			m.mu.Lock()
			m.instances[dir] = &OpenCodeInstance{
				cmd:     nil, // 由于不是当前进程启动的，无法拿到 *exec.Cmd 控制权，但标记为运行中
				workDir: dir,
				port:    port,
				running: true,
			}
			if m.currentDir == dir {
				m.app.serverURL = fmt.Sprintf("http://127.0.0.1:%d", port)
			}
			m.mu.Unlock()

			wailsRuntime.EventsEmit(m.app.ctx, "opencode-status", "connected")
			return nil
		}
	}

	// 获取可用端口（内部会自动清理非本程序管理的僵尸进程）
	port = m.getAvailablePort(preferredPort)
	if port != preferredPort {
		wailsRuntime.EventsEmit(m.app.ctx, "output-log", fmt.Sprintf("端口 %d 不可用，自动切换到端口 %d", preferredPort, port))
	}

	installed, path := m.CheckInstalled()
	if !installed {
		return fmt.Errorf("OpenCode 未安装")
	}

	wailsRuntime.EventsEmit(m.app.ctx, "output-log", fmt.Sprintf("启动 OpenCode: %s (端口 %d)", dir, port))
	wailsRuntime.EventsEmit(m.app.ctx, "opencode-status", "starting")

	cmd := exec.Command(path, "serve", "--port", fmt.Sprintf("%d", port))
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
	if m.currentDir == dir {
		m.app.serverURL = fmt.Sprintf("http://127.0.0.1:%d", port)
	}
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
			// 更加严格的日志过滤
			// 1. 移除 ANSI 转义字符以便进行内容判断
			cleanLine := m.stripAnsi(line)

			// 2. 过滤掉 INFO 日志、业务总线日志、服务器访问日志以及健康检查日志
			lowerLine := strings.ToLower(cleanLine)
			isInfo := strings.Contains(lowerLine, "info")
			isBus := strings.Contains(lowerLine, "service=bus")
			isServer := strings.Contains(lowerLine, "service=server")
			isHealth := strings.Contains(lowerLine, "/global/health")

			if !isInfo && !isBus && !isServer && !isHealth {
				wailsRuntime.EventsEmit(m.app.ctx, "output-log", line)
			}
		}
	}
}

// stripAnsi 移除字符串中的 ANSI 转义序列
func (m *OpenCodeManager) stripAnsi(str string) string {
	// 简单的正则匹配 ANSI 转义序列
	// 这里使用一个基础的实现，处理常见的颜色代码等
	var b strings.Builder
	inSeq := false
	for i := 0; i < len(str); i++ {
		if str[i] == '\x1b' {
			inSeq = true
			continue
		}
		if inSeq {
			if (str[i] >= 'a' && str[i] <= 'z') || (str[i] >= 'A' && str[i] <= 'Z') {
				inSeq = false
			}
			continue
		}
		b.WriteByte(str[i])
	}
	return b.String()
}

func (m *OpenCodeManager) waitForReadyOnPort(port int) {
	// 前 20 次每 200ms 检查一次（~4s），接下来 20 次每 500ms（~10s），总计 ~14s
	for i := 0; i < 40; i++ {
		if i < 20 {
			time.Sleep(200 * time.Millisecond)
		} else {
			time.Sleep(500 * time.Millisecond)
		}
		if m.CheckConnectionForPort(port) {
			wailsRuntime.EventsEmit(m.app.ctx, "output-log", fmt.Sprintf("服务就绪 (端口 %d)", port))
			wailsRuntime.EventsEmit(m.app.ctx, "opencode-status", "connected")

			// 一旦就绪，触发事件让前端重新订阅 SSE 和刷新会话
			wailsRuntime.EventsEmit(m.app.ctx, "opencode-reconnected")
			return
		}
	}
	wailsRuntime.EventsEmit(m.app.ctx, "output-log", "连接超时")
	wailsRuntime.EventsEmit(m.app.ctx, "opencode-status", "timeout")
}

// isPortUsedByOpenCode 检查占用端口的进程是否是 OpenCode 进程
func (m *OpenCodeManager) isPortUsedByOpenCode(port int) bool {
	if goruntime.GOOS == "windows" {
		netstatCmd := exec.Command("netstat", "-ano")
		output, err := netstatCmd.Output()
		if err != nil {
			return false
		}
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, fmt.Sprintf(":%d ", port)) && strings.Contains(line, "LISTENING") {
				fields := strings.Fields(line)
				if len(fields) >= 5 {
					pid := fields[len(fields)-1]
					if pid != "0" {
						tasklistCmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %s", pid), "/FO", "CSV", "/NH")
						if taskOutput, taskErr := tasklistCmd.Output(); taskErr == nil {
							taskInfo := strings.ToLower(string(taskOutput))
							// 排除桌面端本身
							if strings.Contains(taskInfo, "opencode-desktop") {
								return false
							}
							return strings.Contains(taskInfo, "opencode")
						}
					}
				}
			}
		}
	} else {
		checkCmd := exec.Command("bash", "-c", fmt.Sprintf("lsof -ti:%d", port))
		if output, err := checkCmd.Output(); err == nil && len(output) > 0 {
			pids := strings.TrimSpace(string(output))
			if pids != "" {
				for _, pid := range strings.Split(pids, "\n") {
					pid = strings.TrimSpace(pid)
					if pid != "" {
						psCmd := exec.Command("ps", "-p", pid, "-o", "command=")
						if psOutput, psErr := psCmd.Output(); psErr == nil {
							commandLine := strings.ToLower(strings.TrimSpace(string(psOutput)))
							if strings.Contains(commandLine, "opencode-desktop") {
								return false
							}
							if strings.Contains(commandLine, "opencode") {
								return true
							}
						}
					}
				}
			}
		}
	}
	return false
}

// cleanupPortProcesses 清理占用指定端口的 OpenCode 进程，如果清理了进程则返回 true
func (m *OpenCodeManager) cleanupPortProcesses(port int) bool {
	currentPID := fmt.Sprintf("%d", os.Getpid())
	killed := false

	if goruntime.GOOS == "windows" {
		// Windows: 使用 netstat 查找占用端口的进程
		netstatCmd := exec.Command("netstat", "-ano")
		output, err := netstatCmd.Output()
		if err != nil {
			return false
		}

		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, fmt.Sprintf(":%d ", port)) && strings.Contains(line, "LISTENING") {
				fields := strings.Fields(line)
				if len(fields) >= 5 {
					pid := fields[len(fields)-1]
					if pid != "0" && pid != currentPID {
						// 检查是否是 OpenCode 进程
						tasklistCmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %s", pid), "/FO", "CSV", "/NH")
						if taskOutput, taskErr := tasklistCmd.Output(); taskErr == nil {
							taskInfo := strings.ToLower(string(taskOutput))
							// 增加对 opencode 进程路径的检查，防止误杀
							if strings.Contains(taskInfo, "opencode-desktop") {
								continue
							}
							if strings.Contains(taskInfo, "opencode-cli") || strings.Contains(taskInfo, "opencode.exe") || strings.Contains(taskInfo, "opencode") {
								wailsRuntime.EventsEmit(m.app.ctx, "output-log", fmt.Sprintf("发现 OpenCode 进程占用端口 %d (PID %s)，强制清理以防泄露...", port, pid))
								// 直接使用强制关闭，确保端口释放
								exec.Command("taskkill", "/F", "/PID", pid).Run()
								killed = true
							}
						}
					}
				}
			}
		}
	} else {
		// Unix: 使用 lsof 查找占用端口的进程
		checkCmd := exec.Command("bash", "-c", fmt.Sprintf("lsof -ti:%d", port))
		if output, err := checkCmd.Output(); err == nil && len(output) > 0 {
			pids := strings.TrimSpace(string(output))
			if pids != "" {
				// 检查是否是 OpenCode 进程
				for _, pid := range strings.Split(pids, "\n") {
					if pid = strings.TrimSpace(pid); pid != "" && pid != currentPID {
						// 检查进程路径或命令行
						psCmd := exec.Command("ps", "-p", pid, "-o", "command=")
						if psOutput, psErr := psCmd.Output(); psErr == nil {
							commandLine := strings.ToLower(strings.TrimSpace(string(psOutput)))
							if strings.Contains(commandLine, "opencode-desktop") {
								continue
							}
							// 检查是否包含 opencode 路径或命令
							if strings.Contains(commandLine, "opencode") {
								wailsRuntime.EventsEmit(m.app.ctx, "output-log", fmt.Sprintf("发现残留 OpenCode 进程 (PID %s)，强制清理...", pid))
								// 直接使用 SIGKILL，确保端口立即释放
								exec.Command("kill", "-9", pid).Run()
								killed = true
							}
						}
					}
				}
			}
		}
	}

	if killed {
		time.Sleep(500 * time.Millisecond)
	}
	return killed
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
			// 优雅关闭：先发送 SIGTERM，等待一段时间后再强制杀死
			wailsRuntime.EventsEmit(m.app.ctx, "output-log", fmt.Sprintf("正在优雅关闭 OpenCode (PID %d)...", inst.cmd.Process.Pid))

			if goruntime.GOOS == "windows" {
				// Windows 上使用 taskkill 进行优雅关闭
				pid := fmt.Sprintf("%d", inst.cmd.Process.Pid)
				// 先尝试优雅关闭 (不使用 /F 强制标志)
				killCmd := exec.Command("taskkill", "/PID", pid)
				if err := killCmd.Run(); err != nil {
					wailsRuntime.EventsEmit(m.app.ctx, "output-log", fmt.Sprintf("优雅关闭失败: %v", err))
				}

				// 等待进程退出，最多等待 5 秒
				done := make(chan error, 1)
				go func() {
					done <- inst.cmd.Wait()
				}()

				select {
				case <-done:
					wailsRuntime.EventsEmit(m.app.ctx, "output-log", "OpenCode 已优雅关闭")
				case <-time.After(5 * time.Second):
					// 超时后强制杀死
					wailsRuntime.EventsEmit(m.app.ctx, "output-log", "优雅关闭超时，强制终止进程")
					exec.Command("taskkill", "/F", "/PID", pid).Run()
					inst.cmd.Wait()
				}
			} else {
				// Unix 系统上先尝试优雅关闭
				inst.cmd.Process.Signal(os.Interrupt) // SIGINT

				// 等待进程优雅退出，最多等待 5 秒
				done := make(chan error, 1)
				go func() {
					done <- inst.cmd.Wait()
				}()

				select {
				case <-done:
					wailsRuntime.EventsEmit(m.app.ctx, "output-log", "OpenCode 已优雅关闭")
				case <-time.After(5 * time.Second):
					// 超时后强制杀死
					wailsRuntime.EventsEmit(m.app.ctx, "output-log", "优雅关闭超时，强制终止进程")
					inst.cmd.Process.Kill()
					inst.cmd.Wait()
				}
			}
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
	wailsRuntime.EventsEmit(m.app.ctx, "output-log", fmt.Sprintf("正在重启目录: %s", dir))

	// 显式停止当前目录的实例，确保重启时会创建新进程
	m.Stop()

	return m.StartForDir(dir)
}
