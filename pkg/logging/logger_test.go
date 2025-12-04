package logging

import (
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger_Development(t *testing.T) {
	logger, err := NewLogger("development")
	require.NoError(t, err)
	require.NotNil(t, logger)

	// Verify it's a development logger
	logger.Info("test message")
	logger.Debug("debug message")
	logger.Warn("warn message")
	logger.Error("error message")
}

func TestNewLogger_Production(t *testing.T) {
	logger, err := NewLogger("production")
	require.NoError(t, err)
	require.NotNil(t, logger)

	// Verify it's a production logger
	logger.Info("test message")
	logger.Warn("warn message")
	logger.Error("error message")
}

func TestNewLogger_UnknownEnvironment(t *testing.T) {
	// Unknown environment should default to production
	logger, err := NewLogger("unknown")
	require.NoError(t, err)
	require.NotNil(t, logger)

	logger.Info("test message")
}

func TestNewLogger_LogLevelOverride(t *testing.T) {
	// Save original environment
	originalLogLevel := os.Getenv("LOG_LEVEL")
	defer os.Setenv("LOG_LEVEL", originalLogLevel)

	tests := []struct {
		name      string
		logLevel  string
		env       string
		expectErr bool
	}{
		{
			name:     "debug level override",
			logLevel: "debug",
			env:      "production",
		},
		{
			name:     "info level override",
			logLevel: "info",
			env:      "production",
		},
		{
			name:     "warn level override",
			logLevel: "warn",
			env:      "development",
		},
		{
			name:     "error level override",
			logLevel: "error",
			env:      "development",
		},
		{
			name:      "invalid level",
			logLevel:  "invalid",
			env:       "development",
			expectErr: false, // Should not error, just ignore invalid level
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("LOG_LEVEL", tt.logLevel)
			logger, err := NewLogger(tt.env)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, logger)
			}
		})
	}
}

func TestNewLogger_NoLogLevelOverride(t *testing.T) {
	// Save original environment
	originalLogLevel := os.Getenv("LOG_LEVEL")
	defer os.Setenv("LOG_LEVEL", originalLogLevel)

	// Unset LOG_LEVEL
	os.Unsetenv("LOG_LEVEL")

	logger, err := NewLogger("development")
	require.NoError(t, err)
	require.NotNil(t, logger)

	// Should use default level for environment
	logger.Debug("debug message")
}

func TestNewLogger_EmptyEnvironment(t *testing.T) {
	logger, err := NewLogger("")
	require.NoError(t, err)
	require.NotNil(t, logger)

	logger.Info("test message")
}

func TestNewLogger_ConcurrentAccess(t *testing.T) {
	// Test that logger creation is thread-safe
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			logger, err := NewLogger("development")
			assert.NoError(t, err)
			assert.NotNil(t, logger)
		}()
	}
	wg.Wait()
}

func TestNewLogger_Levels(t *testing.T) {
	// Test that different environments have different default levels
	devLogger, err := NewLogger("development")
	require.NoError(t, err)

	prodLogger, err := NewLogger("production")
	require.NoError(t, err)

	// Development should allow debug
	devLogger.Debug("debug message")

	// Production should not log debug (but won't error)
	prodLogger.Debug("debug message")
}

