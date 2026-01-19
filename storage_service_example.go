package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// StorageServiceExample demonstrates how to use the StorageService
func StorageServiceExample() {
	fmt.Println("=== StorageService Example ===")

	// Create a temporary directory for this example
	tempDir, err := os.MkdirTemp("", "storage_example_*")
	if err != nil {
		log.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up

	fmt.Printf("Using temporary directory: %s\n", tempDir)

	// 1. Initialize CryptoService and StorageService
	crypto := NewCryptoService("example-master-key-for-demo")
	storage := NewStorageService(tempDir, crypto)

	fmt.Println("\n1. Created StorageService with CryptoService")

	// 2. Create sample account data
	now := time.Now()
	accountData := &AccountData{
		Version: "1.0",
		Accounts: []*KiroAccount{
			{
				ID:               "demo-account-1",
				Email:            "demo@example.com",
				DisplayName:      "Demo Account",
				Avatar:           "https://example.com/avatar.jpg",
				BearerToken:      "demo-bearer-token-12345",
				RefreshToken:     "demo-refresh-token-67890",
				TokenExpiry:      now.Add(24 * time.Hour),
				LoginMethod:      LoginMethodOAuth,
				Provider:         ProviderGoogle,
				SubscriptionType: SubscriptionPro,
				Quota: QuotaInfo{
					Main:   QuotaDetail{Used: 150, Total: 1000},
					Trial:  QuotaDetail{Used: 0, Total: 100},
					Reward: QuotaDetail{Used: 25, Total: 200},
				},
				Tags:      []string{"work", "primary"},
				Notes:     "Primary work account for demo",
				IsActive:  true,
				LastUsed:  now,
				CreatedAt: now.Add(-30 * 24 * time.Hour),
			},
		},
		ActiveAccountID: "demo-account-1",
		Settings:        DefaultAccountSettings(),
		Tags: []Tag{
			{Name: "work", Color: "#007acc", Description: "Work related accounts"},
			{Name: "primary", Color: "#dc3545", Description: "Primary accounts"},
		},
		LastUpdated: now,
	}

	fmt.Println("\n2. Created sample account data")
	fmt.Printf("   - Account: %s (%s)\n", accountData.Accounts[0].DisplayName, accountData.Accounts[0].Email)
	fmt.Printf("   - Quota Usage: %d/%d (%.1f%%)\n", 
		accountData.Accounts[0].Quota.Main.Used,
		accountData.Accounts[0].Quota.Main.Total,
		accountData.Accounts[0].Quota.Main.GetUsagePercentage()*100)

	// 3. Save account data (encrypted)
	err = storage.SaveAccountData(accountData)
	if err != nil {
		log.Fatalf("Failed to save account data: %v", err)
	}

	fmt.Println("\n3. Saved account data (encrypted)")

	// 4. Load account data
	loadedData, err := storage.LoadAccountData()
	if err != nil {
		log.Fatalf("Failed to load account data: %v", err)
	}

	fmt.Println("\n4. Loaded account data successfully")
	fmt.Printf("   - Loaded %d accounts\n", len(loadedData.Accounts))
	fmt.Printf("   - Active account: %s\n", loadedData.ActiveAccountID)

	// 5. Save and load settings
	customSettings := DefaultAccountSettings()
	customSettings.QuotaRefreshInterval = 600 // 10 minutes
	customSettings.AutoRefreshQuota = false
	customSettings.QuotaAlertThreshold = 0.8 // 80%

	err = storage.SaveSettings(&customSettings)
	if err != nil {
		log.Fatalf("Failed to save settings: %v", err)
	}

	loadedSettings, err := storage.LoadSettings()
	if err != nil {
		log.Fatalf("Failed to load settings: %v", err)
	}

	fmt.Println("\n5. Settings management")
	fmt.Printf("   - Quota refresh interval: %d seconds\n", loadedSettings.QuotaRefreshInterval)
	fmt.Printf("   - Auto refresh quota: %v\n", loadedSettings.AutoRefreshQuota)
	fmt.Printf("   - Alert threshold: %.0f%%\n", loadedSettings.QuotaAlertThreshold*100)

	// 6. Save and load tags
	tags := []Tag{
		{Name: "work", Color: "#007acc", Description: "Work accounts"},
		{Name: "personal", Color: "#28a745", Description: "Personal accounts"},
		{Name: "test", Color: "#ffc107", Description: "Test accounts"},
	}

	err = storage.SaveTags(tags)
	if err != nil {
		log.Fatalf("Failed to save tags: %v", err)
	}

	loadedTags, err := storage.LoadTags()
	if err != nil {
		log.Fatalf("Failed to load tags: %v", err)
	}

	fmt.Println("\n6. Tags management")
	fmt.Printf("   - Loaded %d tags:\n", len(loadedTags))
	for _, tag := range loadedTags {
		fmt.Printf("     * %s (%s): %s\n", tag.Name, tag.Color, tag.Description)
	}

	// 7. Create manual backup
	err = storage.CreateBackup()
	if err != nil {
		log.Fatalf("Failed to create backup: %v", err)
	}

	fmt.Println("\n7. Created manual backup")

	// 8. List backups
	backups, err := storage.ListBackups()
	if err != nil {
		log.Fatalf("Failed to list backups: %v", err)
	}

	fmt.Printf("\n8. Available backups (%d):\n", len(backups))
	for i, backup := range backups {
		fmt.Printf("   %d. %s (%.2f KB, %s)\n", 
			i+1, backup.Name, float64(backup.Size)/1024, backup.ModTime.Format("2006-01-02 15:04:05"))
	}

	// 9. Export data
	exportPath := filepath.Join(tempDir, "export_demo.json")
	err = storage.ExportToFile(accountData, exportPath, false, "")
	if err != nil {
		log.Fatalf("Failed to export data: %v", err)
	}

	fmt.Printf("\n9. Exported data to: %s\n", exportPath)

	// 10. Export encrypted data
	encryptedExportPath := filepath.Join(tempDir, "export_encrypted_demo.json")
	err = storage.ExportToFile(accountData, encryptedExportPath, true, "demo-password")
	if err != nil {
		log.Fatalf("Failed to export encrypted data: %v", err)
	}

	fmt.Printf("10. Exported encrypted data to: %s\n", encryptedExportPath)

	// 11. Import encrypted data
	importedData, err := storage.ImportFromFile(encryptedExportPath, true, "demo-password")
	if err != nil {
		log.Fatalf("Failed to import encrypted data: %v", err)
	}

	fmt.Println("11. Imported encrypted data successfully")
	fmt.Printf("    - Imported %d accounts\n", len(importedData.Accounts))

	// 12. Get storage statistics
	stats, err := storage.GetStorageStats()
	if err != nil {
		log.Fatalf("Failed to get storage stats: %v", err)
	}

	fmt.Println("\n12. Storage statistics:")
	fmt.Printf("    - Data directory: %s\n", stats.DataDir)
	fmt.Printf("    - Backup directory: %s\n", stats.BackupDir)
	fmt.Printf("    - Accounts file size: %.2f KB\n", float64(stats.AccountsFileSize)/1024)
	fmt.Printf("    - Number of backups: %d\n", stats.BackupCount)
	fmt.Printf("    - Total backup size: %.2f KB\n", float64(stats.TotalBackupSize)/1024)
	fmt.Printf("    - Last modified: %s\n", stats.LastModified.Format("2006-01-02 15:04:05"))

	// 13. Validate data integrity
	err = storage.ValidateDataIntegrity()
	if err != nil {
		log.Fatalf("Data integrity check failed: %v", err)
	}

	fmt.Println("\n13. Data integrity validation: PASSED")

	fmt.Println("\n=== StorageService Example Complete ===")
	fmt.Println("All operations completed successfully!")
	fmt.Printf("Temporary files created in: %s\n", tempDir)
	fmt.Println("(Files will be automatically cleaned up)")
}

// RunStorageServiceExample runs the storage service example if called directly
func RunStorageServiceExample() {
	StorageServiceExample()
}

// main function for running the example
// Uncomment the following lines to run the example:
// func main() {
// 	RunStorageServiceExample()
// }