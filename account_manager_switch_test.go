package main

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func findAccountIDByEmail(accounts []*KiroAccount, email string) (string, bool) {
	for _, acc := range accounts {
		if acc != nil && acc.Email == email {
			return acc.ID, true
		}
	}
	return "", false
}

// TestAccountManager_SwitchAccount_Comprehensive tests the complete account switching functionality
func TestAccountManager_SwitchAccount_Comprehensive(t *testing.T) {
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

	accounts := am.ListAccounts()
	account1ID, ok := findAccountIDByEmail(accounts, "test1@example.com")
	if !ok || account1ID == "" {
		t.Fatal("failed to find account1 ID")
	}
	account2ID, ok := findAccountIDByEmail(accounts, "test2@example.com")
	if !ok || account2ID == "" {
		t.Fatal("failed to find account2 ID")
	}

	// Test 1: Verify first account is active initially
	activeAccount, err := am.GetActiveAccount()
	if err != nil {
		t.Fatalf("Failed to get active account: %v", err)
	}
	if activeAccount.ID != account1ID {
		t.Error("First account should be active initially")
	}

	// Test 2: Switch to second account
	err = am.SwitchAccount(account2ID)
	if err != nil {
		t.Fatalf("Failed to switch account: %v", err)
	}

	// Test 3: Verify second account is now active
	activeAccount, err = am.GetActiveAccount()
	if err != nil {
		t.Fatalf("Failed to get active account after switch: %v", err)
	}
	if activeAccount.ID != account2ID {
		t.Error("Second account should be active after switch")
	}

	// Test 4: Verify first account is no longer active
	account1After, err := am.GetAccount(account1ID)
	if err != nil {
		t.Fatalf("Failed to get account1: %v", err)
	}
	if account1After.IsActive {
		t.Error("First account should not be active after switch")
	}

	// Test 5: Verify second account is marked as active
	account2After, err := am.GetAccount(account2ID)
	if err != nil {
		t.Fatalf("Failed to get account2: %v", err)
	}
	if !account2After.IsActive {
		t.Error("Second account should be marked as active")
	}
}

// TestAccountManager_SwitchAccount_NonExistent tests switching to a non-existent account
func TestAccountManager_SwitchAccount_NonExistent(t *testing.T) {
	am, tempDir := setupTestAccountManager(t)
	defer cleanupTestAccountManager(tempDir)

	// Add one test account
	account := &KiroAccount{
		Email:            "test@example.com",
		DisplayName:      "Test User",
		BearerToken:      "test-token",
		TokenExpiry:      time.Now().Add(24 * time.Hour),
		LoginMethod:      LoginMethodToken,
		SubscriptionType: SubscriptionFree,
	}

	am.AddAccount(account)

	// Try to switch to non-existent account
	err := am.SwitchAccount("non-existent-id")
	if err == nil {
		t.Error("Expected error when switching to non-existent account")
	}

	// Verify original account is still active
	activeAccount, err := am.GetActiveAccount()
	if err != nil {
		t.Fatalf("Failed to get active account: %v", err)
	}
	if activeAccount.Email != account.Email {
		t.Error("Original account should still be active after failed switch")
	}
}

// TestAccountManager_SwitchAccount_UpdatesLastUsed tests that LastUsed timestamp is updated
func TestAccountManager_SwitchAccount_UpdatesLastUsed(t *testing.T) {
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
	time.Sleep(10 * time.Millisecond) // Small delay to ensure different timestamps
	am.AddAccount(account2)

	accounts := am.ListAccounts()
	account2ID, ok := findAccountIDByEmail(accounts, "test2@example.com")
	if !ok || account2ID == "" {
		t.Fatal("failed to find account2 ID")
	}

	// Get initial LastUsed time for account2
	account2Before, _ := am.GetAccount(account2ID)
	initialLastUsed := account2Before.LastUsed

	// Wait a bit to ensure timestamp difference
	time.Sleep(10 * time.Millisecond)

	// Switch to account2
	err := am.SwitchAccount(account2ID)
	if err != nil {
		t.Fatalf("Failed to switch account: %v", err)
	}

	// Verify LastUsed was updated
	account2After, err := am.GetAccount(account2ID)
	if err != nil {
		t.Fatalf("Failed to get account2: %v", err)
	}

	if !account2After.LastUsed.After(initialLastUsed) {
		t.Errorf("LastUsed should be updated after switch. Before: %v, After: %v",
			initialLastUsed, account2After.LastUsed)
	}
}

