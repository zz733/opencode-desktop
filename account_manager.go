package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// AccountManager manages multiple Kiro accounts with thread-safe operations
type AccountManager struct {
	accounts     map[string]*KiroAccount
	activeID     string
	settings     AccountSettings
	storage      *StorageService
	crypto       *CryptoService
	authService  *AuthService
	quotaService *QuotaService
	system       *KiroSystem
	mutex        sync.RWMutex
	tags         []Tag
	ctx          context.Context // Wails context for events
}

// NewAccountManager creates a new AccountManager instance
func NewAccountManager(storage *StorageService, crypto *CryptoService) *AccountManager {
	am := &AccountManager{
		accounts:     make(map[string]*KiroAccount),
		settings:     DefaultAccountSettings(),
		storage:      storage,
		crypto:       crypto,
		authService:  NewAuthService(),
		quotaService: NewQuotaService(),
		system:       NewKiroSystem(),
		mutex:        sync.RWMutex{},
		tags:         []Tag{},
	}

	// Load existing accounts from storage
	am.loadAccounts()

	return am
}

// SetContext sets the Wails context for event emission
func (am *AccountManager) SetContext(ctx context.Context) {
	am.ctx = ctx
}

// --- Account CRUD Operations ---

// AddAccount adds a new account to the manager
func (am *AccountManager) AddAccount(account *KiroAccount) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	if account == nil {
		return fmt.Errorf("account cannot be nil")
	}

	// Generate ID if not provided
	if account.ID == "" {
		account.ID = am.generateAccountID()
	}

	// Check if account already exists
	if _, exists := am.accounts[account.ID]; exists {
		return fmt.Errorf("account with ID %s already exists", account.ID)
	}

	// Check for duplicate email
	for _, existingAccount := range am.accounts {
		if existingAccount.Email == account.Email {
			return fmt.Errorf("account with email %s already exists", account.Email)
		}
	}

	// Set creation time if not set
	if account.CreatedAt.IsZero() {
		account.CreatedAt = time.Now()
	}

	// Set last used time
	account.LastUsed = time.Now()

	// If this is the first account, make it active
	if len(am.accounts) == 0 {
		account.IsActive = true
		am.activeID = account.ID
	} else {
		account.IsActive = false
	}

	// Add to accounts map
	am.accounts[account.ID] = account

	// Save to storage
	if err := am.saveAccounts(); err != nil {
		// Rollback
		delete(am.accounts, account.ID)
		return fmt.Errorf("failed to save account: %w", err)
	}

	// Emit event
	am.emitEvent("kiro-account-added", account)

	return nil
}

// RemoveAccount removes an account from the manager
func (am *AccountManager) RemoveAccount(id string) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	account, exists := am.accounts[id]
	if !exists {
		return fmt.Errorf("account with ID %s not found", id)
	}

	// Cannot remove active account if it's the only one
	if account.IsActive && len(am.accounts) == 1 {
		return fmt.Errorf("cannot remove the only active account")
	}

	// If removing active account, switch to another one
	if account.IsActive && len(am.accounts) > 1 {
		for _, otherAccount := range am.accounts {
			if otherAccount.ID != id {
				otherAccount.IsActive = true
				am.activeID = otherAccount.ID
				break
			}
		}
	}

	// Remove from map
	delete(am.accounts, id)

	// Save to storage
	if err := am.saveAccounts(); err != nil {
		// Rollback
		am.accounts[id] = account
		return fmt.Errorf("failed to save accounts after removal: %w", err)
	}

	// Emit event
	am.emitEvent("kiro-account-removed", id)

	return nil
}

