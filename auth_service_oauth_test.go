package main

import (
	"strings"
	"testing"
)

// newTestAuthService creates an AuthService with test OAuth configurations
func newTestAuthService() *AuthService {
	config := DefaultKiroAPIConfig()
	return NewAuthServiceWithConfig(config)
}

// TestStartOAuthFlow tests the OAuth flow initialization
func TestStartOAuthFlow(t *testing.T) {
	as := newTestAuthService()

	tests := []struct {
		name     string
		provider OAuthProvider
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "Google OAuth",
			provider: ProviderGoogle,
			wantErr:  false,
		},
		{
			name:     "GitHub OAuth",
			provider: ProviderGitHub,
			wantErr:  false,
		},
		{
			name:     "AWS Builder ID OAuth",
			provider: ProviderBuilderID,
			wantErr:  true,
			errMsg:   "unsupported provider for Web OAuth",
		},
		{
			name:     "Invalid provider",
			provider: OAuthProvider("invalid"),
			wantErr:  true,
			errMsg:   "unsupported provider for Web OAuth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authURL, err := as.StartOAuthFlow(tt.provider)

			if tt.wantErr {
				if err == nil {
					t.Errorf("StartOAuthFlow() expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("StartOAuthFlow() error = %v, want error containing %v", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("StartOAuthFlow() unexpected error = %v", err)
				return
			}

			if authURL == "" {
				t.Errorf("StartOAuthFlow() returned empty URL")
			}

			// Verify URL contains expected components
			switch tt.provider {
			case ProviderGoogle:
				if !strings.Contains(authURL, "idp=Google") {
					t.Errorf("Google OAuth URL should contain idp=Google, got %s", authURL)
				}
			case ProviderGitHub:
				if !strings.Contains(authURL, "idp=Github") {
					t.Errorf("GitHub OAuth URL should contain idp=Github, got %s", authURL)
				}
			}

			// Verify URL contains state parameter
			if !strings.Contains(authURL, "state=") {
				t.Errorf("OAuth URL should contain state parameter, got %s", authURL)
			}
		})
	}
}

// TestGenerateStateToken tests state token generation
func TestGenerateStateToken(t *testing.T) {
	as := newTestAuthService()

	// Generate multiple tokens
	tokens := make(map[string]bool)
	for i := 0; i < 10; i++ {
		token, err := as.generateStateToken()
		if err != nil {
			t.Fatalf("generateStateToken() error = %v", err)
		}

		if token == "" {
			t.Error("generateStateToken() returned empty token")
		}

		// Check for uniqueness
		if tokens[token] {
			t.Errorf("generateStateToken() generated duplicate token: %s", token)
		}
		tokens[token] = true

		// Check token length (base64 encoded 32 bytes should be ~44 chars)
		if len(token) < 40 {
			t.Errorf("generateStateToken() token too short: %d chars", len(token))
		}
	}
}

// TestHandleOAuthCallback tests OAuth callback handling
func TestHandleOAuthCallback(t *testing.T) {
	as := newTestAuthService()

	tests := []struct {
		name    string
		state   string
		code    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Empty code",
			state:   "state-1",
			code:    "",
			wantErr: true,
			errMsg:  "authorization code is required",
		},
		{
			name:    "Invalid state",
			state:   "missing-state",
			code:    "test-code",
			wantErr: true,
			errMsg:  "invalid or expired state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account, err := as.HandleOAuthCallback(tt.state, tt.code)

			if tt.wantErr {
				if err == nil {
					t.Errorf("HandleOAuthCallback() expected error but got none")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("HandleOAuthCallback() error = %v, want error containing %v", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Logf("HandleOAuthCallback() error = %v (expected in test without real OAuth)", err)
				return
			}

			if account == nil {
				t.Error("HandleOAuthCallback() returned nil account")
			}
		})
	}
}

// TestOAuthStateManagement tests state token management
func TestOAuthStateManagement(t *testing.T) {
	as := newTestAuthService()

	// Start OAuth flow to generate state
	authURL, err := as.StartOAuthFlow(ProviderGoogle)
	if err != nil {
		t.Fatalf("StartOAuthFlow() error = %v", err)
	}

	// Extract state from URL
	if !strings.Contains(authURL, "state=") {
		t.Fatal("OAuth URL missing state parameter")
	}

	// Verify state is stored
	if len(as.pendingLogins) == 0 {
		t.Error("OAuth state not stored after StartOAuthFlow()")
	}

	// Start multiple flows
	for i := 0; i < 3; i++ {
		_, err := as.StartOAuthFlow(ProviderGitHub)
		if err != nil {
			t.Errorf("StartOAuthFlow() iteration %d error = %v", i, err)
		}
	}

	// Verify multiple states are stored
	if len(as.pendingLogins) < 4 {
		t.Errorf("Expected at least 4 stored states, got %d", len(as.pendingLogins))
	}
}

// TestOAuthIntegrationWithAccountCreation tests OAuth flow integration
func TestOAuthIntegrationWithAccountCreation(t *testing.T) {
	as := newTestAuthService()

	// Test that OAuth methods return proper error messages
	// when called without valid OAuth setup

	t.Run("StartOAuthFlow returns URL", func(t *testing.T) {
		url, err := as.StartOAuthFlow(ProviderGoogle)
		if err != nil {
			t.Errorf("StartOAuthFlow() error = %v", err)
		}
		if url == "" {
			t.Error("StartOAuthFlow() returned empty URL")
		}
	})

	t.Run("HandleOAuthCallback validates input", func(t *testing.T) {
		_, err := as.HandleOAuthCallback("", "")
		if err == nil {
			t.Error("HandleOAuthCallback() should fail with empty code")
		}
		if !strings.Contains(err.Error(), "authorization code is required") {
			t.Errorf("HandleOAuthCallback() wrong error message: %v", err)
		}
	})
}

// Benchmark OAuth state token generation
func BenchmarkGenerateStateToken(b *testing.B) {
	as := newTestAuthService()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := as.generateStateToken()
		if err != nil {
			b.Fatalf("generateStateToken() error = %v", err)
		}
	}
}

// Benchmark OAuth flow initialization
func BenchmarkStartOAuthFlow(b *testing.B) {
	as := newTestAuthService()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := as.StartOAuthFlow(ProviderGoogle)
		if err != nil {
			b.Fatalf("StartOAuthFlow() error = %v", err)
		}
	}
}
