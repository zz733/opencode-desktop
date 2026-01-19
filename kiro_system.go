package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

// KiroLocalToken represents the token file structure used by Kiro IDE
type KiroLocalToken struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresAt    string `json:"expiresAt"`
	AuthMethod   string `json:"authMethod"`
	Provider     string `json:"provider"`
	ProfileArn   string `json:"profileArn,omitempty"`
	// IdC specific fields
	ClientIdHash string `json:"clientIdHash,omitempty"`
	Region       string `json:"region,omitempty"`
}

// KiroTelemetryInfo represents telemetry settings in storage.json
type KiroTelemetryInfo struct {
	MachineID        string `json:"telemetry.machineId,omitempty"`
	SqmID            string `json:"telemetry.sqmId,omitempty"`
	DevDeviceID      string `json:"telemetry.devDeviceId,omitempty"`
	ServiceMachineID string `json:"storage.serviceMachineId,omitempty"` // From DB
}

// KiroSystem handles system-level interactions for Kiro IDE
type KiroSystem struct{}

// NewKiroSystem creates a new KiroSystem instance
func NewKiroSystem() *KiroSystem {
	return &KiroSystem{}
}

// GetKiroDataDir returns the Kiro data directory based on OS
func (ks *KiroSystem) GetKiroDataDir() (string, error) {
	var dir string
	switch runtime.GOOS {
	case "windows":
		dir = os.Getenv("APPDATA")
		if dir == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
		return filepath.Join(dir, "Kiro"), nil
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", "Kiro"), nil
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// GetKiroAuthTokenPath returns the path to the Kiro auth token file
func (ks *KiroSystem) GetKiroAuthTokenPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".aws", "sso", "cache", "kiro-auth-token.json"), nil
}

// GenerateMachineID generates/retrieves the unique machine ID
func (ks *KiroSystem) GenerateMachineID() string {
	// Try to get actual system machine ID first
	id := ks.getSystemMachineID()
	if id != "" {
		return id
	}

	// Fallback to random generation if system ID fails
	return ks.generateRandomMachineID()
}

// getSystemMachineID retrieves the OS-specific machine ID
func (ks *KiroSystem) getSystemMachineID() string {
	switch runtime.GOOS {
	case "darwin":
		// macOS: ioreg -rd1 -c IOPlatformExpertDevice | grep IOPlatformUUID
		cmd := exec.Command("ioreg", "-rd1", "-c", "IOPlatformExpertDevice")
		out, err := cmd.Output()
		if err == nil {
			lines := strings.Split(string(out), "\n")
			for _, line := range lines {
				if strings.Contains(line, "IOPlatformUUID") {
					parts := strings.Split(line, "=")
					if len(parts) == 2 {
						return strings.Trim(strings.TrimSpace(parts[1]), "\"")
					}
				}
			}
		}
	case "windows":
		// Windows: reg query HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Cryptography /v MachineGuid
		cmd := exec.Command("reg", "query", "HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Cryptography", "/v", "MachineGuid")
		out, err := cmd.Output()
		if err == nil {
			lines := strings.Split(string(out), "\n")
			for _, line := range lines {
				if strings.Contains(line, "MachineGuid") {
					parts := strings.Fields(line)
					if len(parts) >= 3 {
						return parts[2]
					}
				}
			}
		}
	case "linux":
		// Linux: /etc/machine-id or /var/lib/dbus/machine-id
		if data, err := os.ReadFile("/etc/machine-id"); err == nil {
			return strings.TrimSpace(string(data))
		}
		if data, err := os.ReadFile("/var/lib/dbus/machine-id"); err == nil {
			return strings.TrimSpace(string(data))
		}
	}
	return ""
}

