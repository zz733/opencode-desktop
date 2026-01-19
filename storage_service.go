package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// StorageService handles persistent storage of account data
type StorageService struct {
	dataDir    string
	crypto     *CryptoService
	backupDir  string
	maxBackups int
	settings   *AccountSettings
}

type storedKiroAccount struct {
	ID               string           `json:"id"`
	Email            string           `json:"email"`
	DisplayName      string           `json:"displayName"`
	Avatar           string           `json:"avatar,omitempty"`
	BearerToken      string           `json:"bearerToken,omitempty"`
	RefreshToken     string           `json:"refreshToken,omitempty"`
	TokenExpiry      time.Time        `json:"tokenExpiry"`
	LoginMethod      LoginMethod      `json:"loginMethod"`
	Provider         OAuthProvider    `json:"provider,omitempty"`
	SubscriptionType SubscriptionType `json:"subscriptionType"`
	Quota            QuotaInfo        `json:"quota"`
	Tags             []string         `json:"tags"`
	Notes            string           `json:"notes,omitempty"`
	IsActive         bool             `json:"isActive"`
	LastUsed         time.Time        `json:"lastUsed"`
	CreatedAt        time.Time        `json:"createdAt"`
	MachineID        string           `json:"machineId,omitempty"`
	SqmID            string           `json:"sqmId,omitempty"`
	DevDeviceID      string           `json:"devDeviceId,omitempty"`
}

type storedAccountData struct {
	Version         string               `json:"version"`
	Accounts        []*storedKiroAccount `json:"accounts"`
	ActiveAccountID string               `json:"activeAccountId"`
	Settings        AccountSettings      `json:"settings"`
	Tags            []Tag                `json:"tags"`
	LastUpdated     time.Time            `json:"lastUpdated"`
}

func toStoredAccountData(data *AccountData) *storedAccountData {
	if data == nil {
		return nil
	}

	accounts := make([]*storedKiroAccount, 0, len(data.Accounts))
	for _, account := range data.Accounts {
		if account == nil {
			continue
		}
		accounts = append(accounts, &storedKiroAccount{
			ID:               account.ID,
			Email:            account.Email,
			DisplayName:      account.DisplayName,
			Avatar:           account.Avatar,
			BearerToken:      account.BearerToken,
			RefreshToken:     account.RefreshToken,
			TokenExpiry:      account.TokenExpiry,
			LoginMethod:      account.LoginMethod,
			Provider:         account.Provider,
			SubscriptionType: account.SubscriptionType,
			Quota:            account.Quota,
			Tags:             append([]string(nil), account.Tags...),
			Notes:            account.Notes,
			IsActive:         account.IsActive,
			LastUsed:         account.LastUsed,
			CreatedAt:        account.CreatedAt,
			MachineID:        account.MachineID,
			SqmID:            account.SqmID,
			DevDeviceID:      account.DevDeviceID,
		})
	}

	return &storedAccountData{
		Version:         data.Version,
		Accounts:        accounts,
		ActiveAccountID: data.ActiveAccountID,
		Settings:        data.Settings,
		Tags:            append([]Tag(nil), data.Tags...),
		LastUpdated:     data.LastUpdated,
	}
}

func fromStoredAccountData(data *storedAccountData) *AccountData {
	if data == nil {
		return nil
	}

	accounts := make([]*KiroAccount, 0, len(data.Accounts))
	for _, account := range data.Accounts {
		if account == nil {
			continue
		}
		accounts = append(accounts, &KiroAccount{
			ID:               account.ID,
			Email:            account.Email,
			DisplayName:      account.DisplayName,
			Avatar:           account.Avatar,
			BearerToken:      account.BearerToken,
			RefreshToken:     account.RefreshToken,
			TokenExpiry:      account.TokenExpiry,
			LoginMethod:      account.LoginMethod,
			Provider:         account.Provider,
			SubscriptionType: account.SubscriptionType,
			Quota:            account.Quota,
			Tags:             append([]string(nil), account.Tags...),
			Notes:            account.Notes,
			IsActive:         account.IsActive,
			LastUsed:         account.LastUsed,
			CreatedAt:        account.CreatedAt,
			MachineID:        account.MachineID,
			SqmID:            account.SqmID,
			DevDeviceID:      account.DevDeviceID,
		})
	}

	return &AccountData{
		Version:         data.Version,
		Accounts:        accounts,
		ActiveAccountID: data.ActiveAccountID,
		Settings:        data.Settings,
		Tags:            append([]Tag(nil), data.Tags...),
		LastUpdated:     data.LastUpdated,
	}
}

