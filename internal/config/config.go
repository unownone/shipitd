package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server     ServerConfig     `mapstructure:"server" validate:"required"`
	Auth       AuthConfig       `mapstructure:"auth" validate:"required"`
	Tunnels    []TunnelConfig   `mapstructure:"tunnels" validate:"dive"`
	Connection ConnectionConfig  `mapstructure:"connection"`
	Logging    LoggingConfig    `mapstructure:"logging"`
}

// ServerConfig represents server connection settings
type ServerConfig struct {
	Domain         string `mapstructure:"domain" validate:"required"`
	APIPort        int    `mapstructure:"api_port" validate:"required,min=1,max=65535"`
	DataPlanePort  int    `mapstructure:"data_plane_port" validate:"required,min=1,max=65535"`
	TLSVerify      bool   `mapstructure:"tls_verify"`
}

// AuthConfig represents authentication settings
type AuthConfig struct {
	APIKey      string `mapstructure:"api_key" validate:"required"`
	AutoRefresh bool   `mapstructure:"auto_refresh"`
}

// TunnelConfig represents a tunnel configuration
type TunnelConfig struct {
	Name       string `mapstructure:"name" validate:"required"`
	Protocol   string `mapstructure:"protocol" validate:"required,oneof=http tcp"`
	LocalPort  int    `mapstructure:"local_port" validate:"required,min=1,max=65535"`
	Subdomain  string `mapstructure:"subdomain"`
	AutoStart  bool   `mapstructure:"auto_start"`
}

// ConnectionConfig represents connection pool settings
type ConnectionConfig struct {
	PoolSize              int           `mapstructure:"pool_size" validate:"min=1,max=100"`
	HeartbeatInterval     time.Duration `mapstructure:"heartbeat_interval" validate:"min=5s,max=5m"`
	ReconnectInterval     time.Duration `mapstructure:"reconnect_interval" validate:"min=1s,max=1m"`
	MaxReconnectAttempts  int           `mapstructure:"max_reconnect_attempts" validate:"min=1,max=100"`
	ConnectionTimeout     time.Duration `mapstructure:"connection_timeout" validate:"min=5s,max=5m"`
}

// LoggingConfig represents logging settings
type LoggingConfig struct {
	Level  string `mapstructure:"level" validate:"oneof=debug info warn error"`
	Format string `mapstructure:"format" validate:"oneof=json text"`
	File   string `mapstructure:"file"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Domain:        "your-shipit-server.com",
			APIPort:       443,
			DataPlanePort: 7223,
			TLSVerify:     true,
		},
		Auth: AuthConfig{
			APIKey:      "",
			AutoRefresh: true,
		},
		Tunnels: []TunnelConfig{
			{
				Name:      "web-app",
				Protocol:  "http",
				LocalPort: 3000,
				Subdomain: "myapp",
				AutoStart: true,
			},
		},
		Connection: ConnectionConfig{
			PoolSize:              10,
			HeartbeatInterval:     30 * time.Second,
			ReconnectInterval:     5 * time.Second,
			MaxReconnectAttempts:  10,
			ConnectionTimeout:     30 * time.Second,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			File:   "",
		},
	}
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	// Set up Viper
	v := viper.New()
	
	// Set default values
	setDefaults(v)
	
	// Set config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// Search for config file in common locations
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		v.AddConfigPath("$HOME/.shipitd")
		v.AddConfigPath("/etc/shipit")
	}
	
	// Read environment variables
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetEnvPrefix("SHIPIT")
	v.AutomaticEnv()
	
	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found, use defaults
	}
	
	// Unmarshal into struct
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	
	return &config, nil
}

// setDefaults sets default values in Viper
func setDefaults(v *viper.Viper) {
	defaults := DefaultConfig()
	
	// Server defaults
	v.SetDefault("server.domain", defaults.Server.Domain)
	v.SetDefault("server.api_port", defaults.Server.APIPort)
	v.SetDefault("server.data_plane_port", defaults.Server.DataPlanePort)
	v.SetDefault("server.tls_verify", defaults.Server.TLSVerify)
	
	// Auth defaults
	v.SetDefault("auth.auto_refresh", defaults.Auth.AutoRefresh)
	
	// Connection defaults
	v.SetDefault("connection.pool_size", defaults.Connection.PoolSize)
	v.SetDefault("connection.heartbeat_interval", defaults.Connection.HeartbeatInterval)
	v.SetDefault("connection.reconnect_interval", defaults.Connection.ReconnectInterval)
	v.SetDefault("connection.max_reconnect_attempts", defaults.Connection.MaxReconnectAttempts)
	v.SetDefault("connection.connection_timeout", defaults.Connection.ConnectionTimeout)
	
	// Logging defaults
	v.SetDefault("logging.level", defaults.Logging.Level)
	v.SetDefault("logging.format", defaults.Logging.Format)
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	validate := validator.New()
	return validate.Struct(config)
}

// SaveConfig saves configuration to file
func SaveConfig(config *Config, configPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Create a map that matches the expected YAML structure
	configMap := map[string]interface{}{
		"server": map[string]interface{}{
			"domain":           config.Server.Domain,
			"api_port":         config.Server.APIPort,
			"data_plane_port":  config.Server.DataPlanePort,
			"tls_verify":       config.Server.TLSVerify,
		},
		"auth": map[string]interface{}{
			"api_key":      config.Auth.APIKey,
			"auto_refresh": config.Auth.AutoRefresh,
		},
		"tunnels": func() []map[string]interface{} {
			tunnels := make([]map[string]interface{}, len(config.Tunnels))
			for i, tunnel := range config.Tunnels {
				tunnels[i] = map[string]interface{}{
					"name":       tunnel.Name,
					"protocol":   tunnel.Protocol,
					"local_port": tunnel.LocalPort,
					"subdomain":  tunnel.Subdomain,
					"auto_start": tunnel.AutoStart,
				}
			}
			return tunnels
		}(),
		"connection": map[string]interface{}{
			"pool_size":               config.Connection.PoolSize,
			"heartbeat_interval":      config.Connection.HeartbeatInterval,
			"reconnect_interval":      config.Connection.ReconnectInterval,
			"max_reconnect_attempts":  config.Connection.MaxReconnectAttempts,
			"connection_timeout":      config.Connection.ConnectionTimeout,
		},
		"logging": map[string]interface{}{
			"level":  config.Logging.Level,
			"format": config.Logging.Format,
			"file":   config.Logging.File,
		},
	}
	
	// Marshal to YAML
	yamlData, err := yaml.Marshal(configMap)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	// Write to file
	return os.WriteFile(configPath, yamlData, 0644)
}

// GetConfigPath returns the default config file path
func GetConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "./client.yaml"
	}
	return filepath.Join(home, ".shipitd", "config.yaml")
}

// CreateDefaultConfig creates a default configuration file
func CreateDefaultConfig(configPath string) error {
	config := DefaultConfig()
	return SaveConfig(config, configPath)
}

// ValidateConfigFile validates a configuration file
func ValidateConfigFile(configPath string) error {
	config, err := LoadConfig(configPath)
	if err != nil {
		return err
	}
	
	// Additional validation logic can be added here
	return validateConfig(config)
} 