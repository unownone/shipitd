package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/unownone/shipitd/internal/config"
	"github.com/unownone/shipitd/internal/protocol"
	"github.com/unownone/shipitd/pkg/types"
	"github.com/sirupsen/logrus"
)

// DataPlaneClient handles TLS protocol communication with the ShipIt server
type DataPlaneClient struct {
	serverAddr string
	tlsConfig  *tls.Config
	logger     *logrus.Logger
	conn       net.Conn
	reader     *protocol.Reader
	writer     *protocol.Writer
	mu         sync.RWMutex
	connected  bool
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewDataPlaneClient creates a new data plane client
func NewDataPlaneClient(cfg *config.Config, logger *logrus.Logger) *DataPlaneClient {
	// Create TLS configuration
	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		ServerName:         cfg.Server.Domain,
		InsecureSkipVerify: !cfg.Server.TLSVerify,
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &DataPlaneClient{
		serverAddr: fmt.Sprintf("%s:%d", cfg.Server.Domain, cfg.Server.DataPlanePort),
		tlsConfig:  tlsConfig,
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Connect establishes a TLS connection to the server
func (d *DataPlaneClient) Connect() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.connected {
		return fmt.Errorf("already connected")
	}

	d.logger.WithField("server_addr", d.serverAddr).Info("Connecting to data plane")

	// Establish TCP connection
	conn, err := net.DialTimeout("tcp", d.serverAddr, 30*time.Second)
	if err != nil {
		return fmt.Errorf("failed to establish TCP connection: %w", err)
	}

	// Perform TLS handshake
	tlsConn := tls.Client(conn, d.tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		conn.Close()
		return fmt.Errorf("TLS handshake failed: %w", err)
	}

	d.conn = tlsConn
	d.reader = protocol.NewReader(tlsConn, d.logger)
	d.writer = protocol.NewWriter(tlsConn, d.logger)
	d.connected = true

	d.logger.WithFields(logrus.Fields{
		"server_addr": d.serverAddr,
		"tls_version": tlsConn.ConnectionState().Version,
		"cipher_suite": tlsConn.ConnectionState().CipherSuite,
	}).Info("Successfully connected to data plane")

	return nil
}

// Disconnect closes the connection to the server
func (d *DataPlaneClient) Disconnect() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.connected {
		return nil
	}

	d.logger.Info("Disconnecting from data plane")

	if d.conn != nil {
		if err := d.conn.Close(); err != nil {
			d.logger.WithError(err).Warn("Error closing connection")
		}
	}

	d.connected = false
	d.conn = nil
	d.reader = nil
	d.writer = nil

	return nil
}

// IsConnected returns whether the client is connected
func (d *DataPlaneClient) IsConnected() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.connected
}

// RegisterTunnel registers a tunnel with the server
func (d *DataPlaneClient) RegisterTunnel(tunnel *Tunnel) error {
	d.mu.RLock()
	if !d.connected {
		d.mu.RUnlock()
		return fmt.Errorf("not connected to server")
	}
	d.mu.RUnlock()

	payload := &types.TunnelRegistrationPayload{
		Protocol:      tunnel.Protocol,
		LocalPort:     tunnel.LocalPort,
		Subdomain:     tunnel.Subdomain,
		MaxConnections: 10, // Default value, could be configurable
	}

	d.logger.WithFields(logrus.Fields{
		"tunnel_id":  tunnel.ID,
		"protocol":   tunnel.Protocol,
		"local_port": tunnel.LocalPort,
		"subdomain":  tunnel.Subdomain,
	}).Info("Registering tunnel with server")

	return d.writer.WriteTunnelRegistration(tunnel.ID, payload)
}

// SendHeartbeat sends a heartbeat message to the server
func (d *DataPlaneClient) SendHeartbeat(tunnelID string, activeConns, totalRequests int) error {
	d.mu.RLock()
	if !d.connected {
		d.mu.RUnlock()
		return fmt.Errorf("not connected to server")
	}
	d.mu.RUnlock()

	payload := &types.HeartbeatPayload{
		Timestamp:     time.Now().Unix(),
		ActiveConns:   activeConns,
		TotalRequests: totalRequests,
	}

	d.logger.WithFields(logrus.Fields{
		"tunnel_id":     tunnelID,
		"active_conns":  activeConns,
		"total_requests": totalRequests,
	}).Debug("Sending heartbeat")

	return d.writer.WriteHeartbeat(tunnelID, payload)
}

