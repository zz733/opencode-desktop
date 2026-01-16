//go:build !windows

package main

import (
	"os/exec"
	"runtime"
)

// openInFileManager 在文件管理器中打开路径（macOS/Linux）
func openInFileManager(path string) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		// macOS: 使用 open -R 在 Finder 中显示并选中文件
		cmd = exec.Command("open", "-R", path)
	} else {
		// Linux: 使用 xdg-open 打开文件所在目录
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}
