# Configuration Manager Implementation

## Overview

The ConfigManager is a comprehensive configuration file and directory structure management system for the Kiro Multi-Account Manager. It provides centralized management of application configuration, directory structure, and file organization.

## Features

### 1. Cross-Platform Directory Management
- **Automatic OS Detection**: Determines appropriate configuration directories based on the operating system
- **Windows**: `%APPDATA%\OpenCode\KiroAccountManager`
- **macOS**: `~/Library/Application Support/OpenCode/KiroAccountManager`
- **Linux/Unix**: `~/.config/opencode/kiro-account-manager`

### 2. Directory Structure
```
Base Directory/
├── config/           # Configuration files
│   └── app.json     # Main application configuration
├── data/            # Application data
│   ├── accounts.json.enc  # Encrypted account data
│   ├── settings.json      # User settings
│   └── tags.json          # Tag definitions
├── logs/            # Log files
├── backups/         # Backup files
└── temp/            # Temporary files
```

### 3. Configuration Management
- **Default Configuration**: Provides sensible defaults for all settings
- **Configuration Validation**: Validates configuration integrity
- **Configuration Reset**: Reset to defaults with automatic backup
- **Version Management**: Handles configuration migrations

### 4. Security Features
- **Encrypted Storage**: Sensitive data is encrypted using AES-256-GCM
- **Secure Defaults**: Security-first configuration defaults
- **Key Management**: Integrated with CryptoService for encryption

### 5. Storage Management
- **Directory Size Calculation**: Monitor storage usage
- **Temporary File Management**: Automatic cleanup of temporary files
- **Backup Management**: Automatic backup creation and cleanup

## Implementation Details

### Core Components

#### ConfigManager
The main configuration management class that handles:
- Directory creation and validation
- Configuration file management
- Integration with other services
- Storage monitoring

#### Configuration Structures

##### AppConfig
Main application configuration containing:
- Application metadata (version, name)
- Security settings
- Storage configuration
- Logging configuration
- Directory paths

##### SecurityConfig
Security-related settings:
- Encryption preferences
- Key derivation methods
- Auto-lock timeout
- Password requirements

##### StorageConfig
Storage management settings:
- Backup configuration
- Cleanup policies
- Retention periods

##### LoggingConfig
Logging configuration:
- Log levels
- File rotation
- Size limits

### API Methods

#### Core Operations
```go
// Initialize the configuration manager
func (cm *ConfigManager) Initialize() error

// Load application configuration
func (cm *ConfigManager) LoadAppConfig() (*AppConfig, error)

// Save application configuration
func (cm *ConfigManager) SaveAppConfig(config *AppConfig) error

// Validate configuration
func (cm *ConfigManager) Validate() error
```

#### Directory Management
```go
// Get configuration paths
func (cm *ConfigManager) GetPaths() ConfigPaths

// Get specific directory paths
func (cm *ConfigManager) GetDataDirectory() string
func (cm *ConfigManager) GetConfigDirectory() string
func (cm *ConfigManager) GetLogsDirectory() string
func (cm *ConfigManager) GetBackupDirectory() string
func (cm *ConfigManager) GetTempDirectory() string
```

#### Storage Operations
```go
// Get storage information
func (cm *ConfigManager) GetStorageInfo() (map[string]interface{}, error)

// Calculate directory size
func (cm *ConfigManager) GetDirectorySize(dirPath string) (int64, error)

// Cleanup temporary files
func (cm *ConfigManager) CleanupTempDirectory() error

// Create temporary file
func (cm *ConfigManager) CreateTempFile(prefix string) (*os.File, error)
```

#### Maintenance Operations
```go
// Reset configuration to defaults
func (cm *ConfigManager) Reset() error

// Migrate configuration from older versions
func (cm *ConfigManager) MigrateConfiguration(fromVersion string) error
```

### Integration with Existing Services

#### StorageService Integration
The ConfigManager provides directory paths to the StorageService:
```go
// Get data directory from config manager
dataDir := configMgr.GetDataDirectory()

// Initialize storage service with config-managed directory
storage := NewStorageService(dataDir, crypto)
```

#### AccountManager Integration
The AccountManager uses ConfigManager-provided paths through StorageService:
```go
// Initialize account manager with config-managed storage
accountMgr := NewAccountManager(storage, crypto)
```

#### Wails API Integration
Configuration management is exposed through Wails API methods:
```go
// Get configuration paths
func (a *App) GetConfigPaths() (ConfigPaths, error)

// Get application configuration
func (a *App) GetAppConfig() (*AppConfig, error)

// Update application configuration
func (a *App) UpdateAppConfig(config *AppConfig) error

// Get storage information
func (a *App) GetStorageInfo() (map[string]interface{}, error)

// Cleanup temporary files
func (a *App) CleanupTempFiles() error

// Validate configuration
func (a *App) ValidateConfiguration() error

// Reset configuration
func (a *App) ResetConfiguration() error
```

## Usage Examples

