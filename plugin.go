package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// OhMyOpenCodeStatus oh-my-opencode 状态
type OhMyOpenCodeStatus struct {
	Installed bool   `json:"installed"`
	Version   string `json:"version"`
}

// AntigravityAuthStatus antigravity-auth 状态
type AntigravityAuthStatus struct {
	Installed bool   `json:"installed"`
	Version   string `json:"version"`
}

// KiroAuthStatus kiro-auth 状态
type KiroAuthStatus struct {
	Installed bool   `json:"installed"`
	Version   string `json:"version"`
}

// UIUXProMaxStatus ui-ux-pro-max 状态
type UIUXProMaxStatus struct {
	Installed bool   `json:"installed"`
	Version   string `json:"version"`
}

// GetOhMyOpenCodeStatus 获取 oh-my-opencode 状态
func (a *App) GetOhMyOpenCodeStatus() *OhMyOpenCodeStatus {
	status := &OhMyOpenCodeStatus{Installed: false}
	status.Installed, status.Version = a.checkPluginInstalled("oh-my-opencode")
	return status
}

// GetAntigravityAuthStatus 获取 antigravity-auth 状态
func (a *App) GetAntigravityAuthStatus() *AntigravityAuthStatus {
	status := &AntigravityAuthStatus{Installed: false}
	status.Installed, status.Version = a.checkPluginInstalled("opencode-antigravity-auth")
	return status
}

// GetKiroAuthStatus 获取 kiro-auth 状态
func (a *App) GetKiroAuthStatus() *KiroAuthStatus {
	status := &KiroAuthStatus{Installed: false}
	status.Installed, status.Version = a.checkPluginInstalled("opencode-kiro-auth")
	return status
}

// GetUIUXProMaxStatus 获取 ui-ux-pro-max 状态
func (a *App) GetUIUXProMaxStatus() *UIUXProMaxStatus {
	status := &UIUXProMaxStatus{Installed: false}
	
	// 检查 CLI 是否安装
	var cmd *exec.Cmd
	if goruntime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "uipro", "--version")
	} else {
		cmd = exec.Command("uipro", "--version")
	}
	
	if output, err := cmd.Output(); err == nil {
		status.Installed = true
		// 提取版本号
		versionStr := strings.TrimSpace(string(output))
		if strings.Contains(versionStr, "uipro-cli") {
			parts := strings.Fields(versionStr)
			if len(parts) >= 2 {
				status.Version = parts[1]
			}
		}
	}
	
	// 同时检查是否有 steering 文件
	if status.Installed {
		workDir := a.openCode.GetWorkDir()
		if workDir != "" {
			steeringFile := filepath.Join(workDir, ".kiro", "steering", "ui-ux-pro-max.md")
			if _, err := os.Stat(steeringFile); err != nil {
				status.Installed = false // CLI 存在但未初始化
			}
		}
	}
	
	return status
}

