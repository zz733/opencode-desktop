//go:build windows

package main

import (
	"os/exec"
	"path/filepath"
)

// openInFileManager 在资源管理器中打开路径（Windows）
func openInFileManager(path string) error {
	// Windows: 使用 explorer /select, 在资源管理器中显示并选中文件
	absPath, _ := filepath.Abs(path)
	cmd := exec.Command("explorer", "/select,", absPath)
	return cmd.Start()
}
