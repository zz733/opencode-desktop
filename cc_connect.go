package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"syscall"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ccConnectProcesses 存储正在运行的机器人进程
var ccConnectProcesses = struct {
	sync.Mutex
	m map[string]*exec.Cmd
}{m: make(map[string]*exec.Cmd)}

// CCConnectPlatformConfig 表示单个平台的配置
type CCConnectPlatformConfig struct {
	Platform string            `json:"platform"` // dingtalk, wechat, wxwork, qq, feishu 等
	Enabled  bool              `json:"enabled"`
	Config   map[string]string `json:"config"`
}

// CCConnectConfig 存储所有平台的配置
type CCConnectConfig struct {
	Platforms     map[string]CCConnectPlatformConfig `json:"platforms"`
	SelectedModel string                             `json:"selectedModel,omitempty"`
}

// GetCCConnectConfigPath 获取配置文件路径
func (a *App) GetCCConnectConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "cc_connect.json"
	}
	return filepath.Join(homeDir, ".config", "opencode", "cc_connect.json")
}

// LoadCCConnectConfig 加载配置
func (a *App) LoadCCConnectConfig() (CCConnectConfig, error) {
	configPath := a.GetCCConnectConfigPath()
	var config CCConnectConfig
	config.Platforms = make(map[string]CCConnectPlatformConfig)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return config, nil // 返回空配置
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return config, fmt.Errorf("读取配置失败: %v", err)
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return config, fmt.Errorf("解析配置失败: %v", err)
	}

	if config.Platforms == nil {
		config.Platforms = make(map[string]CCConnectPlatformConfig)
	}

	return config, nil
}

// SaveCCConnectConfig 保存配置
func (a *App) SaveCCConnectConfig(config CCConnectConfig) error {
	configPath := a.GetCCConnectConfigPath()
	if strings.TrimSpace(config.SelectedModel) == "" {
		existingConfig, err := a.LoadCCConnectConfig()
		if err == nil {
			config.SelectedModel = existingConfig.SelectedModel
		}
	}

	// 确保目录存在
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %v", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置失败: %v", err)
	}

	return nil
}

// CCConnectStatus 插件状态
type CCConnectStatus struct {
	Installed bool   `json:"installed"`
	Version   string `json:"version"`
}

// GetCCConnectStatus 获取 cc-connect 插件状态
func (a *App) GetCCConnectStatus() *CCConnectStatus {
	// 检查全局是否安装了 cc-connect
	cmd := exec.Command("cc-connect", "--version")
	out, err := cmd.Output()
	if err == nil {
		return &CCConnectStatus{
			Installed: true,
			Version:   string(out),
		}
	}

	// 尝试通过 npx 检查 (有些用户可能没装到全局，但缓存里有)
	// 这里为了准确起见，我们只认全局安装的，这样启动也方便
	return &CCConnectStatus{Installed: false}
}

// InstallCCConnect 安装 cc-connect 插件
func (a *App) InstallCCConnect() error {
	runtime.EventsEmit(a.ctx, "output-log", "开始使用 npm 安装 cc-connect 插件...")

	// 检查 npm 是否存在
	if _, err := exec.LookPath("npm"); err != nil {
		return fmt.Errorf("未找到 npm，请先安装 Node.js 和 npm")
	}

	// 执行 npm install -g cc-connect@beta
	cmd := exec.Command("npm", "install", "-g", "cc-connect@beta")

	// 捕获输出并发送给前端
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("创建标准输出管道失败: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("创建标准错误管道失败: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 npm 安装失败: %v", err)
	}

	// 异步读取输出
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				runtime.EventsEmit(a.ctx, "output-log", string(buf[:n]))
			}
			if err != nil {
				break
			}
		}
	}()
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				runtime.EventsEmit(a.ctx, "output-log", string(buf[:n]))
			}
			if err != nil {
				break
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("安装失败: %v", err)
	}

	runtime.EventsEmit(a.ctx, "output-log", "cc-connect 插件安装成功！")
	return nil
}

