package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// setupTestStorageService creates a test storage service with temporary directory
func setupTestStorageService(t *testing.T) (*StorageService, string, func()) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "storage_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create crypto service with test key
	crypto := NewCryptoService("test-master-key-for-storage-service")

	// Create storage service
	storage := NewStorageService(tempDir, crypto)

	// Return cleanup function
	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return storage, tempDir, cleanup
}

// createTestAccountData creates sample account data for testing
func createTestAccountData() *AccountData {
	now := time.Now()
	return &AccountData{
		Version: "1.0",
		Accounts: []*KiroAccount{
			{
				ID:               "test-account-1",
				Email:            "test1@example.com",
				DisplayName:      "Test Account 1",
				Avatar:           "https://example.com/avatar1.jpg",
				BearerToken:      "test-token-1",
				RefreshToken:     "test-refresh-1",
				TokenExpiry:      now.Add(24 * time.Hour),
				LoginMethod:      LoginMethodOAuth,
				Provider:         ProviderGoogle,
				SubscriptionType: SubscriptionPro,
				Quota: QuotaInfo{
					Main:   QuotaDetail{Used: 100, Total: 1000},
					Trial:  QuotaDetail{Used: 0, Total: 100},
					Reward: QuotaDetail{Used: 50, Total: 200},
				},
				Tags:      []string{"work", "primary"},
				Notes:     "Primary work account",
				IsActive:  true,
				LastUsed:  now,
				CreatedAt: now.Add(-30 * 24 * time.Hour),
			},
			{
				ID:               "test-account-2",
				Email:            "test2@example.com",
				DisplayName:      "Test Account 2",
				TokenExpiry:      now.Add(12 * time.Hour),
				LoginMethod:      LoginMethodToken,
				SubscriptionType: SubscriptionFree,
				Quota: QuotaInfo{
					Main:   QuotaDetail{Used: 50, Total: 100},
					Trial:  QuotaDetail{Used: 10, Total: 50},
					Reward: QuotaDetail{Used: 0, Total: 0},
				},
				Tags:      []string{"personal"},
				IsActive:  false,
				LastUsed:  now.Add(-2 * time.Hour),
				CreatedAt: now.Add(-7 * 24 * time.Hour),
			},
		},
		ActiveAccountID: "test-account-1",
		Settings:        DefaultAccountSettings(),
		Tags: []Tag{
			{Name: "work", Color: "#007acc", Description: "Work related accounts"},
			{Name: "personal", Color: "#28a745", Description: "Personal accounts"},
			{Name: "primary", Color: "#dc3545", Description: "Primary accounts"},
		},
		LastUpdated: now,
	}
}

func TestNewStorageService(t *testing.T) {
	storage, tempDir, cleanup := setupTestStorageService(t)
	defer cleanup()

	// Verify storage service is created correctly
	if storage.dataDir != tempDir {
		t.Errorf("Expected dataDir %s, got %s", tempDir, storage.dataDir)
	}

	expectedBackupDir := filepath.Join(tempDir, "backups")
	if storage.backupDir != expectedBackupDir {
		t.Errorf("Expected backupDir %s, got %s", expectedBackupDir, storage.backupDir)
	}

	if storage.maxBackups != 10 {
		t.Errorf("Expected maxBackups 10, got %d", storage.maxBackups)
	}

	// Verify directories are created
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Error("Data directory was not created")
	}

	if _, err := os.Stat(expectedBackupDir); os.IsNotExist(err) {
		t.Error("Backup directory was not created")
	}
}

func TestSaveAndLoadAccountData(t *testing.T) {
	storage, _, cleanup := setupTestStorageService(t)
	defer cleanup()

	// Create test data
	testData := createTestAccountData()

	// Test saving account data
	err := storage.SaveAccountData(testData)
	if err != nil {
		t.Fatalf("Failed to save account data: %v", err)
	}

	// Test loading account data
	loadedData, err := storage.LoadAccountData()
	if err != nil {
		t.Fatalf("Failed to load account data: %v", err)
	}

	// Verify data integrity
	if loadedData.Version != testData.Version {
		t.Errorf("Expected version %s, got %s", testData.Version, loadedData.Version)
	}

	if len(loadedData.Accounts) != len(testData.Accounts) {
		t.Errorf("Expected %d accounts, got %d", len(testData.Accounts), len(loadedData.Accounts))
	}

	if loadedData.ActiveAccountID != testData.ActiveAccountID {
		t.Errorf("Expected active account ID %s, got %s", testData.ActiveAccountID, loadedData.ActiveAccountID)
	}

	// Verify first account details
	if len(loadedData.Accounts) > 0 {
		account := loadedData.Accounts[0]
		expectedAccount := testData.Accounts[0]

		if account.ID != expectedAccount.ID {
			t.Errorf("Expected account ID %s, got %s", expectedAccount.ID, account.ID)
		}

		if account.Email != expectedAccount.Email {
			t.Errorf("Expected email %s, got %s", expectedAccount.Email, account.Email)
		}

		if account.DisplayName != expectedAccount.DisplayName {
			t.Errorf("Expected display name %s, got %s", expectedAccount.DisplayName, account.DisplayName)
		}
	}
}

