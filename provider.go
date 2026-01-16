package main

import (
	"encoding/json"
	"fmt"
)

// Provider 信息
type Provider struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Model 信息
type Model struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
}

// ProviderInfo 完整的 provider 信息
type ProviderInfo struct {
	All       []Provider        `json:"all"`
	Connected []string          `json:"connected"`
	Default   map[string]string `json:"default"`
}

// ConfigInfo 配置信息
type ConfigInfo struct {
	Model string `json:"model"`
}

// GetProviders 获取所有 provider 和模型信息
func (a *App) GetProviders() (*ProviderInfo, error) {
	resp, err := a.httpClient.Get(a.serverURL + "/provider")
	if err != nil {
		return nil, fmt.Errorf("获取 provider 失败: %v", err)
	}
	defer resp.Body.Close()

	var info ProviderInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

// GetConfig 获取当前配置
func (a *App) GetConfig() (*ConfigInfo, error) {
	resp, err := a.httpClient.Get(a.serverURL + "/config")
	if err != nil {
		return nil, fmt.Errorf("获取配置失败: %v", err)
	}
	defer resp.Body.Close()

	var config ConfigInfo
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, err
	}
	return &config, nil
}