// checkPluginInstalled 检查插件是否安装
func (a *App) checkPluginInstalled(pluginName string) (bool, string) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false, ""
	}

	configPath := filepath.Join(homeDir, ".config", "opencode", "opencode.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false, ""
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return false, ""
	}

	// 检查 plugin 数组
	if plugins, ok := config["plugin"].([]interface{}); ok {
		for _, p := range plugins {
			if ps, ok := p.(string); ok {
				// 对于 kiro-auth，检查两种可能的包名
				if pluginName == "opencode-kiro-auth" {
					if strings.Contains(ps, "opencode-kiro-auth") || strings.Contains(ps, "@zhafron/opencode-kiro-auth") {
						version := ""
						if strings.Contains(ps, "@") {
							parts := strings.Split(ps, "@")
							if len(parts) > 1 {
								version = parts[len(parts)-1]
							}
						}
						return true, version
					}
				} else if strings.Contains(ps, pluginName) {
					version := ""
					if strings.Contains(ps, "@") {
						parts := strings.Split(ps, "@")
						if len(parts) > 1 {
							version = parts[len(parts)-1]
						}
					}
					return true, version
				}
			}
		}
	}

	// 也检查 plugins 字段（复数形式，兼容旧配置）
	if plugins, ok := config["plugins"].([]interface{}); ok {
		for _, p := range plugins {
			if ps, ok := p.(string); ok {
				// 对于 kiro-auth，检查两种可能的包名
				if pluginName == "opencode-kiro-auth" {
					if strings.Contains(ps, "opencode-kiro-auth") || strings.Contains(ps, "@zhafron/opencode-kiro-auth") {
						version := ""
						if strings.Contains(ps, "@") {
							parts := strings.Split(ps, "@")
							if len(parts) > 1 {
								version = parts[len(parts)-1]
							}
						}
						return true, version
					}
				} else if strings.Contains(ps, pluginName) {
					version := ""
					if strings.Contains(ps, "@") {
						parts := strings.Split(ps, "@")
						if len(parts) > 1 {
							version = parts[len(parts)-1]
						}
					}
					return true, version
				}
			}
		}
	}

	return false, ""
}

// InstallOhMyOpenCode 安装 oh-my-opencode
func (a *App) InstallOhMyOpenCode() error {
	runtime.EventsEmit(a.ctx, "output-log", "正在安装 oh-my-opencode...")

	// 使用 npx 运行安装程序
	var cmd *exec.Cmd
	if goruntime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "npx", "oh-my-opencode", "--non-interactive")
	} else {
		cmd = exec.Command("npx", "oh-my-opencode", "--non-interactive")
	}
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		// 如果 npx 失败，尝试直接添加到配置
		runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("npx 安装失败，尝试直接配置: %s", string(output)))
		if addErr := a.addPlugin("oh-my-opencode"); addErr != nil {
			return fmt.Errorf("安装失败: %v", addErr)
		}
	}

	runtime.EventsEmit(a.ctx, "output-log", "oh-my-opencode 安装成功")
	return nil
}

// UninstallOhMyOpenCode 卸载 oh-my-opencode
func (a *App) UninstallOhMyOpenCode() error {
	if err := a.removePlugin("oh-my-opencode"); err != nil {
		return err
	}

	// 删除配置文件
	homeDir, _ := os.UserHomeDir()
	ohMyConfigPath := filepath.Join(homeDir, ".config", "opencode", "oh-my-opencode.json")
	os.Remove(ohMyConfigPath)

	runtime.EventsEmit(a.ctx, "output-log", "oh-my-opencode 已卸载")
	return nil
}

// FixOhMyOpenCode 修复 oh-my-opencode 配置（禁用 Google 认证）
func (a *App) FixOhMyOpenCode() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, ".config", "opencode", "oh-my-opencode.json")

	// 创建配置：禁用 Google 认证
	config := map[string]interface{}{
		"google_auth": false,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return err
	}

	runtime.EventsEmit(a.ctx, "output-log", "oh-my-opencode 已修复：禁用 Google 认证")
	return nil
}

