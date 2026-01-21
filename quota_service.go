package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// QuotaService handles quota-related operations for Kiro accounts
type QuotaService struct {
	config      *KiroAPIConfig
	cache       map[string]*QuotaCacheEntry
	cacheTTL    time.Duration
	mutex       sync.RWMutex
	usageClient UsageLimitsFetcher
}

type UsageLimitsFetcher interface {
	GetUserInfo(accessToken string) (*UsageLimitsResponse, error)
}

var kiroMachineIDOnce sync.Once
var kiroMachineID string

func getKiroMachineID() string {
	kiroMachineIDOnce.Do(func() {
		b := make([]byte, 16)
		_, _ = rand.Read(b)
		kiroMachineID = hex.EncodeToString(b)
	})
	return kiroMachineID
}

// QuotaCacheEntry represents a cached quota entry
type QuotaCacheEntry struct {
	Quota     *QuotaInfo
	Timestamp time.Time
}

// NewQuotaService creates a new QuotaService instance
func NewQuotaService() *QuotaService {
	config := DefaultKiroAPIConfig()

	return &QuotaService{
		config:      config,
		cache:       make(map[string]*QuotaCacheEntry),
		cacheTTL:    5 * time.Minute,
		mutex:       sync.RWMutex{},
		usageClient: NewKiroDesktopClient(getKiroMachineID()),
	}
}

// NewQuotaServiceWithConfig creates a new QuotaService instance with custom configuration
func NewQuotaServiceWithConfig(config *KiroAPIConfig) *QuotaService {
	if err := config.Validate(); err != nil {
		// Fall back to default config if validation fails
		config = DefaultKiroAPIConfig()
	}

	return &QuotaService{
		config:      config,
		cache:       make(map[string]*QuotaCacheEntry),
		cacheTTL:    5 * time.Minute,
		mutex:       sync.RWMutex{},
		usageClient: NewKiroDesktopClient(getKiroMachineID()),
	}
}

// GetQuota retrieves quota information for an account
func (qs *QuotaService) GetQuota(token string) (*QuotaInfo, error) {
	if token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	// Check cache first
	qs.mutex.RLock()
	if entry, exists := qs.cache[token]; exists {
		if time.Since(entry.Timestamp) < qs.cacheTTL {
			qs.mutex.RUnlock()
			// Return a copy to prevent external modification
			quotaCopy := *entry.Quota
			return &quotaCopy, nil
		}
	}
	qs.mutex.RUnlock()

	// Fetch from API
	quota, err := qs.fetchQuotaFromAPI(token)
	if err != nil {
		return nil, err
	}

	// Update cache
	qs.mutex.Lock()
	qs.cache[token] = &QuotaCacheEntry{
		Quota:     quota,
		Timestamp: time.Now(),
	}
	qs.mutex.Unlock()

	// Return a copy
	quotaCopy := *quota
	return &quotaCopy, nil
}