// UpdateAccount updates an existing account
func (am *AccountManager) UpdateAccount(id string, updates map[string]interface{}) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	account, exists := am.accounts[id]
	if !exists {
		return fmt.Errorf("account with ID %s not found", id)
	}

	// Create a backup for rollback
	backup := *account
	quotaChanged := false
	var updatedQuota QuotaInfo
	tokenExpiryChanged := false
	var updatedTokenExpiry time.Time

	// Apply updates
	for key, value := range updates {
		switch key {
		case "displayName":
			if displayName, ok := value.(string); ok {
				account.DisplayName = displayName
			}
		case "avatar":
			if avatar, ok := value.(string); ok {
				account.Avatar = avatar
			}
		case "tags":
			if tags, ok := value.([]string); ok {
				account.Tags = tags
			}
		case "notes":
			if notes, ok := value.(string); ok {
				account.Notes = notes
			}
		case "bearerToken":
			if token, ok := value.(string); ok {
				account.BearerToken = token
			}
		case "refreshToken":
			if token, ok := value.(string); ok {
				account.RefreshToken = token
			}
		case "tokenExpiry":
			if expiry, ok := value.(time.Time); ok {
				account.TokenExpiry = expiry
				tokenExpiryChanged = true
				updatedTokenExpiry = expiry
			}
		case "quota":
			if quota, ok := value.(QuotaInfo); ok {
				account.Quota = quota
				quotaChanged = true
				updatedQuota = quota
			}
		case "subscriptionType":
			if subType, ok := value.(SubscriptionType); ok {
				account.SubscriptionType = subType
			}
		default:
			return fmt.Errorf("unknown field: %s", key)
		}
	}

	// Save to storage
	if err := am.saveAccounts(); err != nil {
		// Rollback
		am.accounts[id] = &backup
		return fmt.Errorf("failed to save account updates: %w", err)
	}

	// Emit event
	am.emitEvent("kiro-account-updated", account)
	if quotaChanged {
		am.emitEvent("kiro-quota-updated", id, updatedQuota)
	}
	if tokenExpiryChanged {
		am.emitEvent("kiro-token-refreshed", id, map[string]interface{}{
			"expiresAt": updatedTokenExpiry,
		})
	}

	return nil
}

// GetAccount retrieves an account by ID
func (am *AccountManager) GetAccount(id string) (*KiroAccount, error) {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	account, exists := am.accounts[id]
	if !exists {
		return nil, fmt.Errorf("account with ID %s not found", id)
	}

	// Return a copy to prevent external modification
	accountCopy := *account
	return &accountCopy, nil
}

// GetSettings returns account settings
func (am *AccountManager) GetSettings() (AccountSettings, error) {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	return am.settings, nil
}

// UpdateSettings updates account settings
func (am *AccountManager) UpdateSettings(settings AccountSettings) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	am.settings = settings
	return am.saveAccounts()
}

// ListAccounts returns all accounts
func (am *AccountManager) ListAccounts() []*KiroAccount {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	accounts := make([]*KiroAccount, 0, len(am.accounts))
	for _, account := range am.accounts {
		// Return copies to prevent external modification
		accountCopy := *account
		accounts = append(accounts, &accountCopy)
	}

	return accounts
}

// --- Account Switching ---

// SwitchAccount switches the active account
func (am *AccountManager) SwitchAccount(id string) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	newAccount, exists := am.accounts[id]
	if !exists {
		return fmt.Errorf("account with ID %s not found", id)
	}

	// Get current active account
	var oldActiveID string
	for _, account := range am.accounts {
		if account.IsActive && account.ID != id {
			account.IsActive = false
			oldActiveID = account.ID
			break
		}
	}

	// Set new active account (even if it's already active, we reapply it)
	newAccount.IsActive = true
	newAccount.LastUsed = time.Now()
	am.activeID = id

	// Apply to OpenCode (write to kiro-accounts.json)
	openCodeSystem := NewOpenCodeKiroSystem()
	if err := openCodeSystem.ApplyAccountToOpenCode(newAccount); err != nil {
		return fmt.Errorf("failed to apply account to OpenCode: %w", err)
	}

	// If AutoChangeMachineID is enabled, also update Kiro IDE machine IDs
	// (This is for users who also use Kiro IDE)
	if am.settings.AutoChangeMachineID {
		if err := am.system.ApplyAccountToSystem(newAccount, true); err != nil {
			// Log warning but don't fail the switch
			fmt.Printf("Warning: failed to update Kiro IDE machine ID: %v\n", err)
		}
	}

	// Save to storage
	if err := am.saveAccounts(); err != nil {
		// Rollback
		if oldActiveID != "" && oldActiveID != id {
			am.accounts[oldActiveID].IsActive = true
		}
		newAccount.IsActive = false
		am.activeID = oldActiveID
		return fmt.Errorf("failed to save account switch: %w", err)
	}

	// Emit event
	am.emitEvent("kiro-account-switched", map[string]string{
		"newAccountId": id,
		"oldAccountId": oldActiveID,
	})

	return nil
}

