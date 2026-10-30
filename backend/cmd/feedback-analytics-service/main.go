package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/jaydeepjr28/panchakarma-flow/internal/services/feedback/handler"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/feedback/repository"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/feedback/service"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/auth"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/config"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/database"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/kafka"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/logger"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/redis"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/server"
)

func main() {
	cfg, err := config.LoadBaseConfig("feedback-analytics-service", "feedback_analytics_db", 8086, 0)
	if err != nil {
		slog.Error("Configuration failure", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Setup(cfg.LogLevel, cfg.Environment == "development")
	logger.Info("Starting Feedback & Analytics Service",
		slog.String("service", cfg.AppName),
		slog.Int("http_port", cfg.HTTPPort),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Connect to PostgreSQL
	db, err := database.Connect(ctx, &cfg.Database)
	if err != nil {
		logger.Warn("Database initial connection warning", slog.String("error", err.Error()))
	} else {
		defer db.Close()
		_ = db.RunMigrations(ctx, "migrations/feedback_analytics")
	}

	// 2. Connect to Redis
	rdb, err := redis.Connect(ctx, &cfg.Redis)
	if err != nil {
		logger.Warn("Redis initial connection warning", slog.String("error", err.Error()))
	} else {
		defer rdb.Close()
	}

	// 3. Initialize TokenManager
	tokenManager, err := auth.NewTokenManager(cfg.Auth.JWTSecret, cfg.Auth.AccessTTL, cfg.Auth.RefreshTTL, rdb)
	if err != nil {
		logger.Error("Failed to initialize TokenManager", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 4. Initialize Repository & Service
	var repo repository.FeedbackRepository
	if db != nil {
		repo = repository.NewFeedbackRepository(db)
	}

	feedbackSvc := service.NewFeedbackService(repo)

	// 5. Initialize Kafka Producer & Outbox Publisher
	var producer *kafka.Producer
	if len(cfg.Kafka.Brokers) > 0 {
		producer = kafka.NewProducer(&cfg.Kafka)
		defer producer.Close()
	}

	var outboxPub *service.OutboxPublisher
	if repo != nil {
		outboxPub = service.NewOutboxPublisher(repo, producer, 2*time.Second)
		outboxPub.Start()
		defer outboxPub.Stop()
	}

	// 6. Initialize Kafka Consumer Manager
	var consumerMgr *service.ConsumerManager
	if repo != nil && len(cfg.Kafka.Brokers) > 0 {
		consumerMgr = service.NewConsumerManager(repo, &cfg.Kafka, "feedback-analytics-group")
		_ = consumerMgr.Start(ctx)
		defer consumerMgr.Stop()
	}

	// 7. Setup HTTP Server
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

	httpHandler := handler.NewFeedbackHTTPHandler(feedbackSvc, tokenManager)

	apiV1 := srv.Router.Group("/api/v1")
	httpHandler.RegisterRoutes(apiV1)

	if err := srv.Start(); err != nil {
		logger.Error("Failed to start server", slog.String("error", err.Error()))
		os.Exit(1)
	}

	srv.WaitForShutdown()
}
