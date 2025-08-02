package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kardianos/service"
	"github.com/unownone/shipitd/internal/client"
	"github.com/unownone/shipitd/internal/config"
	"github.com/sirupsen/logrus"
)

// DaemonService represents the daemon service
type DaemonService struct {
	config       *config.Config
	logger       *logrus.Logger
	tunnelMgr    *client.TunnelManager
	ctx          context.Context
	cancel       context.CancelFunc
	service      service.Service
}

// NewDaemonService creates a new daemon service
func NewDaemonService(cfg *config.Config, logger *logrus.Logger) *DaemonService {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &DaemonService{
		config:    cfg,
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start starts the daemon service
func (ds *DaemonService) Start(s service.Service) error {
	ds.logger.Info("Starting ShipIt client daemon")
	
	// Initialize tunnel manager
	ds.tunnelMgr = client.NewTunnelManager(ds.config, ds.logger)

	// Start configured tunnels
	for _, tunnelConfig := range ds.config.Tunnels {
		if tunnelConfig.AutoStart {
			if err := ds.tunnelMgr.StartTunnel(&tunnelConfig); err != nil {
				ds.logger.WithError(err).WithField("tunnel", tunnelConfig.Name).Error("Failed to start tunnel")
			}
		}
	}

	// Start health monitoring
	go ds.healthMonitor()

	// Start signal handling
	go ds.handleSignals()

	ds.logger.Info("ShipIt client daemon started successfully")
	return nil
}

// Stop stops the daemon service
func (ds *DaemonService) Stop(s service.Service) error {
	ds.logger.Info("Stopping ShipIt client daemon")
	
	// Cancel context to stop all goroutines
	ds.cancel()

	// Stop tunnel manager
	if ds.tunnelMgr != nil {
		ds.tunnelMgr.Stop()
	}

	ds.logger.Info("ShipIt client daemon stopped")
	return nil
}

// healthMonitor monitors the health of the daemon
func (ds *DaemonService) healthMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ds.ctx.Done():
			return
		case <-ticker.C:
			ds.checkHealth()
		}
	}
}

// checkHealth performs health checks
func (ds *DaemonService) checkHealth() {
	if ds.tunnelMgr == nil {
		ds.logger.Error("Tunnel manager is nil")
		return
	}

	// Check tunnel manager health
	stats := ds.tunnelMgr.GetStats()
	tunnels := ds.tunnelMgr.ListTunnels()
	
	activeTunnels := 0
	for _, tunnel := range tunnels {
		if tunnel.State == client.TunnelStateActive {
			activeTunnels++
		}
	}
	
	ds.logger.WithFields(logrus.Fields{
		"active_tunnels": activeTunnels,
		"total_tunnels":  len(tunnels),
		"stats":          stats,
	}).Debug("Health check completed")

	// Log warnings for unhealthy states
	if activeTunnels == 0 && len(tunnels) > 0 {
		ds.logger.Warn("No active tunnels")
	}
}

// handleSignals handles system signals
func (ds *DaemonService) handleSignals() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-ds.ctx.Done():
			return
		case sig := <-sigChan:
			ds.logger.WithField("signal", sig.String()).Info("Received signal")
			ds.cancel()
			return
		}
	}
}

// RunDaemon runs the daemon in the foreground
func RunDaemon(cfg *config.Config, logger *logrus.Logger) error {
	daemon := NewDaemonService(cfg, logger)
	
	// Create service configuration
	svcConfig := &service.Config{
		Name:        "shipit-client",
		DisplayName: "ShipIt Client Daemon",
		Description: "ShipIt client daemon for tunnel management",
	}

	// Create service
	svc, err := service.New(daemon, svcConfig)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	daemon.service = svc

	// Run the service
	if err := svc.Run(); err != nil {
		return fmt.Errorf("service run failed: %w", err)
	}

	return nil
}

// InstallService installs the daemon as a system service
func InstallService(cfg *config.Config, logger *logrus.Logger) error {
	daemon := NewDaemonService(cfg, logger)
	
	svcConfig := &service.Config{
		Name:        "shipit-client",
		DisplayName: "ShipIt Client Daemon",
		Description: "ShipIt client daemon for tunnel management",
	}

	svc, err := service.New(daemon, svcConfig)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	if err := svc.Install(); err != nil {
		return fmt.Errorf("failed to install service: %w", err)
	}

	logger.Info("Service installed successfully")
	return nil
}

// UninstallService uninstalls the daemon service
func UninstallService(cfg *config.Config, logger *logrus.Logger) error {
	daemon := NewDaemonService(cfg, logger)
	
	svcConfig := &service.Config{
		Name:        "shipit-client",
		DisplayName: "ShipIt Client Daemon",
		Description: "ShipIt client daemon for tunnel management",
	}

	svc, err := service.New(daemon, svcConfig)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	if err := svc.Uninstall(); err != nil {
		return fmt.Errorf("failed to uninstall service: %w", err)
	}

	logger.Info("Service uninstalled successfully")
	return nil
}

// StartService starts the daemon service
func StartService(cfg *config.Config, logger *logrus.Logger) error {
	daemon := NewDaemonService(cfg, logger)
	
	svcConfig := &service.Config{
		Name:        "shipit-client",
		DisplayName: "ShipIt Client Daemon",
		Description: "ShipIt client daemon for tunnel management",
	}

	svc, err := service.New(daemon, svcConfig)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	if err := svc.Start(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	logger.Info("Service started successfully")
	return nil
}

// StopService stops the daemon service
func StopService(cfg *config.Config, logger *logrus.Logger) error {
	daemon := NewDaemonService(cfg, logger)
	
	svcConfig := &service.Config{
		Name:        "shipit-client",
		DisplayName: "ShipIt Client Daemon",
		Description: "ShipIt client daemon for tunnel management",
	}

	svc, err := service.New(daemon, svcConfig)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	if err := svc.Stop(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	logger.Info("Service stopped successfully")
	return nil
}

// GetServiceStatus gets the status of the daemon service
func GetServiceStatus(cfg *config.Config, logger *logrus.Logger) (service.Status, error) {
	daemon := NewDaemonService(cfg, logger)
	
	svcConfig := &service.Config{
		Name:        "shipit-client",
		DisplayName: "ShipIt Client Daemon",
		Description: "ShipIt client daemon for tunnel management",
	}

	svc, err := service.New(daemon, svcConfig)
	if err != nil {
		return service.StatusUnknown, fmt.Errorf("failed to create service: %w", err)
	}

	status, err := svc.Status()
	if err != nil {
		return service.StatusUnknown, fmt.Errorf("failed to get service status: %w", err)
	}

	return status, nil
} 