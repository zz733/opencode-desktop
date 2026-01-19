package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// ConfigManager handles configuration files and directory structure management
type ConfigManager struct {
	baseDir       string
	dataDir       string
	configDir     string
	logsDir       string
	backupDir     string
	tempDir       string
	crypto        *CryptoService
	initialized   bool
}

// ConfigPaths contains all the important configuration paths
type ConfigPaths struct {
	BaseDir       string `json:"baseDir"`
	DataDir       string `json:"dataDir"`
	ConfigDir     string `json:"configDir"`
	LogsDir       string `json:"logsDir"`
	BackupDir     string `json:"backupDir"`
	TempDir       string `json:"tempDir"`
	AccountsFile  string `json:"accountsFile"`
	SettingsFile  string `json:"settingsFile"`
	TagsFile      string `json:"tagsFile"`
}

// AppConfig represents the main application configuration
type AppConfig struct {
	Version         string          `json:"version"`
	AppName         string          `json:"appName"`
	DataVersion     string          `json:"dataVersion"`
	Paths           ConfigPaths     `json:"paths"`
	Security        SecurityConfig  `json:"security"`
	Storage         StorageConfig   `json:"storage"`
	Logging         LoggingConfig   `json:"logging"`
	CreatedAt       time.Time       `json:"createdAt"`
	LastUpdated     time.Time       `json:"lastUpdated"`
}

// SecurityConfig contains security-related configuration
type SecurityConfig struct {
	EncryptionEnabled    bool   `json:"encryptionEnabled"`
	KeyDerivationMethod  string `json:"keyDerivationMethod"`
	EncryptionAlgorithm  string `json:"encryptionAlgorithm"`
	TokenStorageMethod   string `json:"tokenStorageMethod"`
	AutoLockTimeout      int    `json:"autoLockTimeout"` // minutes
	RequirePasswordOnStart bool `json:"requirePasswordOnStart"`
}

// StorageConfig contains storage-related configuration
type StorageConfig struct {
	MaxBackups          int  `json:"maxBackups"`
	AutoBackupEnabled   bool `json:"autoBackupEnabled"`
	BackupInterval      int  `json:"backupInterval"` // hours
	CompressBackups     bool `json:"compressBackups"`
	CleanupOldBackups   bool `json:"cleanupOldBackups"`
	BackupRetentionDays int  `json:"backupRetentionDays"`
}

// LoggingConfig contains logging-related configuration
type LoggingConfig struct {
	Enabled         bool   `json:"enabled"`
	Level           string `json:"level"`
	MaxFileSize     int64  `json:"maxFileSize"` // bytes
	MaxFiles        int    `json:"maxFiles"`
	RotateDaily     bool   `json:"rotateDaily"`
	LogToConsole    bool   `json:"logToConsole"`
}

// NewConfigManager creates a new ConfigManager instance
func NewConfigManager(crypto *CryptoService) (*ConfigManager, error) {
	cm := &ConfigManager{
		crypto:      crypto,
		initialized: false,
	}

	// Determine base directory based on OS
	baseDir, err := cm.getDefaultBaseDirectory()
	if err != nil {
		return nil, fmt.Errorf("failed to determine base directory: %w", err)
	}

	cm.baseDir = baseDir
	cm.setupDirectoryPaths()

	return cm, nil
}

// Initialize initializes the configuration manager and creates necessary directories
func (cm *ConfigManager) Initialize() error {
	if cm.initialized {
		return nil
	}

	// Create all necessary directories
	if err := cm.createDirectories(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Initialize configuration file if it doesn't exist
	if err := cm.initializeConfigFile(); err != nil {
		return fmt.Errorf("failed to initialize config file: %w", err)
	}

	// Validate directory structure
	if err := cm.validateDirectoryStructure(); err != nil {
		return fmt.Errorf("directory structure validation failed: %w", err)
	}

	cm.initialized = true
	return nil
}

// getDefaultBaseDirectory returns the default base directory for the application
func (cm *ConfigManager) getDefaultBaseDirectory() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	var baseDir string
	switch runtime.GOOS {
	case "windows":
		// Use AppData/Roaming on Windows
		appData := os.Getenv("APPDATA")
		if appData != "" {
			baseDir = filepath.Join(appData, "OpenCode", "KiroAccountManager")
		} else {
			baseDir = filepath.Join(homeDir, "AppData", "Roaming", "OpenCode", "KiroAccountManager")
		}
	case "darwin":
		// Use Application Support on macOS
		baseDir = filepath.Join(homeDir, "Library", "Application Support", "OpenCode", "KiroAccountManager")
	default:
		// Use .config on Linux and other Unix-like systems
		configHome := os.Getenv("XDG_CONFIG_HOME")
		if configHome != "" {
			baseDir = filepath.Join(configHome, "opencode", "kiro-account-manager")
		} else {
			baseDir = filepath.Join(homeDir, ".config", "opencode", "kiro-account-manager")
		}
	}

	return baseDir, nil
}

