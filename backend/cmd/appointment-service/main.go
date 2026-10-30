package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/jaydeepjr28/panchakarma-flow/internal/services/appointment/handler"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/appointment/repository"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/appointment/service"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/auth"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/config"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/database"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/kafka"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/logger"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/redis"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/server"
	appointmentv1 "github.com/jaydeepjr28/panchakarma-flow/proto/gen/go/appointment/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg, err := config.LoadBaseConfig("appointment-service", "appointment_db", 8083, 9083)
	if err != nil {
		slog.Error("Configuration failure", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Setup(cfg.LogLevel, cfg.Environment == "development")
	logger.Info("Starting Appointment Service",
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
		_ = db.RunMigrations(ctx, "migrations/appointment")
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

	// Connect to Clinic Service via gRPC with fallback
	var clinicConn *grpc.ClientConn
	clinicTarget := os.Getenv("CLINIC_GRPC_TARGET")
	if clinicTarget == "" {
		clinicTarget = "localhost:9082"
	}
	conn, err := grpc.Dial(clinicTarget, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err == nil {
		clinicConn = conn
		defer clinicConn.Close()
	}

	var kafkaProducer *kafka.Producer
	if len(cfg.Kafka.Brokers) > 0 {
		kafkaProducer = kafka.NewProducer(&cfg.Kafka)
		defer kafkaProducer.Close()
	}

	var repo repository.AppointmentRepository
	if db != nil {
		repo = repository.NewPostgresAppointmentRepository(db)
	}

	clinicValidator := service.NewGRPCClinicValidator(clinicConn)
	apptService := service.NewAppointmentService(repo, rdb, clinicValidator)

	// Transactional outbox publisher & background hold expiry ticker
	if repo != nil {
		outboxPublisher := service.NewOutboxPublisher(repo, kafkaProducer, 2*time.Second, 50)
		outboxPublisher.Start(ctx)
		defer outboxPublisher.Stop()

		// Background ticker for hold cleanup
		go func() {
			ticker := time.NewTicker(2 * time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					_, _ = repo.ExpireStaleHolds(ctx)
				}
			}
		}()
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

	httpHandler := handler.NewAppointmentHTTPHandler(apptService, tokenManager)
	grpcHandler := handler.NewAppointmentGRPCHandler(repo)

	apiV1 := srv.Router.Group("/api/v1")
	httpHandler.RegisterRoutes(apiV1)

	if srv.GRPCServer != nil {
		appointmentv1.RegisterAppointmentServiceServer(srv.GRPCServer, grpcHandler)
	}

	if err := srv.Start(); err != nil {
		logger.Error("Failed to start server", slog.String("error", err.Error()))
		os.Exit(1)
	}

	srv.WaitForShutdown()
}
