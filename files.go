package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// FileInfo 文件信息
type FileInfo struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	Type     string      `json:"type"` // "file" or "folder"
	Size     int64       `json:"size"`
	Children []*FileInfo `json:"children,omitempty"`
}

// FileManager 文件管理器
type FileManager struct {
	app          *App
	rootDir      string
	watcher      *fsnotify.Watcher
	watchedFiles map[string]bool
	mu           sync.Mutex
}

// NewFileManager 创建文件管理器
func NewFileManager(app *App) *FileManager {
	homeDir, _ := os.UserHomeDir()
	fm := &FileManager{
		app:          app,
		rootDir:      homeDir,
		watchedFiles: make(map[string]bool),
	}
	return fm
}

// StartWatcher 启动文件监听
func (fm *FileManager) StartWatcher() error {
	if fm.watcher != nil {
		return nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	fm.watcher = watcher

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					// 文件被修改，通知前端
					wailsRuntime.EventsEmit(fm.app.ctx, "file-changed", event.Name)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Println("文件监听错误:", err)
			}
		}
	}()

	return nil
}

// WatchFile 监听指定文件
func (fm *FileManager) WatchFile(path string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if fm.watcher == nil {
		if err := fm.StartWatcher(); err != nil {
			return err
		}
	}

	if fm.watchedFiles[path] {
		return nil
	}

	if err := fm.watcher.Add(path); err != nil {
		return err
	}
	fm.watchedFiles[path] = true
	return nil
}

// UnwatchFile 取消监听文件
func (fm *FileManager) UnwatchFile(path string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if fm.watcher != nil && fm.watchedFiles[path] {
		fm.watcher.Remove(path)
		delete(fm.watchedFiles, path)
	}
}

// SetRootDir 设置根目录
func (fm *FileManager) SetRootDir(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("目录不存在: %v", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("不是目录: %s", dir)
	}
	fm.rootDir = dir
	return nil
}

// GetRootDir 获取根目录
func (fm *FileManager) GetRootDir() string {
	return fm.rootDir
}

// ListDir 列出目录内容
func (fm *FileManager) ListDir(dir string) ([]*FileInfo, error) {
	// 如果是相对路径，基于 rootDir
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(fm.rootDir, dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %v", err)
	}

	var files []*FileInfo
	var folders []*FileInfo

	for _, entry := range entries {
		// 跳过隐藏文件和常见忽略目录
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if name == "node_modules" || name == "__pycache__" || name == "vendor" {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		fileInfo := &FileInfo{
			Name: name,
			Path: filepath.Join(dir, name),
			Size: info.Size(),
		}

		if entry.IsDir() {
			fileInfo.Type = "folder"
			folders = append(folders, fileInfo)
		} else {
			fileInfo.Type = "file"
			files = append(files, fileInfo)
		}
	}

	// 文件夹在前，文件在后，各自按名称排序
	sort.Slice(folders, func(i, j int) bool {
		return strings.ToLower(folders[i].Name) < strings.ToLower(folders[j].Name)
	})
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	return append(folders, files...), nil
}

// ReadFile 读取文件内容
func (fm *FileManager) ReadFile(path string) (string, error) {
	// 如果是相对路径，基于 rootDir
	if !filepath.IsAbs(path) {
		path = filepath.Join(fm.rootDir, path)
	}

	// 检查文件大小，限制 5MB
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("文件不存在: %v", err)
	}
	if info.Size() > 5*1024*1024 {
		return "", fmt.Errorf("文件太大 (>5MB)")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %v", err)
	}

	return string(content), nil
}

// WriteFile 写入文件内容
func (fm *FileManager) WriteFile(path, content string) error {
	// 如果是相对路径，基于 rootDir
	if !filepath.IsAbs(path) {
		path = filepath.Join(fm.rootDir, path)
	}

	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	return nil
}

// GetFileInfo 获取文件信息
func (fm *FileManager) GetFileInfo(path string) (*FileInfo, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(fm.rootDir, path)
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %v", err)
	}

	fileType := "file"
	if info.IsDir() {
		fileType = "folder"
	}

	return &FileInfo{
		Name: info.Name(),
		Path: path,
		Type: fileType,
		Size: info.Size(),
	}, nil
}
