package proxy

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/unownone/shipitd/internal/client"
	"github.com/sirupsen/logrus"
)

// TCPProxy handles TCP connection forwarding from ShipIt server to local services
type TCPProxy struct {
	localPort int
	tunnel    *client.Tunnel
	logger    *logrus.Logger
	connections map[string]*TCPConnection
	mutex      sync.RWMutex
}

// TCPConnection represents a TCP connection between server and local service
type TCPConnection struct {
	ID           string
	ServerConn   net.Conn
	LocalConn    net.Conn
	CreatedAt    time.Time
	LastActivity time.Time
	Closed       bool
	mutex        sync.Mutex
}

// NewTCPProxy creates a new TCP proxy instance
func NewTCPProxy(localPort int, tunnel *client.Tunnel, logger *logrus.Logger) *TCPProxy {
	return &TCPProxy{
		localPort:    localPort,
		tunnel:       tunnel,
		logger:       logger,
		connections:  make(map[string]*TCPConnection),
	}
}

// HandleConnection processes a new TCP connection from the ShipIt server
func (tp *TCPProxy) HandleConnection(connectionID string, serverConn net.Conn) error {
	tp.logger.WithFields(logrus.Fields{
		"connection_id": connectionID,
		"local_port":    tp.localPort,
		"remote_addr":   serverConn.RemoteAddr(),
	}).Info("Handling new TCP connection")

	// Connect to local service
	localConn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", tp.localPort), 10*time.Second)
	if err != nil {
		tp.logger.WithError(err).Error("Failed to connect to local service")
		serverConn.Close()
		return fmt.Errorf("failed to connect to local service: %w", err)
	}

	// Create connection object
	conn := &TCPConnection{
		ID:           connectionID,
		ServerConn:   serverConn,
		LocalConn:    localConn,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		Closed:       false,
	}

	// Store connection
	tp.mutex.Lock()
	tp.connections[connectionID] = conn
	tp.mutex.Unlock()

	// Start bidirectional forwarding
	go tp.forwardData(conn, serverConn, localConn, "server->local")
	go tp.forwardData(conn, localConn, serverConn, "local->server")

	tp.logger.WithField("connection_id", connectionID).Debug("TCP connection established")
	return nil
}

// forwardData handles data forwarding in one direction
func (tp *TCPProxy) forwardData(conn *TCPConnection, src, dst net.Conn, direction string) {
	defer func() {
		conn.mutex.Lock()
		if !conn.Closed {
			conn.Closed = true
			src.Close()
			dst.Close()
		}
		conn.mutex.Unlock()

		// Remove from connections map
		tp.mutex.Lock()
		delete(tp.connections, conn.ID)
		tp.mutex.Unlock()

		tp.logger.WithFields(logrus.Fields{
			"connection_id": conn.ID,
			"direction":     direction,
		}).Debug("TCP connection closed")
	}()

	buffer := make([]byte, 4096)
	for {
		// Set read timeout
		src.SetReadDeadline(time.Now().Add(30 * time.Second))

		// Read data from source
		n, err := src.Read(buffer)
		if err != nil {
			if err == io.EOF {
				tp.logger.WithFields(logrus.Fields{
					"connection_id": conn.ID,
					"direction":     direction,
				}).Debug("Connection closed by peer")
			} else {
				tp.logger.WithError(err).WithFields(logrus.Fields{
					"connection_id": conn.ID,
					"direction":     direction,
				}).Debug("Read error")
			}
			return
		}

		if n == 0 {
			continue
		}

		// Update last activity
		conn.mutex.Lock()
		conn.LastActivity = time.Now()
		conn.mutex.Unlock()

		// Write data to destination
		_, err = dst.Write(buffer[:n])
		if err != nil {
			tp.logger.WithError(err).WithFields(logrus.Fields{
				"connection_id": conn.ID,
				"direction":     direction,
			}).Debug("Write error")
			return
		}

		tp.logger.WithFields(logrus.Fields{
			"connection_id": conn.ID,
			"direction":     direction,
			"bytes":         n,
		}).Trace("Data forwarded")
	}
}

// CloseConnection closes a specific TCP connection
func (tp *TCPProxy) CloseConnection(connectionID string) error {
	tp.mutex.RLock()
	conn, exists := tp.connections[connectionID]
	tp.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("connection %s not found", connectionID)
	}

	conn.mutex.Lock()
	if !conn.Closed {
		conn.Closed = true
		conn.ServerConn.Close()
		conn.LocalConn.Close()
	}
	conn.mutex.Unlock()

	tp.logger.WithField("connection_id", connectionID).Info("TCP connection closed")
	return nil
}

// CloseAllConnections closes all active TCP connections
func (tp *TCPProxy) CloseAllConnections() {
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	for connectionID, conn := range tp.connections {
		conn.mutex.Lock()
		if !conn.Closed {
			conn.Closed = true
			conn.ServerConn.Close()
			conn.LocalConn.Close()
		}
		conn.mutex.Unlock()

		tp.logger.WithField("connection_id", connectionID).Debug("Closed TCP connection")
	}

	tp.connections = make(map[string]*TCPConnection)
	tp.logger.Info("All TCP connections closed")
}

// GetConnectionStats returns statistics about active connections
func (tp *TCPProxy) GetConnectionStats() map[string]interface{} {
	tp.mutex.RLock()
	defer tp.mutex.RUnlock()

	stats := make(map[string]interface{})
	stats["total_connections"] = len(tp.connections)
	stats["active_connections"] = 0

	now := time.Now()
	for _, conn := range tp.connections {
		conn.mutex.Lock()
		if !conn.Closed {
			stats["active_connections"] = stats["active_connections"].(int) + 1
			stats[fmt.Sprintf("conn_%s_age", conn.ID)] = now.Sub(conn.CreatedAt).Seconds()
			stats[fmt.Sprintf("conn_%s_last_activity", conn.ID)] = now.Sub(conn.LastActivity).Seconds()
		}
		conn.mutex.Unlock()
	}

	return stats
}

// HealthCheck performs a health check on the local service
func (tp *TCPProxy) HealthCheck() error {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", tp.localPort), 5*time.Second)
	if err != nil {
		return fmt.Errorf("TCP health check failed: %w", err)
	}
	defer conn.Close()
	return nil
}

// GetLocalURL returns the local URL for this proxy
func (tp *TCPProxy) GetLocalURL() string {
	return fmt.Sprintf("tcp://localhost:%d", tp.localPort)
}

// GetTunnel returns the associated tunnel
func (tp *TCPProxy) GetTunnel() *client.Tunnel {
	return tp.tunnel
}

// CleanupInactiveConnections removes connections that have been inactive for too long
func (tp *TCPProxy) CleanupInactiveConnections(maxIdleTime time.Duration) {
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	now := time.Now()
	for connectionID, conn := range tp.connections {
		conn.mutex.Lock()
		if !conn.Closed && now.Sub(conn.LastActivity) > maxIdleTime {
			conn.Closed = true
			conn.ServerConn.Close()
			conn.LocalConn.Close()
			delete(tp.connections, connectionID)
			tp.logger.WithField("connection_id", connectionID).Debug("Cleaned up inactive TCP connection")
		}
		conn.mutex.Unlock()
	}
} 