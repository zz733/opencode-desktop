package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// OpenCodeKiroAccount represents an account in kiro-accounts.json
type OpenCodeKiroAccount struct {
	ID                 string `json:"id"`
	Email              string `json:"email"`
	AuthMethod         string `json:"authMethod"`
	Region             string `json:"region"`
	ClientID           string `json:"clientId,omitempty"`
	ClientSecret       string `json:"clientSecret,omitempty"`
	RefreshToken       string `json:"refreshToken"`
	AccessToken        string `json:"accessToken"`
	ExpiresAt          int64  `json:"expiresAt"`
	RateLimitResetTime int64  `json:"rateLimitResetTime"`
	IsHealthy          bool   `json:"isHealthy"`
	RealEmail          string `json:"realEmail,omitempty"`
	UserID             string `json:"userId,omitempty"`
	ProfileArn         string `json:"profileArn,omitempty"`
}

// OpenCodeKiroAccountsFile represents the kiro-accounts.json file structure
type OpenCodeKiroAccountsFile struct {
	Version     int                   `json:"version"`
	Accounts    []OpenCodeKiroAccount `json:"accounts"`
	ActiveIndex int                   `json:"activeIndex"`
}

// OpenCodeKiroUsageEntry represents a single account's usage in kiro-usage.json
type OpenCodeKiroUsageEntry struct {
	UsedCount  int    `json:"usedCount"`
	LimitCount int    `json:"limitCount"`
	RealEmail  string `json:"realEmail"`
	LastSync   int64  `json:"lastSync"`
}

// OpenCodeKiroUsageFile represents the kiro-usage.json file structure
type OpenCodeKiroUsageFile struct {
	Version int                               `json:"version"`
	Usage   map[string]OpenCodeKiroUsageEntry `json:"usage"`
}

// OpenCodeKiroSystem handles OpenCode Kiro plugin configuration
type OpenCodeKiroSystem struct{}

// NewOpenCodeKiroSystem creates a new OpenCodeKiroSystem instance
func NewOpenCodeKiroSystem() *OpenCodeKiroSystem {
	return &OpenCodeKiroSystem{}
}

// GetOpenCodeConfigDir returns the OpenCode config directory path
func (oks *OpenCodeKiroSystem) GetOpenCodeConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "opencode"), nil
}

// GetKiroAccountsPath returns the path to kiro-accounts.json
func (oks *OpenCodeKiroSystem) GetKiroAccountsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "opencode", "kiro-accounts.json"), nil
}

// ReadKiroAccounts reads the kiro-accounts.json file
func (oks *OpenCodeKiroSystem) ReadKiroAccounts() (*OpenCodeKiroAccountsFile, error) {
	path, err := oks.GetKiroAccountsPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, return empty structure
			return &OpenCodeKiroAccountsFile{
				Version:     1,
				Accounts:    []OpenCodeKiroAccount{},
				ActiveIndex: 0,
			}, nil
		}
		return nil, err
	}

	var accountsFile OpenCodeKiroAccountsFile
	if err := json.Unmarshal(data, &accountsFile); err != nil {
		return nil, err
	}

	return &accountsFile, nil
}

