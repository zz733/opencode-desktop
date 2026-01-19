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
	ID               string `json:"id"`
	Email            string `json:"email"`
	AuthMethod       string `json:"authMethod"`
	Region           string `json:"region"`
	ClientID         string `json:"clientId,omitempty"`
	ClientSecret     string `json:"clientSecret,omitempty"`
	RefreshToken     string `json:"refreshToken"`
	AccessToken      string `json:"accessToken"`
	ExpiresAt        int64  `json:"expiresAt"`
	RateLimitResetTime int64 `json:"rateLimitResetTime"`
	IsHealthy        bool   `json:"isHealthy"`
	RealEmail        string `json:"realEmail,omitempty"`
}

// OpenCodeKiroAccountsFile represents the kiro-accounts.json file structure
type OpenCodeKiroAccountsFile struct {
	Version     int                     `json:"version"`
	Accounts    []OpenCodeKiroAccount   `json:"accounts"`
	ActiveIndex int                     `json:"activeIndex"`
}

// OpenCodeKiroSystem handles OpenCode Kiro plugin configuration
type OpenCodeKiroSystem struct{}

// NewOpenCodeKiroSystem creates a new OpenCodeKiroSystem instance
func NewOpenCodeKiroSystem() *OpenCodeKiroSystem {
	return &OpenCodeKiroSystem{}
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
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(accountsFile, "", "  ")
	if err != nil {
		return err
	}

	// Atomic write
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}

	return os.Rename(tmpPath, path)
}

// ApplyAccountToOpenCode applies a Kiro account to OpenCode configuration
// This updates or adds the account in kiro-accounts.json and sets it as active
// NOTE: This will REPLACE all accounts in the file with only the current account
func (oks *OpenCodeKiroSystem) ApplyAccountToOpenCode(account *KiroAccount) error {
	fmt.Printf("  → ApplyAccountToOpenCode 开始 (email=%s)\n", account.Email)

	// Create OpenCode account structure
	openCodeAccount := OpenCodeKiroAccount{
		ID:                 account.ID,
		Email:              account.Email,
		AuthMethod:         "social", // Default to social, could be "idc" for IdC accounts
		Region:             "us-east-1",
		RefreshToken:       account.RefreshToken,
		AccessToken:        account.BearerToken,
		ExpiresAt:          time.Now().Add(1 * time.Hour).UnixMilli(),
		RateLimitResetTime: 0,
		IsHealthy:          true,
		RealEmail:          account.Email,
	}
	fmt.Printf("  → 创建 OpenCode 账号结构: ID=%s, Email=%s\n", openCodeAccount.ID, openCodeAccount.Email)

	// Create a new accounts file with ONLY this account
	accountsFile := &OpenCodeKiroAccountsFile{
		Version:     1,
		Accounts:    []OpenCodeKiroAccount{openCodeAccount},
		ActiveIndex: 0, // Always 0 since we only have one account
	}
	fmt.Printf("  → 创建新的账号文件（只包含当前账号）\n")

	// Write back to file
	path, _ := oks.GetKiroAccountsPath()
	fmt.Printf("  → 准备写入文件: %s\n", path)
	if err := oks.WriteKiroAccounts(accountsFile); err != nil {
		fmt.Printf("  ✗ 写入失败: %v\n", err)
		return fmt.Errorf("failed to write kiro-accounts.json: %w", err)
	}
	fmt.Println("  ✓ 文件写入成功（已替换为当前账号）")

	return nil
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
