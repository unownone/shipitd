# ShipIt Client Implementation Guide

## Overview

This document provides a comprehensive guide for implementing a ShipIt client daemon that communicates with the ShipIt server. The client operates in two planes:

1. **Control Plane**: HTTP API communication for tunnel management and authentication
2. **Data Plane**: Custom TLS protocol for high-throughput traffic forwarding

## Architecture Overview

```mermaid
graph TB
    subgraph "Client Daemon"
        A[CLI Interface] --> B[Control Plane Client]
        A --> C[Data Plane Client]
        B --> D[HTTP Client]
        C --> E[TLS Connection Pool]
        E --> F[Message Handler]
        F --> G[Local Service Proxy]
    end
    
    subgraph "ShipIt Server"
        H[HTTP API Server] --> I[Authentication]
        H --> J[Tunnel Management]
        K[Agent Listener] --> L[Data Plane Handler]
        L --> M[Tunnel Manager]
    end
    
    subgraph "Local Services"
        N[Web App :3000]
        O[Database :5432]
        P[API Server :8080]
    end
    
    B -.->|HTTP/HTTPS| H
    C -.->|TLS :7223| K
    G --> N
    G --> O
    G --> P
    
    style A fill:#e1f5fe
    style H fill:#f3e5f5
    style K fill:#f3e5f5
    style N fill:#e8f5e8
    style O fill:#e8f5e8
    style P fill:#e8f5e8
```

## 1. Authentication System

### 1.1 API Key Authentication

The client uses API keys for authentication with the server. API keys are long-lived tokens that provide programmatic access to tunnel operations.

#### API Key Format

```
shipit_<32_random_characters>
```

#### Authentication Headers

```http
Authorization: Bearer shipit_abc123def456ghi789jkl012mno345pqr678stu901vwx234yz
```

#### API Key Validation Endpoint

```http
POST /api/v1/auth/validate
Content-Type: application/json
Authorization: Bearer <api_key>

Response:
{
  "valid": true,
  "user_id": "uuid",
  "auth_type": "api_key"
}
```

### 1.2 Authentication Flow

```mermaid
sequenceDiagram
    participant Client
    participant Server
    
    Client->>Server: POST /api/v1/auth/validate
    Note over Client,Server: Authorization: Bearer <api_key>
    Server->>Server: Validate API Key
    Server->>Client: 200 OK
    Note over Client,Server: {"valid": true, "user_id": "uuid"}
    
    Client->>Server: POST /api/v1/tunnels
    Note over Client,Server: Create tunnel via API
    Server->>Client: 201 Created
    Note over Client,Server: {"tunnel_id": "uuid", "public_url": "..."}
    
    Client->>Server: TLS Connection :7223
    Note over Client,Server: Data plane connection
    Client->>Server: Tunnel Registration Message
    Server->>Client: Acknowledgment
    Note over Client,Server: Maintain persistent connection
```

## 2. Control Plane API

### 2.1 Base URL

```
https://your-shipit-server.com/api/v1
```

### 2.2 Authentication Endpoints

#### Validate Token

```http
POST /api/v1/auth/validate
Authorization: Bearer <api_key>

Response:
{
  "valid": true,
  "user_id": "uuid",
  "auth_type": "api_key"
}
```

#### Get Token Info

```http
GET /api/v1/auth/token/info
Authorization: Bearer <api_key>

Response:
{
  "valid": true,
  "auth_type": "api_key",
  "user_id": "uuid",
  "user_email": "user@example.com",
  "user_name": "John Doe",
  "user_role": "user",
  "key_info": {
    "id": "uuid",
    "name": "Development Key",
    "created_at": "2024-01-01T00:00:00Z",
    "last_used_at": "2024-01-01T12:00:00Z",
    "expires_at": "2024-12-31T23:59:59Z"
  }
}
```

### 2.3 Tunnel Management Endpoints

#### Create Tunnel

```http
POST /api/v1/tunnels
Authorization: Bearer <api_key>
Content-Type: application/json

{
  "protocol": "http",
  "local_port": 3000,
  "subdomain": "myapp"  // Optional
}

Response:
{
  "tunnel_id": "uuid",
  "public_url": "https://myapp.yourdomain.com",
  "status": "active",
  "protocol": "http",
  "created_at": "2024-01-01T00:00:00Z"
}
```