// WriteKiroAccounts writes the kiro-accounts.json file
func (oks *OpenCodeKiroSystem) WriteKiroAccounts(accountsFile *OpenCodeKiroAccountsFile) error {
	path, err := oks.GetKiroAccountsPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(accountsFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal accounts: %w", err)
	}

	// Atomic write with proper error handling
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Remove old file if exists (to avoid permission issues)
	if _, err := os.Stat(path); err == nil {
		if err := os.Remove(path); err != nil {
			os.Remove(tmpPath) // cleanup temp file
			return fmt.Errorf("failed to remove old file: %w", err)
		}
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath) // cleanup temp file
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// ApplyAccountToOpenCode applies a Kiro account to OpenCode configuration
// This updates or adds the account in kiro-accounts.json and sets it as active
// NOTE: This will REPLACE all accounts in the file with only the current account
func (oks *OpenCodeKiroSystem) ApplyAccountToOpenCode(account *KiroAccount) error {
	fmt.Printf("\n========================================\n")
	fmt.Printf("  → ApplyAccountToOpenCode 开始\n")
	fmt.Printf("  → 账号 ID: %s\n", account.ID)
	fmt.Printf("  → 账号邮箱: %s\n", account.Email)
	fmt.Printf("  → RefreshToken 长度: %d\n", len(account.RefreshToken))
	fmt.Printf("  → BearerToken 长度: %d\n", len(account.BearerToken))

	// 检查 token 是否过期，如果过期则刷新
	bearerToken := account.BearerToken
	refreshToken := account.RefreshToken
	expiresAt := account.TokenExpiry
	
	if time.Now().After(expiresAt.Add(-5 * time.Minute)) {
		fmt.Printf("  ⚠ Token 已过期或即将过期 (过期时间: %v)\n", expiresAt)
		fmt.Printf("  → 正在刷新 Token...\n")
		
		// 使用 Kiro API 客户端刷新 token
		kiroClient := NewKiroAPIClient()
		tokenResp, err := kiroClient.RefreshKiroToken(refreshToken)
		if err != nil {
			fmt.Printf("  ✗ Token 刷新失败: %v\n", err)
			return fmt.Errorf("token 刷新失败: %w", err)
		}
		
		bearerToken = tokenResp.AccessToken
		if tokenResp.RefreshToken != "" {
			refreshToken = tokenResp.RefreshToken
		}
		expiresAt = time.Now().Add(1 * time.Hour)
		
		fmt.Printf("  ✓ Token 刷新成功\n")
		
		// 更新账号对象中的 token（这样调用者也能获得新 token）
		account.BearerToken = bearerToken
		account.RefreshToken = refreshToken
		account.TokenExpiry = expiresAt
	} else {
		fmt.Printf("  ✓ Token 有效 (过期时间: %v)\n", expiresAt)
	}

	// Create OpenCode account structure
	// OpenCode Kiro 插件要求 authMethod='idc' 时必须有 clientId 和 clientSecret
	// 对于 Social 账号，我们提供假的凭据，实际认证通过 profileArn
	authMethod := "idc"
	
	// 如果没有 ProfileArn，使用默认值
	profileArn := account.ProfileArn
	if profileArn == "" {
		profileArn = "arn:aws:codewhisperer:us-east-1:699475941385:profile/EHGA3GRVQMUK"
	}
	
	// 生成假的 clientId 和 clientSecret（插件验证需要，但实际不使用）
	clientID := "kiro-social-dummy-client"
	clientSecret := "kiro-social-dummy-secret"

	openCodeAccount := OpenCodeKiroAccount{
		ID:                 account.ID,
		Email:              account.Email,
		AuthMethod:         authMethod,
		Region:             "us-east-1",
		ClientID:           clientID,
		ClientSecret:       clientSecret,
		RefreshToken:       refreshToken,
		AccessToken:        bearerToken,
		ExpiresAt:          expiresAt.UnixMilli(),
		RateLimitResetTime: 0,
		IsHealthy:          true,
		RealEmail:          account.Email,
		UserID:             account.UserID,
		ProfileArn:         profileArn,
	}
	fmt.Printf("  ✓ 创建 OpenCode 账号结构完成 (authMethod=%s, profileArn=%s)\n", authMethod, profileArn)

	// Create a new accounts file with ONLY this account
	accountsFile := &OpenCodeKiroAccountsFile{
		Version:     1,
		Accounts:    []OpenCodeKiroAccount{openCodeAccount},
		ActiveIndex: 0, // Always 0 since we only have one account
	}
	fmt.Printf("  ✓ 创建账号文件结构完成（账号数: %d）\n", len(accountsFile.Accounts))

	// Get file path
	path, err := oks.GetKiroAccountsPath()
	if err != nil {
		fmt.Printf("  ✗ 获取文件路径失败: %v\n", err)
		return fmt.Errorf("failed to get kiro-accounts path: %w", err)
	}
	fmt.Printf("  → 目标文件路径: %s\n", path)

	// Check if file exists and get current mod time
	if info, err := os.Stat(path); err == nil {
		fmt.Printf("  → 文件当前修改时间: %v\n", info.ModTime())
	}

	// Write to file
	fmt.Printf("  → 开始写入文件...\n")
	if err := oks.WriteKiroAccounts(accountsFile); err != nil {
		fmt.Printf("  ✗ 写入失败: %v\n", err)
		return fmt.Errorf("failed to write kiro-accounts.json: %w", err)
	}

	// Verify write
	if info, err := os.Stat(path); err == nil {
		fmt.Printf("  ✓ 文件写入成功！新修改时间: %v\n", info.ModTime())
	}

	// Read back to verify
	verifyFile, err := oks.ReadKiroAccounts()
	if err != nil {
		fmt.Printf("  ⚠ 警告: 无法读取文件验证: %v\n", err)
	} else {
		fmt.Printf("  ✓ 验证: 文件中账号数 = %d\n", len(verifyFile.Accounts))
		if len(verifyFile.Accounts) > 0 {
			fmt.Printf("  ✓ 验证: 第一个账号邮箱 = %s\n", verifyFile.Accounts[0].Email)
		}
	}

	// 同时更新 kiro-usage.json 文件
	fmt.Printf("  → 更新 kiro-usage.json...\n")
	if err := oks.UpdateKiroUsage(account); err != nil {
		fmt.Printf("  ⚠ 警告: 更新 usage 文件失败: %v\n", err)
		// 不返回错误，因为主要的账号切换已经成功
	} else {
		fmt.Printf("  ✓ usage 文件更新成功\n")
	}

	fmt.Printf("========================================\n\n")
	return nil
}

// UpdateKiroUsage updates the kiro-usage.json file with the current account
// This ensures the usage tracking file is in sync with kiro-accounts.json
func (oks *OpenCodeKiroSystem) UpdateKiroUsage(account *KiroAccount) error {
	// Create usage structure with only the current account
	usageFile := &OpenCodeKiroUsageFile{
		Version: 1,
		Usage: map[string]OpenCodeKiroUsageEntry{
			account.ID: {
				UsedCount:  0, // 重置使用计数
				LimitCount: account.Quota.GetTotalAvailable(),
				RealEmail:  account.Email,
				LastSync:   time.Now().UnixMilli(),
			},
		},
	}

	// Write to file
	return oks.WriteKiroUsage(usageFile)
}

// GetKiroUsagePath returns the path to kiro-usage.json
func (oks *OpenCodeKiroSystem) GetKiroUsagePath() (string, error) {
	configDir, err := oks.GetOpenCodeConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "kiro-usage.json"), nil
}