// setupDirectoryPaths sets up all the directory paths
func (cm *ConfigManager) setupDirectoryPaths() {
	cm.dataDir = filepath.Join(cm.baseDir, "data")
	cm.configDir = filepath.Join(cm.baseDir, "config")
	cm.logsDir = filepath.Join(cm.baseDir, "logs")
	cm.backupDir = filepath.Join(cm.baseDir, "backups")
	cm.tempDir = filepath.Join(cm.baseDir, "temp")
}

// createDirectories creates all necessary directories
func (cm *ConfigManager) createDirectories() error {
	directories := []string{
		cm.baseDir,
		cm.dataDir,
		cm.configDir,
		cm.logsDir,
		cm.backupDir,
		cm.tempDir,
	}

	for _, dir := range directories {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// initializeConfigFile creates the main configuration file if it doesn't exist
func (cm *ConfigManager) initializeConfigFile() error {
	configFile := filepath.Join(cm.configDir, "app.json")
	
	// Check if config file already exists
	if _, err := os.Stat(configFile); err == nil {
		return nil // Config file already exists
	}

	// Create default configuration
	config := cm.getDefaultAppConfig()
	
	// Save configuration file
	return cm.SaveAppConfig(config)
}

// getDefaultAppConfig returns the default application configuration
func (cm *ConfigManager) getDefaultAppConfig() *AppConfig {
	now := time.Now()
	
	return &AppConfig{
		Version:     "1.0.0",
		AppName:     "Kiro Account Manager",
		DataVersion: "1.0",
		Paths: ConfigPaths{
			BaseDir:      cm.baseDir,
			DataDir:      cm.dataDir,
			ConfigDir:    cm.configDir,
			LogsDir:      cm.logsDir,
			BackupDir:    cm.backupDir,
			TempDir:      cm.tempDir,
			AccountsFile: filepath.Join(cm.dataDir, "accounts.json.enc"),
			SettingsFile: filepath.Join(cm.dataDir, "settings.json"),
			TagsFile:     filepath.Join(cm.dataDir, "tags.json"),
		},
		Security: SecurityConfig{
			EncryptionEnabled:      true,
			KeyDerivationMethod:    "PBKDF2",
			EncryptionAlgorithm:    "AES-256-GCM",
			TokenStorageMethod:     "encrypted",
			AutoLockTimeout:        30, // 30 minutes
			RequirePasswordOnStart: false,
		},
		Storage: StorageConfig{
			MaxBackups:          10,
			AutoBackupEnabled:   true,
			BackupInterval:      24, // 24 hours
			CompressBackups:     false,
			CleanupOldBackups:   true,
			BackupRetentionDays: 30,
		},
		Logging: LoggingConfig{
			Enabled:      true,
			Level:        "INFO",
			MaxFileSize:  10 * 1024 * 1024, // 10MB
			MaxFiles:     5,
			RotateDaily:  true,
			LogToConsole: false,
		},
		CreatedAt:   now,
		LastUpdated: now,
	}
}

// LoadAppConfig loads the application configuration from file
func (cm *ConfigManager) LoadAppConfig() (*AppConfig, error) {
	configFile := filepath.Join(cm.configDir, "app.json")
	
	// Check if file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// Return default config if file doesn't exist
		return cm.getDefaultAppConfig(), nil
	}

	// Read config file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	var config AppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// SaveAppConfig saves the application configuration to file
func (cm *ConfigManager) SaveAppConfig(config *AppConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Update last modified time
	config.LastUpdated = time.Now()

	// Serialize to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	configFile := filepath.Join(cm.configDir, "app.json")
	if err := os.WriteFile(configFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// validateDirectoryStructure validates that all required directories exist and are accessible
func (cm *ConfigManager) validateDirectoryStructure() error {
	directories := map[string]string{
		"base":   cm.baseDir,
		"data":   cm.dataDir,
		"config": cm.configDir,
		"logs":   cm.logsDir,
		"backup": cm.backupDir,
		"temp":   cm.tempDir,
	}

	for name, dir := range directories {
		// Check if directory exists
		info, err := os.Stat(dir)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("%s directory does not exist: %s", name, dir)
			}
			return fmt.Errorf("cannot access %s directory: %w", name, err)
		}

		// Check if it's actually a directory
		if !info.IsDir() {
			return fmt.Errorf("%s path is not a directory: %s", name, dir)
		}

		// Check if directory is writable
		testFile := filepath.Join(dir, ".write_test")
		if err := os.WriteFile(testFile, []byte("test"), 0600); err != nil {
			return fmt.Errorf("%s directory is not writable: %w", name, err)
		}
		os.Remove(testFile) // Clean up test file
	}

	return nil
}

// GetPaths returns the configuration paths
func (cm *ConfigManager) GetPaths() ConfigPaths {
	return ConfigPaths{
		BaseDir:      cm.baseDir,
		DataDir:      cm.dataDir,
		ConfigDir:    cm.configDir,
		LogsDir:      cm.logsDir,
		BackupDir:    cm.backupDir,
		TempDir:      cm.tempDir,
		AccountsFile: filepath.Join(cm.dataDir, "accounts.json.enc"),
		SettingsFile: filepath.Join(cm.dataDir, "settings.json"),
		TagsFile:     filepath.Join(cm.dataDir, "tags.json"),
	}
}

// GetDataDirectory returns the data directory path
func (cm *ConfigManager) GetDataDirectory() string {
	return cm.dataDir
}

// GetConfigDirectory returns the config directory path
func (cm *ConfigManager) GetConfigDirectory() string {
	return cm.configDir
}

// GetLogsDirectory returns the logs directory path
func (cm *ConfigManager) GetLogsDirectory() string {
	return cm.logsDir
}

// GetBackupDirectory returns the backup directory path
func (cm *ConfigManager) GetBackupDirectory() string {
	return cm.backupDir
}

// GetTempDirectory returns the temp directory path
func (cm *ConfigManager) GetTempDirectory() string {
	return cm.tempDir
}

// CleanupTempDirectory cleans up temporary files
func (cm *ConfigManager) CleanupTempDirectory() error {
	entries, err := os.ReadDir(cm.tempDir)
	if err != nil {
		return fmt.Errorf("failed to read temp directory: %w", err)
	}

	for _, entry := range entries {
		path := filepath.Join(cm.tempDir, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			// Log error but continue cleanup
			fmt.Printf("Warning: failed to remove temp file %s: %v\n", path, err)
		}
	}

	return nil
}

// CreateTempFile creates a temporary file in the temp directory
func (cm *ConfigManager) CreateTempFile(prefix string) (*os.File, error) {
	return os.CreateTemp(cm.tempDir, prefix)
}

// GetDirectorySize calculates the size of a directory
func (cm *ConfigManager) GetDirectorySize(dirPath string) (int64, error) {
	var size int64
	
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	
	return size, err
}

// GetStorageInfo returns information about storage usage
func (cm *ConfigManager) GetStorageInfo() (map[string]interface{}, error) {
	info := make(map[string]interface{})
	
	// Get directory sizes
	directories := map[string]string{
		"data":   cm.dataDir,
		"config": cm.configDir,
		"logs":   cm.logsDir,
		"backup": cm.backupDir,
		"temp":   cm.tempDir,
	}
	
	totalSize := int64(0)
	for name, dir := range directories {
		size, err := cm.GetDirectorySize(dir)
		if err != nil {
			info[name+"_size"] = "error"
			info[name+"_error"] = err.Error()
		} else {
			info[name+"_size"] = size
			totalSize += size
		}
	}
	
	info["total_size"] = totalSize
	info["base_directory"] = cm.baseDir
	
	return info, nil
}

// MigrateConfiguration migrates configuration from old versions
func (cm *ConfigManager) MigrateConfiguration(fromVersion string) error {
	// This method can be extended to handle configuration migrations
	// For now, it's a placeholder for future migration needs
	
	switch fromVersion {
	case "0.9":
		// Handle migration from version 0.9 to 1.0
		return cm.migrateFrom09()
	default:
		// No migration needed or unsupported version
		return nil
	}
}

// migrateFrom09 handles migration from version 0.9
func (cm *ConfigManager) migrateFrom09() error {
	// Placeholder for migration logic
	// This would handle moving files, updating formats, etc.
	return nil
}

// Reset resets the configuration to defaults (dangerous operation)
func (cm *ConfigManager) Reset() error {
	// Create backup before reset
	backupDir := filepath.Join(cm.baseDir, "reset_backup_"+time.Now().Format("20060102_150405"))
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Copy current config to backup
	configFile := filepath.Join(cm.configDir, "app.json")
	if _, err := os.Stat(configFile); err == nil {
		backupFile := filepath.Join(backupDir, "app.json")
		if data, err := os.ReadFile(configFile); err == nil {
			os.WriteFile(backupFile, data, 0600)
		}
	}

	// Create new default configuration
	config := cm.getDefaultAppConfig()
	return cm.SaveAppConfig(config)
}

// IsInitialized returns whether the configuration manager has been initialized
func (cm *ConfigManager) IsInitialized() bool {
	return cm.initialized
}

// Validate validates the current configuration
func (cm *ConfigManager) Validate() error {
	if !cm.initialized {
		return fmt.Errorf("configuration manager not initialized")
	}

	// Validate directory structure
	if err := cm.validateDirectoryStructure(); err != nil {
		return fmt.Errorf("directory structure validation failed: %w", err)
	}

	// Load and validate configuration file
	config, err := cm.LoadAppConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration values
	if config.Version == "" {
		return fmt.Errorf("configuration version is empty")
	}

	if config.AppName == "" {
		return fmt.Errorf("application name is empty")
	}

	// Validate paths
	paths := config.Paths
	if paths.DataDir == "" || paths.ConfigDir == "" {
		return fmt.Errorf("essential paths are not configured")
	}

	return nil
}