// NewStorageService creates a new StorageService instance
func NewStorageService(dataDir string, crypto *CryptoService) *StorageService {
	ss := &StorageService{
		dataDir:    dataDir,
		crypto:     crypto,
		backupDir:  filepath.Join(dataDir, "backups"),
		maxBackups: 10,  // Keep last 10 backups
		settings:   nil, // Will be loaded from file
	}

	// Ensure directories exist
	ss.ensureDirectories()

	return ss
}

// ensureDirectories creates necessary directories if they don't exist
func (ss *StorageService) ensureDirectories() error {
	dirs := []string{ss.dataDir, ss.backupDir}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// SaveAccountData saves account data to encrypted storage
func (ss *StorageService) SaveAccountData(data *AccountData) error {
	if data == nil {
		return fmt.Errorf("account data cannot be nil")
	}

	// Update last updated timestamp
	data.LastUpdated = time.Now()

	// Serialize to JSON
	storedData := toStoredAccountData(data)
	jsonData, err := json.MarshalIndent(storedData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal account data: %w", err)
	}

	// Encrypt the data
	encryptedData, err := ss.crypto.Encrypt(jsonData)
	if err != nil {
		return fmt.Errorf("failed to encrypt account data: %w", err)
	}

	// Write to file
	accountsFile := filepath.Join(ss.dataDir, "accounts.json.enc")
	if err := os.WriteFile(accountsFile, encryptedData, 0600); err != nil {
		return fmt.Errorf("failed to write account data: %w", err)
	}

	// Create backup
	if err := ss.createBackup(encryptedData); err != nil {
		// Log error but don't fail the save operation
		fmt.Printf("Warning: failed to create backup: %v\n", err)
	}

	return nil
}

// LoadAccountData loads account data from encrypted storage
func (ss *StorageService) LoadAccountData() (*AccountData, error) {
	accountsFile := filepath.Join(ss.dataDir, "accounts.json.enc")

	// Check if file exists
	if _, err := os.Stat(accountsFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("account data file not found")
	}

	// Read encrypted data
	encryptedData, err := os.ReadFile(accountsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read account data: %w", err)
	}

	// Decrypt the data
	jsonData, err := ss.crypto.Decrypt(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt account data: %w", err)
	}

	// Deserialize from JSON
	var storedData storedAccountData
	if err := json.Unmarshal(jsonData, &storedData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal account data: %w", err)
	}

	return fromStoredAccountData(&storedData), nil
}

// SerializeAccountData serializes account data to JSON bytes
func (ss *StorageService) SerializeAccountData(data *AccountData) ([]byte, error) {
	if data == nil {
		return nil, fmt.Errorf("account data cannot be nil")
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal account data: %w", err)
	}

	return jsonData, nil
}

// DeserializeAccountData deserializes account data from JSON bytes
func (ss *StorageService) DeserializeAccountData(data []byte) (*AccountData, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data cannot be empty")
	}

	var accountData AccountData
	if err := json.Unmarshal(data, &accountData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal account data: %w", err)
	}

	return &accountData, nil
}

// createBackup creates a backup of the account data
func (ss *StorageService) createBackup(encryptedData []byte) error {
	// Generate backup filename with timestamp (including milliseconds for uniqueness)
	timestamp := time.Now().Format("20060102_150405.000")
	backupFile := filepath.Join(ss.backupDir, fmt.Sprintf("accounts_%s.json.enc", timestamp))

	// Write backup file
	if err := os.WriteFile(backupFile, encryptedData, 0600); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	// Clean up old backups
	if err := ss.cleanupOldBackups(); err != nil {
		// Log error but don't fail the backup operation
		fmt.Printf("Warning: failed to cleanup old backups: %v\n", err)
	}

	return nil
}

