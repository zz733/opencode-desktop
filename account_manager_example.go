package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ExampleAccountManager demonstrates the AccountManager functionality
func ExampleAccountManager() {
	fmt.Println("=== Kiro Account Manager Example ===")

	// Setup temporary directory for demo
	tempDir, err := os.MkdirTemp("", "kiro_demo_*")
	if err != nil {
		fmt.Printf("Failed to create temp dir: %v\n", err)
		return
	}
	defer os.RemoveAll(tempDir)

	// Initialize services
	crypto := NewCryptoService("demo-master-key")
	storage := NewStorageService(tempDir, crypto)
	accountMgr := NewAccountManager(storage, crypto)

	fmt.Printf("Initialized AccountManager with data directory: %s\n", tempDir)

	// Create sample accounts
	accounts := []*KiroAccount{
		{
			Email:            "alice@example.com",
			DisplayName:      "Alice Smith",
			BearerToken:      "alice-bearer-token-123",
			RefreshToken:     "alice-refresh-token-456",
			TokenExpiry:      time.Now().Add(24 * time.Hour),
			LoginMethod:      LoginMethodOAuth,
			Provider:         ProviderGoogle,
			SubscriptionType: SubscriptionPro,
			Quota: QuotaInfo{
				Main:   QuotaDetail{Used: 150, Total: 1000},
				Trial:  QuotaDetail{Used: 0, Total: 100},
				Reward: QuotaDetail{Used: 25, Total: 200},
			},
			Tags:  []string{"work", "primary"},
			Notes: "Primary work account",
		},
		{
			Email:            "bob@example.com",
			DisplayName:      "Bob Johnson",
			BearerToken:      "bob-bearer-token-789",
			RefreshToken:     "bob-refresh-token-012",
			TokenExpiry:      time.Now().Add(12 * time.Hour),
			LoginMethod:      LoginMethodToken,
			SubscriptionType: SubscriptionFree,
			Quota: QuotaInfo{
				Main:   QuotaDetail{Used: 80, Total: 100},
				Trial:  QuotaDetail{Used: 45, Total: 50},
				Reward: QuotaDetail{Used: 0, Total: 0},
			},
			Tags:  []string{"personal", "testing"},
			Notes: "Personal testing account",
		},
		{
			Email:            "charlie@example.com",
			DisplayName:      "Charlie Brown",
			BearerToken:      "charlie-bearer-token-345",
			RefreshToken:     "charlie-refresh-token-678",
			TokenExpiry:      time.Now().Add(-1 * time.Hour), // Expired
			LoginMethod:      LoginMethodOAuth,
			Provider:         ProviderGitHub,
			SubscriptionType: SubscriptionProPlus,
			Quota: QuotaInfo{
				Main:   QuotaDetail{Used: 500, Total: 2000},
				Trial:  QuotaDetail{Used: 0, Total: 0},
				Reward: QuotaDetail{Used: 100, Total: 500},
			},
			Tags:  []string{"development", "github"},
			Notes: "Development account with GitHub integration",
		},
	}

	// Add accounts
	fmt.Println("\n=== Adding Accounts ===")
	for i, account := range accounts {
		if err := accountMgr.AddAccount(account); err != nil {
			fmt.Printf("Failed to add account %d: %v\n", i+1, err)
		} else {
			fmt.Printf("✓ Added account: %s (%s)\n", account.DisplayName, account.Email)
		}
	}

	// List all accounts
	fmt.Println("\n=== Account List ===")
	allAccounts := accountMgr.ListAccounts()
	for i, account := range allAccounts {
		status := "Active"
		if !account.IsActive {
			status = "Inactive"
		}
		if account.IsTokenExpired() {
			status += " (Token Expired)"
		}

		fmt.Printf("%d. %s (%s) - %s - %s\n",
			i+1,
			account.DisplayName,
			account.Email,
			account.SubscriptionType,
			status,
		)
		fmt.Printf("   ID: %s\n", account.ID)
		fmt.Printf("   Login: %s", account.LoginMethod)
		if account.Provider != "" {
			fmt.Printf(" via %s", account.Provider)
		}
		fmt.Printf("\n")
		fmt.Printf("   Quota: Main %d/%d (%.1f%%), Trial %d/%d, Reward %d/%d\n",
			account.Quota.Main.Used, account.Quota.Main.Total, account.Quota.Main.GetUsagePercentage()*100,
			account.Quota.Trial.Used, account.Quota.Trial.Total,
			account.Quota.Reward.Used, account.Quota.Reward.Total,
		)
		fmt.Printf("   Tags: %v\n", account.Tags)
		if account.Notes != "" {
			fmt.Printf("   Notes: %s\n", account.Notes)
		}
		fmt.Println()
	}

	// Get active account
	fmt.Println("=== Active Account ===")
	activeAccount, err := accountMgr.GetActiveAccount()
	if err != nil {
		fmt.Printf("Error getting active account: %v\n", err)
	} else {
		fmt.Printf("Active: %s (%s)\n", activeAccount.DisplayName, activeAccount.Email)
	}

	// Switch account
	fmt.Println("\n=== Account Switching ===")
	if len(allAccounts) > 1 {
		newActiveID := allAccounts[1].ID
		fmt.Printf("Switching to: %s\n", allAccounts[1].DisplayName)
		if err := accountMgr.SwitchAccount(newActiveID); err != nil {
			fmt.Printf("Failed to switch account: %v\n", err)
		} else {
			fmt.Println("✓ Account switched successfully")
			
			// Verify switch
			activeAccount, _ = accountMgr.GetActiveAccount()
			fmt.Printf("New active account: %s\n", activeAccount.DisplayName)
		}
	}

	// Update account
	fmt.Println("\n=== Account Update ===")
	if len(allAccounts) > 0 {
		accountID := allAccounts[0].ID
		updates := map[string]interface{}{
			"displayName": "Alice Smith (Updated)",
			"tags":        []string{"work", "primary", "updated"},
			"notes":       "Updated primary work account with new features",
		}
		
		fmt.Printf("Updating account: %s\n", allAccounts[0].DisplayName)
		if err := accountMgr.UpdateAccount(accountID, updates); err != nil {
			fmt.Printf("Failed to update account: %v\n", err)
		} else {
			fmt.Println("✓ Account updated successfully")
			
			// Show updated account
			updatedAccount, _ := accountMgr.GetAccount(accountID)
			fmt.Printf("Updated name: %s\n", updatedAccount.DisplayName)
			fmt.Printf("Updated tags: %v\n", updatedAccount.Tags)
			fmt.Printf("Updated notes: %s\n", updatedAccount.Notes)
		}
	}

	// Batch operations
	fmt.Println("\n=== Batch Operations ===")
	accountIDs := make([]string, len(allAccounts))
	for i, account := range allAccounts {
		accountIDs[i] = account.ID
	}

	// Batch add tags
	newTags := []string{"demo", "example"}
	fmt.Printf("Adding tags %v to all accounts\n", newTags)
	if err := accountMgr.BatchAddTags(accountIDs, newTags); err != nil {
		fmt.Printf("Failed to batch add tags: %v\n", err)
	} else {
		fmt.Println("✓ Tags added to all accounts")
	}

	// Account statistics
	fmt.Println("\n=== Account Statistics ===")
	stats := accountMgr.GetAccountStats()
	fmt.Printf("Total accounts: %v\n", stats["totalAccounts"])
	fmt.Printf("Active account ID: %v\n", stats["activeAccountId"])
	fmt.Printf("Subscription types: %v\n", stats["subscriptionTypes"])
	fmt.Printf("Login methods: %v\n", stats["loginMethods"])
	fmt.Printf("Expired tokens: %v\n", stats["expiredTokens"])
	fmt.Printf("Tokens expiring soon: %v\n", stats["expiringSoon"])

	// Quota alerts
	fmt.Println("\n=== Quota Alerts ===")
	alerts := accountMgr.GetQuotaAlerts(0.8) // 80% threshold
	if len(alerts) > 0 {
		fmt.Printf("Found %d quota alerts:\n", len(alerts))
		for _, alert := range alerts {
			fmt.Printf("⚠️  %s: %s (%.1f%% used)\n", alert.AccountName, alert.Message, alert.Usage*100)
		}
	} else {
		fmt.Println("No quota alerts")
	}

	// Export accounts
	fmt.Println("\n=== Export/Import ===")
	exportData, err := accountMgr.ExportAccounts("")
	if err != nil {
		fmt.Printf("Failed to export accounts: %v\n", err)
	} else {
		fmt.Printf("✓ Exported %d bytes of account data\n", len(exportData))
		
		// Save to file
		exportFile := filepath.Join(tempDir, "accounts_export.json")
		if err := os.WriteFile(exportFile, exportData, 0600); err != nil {
			fmt.Printf("Failed to save export file: %v\n", err)
		} else {
			fmt.Printf("✓ Saved export to: %s\n", exportFile)
		}
	}

	// Demonstrate persistence
	fmt.Println("\n=== Persistence Test ===")
	// Create new account manager with same storage
	accountMgr2 := NewAccountManager(storage, crypto)
	persistedAccounts := accountMgr2.ListAccounts()
	fmt.Printf("✓ Loaded %d accounts from persistent storage\n", len(persistedAccounts))
	
	for _, account := range persistedAccounts {
		fmt.Printf("  - %s (%s)\n", account.DisplayName, account.Email)
	}

	fmt.Println("\n=== Demo Complete ===")
	fmt.Printf("AccountManager successfully demonstrated all core functionality!\n")
	fmt.Printf("Data was stored in: %s\n", tempDir)
}

// Uncomment the following lines to run the example:
// func main() {
//     ExampleAccountManager()
// }