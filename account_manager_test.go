package main

import (
	"os"
	"testing"
	"time"
)

// setupTestAccountManager creates a test account manager with temporary storage
func setupTestAccountManager(t *testing.T) (*AccountManager, string) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "kiro_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Initialize services
	crypto := NewCryptoService("test-master-key")
	storage := NewStorageService(tempDir, crypto)
	
	// Create account manager
	am := NewAccountManager(storage, crypto)
	
	return am, tempDir
}

// cleanupTestAccountManager removes temporary test data
func cleanupTestAccountManager(tempDir string) {
	os.RemoveAll(tempDir)
}

func TestAccountManager_AddAccount(t *testing.T) {
	am, tempDir := setupTestAccountManager(t)
	defer cleanupTestAccountManager(tempDir)

	// Create test account
	account := &KiroAccount{
		Email:            "test@example.com",
		DisplayName:      "Test User",
		BearerToken:      "test-token",
		TokenExpiry:      time.Now().Add(24 * time.Hour),
		LoginMethod:      LoginMethodToken,
		SubscriptionType: SubscriptionFree,
		Quota: QuotaInfo{
			Main:   QuotaDetail{Used: 10, Total: 100},
			Trial:  QuotaDetail{Used: 0, Total: 50},
			Reward: QuotaDetail{Used: 5, Total: 25},
		},
		Tags: []string{"test"},
	}

	// Add account
	err := am.AddAccount(account)
	if err != nil {
		t.Fatalf("Failed to add account: %v", err)
	}

	// Verify account was added
	accounts := am.ListAccounts()
	if len(accounts) != 1 {
		t.Fatalf("Expected 1 account, got %d", len(accounts))
	}

	// Verify account details
	addedAccount := accounts[0]
	if addedAccount.Email != account.Email {
		t.Errorf("Email mismatch: got %s, want %s", addedAccount.Email, account.Email)
	}
	if addedAccount.DisplayName != account.DisplayName {
		t.Errorf("DisplayName mismatch: got %s, want %s", addedAccount.DisplayName, account.DisplayName)
	}
	if !addedAccount.IsActive {
		t.Error("First account should be active")
	}
	if addedAccount.ID == "" {
		t.Error("Account ID should be generated")
	}
}

func TestAccountManager_AddDuplicateAccount(t *testing.T) {
	am, tempDir := setupTestAccountManager(t)
	defer cleanupTestAccountManager(tempDir)

	// Create test account
	account1 := &KiroAccount{
		Email:            "test@example.com",
		DisplayName:      "Test User 1",
		BearerToken:      "test-token-1",
		TokenExpiry:      time.Now().Add(24 * time.Hour),
		LoginMethod:      LoginMethodToken,
		SubscriptionType: SubscriptionFree,
	}

	account2 := &KiroAccount{
		Email:            "test@example.com", // Same email
		DisplayName:      "Test User 2",
		BearerToken:      "test-token-2",
		TokenExpiry:      time.Now().Add(24 * time.Hour),
		LoginMethod:      LoginMethodToken,
		SubscriptionType: SubscriptionFree,
	}

	// Add first account
	err := am.AddAccount(account1)
	if err != nil {
		t.Fatalf("Failed to add first account: %v", err)
	}

	// Try to add duplicate account
	err = am.AddAccount(account2)
	if err == nil {
		t.Error("Expected error when adding duplicate account")
	}

	// Verify only one account exists
	accounts := am.ListAccounts()
	if len(accounts) != 1 {
		t.Fatalf("Expected 1 account, got %d", len(accounts))
	}
}

func TestAccountManager_RemoveAccount(t *testing.T) {
	am, tempDir := setupTestAccountManager(t)
	defer cleanupTestAccountManager(tempDir)

	// Add test accounts
	account1 := &KiroAccount{
		Email:            "test1@example.com",
		DisplayName:      "Test User 1",
		BearerToken:      "test-token-1",
		TokenExpiry:      time.Now().Add(24 * time.Hour),
		LoginMethod:      LoginMethodToken,
		SubscriptionType: SubscriptionFree,
	}

	account2 := &KiroAccount{
		Email:            "test2@example.com",
		DisplayName:      "Test User 2",
		BearerToken:      "test-token-2",
		TokenExpiry:      time.Now().Add(24 * time.Hour),
		LoginMethod:      LoginMethodToken,
		SubscriptionType: SubscriptionFree,
	}

	am.AddAccount(account1)
	am.AddAccount(account2)

	// Get account IDs
	accounts := am.ListAccounts()
	if len(accounts) != 2 {
		t.Fatalf("Expected 2 accounts, got %d", len(accounts))
	}

	// Remove one account
	err := am.RemoveAccount(accounts[1].ID)
	if err != nil {
		t.Fatalf("Failed to remove account: %v", err)
	}

	// Verify account was removed
	remainingAccounts := am.ListAccounts()
	if len(remainingAccounts) != 1 {
		t.Fatalf("Expected 1 account after removal, got %d", len(remainingAccounts))
	}
}

