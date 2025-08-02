package security

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

// CredentialManager handles secure storage of credentials
type CredentialManager struct {
	serviceName string
	username    string
}

// NewCredentialManager creates a new credential manager
func NewCredentialManager(serviceName, username string) *CredentialManager {
	return &CredentialManager{
		serviceName: serviceName,
		username:    username,
	}
}

// StoreAPIKey securely stores an API key
func (cm *CredentialManager) StoreAPIKey(apiKey string) error {
	return keyring.Set(cm.serviceName, cm.username, apiKey)
}

// GetAPIKey retrieves a stored API key
func (cm *CredentialManager) GetAPIKey() (string, error) {
	return keyring.Get(cm.serviceName, cm.username)
}

// DeleteAPIKey removes a stored API key
func (cm *CredentialManager) DeleteAPIKey() error {
	return keyring.Delete(cm.serviceName, cm.username)
}

// StoreCredentials stores multiple credentials
func (cm *CredentialManager) StoreCredentials(credentials map[string]string) error {
	data, err := json.Marshal(credentials)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	return keyring.Set(cm.serviceName, cm.username, string(data))
}

// GetCredentials retrieves stored credentials
func (cm *CredentialManager) GetCredentials() (map[string]string, error) {
	data, err := keyring.Get(cm.serviceName, cm.username)
	if err != nil {
		return nil, err
	}

	var credentials map[string]string
	if err := json.Unmarshal([]byte(data), &credentials); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	return credentials, nil
}

// SecureConfigFile handles secure configuration file operations
type SecureConfigFile struct {
	filePath string
}

// NewSecureConfigFile creates a new secure config file handler
func NewSecureConfigFile(filePath string) *SecureConfigFile {
	return &SecureConfigFile{
		filePath: filePath,
	}
}

// SetSecurePermissions sets secure file permissions
func (scf *SecureConfigFile) SetSecurePermissions() error {
	// Set file permissions to 600 (owner read/write only)
	return os.Chmod(scf.filePath, 0600)
}

// ValidateFilePermissions checks if file permissions are secure
func (scf *SecureConfigFile) ValidateFilePermissions() error {
	info, err := os.Stat(scf.filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	mode := info.Mode()
	if mode&0077 != 0 {
		return fmt.Errorf("insecure file permissions: %v", mode)
	}

	return nil
}

// CreateSecureDirectory creates a directory with secure permissions
func CreateSecureDirectory(dirPath string) error {
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return nil
}

// ValidateSecurePath ensures a path is secure
func ValidateSecurePath(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if path is within user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	if !filepath.HasPrefix(absPath, homeDir) {
		return fmt.Errorf("path is outside home directory: %s", absPath)
	}

	return nil
}

// RotateCredentials handles credential rotation
type CredentialRotator struct {
	credentialManager *CredentialManager
}

// NewCredentialRotator creates a new credential rotator
func NewCredentialRotator(serviceName, username string) *CredentialRotator {
	return &CredentialRotator{
		credentialManager: NewCredentialManager(serviceName, username),
	}
}

// RotateAPIKey rotates an API key
func (cr *CredentialRotator) RotateAPIKey(newAPIKey string) error {
	// Store the new API key
	if err := cr.credentialManager.StoreAPIKey(newAPIKey); err != nil {
		return fmt.Errorf("failed to store new API key: %w", err)
	}

	return nil
}

// ValidateAPIKey validates an API key format
func ValidateAPIKey(apiKey string) error {
	if len(apiKey) < 32 {
		return fmt.Errorf("API key too short: minimum 32 characters required")
	}

	if len(apiKey) > 256 {
		return fmt.Errorf("API key too long: maximum 256 characters allowed")
	}

	// Check for valid characters (alphanumeric and some special chars)
	for _, char := range apiKey {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_' || char == '-') {
			return fmt.Errorf("API key contains invalid character: %c", char)
		}
	}

	return nil
}

// SanitizeInput sanitizes user input
func SanitizeInput(input string) string {
	// Remove null bytes and control characters
	var sanitized []rune
	for _, char := range input {
		if char >= 32 && char != 127 {
			sanitized = append(sanitized, char)
		}
	}
	return string(sanitized)
}

// ValidateURL validates a URL format
func ValidateURL(url string) error {
	if url == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	// Basic URL validation
	if len(url) > 2048 {
		return fmt.Errorf("URL too long: maximum 2048 characters allowed")
	}

	// Check for valid protocol
	if len(url) < 7 {
		return fmt.Errorf("URL too short")
	}

	protocol := url[:7]
	if protocol != "http://" && protocol != "https://" {
		return fmt.Errorf("invalid protocol: only http:// and https:// are allowed")
	}

	return nil
} 