// InstallAntigravityAuth 安装 opencode-antigravity-auth
func (a *App) InstallAntigravityAuth() error {
	runtime.EventsEmit(a.ctx, "output-log", "正在安装 opencode-antigravity-auth...")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, ".config", "opencode", "opencode.json")

	// 确保目录存在
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	var config map[string]interface{}

	// 读取现有配置
	data, err := os.ReadFile(configPath)
	if err != nil {
		// 文件不存在，创建新配置
		config = map[string]interface{}{
			"$schema": "https://opencode.ai/config.json",
		}
	} else {
		if err := json.Unmarshal(data, &config); err != nil {
			return err
		}
	}

	// 添加插件 - 使用修复版本（修复了 Gemini 工具格式问题）
	pluginName := "opencode-antigravity-auth-fixed"
	if plugins, ok := config["plugin"].([]interface{}); ok {
		found := false
		for _, p := range plugins {
			if ps, ok := p.(string); ok && strings.Contains(ps, "opencode-antigravity-auth") {
				found = true
				break
			}
		}
		if !found {
			config["plugin"] = append(plugins, pluginName)
		}
	} else {
		config["plugin"] = []interface{}{pluginName}
	}

	// 添加 provider 配置（这是认证所必需的）
	googleModels := map[string]interface{}{
		"antigravity-gemini-3-pro": map[string]interface{}{
			"name":  "Gemini 3 Pro (Antigravity)",
			"limit": map[string]interface{}{"context": 1048576, "output": 65535},
			"modalities": map[string]interface{}{
				"input":  []string{"text", "image", "pdf"},
				"output": []string{"text"},
			},
			"variants": map[string]interface{}{
				"low":  map[string]interface{}{"thinkingLevel": "low"},
				"high": map[string]interface{}{"thinkingLevel": "high"},
			},
		},
		"antigravity-gemini-3-flash": map[string]interface{}{
			"name":  "Gemini 3 Flash (Antigravity)",
			"limit": map[string]interface{}{"context": 1048576, "output": 65536},
			"modalities": map[string]interface{}{
				"input":  []string{"text", "image", "pdf"},
				"output": []string{"text"},
			},
			"variants": map[string]interface{}{
				"minimal": map[string]interface{}{"thinkingLevel": "minimal"},
				"low":     map[string]interface{}{"thinkingLevel": "low"},
				"medium":  map[string]interface{}{"thinkingLevel": "medium"},
				"high":    map[string]interface{}{"thinkingLevel": "high"},
			},
		},
		"antigravity-claude-sonnet-4-5": map[string]interface{}{
			"name":  "Claude Sonnet 4.5 (Antigravity)",
			"limit": map[string]interface{}{"context": 200000, "output": 64000},
			"modalities": map[string]interface{}{
				"input":  []string{"text", "image", "pdf"},
				"output": []string{"text"},
			},
		},
		"antigravity-claude-sonnet-4-5-thinking": map[string]interface{}{
			"name":  "Claude Sonnet 4.5 Thinking (Antigravity)",
			"limit": map[string]interface{}{"context": 200000, "output": 64000},
			"modalities": map[string]interface{}{
				"input":  []string{"text", "image", "pdf"},
				"output": []string{"text"},
			},
			"variants": map[string]interface{}{
				"low": map[string]interface{}{"thinkingConfig": map[string]interface{}{"thinkingBudget": 8192}},
				"max": map[string]interface{}{"thinkingConfig": map[string]interface{}{"thinkingBudget": 32768}},
			},
		},
		"antigravity-claude-opus-4-5-thinking": map[string]interface{}{
			"name":  "Claude Opus 4.5 Thinking (Antigravity)",
			"limit": map[string]interface{}{"context": 200000, "output": 64000},
			"modalities": map[string]interface{}{
				"input":  []string{"text", "image", "pdf"},
				"output": []string{"text"},
			},
			"variants": map[string]interface{}{
				"low": map[string]interface{}{"thinkingConfig": map[string]interface{}{"thinkingBudget": 8192}},
				"max": map[string]interface{}{"thinkingConfig": map[string]interface{}{"thinkingBudget": 32768}},
			},
		},
		"gemini-2.5-flash": map[string]interface{}{
			"name":  "Gemini 2.5 Flash (Gemini CLI)",
			"limit": map[string]interface{}{"context": 1048576, "output": 65536},
			"modalities": map[string]interface{}{
				"input":  []string{"text", "image", "pdf"},
				"output": []string{"text"},
			},
		},
		"gemini-2.5-pro": map[string]interface{}{
			"name":  "Gemini 2.5 Pro (Gemini CLI)",
			"limit": map[string]interface{}{"context": 1048576, "output": 65536},
			"modalities": map[string]interface{}{
				"input":  []string{"text", "image", "pdf"},
				"output": []string{"text"},
			},
		},
	}

	// 设置 provider 配置
	provider, ok := config["provider"].(map[string]interface{})
	if !ok {
		provider = make(map[string]interface{})
	}

	// 设置 google provider
	provider["google"] = map[string]interface{}{
		"models": googleModels,
	}
	config["provider"] = provider

	// 保存配置
	newData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, newData, 0644); err != nil {
		return err
	}

	runtime.EventsEmit(a.ctx, "output-log", "opencode-antigravity-auth 安装成功，配置已写入")
	runtime.EventsEmit(a.ctx, "output-log", "请点击'认证'按钮进行 Google 账号认证")
	return nil
}

