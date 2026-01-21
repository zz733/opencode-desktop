package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// AuthService handles authentication operations for Kiro accounts
type AuthService struct {
	httpClient    *http.Client
	config        *KiroAPIConfig
	kiroClient    KiroDesktopAPI
	pendingLogins map[string]*PendingLogin // state -> PendingLogin
	onAuthSuccess func(*KiroAccount) error // Callback when authentication succeeds
}

type KiroDesktopAPI interface {
	InitiateLogin(provider, redirectUri, codeChallenge, state string) string
	ExchangeToken(code, codeVerifier, redirectUri string) (*DesktopExchangeTokenResponse, error)
	RefreshToken(refreshToken string) (*DesktopExchangeTokenResponse, error)
	GetUserInfo(accessToken string) (*UsageLimitsResponse, error)
}

type PendingLogin struct {
	Idp           string
	CodeVerifier  string
	State         string
	Provider      OAuthProvider
	RedirectUri   string
	UseDesktopAPI bool // Flag to use Desktop API for token exchange
}

// NewAuthService creates a new AuthService instance
func NewAuthService() *AuthService {
	config := DefaultKiroAPIConfig()

	as := &AuthService{
		httpClient: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
		config:        config,
		kiroClient:    NewKiroDesktopClient(""),
		pendingLogins: make(map[string]*PendingLogin),
	}

	// Start the local callback server
	go as.StartCallbackServer()

	return as
}

// SetAuthSuccessCallback sets the callback function to be called when authentication succeeds
func (as *AuthService) SetAuthSuccessCallback(callback func(*KiroAccount) error) {
	as.onAuthSuccess = callback
}

// StartCallbackServer starts the local HTTP server for OAuth callbacks
func (as *AuthService) StartCallbackServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/callback", as.handleCallbackHTTP)

	server := &http.Server{
		Addr:    "127.0.0.1:54321",
		Handler: mux,
	}

	fmt.Println("Starting OAuth callback server on 127.0.0.1:54321 (CORS enabled)")
	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("OAuth callback server failed: %v\n", err)
	}
}

// handleCallbackHTTP handles the HTTP request for the OAuth callback
func (as *AuthService) handleCallbackHTTP(w http.ResponseWriter, r *http.Request) {
	// 添加 CORS 支持
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "*")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	query := r.URL.Query()

	// 支持通过 fullUrl 参数传递完整的跳转地址
	fullUrl := query.Get("fullUrl")
	if fullUrl != "" {
		parsedUrl, err := url.Parse(fullUrl)
		if err == nil {
			query = parsedUrl.Query()
		}
	}

	// Check for errors in the query params
	if errStr := query.Get("error"); errStr != "" {
		as.writeAuthResponse(w, false, fmt.Sprintf("Authentication failed: %s", errStr))
		return
	}

	state := query.Get("state")
	code := query.Get("code")

	if state == "" || code == "" {
		as.writeAuthResponse(w, false, "Missing state or code parameter")
		return
	}

	account, err := as.HandleOAuthCallback(state, code)
	if err != nil {
		as.writeAuthResponse(w, false, fmt.Sprintf("Authentication failed: %v", err))
		return
	}

	// Notify listener
	if as.onAuthSuccess != nil {
		if err := as.onAuthSuccess(account); err != nil {
			as.writeAuthResponse(w, false, fmt.Sprintf("Failed to save account: %v", err))
			return
		}
	}

	as.writeAuthResponse(w, true, "Authentication successful! You can close this window.")
}

