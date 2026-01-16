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
			if ps, ok := p.(string); ok && strings.Contains(ps, pluginName) {
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

	// 也检查 plugins 字段（复数形式，兼容旧配置）
	if plugins, ok := config["plugins"].([]interface{}); ok {
		for _, p := range plugins {
			if ps, ok := p.(string); ok && strings.Contains(ps, pluginName) {
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

// RestartOpenCode 重启 OpenCode 服务
func (a *App) RestartOpenCode() error {
	runtime.EventsEmit(a.ctx, "output-log", "正在重启 OpenCode...")

	// 发送连接断开事件
	runtime.EventsEmit(a.ctx, "opencode-status", "restarting")

	// 先停止当前目录的 OpenCode 实例
	a.openCode.Stop()

	// 等待进程完全退出
	time.Sleep(2 * time.Second)

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
