package main

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestGetQuota tests the GetQuota method
func TestGetQuota(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		serverResponse *UsageLimitsResponse
		expectedQuota  *QuotaInfo
		serverStatus   int
		expectError    bool
		errorContains  string
	}{
		{
			name:  "valid quota request",
			token: "valid-token",
			expectedQuota: &QuotaInfo{
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
			errorContains: "获取配额失败 (状态码: 401)",
		},
		{
			name:          "forbidden - 403",
			token:         "valid-token-no-permissions",
			serverStatus:  http.StatusForbidden,
			expectError:   true,
			errorContains: "获取配额失败 (状态码: 403)",
		},
		{
			name:          "not found - 404",
			token:         "valid-token-no-quota",
			serverStatus:  http.StatusNotFound,
			expectError:   true,
			errorContains: "获取配额失败 (状态码: 404)",
		},
		{
			name:          "rate limited - 429",
			token:         "valid-token-rate-limited",
			serverStatus:  http.StatusTooManyRequests,
			expectError:   true,
			errorContains: "获取配额失败 (状态码: 429)",
		},
	}

	for _, tt := range tests {
		if tt.expectedQuota != nil {
			tt.serverResponse = buildUsageLimitsResponse(tt.expectedQuota)
		}

		t.Run(tt.name, func(t *testing.T) {
			config := &KiroAPIConfig{
				BaseURL:      "http://localhost",
				UserQuotaURL: "http://localhost/user/quota",
				Timeout:      30,
			}
			quotaService := NewQuotaServiceWithConfig(config)
			quotaService.usageClient = &mockQuotaUsageClient{
				handler: func(token string) (*UsageLimitsResponse, error) {
					if tt.serverStatus != http.StatusOK {
						return nil, fmt.Errorf("获取配额失败 (状态码: %d)", tt.serverStatus)
					}
					return tt.serverResponse, nil
				},
			}
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
					if quota.Main.Used != tt.expectedQuota.Main.Used {
						t.Errorf("Expected Main.Used %d, got %d", tt.expectedQuota.Main.Used, quota.Main.Used)
					}
					if quota.Main.Total != tt.expectedQuota.Main.Total {
						t.Errorf("Expected Main.Total %d, got %d", tt.expectedQuota.Main.Total, quota.Main.Total)
					}
				}
			}
		})
	}
}

