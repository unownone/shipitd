package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/unownone/shipitd/internal/config"
	"github.com/unownone/shipitd/pkg/types"
	"github.com/sirupsen/logrus"
)

// TunnelState represents the state of a tunnel
type TunnelState string

const (
	TunnelStateInitializing TunnelState = "initializing"
	TunnelStateCreating     TunnelState = "creating"
	TunnelStateConnecting   TunnelState = "connecting"
	TunnelStateRegistering  TunnelState = "registering"
	TunnelStateActive       TunnelState = "active"
	TunnelStateError        TunnelState = "error"
	TunnelStateDisconnected TunnelState = "disconnected"
)

// TunnelInfo represents detailed tunnel information
type TunnelInfo struct {
	Tunnel     *Tunnel
	State      TunnelState
	Error      error
	CreatedAt  time.Time
	UpdatedAt  time.Time
	mu         sync.RWMutex
}

// TunnelManager orchestrates tunnel lifecycle and coordinates between control and data planes
type TunnelManager struct {
	controlPlane *ControlPlaneClient
	dataPlane    *DataPlaneClient
	connectionPool *ConnectionPool
	config       *config.Config
	logger       *logrus.Logger
	tunnels      map[string]*TunnelInfo
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewTunnelManager creates a new tunnel manager
func NewTunnelManager(cfg *config.Config, logger *logrus.Logger) *TunnelManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &TunnelManager{
		controlPlane:   NewControlPlaneClient(cfg, logger),
		dataPlane:      NewDataPlaneClient(cfg, logger),
		connectionPool: NewConnectionPool(cfg, logger),
		config:         cfg,
		logger:         logger,
		tunnels:        make(map[string]*TunnelInfo),
		ctx:            ctx,
		cancel:         cancel,
	}
}

