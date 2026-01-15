//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

// setupHiddenProcess 设置进程为后台运行（Unix/macOS）
func (m *OpenCodeManager) setupHiddenProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // 创建新会话，脱离终端
	}
}
