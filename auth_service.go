package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

// AuthService handles authentication operations for Kiro accounts
type AuthService struct {
	httpClient   *http.Client
	config       *KiroAPIConfig
	oauthConfigs map[OAuthProvider]*oauth2.Config
	oauthStates  map[string]OAuthProvider // state -> provider mapping for security
}

// NewAuthService creates a new AuthService instance
func NewAuthService() *AuthService {
	config := DefaultKiroAPIConfig()

	as := &AuthService{
		httpClient: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
		config:       config,
		oauthConfigs: make(map[OAuthProvider]*oauth2.Config),
		oauthStates:  make(map[string]OAuthProvider),
	}

	// Initialize OAuth configurations
	as.initOAuthConfigs()

	return as
}

// NewAuthServiceWithConfig creates a new AuthService instance with custom configuration
func NewAuthServiceWithConfig(config *KiroAPIConfig) *AuthService {
	if err := config.Validate(); err != nil {
		// Fall back to default config if validation fails
		config = DefaultKiroAPIConfig()
	}

	as := &AuthService{
		httpClient: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
		config:       config,
		oauthConfigs: make(map[OAuthProvider]*oauth2.Config),
		oauthStates:  make(map[string]OAuthProvider),
	}

	// Initialize OAuth configurations
	as.initOAuthConfigs()

	return as
}

// initOAuthConfigs initializes OAuth configurations for different providers
func (as *AuthService) initOAuthConfigs() {
	// Google OAuth configuration
	as.oauthConfigs[ProviderGoogle] = &oauth2.Config{
		ClientID:     getEnvOrDefault("GOOGLE_CLIENT_ID", ""),
		ClientSecret: getEnvOrDefault("GOOGLE_CLIENT_SECRET", ""),
		RedirectURL:  getEnvOrDefault("OAUTH_REDIRECT_URL", "http://localhost:34115/oauth/callback"),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	// GitHub OAuth configuration
	as.oauthConfigs[ProviderGitHub] = &oauth2.Config{
		ClientID:     getEnvOrDefault("GITHUB_CLIENT_ID", ""),
		ClientSecret: getEnvOrDefault("GITHUB_CLIENT_SECRET", ""),
		RedirectURL:  getEnvOrDefault("OAUTH_REDIRECT_URL", "http://localhost:34115/oauth/callback"),
		Scopes: []string{
			"user:email",
			"read:user",
		},
		Endpoint: github.Endpoint,
	}

	// AWS Builder ID OAuth configuration
	as.oauthConfigs[ProviderBuilderID] = &oauth2.Config{
		ClientID:     getEnvOrDefault("AWS_BUILDERID_CLIENT_ID", ""),
		ClientSecret: getEnvOrDefault("AWS_BUILDERID_CLIENT_SECRET", ""),
		RedirectURL:  getEnvOrDefault("OAUTH_REDIRECT_URL", "http://localhost:34115/oauth/callback"),
		Scopes: []string{
			"openid",
			"profile",
			"email",
		},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://auth.aws.amazon.com/oauth2/authorize",
			TokenURL: "https://auth.aws.amazon.com/oauth2/token",
		},
	}
}

// getEnvOrDefault gets environment variable or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	// TODO: Implement proper environment variable reading
	// For now, return default value
	return defaultValue
}

