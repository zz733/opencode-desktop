package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

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
