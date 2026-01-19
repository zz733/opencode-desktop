package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Example demonstrating ConfigManager usage
func ExampleConfigManager() {
	fmt.Println("=== Kiro Account Manager - Configuration Management Example ===")

	// 1. Create crypto service
	fmt.Println("1. Creating crypto service...")
	crypto := NewCryptoService("example-master-key")
	fmt.Println("   ✓ Crypto service created")

	// 2. Create config manager
	fmt.Println("\n2. Creating configuration manager...")
	configMgr, err := NewConfigManager(crypto)
	if err != nil {
		log.Fatalf("Failed to create config manager: %v", err)
	}
	fmt.Println("   ✓ Configuration manager created")

	// 3. Initialize configuration
	fmt.Println("\n3. Initializing configuration and directory structure...")
	err = configMgr.Initialize()
	if err != nil {
		log.Fatalf("Failed to initialize config manager: %v", err)
	}
	fmt.Println("   ✓ Configuration initialized")

	// 4. Display configuration paths
	fmt.Println("\n4. Configuration paths:")
	paths := configMgr.GetPaths()
	fmt.Printf("   - Base directory: %s\n", paths.BaseDir)
	fmt.Printf("   - Data directory: %s\n", paths.DataDir)
	fmt.Printf("   - Config directory: %s\n", paths.ConfigDir)
	fmt.Printf("   - Logs directory: %s\n", paths.LogsDir)
	fmt.Printf("   - Backup directory: %s\n", paths.BackupDir)
	fmt.Printf("   - Temp directory: %s\n", paths.TempDir)
	fmt.Printf("   - Accounts file: %s\n", paths.AccountsFile)
	fmt.Printf("   - Settings file: %s\n", paths.SettingsFile)
	fmt.Printf("   - Tags file: %s\n", paths.TagsFile)

	// 5. Load and display application configuration
	fmt.Println("\n5. Loading application configuration...")
	appConfig, err := configMgr.LoadAppConfig()
	if err != nil {
		log.Fatalf("Failed to load app config: %v", err)
	}
	fmt.Printf("   - App name: %s\n", appConfig.AppName)
	fmt.Printf("   - Version: %s\n", appConfig.Version)
	fmt.Printf("   - Data version: %s\n", appConfig.DataVersion)
	fmt.Printf("   - Encryption enabled: %v\n", appConfig.Security.EncryptionEnabled)
	fmt.Printf("   - Auto backup enabled: %v\n", appConfig.Storage.AutoBackupEnabled)
	fmt.Printf("   - Max backups: %d\n", appConfig.Storage.MaxBackups)
	fmt.Printf("   - Logging enabled: %v\n", appConfig.Logging.Enabled)
	fmt.Printf("   - Log level: %s\n", appConfig.Logging.Level)

	// 6. Modify and save configuration
	fmt.Println("\n6. Modifying configuration...")
	appConfig.Security.AutoLockTimeout = 45 // Change from default 30 to 45 minutes
	appConfig.Storage.MaxBackups = 15       // Change from default 10 to 15
	appConfig.Logging.Level = "DEBUG"       // Change from INFO to DEBUG

	err = configMgr.SaveAppConfig(appConfig)
	if err != nil {
		log.Fatalf("Failed to save app config: %v", err)
	}
	fmt.Println("   ✓ Configuration updated and saved")

	// 7. Validate configuration
	fmt.Println("\n7. Validating configuration...")
	err = configMgr.Validate()
	if err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}
	fmt.Println("   ✓ Configuration is valid")

	// 8. Create and manage temporary files
	fmt.Println("\n8. Testing temporary file management...")
	tempFile, err := configMgr.CreateTempFile("example_")
	if err != nil {
		log.Fatalf("Failed to create temp file: %v", err)
	}
	
	tempPath := tempFile.Name()
	fmt.Printf("   - Created temp file: %s\n", tempPath)
	
	// Write some data to temp file
	_, err = tempFile.WriteString("This is a temporary file for testing.")
	if err != nil {
		log.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	// Create another temp file
	tempFile2, err := configMgr.CreateTempFile("example2_")
	if err != nil {
		log.Fatalf("Failed to create second temp file: %v", err)
	}
	tempFile2.Close()
	fmt.Printf("   - Created second temp file: %s\n", tempFile2.Name())

	// Cleanup temp directory
	fmt.Println("   - Cleaning up temporary files...")
	err = configMgr.CleanupTempDirectory()
	if err != nil {
		log.Fatalf("Failed to cleanup temp directory: %v", err)
	}
	fmt.Println("   ✓ Temporary files cleaned up")

	// 9. Get storage information
	fmt.Println("\n9. Storage information:")
	storageInfo, err := configMgr.GetStorageInfo()
	if err != nil {
		log.Fatalf("Failed to get storage info: %v", err)
	}

	for key, value := range storageInfo {
		switch key {
		case "data_size", "config_size", "logs_size", "backup_size", "temp_size", "total_size":
			if size, ok := value.(int64); ok {
				fmt.Printf("   - %s: %.2f KB\n", key, float64(size)/1024)
			}
		case "base_directory":
			fmt.Printf("   - %s: %s\n", key, value)
		}
	}

	// 10. Test directory size calculation
	fmt.Println("\n10. Testing directory size calculation...")
	
	// Create some test files in data directory
	testFile1 := filepath.Join(configMgr.GetDataDirectory(), "test1.txt")
	testFile2 := filepath.Join(configMgr.GetDataDirectory(), "test2.txt")
	
	err = os.WriteFile(testFile1, []byte("Hello, World! This is test file 1."), 0600)
	if err != nil {
		log.Fatalf("Failed to create test file 1: %v", err)
	}
	
	err = os.WriteFile(testFile2, []byte("This is test file 2 with more content for testing."), 0600)
	if err != nil {
		log.Fatalf("Failed to create test file 2: %v", err)
	}

	dataSize, err := configMgr.GetDirectorySize(configMgr.GetDataDirectory())
	if err != nil {
		log.Fatalf("Failed to get data directory size: %v", err)
	}
	fmt.Printf("   - Data directory size: %.2f KB\n", float64(dataSize)/1024)

	// Clean up test files
	os.Remove(testFile1)
	os.Remove(testFile2)

	// 11. Demonstrate configuration reset (with backup)
	fmt.Println("\n11. Testing configuration reset...")
	
	// First, modify config significantly
	appConfig.Version = "modified-version"
	appConfig.AppName = "Modified App Name"
	err = configMgr.SaveAppConfig(appConfig)
	if err != nil {
		log.Fatalf("Failed to save modified config: %v", err)
	}
	fmt.Println("   - Configuration modified")

	// Reset to defaults
	err = configMgr.Reset()
	if err != nil {
		log.Fatalf("Failed to reset configuration: %v", err)
	}
	fmt.Println("   ✓ Configuration reset to defaults (backup created)")

	// Verify reset worked
	resetConfig, err := configMgr.LoadAppConfig()
	if err != nil {
		log.Fatalf("Failed to load reset config: %v", err)
	}
	fmt.Printf("   - App name after reset: %s\n", resetConfig.AppName)
	fmt.Printf("   - Version after reset: %s\n", resetConfig.Version)

	// 12. Final validation
	fmt.Println("\n12. Final validation...")
	err = configMgr.Validate()
	if err != nil {
		log.Fatalf("Final validation failed: %v", err)
	}
	fmt.Println("   ✓ All configuration is valid")

	fmt.Println("\n=== Configuration Management Example Completed Successfully ===")
}

