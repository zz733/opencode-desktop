package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ProviderModel OpenCode API 返回的模型信息
type ProviderModel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ProviderResponse OpenCode /provider API 返回的单个 provider
type ProviderResponse struct {
	ID     string                   `json:"id"`
	Name   string                   `json:"name"`
	Models map[string]ProviderModel `json:"models"`
}

// GetAllModels 从 OpenCode API 获取所有模型列表
// 只返回 Kiro 模型和 Antigravity/Gemini 模型，与桌面端保持一致
func (a *App) GetAllModels() ([]ConfigModel, error) {
	if cliModels, err := a.getModelsFromCLI(); err == nil && len(cliModels) > 0 {
		return cliModels, nil
	}

	var models []ConfigModel

	// 调用 OpenCode /provider API
	resp, err := a.apiClient.Get(a.serverURL + "/provider")
	if err != nil {
		fmt.Printf("❌ 获取 provider 失败: %v\n", err)
		// 降级到配置文件
		return a.GetConfigModels()
	}
	defer resp.Body.Close()

	// 先读取响应体用于调试
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ 读取响应失败: %v\n", err)
		return a.GetConfigModels()
	}

	// 解析返回的 JSON
	var providerResp struct {
		All []struct {
			ID     string                 `json:"id"`
			Name   string                 `json:"name"`
			Models map[string]interface{} `json:"models"`
		} `json:"all"`
		Connected []string `json:"connected"`
	}

	if err := json.Unmarshal(body, &providerResp); err != nil {
		fmt.Printf("❌ 解析 provider 响应失败: %v\n", err)
		return a.GetConfigModels()
	}

	fmt.Printf("📋 从 OpenCode API 获取到 %d 个 provider\n", len(providerResp.All))

	// 建立 connected 集合以便快速查找
	connectedMap := make(map[string]bool)
	for _, c := range providerResp.Connected {
		connectedMap[c] = true
	}
	configuredMap := a.getConfiguredProviderMap()

	// 遍历每个 provider，添加「已连接」或「已在配置文件中声明」的提供商模型
	for _, provider := range providerResp.All {
		if provider.Models == nil {
			continue
		}
		if !connectedMap[provider.ID] && !configuredMap[provider.ID] {
			continue
		}

		fmt.Printf("  - Provider: %s (包含 %d 个模型)\n", provider.ID, len(provider.Models))

		for modelID, modelData := range provider.Models {
			// 获取模型名称
			modelName := modelID
			isFree := false

			if modelMap, ok := modelData.(map[string]interface{}); ok {
				if name, ok := modelMap["name"].(string); ok && name != "" {
					modelName = name
				}

				// 判断是否免费
				if costObj, ok := modelMap["cost"].(map[string]interface{}); ok {
					inputCost, _ := costObj["input"].(float64)
					outputCost, _ := costObj["output"].(float64)
					if inputCost == 0 && outputCost == 0 {
						isFree = true
					}
				}
			}

			models = append(models, ConfigModel{
				ID:       fmt.Sprintf("%s/%s", provider.ID, modelID),
				Name:     modelName,
				Provider: provider.ID,
				Free:     isFree,
			})
		}
	}

	fmt.Printf("✅ 筛选后返回 %d 个模型\n", len(models))

	// 如果没有获取到任何模型，降级到配置文件
	if len(models) == 0 {
		fmt.Printf("⚠️ API 未返回符合条件的模型，降级到配置文件\n")
		return a.GetConfigModels()
	}

	return models, nil
}

func (a *App) getModelsFromCLI() ([]ConfigModel, error) {
	workDir := strings.TrimSpace(a.openCode.GetWorkDir())
	if workDir == "" {
		workDir, _ = os.Getwd()
	}

	cmd := exec.Command("opencode", "models")
	if workDir != "" {
		cmd.Dir = workDir
	}
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	seen := make(map[string]bool)
	models := make([]ConfigModel, 0, len(lines))
	for _, line := range lines {
		id := strings.TrimSpace(line)
		if id == "" || !strings.Contains(id, "/") {
			continue
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		parts := strings.SplitN(id, "/", 2)
		providerID := parts[0]
		modelName := parts[1]
		models = append(models, ConfigModel{
			ID:       id,
			Name:     modelName,
			Provider: providerID,
		})
	}
	return models, nil
}

func (a *App) getConfiguredProviderMap() map[string]bool {
	result := make(map[string]bool)
	homeDir, err := os.UserHomeDir()
	if err == nil {
		a.collectProvidersFromConfig(filepath.Join(homeDir, ".config", "opencode", "opencode.json"), result)
	}
	workDir := strings.TrimSpace(a.openCode.GetWorkDir())
	if workDir != "" {
		a.collectProvidersFromConfig(filepath.Join(workDir, "opencode.json"), result)
	}
	return result
}

func (a *App) collectProvidersFromConfig(configPath string, target map[string]bool) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return
	}
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return
	}
	provider, ok := config["provider"].(map[string]interface{})
	if !ok {
		return
	}
	for providerID := range provider {
		target[providerID] = true
	}
}