// StartTunnel starts a tunnel with the given configuration
func (tm *TunnelManager) StartTunnel(tunnelConfig *config.TunnelConfig) error {
	tm.logger.WithFields(logrus.Fields{
		"name":       tunnelConfig.Name,
		"protocol":   tunnelConfig.Protocol,
		"local_port": tunnelConfig.LocalPort,
		"subdomain":  tunnelConfig.Subdomain,
	}).Info("Starting tunnel")

	// Create tunnel info
	tunnelInfo := &TunnelInfo{
		State:     TunnelStateInitializing,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Start tunnel in background
	go tm.runTunnel(tunnelConfig, tunnelInfo)

	return nil
}

// runTunnel runs the tunnel lifecycle
func (tm *TunnelManager) runTunnel(tunnelConfig *config.TunnelConfig, tunnelInfo *TunnelInfo) {
	defer func() {
		if r := recover(); r != nil {
			tm.logger.WithField("panic", r).Error("Tunnel panic recovered")
			tm.updateTunnelState(tunnelInfo, TunnelStateError, fmt.Errorf("panic: %v", r))
		}
	}()

	// Step 1: Create tunnel via control plane
	tm.updateTunnelState(tunnelInfo, TunnelStateCreating, nil)
	
	tunnel, err := tm.createTunnel(tunnelConfig)
	if err != nil {
		tm.updateTunnelState(tunnelInfo, TunnelStateError, err)
		return
	}

	tunnelInfo.Tunnel = tunnel
	tm.addTunnel(tunnel.ID, tunnelInfo)

	// Step 2: Connect to data plane
	tm.updateTunnelState(tunnelInfo, TunnelStateConnecting, nil)
	
	if err := tm.dataPlane.Connect(); err != nil {
		tm.updateTunnelState(tunnelInfo, TunnelStateError, err)
		return
	}

	// Step 3: Register tunnel with data plane
	tm.updateTunnelState(tunnelInfo, TunnelStateRegistering, nil)
	
	if err := tm.dataPlane.RegisterTunnel(tunnel); err != nil {
		tm.updateTunnelState(tunnelInfo, TunnelStateError, err)
		return
	}

	// Step 4: Start heartbeat
	tm.dataPlane.StartHeartbeat(tunnel.ID, tm.config.Connection.HeartbeatInterval)

	// Step 5: Mark as active and start message processing
	tm.updateTunnelState(tunnelInfo, TunnelStateActive, nil)
	
	tm.processMessages(tunnel.ID)
}

// createTunnel creates a tunnel via the control plane
func (tm *TunnelManager) createTunnel(tunnelConfig *config.TunnelConfig) (*Tunnel, error) {
	req := &CreateTunnelRequest{
		Protocol:  tunnelConfig.Protocol,
		LocalPort: tunnelConfig.LocalPort,
		Subdomain: tunnelConfig.Subdomain,
	}

	ctx, cancel := context.WithTimeout(tm.ctx, 30*time.Second)
	defer cancel()

	return tm.controlPlane.CreateTunnel(ctx, req)
}

// processMessages processes incoming messages for a tunnel
func (tm *TunnelManager) processMessages(tunnelID string) {
	messageChan := make(chan *types.Message, 100)
	errorChan := make(chan error, 10)

	// Start reading messages asynchronously
	go tm.dataPlane.ReadMessageAsync(messageChan, errorChan)

	for {
		select {
		case <-tm.ctx.Done():
			tm.logger.Info("Message processing stopped due to context cancellation")
			return
		case message := <-messageChan:
			tm.handleMessage(tunnelID, message)
		case err := <-errorChan:
			tm.logger.WithError(err).Error("Error reading message")
			tm.handleConnectionError(tunnelID, err)
			return
		}
	}
}

// handleMessage handles an incoming message
func (tm *TunnelManager) handleMessage(tunnelID string, message *types.Message) {
	tm.logger.WithFields(logrus.Fields{
		"tunnel_id":   tunnelID,
		"message_type": message.Type,
		"payload_size": len(message.Payload),
	}).Debug("Handling message")

	switch message.Type {
	case types.MessageTypeDataForward:
		tm.handleDataForward(tunnelID, message)
	case types.MessageTypeAcknowledge:
		tm.handleAcknowledge(tunnelID, message)
	case types.MessageTypeError:
		tm.handleError(tunnelID, message)
	case types.MessageTypeHeartbeat:
		tm.handleHeartbeat(tunnelID, message)
	default:
		tm.logger.WithField("message_type", message.Type).Warn("Unknown message type")
	}
}

// handleDataForward handles a data forward message
func (tm *TunnelManager) handleDataForward(tunnelID string, message *types.Message) {
	payload, err := message.ParsePayload()
	if err != nil {
		tm.logger.WithError(err).Error("Failed to parse data forward payload")
		return
	}

	dataForward, ok := payload.(*types.DataForwardPayload)
	if !ok {
		tm.logger.Error("Invalid data forward payload type")
		return
	}

	tm.logger.WithFields(logrus.Fields{
		"tunnel_id":     tunnelID,
		"connection_id": dataForward.ConnectionID,
		"request_id":    dataForward.RequestID,
		"method":        dataForward.Method,
		"path":          dataForward.Path,
	}).Debug("Handling data forward")

	// TODO: Forward to local service and send response back
	// This will be implemented when we add the proxy components
}

// handleAcknowledge handles an acknowledgment message
func (tm *TunnelManager) handleAcknowledge(tunnelID string, message *types.Message) {
	payload, err := message.ParsePayload()
	if err != nil {
		tm.logger.WithError(err).Error("Failed to parse acknowledge payload")
		return
	}

	ack, ok := payload.(*types.AcknowledgePayload)
	if !ok {
		tm.logger.Error("Invalid acknowledge payload type")
		return
	}

	tm.logger.WithFields(logrus.Fields{
		"tunnel_id":  tunnelID,
		"message_id": ack.MessageID,
		"status":     ack.Status,
	}).Debug("Received acknowledgment")
}

// handleError handles an error message
func (tm *TunnelManager) handleError(tunnelID string, message *types.Message) {
	payload, err := message.ParsePayload()
	if err != nil {
		tm.logger.WithError(err).Error("Failed to parse error payload")
		return
	}

	errPayload, ok := payload.(*types.ErrorPayload)
	if !ok {
		tm.logger.Error("Invalid error payload type")
		return
	}

	tm.logger.WithFields(logrus.Fields{
		"tunnel_id": tunnelID,
		"code":      errPayload.Code,
		"message":   errPayload.Message,
		"details":   errPayload.Details,
	}).Error("Received error from server")
}

// handleHeartbeat handles a heartbeat message
func (tm *TunnelManager) handleHeartbeat(tunnelID string, message *types.Message) {
	payload, err := message.ParsePayload()
	if err != nil {
		tm.logger.WithError(err).Error("Failed to parse heartbeat payload")
		return
	}

	heartbeat, ok := payload.(*types.HeartbeatPayload)
	if !ok {
		tm.logger.Error("Invalid heartbeat payload type")
		return
	}

	tm.logger.WithFields(logrus.Fields{
		"tunnel_id":     tunnelID,
		"timestamp":     heartbeat.Timestamp,
		"active_conns":  heartbeat.ActiveConns,
		"total_requests": heartbeat.TotalRequests,
	}).Debug("Received heartbeat")
}

// handleConnectionError handles connection errors
func (tm *TunnelManager) handleConnectionError(tunnelID string, err error) {
	tm.logger.WithFields(logrus.Fields{
		"tunnel_id": tunnelID,
		"error":     err,
	}).Error("Connection error occurred")

	// Mark tunnel as disconnected
	tm.updateTunnelStateByID(tunnelID, TunnelStateDisconnected, err)

	// Attempt reconnection
	go tm.reconnectTunnel(tunnelID)
}

// reconnectTunnel attempts to reconnect a tunnel
func (tm *TunnelManager) reconnectTunnel(tunnelID string) {
	tm.logger.WithField("tunnel_id", tunnelID).Info("Attempting to reconnect tunnel")

	// Get tunnel info
	tunnelInfo := tm.getTunnel(tunnelID)
	if tunnelInfo == nil {
		tm.logger.WithField("tunnel_id", tunnelID).Error("Tunnel not found for reconnection")
		return
	}

	// Disconnect current connection
	tm.dataPlane.Disconnect()

	// Wait before reconnecting
	time.Sleep(tm.config.Connection.ReconnectInterval)

	// Attempt to reconnect
	if err := tm.dataPlane.Connect(); err != nil {
		tm.logger.WithError(err).Error("Failed to reconnect")
		tm.updateTunnelState(tunnelInfo, TunnelStateError, err)
		return
	}

	// Re-register tunnel
	if err := tm.dataPlane.RegisterTunnel(tunnelInfo.Tunnel); err != nil {
		tm.logger.WithError(err).Error("Failed to re-register tunnel")
		tm.updateTunnelState(tunnelInfo, TunnelStateError, err)
		return
	}

	tm.updateTunnelState(tunnelInfo, TunnelStateActive, nil)
	tm.logger.WithField("tunnel_id", tunnelID).Info("Tunnel reconnected successfully")
}

// updateTunnelState updates the state of a tunnel
func (tm *TunnelManager) updateTunnelState(tunnelInfo *TunnelInfo, state TunnelState, err error) {
	tunnelInfo.mu.Lock()
	defer tunnelInfo.mu.Unlock()

	tunnelInfo.State = state
	tunnelInfo.Error = err
	tunnelInfo.UpdatedAt = time.Now()

	tm.logger.WithFields(logrus.Fields{
		"tunnel_id": tunnelInfo.Tunnel.ID,
		"state":     state,
		"error":     err,
	}).Info("Tunnel state updated")
}

// updateTunnelStateByID updates the state of a tunnel by ID
func (tm *TunnelManager) updateTunnelStateByID(tunnelID string, state TunnelState, err error) {
	tunnelInfo := tm.getTunnel(tunnelID)
	if tunnelInfo != nil {
		tm.updateTunnelState(tunnelInfo, state, err)
	}
}

// addTunnel adds a tunnel to the manager
func (tm *TunnelManager) addTunnel(tunnelID string, tunnelInfo *TunnelInfo) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.tunnels[tunnelID] = tunnelInfo
}

