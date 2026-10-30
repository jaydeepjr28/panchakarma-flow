package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jaydeepjr28/panchakarma-flow/pkg/config"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/logger"
	goredis "github.com/redis/go-redis/v9"
)

// Client wraps go-redis client with graceful failure and convenience locking methods.
type Client struct {
	*goredis.Client
	cfg *config.RedisConfig
}

// Connect initializes the Redis client with configured timeouts and pools.
func Connect(ctx context.Context, cfg *config.RedisConfig) (*Client, error) {
	rdb := goredis.NewClient(&goredis.Options{
		Addr:         cfg.Addr(),
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolSize:     cfg.PoolSize,
	})

	pingCtx, cancel := context.WithTimeout(ctx, cfg.DialTimeout)
	defer cancel()

	if err := rdb.Ping(pingCtx).Err(); err != nil {
		logger.Warn("Redis connection failed (running in degraded/offline mode)",
			slog.String("addr", cfg.Addr()),
			slog.String("error", err.Error()),
		)
		// Return client instance even if ping failed so service can start with degraded caching if configured
		return &Client{Client: rdb, cfg: cfg}, fmt.Errorf("failed to connect to Redis at %s: %w", cfg.Addr(), err)
	}

	logger.Info("Connected to Redis successfully",
		slog.String("addr", cfg.Addr()),
		slog.Int("db", cfg.DB),
		slog.Int("pool_size", cfg.PoolSize),
	)

	return &Client{Client: rdb, cfg: cfg}, nil
}

// PingCheck performs a ping with timeout for readiness probes.
func (c *Client) PingCheck(ctx context.Context) error {
	if c == nil || c.Client == nil {
		return fmt.Errorf("redis client is not initialized")
	}
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return c.Ping(pingCtx).Err()
}

// AcquireHoldLock attempts to atomically acquire a temporary slot hold lock with TTL.
func (c *Client) AcquireHoldLock(ctx context.Context, key, holdID string, ttl time.Duration) (bool, error) {
	return c.SetNX(ctx, key, holdID, ttl).Result()
}

// ReleaseHoldLock safely releases a hold lock only if the value matches holdID (atomic Lua script).
func (c *Client) ReleaseHoldLock(ctx context.Context, key, holdID string) (bool, error) {
	luaScript := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	res, err := c.Eval(ctx, luaScript, []string{key}, holdID).Result()
	if err != nil {
		return false, err
	}
	if val, ok := res.(int64); ok && val == 1 {
		return true, nil
	}
	return false, nil
}
