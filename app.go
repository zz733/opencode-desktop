package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx           context.Context
	serverURL     string
	httpClient    *http.Client
	termMgr       *TerminalManager
	openCode      *OpenCodeManager
	fileMgr       *FileManager
	sseCancel     context.CancelFunc // 用于取消 SSE 订阅
	sseSubscribed bool
}

// NewApp creates a new App application struct
func NewApp() *App {
	app := &App{
		serverURL: "http://localhost:4096",
		httpClient: &http.Client{
			Timeout: 0, // no timeout for SSE
		},
	}
	app.termMgr = NewTerminalManager(app)
	app.openCode = NewOpenCodeManager(app)
	app.fileMgr = NewFileManager(app)
	return app
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// SetServerURL 设置服务器地址
func (a *App) SetServerURL(url string) {
	a.serverURL = strings.TrimSuffix(url, "/")
}

// GetServerURL 获取服务器地址
func (a *App) GetServerURL() string {
	return a.serverURL
}

// CreateTerminal 创建新终端
func (a *App) CreateTerminal() (int, error) {
	return a.termMgr.CreateTerminal()
}

// WriteTerminal 写入终端
func (a *App) WriteTerminal(id int, data string) error {
	return a.termMgr.WriteTerminal(id, data)
}

// ResizeTerminal 调整终端大小
func (a *App) ResizeTerminal(id int, cols, rows int) error {
	return a.termMgr.ResizeTerminal(id, cols, rows)
}

// CloseTerminal 关闭终端
func (a *App) CloseTerminal(id int) {
	a.termMgr.CloseTerminal(id)
}

// GetTerminals 获取所有终端
func (a *App) GetTerminals() []int {
	return a.termMgr.GetTerminals()
}

// --- OpenCode 管理 ---

// GetOpenCodeStatus 获取 OpenCode 状态
func (a *App) GetOpenCodeStatus() *OpenCodeStatus {
	return a.openCode.GetStatus()
}

// InstallOpenCode 安装 OpenCode
func (a *App) InstallOpenCode() error {
	return a.openCode.Install()
}

// StartOpenCode 启动 OpenCode
func (a *App) StartOpenCode() error {
	return a.openCode.Start()
}

// StopOpenCode 停止 OpenCode
func (a *App) StopOpenCode() {
	a.openCode.Stop()
}

// AutoStartOpenCode 自动检测并启动 OpenCode
func (a *App) AutoStartOpenCode() error {
	return a.openCode.AutoStart()
}

// SetOpenCodeWorkDir 设置 OpenCode 工作目录
func (a *App) SetOpenCodeWorkDir(dir string) error {
	a.openCode.SetWorkDir(dir)
	// 同时更新文件管理器的根目录
	a.fileMgr.SetRootDir(dir)
	// 不自动重启，让 autoConnect 处理连接
	return nil
}

// --- 文件管理 ---

// SetWorkDir 设置工作目录
func (a *App) SetWorkDir(dir string) error {
	return a.fileMgr.SetRootDir(dir)
}

// GetWorkDir 获取工作目录
func (a *App) GetWorkDir() string {
	return a.fileMgr.GetRootDir()
}

// ListDir 列出目录内容
func (a *App) ListDir(dir string) ([]*FileInfo, error) {
	return a.fileMgr.ListDir(dir)
}

// ReadFile 读取文件内容
func (a *App) ReadFileContent(path string) (string, error) {
	return a.fileMgr.ReadFile(path)
}

// WriteFile 写入文件内容
func (a *App) WriteFileContent(path, content string) error {
	return a.fileMgr.WriteFile(path, content)
}

// WatchFile 监听文件变化
func (a *App) WatchFile(path string) error {
	return a.fileMgr.WatchFile(path)
}

// UnwatchFile 取消监听文件
func (a *App) UnwatchFile(path string) {
	a.fileMgr.UnwatchFile(path)
}

// RunFile 运行文件
func (a *App) RunFile(filePath string) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("文件路径为空")
	}

	// 获取文件扩展名和目录
	ext := strings.ToLower(filepath.Ext(filePath))
	dir := filepath.Dir(filePath)
	fileName := filepath.Base(filePath)
	fileNameNoExt := strings.TrimSuffix(fileName, ext)

	var cmd string
	var args []string

	switch ext {
	case ".py":
		cmd = "python3"
		args = []string{filePath}
	case ".go":
		cmd = "go"
		args = []string{"run", filePath}
	case ".js":
		cmd = "node"
		args = []string{filePath}
	case ".ts":
		cmd = "npx"
		args = []string{"ts-node", filePath}
	case ".java":
		// 检测是否是 Maven 项目
		if a.isMavenProject(dir) {
			cmd = "mvn"
			args = []string{"-f", a.findPomXml(dir), "compile", "exec:java", "-Dexec.mainClass=" + a.getJavaMainClass(filePath)}
		} else if a.isGradleProject(dir) {
			cmd = "gradle"
			args = []string{"-p", a.findGradleProject(dir), "run"}
		} else {
			// 单文件编译运行
			cmd = "sh"
			args = []string{"-c", fmt.Sprintf("cd %s && javac %s && java %s", dir, fileName, fileNameNoExt)}
		}
	case ".rs":
		cmd = "cargo"
		args = []string{"run", "--manifest-path", a.findCargoToml(dir)}
	case ".rb":
		cmd = "ruby"
		args = []string{filePath}
	case ".php":
		cmd = "php"
		args = []string{filePath}
	case ".sh":
		cmd = "bash"
		args = []string{filePath}
	case ".html", ".htm":
		// 返回特殊标记，前端处理打开浏览器
		return "OPEN_BROWSER:" + filePath, nil
	default:
		return "", fmt.Errorf("不支持运行 %s 文件", ext)
	}

	// 返回命令让前端在终端中执行
	fullCmd := cmd + " " + strings.Join(args, " ")
	return fullCmd, nil
}