// TestAccountManager_SwitchAccount_Persistence tests that switch persists across restarts
func TestAccountManager_SwitchAccount_Persistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kiro_switch_persistence_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create first account manager and add accounts
	crypto := NewCryptoService("test-master-key")
	storage := NewStorageService(tempDir, crypto)
	am1 := NewAccountManager(storage, crypto)

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

	am1.AddAccount(account1)
	am1.AddAccount(account2)

	accounts := am1.ListAccounts()
	account2ID, ok := findAccountIDByEmail(accounts, "test2@example.com")
	if !ok || account2ID == "" {
		t.Fatal("failed to find account2 ID")
	}

	// Switch to account2
	err = am1.SwitchAccount(account2ID)
	if err != nil {
		t.Fatalf("Failed to switch account: %v", err)
	}

	// Create second account manager with same storage
	am2 := NewAccountManager(storage, crypto)

	// Verify account2 is still active after reload
	activeAccount, err := am2.GetActiveAccount()
	if err != nil {
		t.Fatalf("Failed to get active account after reload: %v", err)
	}

	if activeAccount.ID != account2ID {
		t.Error("Account2 should still be active after persistence reload")
	}

	if activeAccount.Email != "test2@example.com" {
		t.Errorf("Active account email mismatch: got %s, want %s",
			activeAccount.Email, "test2@example.com")
	}
}

// TestAccountManager_SwitchAccount_AlreadyActive tests switching to already active account
func TestAccountManager_SwitchAccount_AlreadyActive(t *testing.T) {
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
	}

	am.AddAccount(account)

	accounts := am.ListAccounts()
	accountID := accounts[0].ID

	// Get initial LastUsed time
	accountBefore, _ := am.GetAccount(accountID)
	initialLastUsed := accountBefore.LastUsed

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Switch to the same account (already active)
	err := am.SwitchAccount(accountID)
	if err != nil {
		t.Fatalf("Switching to already active account should not error: %v", err)
	}

	// Verify account is still active
	activeAccount, err := am.GetActiveAccount()
	if err != nil {
		t.Fatalf("Failed to get active account: %v", err)
	}
	if activeAccount.ID != accountID {
		t.Error("Account should still be active")
	}

	// Verify LastUsed was still updated
	accountAfter, _ := am.GetAccount(accountID)
	if !accountAfter.LastUsed.After(initialLastUsed) {
		t.Error("LastUsed should be updated even when switching to already active account")
	}
}

// TestAccountManager_SwitchAccount_ThreadSafety tests concurrent account switching
func TestAccountManager_SwitchAccount_ThreadSafety(t *testing.T) {
	am, tempDir := setupTestAccountManager(t)
	defer cleanupTestAccountManager(tempDir)

	// Add multiple test accounts
	numAccounts := 5
	accountIDs := make([]string, numAccounts)

	for i := 0; i < numAccounts; i++ {
		account := &KiroAccount{
			Email:            fmt.Sprintf("test%d@example.com", i),
			DisplayName:      fmt.Sprintf("Test User %d", i),
			BearerToken:      fmt.Sprintf("test-token-%d", i),
			TokenExpiry:      time.Now().Add(24 * time.Hour),
			LoginMethod:      LoginMethodToken,
			SubscriptionType: SubscriptionFree,
		}
		am.AddAccount(account)
	}

	accounts := am.ListAccounts()
	for i, acc := range accounts {
		accountIDs[i] = acc.ID
	}

	// Perform concurrent switches
	numGoroutines := 20
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			// Each goroutine switches to a random account
			targetID := accountIDs[index%numAccounts]
			am.SwitchAccount(targetID)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify that exactly one account is active
	activeCount := 0
	var activeID string
	for _, id := range accountIDs {
		account, err := am.GetAccount(id)
		if err != nil {
			t.Fatalf("Failed to get account %s: %v", id, err)
		}
		if account.IsActive {
			activeCount++
			activeID = id
		}
	}

	if activeCount != 1 {
		t.Errorf("Expected exactly 1 active account, got %d", activeCount)
	}

	// Verify GetActiveAccount returns the same account
	activeAccount, err := am.GetActiveAccount()
	if err != nil {
		t.Fatalf("Failed to get active account: %v", err)
	}
	if activeAccount.ID != activeID {
		t.Errorf("Active account mismatch: GetActiveAccount returned %s, but %s is marked active",
			activeAccount.ID, activeID)
	}
}