// generateRandomMachineID generates a random machine ID (fallback)
func (ks *KiroSystem) generateRandomMachineID() string {
	id := uuid.New().String()
	timestamp := time.Now().UnixNano()
	input := fmt.Sprintf("%s%d", id, timestamp)
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// GenerateSqmID generates a SQM ID (GUID format with braces)
func (ks *KiroSystem) GenerateSqmID() string {
	return fmt.Sprintf("{%s}", uuid.New().String())
}

// GenerateDevDeviceID generates a Dev Device ID (UUID format)
func (ks *KiroSystem) GenerateDevDeviceID() string {
	return uuid.New().String()
}

// ApplyAccountToSystem applies the account settings to the system
// This involves writing the token file and updating machine IDs if needed
func (ks *KiroSystem) ApplyAccountToSystem(account *KiroAccount, useBoundMachineID bool) error {
	// 1. Write Token File
	if err := ks.writeTokenFile(account); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	// 2. Handle Machine ID
	if useBoundMachineID {
		// If account has no bound ID, generate one and bind it
		if account.MachineID == "" {
			account.MachineID = ks.GenerateMachineID()
			account.SqmID = ks.GenerateSqmID()
			account.DevDeviceID = ks.GenerateDevDeviceID()
			// Note: The caller (AccountManager) needs to save the account after this!
		}

		if err := ks.updateMachineID(account.MachineID, account.SqmID, account.DevDeviceID); err != nil {
			return fmt.Errorf("failed to update machine ID: %w", err)
		}
	} else {
		// If not using bound ID, we might want to generate a random one to rotate it
		// Or keep existing. For "privacy", rotating is better.
		// Let's generate new random IDs if we are not using a bound one (simulating "reset")
		// But usually "useBoundMachineID" means "restore this account's specific ID".
		// If false, we might just leave it as is, OR generate new one if requested.
		// For this function, let's assume if useBoundMachineID is false, we do nothing to Machine ID
		// unless explicitly asked via another method.
	}

	return nil
}

// ResetMachineID generates new machine IDs and applies them
func (ks *KiroSystem) ResetMachineID() (*KiroTelemetryInfo, error) {
	mID := ks.GenerateMachineID()
	sqmID := ks.GenerateSqmID()
	devID := ks.GenerateDevDeviceID()

	if err := ks.updateMachineID(mID, sqmID, devID); err != nil {
		return nil, err
	}

	return &KiroTelemetryInfo{
		MachineID:   mID,
		SqmID:       sqmID,
		DevDeviceID: devID,
	}, nil
}

// updateMachineID updates the machine ID in storage.json and state.vscdb
func (ks *KiroSystem) updateMachineID(machineID, sqmID, devDeviceID string) error {
	kiroDir, err := ks.GetKiroDataDir()
	if err != nil {
		return err
	}

	// 1. Update storage.json
	storagePath := filepath.Join(kiroDir, "User", "globalStorage", "storage.json")
	if err := ks.updateStorageJson(storagePath, machineID, sqmID, devDeviceID); err != nil {
		return fmt.Errorf("failed to update storage.json: %w", err)
	}

	// 2. Update state.vscdb
	dbPath := filepath.Join(kiroDir, "User", "globalStorage", "state.vscdb")
	// serviceMachineID is usually same as machineID or generated similarly. Rust code generated a new one.
	// We'll use the machineID for consistency or generate new one. Rust generated new one.
	serviceMachineID := ks.GenerateMachineID()
	if err := ks.updateVscdb(dbPath, serviceMachineID); err != nil {
		// Log error but don't fail hard if DB is locked (Kiro running)
		fmt.Printf("Warning: failed to update state.vscdb: %v\n", err)
	}

	return nil
}

// writeTokenFile writes the auth token to disk
func (ks *KiroSystem) writeTokenFile(account *KiroAccount) error {
	tokenPath, err := ks.GetKiroAuthTokenPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(tokenPath), 0755); err != nil {
		return err
	}

	// Default values
	provider := "builderid"
	if account.Provider != "" {
		provider = string(account.Provider)
	}

	// Default ARN if missing (copied from Rust)
	profileArn := "arn:aws:codewhisperer:us-east-1:699475941385:profile/EHGA3GRVQMUK"

	// Construct token data
	// Note: We are assuming "social" auth method for now as IdC fields are not yet in KiroAccount
	tokenData := KiroLocalToken{
		AccessToken:  account.BearerToken,
		RefreshToken: account.RefreshToken,
		ExpiresAt:    time.Now().Add(1 * time.Hour).Format(time.RFC3339),
		AuthMethod:   "social",
		Provider:     provider,
		ProfileArn:   profileArn,
	}

	data, err := json.MarshalIndent(tokenData, "", "  ")
	if err != nil {
		return err
	}

	// Atomic write
	tmpPath := tokenPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return err
	}

	return os.Rename(tmpPath, tokenPath)
}

// updateStorageJson updates the telemetry IDs in storage.json
func (ks *KiroSystem) updateStorageJson(path string, machineID, sqmID, devDeviceID string) error {
	// Read existing file
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// If not exists, maybe create? But Kiro should have created it.
			return nil
		}
		return err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return err
	}

	// Update fields
	data["telemetry.machineId"] = machineID
	data["telemetry.sqmId"] = sqmID
	data["telemetry.devDeviceId"] = devDeviceID

	// Write back
	newContent, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, newContent, 0644)
}

// updateVscdb updates the service machine ID in the SQLite database
func (ks *KiroSystem) updateVscdb(path string, serviceMachineID string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return err
	}
	defer db.Close()

	query := "UPDATE ItemTable SET value = ? WHERE key = 'storage.serviceMachineId'"
	_, err = db.Exec(query, serviceMachineID)
	return err
}