func TestAccountManager_UpdateAccount(t *testing.T) {
	am, tempDir := setupTestAccountManager(t)
	defer cleanupTestAccountManager(tempDir)

	// Add test account
	account := &KiroAccount{
		Email:            "test@example.com",
		DisplayName:      "Test User",
		BearerToken:      "test-token",
		TokenExpiry:      time.Now().Add(24 * time.Hour),
		LoginMethod:      LoginMethodToken,
		SubscriptionType: SubscriptionFree,
		Tags:             []string{"original"},
		Notes:            "Original notes",
	}

	am.AddAccount(account)
	accounts := am.ListAccounts()
	accountID := accounts[0].ID

	// Update account
	updates := map[string]interface{}{
		"displayName": "Updated User",
		"tags":        []string{"updated", "test"},
		"notes":       "Updated notes",
	}

	err := am.UpdateAccount(accountID, updates)
	if err != nil {
		t.Fatalf("Failed to update account: %v", err)
	}

	// Verify updates
	updatedAccount, err := am.GetAccount(accountID)
	if err != nil {
		t.Fatalf("Failed to get updated account: %v", err)
	}

	if updatedAccount.DisplayName != "Updated User" {
		t.Errorf("DisplayName not updated: got %s, want %s", updatedAccount.DisplayName, "Updated User")
	}
	if len(updatedAccount.Tags) != 2 || updatedAccount.Tags[0] != "updated" {
		t.Errorf("Tags not updated correctly: got %v", updatedAccount.Tags)
	}
	if updatedAccount.Notes != "Updated notes" {
		t.Errorf("Notes not updated: got %s, want %s", updatedAccount.Notes, "Updated notes")
	}
}

func TestAccountManager_BatchOperations(t *testing.T) {
	am, tempDir := setupTestAccountManager(t)
	defer cleanupTestAccountManager(tempDir)

	// Add test accounts
	accounts := []*KiroAccount{
		{
			Email:            "test1@example.com",
			DisplayName:      "Test User 1",
			BearerToken:      "test-token-1",
			RefreshToken:     "refresh-token-1",
			TokenExpiry:      time.Now().Add(24 * time.Hour),
			LoginMethod:      LoginMethodToken,
			SubscriptionType: SubscriptionFree,
			Tags:             []string{"original"},
		},
		{
			Email:            "test2@example.com",
			DisplayName:      "Test User 2",
			BearerToken:      "test-token-2",
			RefreshToken:     "refresh-token-2",
			TokenExpiry:      time.Now().Add(24 * time.Hour),
			LoginMethod:      LoginMethodToken,
			SubscriptionType: SubscriptionFree,
			Tags:             []string{"original"},
		},
	}

	for _, account := range accounts {
		am.AddAccount(account)
	}

	allAccounts := am.ListAccounts()
	accountIDs := make([]string, len(allAccounts))
	for i, acc := range allAccounts {
		accountIDs[i] = acc.ID
	}

	// Test batch add tags
	newTags := []string{"batch", "test"}
	err := am.BatchAddTags(accountIDs, newTags)
	if err != nil {
		t.Fatalf("Failed to batch add tags: %v", err)
	}

	// Verify tags were added
	for _, id := range accountIDs {
		account, err := am.GetAccount(id)
		if err != nil {
			t.Fatalf("Failed to get account %s: %v", id, err)
		}
		
		hasNewTags := false
		for _, tag := range newTags {
			if account.HasTag(tag) {
				hasNewTags = true
				break
			}
		}
		if !hasNewTags {
			t.Errorf("Account %s missing new tags", id)
		}
	}

	// Test batch delete (delete one account)
	err = am.BatchDeleteAccounts([]string{accountIDs[0]})
	if err != nil {
		t.Fatalf("Failed to batch delete accounts: %v", err)
	}

	// Verify account was deleted
	remainingAccounts := am.ListAccounts()
	if len(remainingAccounts) != 1 {
		t.Fatalf("Expected 1 account after batch delete, got %d", len(remainingAccounts))
	}
}

