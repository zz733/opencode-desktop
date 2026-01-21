package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestGetUserProfileIntegration tests the complete user profile retrieval flow
func TestGetUserProfileIntegration(t *testing.T) {
	tests := []struct {
		name          string
		token         string
		usageResponse *UsageLimitsResponse
		clientErr     error
		expectError   bool
		errorContains string
	}{
		{
			name:  "successful profile retrieval",
			token: "valid-bearer-token-with-sufficient-length",
			usageResponse: &UsageLimitsResponse{
				UserInfo: &struct {
					Email  string `json:"email"`
					UserID string `json:"userId"`
				}{
					Email:  "test@example.com",
					UserID: "user123",
				},
			},
			expectError: false,
		},
		{
			name:          "usage api error",
			token:         "invalid-token-with-sufficient-length",
			clientErr:     fmt.Errorf("unauthorized"),
			expectError:   true,
			errorContains: "unauthorized",
		},
		{
			name:          "missing user info",
			token:         "valid-token-with-incomplete-profile",
			usageResponse: &UsageLimitsResponse{},
			expectError:   true,
			errorContains: "user info not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultKiroAPIConfig()
			authService := NewAuthServiceWithConfig(config)
			authService.kiroClient = &mockDesktopClient{
				getUserInfo: func(token string) (*UsageLimitsResponse, error) {
					if tt.clientErr != nil {
						return nil, tt.clientErr
					}
					return tt.usageResponse, nil
				},
			}

			// Test GetUserProfile
			profile, err := authService.GetUserProfile(tt.token)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !containsStr(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if profile == nil {
					t.Errorf("Expected profile but got nil")
				} else {
					if profile.Email != "test@example.com" {
						t.Errorf("Expected email 'test@example.com', got '%s'", profile.Email)
					}
					if profile.Name != "test@example.com" {
						t.Errorf("Expected name 'test@example.com', got '%s'", profile.Name)
					}
				}
			}
		})
	}
}

// TestGetQuotaIntegration tests the complete quota retrieval flow
func TestGetQuotaIntegration(t *testing.T) {
	tests := []struct {
		name          string
		token         string
		usageResponse *UsageLimitsResponse
		clientErr     error
		expectError   bool
		errorContains string
	}{
		{
			name:  "successful quota retrieval",
			token: "valid-bearer-token-with-sufficient-length",
			usageResponse: buildUsageLimitsResponse(&QuotaInfo{
				Main:   QuotaDetail{Used: 100, Total: 1000},
				Trial:  QuotaDetail{Used: 50, Total: 100},
				Reward: QuotaDetail{Used: 0, Total: 200},
			}),
			expectError: false,
		},
		{
			name:          "quota retrieval error",
			token:         "invalid-token-with-sufficient-length",
			clientErr:     fmt.Errorf("quota error"),
			expectError:   true,
			errorContains: "quota error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultKiroAPIConfig()
			quotaService := NewQuotaServiceWithConfig(config)
			quotaService.usageClient = &mockUsageClient{
				handler: func(token string) (*UsageLimitsResponse, error) {
					if tt.clientErr != nil {
						return nil, tt.clientErr
					}
					return tt.usageResponse, nil
				},
			}

			// Test GetQuota
			quota, err := quotaService.GetQuota(tt.token)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !containsStr(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if quota == nil {
					t.Errorf("Expected quota but got nil")
				} else {
					if quota.Main.Used != 100 || quota.Main.Total != 1000 {
						t.Errorf("Unexpected quota values: %+v", quota.Main)
					}
				}
			}
		})
	}
}

