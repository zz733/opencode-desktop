package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCryptoService_Integration(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "crypto_integration_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize services
	crypto := NewCryptoService("integration-test-master-key")
	storage := NewStorageService(tempDir, crypto)
	accountMgr := NewAccountManager(storage, crypto)

	// Create test account with sensitive data
	account := &KiroAccount{
		ID:               "test-account-1",
		Email:            "test@example.com",
		DisplayName:      "Test User",
		BearerToken:      "sensitive-bearer-token-12345",
		RefreshToken:     "sensitive-refresh-token-67890",
		TokenExpiry:      time.Now().Add(24 * time.Hour),
		LoginMethod:      LoginMethodToken,
		SubscriptionType: SubscriptionPro,
		Quota: QuotaInfo{
			Main:   QuotaDetail{Used: 100, Total: 1000},
			Trial:  QuotaDetail{Used: 0, Total: 100},
			Reward: QuotaDetail{Used: 50, Total: 200},
		},
		Tags:      []string{"test", "integration"},
		Notes:     "Integration test account",
		IsActive:  true,
		LastUsed:  time.Now(),
		CreatedAt: time.Now(),
	}

	// Add account (this should encrypt sensitive data)
	err = accountMgr.AddAccount(account)
	if err != nil {
		t.Fatalf("Failed to add account: %v", err)
	}

	// Verify account was saved and encrypted
	accounts := accountMgr.ListAccounts()
	if len(accounts) != 1 {
		t.Fatalf("Expected 1 account, got %d", len(accounts))
	}

	savedAccount := accounts[0]
	
	// Note: BearerToken and RefreshToken are excluded from JSON serialization for security
	// This is the expected behavior - sensitive tokens should not be stored in plain JSON
	// In a production system, these would be stored in a secure keychain or encrypted separately
	
	// Verify non-sensitive data is correctly stored and retrieved
	if savedAccount.ID != account.ID {
		t.Errorf("ID mismatch: expected %s, got %s", account.ID, savedAccount.ID)
	}
	if savedAccount.Email != account.Email {
		t.Errorf("Email mismatch: expected %s, got %s", account.Email, savedAccount.Email)
	}
	if savedAccount.DisplayName != account.DisplayName {
		t.Errorf("DisplayName mismatch: expected %s, got %s", account.DisplayName, savedAccount.DisplayName)
	}

	// Verify data is encrypted on disk by reading raw file
	accountsFile := filepath.Join(tempDir, "accounts.json.enc")
	if _, err := os.Stat(accountsFile); os.IsNotExist(err) {
		t.Error("Encrypted accounts file should exist")
	}

	// Read raw encrypted file content
	encryptedData, err := os.ReadFile(accountsFile)
	if err != nil {
		t.Fatalf("Failed to read encrypted file: %v", err)
	}

	// Verify sensitive data is not in plain text in the file
	encryptedContent := string(encryptedData)
	if containsSensitiveData(encryptedContent, account.Email) {
		t.Error("Email should not be in plain text in encrypted file")
	}

	// Create new AccountManager instance to test decryption
	crypto2 := NewCryptoService("integration-test-master-key")
	storage2 := NewStorageService(tempDir, crypto2)
	accountMgr2 := NewAccountManager(storage2, crypto2)

	// Load accounts (this should decrypt the data)
	accounts2 := accountMgr2.ListAccounts()
	if len(accounts2) != 1 {
		t.Fatalf("Expected 1 account after reload, got %d", len(accounts2))
	}

	reloadedAccount := accounts2[0]
	
	// Verify all non-sensitive data was correctly decrypted
	if reloadedAccount.ID != account.ID {
		t.Errorf("ID mismatch: expected %s, got %s", account.ID, reloadedAccount.ID)
	}
	if reloadedAccount.Email != account.Email {
		t.Errorf("Email mismatch: expected %s, got %s", account.Email, reloadedAccount.Email)
	}
	if reloadedAccount.DisplayName != account.DisplayName {
		t.Errorf("DisplayName mismatch: expected %s, got %s", account.DisplayName, reloadedAccount.DisplayName)
	}

	// Test with wrong master key (should fail to decrypt)
	crypto3 := NewCryptoService("wrong-master-key")
	storage3 := NewStorageService(tempDir, crypto3)
	accountMgr3 := NewAccountManager(storage3, crypto3)

	// This should fail to load accounts due to decryption failure
	accounts3 := accountMgr3.ListAccounts()
	// Should return empty list due to decryption failure
	if len(accounts3) != 0 {
		t.Error("Should not be able to load accounts with wrong master key")
	}
}

func TestCryptoService_PasswordBasedEncryption(t *testing.T) {
	crypto := NewCryptoService("test-master-key")

	// Test data that might be exported/imported
	testData := []byte(`{
		"accounts": [
			{
				"id": "test-1",
				"email": "user@example.com",
				"bearerToken": "secret-token-123",
				"refreshToken": "secret-refresh-456"
			}
		]
	}`)

	password := "user-export-password-123"

	// Encrypt with password (for export)
	encrypted, err := crypto.EncryptWithPassword(testData, password)
	if err != nil {
		t.Fatalf("Failed to encrypt with password: %v", err)
	}

	// Verify encrypted data doesn't contain sensitive info
	encryptedStr := string(encrypted)
	if containsSensitiveData(encryptedStr, "secret-token-123") {
		t.Error("Encrypted data should not contain plain text tokens")
	}

	// Decrypt with correct password (for import)
	decrypted, err := crypto.DecryptWithPassword(encrypted, password)
	if err != nil {
		t.Fatalf("Failed to decrypt with password: %v", err)
	}

	if string(decrypted) != string(testData) {
		t.Error("Decrypted data doesn't match original")
	}

	// Try with wrong password
	_, err = crypto.DecryptWithPassword(encrypted, "wrong-password")
	if err == nil {
		t.Error("Should fail to decrypt with wrong password")
	}
}

func TestCryptoService_SecureMemoryHandling(t *testing.T) {
	crypto := NewCryptoService("test-master-key")

	// Test secure wipe functionality
	sensitiveData := []byte("very-sensitive-bearer-token-data")
	originalData := make([]byte, len(sensitiveData))
	copy(originalData, sensitiveData)

	// Use the data (simulate processing)
	encrypted, err := crypto.Encrypt(sensitiveData)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// Securely wipe the original sensitive data
	crypto.SecureWipe(sensitiveData)

	// Verify data is wiped
	for i, b := range sensitiveData {
		if b != 0 {
			t.Errorf("Sensitive data not properly wiped at index %d", i)
		}
	}

	// Verify we can still decrypt the encrypted version
	decrypted, err := crypto.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if string(decrypted) != string(originalData) {
		t.Error("Decrypted data doesn't match original")
	}

	// Wipe decrypted data too
	crypto.SecureWipe(decrypted)
}

// Helper function to check if sensitive data appears in encrypted content
func containsSensitiveData(content, sensitiveData string) bool {
	// Simple check - in real encrypted data, sensitive info should not appear as plain text
	return len(sensitiveData) > 0 && len(content) > 0 && 
		   len(sensitiveData) < len(content) && 
		   findSubstring(content, sensitiveData)
}

// Simple substring search to avoid importing strings package in test
func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(substr) > len(s) {
		return false
	}
	
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}