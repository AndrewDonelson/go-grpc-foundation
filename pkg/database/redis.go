package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host         string
	Port         int
	DB           int
	Password     string
	PoolSize     int
	MinIdleConns int
}

// NewRedisClient creates a new Redis client
func NewRedisClient(ctx context.Context, config *RedisConfig, logger *zap.Logger) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	logger.Info("redis client created",
		zap.String("host", config.Host),
		zap.Int("port", config.Port),
		zap.Int("db", config.DB),
	)

	return client, nil
}

// DefaultRedisConfig returns default Redis configuration
func DefaultRedisConfig() *RedisConfig {
	return &RedisConfig{
		Host:         "localhost",
		Port:         6379,
		DB:           0,
		Password:     "",
		PoolSize:     10,
		MinIdleConns: 2,
	}
}

