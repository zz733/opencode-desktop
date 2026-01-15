package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"

	"github.com/creack/pty"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// TerminalInstance 单个终端实例
type TerminalInstance struct {
	ID     int
	ptmx   *os.File
	cmd    *exec.Cmd
	active bool
}

// TerminalManager 终端管理器（支持多终端）
type TerminalManager struct {
	app       *App
	terminals map[int]*TerminalInstance
	mu        sync.Mutex
	nextID    int32
}

// NewTerminalManager 创建终端管理器
func NewTerminalManager(app *App) *TerminalManager {
	return &TerminalManager{
		app:       app,
		terminals: make(map[int]*TerminalInstance),
	}
}

// CreateTerminal 创建新终端，返回终端ID
func (tm *TerminalManager) CreateTerminal() (int, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	id := int(atomic.AddInt32(&tm.nextID, 1))

	// 获取默认 shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}

	// 创建命令
	cmd := exec.Command(shell)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	// 启动 PTY
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return 0, err
	}

	instance := &TerminalInstance{
		ID:     id,
		ptmx:   ptmx,
		cmd:    cmd,
		active: true,
	}
	tm.terminals[id] = instance

	// 读取输出并发送到前端
	go tm.readOutput(instance)

	return id, nil
}

// readOutput 读取终端输出
func (tm *TerminalManager) readOutput(inst *TerminalInstance) {
	buf := make([]byte, 4096)
	for {
		n, err := inst.ptmx.Read(buf)
		if err != nil {
			if err != io.EOF {
				runtime.EventsEmit(tm.app.ctx, fmt.Sprintf("terminal-error-%d", inst.ID), err.Error())
			}
			break
		}
		if n > 0 {
			runtime.EventsEmit(tm.app.ctx, fmt.Sprintf("terminal-output-%d", inst.ID), string(buf[:n]))
		}
	}
	tm.mu.Lock()
	inst.active = false
	tm.mu.Unlock()
}

// WriteTerminal 写入指定终端
func (tm *TerminalManager) WriteTerminal(id int, data string) error {
	tm.mu.Lock()
	inst, ok := tm.terminals[id]
	tm.mu.Unlock()

	if !ok || !inst.active || inst.ptmx == nil {
		return fmt.Errorf("terminal %d not found or inactive", id)
	}

	_, err := inst.ptmx.Write([]byte(data))
	return err
}

// ResizeTerminal 调整指定终端大小
func (tm *TerminalManager) ResizeTerminal(id int, cols, rows int) error {
	tm.mu.Lock()
	inst, ok := tm.terminals[id]
	tm.mu.Unlock()

	if !ok || !inst.active || inst.ptmx == nil {
		return nil
	}

	return pty.Setsize(inst.ptmx, &pty.Winsize{
		Cols: uint16(cols),
		Rows: uint16(rows),
	})
}

// CloseTerminal 关闭指定终端
func (tm *TerminalManager) CloseTerminal(id int) {
	tm.mu.Lock()
	inst, ok := tm.terminals[id]
	if ok {
		delete(tm.terminals, id)
	}
	tm.mu.Unlock()

	if ok && inst != nil {
		if inst.ptmx != nil {
			inst.ptmx.Close()
		}
		if inst.cmd != nil && inst.cmd.Process != nil {
			inst.cmd.Process.Kill()
		}
	}
}

// GetTerminals 获取所有终端ID列表
func (tm *TerminalManager) GetTerminals() []int {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	ids := make([]int, 0, len(tm.terminals))
	for id, inst := range tm.terminals {
		if inst.active {
			ids = append(ids, id)
		}
	}
	return ids
}

// CloseAll 关闭所有终端
func (tm *TerminalManager) CloseAll() {
	tm.mu.Lock()
	terminals := make([]*TerminalInstance, 0, len(tm.terminals))
	for _, inst := range tm.terminals {
		terminals = append(terminals, inst)
	}
	tm.terminals = make(map[int]*TerminalInstance)
	tm.mu.Unlock()

	for _, inst := range terminals {
		if inst.ptmx != nil {
			inst.ptmx.Close()
		}
		if inst.cmd != nil && inst.cmd.Process != nil {
			inst.cmd.Process.Kill()
		}
	}
}