// ValidateToken validates a bearer token and returns token information
// This method performs both local format validation and remote API validation
func (as *AuthService) ValidateToken(token string) (*TokenInfo, error) {
	// Step 1: Basic validation
	if token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	// Step 2: Format validation
	if err := as.validateTokenFormat(token); err != nil {
		return nil, fmt.Errorf("invalid token format: %w", err)
	}

	// Step 3: Remote validation via Kiro API
	req, err := http.NewRequest("GET", as.config.AuthValidateURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authorization header
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Kiro-Account-Manager/1.0")

	// Send request
	resp, err := as.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Step 4: Handle response status with detailed error messages
	switch resp.StatusCode {
	case http.StatusOK:
		// Success - continue to parse response
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("token is invalid or expired")
	case http.StatusForbidden:
		return nil, fmt.Errorf("token does not have required permissions")
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("rate limit exceeded, please try again later")
	default:
		return nil, fmt.Errorf("token validation failed with status: %d", resp.StatusCode)
	}

	// Step 5: Parse response
	var tokenInfo TokenInfo
	if err := json.NewDecoder(resp.Body).Decode(&tokenInfo); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Step 6: Validate token expiry
	if err := as.validateTokenExpiry(&tokenInfo); err != nil {
		return nil, fmt.Errorf("token expiry validation failed: %w", err)
	}

	return &tokenInfo, nil
}

// validateTokenFormat validates the format of a bearer token
// Checks for basic structure and length requirements
func (as *AuthService) validateTokenFormat(token string) error {
	// Remove "Bearer " prefix if present
	token = strings.TrimPrefix(token, "Bearer ")
	token = strings.TrimSpace(token)

	// Check minimum length (typical JWT or API tokens are at least 20 characters)
	if len(token) < 20 {
		return fmt.Errorf("token is too short (minimum 20 characters)")
	}

	// Check maximum length (prevent extremely long tokens)
	if len(token) > 2048 {
		return fmt.Errorf("token is too long (maximum 2048 characters)")
	}

	// Check for invalid characters (tokens should be alphanumeric with some special chars)
	// Allow: letters, numbers, dots, hyphens, underscores, tildes, plus, slash, equals
	for _, char := range token {
		if !isValidTokenChar(char) {
			return fmt.Errorf("token contains invalid characters")
		}
	}

	return nil
}

// isValidTokenChar checks if a character is valid in a token
func isValidTokenChar(char rune) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9') ||
		char == '-' || char == '_' || char == '.' ||
		char == '~' || char == '+' || char == '/' || char == '='
}

// validateTokenExpiry checks if a token has expired or is about to expire
func (as *AuthService) validateTokenExpiry(tokenInfo *TokenInfo) error {
	if tokenInfo == nil {
		return fmt.Errorf("token info is nil")
	}

	// Check if token has already expired
	if time.Now().After(tokenInfo.ExpiresAt) {
		return fmt.Errorf("token has expired at %s", tokenInfo.ExpiresAt.Format(time.RFC3339))
	}

	// Warn if token expires soon (within 5 minutes)
	if time.Until(tokenInfo.ExpiresAt) < 5*time.Minute {
		// Note: This is a warning, not an error
		// The token is still valid but should be refreshed soon
		fmt.Printf("Warning: token expires soon at %s\n", tokenInfo.ExpiresAt.Format(time.RFC3339))
	}

	return nil
}

// ValidateTokenWithRetry validates a token with automatic retry on transient failures
func (as *AuthService) ValidateTokenWithRetry(token string, maxRetries int) (*TokenInfo, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		tokenInfo, err := as.ValidateToken(token)
		if err == nil {
			return tokenInfo, nil
		}

		lastErr = err

		// Don't retry on authentication errors (invalid token)
		if strings.Contains(err.Error(), "invalid or expired") ||
			strings.Contains(err.Error(), "invalid token format") {
			return nil, err
		}

		// Don't retry on the last attempt
		if attempt < maxRetries {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			time.Sleep(backoff)
		}
	}

	return nil, fmt.Errorf("token validation failed after %d attempts: %w", maxRetries+1, lastErr)
}