// SendDataResponse sends a data response to the server
func (d *DataPlaneClient) SendDataResponse(tunnelID string, payload *types.DataResponsePayload) error {
	d.mu.RLock()
	if !d.connected {
		d.mu.RUnlock()
		return fmt.Errorf("not connected to server")
	}
	d.mu.RUnlock()

	d.logger.WithFields(logrus.Fields{
		"tunnel_id":     tunnelID,
		"connection_id": payload.ConnectionID,
		"request_id":    payload.RequestID,
		"status_code":   payload.StatusCode,
		"data_size":     len(payload.Data),
	}).Debug("Sending data response")

	return d.writer.WriteDataResponse(tunnelID, payload)
}

// SendError sends an error message to the server
func (d *DataPlaneClient) SendError(tunnelID string, code, message, details string) error {
	d.mu.RLock()
	if !d.connected {
		d.mu.RUnlock()
		return fmt.Errorf("not connected to server")
	}
	d.mu.RUnlock()

	payload := &types.ErrorPayload{
		Code:    code,
		Message: message,
		Details: details,
	}

	d.logger.WithFields(logrus.Fields{
		"tunnel_id": tunnelID,
		"code":      code,
		"message":   message,
		"details":   details,
	}).Warn("Sending error message")

	return d.writer.WriteError(tunnelID, payload)
}

// SendAcknowledge sends an acknowledgment message to the server
func (d *DataPlaneClient) SendAcknowledge(tunnelID, messageID, status string) error {
	d.mu.RLock()
	if !d.connected {
		d.mu.RUnlock()
		return fmt.Errorf("not connected to server")
	}
	d.mu.RUnlock()

	payload := &types.AcknowledgePayload{
		MessageID: messageID,
		Status:    status,
	}

	d.logger.WithFields(logrus.Fields{
		"tunnel_id":  tunnelID,
		"message_id": messageID,
		"status":     status,
	}).Debug("Sending acknowledgment")

	return d.writer.WriteAcknowledge(tunnelID, payload)
}

// SendConnectionClose sends a connection close message to the server
func (d *DataPlaneClient) SendConnectionClose(tunnelID, connectionID, reason string) error {
	d.mu.RLock()
	if !d.connected {
		d.mu.RUnlock()
		return fmt.Errorf("not connected to server")
	}
	d.mu.RUnlock()

	payload := &types.ConnectionClosePayload{
		ConnectionID: connectionID,
		Reason:       reason,
	}

	d.logger.WithFields(logrus.Fields{
		"tunnel_id":     tunnelID,
		"connection_id": connectionID,
		"reason":        reason,
	}).Debug("Sending connection close")

	return d.writer.WriteConnectionClose(tunnelID, payload)
}

// ReadMessage reads a message from the server
func (d *DataPlaneClient) ReadMessage() (*types.Message, error) {
	d.mu.RLock()
	if !d.connected {
		d.mu.RUnlock()
		return nil, fmt.Errorf("not connected to server")
	}
	d.mu.RUnlock()

	return d.reader.ReadMessage()
}

// ReadMessageWithTimeout reads a message with a timeout
func (d *DataPlaneClient) ReadMessageWithTimeout(timeout time.Duration) (*types.Message, error) {
	d.mu.RLock()
	if !d.connected {
		d.mu.RUnlock()
		return nil, fmt.Errorf("not connected to server")
	}
	d.mu.RUnlock()

	return d.reader.ReadMessageWithTimeout(timeout)
}

// ReadMessageAsync reads messages asynchronously
func (d *DataPlaneClient) ReadMessageAsync(messageChan chan<- *types.Message, errorChan chan<- error) {
	d.mu.RLock()
	if !d.connected {
		d.mu.RUnlock()
		errorChan <- fmt.Errorf("not connected to server")
		return
	}
	d.mu.RUnlock()

	d.reader.ReadMessageAsync(messageChan, errorChan)
}

// StartHeartbeat starts sending periodic heartbeat messages
func (d *DataPlaneClient) StartHeartbeat(tunnelID string, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-d.ctx.Done():
				d.logger.Info("Heartbeat stopped due to context cancellation")
				return
			case <-ticker.C:
				if err := d.SendHeartbeat(tunnelID, 0, 0); err != nil {
					d.logger.WithError(err).Error("Failed to send heartbeat")
				}
			}
		}
	}()
}

// Stop stops the data plane client
func (d *DataPlaneClient) Stop() {
	d.logger.Info("Stopping data plane client")
	d.cancel()
	d.Disconnect()
}

// GetConnection returns the underlying connection
func (d *DataPlaneClient) GetConnection() net.Conn {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.conn
}

// GetServerAddr returns the server address
func (d *DataPlaneClient) GetServerAddr() string {
	return d.serverAddr
} 