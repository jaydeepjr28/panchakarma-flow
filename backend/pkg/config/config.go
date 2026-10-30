package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// BaseConfig contains common infrastructure parameters shared across all microservices.
type BaseConfig struct {
	AppName        string        `json:"app_name"`
	Environment    string        `json:"environment"` // "development", "staging", "production"
	HTTPPort       int           `json:"http_port"`
	GRPCPort       int           `json:"grpc_port"`
	LogLevel       string        `json:"log_level"`
	AllowedOrigins []string      `json:"allowed_origins"`
	Database       DatabaseConfig `json:"database"`
	Redis          RedisConfig    `json:"redis"`
	Kafka          KafkaConfig    `json:"kafka"`
	Auth           AuthConfig     `json:"auth"`
}

// AuthConfig holds JWT and security authentication configurations.
type AuthConfig struct {
	JWTSecret  string        `json:"-"`
	AccessTTL  time.Duration `json:"access_ttl"`
	RefreshTTL time.Duration `json:"refresh_ttl"`
}

// DatabaseConfig holds PostgreSQL connection parameters.
type DatabaseConfig struct {
	Host               string        `json:"host"`
	Port               int           `json:"port"`
	User               string        `json:"user"`
	Password           string        `json:"-"`
	DBName             string        `json:"db_name"`
	SSLMode            string        `json:"ssl_mode"`
	MaxOpenConns       int           `json:"max_open_conns"`
	MaxIdleConns       int           `json:"max_idle_conns"`
	ConnMaxLifetime    time.Duration `json:"conn_max_lifetime"`
	ConnMaxIdleTime    time.Duration `json:"conn_max_idle_time"`
}

// DSN returns the formatted PostgreSQL connection string.
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.SSLMode)
}

// RedisConfig holds Redis connection parameters.
type RedisConfig struct {
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	Password     string        `json:"-"`
	DB           int           `json:"db"`
	DialTimeout  time.Duration `json:"dial_timeout"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	PoolSize     int           `json:"pool_size"`
}

// Addr returns the Redis host:port address string.
func (r *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// KafkaConfig holds Apache Kafka connection parameters.
type KafkaConfig struct {
	Brokers       []string `json:"brokers"`
	ClientID      string   `json:"client_id"`
	ConsumerGroup string   `json:"consumer_group"`
}

// LoadBaseConfig loads configuration for a given microservice from the environment.
func LoadBaseConfig(appName, defaultDBName string, defaultHTTPPort, defaultGRPCPort int) (*BaseConfig, error) {
	// Attempt loading .env if present (non-fatal if absent)
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")
	_ = godotenv.Load("../../.env")

	cfg := &BaseConfig{
		AppName:        getEnv("APP_NAME", appName),
		Environment:    getEnv("APP_ENV", "development"),
		HTTPPort:       getEnvInt("HTTP_PORT", defaultHTTPPort),
		GRPCPort:       getEnvInt("GRPC_PORT", defaultGRPCPort),
		LogLevel:       getEnv("LOG_LEVEL", "info"),
		AllowedOrigins: strings.Split(getEnv("ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:3000,http://127.0.0.1:5173"), ","),
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnvInt("DB_PORT", 5440),
			User:            getEnv("DB_USER", "panchakarma"),
			Password:        getEnv("DB_PASSWORD", "panchakarma_secret_2026"),
			DBName:          getEnv("DB_NAME", defaultDBName),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: time.Duration(getEnvInt("DB_CONN_MAX_LIFETIME_MINS", 30)) * time.Minute,
			ConnMaxIdleTime: time.Duration(getEnvInt("DB_CONN_MAX_IDLE_MINS", 10)) * time.Minute,
		},
		Redis: RedisConfig{
			Host:         getEnv("REDIS_HOST", "localhost"),
			Port:         getEnvInt("REDIS_PORT", 6379),
			Password:     getEnv("REDIS_PASSWORD", ""),
			DB:           getEnvInt("REDIS_DB", 0),
			DialTimeout:  time.Duration(getEnvInt("REDIS_DIAL_TIMEOUT_MS", 3000)) * time.Millisecond,
			ReadTimeout:  time.Duration(getEnvInt("REDIS_READ_TIMEOUT_MS", 2000)) * time.Millisecond,
			WriteTimeout: time.Duration(getEnvInt("REDIS_WRITE_TIMEOUT_MS", 2000)) * time.Millisecond,
			PoolSize:     getEnvInt("REDIS_POOL_SIZE", 20),
		},
		Kafka: KafkaConfig{
			Brokers:       strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
			ClientID:      getEnv("KAFKA_CLIENT_ID", appName),
			ConsumerGroup: getEnv("KAFKA_CONSUMER_GROUP", fmt.Sprintf("%s-group", appName)),
		},
		Auth: AuthConfig{
			JWTSecret:  getEnv("JWT_SECRET", "panchakarma_flow_jwt_super_secret_signing_key_2026_at_least_32_bytes_long"),
			AccessTTL:  time.Duration(getEnvInt("JWT_ACCESS_TTL_MINS", 15)) * time.Minute,
			RefreshTTL: time.Duration(getEnvInt("JWT_REFRESH_TTL_HOURS", 168)) * time.Hour,
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed for %s: %w", appName, err)
	}

	return cfg, nil
}

// Validate ensures all mandatory configuration values are sound and actionable.
func (c *BaseConfig) Validate() error {
	var errs []string

	if c.Auth.JWTSecret != "" && len(c.Auth.JWTSecret) < 32 {
		errs = append(errs, "JWT_SECRET must be at least 32 characters long")
	}
	if c.HTTPPort <= 0 || c.HTTPPort > 65535 {
		errs = append(errs, fmt.Sprintf("invalid HTTP_PORT %d: must be between 1 and 65535", c.HTTPPort))
	}
	if c.GRPCPort < 0 || c.GRPCPort > 65535 {
		errs = append(errs, fmt.Sprintf("invalid GRPC_PORT %d: must be between 0 and 65535", c.GRPCPort))
	}
	if c.Database.Host == "" {
		errs = append(errs, "DB_HOST cannot be empty")
	}
	if c.Database.DBName == "" {
		errs = append(errs, "DB_NAME cannot be empty")
	}
	if c.Database.User == "" {
		errs = append(errs, "DB_USER cannot be empty")
	}
	if c.Redis.Host == "" {
		errs = append(errs, "REDIS_HOST cannot be empty")
	}
	if len(c.Kafka.Brokers) == 0 || c.Kafka.Brokers[0] == "" {
		errs = append(errs, "KAFKA_BROKERS must contain at least one broker address")
	}

	if len(errs) > 0 {
		return fmt.Errorf("missing or invalid configuration:\n - %s", strings.Join(errs, "\n - "))
	}
	return nil
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if valStr, ok := os.LookupEnv(key); ok && valStr != "" {
		if val, err := strconv.Atoi(valStr); err == nil {
			return val
		}
	}
	return defaultVal
}