// RefreshToken refreshes an access token using a refresh token
// Kiro API 使用 refreshToken 字段（camelCase）
func (as *AuthService) RefreshToken(refreshToken string) (*TokenInfo, error) {
	if refreshToken == "" {
		return nil, fmt.Errorf("refresh token cannot be empty")
	}

	// Log the API endpoint being used (for debugging)
	fmt.Printf("Attempting to refresh token using endpoint: %s\n", as.config.AuthRefreshURL)

	// Check if we are using the placeholder URL
	if strings.Contains(as.config.AuthRefreshURL, "api.kiro.ai") {
		return nil, fmt.Errorf("API URL not configured (using placeholder)")
	}

	// Create refresh request payload - Kiro 使用 camelCase
	payload := map[string]string{
		"refreshToken": refreshToken,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", as.config.AuthRefreshURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Kiro-Account-Manager/1.0")

	// Send request with retry
	var resp *http.Response
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Second)
		}

		resp, lastErr = as.httpClient.Do(req)
		if lastErr == nil {
			break
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to send request after 3 attempts: %w", lastErr)
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check response status
	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("RefreshToken 已过期或无效")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse Kiro API response format
	var kiroResp struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		ExpiresIn    int64  `json:"expiresIn"`
		ProfileArn   string `json:"profileArn"`
		CsrfToken    string `json:"csrfToken,omitempty"`
	}

	if err := json.Unmarshal(bodyBytes, &kiroResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to TokenInfo
	expiresAt := time.Now().Add(time.Duration(kiroResp.ExpiresIn) * time.Second)
	tokenInfo := &TokenInfo{
		AccessToken:  kiroResp.AccessToken,
		RefreshToken: kiroResp.RefreshToken,
		ExpiresAt:    expiresAt,
		TokenType:    "Bearer",
	}

	return tokenInfo, nil
}

