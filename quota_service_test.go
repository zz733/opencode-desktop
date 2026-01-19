package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestGetQuota tests the GetQuota method
func TestGetQuota(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		serverResponse *QuotaInfo
		serverStatus   int
		expectError    bool
		errorContains  string
	}{
		{
			name:  "valid quota request",
			token: "valid-token",
			serverResponse: &QuotaInfo{
				Main:   QuotaDetail{Used: 1000, Total: 10000},
				Trial:  QuotaDetail{Used: 50, Total: 100},
				Reward: QuotaDetail{Used: 0, Total: 500},
			},
			serverStatus: http.StatusOK,
			expectError:  false,
		},
		{
			name:          "empty token",
			token:         "",
			serverStatus:  http.StatusOK,
			expectError:   true,
			errorContains: "token cannot be empty",
		},
		{
			name:          "unauthorized - 401",
			token:         "invalid-token",
			serverStatus:  http.StatusUnauthorized,
			expectError:   true,
			errorContains: "invalid or expired",
		},
		{
			name:          "forbidden - 403",
			token:         "valid-token-no-permissions",
			serverStatus:  http.StatusForbidden,
			expectError:   true,
			errorContains: "does not have required permissions",
		},
		{
			name:          "not found - 404",
			token:         "valid-token-no-quota",
			serverStatus:  http.StatusNotFound,
			expectError:   true,
			errorContains: "quota information not found",
		},
		{
			name:          "rate limited - 429",
			token:         "valid-token-rate-limited",
			serverStatus:  http.StatusTooManyRequests,
			expectError:   true,
			errorContains: "rate limit exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/user/quota" {
					t.Errorf("Expected path /user/quota, got %s", r.URL.Path)
				}

				// Verify Authorization header
				if tt.token != "" {
					authHeader := r.Header.Get("Authorization")
					expectedPrefix := "Bearer "
					if !strings.HasPrefix(authHeader, expectedPrefix) {
						t.Errorf("Expected Authorization header to start with '%s'", expectedPrefix)
					}
				}

				w.WriteHeader(tt.serverStatus)
				if tt.serverStatus == http.StatusOK && tt.serverResponse != nil {
					json.NewEncoder(w).Encode(tt.serverResponse)
				}
			}))
			defer server.Close()

			config := &KiroAPIConfig{
				BaseURL:      server.URL,
				UserQuotaURL: server.URL + "/user/quota",
				Timeout:      30,
			}
			quotaService := NewQuotaServiceWithConfig(config)
			quota, err := quotaService.GetQuota(tt.token)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if quota == nil {
					t.Errorf("Expected quota but got nil")
				} else {
					// Verify quota values
					if quota.Main.Used != tt.serverResponse.Main.Used {
						t.Errorf("Expected Main.Used %d, got %d", tt.serverResponse.Main.Used, quota.Main.Used)
					}
					if quota.Main.Total != tt.serverResponse.Main.Total {
						t.Errorf("Expected Main.Total %d, got %d", tt.serverResponse.Main.Total, quota.Main.Total)
					}
				}
			}
		})
	}
}

// TestGetQuotaCache tests the quota caching mechanism
func TestGetQuotaCache(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		quota := QuotaInfo{
			Main:   QuotaDetail{Used: 1000, Total: 10000},
			Trial:  QuotaDetail{Used: 50, Total: 100},
			Reward: QuotaDetail{Used: 0, Total: 500},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(quota)
	}))
	defer server.Close()

	config := &KiroAPIConfig{
		BaseURL:      server.URL,
		UserQuotaURL: server.URL + "/user/quota",
		Timeout:      30,
	}
	quotaService := NewQuotaServiceWithConfig(config)

	token := "test-token"

	// First request - should hit the API
	_, err := quotaService.GetQuota(token)
	if err != nil {
		t.Fatalf("Unexpected error on first request: %v", err)
	}
	if requestCount != 1 {
		t.Errorf("Expected 1 API request, got %d", requestCount)
	}

	// Second request - should use cache
	_, err = quotaService.GetQuota(token)
	if err != nil {
		t.Fatalf("Unexpected error on second request: %v", err)
	}
	if requestCount != 1 {
		t.Errorf("Expected cache hit (1 API request total), got %d requests", requestCount)
	}

	// Clear cache and request again - should hit the API
	quotaService.ClearCache()
	_, err = quotaService.GetQuota(token)
	if err != nil {
		t.Fatalf("Unexpected error after cache clear: %v", err)
	}
	if requestCount != 2 {
		t.Errorf("Expected 2 API requests after cache clear, got %d", requestCount)
	}
}