// writeAuthResponse writes an HTML response to the callback request
func (as *AuthService) writeAuthResponse(w http.ResponseWriter, success bool, message string) {
	status := "success"
	if !success {
		status = "error"
	}

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Kiro Authentication</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background-color: #f5f5f7; }
        .card { background: white; padding: 2rem; border-radius: 12px; box-shadow: 0 4px 6px rgba(0,0,0,0.1); text-align: center; max-width: 400px; width: 100%%; }
        .success { color: #28a745; }
        .error { color: #dc3545; }
        h1 { margin-bottom: 1rem; }
        p { color: #666; margin-bottom: 1.5rem; }
        button { background-color: #007bff; color: white; border: none; padding: 0.5rem 1rem; border-radius: 6px; cursor: pointer; font-size: 1rem; }
        button:hover { background-color: #0056b3; }
    </style>
</head>
<body>
    <div class="card">
        <h1 class="%s">%s</h1>
        <p>%s</p>
        <button onclick="window.close()">Close Window</button>
    </div>
    <script>
        // Notify the main application
        if (window.opener) {
            window.opener.postMessage({
                type: 'oauth-complete',
                success: %v,
                message: '%s'
            }, '*');
        }
        // Auto-close after 3 seconds if successful
        if (%v) {
            setTimeout(() => window.close(), 3000);
        }
    </script>
</body>
</html>
	`, status, status, message, success, message, success)

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
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
		config:        config,
		kiroClient:    NewKiroDesktopClient(""),
		pendingLogins: make(map[string]*PendingLogin),
	}

	return as
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

	result, err := as.kiroClient.RefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	return &TokenInfo{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    expiresAt,
		TokenType:    "Bearer",
		ProfileArn:   result.ProfileArn,
	}, nil
}

// GetUserProfile retrieves user profile information using a bearer token
// Kiro 的用户信息包含在 getUsageLimits API 响应中
func (as *AuthService) GetUserProfile(token string) (*UserProfile, error) {
	if token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	usageLimits, err := as.kiroClient.GetUserInfo(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	if usageLimits.UserInfo == nil {
		return nil, fmt.Errorf("user info not found in response")
	}

	return &UserProfile{
		ID:       usageLimits.UserInfo.UserID,
		Email:    usageLimits.UserInfo.Email,
		Name:     usageLimits.UserInfo.Email,
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

// KiroWebOAuthRedirectURI is the official redirect URI for Kiro Web OAuth
const KiroWebOAuthRedirectURI = "https://app.kiro.dev/signin/oauth"

// StartOAuthFlow starts an OAuth authentication flow using Web OAuth API
func (as *AuthService) StartOAuthFlow(provider OAuthProvider) (string, error) {
	// Determine IDP string
	var idp string
	switch provider {
	case ProviderGoogle:
		idp = "Google"
	case ProviderGitHub:
		idp = "Github"
	case ProviderBuilderID:
		idp = "BuilderID"
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}

	// Generate state
	state, err := as.generateStateToken()
	if err != nil {
		return "", err
	}

	// Generate PKCE
	codeVerifier, err := GenerateCodeVerifier()
	if err != nil {
		return "", err
	}
	codeChallenge := GenerateCodeChallenge(codeVerifier)

	// Use official redirect URI
	redirectUri := KiroWebOAuthRedirectURI

	// Use Web OAuth API (CBOR)
	webClient := NewKiroWebPortalClient()
	initiateResp, err := webClient.InitiateLogin(idp, redirectUri, codeChallenge, state)
	if err != nil {
		return "", fmt.Errorf("failed to initiate login: %w", err)
	}

	if initiateResp.RedirectUrl == "" {
		return "", fmt.Errorf("no redirect URL in InitiateLogin response")
	}

	// Store pending login
	as.pendingLogins[state] = &PendingLogin{
		Idp:           idp,
		CodeVerifier:  codeVerifier,
		State:         state,
		Provider:      provider,
		RedirectUri:   redirectUri,
		UseDesktopAPI: false, // Use Web OAuth API
	}

	return initiateResp.RedirectUrl, nil
}

// HandleOAuthCallback handles the OAuth callback and creates an account
// It uses Desktop Auth API for token exchange when UseDesktopAPI flag is set
func (as *AuthService) HandleOAuthCallback(state, code string) (*KiroAccount, error) {
	if code == "" {
		return nil, fmt.Errorf("authorization code is required")
	}

	// Retrieve pending login
	pending, ok := as.pendingLogins[state]
	if !ok {
		return nil, fmt.Errorf("invalid or expired state: %s", state)
	}
	defer delete(as.pendingLogins, state)

	fmt.Printf("[OAuth] Handling callback for state: %s..., code: %s..., useDesktopAPI: %v\n", state[:20], code[:min(len(code), 20)], pending.UseDesktopAPI)

	var accessToken, refreshToken string
	var expiresIn int64

	if pending.UseDesktopAPI {
		// Use Desktop Auth API for token exchange
		desktopClient := NewKiroDesktopClient("")
		tokenResult, err := desktopClient.ExchangeToken(code, pending.CodeVerifier, pending.RedirectUri)
		if err != nil {
			fmt.Printf("[OAuth] Desktop token exchange failed: %v\n", err)
			return nil, fmt.Errorf("failed to exchange token: %w", err)
		}

		accessToken = tokenResult.AccessToken
		refreshToken = tokenResult.RefreshToken
		expiresIn = tokenResult.ExpiresIn

		fmt.Printf("[OAuth] Desktop token exchange successful, accessToken length: %d\n", len(accessToken))
	} else {
		// Fall back to Web OAuth (CBOR API)
		webClient := NewKiroWebPortalClient()
		tokenResult, err := webClient.ExchangeToken(pending.Idp, code, pending.CodeVerifier, pending.RedirectUri, state)
		if err != nil {
			return nil, fmt.Errorf("failed to exchange token: %w", err)
		}

		accessToken = tokenResult.AccessToken
		refreshToken = tokenResult.SessionToken
		expiresIn = tokenResult.ExpiresIn
	}

	fmt.Printf("[OAuth] Token exchange successful, getting user info...\n")

	// Get User Info using Desktop API (works for both Desktop and Web tokens)
	desktopClient := NewKiroDesktopClient("")
	usageLimits, err := desktopClient.GetUserInfo(accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	if usageLimits.UserInfo == nil {
		return nil, fmt.Errorf("user info not found in response")
	}

	fmt.Printf("[OAuth] User info retrieved: %s\n", usageLimits.UserInfo.Email)

	// Map subscription info
	subscriptionType := SubscriptionFree
	if usageLimits.SubscriptionInfo != nil {
		switch usageLimits.SubscriptionInfo.Type {
		case "pro":
			subscriptionType = SubscriptionPro
		case "pro_plus":
			subscriptionType = SubscriptionProPlus
		}
	}

	// Map quota info
	quota := QuotaInfo{
		Main:   QuotaDetail{Used: 0, Total: 0},
		Trial:  QuotaDetail{Used: 0, Total: 0},
		Reward: QuotaDetail{Used: 0, Total: 0},
	}

	for _, breakdown := range usageLimits.UsageBreakdownList {
		if breakdown.ResourceType == "chat" {
			quota.Main.Used = breakdown.CurrentUsage
			quota.Main.Total = breakdown.UsageLimit

			if breakdown.FreeTrialInfo != nil {
				quota.Trial.Used = breakdown.FreeTrialInfo.CurrentUsage
				quota.Trial.Total = breakdown.FreeTrialInfo.UsageLimit
			}

			for _, bonus := range breakdown.Bonuses {
				quota.Reward.Used += int(bonus.CurrentUsage)
				quota.Reward.Total += int(bonus.UsageLimit)
			}
			break
		}
	}

	// Create account
	account := &KiroAccount{
		Email:            usageLimits.UserInfo.Email,
		DisplayName:      usageLimits.UserInfo.Email,
		Avatar:           "",
		BearerToken:      accessToken,
		RefreshToken:     refreshToken,
		TokenExpiry:      time.Now().Add(time.Duration(expiresIn) * time.Second),
		LoginMethod:      LoginMethodOAuth,
		Provider:         pending.Provider,
		SubscriptionType: subscriptionType,
		Quota:            quota,
		Tags:             []string{},
		Notes:            "",
		IsActive:         false,
		LastUsed:         time.Now(),
		CreatedAt:        time.Now(),
	}

	fmt.Printf("[OAuth] Account created successfully for: %s\n", account.Email)

	return account, nil
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