func TestSaveAccountDataNil(t *testing.T) {
	storage, _, cleanup := setupTestStorageService(t)
	defer cleanup()

	// Test saving nil data
	err := storage.SaveAccountData(nil)
	if err == nil {
		t.Error("Expected error when saving nil account data")
	}
}

func TestLoadAccountDataNotFound(t *testing.T) {
	storage, _, cleanup := setupTestStorageService(t)
	defer cleanup()

	// Test loading when file doesn't exist
	_, err := storage.LoadAccountData()
	if err == nil {
		t.Error("Expected error when loading non-existent account data")
	}
}

func TestSerializeDeserializeAccountData(t *testing.T) {
	storage, _, cleanup := setupTestStorageService(t)
	defer cleanup()

	testData := createTestAccountData()

	// Test serialization
	jsonData, err := storage.SerializeAccountData(testData)
	if err != nil {
		t.Fatalf("Failed to serialize account data: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("Serialized data is empty")
	}

	// Test deserialization
	deserializedData, err := storage.DeserializeAccountData(jsonData)
	if err != nil {
		t.Fatalf("Failed to deserialize account data: %v", err)
	}

	// Verify data integrity
	if deserializedData.Version != testData.Version {
		t.Errorf("Expected version %s, got %s", testData.Version, deserializedData.Version)
	}

	if len(deserializedData.Accounts) != len(testData.Accounts) {
		t.Errorf("Expected %d accounts, got %d", len(testData.Accounts), len(deserializedData.Accounts))
	}
}

func TestSerializeNilData(t *testing.T) {
	storage, _, cleanup := setupTestStorageService(t)
	defer cleanup()

	// Test serializing nil data
	_, err := storage.SerializeAccountData(nil)
	if err == nil {
		t.Error("Expected error when serializing nil data")
	}
}

func TestDeserializeEmptyData(t *testing.T) {
	storage, _, cleanup := setupTestStorageService(t)
	defer cleanup()

	// Test deserializing empty data
	_, err := storage.DeserializeAccountData([]byte{})
	if err == nil {
		t.Error("Expected error when deserializing empty data")
	}
}

func TestBackupCreationAndCleanup(t *testing.T) {
	storage, _, cleanup := setupTestStorageService(t)
	defer cleanup()

	testData := createTestAccountData()

	// Save account data (this should create a backup)
	err := storage.SaveAccountData(testData)
	if err != nil {
		t.Fatalf("Failed to save account data: %v", err)
	}

	// Check if backup was created
	backups, err := storage.ListBackups()
	if err != nil {
		t.Fatalf("Failed to list backups: %v", err)
	}

	if len(backups) == 0 {
		t.Error("No backup was created")
	}

	// Wait a bit to ensure different timestamp
	time.Sleep(100 * time.Millisecond)

	// Test manual backup creation
	err = storage.CreateBackup()
	if err != nil {
		t.Fatalf("Failed to create manual backup: %v", err)
	}

	// Check if additional backup was created
	backupsAfter, err := storage.ListBackups()
	if err != nil {
		t.Fatalf("Failed to list backups after manual creation: %v", err)
	}

	if len(backupsAfter) <= len(backups) {
		t.Errorf("Manual backup was not created. Before: %d, After: %d", len(backups), len(backupsAfter))
	}
}

func TestRestoreFromBackup(t *testing.T) {
	storage, _, cleanup := setupTestStorageService(t)
	defer cleanup()

	testData := createTestAccountData()
	originalDisplayName := testData.Accounts[0].DisplayName

	// Save initial data
	err := storage.SaveAccountData(testData)
	if err != nil {
		t.Fatalf("Failed to save account data: %v", err)
	}

	// Get backup list
	backups, err := storage.ListBackups()
	if err != nil {
		t.Fatalf("Failed to list backups: %v", err)
	}

	if len(backups) == 0 {
		t.Fatal("No backups available for restore test")
	}

	// Wait a bit to ensure different timestamp
	time.Sleep(100 * time.Millisecond)

	// Modify the data and save again
	testData.Accounts[0].DisplayName = "Modified Account"
	err = storage.SaveAccountData(testData)
	if err != nil {
		t.Fatalf("Failed to save modified data: %v", err)
	}

	// Verify the modification was saved
	modifiedData, err := storage.LoadAccountData()
	if err != nil {
		t.Fatalf("Failed to load modified data: %v", err)
	}

	if modifiedData.Accounts[0].DisplayName != "Modified Account" {
		t.Fatal("Modification was not saved properly")
	}

	// Restore from the first backup (original data)
	err = storage.RestoreFromBackup(backups[0].Path)
	if err != nil {
		t.Fatalf("Failed to restore from backup: %v", err)
	}

	// Load restored data
	restoredData, err := storage.LoadAccountData()
	if err != nil {
		t.Fatalf("Failed to load restored data: %v", err)
	}

	// Verify restoration (should have the original name, not the modification)
	if restoredData.Accounts[0].DisplayName != originalDisplayName {
		t.Errorf("Data was not properly restored from backup. Expected: %s, Got: %s", 
			originalDisplayName, restoredData.Accounts[0].DisplayName)
	}
}

func TestSaveAndLoadSettings(t *testing.T) {
	storage, _, cleanup := setupTestStorageService(t)
	defer cleanup()

	// Create test settings
	testSettings := DefaultAccountSettings()
	testSettings.QuotaRefreshInterval = 600
	testSettings.AutoRefreshQuota = false
	testSettings.QuotaAlertThreshold = 0.8

	// Test saving settings
	err := storage.SaveSettings(&testSettings)
	if err != nil {
		t.Fatalf("Failed to save settings: %v", err)
	}

	// Test loading settings
	loadedSettings, err := storage.LoadSettings()
	if err != nil {
		t.Fatalf("Failed to load settings: %v", err)
	}

	// Verify settings
	if loadedSettings.QuotaRefreshInterval != testSettings.QuotaRefreshInterval {
		t.Errorf("Expected QuotaRefreshInterval %d, got %d", 
			testSettings.QuotaRefreshInterval, loadedSettings.QuotaRefreshInterval)
	}

	if loadedSettings.AutoRefreshQuota != testSettings.AutoRefreshQuota {
		t.Errorf("Expected AutoRefreshQuota %v, got %v", 
			testSettings.AutoRefreshQuota, loadedSettings.AutoRefreshQuota)
	}

	if loadedSettings.QuotaAlertThreshold != testSettings.QuotaAlertThreshold {
		t.Errorf("Expected QuotaAlertThreshold %f, got %f", 
			testSettings.QuotaAlertThreshold, loadedSettings.QuotaAlertThreshold)
	}
}

func TestLoadSettingsDefault(t *testing.T) {
	storage, _, cleanup := setupTestStorageService(t)
	defer cleanup()

	// Test loading settings when file doesn't exist (should return defaults)
	settings, err := storage.LoadSettings()
	if err != nil {
		t.Fatalf("Failed to load default settings: %v", err)
	}

	defaultSettings := DefaultAccountSettings()
	if settings.QuotaRefreshInterval != defaultSettings.QuotaRefreshInterval {
		t.Errorf("Default settings not loaded correctly")
	}
}

func TestSaveAndLoadTags(t *testing.T) {
	storage, _, cleanup := setupTestStorageService(t)
	defer cleanup()

	// Create test tags
	testTags := []Tag{
		{Name: "work", Color: "#007acc", Description: "Work accounts"},
		{Name: "personal", Color: "#28a745", Description: "Personal accounts"},
		{Name: "test", Color: "#ffc107", Description: "Test accounts"},
	}

	// Test saving tags
	err := storage.SaveTags(testTags)
	if err != nil {
		t.Fatalf("Failed to save tags: %v", err)
	}

	// Test loading tags
	loadedTags, err := storage.LoadTags()
	if err != nil {
		t.Fatalf("Failed to load tags: %v", err)
	}

	// Verify tags
	if len(loadedTags) != len(testTags) {
		t.Errorf("Expected %d tags, got %d", len(testTags), len(loadedTags))
	}

	for i, tag := range loadedTags {
		if i < len(testTags) {
			if tag.Name != testTags[i].Name {
				t.Errorf("Expected tag name %s, got %s", testTags[i].Name, tag.Name)
			}
			if tag.Color != testTags[i].Color {
				t.Errorf("Expected tag color %s, got %s", testTags[i].Color, tag.Color)
			}
		}
	}
}

func TestLoadTagsEmpty(t *testing.T) {
	storage, _, cleanup := setupTestStorageService(t)
	defer cleanup()

	// Test loading tags when file doesn't exist (should return empty array)
	tags, err := storage.LoadTags()
	if err != nil {
		t.Fatalf("Failed to load empty tags: %v", err)
	}

	if len(tags) != 0 {
		t.Errorf("Expected empty tags array, got %d tags", len(tags))
	}
}

func TestExportAndImportToFile(t *testing.T) {
	storage, tempDir, cleanup := setupTestStorageService(t)
	defer cleanup()

	testData := createTestAccountData()
	exportPath := filepath.Join(tempDir, "export_test.json")

	// Test export without encryption
	err := storage.ExportToFile(testData, exportPath, false, "")
	if err != nil {
		t.Fatalf("Failed to export data: %v", err)
	}

	// Test import without decryption
	importedData, err := storage.ImportFromFile(exportPath, false, "")
	if err != nil {
		t.Fatalf("Failed to import data: %v", err)
	}

	// Verify imported data
	if importedData.Version != testData.Version {
		t.Errorf("Expected version %s, got %s", testData.Version, importedData.Version)
	}

	if len(importedData.Accounts) != len(testData.Accounts) {
		t.Errorf("Expected %d accounts, got %d", len(testData.Accounts), len(importedData.Accounts))
	}
}

func TestExportAndImportWithEncryption(t *testing.T) {
	storage, tempDir, cleanup := setupTestStorageService(t)
	defer cleanup()

	testData := createTestAccountData()
	exportPath := filepath.Join(tempDir, "export_encrypted_test.json")
	password := "test-password-123"

	// Test export with encryption
	err := storage.ExportToFile(testData, exportPath, true, password)
	if err != nil {
		t.Fatalf("Failed to export encrypted data: %v", err)
	}

	// Test import with decryption
	importedData, err := storage.ImportFromFile(exportPath, true, password)
	if err != nil {
		t.Fatalf("Failed to import encrypted data: %v", err)
	}

	// Verify imported data
	if importedData.Version != testData.Version {
		t.Errorf("Expected version %s, got %s", testData.Version, importedData.Version)
	}

	if len(importedData.Accounts) != len(testData.Accounts) {
		t.Errorf("Expected %d accounts, got %d", len(testData.Accounts), len(importedData.Accounts))
	}
}

func TestGetStorageStats(t *testing.T) {
	storage, _, cleanup := setupTestStorageService(t)
	defer cleanup()

	testData := createTestAccountData()

	// Save some data first
	err := storage.SaveAccountData(testData)
	if err != nil {
		t.Fatalf("Failed to save account data: %v", err)
	}

	// Get storage stats
	stats, err := storage.GetStorageStats()
	if err != nil {
		t.Fatalf("Failed to get storage stats: %v", err)
	}

	// Verify stats
	if stats.DataDir != storage.dataDir {
		t.Errorf("Expected data dir %s, got %s", storage.dataDir, stats.DataDir)
	}

	if stats.BackupDir != storage.backupDir {
		t.Errorf("Expected backup dir %s, got %s", storage.backupDir, stats.BackupDir)
	}

	if stats.AccountsFileSize <= 0 {
		t.Error("Expected positive accounts file size")
	}

	if stats.BackupCount <= 0 {
		t.Error("Expected at least one backup")
	}
}

func TestValidateDataIntegrity(t *testing.T) {
	storage, _, cleanup := setupTestStorageService(t)
	defer cleanup()

	// Test with no data (should pass)
	err := storage.ValidateDataIntegrity()
	if err != nil {
		t.Errorf("Validation should pass with no data: %v", err)
	}

	// Save valid data
	testData := createTestAccountData()
	err = storage.SaveAccountData(testData)
	if err != nil {
		t.Fatalf("Failed to save test data: %v", err)
	}

	// Test with valid data (should pass)
	err = storage.ValidateDataIntegrity()
	if err != nil {
		t.Errorf("Validation should pass with valid data: %v", err)
	}
}

func TestGetDirectories(t *testing.T) {
	storage, tempDir, cleanup := setupTestStorageService(t)
	defer cleanup()

	// Test GetDataDirectory
	if storage.GetDataDirectory() != tempDir {
		t.Errorf("Expected data directory %s, got %s", tempDir, storage.GetDataDirectory())
	}

	// Test GetBackupDirectory
	expectedBackupDir := filepath.Join(tempDir, "backups")
	if storage.GetBackupDirectory() != expectedBackupDir {
		t.Errorf("Expected backup directory %s, got %s", expectedBackupDir, storage.GetBackupDirectory())
	}
}