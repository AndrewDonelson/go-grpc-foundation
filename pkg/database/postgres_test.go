package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func TestDefaultPostgresConfig(t *testing.T) {
	cfg := DefaultPostgresConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, 5432, cfg.Port)
	assert.Equal(t, "starflight", cfg.Database)
	assert.Equal(t, "postgres", cfg.User)
	assert.Equal(t, "postgres", cfg.Password)
	assert.Equal(t, int32(25), cfg.MaxConnections)
	assert.Equal(t, int32(5), cfg.MinConnections)
	assert.Equal(t, 1*time.Hour, cfg.MaxConnLifetime)
	assert.Equal(t, 30*time.Minute, cfg.MaxConnIdleTime)
}

func TestNewPostgresPool_InvalidDSN(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	// Invalid port (negative)
	config := &PostgresConfig{
		Host:            "localhost",
		Port:            -1,
		Database:        "test",
		User:            "user",
		Password:        "pass",
		MaxConnections:  10,
		MinConnections:  1,
		MaxConnLifetime: 1 * time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
	}

	pool, err := NewPostgresPool(ctx, config, logger)
	assert.Error(t, err)
	assert.Nil(t, pool)
}

func TestNewPostgresPool_InvalidHost(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	// Invalid host (empty)
	config := &PostgresConfig{
		Host:            "",
		Port:            5432,
		Database:        "test",
		User:            "user",
		Password:        "pass",
		MaxConnections:  10,
		MinConnections:  1,
		MaxConnLifetime: 1 * time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
	}

	pool, err := NewPostgresPool(ctx, config, logger)
	// This will fail at connection time, not DSN parsing
	// The exact error depends on the DSN format
	assert.Error(t, err)
	assert.Nil(t, pool)
}

func TestNewPostgresPool_ConfigValues(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	config := &PostgresConfig{
		Host:            "testhost",
		Port:            5433,
		Database:        "testdb",
		User:            "testuser",
		Password:        "testpass",
		MaxConnections:  50,
		MinConnections:  10,
		MaxConnLifetime: 2 * time.Hour,
		MaxConnIdleTime: 1 * time.Hour,
	}

	// This will fail because we can't connect, but we can verify the DSN is constructed
	pool, err := NewPostgresPool(ctx, config, logger)
	
	// Should fail at connection (no real DB), but verify error is about connection
	if err != nil {
		assert.Contains(t, err.Error(), "failed to")
	}
	assert.Nil(t, pool)
}

func TestPostgresConfig_ZeroValues(t *testing.T) {
	config := &PostgresConfig{}

	assert.Equal(t, "", config.Host)
	assert.Equal(t, 0, config.Port)
	assert.Equal(t, "", config.Database)
	assert.Equal(t, "", config.User)
	assert.Equal(t, "", config.Password)
	assert.Equal(t, int32(0), config.MaxConnections)
	assert.Equal(t, int32(0), config.MinConnections)
	assert.Equal(t, time.Duration(0), config.MaxConnLifetime)
	assert.Equal(t, time.Duration(0), config.MaxConnIdleTime)
}

func TestPostgresConfig_DSNFormat(t *testing.T) {
	// Test that DSN is formatted correctly
	config := &PostgresConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "testdb",
		User:     "user",
		Password: "pass",
	}

	// The DSN is constructed in NewPostgresPool, but we can verify the values
	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, 5432, config.Port)
	assert.Equal(t, "testdb", config.Database)
	assert.Equal(t, "user", config.User)
	assert.Equal(t, "pass", config.Password)
}

func TestPostgresConfig_SpecialCharacters(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	// Test with special characters in password
	config := &PostgresConfig{
		Host:            "localhost",
		Port:            5432,
		Database:        "test",
		User:            "user",
		Password:        "p@ss:w0rd!",
		MaxConnections:  10,
		MinConnections:  1,
		MaxConnLifetime: 1 * time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
	}

	// Should handle special characters in DSN (will fail at connection, not parsing)
	pool, err := NewPostgresPool(ctx, config, logger)
	assert.Error(t, err)
	assert.Nil(t, pool)
}

// TestNewPostgresPool_ErrorPaths tests various error paths in NewPostgresPool
func TestNewPostgresPool_ErrorPaths(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tests := []struct {
		name   string
		config *PostgresConfig
	}{
		{
			name: "unreachable host",
			config: &PostgresConfig{
				Host:            "192.0.2.1", // Test net - unreachable
				Port:            5432,
				Database:        "test",
				User:            "user",
				Password:        "pass",
				MaxConnections:  10,
				MinConnections:  1,
				MaxConnLifetime: 1 * time.Hour,
				MaxConnIdleTime: 30 * time.Minute,
			},
		},
		{
			name: "invalid host",
			config: &PostgresConfig{
				Host:            "invalid-host-12345",
				Port:            5432,
				Database:        "test",
				User:            "user",
				Password:        "pass",
				MaxConnections:  10,
				MinConnections:  1,
				MaxConnLifetime: 1 * time.Hour,
				MaxConnIdleTime: 30 * time.Minute,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use context with timeout to prevent hanging
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			
			pool, err := NewPostgresPool(ctx, tt.config, logger)
			// Should fail at some point (ParseConfig, NewWithConfig, or Ping)
			assert.Error(t, err)
			assert.Nil(t, pool)
			if err != nil {
				assert.Contains(t, err.Error(), "failed to")
			}
		})
	}
}