// UninstallAntigravityAuth 卸载 opencode-antigravity-auth
func (a *App) UninstallAntigravityAuth() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, ".config", "opencode", "opencode.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	// 从 plugin 中移除
	if plugins, ok := config["plugin"].([]interface{}); ok {
		newPlugins := []interface{}{}
		for _, p := range plugins {
			if ps, ok := p.(string); ok && !strings.Contains(ps, "opencode-antigravity-auth") {
				newPlugins = append(newPlugins, p)
			}
		}
		config["plugin"] = newPlugins
	}

	// 移除 google provider 配置
	if provider, ok := config["provider"].(map[string]interface{}); ok {
		delete(provider, "google")
		if len(provider) == 0 {
			delete(config, "provider")
		} else {
			config["provider"] = provider
		}
	}

	// 保存配置
	newData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, newData, 0644); err != nil {
		return err
	}

	// 删除认证文件
	antigravityAccountsPath := filepath.Join(homeDir, ".config", "opencode", "antigravity-accounts.json")
	os.Remove(antigravityAccountsPath)

	runtime.EventsEmit(a.ctx, "output-log", "opencode-antigravity-auth 已卸载")
	return nil
}

// InstallKiroAuth 安装 opencode-kiro-auth
func (a *App) InstallKiroAuth() error {
	runtime.EventsEmit(a.ctx, "output-log", "正在安装 opencode-kiro-auth...")

	// 首先确保插件已全局安装
	var cmd *exec.Cmd
	if goruntime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "npm", "install", "-g", "@zhafron/opencode-kiro-auth")
	} else {
		cmd = exec.Command("npm", "install", "-g", "@zhafron/opencode-kiro-auth")
	}
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("全局安装插件失败: %s", string(output)))
		return fmt.Errorf("全局安装插件失败: %v", err)
	}
	
	runtime.EventsEmit(a.ctx, "output-log", "插件全局安装成功")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, ".config", "opencode", "opencode.json")

	// 确保目录存在
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	var config map[string]interface{}

	// 读取现有配置
	data, err := os.ReadFile(configPath)
	if err != nil {
		// 文件不存在，创建新配置
		config = map[string]interface{}{
			"$schema": "https://opencode.ai/config.json",
		}
	} else {
		if err := json.Unmarshal(data, &config); err != nil {
			return err
		}
	}

	// 添加插件 - 使用正确的包名
	pluginName := "@zhafron/opencode-kiro-auth"
	if plugins, ok := config["plugin"].([]interface{}); ok {
		found := false
		for _, p := range plugins {
			if ps, ok := p.(string); ok && strings.Contains(ps, "opencode-kiro-auth") {
				found = true
				break
			}
		}
		if !found {
			config["plugin"] = append(plugins, pluginName)
		}
	} else {
		config["plugin"] = []interface{}{pluginName}
	}

	// 添加 provider 配置（Kiro 模型配置）
	kiroModels := map[string]interface{}{
		"claude-opus-4-5": map[string]interface{}{
			"name":  "Claude Opus 4.5",
			"limit": map[string]interface{}{"context": 200000, "output": 64000},
			"modalities": map[string]interface{}{
				"input":  []string{"text", "image"},
				"output": []string{"text"},
			},
		},
		"claude-opus-4-5-thinking": map[string]interface{}{
			"name":  "Claude Opus 4.5 Thinking",
			"limit": map[string]interface{}{"context": 200000, "output": 64000},
			"modalities": map[string]interface{}{
				"input":  []string{"text", "image"},
				"output": []string{"text"},
			},
			"variants": map[string]interface{}{
				"low":    map[string]interface{}{"thinkingConfig": map[string]interface{}{"thinkingBudget": 8192}},
				"medium": map[string]interface{}{"thinkingConfig": map[string]interface{}{"thinkingBudget": 16384}},
				"max":    map[string]interface{}{"thinkingConfig": map[string]interface{}{"thinkingBudget": 32768}},
			},
		},
		"claude-sonnet-4-5": map[string]interface{}{
			"name":  "Claude Sonnet 4.5",
			"limit": map[string]interface{}{"context": 200000, "output": 64000},
			"modalities": map[string]interface{}{
				"input":  []string{"text", "image"},
				"output": []string{"text"},
			},
		},
		"claude-sonnet-4-5-thinking": map[string]interface{}{
			"name":  "Claude Sonnet 4.5 Thinking",
			"limit": map[string]interface{}{"context": 200000, "output": 64000},
			"modalities": map[string]interface{}{
				"input":  []string{"text", "image"},
				"output": []string{"text"},
			},
			"variants": map[string]interface{}{
				"low":    map[string]interface{}{"thinkingConfig": map[string]interface{}{"thinkingBudget": 8192}},
				"medium": map[string]interface{}{"thinkingConfig": map[string]interface{}{"thinkingBudget": 16384}},
				"max":    map[string]interface{}{"thinkingConfig": map[string]interface{}{"thinkingBudget": 32768}},
			},
		},
		"claude-haiku-4-5": map[string]interface{}{
			"name":  "Claude Haiku 4.5",
			"limit": map[string]interface{}{"context": 200000, "output": 64000},
			"modalities": map[string]interface{}{
				"input":  []string{"text", "image"},
				"output": []string{"text"},
			},
		},
	}

	// 设置 provider 配置
	provider, ok := config["provider"].(map[string]interface{})
	if !ok {
		provider = make(map[string]interface{})
	}

	// 设置 kiro provider
	provider["kiro"] = map[string]interface{}{
		"models": kiroModels,
	}
	config["provider"] = provider

	// 保存配置
	newData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, newData, 0644); err != nil {
		return err
	}

	runtime.EventsEmit(a.ctx, "output-log", "opencode-kiro-auth 安装成功，配置已写入")
	runtime.EventsEmit(a.ctx, "output-log", "重要提示：认证时浏览器可能不会自动打开")
	runtime.EventsEmit(a.ctx, "output-log", "如果浏览器没有自动打开，请手动访问显示的 URL 完成认证")
	return nil
}

