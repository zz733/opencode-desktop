package main

import (
	"fmt"
	"os"
	"testing"
)

// TestAddAccountManual simulates the AddKiroAccount call
func TestAddAccountManual(t *testing.T) {
	// Setup App
	app := &App{}

	refreshToken := os.Getenv("KIRO_TEST_REFRESH_TOKEN")
	if refreshToken == "" {
		t.Skip("未设置 KIRO_TEST_REFRESH_TOKEN，跳过手动集成测试")
	}

	// Create temporary data directory
	tmpDir, err := os.MkdirTemp("", "kiro-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize services
	cryptoService := NewCryptoService("test-key")
	storageService := NewStorageService(tmpDir, cryptoService)

	// Initialize AccountManager
	accountMgr := NewAccountManager(storageService, cryptoService)

	app.accountMgr = accountMgr

	// Test data
	data := map[string]interface{}{
		"refreshToken": refreshToken,
		"displayName":  "Manual Test Account",
	}

	// Call AddKiroAccount
	err = app.AddKiroAccount("token", data)
	if err != nil {
		t.Fatalf("AddKiroAccount failed: %v", err)
	}

	// Verify account was added
	accounts, err := app.GetKiroAccounts()
	if err != nil {
		t.Fatalf("GetKiroAccounts failed: %v", err)
	}
	if len(accounts) == 0 {
		t.Fatal("No accounts found after adding")
	}

	account := accounts[0]
	fmt.Printf("Successfully added account: id=%s email=%s\n", account.ID, account.Email)
}
