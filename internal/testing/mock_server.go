package testing

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/unownone/shipitd/pkg/types"
)

// MockShipItServer represents a mock ShipIt server for testing
type MockShipItServer struct {
	server       *httptest.Server
	tlsServer    *httptest.Server
	tunnels      map[string]*types.Tunnel
	apiKeys      map[string]string
	mu           sync.RWMutex
	authCalls    int
	tunnelCalls  int
	tunnelCounter int
}

// MockServerConfig holds configuration for the mock server
type MockServerConfig struct {
	Port           int
	TLSPort        int
	ValidAPIKeys   []string
	DefaultTunnels []types.Tunnel
}

// NewMockShipItServer creates a new mock ShipIt server
func NewMockShipItServer(config *MockServerConfig) *MockShipItServer {
	mock := &MockShipItServer{
		tunnels: make(map[string]*types.Tunnel),
		apiKeys: make(map[string]string),
	}

	// Initialize API keys
	for _, key := range config.ValidAPIKeys {
		mock.apiKeys[key] = "test-user-id"
	}

	// Initialize default tunnels
	for _, tunnel := range config.DefaultTunnels {
		mock.tunnels[tunnel.ID] = &tunnel
	}

	// Create HTTP server for control plane
	mock.server = httptest.NewServer(http.HandlerFunc(mock.handleHTTP))

	// Create TLS server for data plane
	mock.tlsServer = httptest.NewTLSServer(http.HandlerFunc(mock.handleTLS))

	return mock
}

// URL returns the HTTP server URL
func (m *MockShipItServer) URL() string {
	return m.server.URL
}

// TLSURL returns the TLS server URL
func (m *MockShipItServer) TLSURL() string {
	return m.tlsServer.URL
}

// Close shuts down the mock server
func (m *MockShipItServer) Close() {
	m.server.Close()
	m.tlsServer.Close()
}

// Reset clears the mock server state
func (m *MockShipItServer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tunnels = make(map[string]*types.Tunnel)
	m.authCalls = 0
	m.tunnelCalls = 0
	m.tunnelCounter = 0
}

// GetAuthCalls returns the number of authentication calls
func (m *MockShipItServer) GetAuthCalls() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.authCalls
}

// GetTunnelCalls returns the number of tunnel management calls
func (m *MockShipItServer) GetTunnelCalls() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tunnelCalls
}

// AddTunnel adds a tunnel to the mock server
func (m *MockShipItServer) AddTunnel(tunnel *types.Tunnel) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tunnels[tunnel.ID] = tunnel
}

// RemoveTunnel removes a tunnel from the mock server
func (m *MockShipItServer) RemoveTunnel(tunnelID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tunnels, tunnelID)
}

// handleHTTP handles HTTP requests for the control plane
func (m *MockShipItServer) handleHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.URL.Path {
	case "/api/v1/auth/validate":
		m.handleAuthValidate(w, r)
	case "/api/v1/tunnels":
		m.handleTunnels(w, r)
	default:
		// Handle path parameters for tunnel operations
		if strings.HasPrefix(r.URL.Path, "/api/v1/tunnels/") {
			m.handleTunnelWithID(w, r)
		} else {
			http.NotFound(w, r)
		}
	}
}

// handleTLS handles TLS requests for the data plane
func (m *MockShipItServer) handleTLS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Simulate data plane protocol
	switch r.URL.Path {
	case "/data":
		m.handleDataPlane(w, r)
	default:
		http.NotFound(w, r)
	}
}

// handleAuthValidate handles authentication validation
func (m *MockShipItServer) handleAuthValidate(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	m.authCalls++
	m.mu.Unlock()

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, `{"error": "Missing authorization header"}`, http.StatusUnauthorized)
		return
	}

	// Extract API key from Bearer token
	apiKey := authHeader
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		apiKey = authHeader[7:]
	}

	m.mu.RLock()
	userID, exists := m.apiKeys[apiKey]
	m.mu.RUnlock()

	if !exists {
		http.Error(w, `{"error": "Invalid API key"}`, http.StatusUnauthorized)
		return
	}

	response := map[string]interface{}{
		"valid":     true,
		"user_id":   userID,
		"auth_type": "api_key",
	}

	json.NewEncoder(w).Encode(response)
}

