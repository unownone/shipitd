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
	"github.com/sirupsen/logrus"
)

// Connection represents a TLS connection to the server
type Connection struct {
	ID         string
	Conn       net.Conn
	Reader     *protocol.Reader
	Writer     *protocol.Writer
	IsHealthy  bool
	LastUsed   time.Time
	CreatedAt  time.Time
	mu         sync.RWMutex
}

// ConnectionPool manages multiple TLS connections to the server
type ConnectionPool struct {
	serverAddr string
	tlsConfig  *tls.Config
	poolSize   int
	logger     *logrus.Logger
	connections []*Connection
	roundRobin int
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(cfg *config.Config, logger *logrus.Logger) *ConnectionPool {
	// Create TLS configuration
	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		ServerName:         cfg.Server.Domain,
		InsecureSkipVerify: !cfg.Server.TLSVerify,
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &ConnectionPool{
		serverAddr:  fmt.Sprintf("%s:%d", cfg.Server.Domain, cfg.Server.DataPlanePort),
		tlsConfig:   tlsConfig,
		poolSize:    cfg.Connection.PoolSize,
		logger:      logger,
		connections: make([]*Connection, 0, cfg.Connection.PoolSize),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Initialize creates the initial connection pool
func (cp *ConnectionPool) Initialize() error {
	cp.logger.WithFields(logrus.Fields{
		"server_addr": cp.serverAddr,
		"pool_size":   cp.poolSize,
	}).Info("Initializing connection pool")

	for i := 0; i < cp.poolSize; i++ {
		conn, err := cp.createConnection()
		if err != nil {
			cp.logger.WithError(err).Error("Failed to create connection")
			continue
		}
		cp.addConnection(conn)
	}

	// Start health monitoring
	go cp.monitorHealth()

	return nil
}

// createConnection creates a new TLS connection
func (cp *ConnectionPool) createConnection() (*Connection, error) {
	// Establish TCP connection
	conn, err := net.DialTimeout("tcp", cp.serverAddr, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to establish TCP connection: %w", err)
	}

	// Perform TLS handshake
	tlsConn := tls.Client(conn, cp.tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("TLS handshake failed: %w", err)
	}

	connection := &Connection{
		ID:        fmt.Sprintf("conn_%d", time.Now().UnixNano()),
		Conn:      tlsConn,
		Reader:    protocol.NewReader(tlsConn, cp.logger),
		Writer:    protocol.NewWriter(tlsConn, cp.logger),
		IsHealthy: true,
		LastUsed:  time.Now(),
		CreatedAt: time.Now(),
	}

	cp.logger.WithFields(logrus.Fields{
		"connection_id": connection.ID,
		"server_addr":   cp.serverAddr,
		"tls_version":   tlsConn.ConnectionState().Version,
	}).Debug("Created new connection")

	return connection, nil
}

// addConnection adds a connection to the pool
func (cp *ConnectionPool) addConnection(conn *Connection) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.connections = append(cp.connections, conn)
	cp.logger.WithField("connection_id", conn.ID).Debug("Added connection to pool")
}

// GetConnection returns a healthy connection from the pool
func (cp *ConnectionPool) GetConnection() (*Connection, error) {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	if len(cp.connections) == 0 {
		return nil, fmt.Errorf("no connections available in pool")
	}

	// Round-robin selection
	cp.roundRobin = (cp.roundRobin + 1) % len(cp.connections)
	conn := cp.connections[cp.roundRobin]

	conn.mu.Lock()
	defer conn.mu.Unlock()

	if !conn.IsHealthy {
		return nil, fmt.Errorf("connection %s is not healthy", conn.ID)
	}

	conn.LastUsed = time.Now()
	cp.logger.WithField("connection_id", conn.ID).Debug("Selected connection from pool")

	return conn, nil
}

// GetHealthyConnections returns all healthy connections
func (cp *ConnectionPool) GetHealthyConnections() []*Connection {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	var healthy []*Connection
	for _, conn := range cp.connections {
		conn.mu.RLock()
		if conn.IsHealthy {
			healthy = append(healthy, conn)
		}
		conn.mu.RUnlock()
	}

	return healthy
}

// MarkConnectionUnhealthy marks a connection as unhealthy
func (cp *ConnectionPool) MarkConnectionUnhealthy(connID string) {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	for _, conn := range cp.connections {
		if conn.ID == connID {
			conn.mu.Lock()
			conn.IsHealthy = false
			conn.mu.Unlock()
			cp.logger.WithField("connection_id", connID).Warn("Marked connection as unhealthy")
			break
		}
	}
}

// ReplaceConnection replaces an unhealthy connection with a new one
func (cp *ConnectionPool) ReplaceConnection(oldConnID string) error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	// Find and remove the old connection
	for i, conn := range cp.connections {
		if conn.ID == oldConnID {
			// Close the old connection
			if err := conn.Conn.Close(); err != nil {
				cp.logger.WithError(err).Warn("Error closing old connection")
			}

			// Create a new connection
			newConn, err := cp.createConnection()
			if err != nil {
				cp.logger.WithError(err).Error("Failed to create replacement connection")
				return err
			}

			// Replace the connection
			cp.connections[i] = newConn
			cp.logger.WithFields(logrus.Fields{
				"old_connection_id": oldConnID,
				"new_connection_id": newConn.ID,
			}).Info("Replaced unhealthy connection")

			return nil
		}
	}

	return fmt.Errorf("connection %s not found in pool", oldConnID)
}

// monitorHealth monitors the health of all connections
func (cp *ConnectionPool) monitorHealth() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cp.ctx.Done():
			cp.logger.Info("Health monitoring stopped")
			return
		case <-ticker.C:
			cp.checkConnectionsHealth()
		}
	}
}