// TestValidateAndCreateAccountIntegration tests the complete account creation flow
func TestValidateAndCreateAccountIntegration(t *testing.T) {
	validToken := "valid-bearer-token-with-sufficient-length-for-testing"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(TokenInfo{
			AccessToken:  validToken,
			RefreshToken: "refresh-token",
			ExpiresAt:    time.Now().Add(24 * time.Hour),
			TokenType:    "Bearer",
		})
	}))
	defer server.Close()

	// Create services with test server URL
	config := &KiroAPIConfig{
		BaseURL:         server.URL,
		AuthValidateURL: server.URL + "/auth/validate",
		Timeout:         30,
	}
	authService := NewAuthServiceWithConfig(config)
	quotaService := NewQuotaServiceWithConfig(config)
	authService.kiroClient = &mockDesktopClient{
		getUserInfo: func(token string) (*UsageLimitsResponse, error) {
			return buildUsageLimitsResponse(&QuotaInfo{
				Main:   QuotaDetail{Used: 100, Total: 1000},
				Trial:  QuotaDetail{Used: 50, Total: 100},
				Reward: QuotaDetail{Used: 0, Total: 200},
			}), nil
		},
	}
	quotaService.usageClient = &mockUsageClient{
		handler: func(token string) (*UsageLimitsResponse, error) {
			return buildUsageLimitsResponse(&QuotaInfo{
				Main:   QuotaDetail{Used: 100, Total: 1000},
				Trial:  QuotaDetail{Used: 50, Total: 100},
				Reward: QuotaDetail{Used: 0, Total: 200},
			}), nil
		},
	}

	// Test ValidateAndCreateAccount
	account, err := authService.ValidateAndCreateAccount(validToken, LoginMethodToken, "", quotaService)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if account == nil {
		t.Fatal("Expected account but got nil")
	}

	// Verify account fields
	if account.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", account.Email)
	}
	if account.DisplayName != account.Email {
		t.Errorf("Expected display name to match email, got '%s'", account.DisplayName)
	}
	if account.BearerToken != validToken {
		t.Errorf("Expected token '%s', got '%s'", validToken, account.BearerToken)
	}
	if account.LoginMethod != LoginMethodToken {
		t.Errorf("Expected login method 'token', got '%s'", account.LoginMethod)
	}

	// Verify quota information
	if account.Quota.Main.Total != 1000 {
		t.Errorf("Expected main quota total 1000, got %d", account.Quota.Main.Total)
	}
	if account.Quota.Main.Used != 100 {
		t.Errorf("Expected main quota used 100, got %d", account.Quota.Main.Used)
	}

	// Verify subscription type detection
	if account.SubscriptionType == "" {
		t.Error("Expected subscription type to be set")
	}
}

// TestQuotaCaching tests the quota caching mechanism
func TestQuotaCaching(t *testing.T) {
	requestCount := 0
	validToken := "valid-bearer-token-with-sufficient-length"

	config := &KiroAPIConfig{
		BaseURL:      "http://localhost",
		UserQuotaURL: "http://localhost/user/quota",
		Timeout:      30,
	}
	quotaService := NewQuotaServiceWithConfig(config)
	quotaService.usageClient = &mockUsageClient{
		handler: func(token string) (*UsageLimitsResponse, error) {
			requestCount++
			return buildUsageLimitsResponse(&QuotaInfo{
				Main:   QuotaDetail{Used: 100, Total: 1000},
				Trial:  QuotaDetail{Used: 50, Total: 100},
				Reward: QuotaDetail{Used: 0, Total: 200},
			}), nil
		},
	}

	// First request - should hit the API
	_, err := quotaService.GetQuota(validToken)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if requestCount != 1 {
		t.Errorf("Expected 1 API request, got %d", requestCount)
	}

	// Second request - should use cache
	_, err = quotaService.GetQuota(validToken)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if requestCount != 1 {
		t.Errorf("Expected 1 API request (cached), got %d", requestCount)
	}

	// Clear cache and request again
	quotaService.ClearCache()
	_, err = quotaService.GetQuota(validToken)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if requestCount != 2 {
		t.Errorf("Expected 2 API requests after cache clear, got %d", requestCount)
	}
}

// Helper function to check if a string contains a substring (integration tests)
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

type mockDesktopClient struct {
	getUserInfo func(accessToken string) (*UsageLimitsResponse, error)
}

func (m *mockDesktopClient) InitiateLogin(provider, redirectUri, codeChallenge, state string) string {
	return ""
}

func (m *mockDesktopClient) ExchangeToken(code, codeVerifier, redirectUri string) (*DesktopExchangeTokenResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDesktopClient) RefreshToken(refreshToken string) (*DesktopExchangeTokenResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDesktopClient) GetUserInfo(accessToken string) (*UsageLimitsResponse, error) {
	return m.getUserInfo(accessToken)
}

type mockUsageClient struct {
	handler func(token string) (*UsageLimitsResponse, error)
}

func (m *mockUsageClient) GetUserInfo(accessToken string) (*UsageLimitsResponse, error) {
	return m.handler(accessToken)
}