// 检测是否是 Maven 项目
func (a *App) isMavenProject(dir string) bool {
	return a.findPomXml(dir) != ""
}

// 查找 pom.xml
func (a *App) findPomXml(dir string) string {
	// 向上查找 pom.xml
	for {
		pomPath := filepath.Join(dir, "pom.xml")
		if _, err := os.Stat(pomPath); err == nil {
			return pomPath
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// 检测是否是 Gradle 项目
func (a *App) isGradleProject(dir string) bool {
	return a.findGradleProject(dir) != ""
}

// 查找 Gradle 项目目录
func (a *App) findGradleProject(dir string) string {
	for {
		if _, err := os.Stat(filepath.Join(dir, "build.gradle")); err == nil {
			return dir
		}
		if _, err := os.Stat(filepath.Join(dir, "build.gradle.kts")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// 查找 Cargo.toml
func (a *App) findCargoToml(dir string) string {
	for {
		cargoPath := filepath.Join(dir, "Cargo.toml")
		if _, err := os.Stat(cargoPath); err == nil {
			return cargoPath
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Join(dir, "Cargo.toml")
}

// 获取 Java 主类名
func (a *App) getJavaMainClass(filePath string) string {
	// 简单实现：从文件路径推断类名
	// 实际应该解析 package 声明
	fileName := filepath.Base(filePath)
	return strings.TrimSuffix(fileName, ".java")
}

// OpenFolder 打开文件夹选择对话框并设置为工作目录
func (a *App) OpenFolder() (string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择工作目录",
	})
	if err != nil {
		return "", err
	}
	if dir != "" {
		// 设置文件管理器根目录
		a.fileMgr.SetRootDir(dir)
		// 设置 OpenCode 工作目录（会自动更新 serverURL）
		a.openCode.SetWorkDir(dir)
		// 更新 app 的 serverURL
		a.serverURL = fmt.Sprintf("http://localhost:%d", a.openCode.GetCurrentPort())
		runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("服务器地址已更新: %s", a.serverURL))
		// 启动该目录的 OpenCode 实例（如果已运行则复用）
		go a.openCode.StartForDir(dir)
	}
	return dir, nil
}

// --- 文件操作（右键菜单） ---

// DeletePath 删除文件或文件夹
func (a *App) DeletePath(path string) error {
	return a.fileMgr.DeletePath(path)
}

// RenamePath 重命名文件或文件夹
func (a *App) RenamePath(oldPath, newName string) (string, error) {
	return a.fileMgr.RenamePath(oldPath, newName)
}

// CopyPath 复制文件或文件夹
func (a *App) CopyPath(src, destDir string) (string, error) {
	return a.fileMgr.CopyPath(src, destDir)
}

// MovePath 移动文件或文件夹
func (a *App) MovePath(src, destDir string) (string, error) {
	return a.fileMgr.MovePath(src, destDir)
}

// CreateNewFile 创建新文件
func (a *App) CreateNewFile(dir, name string) (string, error) {
	return a.fileMgr.CreateFile(dir, name)
}

// CreateNewFolder 创建新文件夹
func (a *App) CreateNewFolder(dir, name string) (string, error) {
	return a.fileMgr.CreateFolder(dir, name)
}

// OpenInFinder 在访达/资源管理器中打开
func (a *App) OpenInFinder(path string) error {
	return openInFileManager(path)
}

// CopyToClipboard 复制文本到剪贴板
func (a *App) CopyToClipboard(text string) error {
	runtime.ClipboardSetText(a.ctx, text)
	return nil
}
