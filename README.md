# ShipIt Client Daemon

[![Go Report Card](https://goreportcard.com/badge/github.com/unownone/shipitd)](https://goreportcard.com/report/github.com/unownone/shipitd)
[![Go Version](https://img.shields.io/github/go-mod/go-version/unownone/shipitd)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![CI/CD](https://github.com/unownone/shipitd/workflows/CI%2FCD%20Pipeline/badge.svg)](https://github.com/unownone/shipitd/actions)

**ShipIt Client Daemon** is a Go application that connects to the ShipIt server to expose local services to the internet through secure tunnels. It operates in two planes:

- **Control Plane**: HTTP API communication for tunnel management and authentication
- **Data Plane**: Custom TLS protocol for high-throughput traffic forwarding

## 🚀 Features

- **🔐 Secure Authentication** - API key-based authentication with secure credential storage
- **🌐 HTTP/TCP Tunneling** - Forward both HTTP and raw TCP traffic to local services
- **🔄 Auto-Reconnection** - Automatic reconnection with exponential backoff
- **📊 Health Monitoring** - Built-in health checks and metrics collection
- **🛡️ Security First** - Input validation, secure defaults, and credential rotation
- **📱 Cross-Platform** - Support for Linux, macOS, and Windows
- **⚡ High Performance** - Optimized for high-throughput traffic forwarding
- **🔧 Easy Configuration** - YAML configuration with environment variable overrides

## 📋 Requirements

- **Go 1.20+** (for building from source)
- **ShipIt Server** - A running ShipIt server instance
- **API Key** - Valid API key from your ShipIt server
- **Network Access** - Ability to connect to the ShipIt server

## 🛠️ Installation

### Quick Install (macOS)

```bash
# Or download the latest release
curl -L https://github.com/unownone/shipitd/releases/latest/download/shipitd-darwin-amd64.tar.gz | tar xz
sudo mv shipitd /usr/local/bin/
```

### Quick Install (Linux)

```bash
# Download and install
curl -L https://github.com/unownone/shipitd/releases/latest/download/shipitd-linux-amd64.tar.gz | tar xz
sudo mv shipitd /usr/local/bin/
```

### Build from Source

```bash
# Clone the repository
git clone https://github.com/unownone/shipitd.git
cd shipitd

# Build the binary
make build

# Install
sudo make install
```

## ⚙️ Configuration

### Initial Setup

1. **Initialize Configuration**

   ```bash
   shipitd config init
   ```

2. **Edit Configuration**

   ```bash
   shipitd config edit
   ```

3. **Validate Configuration**

   ```bash
   shipitd config validate
   ```

### Configuration File

The configuration file is located at `~/.shipitd/config.yaml`:

```yaml
# Server Configuration
server:
  control_plane_url: "https://api.shipit.dev"
  data_plane_url: "tls://data.shipit.dev:7223"
  timeout: 30s
  retry_interval: 5s
  max_retries: 3

# Authentication
auth:
  api_key: "shipit_your_api_key_here"
  # Or use environment variable: SHIPIT_API_KEY

# Tunnel Configuration
tunnel:
  default_protocol: "http"
  max_connections: 100
  connection_timeout: 60s
  idle_timeout: 300s

# Connection Pool
connection:
  max_idle_conns: 10
  max_open_conns: 100
  conn_idle_timeout: 90s
  conn_max_lifetime: 0

# Logging
logging:
  level: "info"  # debug, info, warn, error
  format: "json" # json, text
  output: "stdout" # stdout, stderr, file
  file_path: "/var/log/shipit-client.log"
  max_size: 100
  max_age: 30
  max_backups: 10
```

### Environment Variables

You can override configuration using environment variables:

```bash
export SHIPIT_SERVER_CONTROL_PLANE_URL="https://your-server.com"
export SHIPIT_AUTH_API_KEY="your_api_key"
export SHIPIT_LOGGING_LEVEL="debug"
```

## 🚀 Usage

### Basic Commands

```bash
# Start the daemon
shipitd start

# Stop the daemon
shipitd stop

# Check status
shipitd status

# Test authentication
shipitd auth test
```

### Tunnel Management

```bash
# List all tunnels
shipitd tunnels list

# Create a tunnel
shipitd tunnels create --name "my-app" --local-url "http://localhost:3000" --protocol "http"

# Delete a tunnel
shipitd tunnels delete --id "tunnel-id"
```

### Configuration Management

```bash
# Show current configuration
shipitd config show

# Validate configuration
shipitd config validate

# Edit configuration
shipitd config edit

# Initialize new configuration
shipitd config init
```

### Service Management

```bash
# Install as system service (Linux)
sudo shipitd service install

# Start service
sudo systemctl start shipitd

# Enable auto-start
sudo systemctl enable shipitd

# Check service status
sudo systemctl status shipitd
```

## 🔧 Development

### Prerequisites

- Go 1.20+
- Make
- Docker (optional)

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Build with specific version
VERSION=v1.0.0 make build
```

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run integration tests
go test ./internal/testing/...

# Run benchmarks
go test -bench=. ./internal/testing/...
```

### Development Workflow

```bash
# Install development tools
make install-tools

# Format code
make format

# Run linter
make lint

# Run tests
make test

# Build and run
make dev
```

## 📊 Monitoring

### Health Checks

The daemon provides health check endpoints:

```bash
# Main health check
curl http://localhost:8080/health

# Readiness probe
curl http://localhost:8080/health/ready

# Liveness probe
curl http://localhost:8080/health/live

# Metrics
curl http://localhost:8080/metrics
```

### Health Check Response

```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "checks": [
    {
      "name": "connection",
      "status": "healthy",
      "message": "Connected to ShipIt server",
      "timestamp": "2024-01-15T10:30:00Z",
      "duration": "15ms"
    },
    {
      "name": "memory",
      "status": "healthy",
      "message": "Memory usage: 25 MB",
      "timestamp": "2024-01-15T10:30:00Z"
    }
  ],
  "version": "v1.0.0"
}
```

### Logging

The daemon provides structured logging:

```bash
# View logs
tail -f /var/log/shipit-client.log

# Filter by level
grep '"level":"error"' /var/log/shipit-client.log
```

## 🔒 Security

### API Key Management

```bash
# Store API key securely
shipitd auth store-key

# Rotate API key
shipitd auth rotate-key

# Remove stored API key
shipitd auth remove-key
```

### Security Features

- **Secure Credential Storage** - API keys stored in system keyring
- **Input Validation** - All inputs are validated and sanitized
- **Secure Defaults** - Secure configuration defaults
- **File Permissions** - Configuration files have restricted permissions
- **Path Validation** - All file paths are validated for security

## 🐛 Troubleshooting

### Common Issues

**1. Authentication Failed**

```bash
# Check API key
shipitd auth test

# Verify server URL
shipitd config show | grep control_plane_url
```

**2. Connection Timeout**

```bash
# Check network connectivity
curl -v https://api.shipit.dev/health

# Increase timeout in config
echo "server:
  timeout: 60s" >> ~/.shipitd/config.yaml
```

**3. Service Won't Start**

```bash
# Check logs
journalctl -u shipitd -f

# Verify configuration
shipitd config validate

# Check permissions
ls -la ~/.shipitd/
```

**4. High Memory Usage**

```bash
# Check memory usage
curl http://localhost:8080/metrics | grep memory

# Reduce connection pool
echo "connection:
  max_open_conns: 50" >> ~/.shipitd/config.yaml
```

### Debug Mode

```bash
# Enable debug logging
export SHIPIT_LOGGING_LEVEL="debug"
shipitd start

# Or edit config file
echo "logging:
  level: debug" >> ~/.shipitd/config.yaml
```

## 📚 API Reference

### Control Plane API

The client communicates with the ShipIt server via HTTP API:

```bash
# Authentication
POST /api/v1/auth/validate
Authorization: Bearer <api_key>

# Tunnel Management
GET    /api/v1/tunnels
POST   /api/v1/tunnels
DELETE /api/v1/tunnels?id=<tunnel_id>
```

### Data Plane Protocol

The client uses a custom TLS protocol for high-throughput data forwarding:

- **Message Types**: Registration, Data Forward, Response, Heartbeat
- **Protocol**: Binary over TLS
- **Port**: 7223 (configurable)

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Setup

```bash
# Fork and clone
git clone https://github.com/your-username/shipitd.git
cd shipitd

# Install dependencies
go mod download

# Run tests
make test

# Build
make build
```

### Code Style

- Follow Go conventions
- Use `gofmt` for formatting
- Run `golangci-lint` before committing
- Write tests for new features

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

- **Documentation**: [docs/](docs/)
- **Issues**: [GitHub Issues](https://github.com/unownone/shipitd/issues)
- **Discussions**: [GitHub Discussions](https://github.com/unownone/shipitd/discussions)
- **Security**: [Security Policy](SECURITY.md)

## 🙏 Acknowledgments

- Built with [Go](https://golang.org/)
- Uses [Cobra](https://github.com/spf13/cobra) for CLI
- Uses [Viper](https://github.com/spf13/viper) for configuration
- Uses [Logrus](https://github.com/sirupsen/logrus) for logging
- Uses [Testify](https://github.com/stretchr/testify) for testing

---

**Made with ❤️ by the ShipIt Team**
