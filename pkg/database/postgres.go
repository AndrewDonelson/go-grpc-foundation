package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// PostgresConfig holds PostgreSQL configuration
type PostgresConfig struct {
	Host            string
	Port            int
	Database        string
	User            string
	Password        string
	MaxConnections  int32
	MinConnections  int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// NewPostgresPool creates a new PostgreSQL connection pool
func NewPostgresPool(ctx context.Context, config *PostgresConfig, logger *zap.Logger) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
	)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pool config: %w", err)
	}

	// Configure pool
	poolConfig.MaxConns = config.MaxConnections
	poolConfig.MinConns = config.MinConnections
	poolConfig.MaxConnLifetime = config.MaxConnLifetime
	poolConfig.MaxConnIdleTime = config.MaxConnIdleTime

	// Create pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("postgres connection pool created",
		zap.String("host", config.Host),
		zap.Int("port", config.Port),
		zap.String("database", config.Database),
		zap.Int32("max_conns", config.MaxConnections),
	)

	return pool, nil
}

// DefaultPostgresConfig returns default PostgreSQL configuration
func DefaultPostgresConfig() *PostgresConfig {
	return &PostgresConfig{
		Host:            "localhost",
		Port:            5432,
		Database:        "starflight",
		User:            "postgres",
		Password:        "postgres",
		MaxConnections:  25,
		MinConnections:  5,
		MaxConnLifetime: 1 * time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
	}
}

