package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/unownone/shipitd/internal/client"
	"github.com/unownone/shipitd/pkg/types"
	"github.com/sirupsen/logrus"
)

func TestNewHTTPProxy(t *testing.T) {
	logger := logrus.New()
	tunnel := &client.Tunnel{
		ID:       "test-tunnel",
		Protocol: "http",
		LocalPort: 3000,
	}

	proxy := NewHTTPProxy(3000, tunnel, logger)
	
	if proxy == nil {
		t.Fatal("Expected proxy to be created")
	}
	
	if proxy.localPort != 3000 {
		t.Errorf("Expected local port 3000, got %d", proxy.localPort)
	}
	
	if proxy.tunnel.ID != "test-tunnel" {
		t.Errorf("Expected tunnel ID 'test-tunnel', got %s", proxy.tunnel.ID)
	}
}

func TestHTTPProxyHandleRequest(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "Hello, World!"}`))
	}))
	defer server.Close()

	logger := logrus.New()
	tunnel := &client.Tunnel{
		ID:       "test-tunnel",
		Protocol: "http",
		LocalPort: 3000,
	}

	proxy := NewHTTPProxy(3000, tunnel, logger)

	// Create a test request
	req := &types.DataForwardPayload{
		ConnectionID: "conn-123",
		RequestID:    "req-456",
		Method:       "GET",
		Path:         "/api/test",
		Headers: map[string]string{
			"Host":         "test.example.com",
			"User-Agent":   "test-agent",
			"Content-Type": "application/json",
		},
		Data: []byte(`{"test": "data"}`),
	}

	// Handle the request
	response, err := proxy.HandleRequest(req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if response == nil {
		t.Fatal("Expected response to be created")
	}

	if response.ConnectionID != "conn-123" {
		t.Errorf("Expected connection ID 'conn-123', got %s", response.ConnectionID)
	}

	if response.RequestID != "req-456" {
		t.Errorf("Expected request ID 'req-456', got %s", response.RequestID)
	}
}

func TestHTTPProxyCreateErrorResponse(t *testing.T) {
	logger := logrus.New()
	tunnel := &client.Tunnel{
		ID:       "test-tunnel",
		Protocol: "http",
		LocalPort: 3000,
	}

	proxy := NewHTTPProxy(3000, tunnel, logger)

	req := &types.DataForwardPayload{
		ConnectionID: "conn-123",
		RequestID:    "req-456",
	}

	errorResponse := proxy.createErrorResponse(req, http.StatusInternalServerError, "Test error")

	if errorResponse.ConnectionID != "conn-123" {
		t.Errorf("Expected connection ID 'conn-123', got %s", errorResponse.ConnectionID)
	}

	if errorResponse.RequestID != "req-456" {
		t.Errorf("Expected request ID 'req-456', got %s", errorResponse.RequestID)
	}

	if errorResponse.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, errorResponse.StatusCode)
	}

	if errorResponse.Headers["Content-Type"] != "application/json" {
		t.Errorf("Expected Content-Type header 'application/json', got %s", errorResponse.Headers["Content-Type"])
	}
}

func TestHTTPProxyGetLocalURL(t *testing.T) {
	logger := logrus.New()
	tunnel := &client.Tunnel{
		ID:       "test-tunnel",
		Protocol: "http",
		LocalPort: 3000,
	}

	proxy := NewHTTPProxy(3000, tunnel, logger)

	url := proxy.GetLocalURL()
	expected := "http://localhost:3000"

	if url != expected {
		t.Errorf("Expected URL %s, got %s", expected, url)
	}
}

func TestHTTPProxyGetTunnel(t *testing.T) {
	logger := logrus.New()
	tunnel := &client.Tunnel{
		ID:       "test-tunnel",
		Protocol: "http",
		LocalPort: 3000,
	}

	proxy := NewHTTPProxy(3000, tunnel, logger)

	retrievedTunnel := proxy.GetTunnel()
	if retrievedTunnel.ID != "test-tunnel" {
		t.Errorf("Expected tunnel ID 'test-tunnel', got %s", retrievedTunnel.ID)
	}
} 