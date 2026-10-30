package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/jaydeepjr28/panchakarma-flow/internal/services/clinic/handler"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/clinic/repository"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/clinic/service"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/auth"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/config"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/database"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/logger"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/redis"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/server"
	clinicv1 "github.com/jaydeepjr28/panchakarma-flow/proto/gen/go/clinic/v1"
)

func main() {
	cfg, err := config.LoadBaseConfig("clinic-service", "clinic_db", 8082, 9082)
	if err != nil {
		slog.Error("Configuration failure", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Setup(cfg.LogLevel, cfg.Environment == "development")
	logger.Info("Starting Clinic Service",
		slog.String("service", cfg.AppName),
		slog.Int("http_port", cfg.HTTPPort),
		slog.Int("grpc_port", cfg.GRPCPort),
	)

	ctx := context.Background()

	db, err := database.Connect(ctx, &cfg.Database)
	if err != nil {
		logger.Warn("Database initial connection warning", slog.String("error", err.Error()))
	} else {
		defer db.Close()
		_ = db.RunMigrations(ctx, "migrations/clinic")
	}

	rdb, err := redis.Connect(ctx, &cfg.Redis)
	if err != nil {
		logger.Warn("Redis initial connection warning", slog.String("error", err.Error()))
	} else {
		defer rdb.Close()
	}

	tokenManager, err := auth.NewTokenManager(cfg.Auth.JWTSecret, cfg.Auth.AccessTTL, cfg.Auth.RefreshTTL, rdb)
	if err != nil {
		logger.Error("Failed to initialize TokenManager", slog.String("error", err.Error()))
		os.Exit(1)
	}

	srv := server.NewServer(server.Config{
		ServiceName:    cfg.AppName,
		HTTPPort:       cfg.HTTPPort,
		GRPCPort:       cfg.GRPCPort,
		AllowedOrigins: cfg.AllowedOrigins,
		IsDevelopment:  cfg.Environment == "development",
	})

	if db != nil {
		srv.Health.RegisterCheck("database", db.PingCheck)
	}
	if rdb != nil {
		srv.Health.RegisterCheck("redis", rdb.PingCheck)
	}

	clinicRepo := repository.NewPostgresClinicRepository(db)
	clinicSvc := service.NewClinicService(clinicRepo)
	httpHandler := handler.NewClinicHTTPHandler(clinicSvc, tokenManager)
	grpcHandler := handler.NewClinicGRPCHandler(clinicSvc)

	apiV1 := srv.Router.Group("/api/v1")
	httpHandler.RegisterRoutes(apiV1)

	if srv.GRPCServer != nil {
		clinicv1.RegisterClinicServiceServer(srv.GRPCServer, grpcHandler)
	}

	if err := srv.Start(); err != nil {
		logger.Error("Failed to start server", slog.String("error", err.Error()))
		os.Exit(1)
	}

	srv.WaitForShutdown()
}