#### List Tunnels

```http
GET /api/v1/tunnels
Authorization: Bearer <api_key>

Response:
{
  "tunnels": [
    {
      "tunnel_id": "uuid",
      "protocol": "http",
      "public_url": "https://myapp.yourdomain.com",
      "status": "active",
      "subdomain": "myapp",
      "local_port": 3000,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

#### Get Tunnel Details

```http
GET /api/v1/tunnels/{tunnel_id}
Authorization: Bearer <api_key>

Response:
{
  "tunnel_id": "uuid",
  "protocol": "http",
  "public_url": "https://myapp.yourdomain.com",
  "status": "active",
  "subdomain": "myapp",
  "local_port": 3000,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

#### Delete Tunnel

```http
DELETE /api/v1/tunnels/{tunnel_id}
Authorization: Bearer <api_key>

Response:
{
  "status": "terminated",
  "tunnel_id": "uuid",
  "message": "Tunnel terminated successfully"
}
```

#### Get Tunnel Statistics

```http
GET /api/v1/tunnels/{tunnel_id}/stats
Authorization: Bearer <api_key>

Response:
{
  "tunnel_id": "uuid",
  "active_connections": 5,
  "total_requests": 1234,
  "total_bytes_in": 5242880,
  "total_bytes_out": 3145728,
  "analytics": []
}
```

## 3. Data Plane Protocol

### 3.1 Connection Establishment

The client establishes a persistent TLS connection to the server on port 7223 for data forwarding.

#### Connection Parameters

- **Protocol**: TLS 1.2+
- **Port**: 7223
- **Server Name**: Your ShipIt server domain
- **Connection Pool**: Maintain 10 connections per tunnel by default

### 3.2 Message Format

All messages follow this binary format:

```
┌─────────────────┬─────────────────┬─────────────────┐
│   Message Type  │   Tunnel ID     │     Payload     │
│    (1 byte)     │   (16 bytes)    │   (variable)    │
└─────────────────┴─────────────────┴─────────────────┘
```

#### Message Types

| Type | Value | Description |
|------|-------|-------------|
| `TunnelRegistration` | `0x01` | Register tunnel connection |
| `DataForward` | `0x02` | Server forwards visitor data |
| `DataResponse` | `0x03` | Client sends response data |
| `ConnectionClose` | `0x04` | Close specific connection |
| `Heartbeat` | `0x05` | Keepalive message |
| `Error` | `0x06` | Error notification |
| `Acknowledge` | `0x07` | Acknowledgment message |

### 3.3 Message Payloads

#### Tunnel Registration

```json
{
  "protocol": "http",
  "local_port": 3000,
  "subdomain": "myapp",
  "public_port": null,
  "max_connections": 10
}
```

#### Data Forward (from server)

```json
{
  "connection_id": "conn_123",
  "request_id": "req_456",
  "data": [/* raw HTTP request data */],
  "headers": {
    "Host": "myapp.yourdomain.com",
    "User-Agent": "Mozilla/5.0...",
    "Accept": "text/html,application/xhtml+xml..."
  },
  "method": "GET",
  "path": "/api/users"
}
```

#### Data Response (to server)

```json
{
  "connection_id": "conn_123",
  "request_id": "req_456",
  "data": [/* raw HTTP response data */],
  "status_code": 200,
  "headers": {
    "Content-Type": "application/json",
    "Content-Length": "1234"
  }
}
```

#### Heartbeat

```json
{
  "timestamp": 1704067200,
  "active_conns": 5,
  "total_requests": 1234
}
```

### 3.4 Connection Lifecycle

```mermaid
stateDiagram-v2
    [*] --> Disconnected
    Disconnected --> Connecting : Start Connection
    Connecting --> Connected : TLS Handshake Success
    Connecting --> Disconnected : Connection Failed
    Connected --> Registering : Send Registration
    Registering --> Active : Registration Acknowledged
    Registering --> Disconnected : Registration Failed
    Active --> Heartbeating : Send Heartbeat
    Heartbeating --> Active : Heartbeat Response
    Active --> Processing : Receive Data Forward
    Processing --> Active : Send Data Response
    Active --> Disconnected : Connection Lost
    Disconnected --> [*] : Shutdown
```

## 4. Client Implementation

### 4.1 Client States

```mermaid
stateDiagram-v2
    [*] --> Initializing
    Initializing --> Authenticated : API Key Valid
    Initializing --> Error : Authentication Failed
    Authenticated --> CreatingTunnel : Create Tunnel
    CreatingTunnel --> TunnelCreated : Tunnel Created
    CreatingTunnel --> Error : Tunnel Creation Failed
    TunnelCreated --> ConnectingDataPlane : Connect to Data Plane
    ConnectingDataPlane --> DataPlaneConnected : TLS Connected
    ConnectingDataPlane --> Error : Connection Failed
    DataPlaneConnected --> RegisteringTunnel : Send Registration
    RegisteringTunnel --> Active : Registration Acknowledged
    RegisteringTunnel --> Error : Registration Failed
    Active --> Processing : Handle Traffic
    Processing --> Active : Continue Processing
    Active --> Reconnecting : Connection Lost
    Reconnecting --> DataPlaneConnected : Reconnection Success
    Reconnecting --> Error : Reconnection Failed
    Error --> [*] : Exit
```

### 4.2 Client Configuration

```yaml
# client.yaml
server:
  domain: "your-shipit-server.com"
  api_port: 443
  data_plane_port: 7223
  tls_verify: true

auth:
  api_key: "shipit_abc123def456ghi789jkl012mno345pqr678stu901vwx234yz"
  auto_refresh: true

tunnels:
  - name: "web-app"
    protocol: "http"
    local_port: 3000
    subdomain: "myapp"
    auto_start: true
    
  - name: "database"
    protocol: "tcp"
    local_port: 5432
    auto_start: false

connection:
  pool_size: 10
  heartbeat_interval: 30s
  reconnect_interval: 5s
  max_reconnect_attempts: 10
  connection_timeout: 30s

logging:
  level: "info"
  format: "json"
  file: "/var/log/shipit-client.log"
```

### 4.3 Client Implementation Structure

```
shipit-client/
├── cmd/
│   └── client/
│       └── main.go              # Client entry point
├── internal/
│   ├── client/
│   │   ├── control_plane.go     # HTTP API client
│   │   ├── data_plane.go        # TLS protocol client
│   │   ├── tunnel_manager.go    # Tunnel lifecycle
│   │   └── connection_pool.go   # Connection management
│   ├── protocol/
│   │   ├── message.go           # Message types and serialization
│   │   ├── reader.go            # Message reading
│   │   └── writer.go            # Message writing
│   ├── proxy/
│   │   ├── http_proxy.go        # HTTP request forwarding
│   │   └── tcp_proxy.go         # TCP connection forwarding
│   ├── config/
│   │   └── config.go            # Configuration management
│   └── logger/
│       └── logger.go            # Logging setup
├── pkg/
│   └── types/
│       ├── tunnel.go            # Tunnel data structures
│       └── message.go           # Protocol message types
└── configs/
    └── client.yaml              # Client configuration
```

### 4.4 Core Client Components

#### Control Plane Client

```go
type ControlPlaneClient struct {
    baseURL    string
    apiKey     string
    httpClient *http.Client
}

func (c *ControlPlaneClient) ValidateToken() (*TokenInfo, error)
func (c *ControlPlaneClient) CreateTunnel(req *CreateTunnelRequest) (*Tunnel, error)
func (c *ControlPlaneClient) ListTunnels() ([]*Tunnel, error)
func (c *ControlPlaneClient) DeleteTunnel(tunnelID string) error
```

#### Data Plane Client

```go
type DataPlaneClient struct {
    serverAddr string
    tlsConfig  *tls.Config
    pool       *ConnectionPool
}

func (d *DataPlaneClient) Connect() error
func (d *DataPlaneClient) RegisterTunnel(tunnel *Tunnel) error
func (d *DataPlaneClient) SendHeartbeat() error
func (d *DataPlaneClient) HandleDataForward(msg *Message) error
```

#### Tunnel Manager

```go
type TunnelManager struct {
    controlPlane *ControlPlaneClient
    dataPlane    *DataPlaneClient
    tunnels      map[string]*Tunnel
    config       *Config
}

func (tm *TunnelManager) StartTunnel(config *TunnelConfig) error
func (tm *TunnelManager) StopTunnel(tunnelID string) error
func (tm *TunnelManager) ListTunnels() []*Tunnel
func (tm *TunnelManager) Reconnect() error
```

## 5. Connection Management

### 5.1 Connection Pool

The client maintains a pool of TLS connections to the server for high availability and load balancing.

```go
type ConnectionPool struct {
    connections []*Connection
    mutex       sync.RWMutex
    roundRobin  int
}

type Connection struct {
    ID       string
    Conn     net.Conn
    IsHealthy bool
    LastUsed time.Time
}
```

### 5.2 Heartbeat Management

```mermaid
sequenceDiagram
    participant Client
    participant Server
    
    loop Every 30 seconds
        Client->>Server: Heartbeat Message
        Note over Client,Server: {"timestamp": 1704067200, "active_conns": 5}
        Server->>Client: Heartbeat Response
        Note over Client,Server: {"timestamp": 1704067200}
    end
    
    Note over Client,Server: If no heartbeat for 2 minutes
    Server->>Client: Close Connection
    Client->>Client: Reconnect
```

### 5.3 Reconnection Strategy

```go
type ReconnectionStrategy struct {
    maxAttempts     int
    baseDelay       time.Duration
    maxDelay        time.Duration
    backoffFactor   float64
}

func (rs *ReconnectionStrategy) NextDelay(attempt int) time.Duration {
    delay := rs.baseDelay * time.Duration(math.Pow(rs.backoffFactor, float64(attempt)))
    if delay > rs.maxDelay {
        delay = rs.maxDelay
    }
    return delay
}
```

## 6. Traffic Forwarding

### 6.1 HTTP Proxy

For HTTP tunnels, the client acts as a reverse proxy:

```go
type HTTPProxy struct {
    localPort int
    tunnel    *Tunnel
}

func (hp *HTTPProxy) HandleRequest(conn net.Conn) {
    // 1. Read HTTP request from server
    // 2. Forward to local service
    // 3. Read response from local service
    // 4. Send response back to server
}
```

### 6.2 TCP Proxy

For TCP tunnels, the client forwards raw TCP data:

```go
type TCPProxy struct {
    localPort int
    tunnel    *Tunnel
}

func (tp *TCPProxy) HandleConnection(conn net.Conn) {
    // 1. Establish connection to local service
    // 2. Bidirectional data forwarding
    // 3. Handle connection lifecycle
}
```

## 7. Error Handling

### 7.1 Error Types

| Error Type | Description | Recovery Action |
|------------|-------------|-----------------|
| `AUTHENTICATION_FAILED` | Invalid API key | Exit with error |
| `TUNNEL_NOT_FOUND` | Tunnel doesn't exist | Recreate tunnel |
| `CONNECTION_TIMEOUT` | Network timeout | Reconnect |
| `PROTOCOL_ERROR` | Invalid message format | Close connection |
| `RATE_LIMITED` | Too many requests | Wait and retry |

### 7.2 Error Recovery

```mermaid
flowchart TD
    A[Error Occurs] --> B{Error Type}
    B -->|Authentication| C[Exit with Error]
    B -->|Network| D[Reconnect]
    B -->|Protocol| E[Close Connection]
    B -->|Rate Limit| F[Wait and Retry]
    B -->|Tunnel| G[Recreate Tunnel]
    
    D --> H{Reconnection Success}
    H -->|Yes| I[Continue]
    H -->|No| J[Max Attempts Reached]
    J --> C
    
    E --> K[Establish New Connection]
    K --> I
    
    F --> L[Wait 60s]
    L --> I
    
    G --> M[Create New Tunnel]
    M --> I
```

## 8. Security Considerations

### 8.1 TLS Configuration

```go
tlsConfig := &tls.Config{
    MinVersion: tls.VersionTLS12,
    ServerName: serverDomain,
    InsecureSkipVerify: false, // Always verify in production
}
```

### 8.2 API Key Security

- Store API keys securely (environment variables, keyring)
- Never log API keys
- Rotate keys regularly
- Use least privilege principle

### 8.3 Network Security

- Use TLS for all communications
- Validate server certificates
- Implement connection timeouts
- Rate limit reconnection attempts

## 9. Monitoring and Logging

### 9.1 Metrics to Track

| Metric | Description | Alert Threshold |
|--------|-------------|-----------------|
| Connection Status | Active/Inactive | Alert if inactive > 5min |
| Request Rate | Requests per second | Alert if > 1000 req/s |
| Response Time | Average response time | Alert if > 5s |
| Error Rate | Error percentage | Alert if > 5% |
| Reconnection Count | Reconnection attempts | Alert if > 10/hour |

### 9.2 Logging Levels

```go
type LogLevel int

const (
    DEBUG LogLevel = iota
    INFO
    WARN
    ERROR
)

// Log structured events
logger.WithFields(logrus.Fields{
    "tunnel_id": tunnelID,
    "protocol":  "http",
    "local_port": 3000,
    "status":    "active",
}).Info("Tunnel started successfully")
```

## 10. Deployment

### 10.1 Systemd Service

```ini
# /etc/systemd/system/shipit-client.service
[Unit]
Description=ShipIt Client Daemon
After=network.target

[Service]
Type=simple
User=shipit
Group=shipit
ExecStart=/usr/local/bin/shipit-client
Restart=always
RestartSec=10
Environment=SHIPIT_CONFIG=/etc/shipit/client.yaml

[Install]
WantedBy=multi-user.target
```

### 10.2 Docker Deployment

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o shipit-client cmd/client/main.go

FROM alpine:latest
RUN addgroup -g 1000 shipit && \
    adduser -D -s /bin/sh -u 1000 -G shipit shipit
USER shipit
COPY --from=builder /app/shipit-client /usr/local/bin/
CMD ["shipit-client"]
```

### 10.3 Configuration Management

```bash
#!/bin/bash
# deploy.sh

# Create configuration directory
sudo mkdir -p /etc/shipit

# Copy configuration
sudo cp configs/client.yaml /etc/shipit/

# Set permissions
sudo chown -R shipit:shipit /etc/shipit
sudo chmod 600 /etc/shipit/client.yaml

# Install binary
sudo cp bin/shipit-client /usr/local/bin/
sudo chmod +x /usr/local/bin/shipit-client

# Enable and start service
sudo systemctl enable shipit-client
sudo systemctl start shipit-client
```

## 11. Testing

### 11.1 Unit Tests

```go
func TestTunnelRegistration(t *testing.T) {
    client := NewDataPlaneClient("localhost:7223")
    
    tunnel := &Tunnel{
        ID: "test-tunnel-id",
        Protocol: "http",
        LocalPort: 3000,
    }
    
    err := client.RegisterTunnel(tunnel)
    assert.NoError(t, err)
}
```

### 11.2 Integration Tests

```go
func TestEndToEndTunnel(t *testing.T) {
    // Start local test server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello, World!"))
    }))
    defer server.Close()
    
    // Create tunnel
    client := NewClient(config)
    tunnel, err := client.CreateTunnel(&CreateTunnelRequest{
        Protocol: "http",
        LocalPort: 3000,
    })
    assert.NoError(t, err)
    
    // Test tunnel access
    resp, err := http.Get(tunnel.PublicURL)
    assert.NoError(t, err)
    assert.Equal(t, 200, resp.StatusCode)
}
```

## 12. Troubleshooting

### 12.1 Common Issues

| Issue | Symptoms | Solution |
|-------|----------|----------|
| Authentication Failed | "Invalid API key" | Check API key format and validity |
| Connection Refused | "Connection refused" | Verify server address and port |
| TLS Handshake Failed | "TLS handshake failed" | Check server certificate and domain |
| Tunnel Not Found | "Tunnel not found" | Verify tunnel exists and is active |
| Rate Limited | "Too many requests" | Implement exponential backoff |

### 12.2 Debug Commands

```bash
# Check client status
shipit-client status

# Test API key
shipit-client auth test

# List tunnels
shipit-client tunnels list

# Test data plane connection
shipit-client connect test

# View logs
journalctl -u shipit-client -f

# Check configuration
shipit-client config validate
```

This implementation guide provides a complete framework for building a ShipIt client daemon that can reliably communicate with the ShipIt server, handle authentication, manage tunnels, and forward traffic efficiently.
