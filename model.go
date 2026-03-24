package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

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

	// 遍历每个 provider，只添加已连接的提供商的模型
	for _, provider := range providerResp.All {
		if provider.Models == nil || !connectedMap[provider.ID] {
			continue
		}

		fmt.Printf("  - Provider: %s (包含 %d 个模型)\n", provider.ID, len(provider.Models))

		for modelID, modelData := range provider.Models {
			// 获取模型名称
			modelName := modelID
			if modelMap, ok := modelData.(map[string]interface{}); ok {
				if name, ok := modelMap["name"].(string); ok && name != "" {
					modelName = name
				}
			}

			models = append(models, ConfigModel{
				ID:       fmt.Sprintf("%s/%s", provider.ID, modelID),
				Name:     modelName,
				Provider: provider.ID,
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

// ConfigModel 配置文件中的模型信息
type ConfigModel struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Provider   string `json:"provider"`
	ContextLen int    `json:"contextLen,omitempty"`
	OutputLen  int    `json:"outputLen,omitempty"`
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