// checkConnectionsHealth checks the health of all connections
func (cp *ConnectionPool) checkConnectionsHealth() {
	cp.mu.RLock()
	connections := make([]*Connection, len(cp.connections))
	copy(connections, cp.connections)
	cp.mu.RUnlock()

	for _, conn := range connections {
		conn.mu.RLock()
		if !conn.IsHealthy {
			conn.mu.RUnlock()
			continue
		}
		conn.mu.RUnlock()

		// Check if connection is still alive by trying to read from it
		conn.Conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		_, err := conn.Conn.Read(make([]byte, 1))
		conn.Conn.SetReadDeadline(time.Time{})

		if err != nil {
			cp.logger.WithFields(logrus.Fields{
				"connection_id": conn.ID,
				"error":         err,
			}).Warn("Connection health check failed")

			cp.MarkConnectionUnhealthy(conn.ID)
		}
	}
}

// GetStats returns connection pool statistics
func (cp *ConnectionPool) GetStats() map[string]interface{} {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	total := len(cp.connections)
	healthy := 0
	unhealthy := 0

	for _, conn := range cp.connections {
		conn.mu.RLock()
		if conn.IsHealthy {
			healthy++
		} else {
			unhealthy++
		}
		conn.mu.RUnlock()
	}

	return map[string]interface{}{
		"total_connections":     total,
		"healthy_connections":   healthy,
		"unhealthy_connections": unhealthy,
		"pool_size":            cp.poolSize,
		"round_robin_index":    cp.roundRobin,
	}
}

// Close closes all connections in the pool
func (cp *ConnectionPool) Close() error {
	cp.logger.Info("Closing connection pool")
	cp.cancel()

	cp.mu.Lock()
	defer cp.mu.Unlock()

	for _, conn := range cp.connections {
		if err := conn.Conn.Close(); err != nil {
			cp.logger.WithError(err).Warn("Error closing connection")
		}
	}

	cp.connections = cp.connections[:0]
	cp.logger.Info("Connection pool closed")

	return nil
}

// GetConnectionCount returns the number of connections in the pool
func (cp *ConnectionPool) GetConnectionCount() int {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return len(cp.connections)
}

// GetHealthyConnectionCount returns the number of healthy connections
func (cp *ConnectionPool) GetHealthyConnectionCount() int {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	healthy := 0
	for _, conn := range cp.connections {
		conn.mu.RLock()
		if conn.IsHealthy {
			healthy++
		}
		conn.mu.RUnlock()
	}

	return healthy
} 