// GetDefaultModel 从 opencode.json 读取默认配置的模型，优先使用项目配置
func (a *App) GetDefaultModel() (string, error) {
	// 1. 先读取项目级配置 {workDir}/opencode.json
	workDir := a.openCode.GetWorkDir()
	if workDir != "" {
		projectConfigPath := filepath.Join(workDir, "opencode.json")
		if defaultModel, err := a.readDefaultModelFromConfig(projectConfigPath); err == nil && defaultModel != "" {
			return defaultModel, nil
		}
	}

	// 2. 再读取用户级配置 ~/.config/opencode/opencode.json
	homeDir, err := os.UserHomeDir()
	if err == nil {
		userConfigPath := filepath.Join(homeDir, ".config", "opencode", "opencode.json")
		if defaultModel, err := a.readDefaultModelFromConfig(userConfigPath); err == nil && defaultModel != "" {
			return defaultModel, nil
		}
	}

	return "", nil
}

// readDefaultModelFromConfig 从单个配置文件读取默认模型
func (a *App) readDefaultModelFromConfig(configPath string) (string, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return "", err
	}

	if model, ok := config["model"].(string); ok {
		return model, nil
	}

	return "", nil
}

// ConfigModel 配置文件中的模型信息
type ConfigModel struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Provider   string `json:"provider"`
	ContextLen int    `json:"contextLen,omitempty"`
	OutputLen  int    `json:"outputLen,omitempty"`
	Free       bool   `json:"free,omitempty"`
}

// GetConfigModels 从 opencode.json 配置文件读取模型列表
func (a *App) GetConfigModels() ([]ConfigModel, error) {
	var models []ConfigModel

	// 1. 先读取用户级配置 ~/.config/opencode/opencode.json
	homeDir, err := os.UserHomeDir()
	if err == nil {
		userConfigPath := filepath.Join(homeDir, ".config", "opencode", "opencode.json")
		if userModels, err := a.readModelsFromConfig(userConfigPath); err == nil {
			models = append(models, userModels...)
		}
	}

	// 2. 再读取项目级配置 {workDir}/opencode.json
	workDir := a.openCode.GetWorkDir()
	if workDir != "" {
		projectConfigPath := filepath.Join(workDir, "opencode.json")
		if projectModels, err := a.readModelsFromConfig(projectConfigPath); err == nil {
			// 项目配置优先，去重
			for _, pm := range projectModels {
				found := false
				for i, m := range models {
					if m.ID == pm.ID {
						models[i] = pm // 覆盖
						found = true
						break
					}
				}
				if !found {
					models = append(models, pm)
				}
			}
		}
	}

	runtime.EventsEmit(a.ctx, "output-log", fmt.Sprintf("从配置文件读取到 %d 个模型", len(models)))
	return models, nil
}

// readModelsFromConfig 从单个配置文件读取模型
func (a *App) readModelsFromConfig(configPath string) ([]ConfigModel, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	var models []ConfigModel

	// 解析 provider 配置
	provider, ok := config["provider"].(map[string]interface{})
	if !ok {
		return models, nil
	}

	for providerID, providerConfig := range provider {
		pc, ok := providerConfig.(map[string]interface{})
		if !ok {
			continue
		}

		modelsConfig, ok := pc["models"].(map[string]interface{})
		if !ok {
			continue
		}

		for modelID, modelConfig := range modelsConfig {
			mc, ok := modelConfig.(map[string]interface{})
			if !ok {
				continue
			}

			model := ConfigModel{
				ID:       fmt.Sprintf("%s/%s", providerID, modelID),
				Provider: providerID,
			}

			if name, ok := mc["name"].(string); ok {
				model.Name = name
			} else {
				model.Name = modelID
			}

			// 解析 limit
			if limit, ok := mc["limit"].(map[string]interface{}); ok {
				if ctx, ok := limit["context"].(float64); ok {
					model.ContextLen = int(ctx)
				}
				if out, ok := limit["output"].(float64); ok {
					model.OutputLen = int(out)
				}
			}

			models = append(models, model)
		}
	}

	return models, nil
}
