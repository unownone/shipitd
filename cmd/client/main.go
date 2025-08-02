package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/unownone/shipitd/internal/client"
	"github.com/unownone/shipitd/internal/config"
	"github.com/unownone/shipitd/internal/logger"
)

var (
	cfgFile string
	verbose bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "shipitd",
	Short: "ShipIt Client Daemon - Expose local services to the internet",
	Long: `ShipIt Client Daemon is a Go application that connects to the ShipIt server
to expose local services to the internet through secure tunnels.

The client operates in two planes:
- Control Plane: HTTP API communication for tunnel management and authentication
- Data Plane: Custom TLS protocol for high-throughput traffic forwarding`,
}

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the ShipIt client daemon",
	Long:  `Start the ShipIt client daemon and begin managing tunnels according to configuration.`,
	Run:   runStart,
}

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the ShipIt client daemon",
	Long:  `Stop the ShipIt client daemon and clean up all tunnels.`,
	Run:   runStop,
}

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of the ShipIt client daemon",
	Long:  `Show the current status of the ShipIt client daemon and all tunnels.`,
	Run:   runStatus,
}

// authCmd represents the auth command
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
	Long:  `Commands for managing authentication with the ShipIt server.`,
}

// authTestCmd represents the auth test command
var authTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test authentication with the ShipIt server",
	Long:  `Test the API key authentication with the ShipIt server.`,
	Run:   runAuthTest,
}

// tunnelsCmd represents the tunnels command
var tunnelsCmd = &cobra.Command{
	Use:   "tunnels",
	Short: "Tunnel management commands",
	Long:  `Commands for managing tunnels with the ShipIt server.`,
}

// tunnelsListCmd represents the tunnels list command
var tunnelsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tunnels",
	Long:  `List all tunnels for the authenticated user.`,
	Run:   runTunnelsList,
}

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration commands",
	Long:  `Commands for managing the ShipIt client configuration.`,
}

// configShowCmd represents the config show command
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  `Show the current configuration that will be used by the client.`,
	Run:   runConfigShow,
}

// configInitCmd represents the config init command
var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration file",
	Long:  `Create a new configuration file with default values.`,
	Run:   runConfigInit,
}

// configValidateCmd represents the config validate command
var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration file",
	Long:  `Validate the configuration file for errors and warnings.`,
	Run:   runConfigValidate,
}

// configEditCmd represents the config edit command
var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit configuration file",
	Long:  `Open the configuration file in the default editor.`,
	Run:   runConfigEdit,
}



func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.shipitd/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "verbose output")

	// Add subcommands
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(tunnelsCmd)
	rootCmd.AddCommand(configCmd)

	// Add auth subcommands
	authCmd.AddCommand(authTestCmd)

	// Add tunnels subcommands
	tunnelsCmd.AddCommand(tunnelsListCmd)

	// Add config subcommands
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configEditCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runStart(cmd *cobra.Command, args []string) {
	// Load configuration
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Setup logging
	if err := logger.InitLogger(cfg.Logging.Level, cfg.Logging.Format, cfg.Logging.File); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup logging: %v\n", err)
		os.Exit(1)
	}
	log := logger.GetLogger()
	log.Info("Starting ShipIt client daemon")

	// Create tunnel manager
	tunnelManager := client.NewTunnelManager(cfg, log)

	// Start tunnels from configuration
	for _, tunnelConfig := range cfg.Tunnels {
		if tunnelConfig.AutoStart {
			if err := tunnelManager.StartTunnel(&tunnelConfig); err != nil {
				log.WithError(err).WithField("tunnel_name", tunnelConfig.Name).Error("Failed to start tunnel")
			}
		}
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Info("ShipIt client daemon started successfully")

	// Wait for shutdown signal
	<-sigChan
	log.Info("Received shutdown signal, stopping daemon")

	// Stop tunnel manager
	tunnelManager.Stop()

	log.Info("ShipIt client daemon stopped")
}

func runStop(cmd *cobra.Command, args []string) {
	// Load configuration
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Setup logging
	if err := logger.InitLogger(cfg.Logging.Level, cfg.Logging.Format, cfg.Logging.File); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup logging: %v\n", err)
		os.Exit(1)
	}
	log := logger.GetLogger()
	log.Info("Stopping ShipIt client daemon")

	// TODO: Implement daemon stop logic
	// This would typically involve sending a signal to a running daemon
	// or stopping a system service

	fmt.Println("Daemon stop command not yet implemented")
}

func runStatus(cmd *cobra.Command, args []string) {
	// Load configuration
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Setup logging
	if err := logger.InitLogger(cfg.Logging.Level, cfg.Logging.Format, cfg.Logging.File); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup logging: %v\n", err)
		os.Exit(1)
	}
	log := logger.GetLogger()

	// Create tunnel manager for status check
	tunnelManager := client.NewTunnelManager(cfg, log)

	// Get tunnel statistics
	stats := tunnelManager.GetStats()

	fmt.Println("ShipIt Client Daemon Status")
	fmt.Println("============================")
	fmt.Printf("Total Tunnels: %d\n", stats["total_tunnels"])

	if tunnels, ok := stats["tunnels"].(map[string]interface{}); ok {
		for tunnelID, tunnelInfo := range tunnels {
			if info, ok := tunnelInfo.(map[string]interface{}); ok {
				fmt.Printf("Tunnel %s: %s\n", tunnelID, info["state"])
			}
		}
	}
}