// fetchQuotaFromAPI fetches quota information from the Kiro API
func (qs *QuotaService) fetchQuotaFromAPI(token string) (*QuotaInfo, error) {
	fmt.Printf("→ fetchQuotaFromAPI 开始 (token长度: %d)\n", len(token))

	fmt.Println("  调用 GetUsageLimits...")
	if qs.usageClient == nil {
		qs.usageClient = NewKiroDesktopClient(getKiroMachineID())
	}
	usageResp, err := qs.usageClient.GetUserInfo(token)
	if err != nil {
		fmt.Printf("✗ 获取配额失败: %v\n", err)
		return nil, fmt.Errorf("获取配额失败: %w", err)
	}

	fmt.Println("  ✓ 配额信息获取成功")

	// 转换为 QuotaInfo
	quota := QuotaInfo{
		Main:   QuotaDetail{Used: 0, Total: 0},
		Trial:  QuotaDetail{Used: 0, Total: 0},
		Reward: QuotaDetail{Used: 0, Total: 0},
	}

	var breakdown *struct {
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
	}

	for i := range usageResp.UsageBreakdownList {
		if usageResp.UsageBreakdownList[i].ResourceType == "chat" {
			breakdown = &usageResp.UsageBreakdownList[i]
			break
		}
	}
	if breakdown == nil && len(usageResp.UsageBreakdownList) > 0 {
		breakdown = &usageResp.UsageBreakdownList[0]
	}

	if breakdown != nil {
		quota.Main.Used = breakdown.CurrentUsage
		quota.Main.Total = breakdown.UsageLimit

		fmt.Printf("  【API 返回】主配额: Used=%d, Total=%d\n", breakdown.CurrentUsage, breakdown.UsageLimit)

		if breakdown.FreeTrialInfo != nil {
			quota.Trial.Used = breakdown.FreeTrialInfo.CurrentUsage
			quota.Trial.Total = breakdown.FreeTrialInfo.UsageLimit
			fmt.Printf("  【API 返回】试用配额: Used=%d, Total=%d\n", breakdown.FreeTrialInfo.CurrentUsage, breakdown.FreeTrialInfo.UsageLimit)
		}

		for i, bonus := range breakdown.Bonuses {
			quota.Reward.Used += int(bonus.CurrentUsage)
			quota.Reward.Total += int(bonus.UsageLimit)
			fmt.Printf("  【API 返回】赠送配额[%d]: Used=%.0f, Total=%.0f\n", i, bonus.CurrentUsage, bonus.UsageLimit)
		}

		fmt.Printf("  【计算结果】累计配额: Used=%d, Total=%d\n",
			quota.Main.Used+quota.Trial.Used+quota.Reward.Used,
			quota.Main.Total+quota.Trial.Total+quota.Reward.Total)
	} else {
		fmt.Println("  ⚠ 警告: API 返回的 UsageBreakdownList 为空")
	}

	fmt.Println("✓ fetchQuotaFromAPI 完成")
	return &quota, nil
}

// RefreshQuota refreshes quota information for a specific account
func (qs *QuotaService) RefreshQuota(accountID string, token string) error {
	fmt.Printf("=== RefreshQuota 开始 (accountID=%s) ===\n", accountID)
	if token == "" {
		fmt.Println("✗ token 为空")
		return fmt.Errorf("token cannot be empty")
	}
	fmt.Printf("  Token 长度: %d\n", len(token))

	// Fetch fresh quota from API
	fmt.Println("  调用 fetchQuotaFromAPI...")
	quota, err := qs.fetchQuotaFromAPI(token)
	if err != nil {
		fmt.Printf("✗ 获取配额失败: %v\n", err)
		return fmt.Errorf("failed to fetch quota: %w", err)
	}
	fmt.Println("  ✓ 配额获取成功")

	// Update cache
	qs.mutex.Lock()
	qs.cache[token] = &QuotaCacheEntry{
		Quota:     quota,
		Timestamp: time.Now(),
	}
	qs.mutex.Unlock()
	fmt.Println("  ✓ 缓存已更新")

	fmt.Println("=== RefreshQuota 完成 ===")
	return nil
}

// BatchRefreshQuota refreshes quota for multiple accounts
func (qs *QuotaService) BatchRefreshQuota(accounts []*KiroAccount) error {
	var errors []string
	successCount := 0

	for _, account := range accounts {
		if err := qs.RefreshQuota(account.ID, account.BearerToken); err != nil {
			errors = append(errors, fmt.Sprintf("failed to refresh quota for %s: %v", account.ID, err))
			continue
		}

		// Update account quota
		quota, err := qs.GetQuota(account.BearerToken)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to get updated quota for %s: %v", account.ID, err))
			continue
		}

		account.Quota = *quota
		successCount++
	}

	if len(errors) > 0 {
		return fmt.Errorf("batch quota refresh completed with errors: %v", errors)
	}

	return nil
}

// ClearCache clears the quota cache
func (qs *QuotaService) ClearCache() {
	qs.mutex.Lock()
	defer qs.mutex.Unlock()
	qs.cache = make(map[string]*QuotaCacheEntry)
}

// ClearExpiredCache removes expired entries from the cache
func (qs *QuotaService) ClearExpiredCache() {
	qs.mutex.Lock()
	defer qs.mutex.Unlock()

	now := time.Now()
	for token, entry := range qs.cache {
		if now.Sub(entry.Timestamp) >= qs.cacheTTL {
			delete(qs.cache, token)
		}
	}
}

