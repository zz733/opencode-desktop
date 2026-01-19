package main

import (
	"time"
)

// LoginMethod represents the method used to login to a Kiro account
type LoginMethod string

const (
	LoginMethodOAuth    LoginMethod = "oauth"
	LoginMethodToken    LoginMethod = "token"
	LoginMethodPassword LoginMethod = "password"
)

// OAuthProvider represents the OAuth provider used for authentication
type OAuthProvider string

const (
	ProviderGoogle    OAuthProvider = "google"
	ProviderGitHub    OAuthProvider = "github"
	ProviderBuilderID OAuthProvider = "builderid"
)

// SubscriptionType represents the subscription type of a Kiro account
type SubscriptionType string

const (
	SubscriptionFree    SubscriptionType = "free"
	SubscriptionPro     SubscriptionType = "pro"
	SubscriptionProPlus SubscriptionType = "pro_plus"
)

// QuotaDetail represents detailed quota information for a specific quota type
type QuotaDetail struct {
	Used  int `json:"used"`
	Total int `json:"total"`
}

// QuotaInfo represents the complete quota information for an account
type QuotaInfo struct {
	Main   QuotaDetail `json:"main"`   // 主配额
	Trial  QuotaDetail `json:"trial"`  // 试用配额
	Reward QuotaDetail `json:"reward"` // 奖励配额
}

// KiroAccount represents a Kiro account with all its associated information
type KiroAccount struct {
	ID               string           `json:"id"`                 // 唯一标识
	Email            string           `json:"email"`              // 邮箱
	DisplayName      string           `json:"displayName"`        // 显示名称
	Avatar           string           `json:"avatar,omitempty"`   // 头像 URL
	BearerToken      string           `json:"-"`                  // 认证 Token
	RefreshToken     string           `json:"-"`                  // 刷新 Token
	TokenExpiry      time.Time        `json:"tokenExpiry"`        // Token 过期时间
	LoginMethod      LoginMethod      `json:"loginMethod"`        // 登录方式
	Provider         OAuthProvider    `json:"provider,omitempty"` // OAuth 提供商
	SubscriptionType SubscriptionType `json:"subscriptionType"`   // 订阅类型
	Quota            QuotaInfo        `json:"quota"`              // 配额信息
	Tags             []string         `json:"tags"`               // 标签列表
	Notes            string           `json:"notes,omitempty"`    // 备注
	IsActive         bool             `json:"isActive"`           // 是否为当前激活账号
	LastUsed         time.Time        `json:"lastUsed"`           // 最后使用时间
	CreatedAt        time.Time        `json:"createdAt"`          // 添加时间
	// Machine ID binding
	MachineID   string `json:"machineId,omitempty"`   // 绑定的机器 ID
	SqmID       string `json:"sqmId,omitempty"`       // 绑定的 SQM ID
	DevDeviceID string `json:"devDeviceId,omitempty"` // 绑定的 Dev Device ID
	// Kiro specific fields
	UserID     string `json:"userId,omitempty"`     // Kiro 用户 ID
	ProfileArn string `json:"profileArn,omitempty"` // Kiro Profile ARN
}

// TokenInfo represents token information returned from authentication
type TokenInfo struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// UserProfile represents user profile information from Kiro API
type UserProfile struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Avatar   string `json:"avatar"`
	Provider string `json:"provider"`
}

// QuotaAlert represents a quota warning/alert
type QuotaAlert struct {
	AccountID   string  `json:"accountId"`
	AccountName string  `json:"accountName"`
	QuotaType   string  `json:"quotaType"`
	Usage       float64 `json:"usage"` // 使用百分比 (0.0-1.0)
	Message     string  `json:"message"`
}

// AccountSettings represents settings for account management
type AccountSettings struct {
	QuotaRefreshInterval   int           `json:"quotaRefreshInterval"`   // 配额刷新间隔（秒）
	AutoRefreshQuota       bool          `json:"autoRefreshQuota"`       // 自动刷新配额
	QuotaAlertThreshold    float64       `json:"quotaAlertThreshold"`    // 配额警告阈值 (0.0-1.0)
	ShowQuotaInStatusBar   bool          `json:"showQuotaInStatusBar"`   // 在状态栏显示配额
	DefaultLoginMethod     LoginMethod   `json:"defaultLoginMethod"`     // 默认登录方式
	PreferredOAuthProvider OAuthProvider `json:"preferredOAuthProvider"` // 首选 OAuth 提供商
	ExportEncryption       bool          `json:"exportEncryption"`       // 导出时加密
	AutoBackup             bool          `json:"autoBackup"`             // 自动备份
	BackupRetentionDays    int           `json:"backupRetentionDays"`    // 备份保留天数
	AutoChangeMachineID    bool          `json:"autoChangeMachineId"`    // 切换账号时自动修改机器码
}