// Example of integrating ConfigManager with existing services
func ExampleConfigManagerIntegration() {
	fmt.Println("=== ConfigManager Integration Example ===")

	// 1. Initialize ConfigManager
	crypto := NewCryptoService("integration-example-key")
	configMgr, err := NewConfigManager(crypto)
	if err != nil {
		log.Fatalf("Failed to create config manager: %v", err)
	}

	err = configMgr.Initialize()
	if err != nil {
		log.Fatalf("Failed to initialize config manager: %v", err)
	}

	// 2. Create StorageService using ConfigManager paths
	fmt.Println("1. Creating StorageService with ConfigManager...")
	dataDir := configMgr.GetDataDirectory()
	storage := NewStorageService(dataDir, crypto)
	fmt.Printf("   ✓ StorageService created with data directory: %s\n", dataDir)

	// 3. Create AccountManager
	fmt.Println("\n2. Creating AccountManager...")
	accountMgr := NewAccountManager(storage, crypto)
	fmt.Println("   ✓ AccountManager created")

	// 4. Test the integration by creating a sample account
	fmt.Println("\n3. Testing integration with sample account...")
	
	sampleAccount := &KiroAccount{
		Email:            "integration@example.com",
		DisplayName:      "Integration Test Account",
		BearerToken:      "sample-bearer-token",
		LoginMethod:      LoginMethodToken,
		SubscriptionType: SubscriptionFree,
		Quota: QuotaInfo{
			Main:   QuotaDetail{Used: 100, Total: 1000},
			Trial:  QuotaDetail{Used: 0, Total: 100},
			Reward: QuotaDetail{Used: 25, Total: 200},
		},
		Tags:      []string{"integration", "test"},
		Notes:     "Created during ConfigManager integration example",
		CreatedAt: time.Now(),
	}

	err = accountMgr.AddAccount(sampleAccount)
	if err != nil {
		log.Fatalf("Failed to add sample account: %v", err)
	}
	fmt.Println("   ✓ Sample account created and stored")

	// 5. Verify the account was stored in the correct location
	accounts := accountMgr.ListAccounts()
	fmt.Printf("   - Total accounts: %d\n", len(accounts))
	if len(accounts) > 0 {
		fmt.Printf("   - First account email: %s\n", accounts[0].Email)
		fmt.Printf("   - Storage location: %s\n", filepath.Join(dataDir, "accounts.json.enc"))
	}

	// 6. Show storage statistics
	fmt.Println("\n4. Storage statistics:")
	storageInfo, err := configMgr.GetStorageInfo()
	if err != nil {
		log.Fatalf("Failed to get storage info: %v", err)
	}

	if totalSize, ok := storageInfo["total_size"].(int64); ok {
		fmt.Printf("   - Total storage used: %.2f KB\n", float64(totalSize)/1024)
	}

	fmt.Println("\n=== Integration Example Completed ===")
}

// Run examples if this file is executed directly
func init() {
	// This will run the examples when the package is imported
	// In a real application, you would call these functions explicitly
}