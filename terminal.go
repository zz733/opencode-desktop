package main

import (
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Terminal 终端管理
type Terminal struct {
	app    *App
	ptmx   *os.File
	cmd    *exec.Cmd
	mu     sync.Mutex
	active bool
}

// NewTerminal 创建终端
func NewTerminal(app *App) *Terminal {
	return &Terminal{app: app}
}

// Start 启动终端
func (t *Terminal) Start() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.active {
		return nil
	}

	// 获取默认 shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}

	// 创建命令
	t.cmd = exec.Command(shell)
	t.cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	// 启动 PTY
	var err error
	t.ptmx, err = pty.Start(t.cmd)
	if err != nil {
		return err
	}

	t.active = true

	// 读取输出并发送到前端
	go t.readOutput()

	return nil
}

// readOutput 读取终端输出
func (t *Terminal) readOutput() {
	buf := make([]byte, 4096)
	for {
		n, err := t.ptmx.Read(buf)
		if err != nil {
			if err != io.EOF {
				runtime.EventsEmit(t.app.ctx, "terminal-error", err.Error())
			}
			break
		}
		if n > 0 {
			runtime.EventsEmit(t.app.ctx, "terminal-output", string(buf[:n]))
		}
	}
	t.active = false
}

// Write 写入终端
func (t *Terminal) Write(data string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.active || t.ptmx == nil {
		return nil
	}

	_, err := t.ptmx.Write([]byte(data))
	return err
}

// Resize 调整终端大小
func (t *Terminal) Resize(cols, rows int) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.active || t.ptmx == nil {
		return nil
	}

	return pty.Setsize(t.ptmx, &pty.Winsize{
		Cols: uint16(cols),
		Rows: uint16(rows),
	})
}

// Stop 停止终端
func (t *Terminal) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.ptmx != nil {
		t.ptmx.Close()
	}
	if t.cmd != nil && t.cmd.Process != nil {
		t.cmd.Process.Kill()
	}
	t.active = false
}

// IsActive 检查终端是否活跃
func (t *Terminal) IsActive() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.active
}