### Basic Usage
```go
// Create crypto service
crypto := NewCryptoService("master-key")

// Create and initialize config manager
configMgr, err := NewConfigManager(crypto)
if err != nil {
    log.Fatal(err)
}

err = configMgr.Initialize()
if err != nil {
    log.Fatal(err)
}

// Get configuration paths
paths := configMgr.GetPaths()
fmt.Printf("Data directory: %s\n", paths.DataDir)
```

### Configuration Management
```go
// Load configuration
config, err := configMgr.LoadAppConfig()
if err != nil {
    log.Fatal(err)
}

// Modify configuration
config.Security.AutoLockTimeout = 60
config.Storage.MaxBackups = 20

// Save configuration
err = configMgr.SaveAppConfig(config)
if err != nil {
    log.Fatal(err)
}
```

### Storage Monitoring
```go
// Get storage information
storageInfo, err := configMgr.GetStorageInfo()
if err != nil {
    log.Fatal(err)
}

totalSize := storageInfo["total_size"].(int64)
fmt.Printf("Total storage used: %.2f KB\n", float64(totalSize)/1024)
```

### Temporary File Management
```go
// Create temporary file
tempFile, err := configMgr.CreateTempFile("processing_")
if err != nil {
    log.Fatal(err)
}
defer tempFile.Close()

// Use temporary file
tempFile.WriteString("temporary data")

// Cleanup all temporary files
err = configMgr.CleanupTempDirectory()
if err != nil {
    log.Printf("Cleanup failed: %v", err)
}
```

## Default Configuration

### Security Defaults
- Encryption: Enabled (AES-256-GCM)
- Key Derivation: PBKDF2
- Auto-lock Timeout: 30 minutes
- Password on Start: Disabled

### Storage Defaults
- Max Backups: 10
- Auto Backup: Enabled
- Backup Interval: 24 hours
- Backup Retention: 30 days

### Logging Defaults
- Logging: Enabled
- Level: INFO
- Max File Size: 10MB
- Max Files: 5
- Daily Rotation: Enabled

## Testing

The ConfigManager includes comprehensive tests covering:

### Unit Tests
- Configuration creation and initialization
- Directory management
- Configuration file operations
- Validation logic
- Error handling

### Integration Tests
- Integration with StorageService
- Integration with AccountManager
- End-to-end configuration workflows

### Test Coverage
All tests pass successfully:
```
=== RUN   TestNewConfigManager
--- PASS: TestNewConfigManager (0.00s)
=== RUN   TestInitialize
--- PASS: TestInitialize (0.00s)
=== RUN   TestCreateDirectories
--- PASS: TestCreateDirectories (0.00s)
=== RUN   TestSaveAndLoadAppConfig
--- PASS: TestSaveAndLoadAppConfig (0.00s)
=== RUN   TestLoadAppConfigDefault
--- PASS: TestLoadAppConfigDefault (0.00s)
=== RUN   TestValidateDirectoryStructure
--- PASS: TestValidateDirectoryStructure (0.00s)
=== RUN   TestGetPaths
--- PASS: TestGetPaths (0.00s)
=== RUN   TestCleanupTempDirectory
--- PASS: TestCleanupTempDirectory (0.00s)
=== RUN   TestCreateTempFile
--- PASS: TestCreateTempFile (0.00s)
=== RUN   TestGetDirectorySize
--- PASS: TestGetDirectorySize (0.00s)
=== RUN   TestGetStorageInfo
--- PASS: TestGetStorageInfo (0.00s)
=== RUN   TestValidate
--- PASS: TestValidate (0.00s)
=== RUN   TestReset
--- PASS: TestReset (0.00s)
=== RUN   TestDefaultAppConfig
--- PASS: TestDefaultAppConfig (0.00s)
=== RUN   TestConfigManagerIntegration
--- PASS: TestConfigManagerIntegration (0.00s)
```

## Error Handling

The ConfigManager provides robust error handling for:
- Directory creation failures
- Configuration file corruption
- Permission issues
- Disk space problems
- Invalid configuration values

## Security Considerations

### Data Protection
- Sensitive configuration data can be encrypted
- Secure defaults for all security settings
- Integration with CryptoService for encryption

### File Permissions
- Configuration files: 0600 (owner read/write only)
- Directories: 0755 (owner full access, others read/execute)

### Backup Security
- Backups maintain same security level as originals
- Automatic cleanup of old backups
- Secure temporary file handling

## Future Enhancements

### Planned Features
1. **Configuration Profiles**: Support for multiple configuration profiles
2. **Remote Configuration**: Support for remote configuration management
3. **Configuration Validation Schema**: JSON schema validation for configurations
4. **Configuration Diff**: Show differences between configurations
5. **Configuration Import/Export**: Import/export configuration between installations

### Migration Support
The ConfigManager includes infrastructure for handling configuration migrations when upgrading between versions.

## Conclusion

The ConfigManager provides a robust, secure, and cross-platform solution for configuration and directory management in the Kiro Multi-Account Manager. It integrates seamlessly with existing services while providing comprehensive configuration management capabilities.