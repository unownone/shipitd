package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/unownone/shipitd/internal/config"
	"github.com/sirupsen/logrus"
)

// TokenInfo represents authentication token information
type TokenInfo struct {
	Valid    bool   `json:"valid"`
	UserID   string `json:"user_id"`
	AuthType string `json:"auth_type"`
}

// CreateTunnelRequest represents a tunnel creation request
type CreateTunnelRequest struct {
	Protocol   string `json:"protocol"`
	LocalPort  int    `json:"local_port"`
	Subdomain  string `json:"subdomain,omitempty"`
	PublicPort *int   `json:"public_port,omitempty"`
}

// Tunnel represents a tunnel configuration
type Tunnel struct {
	ID         string    `json:"tunnel_id"`
	Protocol   string    `json:"protocol"`
	PublicURL  string    `json:"public_url"`
	Status     string    `json:"status"`
	Subdomain  string    `json:"subdomain,omitempty"`
	LocalPort  int       `json:"local_port"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TunnelList represents a list of tunnels
type TunnelList struct {
	Tunnels []*Tunnel `json:"tunnels"`
}

// ControlPlaneClient handles HTTP API communication with the ShipIt server
type ControlPlaneClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     *logrus.Logger
}

// NewControlPlaneClient creates a new control plane client
func NewControlPlaneClient(cfg *config.Config, logger *logrus.Logger) *ControlPlaneClient {
	// Create HTTP client with timeouts
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			IdleConnTimeout:     30 * time.Second,
			DisableCompression:  true,
		},
	}

	return &ControlPlaneClient{
		baseURL:    fmt.Sprintf("https://%s:%d/api/v1", cfg.Server.Domain, cfg.Server.APIPort),
		apiKey:     cfg.Auth.APIKey,
		httpClient: httpClient,
		logger:     logger,
	}
}

// SetBaseURL sets the base URL for the client (useful for testing)
func (c *ControlPlaneClient) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
}

// ValidateToken validates the API key with the server
func (c *ControlPlaneClient) ValidateToken(ctx context.Context) (*TokenInfo, error) {
	url := fmt.Sprintf("%s/auth/validate", c.baseURL)
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")

	c.logger.WithField("url", url).Debug("Validating API token")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to validate token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token validation failed with status: %d", resp.StatusCode)
	}

	var tokenInfo TokenInfo
	if err := json.NewDecoder(resp.Body).Decode(&tokenInfo); err != nil {
		return nil, fmt.Errorf("failed to decode token info: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"valid":    tokenInfo.Valid,
		"user_id":  tokenInfo.UserID,
		"auth_type": tokenInfo.AuthType,
	}).Info("Token validation completed")

	return &tokenInfo, nil
}

// GetTokenInfo gets detailed information about the API key
func (c *ControlPlaneClient) GetTokenInfo(ctx context.Context) (*TokenInfo, error) {
	url := fmt.Sprintf("%s/auth/token/info", c.baseURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")

	c.logger.WithField("url", url).Debug("Getting token info")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get token info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get token info failed with status: %d", resp.StatusCode)
	}

	var tokenInfo TokenInfo
	if err := json.NewDecoder(resp.Body).Decode(&tokenInfo); err != nil {
		return nil, fmt.Errorf("failed to decode token info: %w", err)
	}

	return &tokenInfo, nil
}

// CreateTunnel creates a new tunnel
func (c *ControlPlaneClient) CreateTunnel(ctx context.Context, req *CreateTunnelRequest) (*Tunnel, error) {
	url := fmt.Sprintf("%s/tunnels", c.baseURL)
	
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	httpReq.Header.Set("Content-Type", "application/json")

	c.logger.WithFields(logrus.Fields{
		"url":      url,
		"protocol": req.Protocol,
		"port":     req.LocalPort,
		"subdomain": req.Subdomain,
	}).Debug("Creating tunnel")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create tunnel: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("create tunnel failed with status: %d", resp.StatusCode)
	}

	var tunnel Tunnel
	if err := json.NewDecoder(resp.Body).Decode(&tunnel); err != nil {
		return nil, fmt.Errorf("failed to decode tunnel response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"tunnel_id":  tunnel.ID,
		"public_url": tunnel.PublicURL,
		"status":     tunnel.Status,
	}).Info("Tunnel created successfully")

	return &tunnel, nil
}

// ListTunnels lists all tunnels for the authenticated user
func (c *ControlPlaneClient) ListTunnels(ctx context.Context) ([]*Tunnel, error) {
	url := fmt.Sprintf("%s/tunnels", c.baseURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")

	c.logger.WithField("url", url).Debug("Listing tunnels")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list tunnels: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list tunnels failed with status: %d", resp.StatusCode)
	}

	var tunnelList TunnelList
	if err := json.NewDecoder(resp.Body).Decode(&tunnelList); err != nil {
		return nil, fmt.Errorf("failed to decode tunnel list: %w", err)
	}

	c.logger.WithField("tunnel_count", len(tunnelList.Tunnels)).Info("Retrieved tunnel list")

	return tunnelList.Tunnels, nil
}

// GetTunnel gets details for a specific tunnel
func (c *ControlPlaneClient) GetTunnel(ctx context.Context, tunnelID string) (*Tunnel, error) {
	url := fmt.Sprintf("%s/tunnels/%s", c.baseURL, tunnelID)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")

	c.logger.WithFields(logrus.Fields{
		"url":       url,
		"tunnel_id": tunnelID,
	}).Debug("Getting tunnel details")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get tunnel: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get tunnel failed with status: %d", resp.StatusCode)
	}

	var tunnel Tunnel
	if err := json.NewDecoder(resp.Body).Decode(&tunnel); err != nil {
		return nil, fmt.Errorf("failed to decode tunnel response: %w", err)
	}

	return &tunnel, nil
}

// DeleteTunnel deletes a tunnel
func (c *ControlPlaneClient) DeleteTunnel(ctx context.Context, tunnelID string) error {
	url := fmt.Sprintf("%s/tunnels/%s", c.baseURL, tunnelID)
	
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")

	c.logger.WithFields(logrus.Fields{
		"url":       url,
		"tunnel_id": tunnelID,
	}).Debug("Deleting tunnel")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete tunnel: %w", err)
	}
	defer resp.Body.Close()

	// Accept both 200 (OK) and 204 (No Content) as valid delete responses
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete tunnel failed with status: %d", resp.StatusCode)
	}

	c.logger.WithField("tunnel_id", tunnelID).Info("Tunnel deleted successfully")

	return nil
}

// GetTunnelStats gets statistics for a tunnel
func (c *ControlPlaneClient) GetTunnelStats(ctx context.Context, tunnelID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/tunnels/%s/stats", c.baseURL, tunnelID)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")

	c.logger.WithFields(logrus.Fields{
		"url":       url,
		"tunnel_id": tunnelID,
	}).Debug("Getting tunnel statistics")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get tunnel stats: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get tunnel stats failed with status: %d", resp.StatusCode)
	}

	var stats map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode tunnel stats: %w", err)
	}

	return stats, nil
} 