// AutoStartCCConnect 自动启动所有已启用的机器人
func (a *App) AutoStartCCConnect() {
	config, err := a.LoadCCConnectConfig()
	if err != nil {
		return
	}

	for platform, pConfig := range config.Platforms {
		if pConfig.Enabled {
			// 异步启动，避免阻塞
			go func(p string) {
				if err := a.StartCCConnectBot(p); err != nil {
					runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("⚠️ 自动启动 %s 失败: %v", p, err))
				}
			}(platform)
		}
	}
}
func (a *App) UninstallCCConnect() error {
	// 停止所有正在运行的机器人
	for platform := range ccConnectProcesses.m {
		a.StopCCConnectBot(platform)
	}

	runtime.EventsEmit(a.ctx, "output-log", "开始卸载 cc-connect 插件...")

	// 检查 npm 是否存在
	if _, err := exec.LookPath("npm"); err != nil {
		return fmt.Errorf("未找到 npm，无法卸载")
	}

	// 执行 npm uninstall -g cc-connect
	cmd := exec.Command("npm", "uninstall", "-g", "cc-connect")

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("卸载失败: %v\n%s", err, string(out))
	}

	runtime.EventsEmit(a.ctx, "output-log", "cc-connect 插件已成功卸载")
	return nil
}

// generateTOML 生成平台的 TOML 配置
func (a *App) generateTOML(platform string, pConfig CCConnectPlatformConfig, selectedModel string) (string, error) {
	homeDir, _ := os.UserHomeDir()
	tomlPath := filepath.Join(homeDir, ".config", "opencode", fmt.Sprintf("cc_connect_%s.toml", platform))

	workDir := a.openCode.GetWorkDir()
	if workDir == "" {
		workDir = homeDir
	}

	// 动态获取当前项目或全局的默认模型
	modelID := strings.TrimSpace(selectedModel)
	var err error
	if modelID == "" {
		modelID, err = a.GetDefaultModel()
	}
	if err != nil || modelID == "" {
		modelID = "opencode/gpt-5.1-codex" // 提供一个安全的默认值
	}

	// 构造 TOML 内容
	tomlContent := fmt.Sprintf(`[[projects]]
name = "opencode-%s"

[projects.agent]
type = "opencode"

[projects.agent.options]
work_dir = "%s"
mode = "yolo"
model = "%s"

[[projects.platforms]]
type = "%s"

[projects.platforms.options]
`, platform, workDir, modelID, platform)

	// 添加用户配置的键值对
	for k, v := range pConfig.Config {
		// 简单处理，如果是字符串类型加引号
		tomlContent += fmt.Sprintf("%s = \"%s\"\n", k, v)
	}

	// 尝试读取额外的全局配置（如 [speech], [tts] 等），如果存在则追加
	extPath := filepath.Join(homeDir, ".config", "opencode", "cc_connect_ext.toml")
	if extData, err := os.ReadFile(extPath); err == nil {
		tomlContent += "\n\n" + string(extData)
	} else {
		// 如果不存在，生成一个模板文件，方便用户后续修改
		defaultExt := `# 在此配置 cc-connect 的额外全局选项，例如语音消息 [speech] 或语音回复 [tts]
# 保存此文件后，在 OpenCode 桌面端重新开启机器人即可生效。
# 
# [speech]
# enabled = true
# provider = "openai"    # 支持 openai, groq, qwen
# 
# [speech.openai]
# api_key = "sk-xxx"
# base_url = "https://api.openai.com/v1"
# model = "whisper-1"
`
		_ = os.WriteFile(extPath, []byte(defaultExt), 0644)
	}

	if err := os.WriteFile(tomlPath, []byte(tomlContent), 0644); err != nil {
		return "", err
	}

	return tomlPath, nil
}