// UninstallKiroAuth 卸载 opencode-kiro-auth
func (a *App) UninstallKiroAuth() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, ".config", "opencode", "opencode.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	// 从 plugin 中移除
	if plugins, ok := config["plugin"].([]interface{}); ok {
		newPlugins := []interface{}{}
		for _, p := range plugins {
			if ps, ok := p.(string); ok && !strings.Contains(ps, "opencode-kiro-auth") {
				newPlugins = append(newPlugins, p)
			}
		}
		config["plugin"] = newPlugins
	}

	// 移除 kiro provider 配置
	if provider, ok := config["provider"].(map[string]interface{}); ok {
		delete(provider, "kiro")
		if len(provider) == 0 {
			delete(config, "provider")
		} else {
			config["provider"] = provider
		}
	}

	// 保存配置
	newData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, newData, 0644); err != nil {
		return err
	}

	// 删除认证文件
	kiroConfigPath := filepath.Join(homeDir, ".config", "opencode", "kiro.json")
	os.Remove(kiroConfigPath)

	runtime.EventsEmit(a.ctx, "output-log", "opencode-kiro-auth 已卸载")
	return nil
}
// AuthenticateKiro 认证 Kiro Auth - 简化版本，只提供指导
func (a *App) AuthenticateKiro() error {
	runtime.EventsEmit(a.ctx, "output-log", "请在终端中运行以下命令进行 Kiro Auth 认证：")
	runtime.EventsEmit(a.ctx, "output-log", "")
	runtime.EventsEmit(a.ctx, "output-log", "1. 运行命令: opencode auth login")
	runtime.EventsEmit(a.ctx, "output-log", "2. 选择 'Other' 选项")
	runtime.EventsEmit(a.ctx, "output-log", "3. 输入 'kiro' 作为 provider")
	runtime.EventsEmit(a.ctx, "output-log", "4. 在浏览器中完成 AWS Builder ID 认证")
	runtime.EventsEmit(a.ctx, "output-log", "5. 认证完成后重启应用以刷新模型列表")
	runtime.EventsEmit(a.ctx, "output-log", "")
	runtime.EventsEmit(a.ctx, "output-log", "注意：如果浏览器没有自动打开，请手动访问显示的 URL")
	
	return nil
}

