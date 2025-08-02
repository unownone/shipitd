package proxy

import (
	"testing"
	"time"

	"github.com/unownone/shipitd/internal/client"
	"github.com/sirupsen/logrus"
)

func TestNewTCPProxy(t *testing.T) {
	logger := logrus.New()
	tunnel := &client.Tunnel{
		ID:       "test-tunnel",
		Protocol: "tcp",
		LocalPort: 5432,
	}

	proxy := NewTCPProxy(5432, tunnel, logger)
	
	if proxy == nil {
		t.Fatal("Expected proxy to be created")
	}
	
	if proxy.localPort != 5432 {
		t.Errorf("Expected local port 5432, got %d", proxy.localPort)
	}
	
	if proxy.tunnel.ID != "test-tunnel" {
		t.Errorf("Expected tunnel ID 'test-tunnel', got %s", proxy.tunnel.ID)
	}
	
	if len(proxy.connections) != 0 {
		t.Errorf("Expected empty connections map, got %d connections", len(proxy.connections))
	}
}

func TestTCPProxyGetLocalURL(t *testing.T) {
	logger := logrus.New()
	tunnel := &client.Tunnel{
		ID:       "test-tunnel",
		Protocol: "tcp",
		LocalPort: 5432,
	}

	proxy := NewTCPProxy(5432, tunnel, logger)

	url := proxy.GetLocalURL()
	expected := "tcp://localhost:5432"

	if url != expected {
		t.Errorf("Expected URL %s, got %s", expected, url)
	}
}

func TestTCPProxyGetTunnel(t *testing.T) {
	logger := logrus.New()
	tunnel := &client.Tunnel{
		ID:       "test-tunnel",
		Protocol: "tcp",
		LocalPort: 5432,
	}

	proxy := NewTCPProxy(5432, tunnel, logger)

	retrievedTunnel := proxy.GetTunnel()
	if retrievedTunnel.ID != "test-tunnel" {
		t.Errorf("Expected tunnel ID 'test-tunnel', got %s", retrievedTunnel.ID)
	}
}

func TestTCPProxyGetConnectionStats(t *testing.T) {
	logger := logrus.New()
	tunnel := &client.Tunnel{
		ID:       "test-tunnel",
		Protocol: "tcp",
		LocalPort: 5432,
	}

	proxy := NewTCPProxy(5432, tunnel, logger)

	stats := proxy.GetConnectionStats()
	
	if stats["total_connections"] != 0 {
		t.Errorf("Expected 0 total connections, got %v", stats["total_connections"])
	}
	
	if stats["active_connections"] != 0 {
		t.Errorf("Expected 0 active connections, got %v", stats["active_connections"])
	}
}

func TestTCPProxyCloseAllConnections(t *testing.T) {
	logger := logrus.New()
	tunnel := &client.Tunnel{
		ID:       "test-tunnel",
		Protocol: "tcp",
		LocalPort: 5432,
	}

	proxy := NewTCPProxy(5432, tunnel, logger)

	// Close all connections (should not panic even with no connections)
	proxy.CloseAllConnections()

	stats := proxy.GetConnectionStats()
	if stats["total_connections"] != 0 {
		t.Errorf("Expected 0 total connections after close, got %v", stats["total_connections"])
	}
}

func TestTCPProxyCleanupInactiveConnections(t *testing.T) {
	logger := logrus.New()
	tunnel := &client.Tunnel{
		ID:       "test-tunnel",
		Protocol: "tcp",
		LocalPort: 5432,
	}

	proxy := NewTCPProxy(5432, tunnel, logger)

	// Cleanup inactive connections (should not panic even with no connections)
	proxy.CleanupInactiveConnections(5 * time.Minute)

	stats := proxy.GetConnectionStats()
	if stats["total_connections"] != 0 {
		t.Errorf("Expected 0 total connections after cleanup, got %v", stats["total_connections"])
	}
}

func TestTCPProxyCloseConnectionNotFound(t *testing.T) {
	logger := logrus.New()
	tunnel := &client.Tunnel{
		ID:       "test-tunnel",
		Protocol: "tcp",
		LocalPort: 5432,
	}

	proxy := NewTCPProxy(5432, tunnel, logger)

	// Try to close a non-existent connection
	err := proxy.CloseConnection("non-existent")
	if err == nil {
		t.Error("Expected error when closing non-existent connection")
	}
}

func TestTCPProxyHealthCheck(t *testing.T) {
	logger := logrus.New()
	tunnel := &client.Tunnel{
		ID:       "test-tunnel",
		Protocol: "tcp",
		LocalPort: 5432,
	}

	proxy := NewTCPProxy(5432, tunnel, logger)

	// Health check should fail since there's no service on port 5432
	err := proxy.HealthCheck()
	if err == nil {
		t.Error("Expected health check to fail when no service is running")
	}
} 