// GetActiveAccount returns the currently active account
func (am *AccountManager) GetActiveAccount() (*KiroAccount, error) {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	if am.activeID == "" {
		return nil, fmt.Errorf("no active account")
	}

	account, exists := am.accounts[am.activeID]
	if !exists {
		return nil, fmt.Errorf("active account not found")
	}

	// Return a copy
	accountCopy := *account
	return &accountCopy, nil
}

// --- Batch Operations ---

// BatchRefreshTokens refreshes tokens for multiple accounts
func (am *AccountManager) BatchRefreshTokens(ids []string) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	var errors []string
	successCount := 0

	for _, id := range ids {
		account, exists := am.accounts[id]
		if !exists {
			errors = append(errors, fmt.Sprintf("account %s not found", id))
			continue
		}

		if account.RefreshToken == "" {
			errors = append(errors, fmt.Sprintf("account %s has no refresh token", id))
			continue
		}

		// Refresh token using auth service
		tokenInfo, err := am.authService.RefreshToken(account.RefreshToken)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to refresh token for %s: %v", id, err))
			continue
		}

		// Update account with new token info
		account.BearerToken = tokenInfo.AccessToken
		if tokenInfo.RefreshToken != "" {
			account.RefreshToken = tokenInfo.RefreshToken
		}
		account.TokenExpiry = tokenInfo.ExpiresAt

		successCount++
	}

	// Save accounts if any were updated
	if successCount > 0 {
		if err := am.saveAccounts(); err != nil {
			errors = append(errors, fmt.Sprintf("failed to save updated accounts: %v", err))
		}
	}

	// Emit batch refresh event
	am.emitEvent("kiro-batch-refresh-completed", map[string]interface{}{
		"successCount": successCount,
		"totalCount":   len(ids),
		"errors":       errors,
	})

	if len(errors) > 0 {
		return fmt.Errorf("batch refresh completed with errors: %v", errors)
	}

	return nil
}

// BatchDeleteAccounts deletes multiple accounts
func (am *AccountManager) BatchDeleteAccounts(ids []string) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Check if we're trying to delete all accounts
	if len(ids) >= len(am.accounts) {
		return fmt.Errorf("cannot delete all accounts")
	}

	var errors []string
	var deletedAccounts []*KiroAccount
	successCount := 0

	// Check if active account is being deleted
	activeAccountDeleted := false
	for _, id := range ids {
		if account, exists := am.accounts[id]; exists && account.IsActive {
			activeAccountDeleted = true
			break
		}
	}

	// If active account is being deleted, find a replacement
	var newActiveID string
	if activeAccountDeleted {
		for accountID := range am.accounts {
			found := false
			for _, deleteID := range ids {
				if accountID == deleteID {
					found = true
					break
				}
			}
			if !found {
				newActiveID = accountID
				break
			}
		}
	}

	// Delete accounts
	for _, id := range ids {
		account, exists := am.accounts[id]
		if !exists {
			errors = append(errors, fmt.Sprintf("account %s not found", id))
			continue
		}

		deletedAccounts = append(deletedAccounts, account)
		delete(am.accounts, id)
		successCount++
	}

	// Set new active account if needed
	if activeAccountDeleted && newActiveID != "" {
		am.accounts[newActiveID].IsActive = true
		am.activeID = newActiveID
	}

	// Save accounts
	if successCount > 0 {
		if err := am.saveAccounts(); err != nil {
			// Rollback deletions
			for _, account := range deletedAccounts {
				am.accounts[account.ID] = account
			}
			return fmt.Errorf("failed to save after batch delete: %w", err)
		}
	}

	// Emit batch delete event
	am.emitEvent("kiro-batch-delete-completed", map[string]interface{}{
		"successCount": successCount,
		"totalCount":   len(ids),
		"errors":       errors,
	})

	if len(errors) > 0 {
		return fmt.Errorf("batch delete completed with errors: %v", errors)
	}

	return nil
}