// InstallUIUXProMax 安装 UI/UX Pro Max Skill
func (a *App) InstallUIUXProMax() error {
	runtime.EventsEmit(a.ctx, "output-log", "正在安装 UI/UX Pro Max Skill...")
	
	// 1. 检查 Node.js 和 npm 是否可用
	var cmd *exec.Cmd
	if goruntime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "npm", "--version")
	} else {
		cmd = exec.Command("npm", "--version")
	}
	
	if _, err := cmd.Output(); err != nil {
		runtime.EventsEmit(a.ctx, "output-log", "❌ 未找到 npm，请先安装 Node.js")
		return fmt.Errorf("npm 未安装")
	}
	
	runtime.EventsEmit(a.ctx, "output-log", "✅ 检测到 npm，开始安装 CLI...")
	
	// 2. 安装 uipro-cli
	if goruntime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "npm", "install", "-g", "uipro-cli")
	} else {
		cmd = exec.Command("npm", "install", "-g", "uipro-cli")
	}
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("❌ CLI 安装失败: %s", string(output)))
		return fmt.Errorf("CLI 安装失败: %v", err)
	}
	
	runtime.EventsEmit(a.ctx, "output-log", "✅ CLI 安装成功，正在初始化配置...")
	
	// 3. 初始化 Kiro 配置
	if goruntime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "uipro", "init", "--ai", "kiro")
	} else {
		cmd = exec.Command("uipro", "init", "--ai", "kiro")
	}
	workDir := a.openCode.GetWorkDir()
	if workDir != "" {
		cmd.Dir = workDir
	}
	
	output, err = cmd.CombinedOutput()
	if err != nil {
		runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("❌ 配置初始化失败: %s", string(output)))
		return fmt.Errorf("配置初始化失败: %v", err)
	}
	
	runtime.EventsEmit(a.ctx, "output-log", "✅ UI/UX Pro Max Skill 安装成功！")
	runtime.EventsEmit(a.ctx, "output-log", "现在您可以在聊天中使用 UI/UX 设计功能了")
	runtime.EventsEmit(a.ctx, "output-log", "例如：'帮我设计一个现代化的登录页面'")
	
	return nil
}