func runAuthTest(cmd *cobra.Command, args []string) {
	// Load configuration
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Setup logging
	if err := logger.InitLogger(cfg.Logging.Level, cfg.Logging.Format, cfg.Logging.File); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup logging: %v\n", err)
		os.Exit(1)
	}
	log := logger.GetLogger()

	// Create control plane client
	controlPlane := client.NewControlPlaneClient(cfg, log)

	// Test authentication
	ctx := context.Background()
	tokenInfo, err := controlPlane.ValidateToken(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Authentication failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Authentication successful!")
	fmt.Printf("User ID: %s\n", tokenInfo.UserID)
	fmt.Printf("Auth Type: %s\n", tokenInfo.AuthType)
}

func runTunnelsList(cmd *cobra.Command, args []string) {
	// Load configuration
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Setup logging
	if err := logger.InitLogger(cfg.Logging.Level, cfg.Logging.Format, cfg.Logging.File); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup logging: %v\n", err)
		os.Exit(1)
	}
	log := logger.GetLogger()

	// Create control plane client
	controlPlane := client.NewControlPlaneClient(cfg, log)

	// List tunnels
	ctx := context.Background()
	tunnels, err := controlPlane.ListTunnels(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list tunnels: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Tunnels")
	fmt.Println("=======")
	for _, tunnel := range tunnels {
		fmt.Printf("ID: %s\n", tunnel.ID)
		fmt.Printf("  Protocol: %s\n", tunnel.Protocol)
		fmt.Printf("  Public URL: %s\n", tunnel.PublicURL)
		fmt.Printf("  Status: %s\n", tunnel.Status)
		fmt.Printf("  Local Port: %d\n", tunnel.LocalPort)
		if tunnel.Subdomain != "" {
			fmt.Printf("  Subdomain: %s\n", tunnel.Subdomain)
		}
		fmt.Printf("  Created: %s\n", tunnel.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}
}

func runConfigShow(cmd *cobra.Command, args []string) {
	// Load configuration
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Display configuration
	fmt.Println("ShipIt Client Configuration")
	fmt.Println("===========================")
	fmt.Printf("Server Domain: %s\n", cfg.Server.Domain)
	fmt.Printf("API Port: %d\n", cfg.Server.APIPort)
	fmt.Printf("Data Plane Port: %d\n", cfg.Server.DataPlanePort)
	fmt.Printf("TLS Verify: %t\n", cfg.Server.TLSVerify)
	fmt.Printf("API Key: %s...\n", cfg.Auth.APIKey[:10])
	fmt.Printf("Auto Refresh: %t\n", cfg.Auth.AutoRefresh)
	fmt.Printf("Pool Size: %d\n", cfg.Connection.PoolSize)
	fmt.Printf("Heartbeat Interval: %s\n", cfg.Connection.HeartbeatInterval)
	fmt.Printf("Reconnect Interval: %s\n", cfg.Connection.ReconnectInterval)
	fmt.Printf("Max Reconnect Attempts: %d\n", cfg.Connection.MaxReconnectAttempts)
	fmt.Printf("Log Level: %s\n", cfg.Logging.Level)
	fmt.Printf("Log Format: %s\n", cfg.Logging.Format)
	if cfg.Logging.File != "" {
		fmt.Printf("Log File: %s\n", cfg.Logging.File)
	}

	fmt.Println("\nTunnels:")
	for i, tunnel := range cfg.Tunnels {
		fmt.Printf("  %d. %s (%s:%d)\n", i+1, tunnel.Name, tunnel.Protocol, tunnel.LocalPort)
		if tunnel.Subdomain != "" {
			fmt.Printf("     Subdomain: %s\n", tunnel.Subdomain)
		}
		fmt.Printf("     Auto Start: %t\n", tunnel.AutoStart)
	}
}

func runConfigValidate(cmd *cobra.Command, args []string) {
	// Validate configuration
	if err := config.ValidateConfigFile(cfgFile); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration validation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Configuration is valid!")
}

func runConfigInit(cmd *cobra.Command, args []string) {
	// Get config file path
	configPath := cfgFile
	if configPath == "" {
		configPath = config.GetConfigPath()
	}

	// Check if config file already exists
	if _, err := os.Stat(configPath); err == nil {
		fmt.Fprintf(os.Stderr, "Configuration file already exists: %s\n", configPath)
		fmt.Println("Use --config to specify a different location or remove the existing file.")
		os.Exit(1)
	}

	// Create default configuration
	if err := config.CreateDefaultConfig(configPath); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create configuration file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Configuration file created: %s\n", configPath)
	fmt.Println("Please edit the configuration file and set your API key.")
}

func runConfigEdit(cmd *cobra.Command, args []string) {
	// Get config file path
	configPath := cfgFile
	if configPath == "" {
		configPath = config.GetConfigPath()
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Configuration file does not exist: %s\n", configPath)
		fmt.Println("Run 'shipit-client config init' to create a configuration file.")
		os.Exit(1)
	}

	// Get editor from environment
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nano" // Default fallback
	}

	// Open file in editor
	fmt.Printf("Opening configuration file in %s: %s\n", editor, configPath)
	
	// Note: This is a simplified implementation
	// In a real implementation, you would use exec.Command to open the editor
	fmt.Println("Please manually edit the configuration file.")
	fmt.Printf("File location: %s\n", configPath)
} 