// TestAccountManager_SwitchAccount_MultipleSequential tests multiple sequential switches
func TestAccountManager_SwitchAccount_MultipleSequential(t *testing.T) {
	am, tempDir := setupTestAccountManager(t)
	defer cleanupTestAccountManager(tempDir)

	// Add three test accounts
	accounts := []*KiroAccount{
		{
			Email:            "test1@example.com",
			DisplayName:      "Test User 1",
			BearerToken:      "test-token-1",
			TokenExpiry:      time.Now().Add(24 * time.Hour),
			LoginMethod:      LoginMethodToken,
			SubscriptionType: SubscriptionFree,
		},
		{
			Email:            "test2@example.com",
			DisplayName:      "Test User 2",
			BearerToken:      "test-token-2",
			TokenExpiry:      time.Now().Add(24 * time.Hour),
			LoginMethod:      LoginMethodToken,
			SubscriptionType: SubscriptionPro,
		},
		{
			Email:            "test3@example.com",
			DisplayName:      "Test User 3",
			BearerToken:      "test-token-3",
			TokenExpiry:      time.Now().Add(24 * time.Hour),
			LoginMethod:      LoginMethodOAuth,
			SubscriptionType: SubscriptionProPlus,
			Provider:         ProviderGoogle,
		},
	}

	for _, acc := range accounts {
		am.AddAccount(acc)
	}

	allAccounts := am.ListAccounts()
	if len(allAccounts) != 3 {
		t.Fatalf("Expected 3 accounts, got %d", len(allAccounts))
	}

	// Perform multiple sequential switches
	switchSequence := []int{1, 2, 0, 2, 1, 0}

	for _, targetIndex := range switchSequence {
		targetID := allAccounts[targetIndex].ID

		err := am.SwitchAccount(targetID)
		if err != nil {
			t.Fatalf("Failed to switch to account %d: %v", targetIndex, err)
		}

		// Verify correct account is active
		activeAccount, err := am.GetActiveAccount()
		if err != nil {
			t.Fatalf("Failed to get active account: %v", err)
		}

		if activeAccount.ID != targetID {
			t.Errorf("Expected account %d to be active, but got %s",
				targetIndex, activeAccount.ID)
		}

		// Verify only one account is active
		activeCount := 0
		for _, acc := range am.ListAccounts() {
			if acc.IsActive {
				activeCount++
			}
		}

		if activeCount != 1 {
			t.Errorf("Expected exactly 1 active account, got %d", activeCount)
		}
	}
}

// TestAccountManager_SwitchAccount_EmptyID tests switching with empty ID
func TestAccountManager_SwitchAccount_EmptyID(t *testing.T) {
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
	}

	am.AddAccount(account)

	// Try to switch with empty ID
	err := am.SwitchAccount("")
	if err == nil {
		t.Error("Expected error when switching with empty ID")
	}

	// Verify original account is still active
	activeAccount, err := am.GetActiveAccount()
	if err != nil {
		t.Fatalf("Failed to get active account: %v", err)
	}
	if activeAccount.Email != account.Email {
		t.Error("Original account should still be active after failed switch")
	}
}

// TestAccountManager_SwitchAccount_RollbackOnError tests rollback on save error
func TestAccountManager_SwitchAccount_RollbackOnError(t *testing.T) {
	// This test would require mocking the storage service to simulate save errors
	// For now, we'll skip this test as it requires more complex setup
	t.Skip("Rollback testing requires storage mocking - to be implemented")
}
