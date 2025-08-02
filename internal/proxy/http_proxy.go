package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/unownone/shipitd/internal/client"
	"github.com/unownone/shipitd/pkg/types"
	"github.com/sirupsen/logrus"
)

// HTTPProxy handles HTTP request forwarding from ShipIt server to local services
type HTTPProxy struct {
	localPort int
	tunnel    *client.Tunnel
	logger    *logrus.Logger
	client    *http.Client
}

// NewHTTPProxy creates a new HTTP proxy instance
func NewHTTPProxy(localPort int, tunnel *client.Tunnel, logger *logrus.Logger) *HTTPProxy {
	// Create HTTP client with reasonable timeouts
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:        100,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}

	return &HTTPProxy{
		localPort: localPort,
		tunnel:    tunnel,
		logger:    logger,
		client:    client,
	}
}

// HandleRequest processes an incoming HTTP request from the ShipIt server
func (hp *HTTPProxy) HandleRequest(req *types.DataForwardPayload) (*types.DataResponsePayload, error) {
	startTime := time.Now()
	requestID := req.RequestID
	connectionID := req.ConnectionID

	hp.logger.WithFields(logrus.Fields{
		"request_id":    requestID,
		"connection_id": connectionID,
		"method":        req.Method,
		"path":          req.Path,
		"local_port":    hp.localPort,
	}).Debug("Handling HTTP request")

	// Create HTTP request for local service
	localURL := fmt.Sprintf("http://localhost:%d%s", hp.localPort, req.Path)
	httpReq, err := http.NewRequest(req.Method, localURL, strings.NewReader(string(req.Data)))
	if err != nil {
		hp.logger.WithError(err).Error("Failed to create HTTP request")
		return hp.createErrorResponse(req, http.StatusInternalServerError, "Failed to create request"), nil
	}

	// Copy headers from original request
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Set appropriate headers for local forwarding
	httpReq.Header.Set("X-Forwarded-For", "127.0.0.1")
	httpReq.Header.Set("X-Forwarded-Proto", "http")
	if host, exists := req.Headers["Host"]; exists {
		httpReq.Header.Set("X-Forwarded-Host", host)
	}
	httpReq.Header.Set("X-ShipIt-Tunnel", hp.tunnel.ID)

	// Make request to local service
	resp, err := hp.client.Do(httpReq)
	if err != nil {
		hp.logger.WithError(err).Error("Failed to forward request to local service")
		return hp.createErrorResponse(req, http.StatusBadGateway, "Failed to connect to local service"), nil
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		hp.logger.WithError(err).Error("Failed to read response body")
		return hp.createErrorResponse(req, http.StatusInternalServerError, "Failed to read response"), nil
	}

	// Convert response headers
	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0] // Take first value for simplicity
		}
	}

	// Create response message
	response := &types.DataResponsePayload{
		ConnectionID: connectionID,
		RequestID:    requestID,
		Data:         body,
		StatusCode:   resp.StatusCode,
		Headers:      headers,
	}

	duration := time.Since(startTime)
	hp.logger.WithFields(logrus.Fields{
		"request_id":    requestID,
		"connection_id": connectionID,
		"status_code":   resp.StatusCode,
		"duration_ms":   duration.Milliseconds(),
		"response_size": len(body),
	}).Info("HTTP request completed")

	return response, nil
}

// createErrorResponse creates an error response message
func (hp *HTTPProxy) createErrorResponse(req *types.DataForwardPayload, statusCode int, message string) *types.DataResponsePayload {
	errorBody := fmt.Sprintf(`{"error": "%s", "status": %d}`, message, statusCode)
	
	return &types.DataResponsePayload{
		ConnectionID: req.ConnectionID,
		RequestID:    req.RequestID,
		Data:         []byte(errorBody),
		StatusCode:   statusCode,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

// HealthCheck performs a health check on the local service
func (hp *HTTPProxy) HealthCheck() error {
	url := fmt.Sprintf("http://localhost:%d/health", hp.localPort)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}
	
	resp, err := hp.client.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	
	return fmt.Errorf("health check returned status %d", resp.StatusCode)
}

// GetLocalURL returns the local URL for this proxy
func (hp *HTTPProxy) GetLocalURL() string {
	return fmt.Sprintf("http://localhost:%d", hp.localPort)
}

// GetTunnel returns the associated tunnel
func (hp *HTTPProxy) GetTunnel() *client.Tunnel {
	return hp.tunnel
} 