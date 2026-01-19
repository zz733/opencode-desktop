package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestValidateToken tests the ValidateToken method
func TestValidateToken(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		serverResponse *TokenInfo
		serverStatus   int
		expectError    bool
		errorContains  string
	}{
		{
			name:  "valid token",
			token: "valid-bearer-token-with-sufficient-length",
			serverResponse: &TokenInfo{
				AccessToken:  "valid-bearer-token-with-sufficient-length",
				RefreshToken: "refresh-token-123",
				ExpiresAt:    time.Now().Add(24 * time.Hour),
				TokenType:    "Bearer",
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
			name:          "token too short",
			token:         "short",
			serverStatus:  http.StatusOK,
			expectError:   true,
			errorContains: "token is too short",
		},
		{
			name:          "token with invalid characters",
			token:         "invalid-token-with-special-chars-!@#$%^&*()",
			serverStatus:  http.StatusOK,
			expectError:   true,
			errorContains: "invalid characters",
		},
		{
			name:          "invalid token - 401",
			token:         "invalid-token-but-valid-format-length",
			serverStatus:  http.StatusUnauthorized,
			expectError:   true,
			errorContains: "invalid or expired",
		},
		{
			name:          "forbidden - 403",
			token:         "valid-token-but-no-permissions-here",
			serverStatus:  http.StatusForbidden,
			expectError:   true,
			errorContains: "does not have required permissions",
		},
		{
			name:          "rate limited - 429",
			token:         "valid-token-but-rate-limited-now",
			serverStatus:  http.StatusTooManyRequests,
			expectError:   true,
			errorContains: "rate limit exceeded",
		},
		{
			name:  "token expiring soon",
			token: "valid-token-expiring-soon-enough",
			serverResponse: &TokenInfo{
				AccessToken:  "valid-token-expiring-soon-enough",
				RefreshToken: "refresh-token-123",
				ExpiresAt:    time.Now().Add(2 * time.Minute), // Expires in 2 minutes
				TokenType:    "Bearer",
			},
			serverStatus: http.StatusOK,
			expectError:  false, // Should succeed but with warning
		},
		{
			name:  "expired token",
			token: "expired-token-with-valid-format",
			serverResponse: &TokenInfo{
				AccessToken:  "expired-token-with-valid-format",
				RefreshToken: "refresh-token-123",
				ExpiresAt:    time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
				TokenType:    "Bearer",
			},
			serverStatus:  http.StatusOK,
			expectError:   true,
			errorContains: "token has expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/auth/validate" {
					t.Errorf("Expected path /auth/validate, got %s", r.URL.Path)
				}
				
				// Verify Authorization header
				authHeader := r.Header.Get("Authorization")
				if tt.token != "" && len(tt.token) >= 20 && !strings.Contains(tt.token, "!@#") {
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
				BaseURL:         server.URL,
				AuthValidateURL: server.URL + "/auth/validate",
				Timeout:         30,
			}
			authService := NewAuthServiceWithConfig(config)
			tokenInfo, err := authService.ValidateToken(tt.token)

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
				if tokenInfo == nil {
					t.Errorf("Expected tokenInfo but got nil")
				}
			}
		})
	}
}

// TestValidateTokenFormat tests the validateTokenFormat method
func TestValidateTokenFormat(t *testing.T) {
	tests := []struct {
		name          string
		token         string
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid token",
			token:       "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
			expectError: false,
		},
		{
			name:        "valid token with Bearer prefix",
			token:       "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
			expectError: false,
		},
		{
			name:          "token too short",
			token:         "short",
			expectError:   true,
			errorContains: "too short",
		},
		{
			name:          "token too long",
			token:         strings.Repeat("a", 2049),
			expectError:   true,
			errorContains: "too long",
		},
		{
			name:          "token with invalid characters",
			token:         "invalid-token-with-!@#$%^&*()",
			expectError:   true,
			errorContains: "invalid characters",
		},
		{
			name:        "token with valid special characters",
			token:       "valid_token-with.special~chars+and/equals=",
			expectError: false,
		},
	}

	authService := NewAuthService()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := authService.validateTokenFormat(tt.token)
			
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
			}
		})
	}
}

