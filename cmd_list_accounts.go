// +build ignore

package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, "Library", "Application Support", "OpenCode", "KiroAccountManager", "data")
	
	fmt.Println("数据目录:", dataDir)
	fmt.Println()
	
	// 列出目录内容
	files, _ := os.ReadDir(dataDir)
	fmt.Println("目录内容:")
	for _, f := range files {
		info, _ := f.Info()
		fmt.Printf("  %s (%d bytes)\n", f.Name(), info.Size())
	}
}