// WriteKiroUsage writes the usage file to disk
func (oks *OpenCodeKiroSystem) WriteKiroUsage(usageFile *OpenCodeKiroUsageFile) error {
	path, err := oks.GetKiroUsagePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(usageFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal usage file: %w", err)
	}

	// Write to temp file first
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp usage file: %w", err)
	}

	// Atomic rename
	return os.Rename(tmpPath, path)
}

// ReadKiroUsage reads the kiro-usage.json file
func (oks *OpenCodeKiroSystem) ReadKiroUsage() (*OpenCodeKiroUsageFile, error) {
	path, err := oks.GetKiroUsagePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read usage file: %w", err)
	}

	var usageFile OpenCodeKiroUsageFile
	if err := json.Unmarshal(data, &usageFile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal usage file: %w", err)
	}

	return &usageFile, nil
}

// GetActiveOpenCodeAccount returns the currently active account in OpenCode
func (oks *OpenCodeKiroSystem) GetActiveOpenCodeAccount() (*OpenCodeKiroAccount, error) {
	accountsFile, err := oks.ReadKiroAccounts()
	if err != nil {
		return nil, fmt.Errorf("failed to read kiro-accounts.json: %w", err)
	}

	if len(accountsFile.Accounts) == 0 {
		return nil, fmt.Errorf("no accounts in kiro-accounts.json")
	}

	if accountsFile.ActiveIndex < 0 || accountsFile.ActiveIndex >= len(accountsFile.Accounts) {
		return nil, fmt.Errorf("invalid active index: %d", accountsFile.ActiveIndex)
	}

	return &accountsFile.Accounts[accountsFile.ActiveIndex], nil
}

// RemoveAccountFromOpenCode removes an account from OpenCode configuration
func (oks *OpenCodeKiroSystem) RemoveAccountFromOpenCode(accountID string) error {
	accountsFile, err := oks.ReadKiroAccounts()
	if err != nil {
		return fmt.Errorf("failed to read kiro-accounts.json: %w", err)
	}

	// Find and remove account
	newAccounts := []OpenCodeKiroAccount{}
	removedIndex := -1
	for i, acc := range accountsFile.Accounts {
		if acc.ID == accountID {
			removedIndex = i
			continue
		}
		newAccounts = append(newAccounts, acc)
	}

	if removedIndex == -1 {
		return fmt.Errorf("account not found in kiro-accounts.json")
	}

	accountsFile.Accounts = newAccounts

	// Adjust active index if needed
	if accountsFile.ActiveIndex == removedIndex {
		// Removed account was active, set to first account or 0
		accountsFile.ActiveIndex = 0
	} else if accountsFile.ActiveIndex > removedIndex {
		// Active index was after removed account, decrement
		accountsFile.ActiveIndex--
	}

	// Write back to file
	if err := oks.WriteKiroAccounts(accountsFile); err != nil {
		return fmt.Errorf("failed to write kiro-accounts.json: %w", err)
	}

	return nil
}
