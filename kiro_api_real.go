package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Kiro API 实际端点
const (
	KiroAuthAPIBase  = "https://prod.us-east-1.auth.desktop.kiro.dev"
	KiroUsageAPIBase = "https://codewhisperer.us-east-1.amazonaws.com"
	KiroProfileARN   = "arn:aws:codewhisperer:us-east-1:699475941385:profile/EHGA3GRVQMUK"
)

// KiroAPIClient Kiro API 客户端
type KiroAPIClient struct {
	httpClient *http.Client
}

// NewKiroAPIClient 创建新的 Kiro API 客户端
func NewKiroAPIClient() *KiroAPIClient {
	return &KiroAPIClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// KiroRefreshTokenResponse Token 刷新响应
type KiroRefreshTokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
	ProfileArn   string `json:"profileArn"`
}

// RefreshKiroToken 刷新 Kiro 访问令牌
func (c *KiroAPIClient) RefreshKiroToken(refreshToken string) (*KiroRefreshTokenResponse, error) {
	urlStr := fmt.Sprintf("%s/refreshToken", KiroAuthAPIBase)

	body := map[string]string{
		"refreshToken": refreshToken,
	}

	bodyBytes, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", urlStr, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("网络错误: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("RefreshToken 已过期或无效")
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("刷新失败 (状态码: %d)", resp.StatusCode)
	}

	var result KiroRefreshTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &result, nil
}

// KiroUsageLimitsResponse 配额响应
type KiroUsageLimitsResponse struct {
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

// GetKiroUsageLimits 获取 Kiro 配额信息
func (c *KiroAPIClient) GetKiroUsageLimits(accessToken string) (*KiroUsageLimitsResponse, error) {
	urlStr := fmt.Sprintf("%s/getUsageLimits?isEmailRequired=true&origin=AI_EDITOR&profileArn=%s",
		KiroUsageAPIBase,
		url.QueryEscape(KiroProfileARN))

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("网络错误: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// 尝试解析错误响应
		var errorResp struct {
			Message string `json:"message"`
			Reason  string `json:"reason"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil && errorResp.Message != "" {
			if errorResp.Reason == "TEMPORARILY_SUSPENDED" {
				return nil, fmt.Errorf("账号已被临时封禁：%s", errorResp.Message)
			}
			return nil, fmt.Errorf("获取配额失败：%s", errorResp.Message)
		}
		return nil, fmt.Errorf("获取配额失败 (状态码: %d)", resp.StatusCode)
	}

	var result KiroUsageLimitsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &result, nil
}

// ConvertKiroResponseToAccount 转换 Kiro API 响应为账号对象
func ConvertKiroResponseToAccount(tokenResp *KiroRefreshTokenResponse, usageResp *KiroUsageLimitsResponse, accountMgr *AccountManager) *KiroAccount {
	// 转换配额信息
	quota := QuotaInfo{
		Main:   QuotaDetail{Used: 0, Total: 0},
		Trial:  QuotaDetail{Used: 0, Total: 0},
		Reward: QuotaDetail{Used: 0, Total: 0},
	}

	if len(usageResp.UsageBreakdownList) > 0 {
		breakdown := usageResp.UsageBreakdownList[0]
		quota.Main.Used = breakdown.CurrentUsage
		quota.Main.Total = breakdown.UsageLimit
		
		fmt.Printf("  主配额: Used=%d, Total=%d\n", breakdown.CurrentUsage, breakdown.UsageLimit)

		if breakdown.FreeTrialInfo != nil {
			quota.Trial.Used = breakdown.FreeTrialInfo.CurrentUsage
			quota.Trial.Total = breakdown.FreeTrialInfo.UsageLimit
			fmt.Printf("  试用配额: Used=%d, Total=%d\n", breakdown.FreeTrialInfo.CurrentUsage, breakdown.FreeTrialInfo.UsageLimit)
		}

		for i, bonus := range breakdown.Bonuses {
			quota.Reward.Used += int(bonus.CurrentUsage)
			quota.Reward.Total += int(bonus.UsageLimit)
			fmt.Printf("  赠送配额[%d]: Used=%.0f, Total=%.0f\n", i, bonus.CurrentUsage, bonus.UsageLimit)
		}
		
		fmt.Printf("  累计配额: Used=%d, Total=%d\n", 
			quota.Main.Used+quota.Trial.Used+quota.Reward.Used,
			quota.Main.Total+quota.Trial.Total+quota.Reward.Total)
	}

	// 获取用户信息
	email := "user@example.com"
	userID := ""
	if usageResp.UserInfo != nil {
		email = usageResp.UserInfo.Email
		userID = usageResp.UserInfo.UserID
	}

	// 获取订阅类型
	subscriptionType := SubscriptionFree
	if usageResp.SubscriptionInfo != nil {
		switch usageResp.SubscriptionInfo.Type {
		case "PRO":
			subscriptionType = SubscriptionPro
		case "PRO_PLUS":
			subscriptionType = SubscriptionProPlus
		}
	}

	// 计算过期时间
	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return &KiroAccount{
		ID:               accountMgr.generateAccountID(),
		Email:            email,
		DisplayName:      email,
		Avatar:           "",
		BearerToken:      tokenResp.AccessToken,
		RefreshToken:     tokenResp.RefreshToken,
		TokenExpiry:      expiresAt,
		LoginMethod:      LoginMethodToken,
		Provider:         "",
		SubscriptionType: subscriptionType,
		Quota:            quota,
		Tags:             []string{},
		Notes:            fmt.Sprintf("User ID: %s", userID),
		IsActive:         false,
		LastUsed:         time.Now(),
		CreatedAt:        time.Now(),
		UserID:           userID,
		ProfileArn:       tokenResp.ProfileArn,
	}
}
