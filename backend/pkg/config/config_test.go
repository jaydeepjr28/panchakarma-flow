package config_test

import (
	"os"
	"testing"

	"github.com/jaydeepjr28/panchakarma-flow/pkg/config"
)

func TestConfigValidation(t *testing.T) {
	t.Run("ValidConfiguration", func(t *testing.T) {
		cfg := &config.BaseConfig{
			AppName:     "test-service",
			Environment: "test",
			HTTPPort:    8080,
			GRPCPort:    9080,
			Database: config.DatabaseConfig{
				Host:   "localhost",
				Port:   5432,
				User:   "postgres",
				DBName: "test_db",
			},
			Redis: config.RedisConfig{
				Host: "localhost",
				Port: 6379,
			},
			Kafka: config.KafkaConfig{
				Brokers: []string{"localhost:9092"},
			},
		}

		if err := cfg.Validate(); err != nil {
			t.Errorf("expected valid config, got error: %v", err)
		}
	})

	t.Run("InvalidPort", func(t *testing.T) {
		cfg := &config.BaseConfig{
			AppName:  "test-service",
			HTTPPort: 99999, // Invalid
			Database: config.DatabaseConfig{Host: "localhost", DBName: "test", User: "test"},
			Redis:    config.RedisConfig{Host: "localhost"},
			Kafka:    config.KafkaConfig{Brokers: []string{"localhost:9092"}},
		}

		if err := cfg.Validate(); err == nil {
			t.Errorf("expected error for port 99999, got nil")
		}
	})

	t.Run("MissingDatabaseHost", func(t *testing.T) {
		cfg := &config.BaseConfig{
			AppName:  "test-service",
			HTTPPort: 8080,
			Database: config.DatabaseConfig{Host: "", DBName: "test", User: "test"},
			Redis:    config.RedisConfig{Host: "localhost"},
			Kafka:    config.KafkaConfig{Brokers: []string{"localhost:9092"}},
		}

		if err := cfg.Validate(); err == nil {
			t.Errorf("expected error for missing DB host, got nil")
		}
	})
}

func TestLoadBaseConfigFromEnv(t *testing.T) {
	_ = os.Setenv("HTTP_PORT", "8888")
	_ = os.Setenv("DB_NAME", "custom_db")
	defer func() {
		_ = os.Unsetenv("HTTP_PORT")
		_ = os.Unsetenv("DB_NAME")
	}()

	cfg, err := config.LoadBaseConfig("unit-test-service", "default_db", 8080, 9080)
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	if cfg.HTTPPort != 8888 {
		t.Errorf("expected HTTP_PORT 8888, got %d", cfg.HTTPPort)
	}
	if cfg.Database.DBName != "custom_db" {
		t.Errorf("expected DB_NAME 'custom_db', got %s", cfg.Database.DBName)
	}
}
