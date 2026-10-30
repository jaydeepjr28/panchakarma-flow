package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/jaydeepjr28/panchakarma-flow/internal/services/therapy/handler"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/therapy/repository"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/therapy/service"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/auth"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/config"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/database"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/kafka"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/logger"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/redis"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg, err := config.LoadBaseConfig("therapy-service", "therapy_db", 8084, 9084)
	if err != nil {
		slog.Error("Configuration failure", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Setup(cfg.LogLevel, cfg.Environment == "development")
	logger.Info("Starting Therapy Service",
		slog.String("service", cfg.AppName),
		slog.Int("http_port", cfg.HTTPPort),
		slog.Int("grpc_port", cfg.GRPCPort),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := database.Connect(ctx, &cfg.Database)
	if err != nil {
		logger.Warn("Database initial connection warning", slog.String("error", err.Error()))
	} else {
		defer db.Close()
		_ = db.RunMigrations(ctx, "migrations/therapy")
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

	// Connect to Appointment Service via gRPC
	var apptConn *grpc.ClientConn
	apptTarget := os.Getenv("APPOINTMENT_GRPC_TARGET")
	if apptTarget == "" {
		apptTarget = "localhost:9083"
	}
	aConn, err := grpc.Dial(apptTarget, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err == nil {
		apptConn = aConn
		defer apptConn.Close()
	}

	// Connect to Clinic Service via gRPC
	var clinicConn *grpc.ClientConn
	clinicTarget := os.Getenv("CLINIC_GRPC_TARGET")
	if clinicTarget == "" {
		clinicTarget = "localhost:9082"
	}
	cConn, err := grpc.Dial(clinicTarget, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err == nil {
		clinicConn = cConn
		defer clinicConn.Close()
	}

	var kafkaProducer *kafka.Producer
	if len(cfg.Kafka.Brokers) > 0 {
		kafkaProducer = kafka.NewProducer(&cfg.Kafka)
		defer kafkaProducer.Close()
	}

	var repo repository.TherapyRepository
	if db != nil {
		repo = repository.NewPostgresTherapyRepository(db)
	}

	apptVerifier := service.NewGRPCAppointmentVerifier(apptConn)
	pkgProvider := service.NewGRPCPackageSnapshotProvider(clinicConn)
	therapySvc := service.NewTherapyService(repo, apptVerifier, pkgProvider)

	// Transactional outbox background worker
	if repo != nil {
		outboxPublisher := service.NewOutboxPublisher(repo, kafkaProducer, 2*time.Second, 50)
		outboxPublisher.Start(ctx)
		defer outboxPublisher.Stop()
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

	httpHandler := handler.NewTherapyHTTPHandler(therapySvc, tokenManager)
	apiV1 := srv.Router.Group("/api/v1")
	httpHandler.RegisterRoutes(apiV1)

	if err := srv.Start(); err != nil {
		logger.Error("Failed to start server", slog.String("error", err.Error()))
		os.Exit(1)
	}

	srv.WaitForShutdown()
}
