package health

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// CheckFunc is a function that inspects the health of an external dependency.
type CheckFunc func(ctx context.Context) error

// Checker aggregates component checks and evaluates overall service health.
type Checker struct {
	serviceName string
	checks      map[string]CheckFunc
	mu          sync.RWMutex
}

// NewChecker initializes a health checker for a given service.
func NewChecker(serviceName string) *Checker {
	return &Checker{
		serviceName: serviceName,
		checks:      make(map[string]CheckFunc),
	}
}

// RegisterCheck adds a named component check (e.g., "database", "redis").
func (h *Checker) RegisterCheck(name string, check CheckFunc) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checks[name] = check
}

// ComponentStatus represents the health report of a single component.
type ComponentStatus struct {
	Status  string `json:"status"` // "UP" or "DOWN"
	Error   string `json:"error,omitempty"`
	Latency string `json:"latency"`
}

// Report represents the composite health report.
type Report struct {
	Status      string                     `json:"status"` // "UP", "DEGRADED", "DOWN"
	ServiceName string                     `json:"service"`
	Timestamp   string                     `json:"timestamp"`
	Components  map[string]ComponentStatus `json:"components"`
}

// LiveHandler is the HTTP liveness probe handler (/health/live).
func (h *Checker) LiveHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "UP",
		"service":   h.serviceName,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// ReadyHandler is the HTTP readiness probe handler (/health/ready).
func (h *Checker) ReadyHandler(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	ctx, cancel := context.WithTimeout(c.Request.Context(), 4*time.Second)
	defer cancel()

	report := Report{
		Status:      "UP",
		ServiceName: h.serviceName,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Components:  make(map[string]ComponentStatus),
	}

	hasFailure := false
	for name, check := range h.checks {
		start := time.Now()
		err := check(ctx)
		latency := time.Since(start).String()

		if err != nil {
			hasFailure = true
			report.Components[name] = ComponentStatus{
				Status:  "DOWN",
				Error:   err.Error(),
				Latency: latency,
			}
		} else {
			report.Components[name] = ComponentStatus{
				Status:  "UP",
				Latency: latency,
			}
		}
	}

	statusCode := http.StatusOK
	if hasFailure {
		report.Status = "DEGRADED"
		// If all or critical checks fail, return 503
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, report)
}

// GRPCServer implements grpc.health.v1.HealthServer.
type GRPCServer struct {
	grpc_health_v1.UnimplementedHealthServer
	checker *Checker
}

// NewGRPCServer returns a new gRPC health server wrapping the checker.
func NewGRPCServer(checker *Checker) *GRPCServer {
	return &GRPCServer{checker: checker}
}

func (s *GRPCServer) Check(ctx context.Context, in *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	// Simple liveness check for gRPC
	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}, nil
}

func (s *GRPCServer) Watch(in *grpc_health_v1.HealthCheckRequest, server grpc_health_v1.Health_WatchServer) error {
	return server.Send(&grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	})
}
