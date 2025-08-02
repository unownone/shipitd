package testing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/unownone/shipitd/internal/client"
	"github.com/unownone/shipitd/internal/config"
	"github.com/unownone/shipitd/pkg/types"
	"github.com/sirupsen/logrus"
)

// BenchmarkMessageSerialization benchmarks message serialization
func BenchmarkMessageSerialization(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := types.NewMessage(types.MessageTypeTunnelRegistration, "test-tunnel", []byte("test-payload"))
		_, err := msg.Serialize()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMessageDeserialization benchmarks message deserialization
func BenchmarkMessageDeserialization(b *testing.B) {
	msg := types.NewMessage(types.MessageTypeTunnelRegistration, "test-tunnel", []byte("test-payload"))
	data, err := msg.Serialize()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var newMsg types.Message
		err := newMsg.Deserialize(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkConnectionEstablishment benchmarks connection establishment
func BenchmarkConnectionEstablishment(b *testing.B) {
	mockServer := NewMockShipItServer(&MockServerConfig{
		ValidAPIKeys: []string{"benchmark-key"},
	})
	defer mockServer.Close()

	logger := logrus.New()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Domain:  "localhost",
			APIPort: 8080,
		},
		Auth: config.AuthConfig{
			APIKey: "benchmark-key",
		},
	}

	controlPlane := client.NewControlPlaneClient(cfg, logger)
	controlPlane.SetBaseURL(mockServer.URL() + "/api/v1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := controlPlane.ValidateToken(context.Background())
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDataForwarding benchmarks data forwarding
func BenchmarkDataForwarding(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := types.NewMessage(types.MessageTypeDataForward, "test-tunnel", []byte("test-payload"))
		_, err := msg.Serialize()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryUsage benchmarks memory usage
func BenchmarkMemoryUsage(b *testing.B) {
	b.ReportAllocs()

	// Create multiple tunnels to test memory usage
	tunnels := make([]*types.Tunnel, 1000)
	for i := 0; i < 1000; i++ {
		tunnels[i] = &types.Tunnel{
			ID:          fmt.Sprintf("tunnel-%d", i),
			Name:        fmt.Sprintf("Test Tunnel %d", i),
			LocalURL:    fmt.Sprintf("http://localhost:%d", 3000+i),
			Protocol:    "http",
			Description: fmt.Sprintf("Test tunnel %d", i),
			PublicURL:   fmt.Sprintf("https://tunnel-%d.shipit.dev", i),
			Status:      "active",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate processing tunnels
		for _, tunnel := range tunnels {
			_ = tunnel.ID
			_ = tunnel.Name
		}
	}
}

// BenchmarkConcurrentConnections benchmarks concurrent connections
func BenchmarkConcurrentConnections(b *testing.B) {
	mockServer := NewMockShipItServer(&MockServerConfig{
		ValidAPIKeys: []string{"concurrent-key"},
	})
	defer mockServer.Close()

	logger := logrus.New()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Domain:  "localhost",
			APIPort: 8080,
		},
		Auth: config.AuthConfig{
			APIKey: "concurrent-key",
		},
	}

	controlPlane := client.NewControlPlaneClient(cfg, logger)
	controlPlane.SetBaseURL(mockServer.URL() + "/api/v1")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := controlPlane.ValidateToken(context.Background())
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkHighThroughput benchmarks high throughput scenarios
func BenchmarkHighThroughput(b *testing.B) {
	// Simulate high throughput by processing many messages
	messages := make([]*types.Message, 1000)
	for i := 0; i < 1000; i++ {
		messages[i] = types.NewMessage(
			types.MessageTypeDataForward,
			fmt.Sprintf("tunnel-%d", i%10),
			[]byte(fmt.Sprintf("data-%d", i)),
		)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, msg := range messages {
			_, err := msg.Serialize()
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

// BenchmarkMemoryLeakScenarios benchmarks memory leak scenarios
func BenchmarkMemoryLeakScenarios(b *testing.B) {
	b.ReportAllocs()

	// Simulate potential memory leak scenarios
	for i := 0; i < b.N; i++ {
		// Create and discard objects to test garbage collection
		for j := 0; j < 100; j++ {
			msg := types.NewMessage(types.MessageTypeDataForward, "test-tunnel", []byte("test-data"))
			_ = msg
		}

		// Force garbage collection
		if i%10 == 0 {
			// This would normally be runtime.GC(), but we're just simulating
			_ = i
		}
	}
}

// BenchmarkTunnelLifecycle benchmarks complete tunnel lifecycle
func BenchmarkTunnelLifecycle(b *testing.B) {
	mockServer := NewMockShipItServer(&MockServerConfig{
		ValidAPIKeys: []string{"lifecycle-key"},
	})
	defer mockServer.Close()

	logger := logrus.New()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Domain:  "localhost",
			APIPort: 8080,
		},
		Auth: config.AuthConfig{
			APIKey: "lifecycle-key",
		},
	}

	controlPlane := client.NewControlPlaneClient(cfg, logger)
	controlPlane.SetBaseURL(mockServer.URL() + "/api/v1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create tunnel
		tunnelReq := &client.CreateTunnelRequest{
			Protocol:  "http",
			LocalPort: 3000,
			Subdomain: fmt.Sprintf("benchmark-tunnel-%d", i),
		}

		tunnel, err := controlPlane.CreateTunnel(context.Background(), tunnelReq)
		if err != nil {
			b.Fatal(err)
		}

		// List tunnels
		_, err = controlPlane.ListTunnels(context.Background())
		if err != nil {
			b.Fatal(err)
		}

		// Delete tunnel
		err = controlPlane.DeleteTunnel(context.Background(), tunnel.ID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLoadTests runs comprehensive load tests
func BenchmarkLoadTests(b *testing.B) {
	mockServer := NewMockShipItServer(&MockServerConfig{
		ValidAPIKeys: []string{"load-test-key"},
	})
	defer mockServer.Close()

	logger := logrus.New()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Domain:  "localhost",
			APIPort: 8080,
		},
		Auth: config.AuthConfig{
			APIKey: "load-test-key",
		},
	}

	controlPlane := client.NewControlPlaneClient(cfg, logger)
	controlPlane.SetBaseURL(mockServer.URL() + "/api/v1")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Simulate load by performing multiple operations
			// 1. Validate API key
			_, err := controlPlane.ValidateToken(context.Background())
			if err != nil {
				b.Fatal(err)
			}

			// 2. Create tunnel
			tunnelReq := &client.CreateTunnelRequest{
				Protocol:  "http",
				LocalPort: 3000,
				Subdomain: "load-test-tunnel",
			}

			tunnel, err := controlPlane.CreateTunnel(context.Background(), tunnelReq)
			if err != nil {
				b.Fatal(err)
			}

			// 3. List tunnels
			_, err = controlPlane.ListTunnels(context.Background())
			if err != nil {
				b.Fatal(err)
			}

			// 4. Delete tunnel
			err = controlPlane.DeleteTunnel(context.Background(), tunnel.ID)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
} 