// BatchAddTags adds tags to multiple accounts
func (am *AccountManager) BatchAddTags(ids []string, tags []string) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	var errors []string
	successCount := 0

	for _, id := range ids {
		account, exists := am.accounts[id]
		if !exists {
			errors = append(errors, fmt.Sprintf("account %s not found", id))
			continue
		}

		// Add tags to account
		for _, tag := range tags {
			account.AddTag(tag)
		}
		successCount++
	}

	// Save accounts if any were updated
	if successCount > 0 {
		if err := am.saveAccounts(); err != nil {
			errors = append(errors, fmt.Sprintf("failed to save updated accounts: %v", err))
		}
	}

	// Emit batch tag event
	am.emitEvent("kiro-batch-tag-completed", map[string]interface{}{
		"successCount": successCount,
		"totalCount":   len(ids),
		"addedTags":    tags,
		"errors":       errors,
	})

	if len(errors) > 0 {
		return fmt.Errorf("batch tag operation completed with errors: %v", errors)
	}

	return nil
}

// --- Data Management ---

// ExportAccounts exports accounts to JSON with optional encryption
func (am *AccountManager) ExportAccounts(password string) ([]byte, error) {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	// Create export data structure
	exportData := &AccountData{
		Version:         "1.0",
		Accounts:        make([]*KiroAccount, 0, len(am.accounts)),
		ActiveAccountID: am.activeID,
		Settings:        DefaultAccountSettings(),
		Tags:            am.tags,
		LastUpdated:     time.Now(),
	}

	// Add all accounts
	for _, account := range am.accounts {
		exportData.Accounts = append(exportData.Accounts, account)
	}

	// Serialize to JSON
	data, err := am.storage.SerializeAccountData(exportData)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize account data: %w", err)
	}

	// Encrypt if password provided
	if password != "" {
		encryptedData, err := am.crypto.EncryptWithPassword(data, password)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt export data: %w", err)
		}
		return encryptedData, nil
	}

	return data, nil
}

// ImportAccounts imports accounts from JSON data
func (am *AccountManager) ImportAccounts(data []byte, password string) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Decrypt if password provided
	if password != "" {
		decryptedData, err := am.crypto.DecryptWithPassword(data, password)
		if err != nil {
			return fmt.Errorf("failed to decrypt import data: %w", err)
		}
		data = decryptedData
	}

	// Deserialize account data
	accountData, err := am.storage.DeserializeAccountData(data)
	if err != nil {
		return fmt.Errorf("failed to deserialize account data: %w", err)
	}

	// Validate and import accounts
	importedCount := 0
	var errors []string

	for _, account := range accountData.Accounts {
		// Check for duplicate email
		duplicate := false
		for _, existingAccount := range am.accounts {
			if existingAccount.Email == account.Email {
				errors = append(errors, fmt.Sprintf("account with email %s already exists", account.Email))
				duplicate = true
				break
			}
		}

		if duplicate {
			continue
		}

		// Generate new ID to avoid conflicts
		account.ID = am.generateAccountID()
		account.IsActive = false // Don't activate imported accounts by default

		// Add to accounts
		am.accounts[account.ID] = account
		importedCount++
	}

	// Save accounts if any were imported
	if importedCount > 0 {
		if err := am.saveAccounts(); err != nil {
			return fmt.Errorf("failed to save imported accounts: %w", err)
		}
	}

	// Emit import event
	am.emitEvent("kiro-accounts-imported", map[string]interface{}{
		"importedCount": importedCount,
		"totalCount":    len(accountData.Accounts),
		"errors":        errors,
	})

	if len(errors) > 0 {
		return fmt.Errorf("import completed with errors: %v", errors)
	}

	return nil
}

// --- Tag Management ---

// GetTags returns all tags
func (am *AccountManager) GetTags() []Tag {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	tags := make([]Tag, len(am.tags))
	copy(tags, am.tags)
	return tags
}

