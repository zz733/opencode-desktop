package main

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestConfigManager(t *testing.T) (*ConfigManager, string, func()) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "config_manager_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Create crypto service
	crypto := NewCryptoService("test-master-key")

	// Create config manager
	cm := &ConfigManager{
		baseDir:     tempDir,
		crypto:      crypto,
		initialized: false,
	}
	cm.setupDirectoryPaths()

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return cm, tempDir, cleanup
}

func TestNewConfigManager(t *testing.T) {
	crypto := NewCryptoService("test-key")
	cm, err := NewConfigManager(crypto)
	if err != nil {
		t.Fatalf("Failed to create ConfigManager: %v", err)
	}

	if cm.crypto != crypto {
		t.Error("Crypto service not set correctly")
	}

	if cm.initialized {
		t.Error("ConfigManager should not be initialized by default")
	}

	if cm.baseDir == "" {
		t.Error("Base directory should be set")
	}
}

func TestInitialize(t *testing.T) {
	cm, _, cleanup := setupTestConfigManager(t)
	defer cleanup()

	// Test initialization
	err := cm.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize ConfigManager: %v", err)
	}

	if !cm.initialized {
		t.Error("ConfigManager should be initialized")
	}

	// Verify directories were created
	directories := []string{
		cm.baseDir,
		cm.dataDir,
		cm.configDir,
		cm.logsDir,
		cm.backupDir,
		cm.tempDir,
	}

	for _, dir := range directories {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Directory not created: %s", dir)
		}
	}

	// Verify config file was created
	configFile := filepath.Join(cm.configDir, "app.json")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Error("Config file not created")
	}
}

func TestCreateDirectories(t *testing.T) {
	cm, _, cleanup := setupTestConfigManager(t)
	defer cleanup()

	err := cm.createDirectories()
	if err != nil {
		t.Fatalf("Failed to create directories: %v", err)
	}

	// Check all directories exist
	directories := []string{
		cm.baseDir,
		cm.dataDir,
		cm.configDir,
		cm.logsDir,
		cm.backupDir,
		cm.tempDir,
	}

	for _, dir := range directories {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("Directory does not exist: %s", dir)
			continue
		}
		if !info.IsDir() {
			t.Errorf("Path is not a directory: %s", dir)
		}
	}
}

func TestSaveAndLoadAppConfig(t *testing.T) {
	cm, _, cleanup := setupTestConfigManager(t)
	defer cleanup()

	// Initialize to create directories
	err := cm.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Create test config
	testConfig := cm.getDefaultAppConfig()
	testConfig.Version = "1.2.3"
	testConfig.AppName = "Test App"
	testConfig.Security.AutoLockTimeout = 60

	// Save config
	err = cm.SaveAppConfig(testConfig)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load config
	loadedConfig, err := cm.LoadAppConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify config values
	if loadedConfig.Version != testConfig.Version {
		t.Errorf("Expected version %s, got %s", testConfig.Version, loadedConfig.Version)
	}

	if loadedConfig.AppName != testConfig.AppName {
		t.Errorf("Expected app name %s, got %s", testConfig.AppName, loadedConfig.AppName)
	}

	if loadedConfig.Security.AutoLockTimeout != testConfig.Security.AutoLockTimeout {
		t.Errorf("Expected auto lock timeout %d, got %d", 
			testConfig.Security.AutoLockTimeout, loadedConfig.Security.AutoLockTimeout)
	}
}

func TestLoadAppConfigDefault(t *testing.T) {
	cm, _, cleanup := setupTestConfigManager(t)
	defer cleanup()

	// Initialize to create directories
	err := cm.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Remove config file to test default loading
	configFile := filepath.Join(cm.configDir, "app.json")
	os.Remove(configFile)

	// Load config (should return defaults)
	config, err := cm.LoadAppConfig()
	if err != nil {
		t.Fatalf("Failed to load default config: %v", err)
	}

	// Verify it's the default config
	defaultConfig := cm.getDefaultAppConfig()
	if config.Version != defaultConfig.Version {
		t.Errorf("Expected default version %s, got %s", defaultConfig.Version, config.Version)
	}

	if config.AppName != defaultConfig.AppName {
		t.Errorf("Expected default app name %s, got %s", defaultConfig.AppName, config.AppName)
	}
}

func TestValidateDirectoryStructure(t *testing.T) {
	cm, _, cleanup := setupTestConfigManager(t)
	defer cleanup()

	// Test validation before directories are created (should fail)
	err := cm.validateDirectoryStructure()
	if err == nil {
		t.Error("Validation should fail when directories don't exist")
	}

	// Create directories
	err = cm.createDirectories()
	if err != nil {
		t.Fatalf("Failed to create directories: %v", err)
	}

	// Test validation after directories are created (should pass)
	err = cm.validateDirectoryStructure()
	if err != nil {
		t.Errorf("Validation should pass after directories are created: %v", err)
	}

	// Test validation with non-directory file
	testFile := filepath.Join(cm.baseDir, "test_file")
	os.WriteFile(testFile, []byte("test"), 0600)
	
	// Replace data directory with a file
	os.RemoveAll(cm.dataDir)
	os.WriteFile(cm.dataDir, []byte("not a directory"), 0600)
	
	err = cm.validateDirectoryStructure()
	if err == nil {
		t.Error("Validation should fail when path is not a directory")
	}
}