// TestRefreshQuota tests the RefreshQuota method
func TestRefreshQuota(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		quota := QuotaInfo{
			Main:   QuotaDetail{Used: requestCount * 100, Total: 10000},
			Trial:  QuotaDetail{Used: 50, Total: 100},
			Reward: QuotaDetail{Used: 0, Total: 500},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(quota)
	}))
	defer server.Close()

	config := &KiroAPIConfig{
		BaseURL:      server.URL,
		UserQuotaURL: server.URL + "/user/quota",
		Timeout:      30,
	}
	quotaService := NewQuotaServiceWithConfig(config)

	token := "test-token"
	accountID := "account-123"

	// First request
	quota1, err := quotaService.GetQuota(token)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if quota1.Main.Used != 100 {
		t.Errorf("Expected Main.Used 100, got %d", quota1.Main.Used)
	}

	// Refresh quota - should bypass cache
	err = quotaService.RefreshQuota(accountID, token)
	if err != nil {
		t.Fatalf("Unexpected error on refresh: %v", err)
	}

	// Get quota again - should return refreshed value from cache
	quota2, err := quotaService.GetQuota(token)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if quota2.Main.Used != 200 {
		t.Errorf("Expected Main.Used 200 after refresh, got %d", quota2.Main.Used)
	}
}

// TestBatchRefreshQuota tests the BatchRefreshQuota method
func TestBatchRefreshQuota(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		quota := QuotaInfo{
			Main:   QuotaDetail{Used: 1000, Total: 10000},
			Trial:  QuotaDetail{Used: 50, Total: 100},
			Reward: QuotaDetail{Used: 0, Total: 500},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(quota)
	}))
	defer server.Close()

	config := &KiroAPIConfig{
		BaseURL:      server.URL,
		UserQuotaURL: server.URL + "/user/quota",
		Timeout:      30,
	}
	quotaService := NewQuotaServiceWithConfig(config)

	// Create test accounts
	accounts := []*KiroAccount{
		{
			ID:          "account-1",
			BearerToken: "token-1",
			Email:       "user1@example.com",
		},
		{
			ID:          "account-2",
			BearerToken: "token-2",
			Email:       "user2@example.com",
		},
	}

	// Batch refresh
	err := quotaService.BatchRefreshQuota(accounts)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify that accounts were updated
	for _, account := range accounts {
		if account.Quota.Main.Used != 1000 {
			t.Errorf("Expected account %s Main.Used 1000, got %d", account.ID, account.Quota.Main.Used)
		}
	}
}

// TestClearExpiredCache tests the ClearExpiredCache method
func TestClearExpiredCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		quota := QuotaInfo{
			Main:   QuotaDetail{Used: 1000, Total: 10000},
			Trial:  QuotaDetail{Used: 50, Total: 100},
			Reward: QuotaDetail{Used: 0, Total: 500},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(quota)
	}))
	defer server.Close()

	config := &KiroAPIConfig{
		BaseURL:      server.URL,
		UserQuotaURL: server.URL + "/user/quota",
		Timeout:      30,
	}
	quotaService := NewQuotaServiceWithConfig(config)
	
	// Set a very short cache TTL for testing
	quotaService.cacheTTL = 100 * time.Millisecond

	token := "test-token"

	// Add entry to cache
	_, err := quotaService.GetQuota(token)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify cache has entry
	stats := quotaService.GetCacheStats()
	if stats["totalEntries"].(int) != 1 {
		t.Errorf("Expected 1 cache entry, got %d", stats["totalEntries"].(int))
	}

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Clear expired entries
	quotaService.ClearExpiredCache()

	// Verify cache is empty
	stats = quotaService.GetCacheStats()
	if stats["totalEntries"].(int) != 0 {
		t.Errorf("Expected 0 cache entries after clearing expired, got %d", stats["totalEntries"].(int))
	}
}

