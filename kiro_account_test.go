package main

import (
	"encoding/json"
	"testing"
	"time"
)

func TestQuotaDetail_GetUsagePercentage(t *testing.T) {
	tests := []struct {
		name     string
		quota    QuotaDetail
		expected float64
	}{
		{
			name:     "50% usage",
			quota:    QuotaDetail{Used: 50, Total: 100},
			expected: 0.5,
		},
		{
			name:     "100% usage",
			quota:    QuotaDetail{Used: 100, Total: 100},
			expected: 1.0,
		},
		{
			name:     "0% usage",
			quota:    QuotaDetail{Used: 0, Total: 100},
			expected: 0.0,
		},
		{
			name:     "zero total",
			quota:    QuotaDetail{Used: 0, Total: 0},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.quota.GetUsagePercentage()
			if result != tt.expected {
				t.Errorf("GetUsagePercentage() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestQuotaDetail_IsLowQuota(t *testing.T) {
	tests := []struct {
		name      string
		quota     QuotaDetail
		threshold float64
		expected  bool
	}{
		{
			name:      "above threshold",
			quota:     QuotaDetail{Used: 95, Total: 100},
			threshold: 0.9,
			expected:  true,
		},
		{
			name:      "below threshold",
			quota:     QuotaDetail{Used: 80, Total: 100},
			threshold: 0.9,
			expected:  false,
		},
		{
			name:      "exactly at threshold",
			quota:     QuotaDetail{Used: 90, Total: 100},
			threshold: 0.9,
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.quota.IsLowQuota(tt.threshold)
			if result != tt.expected {
				t.Errorf("IsLowQuota() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestQuotaInfo_GetTotalUsed(t *testing.T) {
	quota := QuotaInfo{
		Main:   QuotaDetail{Used: 100, Total: 1000},
		Trial:  QuotaDetail{Used: 50, Total: 100},
		Reward: QuotaDetail{Used: 25, Total: 200},
	}

	expected := 175
	result := quota.GetTotalUsed()
	if result != expected {
		t.Errorf("GetTotalUsed() = %v, want %v", result, expected)
	}
}

func TestQuotaInfo_GetTotalAvailable(t *testing.T) {
	quota := QuotaInfo{
		Main:   QuotaDetail{Used: 100, Total: 1000},
		Trial:  QuotaDetail{Used: 50, Total: 100},
		Reward: QuotaDetail{Used: 25, Total: 200},
	}

	expected := 1300
	result := quota.GetTotalAvailable()
	if result != expected {
		t.Errorf("GetTotalAvailable() = %v, want %v", result, expected)
	}
}

func TestQuotaInfo_GetOverallUsagePercentage(t *testing.T) {
	quota := QuotaInfo{
		Main:   QuotaDetail{Used: 100, Total: 1000},
		Trial:  QuotaDetail{Used: 50, Total: 100},
		Reward: QuotaDetail{Used: 25, Total: 200},
	}

	expected := 175.0 / 1300.0 // approximately 0.1346
	result := quota.GetOverallUsagePercentage()
	if result != expected {
		t.Errorf("GetOverallUsagePercentage() = %v, want %v", result, expected)
	}
}

func TestKiroAccount_IsTokenExpired(t *testing.T) {
	tests := []struct {
		name     string
		expiry   time.Time
		expected bool
	}{
		{
			name:     "expired token",
			expiry:   time.Now().Add(-1 * time.Hour),
			expected: true,
		},
		{
			name:     "valid token",
			expiry:   time.Now().Add(1 * time.Hour),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := &KiroAccount{
				TokenExpiry: tt.expiry,
			}
			result := account.IsTokenExpired()
			if result != tt.expected {
				t.Errorf("IsTokenExpired() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestKiroAccount_IsTokenExpiringSoon(t *testing.T) {
	tests := []struct {
		name     string
		expiry   time.Time
		duration time.Duration
		expected bool
	}{
		{
			name:     "expiring soon",
			expiry:   time.Now().Add(30 * time.Minute),
			duration: 1 * time.Hour,
			expected: true,
		},
		{
			name:     "not expiring soon",
			expiry:   time.Now().Add(2 * time.Hour),
			duration: 1 * time.Hour,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := &KiroAccount{
				TokenExpiry: tt.expiry,
			}
			result := account.IsTokenExpiringSoon(tt.duration)
			if result != tt.expected {
				t.Errorf("IsTokenExpiringSoon() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestKiroAccount_TagManagement(t *testing.T) {
	account := &KiroAccount{
		Tags: []string{"work", "primary"},
	}

	// Test HasTag
	if !account.HasTag("work") {
		t.Error("HasTag() should return true for existing tag")
	}
	if account.HasTag("personal") {
		t.Error("HasTag() should return false for non-existing tag")
	}

	// Test AddTag
	account.AddTag("personal")
	if !account.HasTag("personal") {
		t.Error("AddTag() should add new tag")
	}

	// Test adding duplicate tag
	initialLen := len(account.Tags)
	account.AddTag("work")
	if len(account.Tags) != initialLen {
		t.Error("AddTag() should not add duplicate tag")
	}

	// Test RemoveTag
	account.RemoveTag("work")
	if account.HasTag("work") {
		t.Error("RemoveTag() should remove existing tag")
	}
}

func TestKiroAccount_GetQuotaAlerts(t *testing.T) {
	account := &KiroAccount{
		ID:          "test-account",
		DisplayName: "Test Account",
		Quota: QuotaInfo{
			Main:   QuotaDetail{Used: 95, Total: 100}, // 95% usage
			Trial:  QuotaDetail{Used: 50, Total: 100}, // 50% usage
			Reward: QuotaDetail{Used: 18, Total: 20},  // 90% usage
		},
	}

	alerts := account.GetQuotaAlerts(0.9) // 90% threshold
	
	// Should have alerts for main and reward quotas
	expectedAlerts := 2
	if len(alerts) != expectedAlerts {
		t.Errorf("GetQuotaAlerts() returned %d alerts, want %d", len(alerts), expectedAlerts)
	}

	// Check that main quota alert exists
	foundMainAlert := false
	foundRewardAlert := false
	for _, alert := range alerts {
		if alert.QuotaType == "main" {
			foundMainAlert = true
		}
		if alert.QuotaType == "reward" {
			foundRewardAlert = true
		}
	}

	if !foundMainAlert {
		t.Error("Expected main quota alert not found")
	}
	if !foundRewardAlert {
		t.Error("Expected reward quota alert not found")
	}
}

func TestKiroAccount_JSONSerialization(t *testing.T) {
	now := time.Now()
	account := &KiroAccount{
		ID:               "test-id",
		Email:            "test@example.com",
		DisplayName:      "Test User",
		Avatar:           "https://example.com/avatar.jpg",
		BearerToken:      "secret-token",      // Should not be serialized
		RefreshToken:     "secret-refresh",    // Should not be serialized
		TokenExpiry:      now,
		LoginMethod:      LoginMethodOAuth,
		Provider:         ProviderGoogle,
		SubscriptionType: SubscriptionPro,
		Quota: QuotaInfo{
			Main:   QuotaDetail{Used: 100, Total: 1000},
			Trial:  QuotaDetail{Used: 0, Total: 100},
			Reward: QuotaDetail{Used: 50, Total: 200},
		},
		Tags:      []string{"work", "primary"},
		Notes:     "Test account",
		IsActive:  true,
		LastUsed:  now,
		CreatedAt: now,
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(account)
	if err != nil {
		t.Fatalf("Failed to marshal account: %v", err)
	}

	// Check that sensitive fields are not in JSON
	jsonStr := string(jsonData)
	if contains(jsonStr, "secret-token") {
		t.Error("BearerToken should not be serialized to JSON")
	}
	if contains(jsonStr, "secret-refresh") {
		t.Error("RefreshToken should not be serialized to JSON")
	}

	// Deserialize from JSON
	var deserializedAccount KiroAccount
	err = json.Unmarshal(jsonData, &deserializedAccount)
	if err != nil {
		t.Fatalf("Failed to unmarshal account: %v", err)
	}

	// Verify non-sensitive fields are preserved
	if deserializedAccount.ID != account.ID {
		t.Errorf("ID mismatch: got %s, want %s", deserializedAccount.ID, account.ID)
	}
	if deserializedAccount.Email != account.Email {
		t.Errorf("Email mismatch: got %s, want %s", deserializedAccount.Email, account.Email)
	}
	if deserializedAccount.LoginMethod != account.LoginMethod {
		t.Errorf("LoginMethod mismatch: got %s, want %s", deserializedAccount.LoginMethod, account.LoginMethod)
	}

	// Verify sensitive fields are empty
	if deserializedAccount.BearerToken != "" {
		t.Error("BearerToken should be empty after deserialization")
	}
	if deserializedAccount.RefreshToken != "" {
		t.Error("RefreshToken should be empty after deserialization")
	}
}

func TestDefaultAccountSettings(t *testing.T) {
	settings := DefaultAccountSettings()

	if settings.QuotaRefreshInterval != 300 {
		t.Errorf("QuotaRefreshInterval = %d, want 300", settings.QuotaRefreshInterval)
	}
	if !settings.AutoRefreshQuota {
		t.Error("AutoRefreshQuota should be true by default")
	}
	if settings.QuotaAlertThreshold != 0.9 {
		t.Errorf("QuotaAlertThreshold = %f, want 0.9", settings.QuotaAlertThreshold)
	}
	if settings.DefaultLoginMethod != LoginMethodOAuth {
		t.Errorf("DefaultLoginMethod = %s, want %s", settings.DefaultLoginMethod, LoginMethodOAuth)
	}
	if settings.PreferredOAuthProvider != ProviderGoogle {
		t.Errorf("PreferredOAuthProvider = %s, want %s", settings.PreferredOAuthProvider, ProviderGoogle)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsAt(s, substr, 1)))
}

func containsAt(s, substr string, start int) bool {
	if start >= len(s) {
		return false
	}
	if start+len(substr) <= len(s) && s[start:start+len(substr)] == substr {
		return true
	}
	return containsAt(s, substr, start+1)
}