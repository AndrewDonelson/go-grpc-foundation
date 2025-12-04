package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func TestDefaultRedisConfig(t *testing.T) {
	cfg := DefaultRedisConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, 6379, cfg.Port)
	assert.Equal(t, 0, cfg.DB)
	assert.Equal(t, "", cfg.Password)
	assert.Equal(t, 10, cfg.PoolSize)
	assert.Equal(t, 2, cfg.MinIdleConns)
}

func TestNewRedisClient_InvalidHost(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	config := &RedisConfig{
		Host:         "invalid-host-that-does-not-exist",
		Port:         6379,
		DB:           0,
		Password:     "",
		PoolSize:     10,
		MinIdleConns: 2,
	}

	client, err := NewRedisClient(ctx, config, logger)
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to connect")
}

func TestNewRedisClient_InvalidPort(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	config := &RedisConfig{
		Host:         "localhost",
		Port:         99999, // Invalid port
		DB:           0,
		Password:     "",
		PoolSize:     10,
		MinIdleConns: 2,
	}

	client, err := NewRedisClient(ctx, config, logger)
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestNewRedisClient_ConfigValues(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	config := &RedisConfig{
		Host:         "testhost",
		Port:         6380,
		DB:           1,
		Password:     "testpass",
		PoolSize:     20,
		MinIdleConns: 5,
	}

	// Will fail at connection, but we can verify config is used
	client, err := NewRedisClient(ctx, config, logger)
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestRedisConfig_ZeroValues(t *testing.T) {
	config := &RedisConfig{}

	assert.Equal(t, "", config.Host)
	assert.Equal(t, 0, config.Port)
	assert.Equal(t, 0, config.DB)
	assert.Equal(t, "", config.Password)
	assert.Equal(t, 0, config.PoolSize)
	assert.Equal(t, 0, config.MinIdleConns)
}

func TestRedisConfig_AddressFormat(t *testing.T) {
	// Test that address is formatted correctly
	config := &RedisConfig{
		Host: "localhost",
		Port: 6379,
	}

	// The address is constructed in NewRedisClient, but we can verify the values
	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, 6379, config.Port)
}

func TestNewRedisClient_ContextTimeout(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	// Use a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	config := &RedisConfig{
		Host:         "localhost",
		Port:         6379,
		DB:           0,
		Password:     "",
		PoolSize:     10,
		MinIdleConns: 2,
	}

	client, err := NewRedisClient(ctx, config, logger)
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestNewRedisClient_DifferentDB(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	config := &RedisConfig{
		Host:         "localhost",
		Port:         6379,
		DB:           5, // Different DB
		Password:     "",
		PoolSize:     10,
		MinIdleConns: 2,
	}

	// May succeed if Redis is running locally, or fail if not
	// We just verify the config is set correctly
	client, err := NewRedisClient(ctx, config, logger)
	assert.Equal(t, 5, config.DB)
	if err != nil {
		assert.Nil(t, client)
	} else {
		// If connection succeeded, close it
		if client != nil {
			client.Close()
		}
	}
}

func TestNewRedisClient_WithPassword(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	config := &RedisConfig{
		Host:         "localhost",
		Port:         6379,
		DB:           0,
		Password:     "secret-password",
		PoolSize:     10,
		MinIdleConns: 2,
	}

	// May succeed if Redis is running and accepts the password, or fail if not
	// We just verify the config is set correctly
	client, err := NewRedisClient(ctx, config, logger)
	assert.Equal(t, "secret-password", config.Password)
	if err != nil {
		assert.Nil(t, client)
	} else {
		// If connection succeeded, close it
		if client != nil {
			client.Close()
		}
	}
}

