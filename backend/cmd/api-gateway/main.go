package main

import (
	"log/slog"
	"net/url"
	"os"

	"github.com/jaydeepjr28/panchakarma-flow/internal/gateway"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/config"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/logger"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/server"
)

func parseURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		logger.Error("Failed to parse service URL", slog.String("url", rawURL), slog.String("error", err.Error()))
		return nil
	}
	return u
}

func main() {
	cfg, err := config.LoadBaseConfig("api-gateway", "gateway_dummy", 8080, 0)
	if err != nil {
		slog.Error("Configuration failure", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Setup(cfg.LogLevel, cfg.Environment == "development")
	logger.Info("Starting API Gateway",
		slog.String("service", cfg.AppName),
		slog.Int("http_port", cfg.HTTPPort),
	)

	srv := server.NewServer(server.Config{
		ServiceName:    cfg.AppName,
		HTTPPort:       cfg.HTTPPort,
		GRPCPort:       0,
		AllowedOrigins: cfg.AllowedOrigins,
		IsDevelopment:  cfg.Environment == "development",
	})

	// Configure downstream microservice target URLs
	identityURL := parseURL(os.Getenv("IDENTITY_SERVICE_URL"))
	if identityURL == nil {
		identityURL = parseURL("http://localhost:8081")
	}

	clinicURL := parseURL(os.Getenv("CLINIC_SERVICE_URL"))
	if clinicURL == nil {
		clinicURL = parseURL("http://localhost:8082")
	}

	apptURL := parseURL(os.Getenv("APPOINTMENT_SERVICE_URL"))
	if apptURL == nil {
		apptURL = parseURL("http://localhost:8083")
	}

	therapyURL := parseURL(os.Getenv("THERAPY_SERVICE_URL"))
	if therapyURL == nil {
		therapyURL = parseURL("http://localhost:8084")
	}

	notifURL := parseURL(os.Getenv("NOTIFICATION_SERVICE_URL"))
	if notifURL == nil {
		notifURL = parseURL("http://localhost:8085")
	}

	feedbackURL := parseURL(os.Getenv("FEEDBACK_SERVICE_URL"))
	if feedbackURL == nil {
		feedbackURL = parseURL("http://localhost:8086")
	}

	gateway.RegisterGatewayRoutes(srv.Router, gateway.RouteMap{
		IdentityURL: identityURL,
		ClinicURL:   clinicURL,
		ApptURL:     apptURL,
		TherapyURL:  therapyURL,
		NotifURL:    notifURL,
		FeedbackURL: feedbackURL,
	})

	if err := srv.Start(); err != nil {
		logger.Error("Failed to start API Gateway", slog.String("error", err.Error()))
		os.Exit(1)
	}

	srv.WaitForShutdown()
}
