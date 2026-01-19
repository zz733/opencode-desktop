package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestGetUserProfileIntegration tests the complete user profile retrieval flow
func TestGetUserProfileIntegration(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		responseStatus int
		responseBody   interface{}
		expectError    bool
		errorContains  string
	}{
		{
			name:           "successful profile retrieval",
			token:          "valid-bearer-token-with-sufficient-length",
			responseStatus: http.StatusOK,
			responseBody: UserProfile{
				ID:       "user123",
				Email:    "test@example.com",
				Name:     "Test User",
				Avatar:   "https://example.com/avatar.jpg",
				Provider: "kiro",
			},
			expectError: false,
		},
		{
			name:           "unauthorized token",
			token:          "invalid-token-with-sufficient-length",
			responseStatus: http.StatusUnauthorized,
			responseBody:   map[string]string{"error": "unauthorized"},
			expectError:    true,
			errorContains:  "invalid or expired",
		},
		{
			name:           "forbidden access",
			token:          "valid-but-forbidden-token-with-length",
			responseStatus: http.StatusForbidden,
			responseBody:   map[string]string{"error": "forbidden"},
			expectError:    true,
			errorContains:  "required permissions",
		},
		{
			name:           "profile not found",
			token:          "valid-token-but-no-profile-found-here",
			responseStatus: http.StatusNotFound,
			responseBody:   map[string]string{"error": "not found"},
			expectError:    true,
			errorContains:  "not found",
		},
		{
			name:           "rate limit exceeded",
			token:          "valid-token-but-rate-limited-now-here",
			responseStatus: http.StatusTooManyRequests,
			responseBody:   map[string]string{"error": "rate limit"},
			expectError:    true,
			errorContains:  "rate limit",
		},
		{
			name:           "missing email in profile",
			token:          "valid-token-with-incomplete-profile",
			responseStatus: http.StatusOK,
			responseBody: UserProfile{
				ID:     "user123",
				Email:  "", // Missing email
				Name:   "Test User",
				Avatar: "https://example.com/avatar.jpg",
			},
			expectError:   true,
			errorContains: "missing required field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request headers
				if r.Header.Get("Authorization") != "Bearer "+tt.token {
					t.Errorf("Expected Authorization header 'Bearer %s', got '%s'", tt.token, r.Header.Get("Authorization"))
				}
				if r.Header.Get("User-Agent") != "Kiro-Account-Manager/1.0" {
					t.Errorf("Expected User-Agent 'Kiro-Account-Manager/1.0', got '%s'", r.Header.Get("User-Agent"))
				}

				w.WriteHeader(tt.responseStatus)
				json.NewEncoder(w).Encode(tt.responseBody)
			}))
			defer server.Close()

			// Create auth service with test server URL
			config := &KiroAPIConfig{
				BaseURL:        server.URL,
				UserProfileURL: server.URL + "/user/profile",
				Timeout:        30,
			}
			authService := NewAuthServiceWithConfig(config)

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
					expectedProfile := tt.responseBody.(UserProfile)
					if profile.Email != expectedProfile.Email {
						t.Errorf("Expected email '%s', got '%s'", expectedProfile.Email, profile.Email)
					}
					if profile.Name != expectedProfile.Name {
						t.Errorf("Expected name '%s', got '%s'", expectedProfile.Name, profile.Name)
					}
				}
			}
		})
	}
}

// TestGetQuotaIntegration tests the complete quota retrieval flow
func TestGetQuotaIntegration(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		responseStatus int
		responseBody   interface{}
		expectError    bool
		errorContains  string
	}{
		{
			name:           "successful quota retrieval - direct format",
			token:          "valid-bearer-token-with-sufficient-length",
			responseStatus: http.StatusOK,
			responseBody: QuotaInfo{
				Main:   QuotaDetail{Used: 100, Total: 1000},
				Trial:  QuotaDetail{Used: 50, Total: 100},
				Reward: QuotaDetail{Used: 0, Total: 200},
			},
			expectError: false,
		},
		{
			name:           "successful quota retrieval - wrapped format",
			token:          "valid-bearer-token-with-sufficient-length",
			responseStatus: http.StatusOK,
			responseBody: map[string]interface{}{
				"quota": QuotaInfo{
					Main:   QuotaDetail{Used: 200, Total: 2000},
					Trial:  QuotaDetail{Used: 25, Total: 100},
					Reward: QuotaDetail{Used: 10, Total: 300},
				},
			},
			expectError: false,
		},
		{
			name:           "unauthorized token",
			token:          "invalid-token-with-sufficient-length",
			responseStatus: http.StatusUnauthorized,
			responseBody:   map[string]string{"error": "unauthorized"},
			expectError:    true,
			errorContains:  "invalid or expired",
		},
		{
			name:           "quota not found",
			token:          "valid-token-but-no-quota-found-here",
			responseStatus: http.StatusNotFound,
			responseBody:   map[string]string{"error": "not found"},
			expectError:    true,
			errorContains:  "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request headers
				if r.Header.Get("Authorization") != "Bearer "+tt.token {
					t.Errorf("Expected Authorization header 'Bearer %s', got '%s'", tt.token, r.Header.Get("Authorization"))
				}

				w.WriteHeader(tt.responseStatus)
				json.NewEncoder(w).Encode(tt.responseBody)
			}))
			defer server.Close()

			// Create quota service with test server URL
			config := &KiroAPIConfig{
				BaseURL:      server.URL,
				UserQuotaURL: server.URL + "/user/quota",
				Timeout:      30,
			}
			quotaService := NewQuotaServiceWithConfig(config)

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
				}
			}
		})
	}
}

// TestValidateAndCreateAccountIntegration tests the complete account creation flow
func TestValidateAndCreateAccountIntegration(t *testing.T) {
	validToken := "valid-bearer-token-with-sufficient-length-for-testing"

	// Create test server that handles multiple endpoints
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/validate":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(TokenInfo{
				AccessToken:  validToken,
				RefreshToken: "refresh-token",
				ExpiresAt:    time.Now().Add(24 * time.Hour),
				TokenType:    "Bearer",
			})
		case "/user/profile":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(UserProfile{
				ID:       "user123",
				Email:    "test@example.com",
				Name:     "Test User",
				Avatar:   "https://example.com/avatar.jpg",
				Provider: "kiro",
			})
		case "/user/quota":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(QuotaInfo{
				Main:   QuotaDetail{Used: 100, Total: 1000},
				Trial:  QuotaDetail{Used: 50, Total: 100},
				Reward: QuotaDetail{Used: 0, Total: 200},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create services with test server URL
	config := &KiroAPIConfig{
		BaseURL:         server.URL,
		AuthValidateURL: server.URL + "/auth/validate",
		UserProfileURL:  server.URL + "/user/profile",
		UserQuotaURL:    server.URL + "/user/quota",
		Timeout:         30,
	}
	authService := NewAuthServiceWithConfig(config)
	quotaService := NewQuotaServiceWithConfig(config)

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
	if account.DisplayName != "Test User" {
		t.Errorf("Expected name 'Test User', got '%s'", account.DisplayName)
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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(QuotaInfo{
			Main:   QuotaDetail{Used: 100, Total: 1000},
			Trial:  QuotaDetail{Used: 50, Total: 100},
			Reward: QuotaDetail{Used: 0, Total: 200},
		})
	}))
	defer server.Close()

	config := &KiroAPIConfig{
		BaseURL:      server.URL,
		UserQuotaURL: server.URL + "/user/quota",
		Timeout:      30,
	}
	quotaService := NewQuotaServiceWithConfig(config)

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