// TestValidateTokenExpiry tests the validateTokenExpiry method
func TestValidateTokenExpiry(t *testing.T) {
	tests := []struct {
		name          string
		tokenInfo     *TokenInfo
		expectError   bool
		errorContains string
	}{
		{
			name: "valid token - expires in future",
			tokenInfo: &TokenInfo{
				AccessToken:  "valid-token",
				RefreshToken: "refresh-token",
				ExpiresAt:    time.Now().Add(24 * time.Hour),
				TokenType:    "Bearer",
			},
			expectError: false,
		},
		{
			name: "token expiring soon - still valid",
			tokenInfo: &TokenInfo{
				AccessToken:  "valid-token",
				RefreshToken: "refresh-token",
				ExpiresAt:    time.Now().Add(2 * time.Minute),
				TokenType:    "Bearer",
			},
			expectError: false, // Should succeed but print warning
		},
		{
			name: "expired token",
			tokenInfo: &TokenInfo{
				AccessToken:  "expired-token",
				RefreshToken: "refresh-token",
				ExpiresAt:    time.Now().Add(-1 * time.Hour),
				TokenType:    "Bearer",
			},
			expectError:   true,
			errorContains: "token has expired",
		},
		{
			name:          "nil token info",
			tokenInfo:     nil,
			expectError:   true,
			errorContains: "token info is nil",
		},
	}

	authService := NewAuthService()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := authService.validateTokenExpiry(tt.tokenInfo)
			
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
			}
		})
	}
}

// TestValidateTokenWithRetry tests the ValidateTokenWithRetry method
func TestValidateTokenWithRetry(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		maxRetries     int
		serverBehavior func(attempt *int) (int, *TokenInfo)
		expectError    bool
		errorContains  string
		minAttempts    int
	}{
		{
			name:       "success on first attempt",
			token:      "valid-token-succeeds-immediately",
			maxRetries: 3,
			serverBehavior: func(attempt *int) (int, *TokenInfo) {
				*attempt++
				return http.StatusOK, &TokenInfo{
					AccessToken:  "valid-token-succeeds-immediately",
					RefreshToken: "refresh-token",
					ExpiresAt:    time.Now().Add(24 * time.Hour),
					TokenType:    "Bearer",
				}
			},
			expectError: false,
			minAttempts: 1,
		},
		{
			name:       "success after retry",
			token:      "valid-token-succeeds-after-retry",
			maxRetries: 3,
			serverBehavior: func(attempt *int) (int, *TokenInfo) {
				*attempt++
				if *attempt == 1 {
					return http.StatusInternalServerError, nil
				}
				return http.StatusOK, &TokenInfo{
					AccessToken:  "valid-token-succeeds-after-retry",
					RefreshToken: "refresh-token",
					ExpiresAt:    time.Now().Add(24 * time.Hour),
					TokenType:    "Bearer",
				}
			},
			expectError: false,
			minAttempts: 2,
		},
		{
			name:       "invalid token - no retry",
			token:      "invalid-token-no-retry-needed",
			maxRetries: 3,
			serverBehavior: func(attempt *int) (int, *TokenInfo) {
				*attempt++
				return http.StatusUnauthorized, nil
			},
			expectError:   true,
			errorContains: "invalid or expired",
			minAttempts:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempt := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				status, tokenInfo := tt.serverBehavior(&attempt)
				w.WriteHeader(status)
				if status == http.StatusOK && tokenInfo != nil {
					json.NewEncoder(w).Encode(tokenInfo)
				}
			}))
			defer server.Close()

			config := &KiroAPIConfig{
				BaseURL:         server.URL,
				AuthValidateURL: server.URL + "/auth/validate",
				Timeout:         5, // Shorter timeout for tests
			}
			authService := NewAuthServiceWithConfig(config)
			
			tokenInfo, err := authService.ValidateTokenWithRetry(tt.token, tt.maxRetries)

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
				if tokenInfo == nil {
					t.Errorf("Expected tokenInfo but got nil")
				}
			}

			if attempt < tt.minAttempts {
				t.Errorf("Expected at least %d attempts, got %d", tt.minAttempts, attempt)
			}
		})
	}
}

// TestGetUserProfile tests the GetUserProfile method
func TestGetUserProfile(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		serverResponse *UserProfile
		serverStatus   int
		expectError    bool
		errorContains  string
	}{
		{
			name:  "valid profile request",
			token: "valid-token",
			serverResponse: &UserProfile{
				ID:       "user-123",
				Email:    "user@example.com",
				Name:     "Test User",
				Avatar:   "https://example.com/avatar.jpg",
				Provider: "google",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/user/profile" {
					t.Errorf("Expected path /user/profile, got %s", r.URL.Path)
				}
				w.WriteHeader(tt.serverStatus)
				if tt.serverStatus == http.StatusOK && tt.serverResponse != nil {
					json.NewEncoder(w).Encode(tt.serverResponse)
				}
			}))
			defer server.Close()

			config := &KiroAPIConfig{
				BaseURL:        server.URL,
				UserProfileURL: server.URL + "/user/profile",
				Timeout:        30,
			}
			authService := NewAuthServiceWithConfig(config)
			profile, err := authService.GetUserProfile(tt.token)

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
				if profile == nil {
					t.Errorf("Expected profile but got nil")
				}
			}
		})
	}
}