func TestGetPaths(t *testing.T) {
	cm, tempDir, cleanup := setupTestConfigManager(t)
	defer cleanup()

	paths := cm.GetPaths()

	expectedPaths := ConfigPaths{
		BaseDir:      tempDir,
		DataDir:      filepath.Join(tempDir, "data"),
		ConfigDir:    filepath.Join(tempDir, "config"),
		LogsDir:      filepath.Join(tempDir, "logs"),
		BackupDir:    filepath.Join(tempDir, "backups"),
		TempDir:      filepath.Join(tempDir, "temp"),
		AccountsFile: filepath.Join(tempDir, "data", "accounts.json.enc"),
		SettingsFile: filepath.Join(tempDir, "data", "settings.json"),
		TagsFile:     filepath.Join(tempDir, "data", "tags.json"),
	}

	if paths.BaseDir != expectedPaths.BaseDir {
		t.Errorf("Expected base dir %s, got %s", expectedPaths.BaseDir, paths.BaseDir)
	}

	if paths.DataDir != expectedPaths.DataDir {
		t.Errorf("Expected data dir %s, got %s", expectedPaths.DataDir, paths.DataDir)
	}

	if paths.AccountsFile != expectedPaths.AccountsFile {
		t.Errorf("Expected accounts file %s, got %s", expectedPaths.AccountsFile, paths.AccountsFile)
	}
}

func TestCleanupTempDirectory(t *testing.T) {
	cm, _, cleanup := setupTestConfigManager(t)
	defer cleanup()

	// Initialize to create directories
	err := cm.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Create some temp files
	tempFile1 := filepath.Join(cm.tempDir, "temp1.txt")
	tempFile2 := filepath.Join(cm.tempDir, "temp2.txt")
	os.WriteFile(tempFile1, []byte("temp1"), 0600)
	os.WriteFile(tempFile2, []byte("temp2"), 0600)

	// Verify files exist
	if _, err := os.Stat(tempFile1); os.IsNotExist(err) {
		t.Error("Temp file 1 should exist")
	}
	if _, err := os.Stat(tempFile2); os.IsNotExist(err) {
		t.Error("Temp file 2 should exist")
	}

	// Cleanup temp directory
	err = cm.CleanupTempDirectory()
	if err != nil {
		t.Fatalf("Failed to cleanup temp directory: %v", err)
	}

	// Verify files are removed
	if _, err := os.Stat(tempFile1); !os.IsNotExist(err) {
		t.Error("Temp file 1 should be removed")
	}
	if _, err := os.Stat(tempFile2); !os.IsNotExist(err) {
		t.Error("Temp file 2 should be removed")
	}
}

func TestCreateTempFile(t *testing.T) {
	cm, _, cleanup := setupTestConfigManager(t)
	defer cleanup()

	// Initialize to create directories
	err := cm.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Create temp file
	tempFile, err := cm.CreateTempFile("test_")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer tempFile.Close()

	// Verify file is in temp directory
	tempPath := tempFile.Name()
	expectedDir := cm.tempDir
	actualDir := filepath.Dir(tempPath)
	
	if actualDir != expectedDir {
		t.Errorf("Expected temp file in %s, got %s", expectedDir, actualDir)
	}

	// Verify file can be written to
	_, err = tempFile.WriteString("test content")
	if err != nil {
		t.Errorf("Failed to write to temp file: %v", err)
	}
}

func TestGetDirectorySize(t *testing.T) {
	cm, _, cleanup := setupTestConfigManager(t)
	defer cleanup()

	// Initialize to create directories
	err := cm.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Create some test files
	testFile1 := filepath.Join(cm.dataDir, "test1.txt")
	testFile2 := filepath.Join(cm.dataDir, "test2.txt")
	testContent1 := "Hello, World!"
	testContent2 := "This is a test file."

	os.WriteFile(testFile1, []byte(testContent1), 0600)
	os.WriteFile(testFile2, []byte(testContent2), 0600)

	// Get directory size
	size, err := cm.GetDirectorySize(cm.dataDir)
	if err != nil {
		t.Fatalf("Failed to get directory size: %v", err)
	}

	expectedSize := int64(len(testContent1) + len(testContent2))
	if size != expectedSize {
		t.Errorf("Expected directory size %d, got %d", expectedSize, size)
	}
}

