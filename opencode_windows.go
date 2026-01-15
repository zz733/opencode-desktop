//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

// setupHiddenProcess 设置进程为隐藏运行（Windows）
func (m *OpenCodeManager) setupHiddenProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}
