package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/health"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/logger"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// Server coordinates HTTP (Gin) and gRPC listeners with unified graceful shutdown.
type Server struct {
	ServiceName string
	HTTPPort    int
	GRPCPort    int
	Router      *gin.Engine
	GRPCServer  *grpc.Server
	Health      *health.Checker
	httpServer  *http.Server
}

// Config defines initialization parameters for Server.
type Config struct {
	ServiceName    string
	HTTPPort       int
	GRPCPort       int
	AllowedOrigins []string
	IsDevelopment  bool
}

// NewServer initializes Gin engine and optional gRPC server with standard middlewares.
func NewServer(cfg Config) *Server {
	if !cfg.IsDevelopment {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(middleware.CorrelationID())
	router.Use(middleware.RequestLogger())
	router.Use(middleware.Recovery())
	router.Use(middleware.CORS(cfg.AllowedOrigins))

	checker := health.NewChecker(cfg.ServiceName)
	router.GET("/health/live", checker.LiveHandler)
	router.GET("/health/ready", checker.ReadyHandler)

	var grpcSrv *grpc.Server
	if cfg.GRPCPort > 0 {
		grpcSrv = grpc.NewServer()
		grpc_health_v1.RegisterHealthServer(grpcSrv, health.NewGRPCServer(checker))
		reflection.Register(grpcSrv)
	}

	return &Server{
		ServiceName: cfg.ServiceName,
		HTTPPort:    cfg.HTTPPort,
		GRPCPort:    cfg.GRPCPort,
		Router:      router,
		GRPCServer:  grpcSrv,
		Health:      checker,
	}
}

// Start launches HTTP and gRPC listeners in background goroutines.
func (s *Server) Start() error {
	// Start gRPC server if configured
	if s.GRPCServer != nil && s.GRPCPort > 0 {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.GRPCPort))
		if err != nil {
			return fmt.Errorf("failed to listen on gRPC port %d: %w", s.GRPCPort, err)
		}

		logger.Info("Starting gRPC server",
			slog.String("service", s.ServiceName),
			slog.Int("port", s.GRPCPort),
		)

		go func() {
			if err := s.GRPCServer.Serve(lis); err != nil && err != grpc.ErrServerStopped {
				logger.Error("gRPC server error", slog.String("error", err.Error()))
			}
		}()
	}

	// Start HTTP server
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.HTTPPort),
		Handler:      s.Router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logger.Info("Starting HTTP server",
		slog.String("service", s.ServiceName),
		slog.Int("port", s.HTTPPort),
	)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", slog.String("error", err.Error()))
		}
	}()

	return nil
}

// WaitForShutdown blocks until SIGINT or SIGTERM is caught, then gracefully terminates listeners.
func (s *Server) WaitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	sig := <-quit

	logger.Info("Shutdown signal received, terminating gracefully...",
		slog.String("service", s.ServiceName),
		slog.String("signal", sig.String()),
	)

	// Graceful shutdown with 5-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			logger.Error("HTTP server forced to shutdown", slog.String("error", err.Error()))
		}
	}

	if s.GRPCServer != nil {
		s.GRPCServer.GracefulStop()
	}

	logger.Info("Service stopped cleanly", slog.String("service", s.ServiceName))
}
