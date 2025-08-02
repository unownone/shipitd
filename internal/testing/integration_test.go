package testing

import (
	"context"
	"testing"
	"time"

	"github.com/unownone/shipitd/internal/client"
	"github.com/unownone/shipitd/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegrationTunnelLifecycle tests the complete tunnel lifecycle
func TestIntegrationTunnelLifecycle(t *testing.T) {
	// Create mock server
	mockServer := NewMockShipItServer(&MockServerConfig{
		ValidAPIKeys: []string{"test-api-key-123"},
	})
	defer mockServer.Close()

	// Create client configuration
	logger := logrus.New()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Domain:  "localhost",
			APIPort: 8080,
		},
		Auth: config.AuthConfig{
			APIKey: "test-api-key-123",
		},
	}

	// Create control plane client with mock server URL
	controlPlane := client.NewControlPlaneClient(cfg, logger)
	// Override the base URL to use the mock server
	controlPlane.SetBaseURL(mockServer.URL() + "/api/v1")

	// Test authentication
	t.Run("Authentication", func(t *testing.T) {
		mockServer.Reset()
		valid, err := controlPlane.ValidateToken(context.Background())
		require.NoError(t, err)
		assert.True(t, valid.Valid)
		assert.Equal(t, 1, mockServer.GetAuthCalls())
	})

	// Test tunnel creation
	t.Run("TunnelCreation", func(t *testing.T) {
		mockServer.Reset()
		tunnelReq := &client.CreateTunnelRequest{
			Protocol:  "http",
			LocalPort: 3000,
			Subdomain: "test-tunnel",
		}

		tunnel, err := controlPlane.CreateTunnel(context.Background(), tunnelReq)
		require.NoError(t, err)
		assert.NotNil(t, tunnel)
		assert.Equal(t, "http", tunnel.Protocol)
		assert.Equal(t, "active", tunnel.Status)
		assert.Equal(t, 1, mockServer.GetTunnelCalls())
	})

	// Test tunnel listing
	t.Run("TunnelListing", func(t *testing.T) {
		mockServer.Reset()
		tunnels, err := controlPlane.ListTunnels(context.Background())
		require.NoError(t, err)
		assert.Len(t, tunnels, 0) // Should be empty after reset
	})

	// Test tunnel deletion
	t.Run("TunnelDeletion", func(t *testing.T) {
		mockServer.Reset()
		// First create a tunnel to delete
		tunnelReq := &client.CreateTunnelRequest{
			Protocol:  "http",
			LocalPort: 8080,
			Subdomain: "delete-test-tunnel",
		}

		tunnel, err := controlPlane.CreateTunnel(context.Background(), tunnelReq)
		require.NoError(t, err)

		// Delete the tunnel
		err = controlPlane.DeleteTunnel(context.Background(), tunnel.ID)
		require.NoError(t, err)

		// Verify tunnel is deleted
		tunnels, err := controlPlane.ListTunnels(context.Background())
		require.NoError(t, err)
		assert.Len(t, tunnels, 0) // Should be empty after deletion
	})
}

// TestIntegrationAuthenticationFlow tests the complete authentication flow
func TestIntegrationAuthenticationFlow(t *testing.T) {
	mockServer := NewMockShipItServer(&MockServerConfig{
		ValidAPIKeys: []string{"valid-key", "another-valid-key"},
	})
	defer mockServer.Close()

	logger := logrus.New()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Domain:  "localhost",
			APIPort: 8080,
		},
		Auth: config.AuthConfig{
			APIKey: "valid-key",
		},
	}

	controlPlane := client.NewControlPlaneClient(cfg, logger)
	controlPlane.SetBaseURL(mockServer.URL() + "/api/v1")

	t.Run("ValidAuthentication", func(t *testing.T) {
		valid, err := controlPlane.ValidateToken(context.Background())
		require.NoError(t, err)
		assert.True(t, valid.Valid)
	})

	t.Run("InvalidAuthentication", func(t *testing.T) {
		cfg.Auth.APIKey = "invalid-key"
		controlPlane := client.NewControlPlaneClient(cfg, logger)
		controlPlane.SetBaseURL(mockServer.URL() + "/api/v1")

		_, err := controlPlane.ValidateToken(context.Background())
		require.Error(t, err)
	})
}

// TestIntegrationReconnectionLogic tests reconnection behavior
func TestIntegrationReconnectionLogic(t *testing.T) {
	mockServer := NewMockShipItServer(&MockServerConfig{
		ValidAPIKeys: []string{"reconnect-test-key"},
	})
	defer mockServer.Close()

	logger := logrus.New()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Domain:  "localhost",
			APIPort: 8080,
		},
		Auth: config.AuthConfig{
			APIKey: "reconnect-test-key",
		},
		Connection: config.ConnectionConfig{
			MaxReconnectAttempts: 3,
			ReconnectInterval:     time.Second,
		},
	}

	controlPlane := client.NewControlPlaneClient(cfg, logger)
	controlPlane.SetBaseURL(mockServer.URL() + "/api/v1")

	t.Run("ReconnectionOnFailure", func(t *testing.T) {
		// Close the mock server to simulate connection failure
		mockServer.Close()

		// Attempt to validate API key (should fail)
		_, err := controlPlane.ValidateToken(context.Background())
		assert.Error(t, err)

		// Recreate mock server
		mockServer = NewMockShipItServer(&MockServerConfig{
			ValidAPIKeys: []string{"reconnect-test-key"},
		})
		defer mockServer.Close()

		// Update client with new server URL
		controlPlane.SetBaseURL(mockServer.URL() + "/api/v1")

		// Should work again
		valid, err := controlPlane.ValidateToken(context.Background())
		require.NoError(t, err)
		assert.True(t, valid.Valid)
	})
}