func TestGetStorageInfo(t *testing.T) {
	cm, _, cleanup := setupTestConfigManager(t)
	defer cleanup()

	// Initialize to create directories
	err := cm.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Get storage info
	info, err := cm.GetStorageInfo()
	if err != nil {
		t.Fatalf("Failed to get storage info: %v", err)
	}

	// Verify info contains expected keys
	expectedKeys := []string{
		"data_size", "config_size", "logs_size", "backup_size", "temp_size",
		"total_size", "base_directory",
	}

	for _, key := range expectedKeys {
		if _, exists := info[key]; !exists {
			t.Errorf("Storage info missing key: %s", key)
		}
	}

	// Verify base directory is correct
	if info["base_directory"] != cm.baseDir {
		t.Errorf("Expected base directory %s, got %s", cm.baseDir, info["base_directory"])
	}
}

func TestValidate(t *testing.T) {
	cm, _, cleanup := setupTestConfigManager(t)
	defer cleanup()

	// Test validation before initialization (should fail)
	err := cm.Validate()
	if err == nil {
		t.Error("Validation should fail before initialization")
	}

	// Initialize
	err = cm.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Test validation after initialization (should pass)
	err = cm.Validate()
	if err != nil {
		t.Errorf("Validation should pass after initialization: %v", err)
	}

	// Test validation with corrupted config
	configFile := filepath.Join(cm.configDir, "app.json")
	os.WriteFile(configFile, []byte("invalid json"), 0600)

	err = cm.Validate()
	if err == nil {
		t.Error("Validation should fail with corrupted config")
	}
}

func TestReset(t *testing.T) {
	cm, _, cleanup := setupTestConfigManager(t)
	defer cleanup()

	// Initialize
	err := cm.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Modify config
	config, err := cm.LoadAppConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	originalVersion := config.Version
	config.Version = "modified"
	err = cm.SaveAppConfig(config)
	if err != nil {
		t.Fatalf("Failed to save modified config: %v", err)
	}

	// Reset configuration
	err = cm.Reset()
	if err != nil {
		t.Fatalf("Failed to reset config: %v", err)
	}

	// Verify config is reset to defaults
	resetConfig, err := cm.LoadAppConfig()
	if err != nil {
		t.Fatalf("Failed to load reset config: %v", err)
	}

	if resetConfig.Version != originalVersion {
		t.Errorf("Expected version to be reset to %s, got %s", originalVersion, resetConfig.Version)
	}

	// Verify backup was created
	backupEntries, err := os.ReadDir(cm.baseDir)
	if err != nil {
		t.Fatalf("Failed to read base directory: %v", err)
	}

	backupFound := false
	for _, entry := range backupEntries {
		if entry.IsDir() && len(entry.Name()) > 12 && entry.Name()[:12] == "reset_backup" {
			backupFound = true
			break
		}
	}

	if !backupFound {
		t.Error("Backup directory should be created during reset")
	}
}

func TestDefaultAppConfig(t *testing.T) {
	cm, _, cleanup := setupTestConfigManager(t)
	defer cleanup()

	config := cm.getDefaultAppConfig()

	// Verify default values
	if config.Version == "" {
		t.Error("Default config should have a version")
	}

	if config.AppName == "" {
		t.Error("Default config should have an app name")
	}

	if config.Security.EncryptionEnabled != true {
		t.Error("Encryption should be enabled by default")
	}

	if config.Storage.MaxBackups <= 0 {
		t.Error("Max backups should be positive")
	}

	if config.Logging.Enabled != true {
		t.Error("Logging should be enabled by default")
	}

	// Verify paths are set
	if config.Paths.DataDir == "" {
		t.Error("Data directory path should be set")
	}

	if config.Paths.ConfigDir == "" {
		t.Error("Config directory path should be set")
	}
}

func TestConfigManagerIntegration(t *testing.T) {
	cm, _, cleanup := setupTestConfigManager(t)
	defer cleanup()

	// Test complete workflow
	err := cm.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Verify initialization
	if !cm.IsInitialized() {
		t.Error("ConfigManager should be initialized")
	}

	// Test configuration management
	config, err := cm.LoadAppConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	config.Security.AutoLockTimeout = 45
	err = cm.SaveAppConfig(config)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Test validation
	err = cm.Validate()
	if err != nil {
		t.Errorf("Validation failed: %v", err)
	}

	// Test temp file operations
	tempFile, err := cm.CreateTempFile("integration_test_")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempFile.Close()

	err = cm.CleanupTempDirectory()
	if err != nil {
		t.Errorf("Failed to cleanup temp directory: %v", err)
	}

	// Test storage info
	info, err := cm.GetStorageInfo()
	if err != nil {
		t.Errorf("Failed to get storage info: %v", err)
	}

	if info["total_size"].(int64) < 0 {
		t.Error("Total size should be non-negative")
	}
}