// TestValidateAndCreateAccount tests the ValidateAndCreateAccount method
func TestValidateAndCreateAccount(t *testing.T) {
	validToken := "valid-bearer-token-with-sufficient-length-for-validation"
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/validate":
			tokenInfo := TokenInfo{
				AccessToken:  validToken,
				RefreshToken: "refresh-token-with-sufficient-length",
				ExpiresAt:    time.Now().Add(24 * time.Hour),
				TokenType:    "Bearer",
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(tokenInfo)
		case "/user/profile":
			profile := UserProfile{
				ID:       "user-123",
				Email:    "user@example.com",
				Name:     "Test User",
				Avatar:   "https://example.com/avatar.jpg",
				Provider: "google",
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(profile)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := &KiroAPIConfig{
		BaseURL:         server.URL,
		AuthValidateURL: server.URL + "/auth/validate",
		UserProfileURL:  server.URL + "/user/profile",
		UserQuotaURL:    server.URL + "/user/quota",
		Timeout:         30,
	}
	authService := NewAuthServiceWithConfig(config)

	// Create a quota service for the test
	quotaService := NewQuotaServiceWithConfig(config)

	account, err := authService.ValidateAndCreateAccount(validToken, LoginMethodToken, "", quotaService)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if account.Email != "user@example.com" {
		t.Errorf("Expected Email 'user@example.com', got '%s'", account.Email)
	}
	if account.DisplayName != "Test User" {
		t.Errorf("Expected DisplayName 'Test User', got '%s'", account.DisplayName)
	}
	if account.BearerToken != validToken {
		t.Errorf("Expected BearerToken '%s', got '%s'", validToken, account.BearerToken)
	}
	if account.LoginMethod != LoginMethodToken {
		t.Errorf("Expected LoginMethod 'token', got '%s'", account.LoginMethod)
	}
}

// TestLoginWithPassword tests the LoginWithPassword method
func TestLoginWithPassword(t *testing.T) {
	tests := []struct {
		name           string
		email          string
		password       string
		serverResponse map[string]interface{}
		serverStatus   int
		expectError    bool
		errorContains  string
	}{
		{
			name:     "successful login",
			email:    "user@example.com",
			password: "password123",
			serverResponse: map[string]interface{}{
				"bearer_token":  "test-bearer-token",
				"refresh_token": "test-refresh-token",
				"expires_in":    3600,
				"token_type":    "Bearer",
			},
			serverStatus: http.StatusOK,
			expectError:  false,
		},
		{
			name:          "empty email",
			email:         "",
			password:      "password123",
			serverStatus:  http.StatusOK,
			expectError:   true,
			errorContains: "email cannot be empty",
		},
		{
			name:          "empty password",
			email:         "user@example.com",
			password:      "",
			serverStatus:  http.StatusOK,
			expectError:   true,
			errorContains: "password cannot be empty",
		},
		{
			name:          "invalid credentials",
			email:         "user@example.com",
			password:      "wrongpassword",
			serverStatus:  http.StatusUnauthorized,
			expectError:   true,
			errorContains: "invalid email or password",
		},
		{
			name:          "server error",
			email:         "user@example.com",
			password:      "password123",
			serverStatus:  http.StatusInternalServerError,
			expectError:   true,
			errorContains: "login failed with status: 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request path
				if r.URL.Path == "/auth/login" {
					w.WriteHeader(tt.serverStatus)
					if tt.serverStatus == http.StatusOK && tt.serverResponse != nil {
						json.NewEncoder(w).Encode(tt.serverResponse)
					}
				} else if r.URL.Path == "/user/profile" {
					// Mock user profile response
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(UserProfile{
						ID:       "user-123",
						Email:    tt.email,
						Name:     "Test User",
						Avatar:   "https://example.com/avatar.jpg",
						Provider: "password",
					})
				}
			}))
			defer server.Close()

			// Create auth service with mock server URL
			config := &KiroAPIConfig{
				BaseURL:        server.URL,
				AuthLoginURL:   server.URL + "/auth/login",
				UserProfileURL: server.URL + "/user/profile",
				Timeout:        30,
			}
			authService := NewAuthServiceWithConfig(config)

			// Test login
			account, err := authService.LoginWithPassword(tt.email, tt.password)

			// Verify results
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
				if account == nil {
					t.Errorf("Expected account but got nil")
				} else {
					// Verify account properties
					if account.Email != tt.email {
						t.Errorf("Expected email %s, got %s", tt.email, account.Email)
					}
					if account.LoginMethod != LoginMethodPassword {
						t.Errorf("Expected login method 'password', got '%s'", account.LoginMethod)
					}
					if account.BearerToken == "" {
						t.Errorf("Expected bearer token to be set")
					}
				}
			}
		})
	}
}
