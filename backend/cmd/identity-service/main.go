package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/jaydeepjr28/panchakarma-flow/internal/services/identity/handler"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/identity/repository"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/identity/service"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/auth"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/config"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/database"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/logger"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/redis"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/server"
	identityv1 "github.com/jaydeepjr28/panchakarma-flow/proto/gen/go/identity/v1"
)

func main() {
	// 1. Load strongly-typed configuration
	cfg, err := config.LoadBaseConfig("identity-service", "identity_db", 8081, 9081)
	if err != nil {
		slog.Error("Configuration failure", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 2. Initialize structured logger
	logger.Setup(cfg.LogLevel, cfg.Environment == "development")
	logger.Info("Starting Identity Service",
		slog.String("service", cfg.AppName),
		slog.Int("http_port", cfg.HTTPPort),
		slog.Int("grpc_port", cfg.GRPCPort),
	)

	ctx := context.Background()

	// 3. Connect to PostgreSQL
	db, err := database.Connect(ctx, &cfg.Database)
	if err != nil {
		logger.Warn("Database initial connection warning (will retry on health checks)", slog.String("error", err.Error()))
	} else {
		defer db.Close()
		// Run database migrations if migrations folder exists
		if err := db.RunMigrations(ctx, "migrations/identity"); err != nil {
			logger.Warn("Migration warning", slog.String("error", err.Error()))
		}
	}

	// 4. Connect to Redis
	rdb, err := redis.Connect(ctx, &cfg.Redis)
	if err != nil {
		logger.Warn("Redis initial connection warning", slog.String("error", err.Error()))
	} else {
		defer rdb.Close()
	}

	// 5. Initialize TokenManager
	tokenManager, err := auth.NewTokenManager(cfg.Auth.JWTSecret, cfg.Auth.AccessTTL, cfg.Auth.RefreshTTL, rdb)
	if err != nil {
		logger.Error("Failed to initialize TokenManager", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 6. Build core server scaffold
	srv := server.NewServer(server.Config{
		ServiceName:    cfg.AppName,
		HTTPPort:       cfg.HTTPPort,
		GRPCPort:       cfg.GRPCPort,
		AllowedOrigins: cfg.AllowedOrigins,
		IsDevelopment:  cfg.Environment == "development",
	})

	// 7. Register component health checks
	if db != nil {
		srv.Health.RegisterCheck("database", db.PingCheck)
	}
	if rdb != nil {
		srv.Health.RegisterCheck("redis", rdb.PingCheck)
	}

	// 8. Wire dependencies
	userRepo := repository.NewPostgresUserRepository(db)
	authSvc := service.NewAuthService(userRepo, tokenManager)
	httpHandler := handler.NewAuthHandler(authSvc, tokenManager)
	grpcHandler := handler.NewGRPCHandler(authSvc)

	// 8. Register routes & gRPC services
	apiV1 := srv.Router.Group("/api/v1")
	httpHandler.RegisterRoutes(apiV1)

	if srv.GRPCServer != nil {
		identityv1.RegisterIdentityServiceServer(srv.GRPCServer, grpcHandler)
	}

	// 9. Start listeners
	if err := srv.Start(); err != nil {
		logger.Error("Failed to start server", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 10. Wait for OS signal for graceful shutdown
	srv.WaitForShutdown()
}