// handleTunnels handles tunnel management
func (m *MockShipItServer) handleTunnels(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	m.tunnelCalls++
	m.mu.Unlock()

	switch r.Method {
	case "GET":
		m.handleListTunnels(w, r)
	case "POST":
		m.handleCreateTunnel(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleListTunnels handles listing tunnels
func (m *MockShipItServer) handleListTunnels(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	tunnels := make([]map[string]interface{}, 0, len(m.tunnels))
	for _, tunnel := range m.tunnels {
		// Convert to the format expected by the client
		tunnelResponse := map[string]interface{}{
			"tunnel_id":  tunnel.ID,
			"protocol":   tunnel.Protocol,
			"public_url": tunnel.PublicURL,
			"status":     tunnel.Status,
			"subdomain":  tunnel.Name,
			"local_port": 3000, // Default port
			"created_at": tunnel.CreatedAt,
			"updated_at": tunnel.UpdatedAt,
		}
		tunnels = append(tunnels, tunnelResponse)
	}
	m.mu.RUnlock()

	response := map[string]interface{}{
		"tunnels": tunnels,
		"count":   len(tunnels),
	}

	json.NewEncoder(w).Encode(response)
}

// handleCreateTunnel handles tunnel creation
func (m *MockShipItServer) handleCreateTunnel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Protocol   string `json:"protocol"`
		LocalPort  int    `json:"local_port"`
		Subdomain  string `json:"subdomain,omitempty"`
		PublicPort *int   `json:"public_port,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	m.mu.Lock()
	m.tunnelCounter++
	tunnelID := fmt.Sprintf("tunnel-%d-%d", time.Now().Unix(), m.tunnelCounter)
	m.mu.Unlock()

	tunnel := &types.Tunnel{
		ID:          tunnelID,
		Name:        req.Subdomain,
		LocalURL:    fmt.Sprintf("http://localhost:%d", req.LocalPort),
		Protocol:    req.Protocol,
		Description: fmt.Sprintf("Tunnel for %s", req.Subdomain),
		PublicURL:   fmt.Sprintf("https://%s.shipit.dev", req.Subdomain),
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	m.mu.Lock()
	m.tunnels[tunnelID] = tunnel
	m.mu.Unlock()

	// Return the tunnel in the format expected by the client
	response := map[string]interface{}{
		"tunnel_id":  tunnel.ID,
		"protocol":   tunnel.Protocol,
		"public_url": tunnel.PublicURL,
		"status":     tunnel.Status,
		"subdomain":  req.Subdomain,
		"local_port": req.LocalPort,
		"created_at": tunnel.CreatedAt,
		"updated_at": tunnel.UpdatedAt,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleDeleteTunnel handles tunnel deletion
func (m *MockShipItServer) handleDeleteTunnel(w http.ResponseWriter, r *http.Request) {
	tunnelID := r.URL.Query().Get("id")
	if tunnelID == "" {
		http.Error(w, `{"error": "Missing tunnel ID"}`, http.StatusBadRequest)
		return
	}

	m.mu.Lock()
	delete(m.tunnels, tunnelID)
	m.mu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

// handleTunnelWithID handles tunnel operations with ID in path
func (m *MockShipItServer) handleTunnelWithID(w http.ResponseWriter, r *http.Request) {
	// Extract tunnel ID from path
	pathParts := strings.Split(r.URL.Path, "/")
	
	if len(pathParts) < 5 {
		http.NotFound(w, r)
		return
	}
	
	tunnelID := pathParts[4] // /api/v1/tunnels/{id} -> parts[4] is the ID
	
	switch r.Method {
	case "GET":
		m.handleGetTunnel(w, r, tunnelID)
	case "DELETE":
		m.handleDeleteTunnelByID(w, r, tunnelID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetTunnel handles getting a specific tunnel
func (m *MockShipItServer) handleGetTunnel(w http.ResponseWriter, r *http.Request, tunnelID string) {
	m.mu.RLock()
	tunnel, exists := m.tunnels[tunnelID]
	m.mu.RUnlock()
	
	if !exists {
		http.Error(w, `{"error": "Tunnel not found"}`, http.StatusNotFound)
		return
	}
	
	json.NewEncoder(w).Encode(tunnel)
}

// handleDeleteTunnelByID handles tunnel deletion by ID
func (m *MockShipItServer) handleDeleteTunnelByID(w http.ResponseWriter, r *http.Request, tunnelID string) {
	m.mu.Lock()
	_, exists := m.tunnels[tunnelID]
	if !exists {
		m.mu.Unlock()
		http.Error(w, `{"error": "Tunnel not found"}`, http.StatusNotFound)
		return
	}
	
	delete(m.tunnels, tunnelID)
	m.mu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

// handleDataPlane handles data plane communication
func (m *MockShipItServer) handleDataPlane(w http.ResponseWriter, r *http.Request) {
	// Simulate data plane protocol
	response := map[string]interface{}{
		"status": "connected",
		"time":   time.Now().Unix(),
	}

	json.NewEncoder(w).Encode(response)
}

// MockTLSServer creates a TLS server for testing
func MockTLSServer() (*httptest.Server, *tls.Config) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "connected"}`))
	}))

	return server, &tls.Config{
		InsecureSkipVerify: true,
	}
}

// MockHTTPServer creates an HTTP server for testing
func MockHTTPServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "ok"}`))
	}))
}

// MockTCPServer creates a TCP server for testing
func MockTCPServer() (*httptest.Server, error) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("TCP server response"))
	}))

	return server, nil
} 