// Tag represents a tag that can be applied to accounts
type Tag struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description,omitempty"`
}

// AccountData represents the complete account data structure for storage
type AccountData struct {
	Version         string          `json:"version"`
	Accounts        []*KiroAccount  `json:"accounts"`
	ActiveAccountID string          `json:"activeAccountId"`
	Settings        AccountSettings `json:"settings"`
	Tags            []Tag           `json:"tags"`
	LastUpdated     time.Time       `json:"lastUpdated"`
}

// GetUsagePercentage calculates the usage percentage for a quota detail
func (qd *QuotaDetail) GetUsagePercentage() float64 {
	if qd.Total == 0 {
		return 0.0
	}
	return float64(qd.Used) / float64(qd.Total)
}

// IsLowQuota checks if the quota is below the threshold
func (qd *QuotaDetail) IsLowQuota(threshold float64) bool {
	return qd.GetUsagePercentage() >= threshold
}

// GetTotalUsed returns the total used quota across all quota types
func (qi *QuotaInfo) GetTotalUsed() int {
	return qi.Main.Used + qi.Trial.Used + qi.Reward.Used
}

// GetTotalAvailable returns the total available quota across all quota types
func (qi *QuotaInfo) GetTotalAvailable() int {
	return qi.Main.Total + qi.Trial.Total + qi.Reward.Total
}

// GetOverallUsagePercentage calculates the overall usage percentage
func (qi *QuotaInfo) GetOverallUsagePercentage() float64 {
	total := qi.GetTotalAvailable()
	if total == 0 {
		return 0.0
	}
	return float64(qi.GetTotalUsed()) / float64(total)
}

// IsTokenExpired checks if the account's token has expired
func (ka *KiroAccount) IsTokenExpired() bool {
	return time.Now().After(ka.TokenExpiry)
}

// IsTokenExpiringSoon checks if the token will expire within the given duration
func (ka *KiroAccount) IsTokenExpiringSoon(duration time.Duration) bool {
	return time.Now().Add(duration).After(ka.TokenExpiry)
}

// HasTag checks if the account has a specific tag
func (ka *KiroAccount) HasTag(tag string) bool {
	for _, t := range ka.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// AddTag adds a tag to the account if it doesn't already exist
func (ka *KiroAccount) AddTag(tag string) {
	if !ka.HasTag(tag) {
		ka.Tags = append(ka.Tags, tag)
	}
}

// RemoveTag removes a tag from the account
func (ka *KiroAccount) RemoveTag(tag string) {
	for i, t := range ka.Tags {
		if t == tag {
			ka.Tags = append(ka.Tags[:i], ka.Tags[i+1:]...)
			break
		}
	}
}

// GetQuotaAlerts returns quota alerts for this account based on the threshold
func (ka *KiroAccount) GetQuotaAlerts(threshold float64) []QuotaAlert {
	var alerts []QuotaAlert

	// Check main quota
	if ka.Quota.Main.IsLowQuota(threshold) {
		alerts = append(alerts, QuotaAlert{
			AccountID:   ka.ID,
			AccountName: ka.DisplayName,
			QuotaType:   "main",
			Usage:       ka.Quota.Main.GetUsagePercentage(),
			Message:     "主配额使用率过高",
		})
	}

	// Check trial quota
	if ka.Quota.Trial.IsLowQuota(threshold) {
		alerts = append(alerts, QuotaAlert{
			AccountID:   ka.ID,
			AccountName: ka.DisplayName,
			QuotaType:   "trial",
			Usage:       ka.Quota.Trial.GetUsagePercentage(),
			Message:     "试用配额使用率过高",
		})
	}

	// Check reward quota
	if ka.Quota.Reward.IsLowQuota(threshold) {
		alerts = append(alerts, QuotaAlert{
			AccountID:   ka.ID,
			AccountName: ka.DisplayName,
			QuotaType:   "reward",
			Usage:       ka.Quota.Reward.GetUsagePercentage(),
			Message:     "奖励配额使用率过高",
		})
	}

	return alerts
}

// DefaultAccountSettings returns default settings for account management
func DefaultAccountSettings() AccountSettings {
	return AccountSettings{
		QuotaRefreshInterval:   300, // 5 minutes
		AutoRefreshQuota:       true,
		QuotaAlertThreshold:    0.9, // 90%
		ShowQuotaInStatusBar:   true,
		DefaultLoginMethod:     LoginMethodOAuth,
		PreferredOAuthProvider: ProviderGoogle,
		ExportEncryption:       true,
		AutoBackup:             true,
		BackupRetentionDays:    30,
	}
}