// getTunnel gets a tunnel by ID
func (tm *TunnelManager) getTunnel(tunnelID string) *TunnelInfo {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.tunnels[tunnelID]
}

// ListTunnels returns all tunnels
func (tm *TunnelManager) ListTunnels() []*TunnelInfo {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tunnels := make([]*TunnelInfo, 0, len(tm.tunnels))
	for _, tunnel := range tm.tunnels {
		tunnels = append(tunnels, tunnel)
	}

	return tunnels
}

// StopTunnel stops a tunnel
func (tm *TunnelManager) StopTunnel(tunnelID string) error {
	tm.logger.WithField("tunnel_id", tunnelID).Info("Stopping tunnel")

	tunnelInfo := tm.getTunnel(tunnelID)
	if tunnelInfo == nil {
		return fmt.Errorf("tunnel %s not found", tunnelID)
	}

	// Delete tunnel via control plane
	ctx, cancel := context.WithTimeout(tm.ctx, 30*time.Second)
	defer cancel()

	if err := tm.controlPlane.DeleteTunnel(ctx, tunnelID); err != nil {
		tm.logger.WithError(err).Error("Failed to delete tunnel via control plane")
	}

	// Update state
	tm.updateTunnelState(tunnelInfo, TunnelStateDisconnected, nil)

	// Remove from manager
	tm.mu.Lock()
	delete(tm.tunnels, tunnelID)
	tm.mu.Unlock()

	return nil
}

// Stop stops the tunnel manager
func (tm *TunnelManager) Stop() {
	tm.logger.Info("Stopping tunnel manager")
	tm.cancel()

	// Stop all tunnels
	for tunnelID := range tm.tunnels {
		tm.StopTunnel(tunnelID)
	}

	// Stop data plane
	tm.dataPlane.Stop()

	// Close connection pool
	tm.connectionPool.Close()

	tm.logger.Info("Tunnel manager stopped")
}

// GetStats returns tunnel manager statistics
func (tm *TunnelManager) GetStats() map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	stats := map[string]interface{}{
		"total_tunnels": len(tm.tunnels),
		"tunnels":       make(map[string]interface{}),
	}

	for tunnelID, tunnelInfo := range tm.tunnels {
		tunnelInfo.mu.RLock()
		stats["tunnels"].(map[string]interface{})[tunnelID] = map[string]interface{}{
			"state":     tunnelInfo.State,
			"error":     tunnelInfo.Error,
			"created_at": tunnelInfo.CreatedAt,
			"updated_at": tunnelInfo.UpdatedAt,
		}
		tunnelInfo.mu.RUnlock()
	}

	return stats
} 