// GetUserProfile retrieves user profile information using a bearer token
// Kiro 的用户信息包含在 getUsageLimits API 响应中
func (as *AuthService) GetUserProfile(token string) (*UserProfile, error) {
	if token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	// Kiro 使用 getUsageLimits API 获取用户信息
	// Check if we are using the placeholder URL
	if strings.Contains(as.config.UserProfileURL, "api.kiro.ai") {
		return nil, fmt.Errorf("API URL not configured (using placeholder)")
	}

	requestURL := buildUsageLimitsURL(as.config.UserProfileURL, as.config.ProfileARN)

	// Create request
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authorization header
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-amz-user-agent", "aws-sdk-js/1.0.0 KiroIDE-0.6.18-"+getKiroMachineID())
	req.Header.Set("User-Agent", "Kiro-Account-Manager/1.0")
	req.Header.Set("amz-sdk-request", "attempt=1; max=1")
	req.Header.Set("Connection", "close")

	// Send request with retry
	var resp *http.Response
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Second)
		}

		resp, lastErr = as.httpClient.Do(req)
		if lastErr == nil {
			break
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to send request after 3 attempts: %w", lastErr)
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("token is invalid or expired")
	case http.StatusForbidden:
		return nil, fmt.Errorf("token does not have required permissions")
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("rate limit exceeded, please try again later")
	default:
		var errorResp struct {
			Reason string `json:"reason"`
		}
		if json.Unmarshal(bodyBytes, &errorResp) == nil && errorResp.Reason != "" {
			return nil, fmt.Errorf("账号被封禁: %s", errorResp.Reason)
		}
		return nil, fmt.Errorf("failed to get user profile with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var profile UserProfile
	if err := json.Unmarshal(bodyBytes, &profile); err == nil && (profile.Email != "" || profile.ID != "" || profile.Name != "") {
		if profile.Email == "" {
			return nil, fmt.Errorf("missing required field: email")
		}
		return &profile, nil
	}

	var usageResp struct {
		UserInfo *struct {
			Email  string `json:"email"`
			UserID string `json:"userId"`
		} `json:"userInfo"`
	}
	if err := json.Unmarshal(bodyBytes, &usageResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if usageResp.UserInfo == nil {
		return nil, fmt.Errorf("user info not found in response")
	}
	if usageResp.UserInfo.Email == "" {
		return nil, fmt.Errorf("missing required field: email")
	}

	return &UserProfile{
		ID:       usageResp.UserInfo.UserID,
		Email:    usageResp.UserInfo.Email,
		Name:     usageResp.UserInfo.Email,
		Avatar:   "",
		Provider: "kiro",
	}, nil
}

// ValidateAndCreateAccount validates a token and creates a KiroAccount with quota information
func (as *AuthService) ValidateAndCreateAccount(token string, loginMethod LoginMethod, provider OAuthProvider, quotaService *QuotaService) (*KiroAccount, error) {
	// Validate token
	tokenInfo, err := as.ValidateToken(token)
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	// Get user profile
	profile, err := as.GetUserProfile(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	// Get quota information
	quota, err := quotaService.GetQuota(token)
	if err != nil {
		// Log the error but don't fail account creation
		// Use default empty quota if fetch fails
		quota = &QuotaInfo{
			Main:   QuotaDetail{Used: 0, Total: 0},
			Trial:  QuotaDetail{Used: 0, Total: 0},
			Reward: QuotaDetail{Used: 0, Total: 0},
		}
	}

	// Detect subscription type based on quota data
	subscriptionType := detectSubscriptionType(quota)

	// Create account with quota information
	account := &KiroAccount{
		Email:            profile.Email,
		DisplayName:      profile.Name,
		Avatar:           profile.Avatar,
		BearerToken:      token,
		RefreshToken:     tokenInfo.RefreshToken,
		TokenExpiry:      tokenInfo.ExpiresAt,
		LoginMethod:      loginMethod,
		Provider:         provider,
		SubscriptionType: subscriptionType,
		Quota:            *quota,
		Tags:             []string{},
		Notes:            "",
		IsActive:         false,
		LastUsed:         time.Now(),
		CreatedAt:        time.Now(),
	}

	return account, nil
}

// detectSubscriptionType determines the subscription type based on quota information
func detectSubscriptionType(quota *QuotaInfo) SubscriptionType {
	// Logic to detect subscription type based on quota totals
	// This is a heuristic approach - adjust thresholds based on actual Kiro API behavior

	totalQuota := quota.GetTotalAvailable()

	// Pro+ typically has very high quota limits (e.g., > 100000)
	if totalQuota > 100000 {
		return SubscriptionProPlus
	}

	// Pro typically has moderate quota limits (e.g., > 10000)
	if totalQuota > 10000 {
		return SubscriptionPro
	}

	// Free accounts have limited quota
	// Also check if there's trial quota, which is common for free accounts
	if quota.Trial.Total > 0 {
		return SubscriptionFree
	}

	// Default to free if we can't determine
	return SubscriptionFree
}

// generateStateToken generates a secure random state token for OAuth
func (as *AuthService) generateStateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate state token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// StartOAuthFlow starts an OAuth authentication flow
func (as *AuthService) StartOAuthFlow(provider OAuthProvider) (string, error) {
	// Get OAuth config for the provider
	config, ok := as.oauthConfigs[provider]
	if !ok {
		return "", fmt.Errorf("unsupported OAuth provider: %s", provider)
	}

	// Validate that OAuth is configured
	if config.ClientID == "" {
		return "", fmt.Errorf("OAuth not configured for provider %s: missing client ID", provider)
	}

	// Generate secure state token
	state, err := as.generateStateToken()
	if err != nil {
		return "", err
	}

	// Store state -> provider mapping for callback validation
	as.oauthStates[state] = provider

	// Generate authorization URL
	authURL := config.AuthCodeURL(state, oauth2.AccessTypeOffline)

	return authURL, nil
}

// HandleOAuthCallback handles the OAuth callback and creates an account
func (as *AuthService) HandleOAuthCallback(code string, provider OAuthProvider) (*KiroAccount, error) {
	if code == "" {
		return nil, fmt.Errorf("authorization code is required")
	}

	// Get OAuth config for the provider
	config, ok := as.oauthConfigs[provider]
	if !ok {
		return nil, fmt.Errorf("unsupported OAuth provider: %s", provider)
	}

	// Exchange authorization code for token
	ctx := context.Background()
	token, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Get user information from the provider
	userInfo, err := as.getUserInfoFromProvider(token.AccessToken, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Exchange OAuth token for Kiro bearer token
	// This assumes Kiro has an endpoint to exchange OAuth tokens
	kiroToken, err := as.exchangeOAuthTokenForKiroToken(token.AccessToken, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange OAuth token for Kiro token: %w", err)
	}

	// Create token info
	tokenInfo := &TokenInfo{
		AccessToken:  kiroToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    token.Expiry,
		TokenType:    "Bearer",
	}

	// Get user profile from Kiro
	profile, err := as.GetUserProfile(kiroToken)
	if err != nil {
		// If Kiro profile fetch fails, use OAuth provider info
		profile = &UserProfile{
			ID:       userInfo["id"].(string),
			Email:    userInfo["email"].(string),
			Name:     userInfo["name"].(string),
			Avatar:   userInfo["avatar"].(string),
			Provider: string(provider),
		}
	}

	// Create account
	account := &KiroAccount{
		Email:            profile.Email,
		DisplayName:      profile.Name,
		Avatar:           profile.Avatar,
		BearerToken:      tokenInfo.AccessToken,
		RefreshToken:     tokenInfo.RefreshToken,
		TokenExpiry:      tokenInfo.ExpiresAt,
		LoginMethod:      LoginMethodOAuth,
		Provider:         provider,
		SubscriptionType: SubscriptionFree, // Will be updated when quota is fetched
		Quota: QuotaInfo{
			Main:   QuotaDetail{Used: 0, Total: 0},
			Trial:  QuotaDetail{Used: 0, Total: 0},
			Reward: QuotaDetail{Used: 0, Total: 0},
		},
		Tags:      []string{},
		Notes:     "",
		IsActive:  false,
		LastUsed:  time.Now(),
		CreatedAt: time.Now(),
	}

	return account, nil
}

// getUserInfoFromProvider retrieves user information from the OAuth provider
func (as *AuthService) getUserInfoFromProvider(accessToken string, provider OAuthProvider) (map[string]interface{}, error) {
	var userInfoURL string

	switch provider {
	case ProviderGoogle:
		userInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
	case ProviderGitHub:
		userInfoURL = "https://api.github.com/user"
	case ProviderBuilderID:
		userInfoURL = "https://auth.aws.amazon.com/oauth2/userInfo"
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	// Create request
	req, err := http.NewRequest("GET", userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Send request
	resp, err := as.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info, status: %d", resp.StatusCode)
	}

	// Parse response
	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	// Normalize user info across providers
	return as.normalizeUserInfo(userInfo, provider), nil
}

// normalizeUserInfo normalizes user information from different providers
func (as *AuthService) normalizeUserInfo(userInfo map[string]interface{}, provider OAuthProvider) map[string]interface{} {
	normalized := make(map[string]interface{})

	switch provider {
	case ProviderGoogle:
		normalized["id"] = userInfo["id"]
		normalized["email"] = userInfo["email"]
		normalized["name"] = userInfo["name"]
		normalized["avatar"] = userInfo["picture"]

	case ProviderGitHub:
		normalized["id"] = fmt.Sprintf("%v", userInfo["id"])
		normalized["email"] = userInfo["email"]
		if userInfo["email"] == nil {
			// GitHub might not return email, fetch it separately
			normalized["email"] = userInfo["login"].(string) + "@github.com"
		}
		normalized["name"] = userInfo["name"]
		if userInfo["name"] == nil {
			normalized["name"] = userInfo["login"]
		}
		normalized["avatar"] = userInfo["avatar_url"]

	case ProviderBuilderID:
		normalized["id"] = userInfo["sub"]
		normalized["email"] = userInfo["email"]
		normalized["name"] = userInfo["name"]
		if userInfo["picture"] != nil {
			normalized["avatar"] = userInfo["picture"]
		} else {
			normalized["avatar"] = ""
		}
	}

	return normalized
}

// exchangeOAuthTokenForKiroToken exchanges an OAuth token for a Kiro bearer token
func (as *AuthService) exchangeOAuthTokenForKiroToken(oauthToken string, provider OAuthProvider) (string, error) {
	// Create request payload
	payload := map[string]string{
		"oauth_token": oauthToken,
		"provider":    string(provider),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create request to Kiro API
	req, err := http.NewRequest("POST", as.config.OAuthExchangeURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := as.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to exchange token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token exchange failed with status: %d", resp.StatusCode)
	}

	// Parse response
	var result struct {
		BearerToken string `json:"bearer_token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Return bearer token or access token
	if result.BearerToken != "" {
		return result.BearerToken, nil
	}
	if result.AccessToken != "" {
		return result.AccessToken, nil
	}

	return "", fmt.Errorf("no token returned from exchange")
}

// UpdateAccountQuota updates the quota information for an existing account
func (as *AuthService) UpdateAccountQuota(account *KiroAccount, quotaService *QuotaService) error {
	if account == nil {
		return fmt.Errorf("account cannot be nil")
	}

	// Get fresh quota information
	quota, err := quotaService.GetQuota(account.BearerToken)
	if err != nil {
		return fmt.Errorf("failed to get quota: %w", err)
	}

	// Update account quota
	account.Quota = *quota

	// Update subscription type based on new quota data
	account.SubscriptionType = detectSubscriptionType(quota)

	return nil
}

// LoginWithPassword authenticates a user with email and password
// Returns a KiroAccount with bearer token on success
func (as *AuthService) LoginWithPassword(email, password string) (*KiroAccount, error) {
	// Validate input parameters
	if email == "" {
		return nil, fmt.Errorf("email cannot be empty")
	}
	if password == "" {
		return nil, fmt.Errorf("password cannot be empty")
	}

	// Create login request payload
	payload := map[string]string{
		"email":    email,
		"password": password,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal login payload: %w", err)
	}

	// Create HTTP request to Kiro login endpoint
	req, err := http.NewRequest("POST", as.config.AuthLoginURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := as.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send login request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("invalid email or password")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("login failed with status: %d", resp.StatusCode)
	}

	// Parse response to get token information
	var loginResponse struct {
		AccessToken  string `json:"access_token"`
		BearerToken  string `json:"bearer_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"` // seconds
		TokenType    string `json:"token_type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&loginResponse); err != nil {
		return nil, fmt.Errorf("failed to parse login response: %w", err)
	}

	// Determine which token to use (prefer bearer_token, fallback to access_token)
	bearerToken := loginResponse.BearerToken
	if bearerToken == "" {
		bearerToken = loginResponse.AccessToken
	}
	if bearerToken == "" {
		return nil, fmt.Errorf("no token returned from login")
	}

	// Calculate token expiry time
	expiresAt := time.Now().Add(time.Duration(loginResponse.ExpiresIn) * time.Second)
	if loginResponse.ExpiresIn == 0 {
		// Default to 24 hours if not specified
		expiresAt = time.Now().Add(24 * time.Hour)
	}

	// Get user profile using the bearer token
	profile, err := as.GetUserProfile(bearerToken)
	if err != nil {
		// If profile fetch fails, create a basic profile from email
		profile = &UserProfile{
			ID:       email,
			Email:    email,
			Name:     email,
			Avatar:   "",
			Provider: "password",
		}
	}

	// Create account with token information
	account := &KiroAccount{
		Email:            profile.Email,
		DisplayName:      profile.Name,
		Avatar:           profile.Avatar,
		BearerToken:      bearerToken,
		RefreshToken:     loginResponse.RefreshToken,
		TokenExpiry:      expiresAt,
		LoginMethod:      LoginMethodPassword,
		Provider:         "",               // No OAuth provider for password login
		SubscriptionType: SubscriptionFree, // Will be updated when quota is fetched
		Quota: QuotaInfo{
			Main:   QuotaDetail{Used: 0, Total: 0},
			Trial:  QuotaDetail{Used: 0, Total: 0},
			Reward: QuotaDetail{Used: 0, Total: 0},
		},
		Tags:      []string{},
		Notes:     "",
		IsActive:  false,
		LastUsed:  time.Now(),
		CreatedAt: time.Now(),
	}

	return account, nil
}