// UninstallUIUXProMax 卸载 UI/UX Pro Max Skill
func (a *App) UninstallUIUXProMax() error {
	runtime.EventsEmit(a.ctx, "output-log", "正在卸载 UI/UX Pro Max Skill...")
	
	workDir := a.openCode.GetWorkDir()
	if workDir == "" {
		runtime.EventsEmit(a.ctx, "output-log", "⚠️ 无法获取工作目录")
		return fmt.Errorf("无法获取工作目录")
	}
	
	// 1. 删除本地 steering 文件
	steeringDir := filepath.Join(workDir, ".kiro", "steering")
	steeringFile := filepath.Join(steeringDir, "ui-ux-pro-max.md")
	if err := os.Remove(steeringFile); err != nil && !os.IsNotExist(err) {
		runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("⚠️ 删除 steering 文件失败: %v", err))
	}
	
	// 2. 删除共享资源目录
	sharedDir := filepath.Join(workDir, ".shared", "ui-ux-pro-max")
	if err := os.RemoveAll(sharedDir); err != nil && !os.IsNotExist(err) {
		runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("⚠️ 删除共享资源失败: %v", err))
	}
	
	// 3. 可选：卸载全局 CLI（询问用户）
	runtime.EventsEmit(a.ctx, "output-log", "✅ 本地配置已清理")
	runtime.EventsEmit(a.ctx, "output-log", "注意：全局 CLI 工具仍然保留，如需完全卸载请运行:")
	runtime.EventsEmit(a.ctx, "output-log", "npm uninstall -g uipro-cli")
	
	return nil
}

func (a *App) RestartOpenCode() error {
	runtime.EventsEmit(a.ctx, "output-log", "正在重启 OpenCode...")

	// 发送连接断开事件
	runtime.EventsEmit(a.ctx, "opencode-status", "restarting")

	// 优雅停止当前目录的 OpenCode 实例
	a.openCode.Stop()

	// 等待进程完全退出和端口释放
	time.Sleep(3 * time.Second)

	// 重新启动
	if err := a.openCode.Start(); err != nil {
		runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("重启失败: %v", err))
		runtime.EventsEmit(a.ctx, "opencode-status", "error")
		return err
	}

	runtime.EventsEmit(a.ctx, "output-log", "OpenCode 正在启动，请等待连接...")
	return nil
}

// addPlugin 添加插件到配置
func (a *App) addPlugin(pluginName string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, ".config", "opencode", "opencode.json")

	// 确保目录存在
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	var config map[string]interface{}

	// 读取现有配置
	data, err := os.ReadFile(configPath)
	if err != nil {
		// 文件不存在，创建新配置
		config = map[string]interface{}{
			"$schema": "https://opencode.ai/config.json",
			"plugin":  []interface{}{pluginName},
		}
	} else {
		if err := json.Unmarshal(data, &config); err != nil {
			return err
		}

		// 添加到 plugin 数组
		if plugins, ok := config["plugin"].([]interface{}); ok {
			// 检查是否已存在
			for _, p := range plugins {
				if ps, ok := p.(string); ok && strings.Contains(ps, strings.Split(pluginName, "@")[0]) {
					return nil // 已安装
				}
			}
			config["plugin"] = append(plugins, pluginName)
		} else {
			config["plugin"] = []interface{}{pluginName}
		}
	}

	// 保存配置
	newData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, newData, 0644)
}

// removePlugin 从配置中移除插件
func (a *App) removePlugin(pluginName string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, ".config", "opencode", "opencode.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	// 从 plugin 中移除
	if plugins, ok := config["plugin"].([]interface{}); ok {
		newPlugins := []interface{}{}
		for _, p := range plugins {
			if ps, ok := p.(string); ok && !strings.Contains(ps, pluginName) {
				newPlugins = append(newPlugins, p)
			}
		}
		config["plugin"] = newPlugins
	}

	// 也检查 plugins 字段（兼容旧配置）
	if plugins, ok := config["plugins"].([]interface{}); ok {
		newPlugins := []interface{}{}
		for _, p := range plugins {
			if ps, ok := p.(string); ok && !strings.Contains(ps, pluginName) {
				newPlugins = append(newPlugins, p)
			}
		}
		config["plugins"] = newPlugins
	}

	// 保存配置
	newData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, newData, 0644)
}
