package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheck represents a health check result
type HealthCheck struct {
	Name      string      `json:"name"`
	Status    HealthStatus `json:"status"`
	Message   string      `json:"message,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Duration  time.Duration `json:"duration,omitempty"`
}

// HealthResponse represents the overall health response
type HealthResponse struct {
	Status    HealthStatus `json:"status"`
	Timestamp time.Time   `json:"timestamp"`
	Checks    []HealthCheck `json:"checks"`
	Version   string      `json:"version"`
}

// HealthChecker defines the interface for health checks
type HealthChecker interface {
	Name() string
	Check(ctx context.Context) HealthCheck
}

// HealthServer provides HTTP health check endpoints
type HealthServer struct {
	server     *http.Server
	checkers   []HealthChecker
	logger     *logrus.Logger
	version    string
	mu         sync.RWMutex
	lastCheck  time.Time
	lastStatus HealthStatus
}

// NewHealthServer creates a new health server
func NewHealthServer(port int, version string, logger *logrus.Logger) *HealthServer {
	hs := &HealthServer{
		checkers: make([]HealthChecker, 0),
		logger:   logger,
		version:  version,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", hs.handleHealth)
	mux.HandleFunc("/health/ready", hs.handleReady)
	mux.HandleFunc("/health/live", hs.handleLive)
	mux.HandleFunc("/metrics", hs.handleMetrics)

	hs.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	return hs
}

// AddChecker adds a health checker
func (hs *HealthServer) AddChecker(checker HealthChecker) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	hs.checkers = append(hs.checkers, checker)
}

// Start starts the health server
func (hs *HealthServer) Start() error {
	hs.logger.Info("Starting health server")
	return hs.server.ListenAndServe()
}

// Stop stops the health server
func (hs *HealthServer) Stop(ctx context.Context) error {
	hs.logger.Info("Stopping health server")
	return hs.server.Shutdown(ctx)
}

// handleHealth handles the main health check endpoint
func (hs *HealthServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	ctx := r.Context()
	checks := make([]HealthCheck, 0, len(hs.checkers))

	// Run all health checks
	for _, checker := range hs.checkers {
		check := checker.Check(ctx)
		checks = append(checks, check)
	}

	// Determine overall status
	status := hs.determineOverallStatus(checks)
	hs.lastStatus = status
	hs.lastCheck = time.Now()

	response := HealthResponse{
		Status:    status,
		Timestamp: hs.lastCheck,
		Checks:    checks,
		Version:   hs.version,
	}

	w.Header().Set("Content-Type", "application/json")
	if status == HealthStatusUnhealthy {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else if status == HealthStatusDegraded {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	json.NewEncoder(w).Encode(response)
}

// handleReady handles the readiness probe
func (hs *HealthServer) handleReady(w http.ResponseWriter, r *http.Request) {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	if hs.lastStatus == HealthStatusUnhealthy {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ready"))
}

// handleLive handles the liveness probe
func (hs *HealthServer) handleLive(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("alive"))
}

// handleMetrics handles metrics endpoint
func (hs *HealthServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	metrics := map[string]interface{}{
		"health_status":    hs.lastStatus,
		"last_check":       hs.lastCheck,
		"checkers_count":   len(hs.checkers),
		"uptime_seconds":   time.Since(hs.lastCheck).Seconds(),
		"version":          hs.version,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// determineOverallStatus determines the overall health status
func (hs *HealthServer) determineOverallStatus(checks []HealthCheck) HealthStatus {
	if len(checks) == 0 {
		return HealthStatusHealthy
	}

	unhealthyCount := 0
	degradedCount := 0

	for _, check := range checks {
		switch check.Status {
		case HealthStatusUnhealthy:
			unhealthyCount++
		case HealthStatusDegraded:
			degradedCount++
		}
	}

	if unhealthyCount > 0 {
		return HealthStatusUnhealthy
	}

	if degradedCount > 0 {
		return HealthStatusDegraded
	}

	return HealthStatusHealthy
}

// ConnectionHealthChecker checks connection health
type ConnectionHealthChecker struct {
	name     string
	checkFn  func(context.Context) error
	logger   *logrus.Logger
}

// NewConnectionHealthChecker creates a new connection health checker
func NewConnectionHealthChecker(name string, checkFn func(context.Context) error, logger *logrus.Logger) *ConnectionHealthChecker {
	return &ConnectionHealthChecker{
		name:    name,
		checkFn: checkFn,
		logger:  logger,
	}
}

// Name returns the checker name
func (chc *ConnectionHealthChecker) Name() string {
	return chc.name
}

// Check performs the health check
func (chc *ConnectionHealthChecker) Check(ctx context.Context) HealthCheck {
	start := time.Now()
	
	err := chc.checkFn(ctx)
	
	duration := time.Since(start)
	
	check := HealthCheck{
		Name:      chc.name,
		Timestamp: time.Now(),
		Duration:  duration,
	}

	if err != nil {
		check.Status = HealthStatusUnhealthy
		check.Message = err.Error()
		chc.logger.WithError(err).Warnf("Health check failed: %s", chc.name)
	} else {
		check.Status = HealthStatusHealthy
		chc.logger.Debugf("Health check passed: %s", chc.name)
	}

	return check
}

// MemoryHealthChecker checks memory usage
type MemoryHealthChecker struct {
	maxMemoryMB int64
	logger      *logrus.Logger
}

// NewMemoryHealthChecker creates a new memory health checker
func NewMemoryHealthChecker(maxMemoryMB int64, logger *logrus.Logger) *MemoryHealthChecker {
	return &MemoryHealthChecker{
		maxMemoryMB: maxMemoryMB,
		logger:      logger,
	}
}

// Name returns the checker name
func (mhc *MemoryHealthChecker) Name() string {
	return "memory"
}

// Check performs the memory health check
func (mhc *MemoryHealthChecker) Check(ctx context.Context) HealthCheck {
	check := HealthCheck{
		Name:      mhc.Name(),
		Timestamp: time.Now(),
	}

	// Get memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	memoryMB := int64(m.Alloc / 1024 / 1024)
	
	check.Message = fmt.Sprintf("Memory usage: %d MB", memoryMB)

	if memoryMB > mhc.maxMemoryMB {
		check.Status = HealthStatusDegraded
		mhc.logger.Warnf("High memory usage: %d MB", memoryMB)
	} else {
		check.Status = HealthStatusHealthy
	}

	return check
}

// ServiceHealthChecker checks service status
type ServiceHealthChecker struct {
	name   string
	logger *logrus.Logger
}

// NewServiceHealthChecker creates a new service health checker
func NewServiceHealthChecker(name string, logger *logrus.Logger) *ServiceHealthChecker {
	return &ServiceHealthChecker{
		name:   name,
		logger: logger,
	}
}

// Name returns the checker name
func (shc *ServiceHealthChecker) Name() string {
	return shc.name
}

// Check performs the service health check
func (shc *ServiceHealthChecker) Check(ctx context.Context) HealthCheck {
	check := HealthCheck{
		Name:      shc.Name(),
		Timestamp: time.Now(),
		Status:    HealthStatusHealthy,
		Message:   "Service is running",
	}

	shc.logger.Debugf("Service health check: %s", shc.name)
	return check
} 