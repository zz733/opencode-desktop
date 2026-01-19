package main

import (
	"fmt"
	"os"
	"strings"
)

// KiroAPIConfig holds configuration for Kiro API endpoints
type KiroAPIConfig struct {
	BaseURL          string
	AuthValidateURL  string
	AuthRefreshURL   string
	UserProfileURL   string
	UserQuotaURL     string
	OAuthExchangeURL string
	AuthLoginURL     string
	ProfileARN       string
	Timeout          int // seconds
}

// DefaultKiroAPIConfig returns the default Kiro API configuration
//
// 环境变量配置示例：
//
//	export KIRO_AUTH_BASE_URL="https://prod.us-east-1.auth.desktop.kiro.dev"
//	export KIRO_USAGE_BASE_URL="https://codewhisperer.us-east-1.amazonaws.com"
//	export KIRO_AUTH_REFRESH_URL="https://prod.us-east-1.auth.desktop.kiro.dev/refreshToken"
//	export KIRO_USER_PROFILE_URL="https://codewhisperer.us-east-1.amazonaws.com/getUsageLimits"
//	export KIRO_USER_QUOTA_URL="https://codewhisperer.us-east-1.amazonaws.com/getUsageLimits"
//	export KIRO_PROFILE_ARN="arn:aws:codewhisperer:us-east-1:699475941385:profile/EHGA3GRVQMUK"
//
// 或者在 .env 文件中配置（需要在项目根目录创建 .env 文件）
func DefaultKiroAPIConfig() *KiroAPIConfig {
	authBaseURL := getEnvWithDefault("KIRO_AUTH_BASE_URL", "https://prod.us-east-1.auth.desktop.kiro.dev")
	usageBaseURL := getEnvWithDefault("KIRO_USAGE_BASE_URL", "https://codewhisperer.us-east-1.amazonaws.com")
	baseURL := getEnvWithDefault("KIRO_API_BASE_URL", authBaseURL)

	return &KiroAPIConfig{
		BaseURL:          baseURL,
		AuthValidateURL:  getEnvWithDefault("KIRO_AUTH_VALIDATE_URL", authBaseURL+"/validateToken"),
		AuthRefreshURL:   getEnvWithDefault("KIRO_AUTH_REFRESH_URL", authBaseURL+"/refreshToken"),
		UserProfileURL:   getEnvWithDefault("KIRO_USER_PROFILE_URL", usageBaseURL+"/getUsageLimits"),
		UserQuotaURL:     getEnvWithDefault("KIRO_USER_QUOTA_URL", usageBaseURL+"/getUsageLimits"),
		OAuthExchangeURL: getEnvWithDefault("KIRO_OAUTH_EXCHANGE_URL", authBaseURL+"/v1/oauth/exchange"),
		AuthLoginURL:     getEnvWithDefault("KIRO_AUTH_LOGIN_URL", authBaseURL+"/v1/auth/login"),
		ProfileARN:       getEnvWithDefault("KIRO_PROFILE_ARN", "arn:aws:codewhisperer:us-east-1:699475941385:profile/EHGA3GRVQMUK"),
		Timeout:          getEnvIntWithDefault("KIRO_API_TIMEOUT", 5),
	}
}

// LoadKiroAPIConfigFromEnv loads Kiro API configuration from environment variables
func LoadKiroAPIConfigFromEnv() *KiroAPIConfig {
	return DefaultKiroAPIConfig()
}

// Validate checks if the configuration is valid
func (c *KiroAPIConfig) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("base URL cannot be empty")
	}

	if !strings.HasPrefix(c.BaseURL, "http://") && !strings.HasPrefix(c.BaseURL, "https://") {
		return fmt.Errorf("base URL must start with http:// or https://")
	}

	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	return nil
}

// getEnvWithDefault gets an environment variable or returns a default value
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvIntWithDefault gets an integer environment variable or returns a default value
func getEnvIntWithDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		if _, err := fmt.Sscanf(value, "%d", &intValue); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// KiroAPIResponse represents a standard API response structure
type KiroAPIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   *APIError   `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// APIError represents an error response from the API
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Error implements the error interface
func (e *APIError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