func TestAccountManager_ExportImport(t *testing.T) {
	am, tempDir := setupTestAccountManager(t)
	defer cleanupTestAccountManager(tempDir)

	// Add test account
	account := &KiroAccount{
		Email:            "test@example.com",
		DisplayName:      "Test User",
		BearerToken:      "test-token",
		TokenExpiry:      time.Now().Add(24 * time.Hour),
		LoginMethod:      LoginMethodToken,
		SubscriptionType: SubscriptionFree,
		Tags:             []string{"export", "test"},
		Notes:            "Test account for export",
	}

	am.AddAccount(account)

	// Export accounts
	exportData, err := am.ExportAccounts("")
	if err != nil {
		t.Fatalf("Failed to export accounts: %v", err)
	}

	// Create new account manager for import test
	am2, tempDir2 := setupTestAccountManager(t)
	defer cleanupTestAccountManager(tempDir2)

	// Import accounts
	err = am2.ImportAccounts(exportData, "")
	if err != nil {
		t.Fatalf("Failed to import accounts: %v", err)
	}

	// Verify imported accounts
	importedAccounts := am2.ListAccounts()
	if len(importedAccounts) != 1 {
		t.Fatalf("Expected 1 imported account, got %d", len(importedAccounts))
	}

	importedAccount := importedAccounts[0]
	if importedAccount.Email != account.Email {
		t.Errorf("Email mismatch after import: got %s, want %s", importedAccount.Email, account.Email)
	}
	if importedAccount.DisplayName != account.DisplayName {
		t.Errorf("DisplayName mismatch after import: got %s, want %s", importedAccount.DisplayName, account.DisplayName)
	}
}

func TestAccountManager_GetAccountStats(t *testing.T) {
	am, tempDir := setupTestAccountManager(t)
	defer cleanupTestAccountManager(tempDir)

	// Add test accounts with different subscription types
	accounts := []*KiroAccount{
		{
			Email:            "free@example.com",
			DisplayName:      "Free User",
			BearerToken:      "token-1",
			TokenExpiry:      time.Now().Add(24 * time.Hour),
			LoginMethod:      LoginMethodToken,
			SubscriptionType: SubscriptionFree,
		},
		{
			Email:            "pro@example.com",
			DisplayName:      "Pro User",
			BearerToken:      "token-2",
			TokenExpiry:      time.Now().Add(24 * time.Hour),
			LoginMethod:      LoginMethodOAuth,
			SubscriptionType: SubscriptionPro,
		},
		{
			Email:            "expired@example.com",
			DisplayName:      "Expired User",
			BearerToken:      "token-3",
			TokenExpiry:      time.Now().Add(-1 * time.Hour), // Expired
			LoginMethod:      LoginMethodToken,
			SubscriptionType: SubscriptionFree,
		},
	}

	for _, account := range accounts {
		am.AddAccount(account)
	}

	// Get stats
	stats := am.GetAccountStats()

	// Verify stats
	if stats["totalAccounts"] != 3 {
		t.Errorf("Expected 3 total accounts, got %v", stats["totalAccounts"])
	}

	subscriptionTypes := stats["subscriptionTypes"].(map[string]int)
	if subscriptionTypes["free"] != 2 {
		t.Errorf("Expected 2 free accounts, got %d", subscriptionTypes["free"])
	}
	if subscriptionTypes["pro"] != 1 {
		t.Errorf("Expected 1 pro account, got %d", subscriptionTypes["pro"])
	}

	loginMethods := stats["loginMethods"].(map[string]int)
	if loginMethods["token"] != 2 {
		t.Errorf("Expected 2 token accounts, got %d", loginMethods["token"])
	}
	if loginMethods["oauth"] != 1 {
		t.Errorf("Expected 1 oauth account, got %d", loginMethods["oauth"])
	}

	if stats["expiredTokens"] != 1 {
		t.Errorf("Expected 1 expired token, got %v", stats["expiredTokens"])
	}
}

func TestAccountManager_Persistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kiro_persistence_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create first account manager and add account
	crypto := NewCryptoService("test-master-key")
	storage := NewStorageService(tempDir, crypto)
	am1 := NewAccountManager(storage, crypto)

	account := &KiroAccount{
		Email:            "persistent@example.com",
		DisplayName:      "Persistent User",
		BearerToken:      "persistent-token",
		TokenExpiry:      time.Now().Add(24 * time.Hour),
		LoginMethod:      LoginMethodToken,
		SubscriptionType: SubscriptionFree,
		Tags:             []string{"persistent"},
	}

	am1.AddAccount(account)

	// Create second account manager with same storage
	am2 := NewAccountManager(storage, crypto)

	// Verify account persisted
	accounts := am2.ListAccounts()
	if len(accounts) != 1 {
		t.Fatalf("Expected 1 persisted account, got %d", len(accounts))
	}

	persistedAccount := accounts[0]
	if persistedAccount.Email != account.Email {
		t.Errorf("Email not persisted: got %s, want %s", persistedAccount.Email, account.Email)
	}
	if persistedAccount.DisplayName != account.DisplayName {
		t.Errorf("DisplayName not persisted: got %s, want %s", persistedAccount.DisplayName, account.DisplayName)
	}
}