// AddTag adds a new tag
func (am *AccountManager) AddTag(tag Tag) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Check if tag already exists
	for _, t := range am.tags {
		if t.Name == tag.Name {
			return fmt.Errorf("tag with name %s already exists", tag.Name)
		}
	}

	am.tags = append(am.tags, tag)

	// Save to storage
	if err := am.saveAccounts(); err != nil {
		return fmt.Errorf("failed to save tags: %w", err)
	}

	return nil
}

// DeleteTag deletes a tag
func (am *AccountManager) DeleteTag(tagName string) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	found := false
	for i, t := range am.tags {
		if t.Name == tagName {
			am.tags = append(am.tags[:i], am.tags[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("tag with name %s not found", tagName)
	}

	// Remove tag from all accounts
	for _, account := range am.accounts {
		account.RemoveTag(tagName)
	}

	// Save to storage
	if err := am.saveAccounts(); err != nil {
		return fmt.Errorf("failed to save tags: %w", err)
	}

	return nil
}

// --- Helper Methods ---

// generateAccountID generates a unique account ID
func (am *AccountManager) generateAccountID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return "kiro-" + hex.EncodeToString(bytes)
}

// loadAccounts loads accounts from storage
func (am *AccountManager) loadAccounts() error {
	accountData, err := am.storage.LoadAccountData()
	if err != nil {
		// If no data exists, start with empty state
		return nil
	}

	am.accounts = make(map[string]*KiroAccount)
	am.settings = accountData.Settings // Load settings
	am.tags = accountData.Tags         // Load tags
	for _, account := range accountData.Accounts {
		am.accounts[account.ID] = account
		if account.IsActive {
			am.activeID = account.ID
		}
	}

	return nil
}

// saveAccounts saves accounts to storage
func (am *AccountManager) saveAccounts() error {
	accountData := &AccountData{
		Version:         "1.0",
		Accounts:        make([]*KiroAccount, 0, len(am.accounts)),
		ActiveAccountID: am.activeID,
		Settings:        am.settings, // Save current settings
		Tags:            am.tags,     // Save tags
		LastUpdated:     time.Now(),
	}

	for _, account := range am.accounts {
		accountData.Accounts = append(accountData.Accounts, account)
	}

	return am.storage.SaveAccountData(accountData)
}

// emitEvent emits a Wails event
func (am *AccountManager) emitEvent(eventName string, data ...interface{}) {
	if am.ctx != nil {
		runtime.EventsEmit(am.ctx, eventName, data...)
	}
}

// --- Account Statistics ---

// GetAccountStats returns statistics about managed accounts
func (am *AccountManager) GetAccountStats() map[string]interface{} {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	stats := map[string]interface{}{
		"totalAccounts":     len(am.accounts),
		"activeAccountId":   am.activeID,
		"subscriptionTypes": make(map[string]int),
		"loginMethods":      make(map[string]int),
		"expiredTokens":     0,
		"expiringSoon":      0,
	}

	for _, account := range am.accounts {
		// Count subscription types
		subType := string(account.SubscriptionType)
		if count, exists := stats["subscriptionTypes"].(map[string]int)[subType]; exists {
			stats["subscriptionTypes"].(map[string]int)[subType] = count + 1
		} else {
			stats["subscriptionTypes"].(map[string]int)[subType] = 1
		}

		// Count login methods
		loginMethod := string(account.LoginMethod)
		if count, exists := stats["loginMethods"].(map[string]int)[loginMethod]; exists {
			stats["loginMethods"].(map[string]int)[loginMethod] = count + 1
		} else {
			stats["loginMethods"].(map[string]int)[loginMethod] = 1
		}

		// Count expired and expiring tokens
		if account.IsTokenExpired() {
			stats["expiredTokens"] = stats["expiredTokens"].(int) + 1
		} else if account.IsTokenExpiringSoon(24 * time.Hour) {
			stats["expiringSoon"] = stats["expiringSoon"].(int) + 1
		}
	}

	return stats
}

// GetQuotaAlerts returns quota alerts for all accounts
func (am *AccountManager) GetQuotaAlerts(threshold float64) []QuotaAlert {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	var alerts []QuotaAlert
	for _, account := range am.accounts {
		accountAlerts := account.GetQuotaAlerts(threshold)
		alerts = append(alerts, accountAlerts...)
	}

	return alerts
}
