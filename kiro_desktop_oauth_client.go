package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

const (
	DesktopAuthEndpoint  = "https://prod.us-east-1.auth.desktop.kiro.dev"
	DesktopUsageEndpoint = "https://codewhisperer.us-east-1.amazonaws.com"
)

// UsageLimitsResponse represents the response from GetUsageLimits API
type UsageLimitsResponse struct {
	UserInfo *struct {
		Email  string `json:"email"`
		UserID string `json:"userId"`
	} `json:"userInfo"`
	SubscriptionInfo *struct {
		SubscriptionTitle string `json:"subscriptionTitle"`
		Type              string `json:"type"`
	} `json:"subscriptionInfo"`
	UsageBreakdownList []struct {
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
	} `json:"usageBreakdownList"`
}

// KiroDesktopClient handles communication with Kiro Desktop Auth API
type KiroDesktopClient struct {
	client    *http.Client
	machineID string
}

func NewKiroDesktopClient(machineID string) *KiroDesktopClient {
	return &KiroDesktopClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		machineID: machineID,
	}
}

func (c *KiroDesktopClient) getUserAgent() string {
	if c.machineID == "" {
		return "KiroBatchLoginCLI/1.0.0"
	}
	return fmt.Sprintf("KiroIDE-0.6.18-%s", c.machineID)
}

// InitiateLogin returns the login URL for the desktop flow
func (c *KiroDesktopClient) InitiateLogin(provider, redirectUri, codeChallenge, state string) string {
	baseURL := fmt.Sprintf("%s/login", DesktopAuthEndpoint)
	params := url.Values{}
	params.Set("idp", provider)
	params.Set("redirect_uri", redirectUri)
	params.Set("code_challenge", codeChallenge)
	params.Set("code_challenge_method", "S256")
	params.Set("state", state)

	return fmt.Sprintf("%s?%s", baseURL, params.Encode())
}

type DesktopExchangeTokenRequest struct {
	Code         string `json:"code"`
	CodeVerifier string `json:"code_verifier"`
	RedirectUri  string `json:"redirect_uri"`
}

type DesktopExchangeTokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
	ProfileArn   string `json:"profileArn"`
	CsrfToken    string `json:"csrfToken,omitempty"`
}

// ExchangeToken exchanges authorization code for tokens
func (c *KiroDesktopClient) ExchangeToken(code, codeVerifier, redirectUri string) (*DesktopExchangeTokenResponse, error) {
	url := fmt.Sprintf("%s/oauth/token", DesktopAuthEndpoint)

	reqBody := DesktopExchangeTokenRequest{
		Code:         code,
		CodeVerifier: codeVerifier,
		RedirectUri:  redirectUri,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.getUserAgent())

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("exchange token failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result DesktopExchangeTokenResponse
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	return &result, nil
}

type DesktopRefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// RefreshToken refreshes the access token
func (c *KiroDesktopClient) RefreshToken(refreshToken string) (*DesktopExchangeTokenResponse, error) {
	// Corresponds to kiro_auth_client.rs: refresh_token
	url := fmt.Sprintf("%s/refreshToken", DesktopAuthEndpoint)

	data, err := json.Marshal(map[string]string{
		"refreshToken": refreshToken,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	// Use the same UA as in kiro_auth_client.rs
	req.Header.Set("User-Agent", "KiroBatchLoginCLI/1.0.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh token failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result DesktopExchangeTokenResponse
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	return &result, nil
}

// GetUserInfo retrieves user info and usage limits
// Note: Desktop flow gets user info from usage limits API
func (c *KiroDesktopClient) GetUserInfo(accessToken string) (*UsageLimitsResponse, error) {
	// Re-use the existing model from kiro_api_client.go (UsageLimitsResponse)
	// assuming it matches the DesktopUsageResponse structure (it seems to match based on auth.rs)

	// Update to match auth.rs: getUsageLimits with profileArn
	// Hardcoded profileArn from auth.rs
	profileArn := "arn:aws:codewhisperer:us-east-1:699475941385:profile/EHGA3GRVQMUK"
	url := fmt.Sprintf("%s/getUsageLimits?isEmailRequired=true&origin=AI_EDITOR&profileArn=%s",
		DesktopUsageEndpoint,
		url.QueryEscape(profileArn))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// auth.rs uses simple headers
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Accept", "application/json")
	// Note: auth.rs does not explicitly set User-Agent for this call, but we can keep it or remove it.
	// Since we are emulating the desktop app, let's stick to what auth.rs does (minimal headers).
	// But keeping a UA is generally good practice.
	// req.Header.Set("User-Agent", c.getUserAgent())

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get usage limits failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result UsageLimitsResponse
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	return &result, nil
}
