package main

import (
	"encoding/json"
	"fmt"
	"time"
)

// ExampleKiroAccount demonstrates how to use the KiroAccount data structures
func ExampleKiroAccount() {
	// Create a new Kiro account
	account := &KiroAccount{
		ID:               "kiro-001",
		Email:            "user@example.com",
		DisplayName:      "John Doe",
		Avatar:           "https://example.com/avatar.jpg",
		BearerToken:      "bearer-token-secret",
		RefreshToken:     "refresh-token-secret",
		TokenExpiry:      time.Now().Add(24 * time.Hour),
		LoginMethod:      LoginMethodOAuth,
		Provider:         ProviderGoogle,
		SubscriptionType: SubscriptionPro,
		Quota: QuotaInfo{
			Main:   QuotaDetail{Used: 150, Total: 1000},
			Trial:  QuotaDetail{Used: 0, Total: 100},
			Reward: QuotaDetail{Used: 25, Total: 200},
		},
		Tags:      []string{"work", "primary"},
		Notes:     "Primary work account",
		IsActive:  true,
		LastUsed:  time.Now(),
		CreatedAt: time.Now().Add(-30 * 24 * time.Hour), // Created 30 days ago
	}

	fmt.Println("=== KiroAccount Example ===")
	fmt.Printf("Account: %s (%s)\n", account.DisplayName, account.Email)
	fmt.Printf("Subscription: %s\n", account.SubscriptionType)
	fmt.Printf("Login Method: %s via %s\n", account.LoginMethod, account.Provider)

	// Demonstrate quota calculations
	fmt.Println("\n=== Quota Information ===")
	fmt.Printf("Main Quota: %d/%d (%.1f%%)\n", 
		account.Quota.Main.Used, 
		account.Quota.Main.Total, 
		account.Quota.Main.GetUsagePercentage()*100)
	
	fmt.Printf("Total Used: %d/%d (%.1f%%)\n", 
		account.Quota.GetTotalUsed(), 
		account.Quota.GetTotalAvailable(), 
		account.Quota.GetOverallUsagePercentage()*100)

	// Demonstrate tag management
	fmt.Println("\n=== Tag Management ===")
	fmt.Printf("Current tags: %v\n", account.Tags)
	
	account.AddTag("testing")
	fmt.Printf("After adding 'testing': %v\n", account.Tags)
	
	account.RemoveTag("work")
	fmt.Printf("After removing 'work': %v\n", account.Tags)

	// Demonstrate token expiry checks
	fmt.Println("\n=== Token Status ===")
	fmt.Printf("Token expired: %v\n", account.IsTokenExpired())
	fmt.Printf("Token expiring in 1 hour: %v\n", account.IsTokenExpiringSoon(1*time.Hour))

	// Demonstrate quota alerts
	fmt.Println("\n=== Quota Alerts ===")
	alerts := account.GetQuotaAlerts(0.1) // 10% threshold
	if len(alerts) > 0 {
		for _, alert := range alerts {
			fmt.Printf("Alert: %s (%.1f%% used)\n", alert.Message, alert.Usage*100)
		}
	} else {
		fmt.Println("No quota alerts")
	}

	// Demonstrate JSON serialization (sensitive data excluded)
	fmt.Println("\n=== JSON Serialization ===")
	jsonData, err := json.MarshalIndent(account, "", "  ")
	if err != nil {
		fmt.Printf("Error serializing: %v\n", err)
		return
	}
	
	fmt.Println("Serialized account (sensitive data excluded):")
	fmt.Println(string(jsonData))

	// Demonstrate account data structure
	fmt.Println("\n=== Account Data Structure ===")
	accountData := &AccountData{
		Version:         "1.0",
		Accounts:        []*KiroAccount{account},
		ActiveAccountID: account.ID,
		Settings:        DefaultAccountSettings(),
		Tags: []Tag{
			{Name: "work", Color: "#007acc", Description: "Work related accounts"},
			{Name: "personal", Color: "#28a745", Description: "Personal accounts"},
			{Name: "testing", Color: "#ffc107", Description: "Testing accounts"},
		},
		LastUpdated: time.Now(),
	}

	dataJson, err := json.MarshalIndent(accountData, "", "  ")
	if err != nil {
		fmt.Printf("Error serializing account data: %v\n", err)
		return
	}
	
	fmt.Println("Complete account data structure:")
	fmt.Println(string(dataJson))
}

// This function can be called to run the example
// Uncomment the following lines in main() to run it:
// func main() {
//     ExampleKiroAccount()
// }