// cleanupOldBackups removes old backup files, keeping only the most recent ones
func (ss *StorageService) cleanupOldBackups() error {
	// Read backup directory
	entries, err := os.ReadDir(ss.backupDir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	// Filter backup files and sort by modification time
	var backupFiles []os.FileInfo
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".enc" {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			backupFiles = append(backupFiles, info)
		}
	}

	// If we have more backups than the limit, remove the oldest ones
	if len(backupFiles) > ss.maxBackups {
		// Sort by modification time (oldest first)
		for i := 0; i < len(backupFiles)-1; i++ {
			for j := i + 1; j < len(backupFiles); j++ {
				if backupFiles[i].ModTime().After(backupFiles[j].ModTime()) {
					backupFiles[i], backupFiles[j] = backupFiles[j], backupFiles[i]
				}
			}
		}

		// Remove oldest files
		filesToRemove := len(backupFiles) - ss.maxBackups
		for i := 0; i < filesToRemove; i++ {
			backupPath := filepath.Join(ss.backupDir, backupFiles[i].Name())
			if err := os.Remove(backupPath); err != nil {
				fmt.Printf("Warning: failed to remove old backup %s: %v\n", backupPath, err)
			}
		}
	}

	return nil
}

// RestoreFromBackup restores account data from a backup file
func (ss *StorageService) RestoreFromBackup(backupPath string) error {
	// Check if backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", backupPath)
	}

	// Read backup file
	encryptedData, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// Validate the backup by trying to decrypt and parse it
	jsonData, err := ss.crypto.Decrypt(encryptedData)
	if err != nil {
		return fmt.Errorf("failed to decrypt backup data: %w", err)
	}

	var accountData AccountData
	if err := json.Unmarshal(jsonData, &accountData); err != nil {
		return fmt.Errorf("failed to parse backup data: %w", err)
	}

	// Copy backup to main accounts file
	accountsFile := filepath.Join(ss.dataDir, "accounts.json.enc")
	if err := os.WriteFile(accountsFile, encryptedData, 0600); err != nil {
		return fmt.Errorf("failed to restore from backup: %w", err)
	}

	return nil
}

// ListBackups returns a list of available backup files
func (ss *StorageService) ListBackups() ([]BackupInfo, error) {
	entries, err := os.ReadDir(ss.backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []BackupInfo
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".enc" {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			backup := BackupInfo{
				Name:    entry.Name(),
				Path:    filepath.Join(ss.backupDir, entry.Name()),
				Size:    info.Size(),
				ModTime: info.ModTime(),
			}
			backups = append(backups, backup)
		}
	}

	return backups, nil
}

// GetStorageStats returns statistics about storage usage
func (ss *StorageService) GetStorageStats() (StorageStats, error) {
	stats := StorageStats{
		DataDir:   ss.dataDir,
		BackupDir: ss.backupDir,
	}

	// Get main accounts file info
	accountsFile := filepath.Join(ss.dataDir, "accounts.json.enc")
	if info, err := os.Stat(accountsFile); err == nil {
		stats.AccountsFileSize = info.Size()
		stats.LastModified = info.ModTime()
	}

	// Count backups and calculate total size
	backups, err := ss.ListBackups()
	if err == nil {
		stats.BackupCount = len(backups)
		for _, backup := range backups {
			stats.TotalBackupSize += backup.Size
		}
	}

	return stats, nil
}

// BackupInfo represents information about a backup file
type BackupInfo struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
}

// StorageStats represents storage statistics
type StorageStats struct {
	DataDir          string    `json:"dataDir"`
	BackupDir        string    `json:"backupDir"`
	AccountsFileSize int64     `json:"accountsFileSize"`
	BackupCount      int       `json:"backupCount"`
	TotalBackupSize  int64     `json:"totalBackupSize"`
	LastModified     time.Time `json:"lastModified"`
}

// ExportToFile exports account data to a specified file
func (ss *StorageService) ExportToFile(data *AccountData, filePath string, encrypt bool, password string) error {
	// Serialize data
	jsonData, err := ss.SerializeAccountData(data)
	if err != nil {
		return fmt.Errorf("failed to serialize data: %w", err)
	}

	var outputData []byte

	if encrypt && password != "" {
		// Encrypt with password
		encryptedData, err := ss.crypto.EncryptWithPassword(jsonData, password)
		if err != nil {
			return fmt.Errorf("failed to encrypt data: %w", err)
		}
		outputData = encryptedData
	} else {
		outputData = jsonData
	}

	// Write to file
	if err := os.WriteFile(filePath, outputData, 0600); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	return nil
}