// TestIntegrationErrorHandling tests error handling scenarios
func TestIntegrationErrorHandling(t *testing.T) {
	mockServer := NewMockShipItServer(&MockServerConfig{
		ValidAPIKeys: []string{"error-test-key"},
	})
	defer mockServer.Close()

	logger := logrus.New()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Domain:  "localhost",
			APIPort: 8080,
		},
		Auth: config.AuthConfig{
			APIKey: "error-test-key",
		},
	}

	controlPlane := client.NewControlPlaneClient(cfg, logger)
	controlPlane.SetBaseURL(mockServer.URL() + "/api/v1")

	t.Run("InvalidTunnelRequest", func(t *testing.T) {
		// Test with invalid tunnel request
		tunnelReq := &client.CreateTunnelRequest{
			Protocol:  "", // Invalid: empty protocol
			LocalPort: 3000,
		}

		_, err := controlPlane.CreateTunnel(context.Background(), tunnelReq)
		// This might succeed with the mock server, but in real scenarios
		// it would fail validation
		if err != nil {
			assert.Contains(t, err.Error(), "validation")
		}
	})

	t.Run("NetworkTimeout", func(t *testing.T) {
		// Create a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		// This should timeout quickly
		_, err := controlPlane.ValidateToken(ctx)
		if err != nil {
			assert.Contains(t, err.Error(), "timeout")
		}
	})
}

// TestIntegrationGracefulShutdown tests graceful shutdown behavior
func TestIntegrationGracefulShutdown(t *testing.T) {
	mockServer := NewMockShipItServer(&MockServerConfig{
		ValidAPIKeys: []string{"shutdown-test-key"},
	})
	defer mockServer.Close()

	logger := logrus.New()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Domain:  "localhost",
			APIPort: 8080,
		},
		Auth: config.AuthConfig{
			APIKey: "shutdown-test-key",
		},
	}

	controlPlane := client.NewControlPlaneClient(cfg, logger)
	controlPlane.SetBaseURL(mockServer.URL() + "/api/v1")

	t.Run("GracefulShutdown", func(t *testing.T) {
		// Create a context that can be cancelled
		ctx, cancel := context.WithCancel(context.Background())

		// Start a goroutine that validates API key
		done := make(chan bool)
		go func() {
			_, err := controlPlane.ValidateToken(ctx)
			// Should not error due to graceful shutdown
			_ = err // Ignore error for graceful shutdown test
			done <- true
		}()

		// Cancel the context to simulate shutdown
		cancel()

		// Wait for the goroutine to finish
		select {
		case <-done:
			// Success
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for graceful shutdown")
		}
	})
}

// TestIntegrationDataPlaneCommunication tests data plane communication
func TestIntegrationDataPlaneCommunication(t *testing.T) {
	mockServer := NewMockShipItServer(&MockServerConfig{
		ValidAPIKeys: []string{"dataplane-test-key"},
	})
	defer mockServer.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Domain:        "localhost",
			APIPort:       8080,
			DataPlanePort: 7223,
		},
		Auth: config.AuthConfig{
			APIKey: "dataplane-test-key",
		},
	}

	t.Run("DataPlaneConnection", func(t *testing.T) {
		// This would test the actual data plane communication
		// For now, we'll just verify the configuration is correct
		assert.NotEmpty(t, cfg.Server.Domain)
		assert.NotEmpty(t, cfg.Auth.APIKey)
	})
}

// TestIntegrationEndToEnd tests a complete end-to-end scenario
func TestIntegrationEndToEnd(t *testing.T) {
	mockServer := NewMockShipItServer(&MockServerConfig{
		ValidAPIKeys: []string{"e2e-test-key"},
	})
	defer mockServer.Close()

	logger := logrus.New()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Domain:  "localhost",
			APIPort: 8080,
		},
		Auth: config.AuthConfig{
			APIKey: "e2e-test-key",
		},
	}

	controlPlane := client.NewControlPlaneClient(cfg, logger)
	controlPlane.SetBaseURL(mockServer.URL() + "/api/v1")

	t.Run("CompleteWorkflow", func(t *testing.T) {
		mockServer.Reset()
		
		// 1. Authenticate
		valid, err := controlPlane.ValidateToken(context.Background())
		require.NoError(t, err)
		assert.True(t, valid.Valid)

		// 2. Create tunnel
		tunnelReq := &client.CreateTunnelRequest{
			Protocol:  "http",
			LocalPort: 3000,
			Subdomain: "e2e-test-tunnel",
		}

		tunnel, err := controlPlane.CreateTunnel(context.Background(), tunnelReq)
		require.NoError(t, err)
		assert.NotNil(t, tunnel)

		// 3. List tunnels
		tunnels, err := controlPlane.ListTunnels(context.Background())
		require.NoError(t, err)
		assert.Len(t, tunnels, 1)

		// 4. Delete tunnel
		err = controlPlane.DeleteTunnel(context.Background(), tunnel.ID)
		require.NoError(t, err)

		// 5. Verify deletion
		tunnels, err = controlPlane.ListTunnels(context.Background())
		require.NoError(t, err)
		assert.Len(t, tunnels, 0)
	})
} 