// StartCCConnectBot 启动指定的机器人
func (a *App) StartCCConnectBot(platform string) error {
	ccConnectProcesses.Lock()
	defer ccConnectProcesses.Unlock()

	// 如果已经运行，先停止
	if oldCmd, exists := ccConnectProcesses.m[platform]; exists {
		if oldCmd.Process != nil {
			oldCmd.Process.Kill()
		}
		delete(ccConnectProcesses.m, platform)
	}

	config, err := a.LoadCCConnectConfig()
	if err != nil {
		return err
	}

	pConfig, exists := config.Platforms[platform]
	if !exists || !pConfig.Enabled {
		return fmt.Errorf("平台未配置或未启用")
	}

	// 检查全局是否安装了 cc-connect
	if _, err := exec.LookPath("cc-connect"); err != nil {
		return fmt.Errorf("cc-connect 插件未安装，请先在界面上安装")
	}

	tomlPath, err := a.generateTOML(platform, pConfig, config.SelectedModel)
	if err != nil {
		return fmt.Errorf("生成配置文件失败: %v", err)
	}

	// 启动前先清理可能遗留的旧进程
	if goruntime.GOOS != "windows" {
		killCmd := exec.Command("pkill", "-f", fmt.Sprintf("cc-connect.*%s", filepath.Base(tomlPath)))
		_ = killCmd.Run()
	}

	cmd := exec.Command("cc-connect", "--config", tomlPath)

	// 在后台运行，不等待返回
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动失败: %v", err)
	}

	ccConnectProcesses.m[platform] = cmd

	runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("🤖 成功启动 %s 机器人 (PID: %d)", platform, cmd.Process.Pid))

	// 监控进程退出
	go func(p string, c *exec.Cmd) {
		err := c.Wait()
		ccConnectProcesses.Lock()
		isCurrent := false
		if currentCmd, exists := ccConnectProcesses.m[p]; exists && currentCmd == c {
			delete(ccConnectProcesses.m, p)
			isCurrent = true
		}
		ccConnectProcesses.Unlock()
		if !isCurrent {
			return
		}
		if err != nil {
			runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("⚠️ %s 机器人已退出: %v", p, err))
		} else {
			runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("ℹ️ %s 机器人已停止", p))
		}
	}(platform, cmd)

	return nil
}

func (a *App) SyncCCConnectAgentContext(modelID string) error {
	config, err := a.LoadCCConnectConfig()
	if err != nil {
		return err
	}

	modelID = strings.TrimSpace(modelID)
	if modelID != "" && config.SelectedModel != modelID {
		config.SelectedModel = modelID
		if err := a.SaveCCConnectConfig(config); err != nil {
			return err
		}
	}

	var firstErr error
	restarted := 0
	for platform, pConfig := range config.Platforms {
		if !pConfig.Enabled {
			continue
		}
		if err := a.StartCCConnectBot(platform); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("⚠️ 同步 %s 机器人上下文失败: %v", platform, err))
			continue
		}
		restarted++
	}

	if restarted > 0 {
		if modelID == "" {
			modelID, err = a.GetDefaultModel()
			if err != nil || modelID == "" {
				modelID = "opencode/gpt-5.1-codex"
			}
		}
		runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("已同步 cc-connect 上下文，当前模型: %s", modelID))
	}

	return firstErr
}

// StopCCConnectBot 停止指定的机器人
func (a *App) StopCCConnectBot(platform string) error {
	ccConnectProcesses.Lock()
	defer ccConnectProcesses.Unlock()

	if cmd, exists := ccConnectProcesses.m[platform]; exists {
		if cmd.Process != nil {
			// 在 Windows 上使用 Kill，在 Unix 上也可以用 Kill 或者发信号
			if goruntime.GOOS == "windows" {
				cmd.Process.Kill()
			} else {
				cmd.Process.Signal(syscall.SIGTERM)
			}
		}
		delete(ccConnectProcesses.m, platform)
		runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("已发送停止信号给 %s 机器人", platform))
	}
	return nil
}