// TestGetQuotaCache tests the quota caching mechanism
func TestGetQuotaCache(t *testing.T) {
	requestCount := 0

	config := &KiroAPIConfig{
		BaseURL:      "http://localhost",
		UserQuotaURL: "http://localhost/user/quota",
		Timeout:      30,
	}
	quotaService := NewQuotaServiceWithConfig(config)
	quotaService.usageClient = &mockQuotaUsageClient{
		handler: func(token string) (*UsageLimitsResponse, error) {
			requestCount++
			quota := QuotaInfo{
				Main:   QuotaDetail{Used: 1000, Total: 10000},
				Trial:  QuotaDetail{Used: 50, Total: 100},
				Reward: QuotaDetail{Used: 0, Total: 500},
			}
			return buildUsageLimitsResponse(&quota), nil
		},
	}

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

	config := &KiroAPIConfig{
		BaseURL:      "http://localhost",
		UserQuotaURL: "http://localhost/user/quota",
		Timeout:      30,
	}
	quotaService := NewQuotaServiceWithConfig(config)
	quotaService.usageClient = &mockQuotaUsageClient{
		handler: func(token string) (*UsageLimitsResponse, error) {
			requestCount++
			quota := QuotaInfo{
				Main:   QuotaDetail{Used: requestCount * 100, Total: 10000},
				Trial:  QuotaDetail{Used: 50, Total: 100},
				Reward: QuotaDetail{Used: 0, Total: 500},
			}
			return buildUsageLimitsResponse(&quota), nil
		},
	}

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
	config := &KiroAPIConfig{
		BaseURL:      "http://localhost",
		UserQuotaURL: "http://localhost/user/quota",
		Timeout:      30,
	}
	quotaService := NewQuotaServiceWithConfig(config)
	quotaService.usageClient = &mockQuotaUsageClient{
		handler: func(token string) (*UsageLimitsResponse, error) {
			quota := QuotaInfo{
				Main:   QuotaDetail{Used: 1000, Total: 10000},
				Trial:  QuotaDetail{Used: 50, Total: 100},
				Reward: QuotaDetail{Used: 0, Total: 500},
			}
			return buildUsageLimitsResponse(&quota), nil
		},
	}

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
	config := &KiroAPIConfig{
		BaseURL:      "http://localhost",
		UserQuotaURL: "http://localhost/user/quota",
		Timeout:      30,
	}
	quotaService := NewQuotaServiceWithConfig(config)
	quotaService.usageClient = &mockQuotaUsageClient{
		handler: func(token string) (*UsageLimitsResponse, error) {
			quota := QuotaInfo{
				Main:   QuotaDetail{Used: 1000, Total: 10000},
				Trial:  QuotaDetail{Used: 50, Total: 100},
				Reward: QuotaDetail{Used: 0, Total: 500},
			}
			return buildUsageLimitsResponse(&quota), nil
		},
	}

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
	config := &KiroAPIConfig{
		BaseURL:      "http://localhost",
		UserQuotaURL: "http://localhost/user/quota",
		Timeout:      30,
	}
	quotaService := NewQuotaServiceWithConfig(config)
	quotaService.usageClient = &mockQuotaUsageClient{
		handler: func(token string) (*UsageLimitsResponse, error) {
			quota := QuotaInfo{
				Main:   QuotaDetail{Used: 1000, Total: 10000},
				Trial:  QuotaDetail{Used: 50, Total: 100},
				Reward: QuotaDetail{Used: 0, Total: 500},
			}
			return buildUsageLimitsResponse(&quota), nil
		},
	}

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
	config := &KiroAPIConfig{
		BaseURL:      "http://localhost",
		UserQuotaURL: "http://localhost/user/quota",
		Timeout:      30,
	}
	quotaService := NewQuotaServiceWithConfig(config)
	quotaService.usageClient = &mockQuotaUsageClient{
		handler: func(token string) (*UsageLimitsResponse, error) {
			quota := QuotaInfo{
				Main:   QuotaDetail{Used: 9500, Total: 10000},
				Trial:  QuotaDetail{Used: 50, Total: 100},
				Reward: QuotaDetail{Used: 0, Total: 500},
			}
			return buildUsageLimitsResponse(&quota), nil
		},
	}

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

type mockQuotaUsageClient struct {
	handler func(token string) (*UsageLimitsResponse, error)
}

func (m *mockQuotaUsageClient) GetUserInfo(accessToken string) (*UsageLimitsResponse, error) {
	return m.handler(accessToken)
}

func buildUsageLimitsResponse(quota *QuotaInfo) *UsageLimitsResponse {
	if quota == nil {
		return &UsageLimitsResponse{}
	}

	return &UsageLimitsResponse{
		UserInfo: &struct {
			Email  string `json:"email"`
			UserID string `json:"userId"`
		}{
			Email:  "test@example.com",
			UserID: "user-123",
		},
		UsageBreakdownList: []struct {
			ResourceType  string `json:"resourceType"`
			UsageLimit    int    `json:"usageLimit"`
			CurrentUsage  int    `json:"currentUsage"`
			FreeTrialInfo *struct {
				UsageLimit   int `json:"usageLimit"`
				CurrentUsage int `json:"currentUsage"`
			} `json:"freeTrialInfo"`
			Bonuses []struct {
				UsageLimit   float64 `json:"usageLimit"`
				CurrentUsage float64 `json:"currentUsage"`
			} `json:"bonuses"`
		}{
			{
				ResourceType: "chat",
				UsageLimit:   quota.Main.Total,
				CurrentUsage: quota.Main.Used,
				FreeTrialInfo: &struct {
					UsageLimit   int `json:"usageLimit"`
					CurrentUsage int `json:"currentUsage"`
				}{
					UsageLimit:   quota.Trial.Total,
					CurrentUsage: quota.Trial.Used,
				},
				Bonuses: []struct {
					UsageLimit   float64 `json:"usageLimit"`
					CurrentUsage float64 `json:"currentUsage"`
				}{
					{
						UsageLimit:   float64(quota.Reward.Total),
						CurrentUsage: float64(quota.Reward.Used),
					},
				},
			},
		},
	}
}

// Note: detectSubscriptionType is tested indirectly through ValidateAndCreateAccount
// in auth_service_test.go since it's an internal function
