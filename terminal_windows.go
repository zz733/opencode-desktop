//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"

	"github.com/UserExistsError/conpty"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// TerminalInstance 单个终端实例
type TerminalInstance struct {
	ID     int
	cpty   *conpty.ConPty
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

	// Windows 使用 PowerShell
	shell := os.Getenv("COMSPEC")
	if shell == "" {
		shell = "cmd.exe"
	}
	// 优先使用 PowerShell
	if _, err := exec.LookPath("powershell.exe"); err == nil {
		shell = "powershell.exe"
	}

	// 使用 ConPTY 创建伪终端
	cpty, err := conpty.Start(shell, conpty.ConPtyDimensions(120, 30))
	if err != nil {
		return 0, fmt.Errorf("failed to start conpty: %v", err)
	}

	instance := &TerminalInstance{
		ID:     id,
		cpty:   cpty,
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
	for inst.active {
		n, err := inst.cpty.Read(buf)
		if err != nil {
			break
		}
		if n > 0 {
			runtime.EventsEmit(tm.app.ctx, fmt.Sprintf("terminal-output-%d", inst.ID), string(buf[:n]))
		}
	}
}

// WriteTerminal 写入指定终端
func (tm *TerminalManager) WriteTerminal(id int, data string) error {
	tm.mu.Lock()
	inst, ok := tm.terminals[id]
	tm.mu.Unlock()

	if !ok || !inst.active || inst.cpty == nil {
		return fmt.Errorf("terminal %d not found or inactive", id)
	}

	_, err := inst.cpty.Write([]byte(data))
	return err
}

// ResizeTerminal 调整指定终端大小
func (tm *TerminalManager) ResizeTerminal(id int, cols, rows int) error {
	tm.mu.Lock()
	inst, ok := tm.terminals[id]
	tm.mu.Unlock()

	if !ok || !inst.active || inst.cpty == nil {
		return nil
	}

	return inst.cpty.Resize(cols, rows)
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
		inst.active = false
		if inst.cpty != nil {
			inst.cpty.Close()
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
		inst.active = false
		if inst.cpty != nil {
			inst.cpty.Close()
		}
	}
}