// GetCacheStats returns statistics about the quota cache
func (qs *QuotaService) GetCacheStats() map[string]interface{} {
	qs.mutex.RLock()
	defer qs.mutex.RUnlock()

	stats := map[string]interface{}{
		"totalEntries":   len(qs.cache),
		"expiredEntries": 0,
		"cacheTTL":       qs.cacheTTL.String(),
	}

	now := time.Now()
	for _, entry := range qs.cache {
		if now.Sub(entry.Timestamp) >= qs.cacheTTL {
			stats["expiredEntries"] = stats["expiredEntries"].(int) + 1
		}
	}

	return stats
}

// QuotaMonitor provides quota monitoring functionality
type QuotaMonitor struct {
	service   *QuotaService
	accounts  func() []*KiroAccount // Function to get current accounts
	interval  time.Duration
	threshold float64
	stopChan  chan bool
	running   bool
	mutex     sync.RWMutex
}

// NewQuotaMonitor creates a new QuotaMonitor instance
func NewQuotaMonitor(service *QuotaService, accountsFunc func() []*KiroAccount) *QuotaMonitor {
	return &QuotaMonitor{
		service:   service,
		accounts:  accountsFunc,
		interval:  5 * time.Minute, // Check every 5 minutes
		threshold: 0.9,             // 90% threshold
		stopChan:  make(chan bool),
		running:   false,
		mutex:     sync.RWMutex{},
	}
}

// Start starts the quota monitor
func (qm *QuotaMonitor) Start() {
	qm.mutex.Lock()
	defer qm.mutex.Unlock()

	if qm.running {
		return
	}

	qm.running = true
	go qm.monitorLoop()
}

// Stop stops the quota monitor
func (qm *QuotaMonitor) Stop() {
	qm.mutex.Lock()
	defer qm.mutex.Unlock()

	if !qm.running {
		return
	}

	qm.running = false
	qm.stopChan <- true
}

// SetInterval sets the monitoring interval
func (qm *QuotaMonitor) SetInterval(interval time.Duration) {
	qm.mutex.Lock()
	defer qm.mutex.Unlock()
	qm.interval = interval
}

// SetThreshold sets the quota alert threshold
func (qm *QuotaMonitor) SetThreshold(threshold float64) {
	qm.mutex.Lock()
	defer qm.mutex.Unlock()
	qm.threshold = threshold
}

// monitorLoop runs the monitoring loop
func (qm *QuotaMonitor) monitorLoop() {
	ticker := time.NewTicker(qm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			qm.checkQuotas()
		case <-qm.stopChan:
			return
		}
	}
}

// checkQuotas checks quota for all accounts and generates alerts
func (qm *QuotaMonitor) checkQuotas() {
	accounts := qm.accounts()
	if len(accounts) == 0 {
		return
	}

	var alerts []QuotaAlert

	for _, account := range accounts {
		// Skip if token is expired
		if account.IsTokenExpired() {
			continue
		}

		// Refresh quota
		if err := qm.service.RefreshQuota(account.ID, account.BearerToken); err != nil {
			continue
		}

		// Get updated quota
		quota, err := qm.service.GetQuota(account.BearerToken)
		if err != nil {
			continue
		}

		// Update account quota
		account.Quota = *quota

		// Check for alerts
		accountAlerts := account.GetQuotaAlerts(qm.threshold)
		alerts = append(alerts, accountAlerts...)
	}

	// TODO: Emit quota alerts event
	// This would be handled by the AccountManager
}

// CheckQuotaAlerts manually checks for quota alerts
func (qm *QuotaMonitor) CheckQuotaAlerts() []QuotaAlert {
	accounts := qm.accounts()
	if len(accounts) == 0 {
		return nil
	}

	var alerts []QuotaAlert

	qm.mutex.RLock()
	threshold := qm.threshold
	qm.mutex.RUnlock()

	for _, account := range accounts {
		accountAlerts := account.GetQuotaAlerts(threshold)
		alerts = append(alerts, accountAlerts...)
	}

	return alerts
}

// IsRunning returns whether the monitor is currently running
func (qm *QuotaMonitor) IsRunning() bool {
	qm.mutex.RLock()
	defer qm.mutex.RUnlock()
	return qm.running
}
