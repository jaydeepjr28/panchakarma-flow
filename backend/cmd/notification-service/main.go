package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/jaydeepjr28/panchakarma-flow/internal/services/notification/handler"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/notification/provider"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/notification/repository"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/notification/service"
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
	cfg, err := config.LoadBaseConfig("notification-service", "notification_db", 8085, 0)
	if err != nil {
		slog.Error("Configuration failure", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Setup(cfg.LogLevel, cfg.Environment == "development")
	logger.Info("Starting Notification Service",
		slog.String("service", cfg.AppName),
		slog.Int("http_port", cfg.HTTPPort),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := database.Connect(ctx, &cfg.Database)
	if err != nil {
		logger.Warn("Database initial connection warning", slog.String("error", err.Error()))
	} else {
		defer db.Close()
		_ = db.RunMigrations(ctx, "migrations/notification")
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

	// Kafka Producer for Outbox and DLQ
	var kafkaProducer *kafka.Producer
	if len(cfg.Kafka.Brokers) > 0 {
		kafkaProducer = kafka.NewProducer(&cfg.Kafka)
		defer kafkaProducer.Close()
	}

	// Repositories and Adapters
	var repo repository.NotificationRepository
	if db != nil {
		repo = repository.NewPostgresNotificationRepository(db)
	}
	devProvider := provider.NewDevelopmentDeliveryAdapter()

	// Connect gRPC client to Appointment Service for reminder pre-checks
	var apptChecker service.AppointmentStatusChecker
	apptTarget := os.Getenv("APPOINTMENT_GRPC_TARGET")
	if apptTarget == "" {
		apptTarget = "127.0.0.1:9083"
	}
	apptConn, err := grpc.Dial(apptTarget, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err == nil {
		defer apptConn.Close()
		apptChecker = service.NewGRPCAppointmentStatusChecker(apptConn)
	}

	notificationSvc := service.NewNotificationService(repo, devProvider, apptChecker, cfg.Kafka.ConsumerGroup, kafkaProducer)

	// Launch Kafka Consumer Manager in background
	if len(cfg.Kafka.Brokers) > 0 {
		consumerManager := service.NewConsumerManager(notificationSvc, &cfg.Kafka)
		_ = consumerManager.Start(ctx)
		defer consumerManager.Stop()
	}

	// Launch Scheduled Reminders Worker
	reminderWorker := service.NewReminderWorker(notificationSvc, 5*time.Second)
	reminderWorker.Start(ctx)
	defer reminderWorker.Stop()

	// Launch Transactional Outbox Worker
	if repo != nil && kafkaProducer != nil {
		outboxPublisher := service.NewOutboxPublisher(repo, kafkaProducer, 2*time.Second, 50)
		outboxPublisher.Start(ctx)
		defer outboxPublisher.Stop()
	}

	srv := server.NewServer(server.Config{
		ServiceName:    cfg.AppName,
		HTTPPort:       cfg.HTTPPort,
		GRPCPort:       0,
		AllowedOrigins: cfg.AllowedOrigins,
		IsDevelopment:  cfg.Environment == "development",
	})

	if db != nil {
		srv.Health.RegisterCheck("database", db.PingCheck)
	}
	if rdb != nil {
		srv.Health.RegisterCheck("redis", rdb.PingCheck)
	}

	httpHandler := handler.NewNotificationHTTPHandler(notificationSvc, tokenManager)

	apiV1 := srv.Router.Group("/api/v1")
	httpHandler.RegisterRoutes(apiV1)

	if err := srv.Start(); err != nil {
		logger.Error("Failed to start server", slog.String("error", err.Error()))
		os.Exit(1)
	}

	srv.WaitForShutdown()
}