// ImportFromFile imports account data from a specified file
func (ss *StorageService) ImportFromFile(filePath string, decrypt bool, password string) (*AccountData, error) {
	// Read file
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read import file: %w", err)
	}

	var jsonData []byte

	if decrypt && password != "" {
		// Decrypt with password
		decryptedData, err := ss.crypto.DecryptWithPassword(fileData, password)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt data: %w", err)
		}
		jsonData = decryptedData
	} else {
		jsonData = fileData
	}

	// Deserialize data
	accountData, err := ss.DeserializeAccountData(jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize data: %w", err)
	}

	return accountData, nil
}

// SaveSettings saves account settings to storage
func (ss *StorageService) SaveSettings(settings *AccountSettings) error {
	if settings == nil {
		return fmt.Errorf("settings cannot be nil")
	}

	// Update internal settings
	ss.settings = settings

	// Serialize to JSON
	jsonData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Write to file (settings are not encrypted for easier debugging)
	settingsFile := filepath.Join(ss.dataDir, "settings.json")
	if err := os.WriteFile(settingsFile, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write settings: %w", err)
	}

	return nil
}

// LoadSettings loads account settings from storage
func (ss *StorageService) LoadSettings() (*AccountSettings, error) {
	settingsFile := filepath.Join(ss.dataDir, "settings.json")

	// Check if file exists
	if _, err := os.Stat(settingsFile); os.IsNotExist(err) {
		// Return default settings if file doesn't exist
		defaultSettings := DefaultAccountSettings()
		ss.settings = &defaultSettings
		return &defaultSettings, nil
	}

	// Read settings file
	jsonData, err := os.ReadFile(settingsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read settings: %w", err)
	}

	// Deserialize from JSON
	var settings AccountSettings
	if err := json.Unmarshal(jsonData, &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	ss.settings = &settings
	return &settings, nil
}

// SaveTags saves tags data to storage
func (ss *StorageService) SaveTags(tags []Tag) error {
	if tags == nil {
		tags = []Tag{} // Ensure we have an empty array instead of nil
	}

	// Serialize to JSON
	jsonData, err := json.MarshalIndent(map[string]interface{}{
		"version": "1.0",
		"tags":    tags,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	// Write to file
	tagsFile := filepath.Join(ss.dataDir, "tags.json")
	if err := os.WriteFile(tagsFile, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write tags: %w", err)
	}

	return nil
}

// LoadTags loads tags data from storage
func (ss *StorageService) LoadTags() ([]Tag, error) {
	tagsFile := filepath.Join(ss.dataDir, "tags.json")

	// Check if file exists
	if _, err := os.Stat(tagsFile); os.IsNotExist(err) {
		// Return empty tags if file doesn't exist
		return []Tag{}, nil
	}

	// Read tags file
	jsonData, err := os.ReadFile(tagsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read tags: %w", err)
	}

	// Deserialize from JSON
	var tagsData struct {
		Version string `json:"version"`
		Tags    []Tag  `json:"tags"`
	}
	if err := json.Unmarshal(jsonData, &tagsData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
	}

	return tagsData.Tags, nil
}

// CreateBackup creates a manual backup of the current account data
func (ss *StorageService) CreateBackup() error {
	// Read current account data
	accountsFile := filepath.Join(ss.dataDir, "accounts.json.enc")
	if _, err := os.Stat(accountsFile); os.IsNotExist(err) {
		return fmt.Errorf("no account data to backup")
	}

	encryptedData, err := os.ReadFile(accountsFile)
	if err != nil {
		return fmt.Errorf("failed to read account data for backup: %w", err)
	}

	// Create backup using existing method
	return ss.createBackup(encryptedData)
}

// GetDataDirectory returns the data directory path
func (ss *StorageService) GetDataDirectory() string {
	return ss.dataDir
}

// GetBackupDirectory returns the backup directory path
func (ss *StorageService) GetBackupDirectory() string {
	return ss.backupDir
}

// ValidateDataIntegrity validates the integrity of stored data
func (ss *StorageService) ValidateDataIntegrity() error {
	// Check if accounts file exists and is readable
	accountsFile := filepath.Join(ss.dataDir, "accounts.json.enc")
	if _, err := os.Stat(accountsFile); err != nil {
		if os.IsNotExist(err) {
			return nil // No data to validate
		}
		return fmt.Errorf("cannot access accounts file: %w", err)
	}

	// Try to load and decrypt the data
	_, err := ss.LoadAccountData()
	if err != nil {
		return fmt.Errorf("data integrity check failed: %w", err)
	}

	return nil
}