// TestGetCacheStats tests the GetCacheStats method
func TestGetCacheStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		quota := QuotaInfo{
			Main:   QuotaDetail{Used: 1000, Total: 10000},
			Trial:  QuotaDetail{Used: 50, Total: 100},
			Reward: QuotaDetail{Used: 0, Total: 500},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(quota)
	}))
	defer server.Close()

	config := &KiroAPIConfig{
		BaseURL:      server.URL,
		UserQuotaURL: server.URL + "/user/quota",
		Timeout:      30,
	}
	quotaService := NewQuotaServiceWithConfig(config)

	// Initially empty
	stats := quotaService.GetCacheStats()
	if stats["totalEntries"].(int) != 0 {
		t.Errorf("Expected 0 initial entries, got %d", stats["totalEntries"].(int))
	}

	// Add entries
	quotaService.GetQuota("token-1")
	quotaService.GetQuota("token-2")

	stats = quotaService.GetCacheStats()
	if stats["totalEntries"].(int) != 2 {
		t.Errorf("Expected 2 cache entries, got %d", stats["totalEntries"].(int))
	}
}

// TestQuotaMonitor tests the QuotaMonitor functionality
func TestQuotaMonitor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		quota := QuotaInfo{
			Main:   QuotaDetail{Used: 9500, Total: 10000}, // 95% usage
			Trial:  QuotaDetail{Used: 50, Total: 100},
			Reward: QuotaDetail{Used: 0, Total: 500},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(quota)
	}))
	defer server.Close()

	config := &KiroAPIConfig{
		BaseURL:      server.URL,
		UserQuotaURL: server.URL + "/user/quota",
		Timeout:      30,
	}
	quotaService := NewQuotaServiceWithConfig(config)

	// Create test accounts
	accounts := []*KiroAccount{
		{
			ID:          "account-1",
			BearerToken: "token-1",
			DisplayName: "Test User 1",
			Email:       "user1@example.com",
			TokenExpiry: time.Now().Add(24 * time.Hour),
			Quota: QuotaInfo{
				Main:   QuotaDetail{Used: 9500, Total: 10000},
				Trial:  QuotaDetail{Used: 50, Total: 100},
				Reward: QuotaDetail{Used: 0, Total: 500},
			},
		},
	}

	accountsFunc := func() []*KiroAccount {
		return accounts
	}

	monitor := NewQuotaMonitor(quotaService, accountsFunc)

	// Test initial state
	if monitor.IsRunning() {
		t.Error("Monitor should not be running initially")
	}

	// Test start
	monitor.Start()
	if !monitor.IsRunning() {
		t.Error("Monitor should be running after Start()")
	}

	// Test check quota alerts
	alerts := monitor.CheckQuotaAlerts()
	if len(alerts) == 0 {
		t.Error("Expected quota alerts for high usage account")
	}

	// Test stop
	monitor.Stop()
	time.Sleep(100 * time.Millisecond) // Give it time to stop
	if monitor.IsRunning() {
		t.Error("Monitor should not be running after Stop()")
	}

	// Test double start (should be idempotent)
	monitor.Start()
	monitor.Start()
	if !monitor.IsRunning() {
		t.Error("Monitor should be running after double Start()")
	}
	monitor.Stop()
}

// TestQuotaMonitorSetters tests the SetInterval and SetThreshold methods
func TestQuotaMonitorSetters(t *testing.T) {
	config := DefaultKiroAPIConfig()
	quotaService := NewQuotaServiceWithConfig(config)

	accountsFunc := func() []*KiroAccount {
		return []*KiroAccount{}
	}

	monitor := NewQuotaMonitor(quotaService, accountsFunc)

	// Test SetInterval
	newInterval := 10 * time.Minute
	monitor.SetInterval(newInterval)
	if monitor.interval != newInterval {
		t.Errorf("Expected interval %v, got %v", newInterval, monitor.interval)
	}

	// Test SetThreshold
	newThreshold := 0.8
	monitor.SetThreshold(newThreshold)
	if monitor.threshold != newThreshold {
		t.Errorf("Expected threshold %v, got %v", newThreshold, monitor.threshold)
	}
}

// Note: detectSubscriptionType is tested indirectly through ValidateAndCreateAccount
// in auth_service_test.go since it's an internal function
