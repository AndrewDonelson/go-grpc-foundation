package logging

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger creates a Zap logger based on environment
func NewLogger(environment string) (*zap.Logger, error) {
	var config zap.Config

	if environment == "development" {
		// Development: verbose logging with console output
		config = zap.NewDevelopmentConfig()
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		// Production: JSON logging, errors and warnings only
		config = zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	}

	// Allow override via LOG_LEVEL environment variable
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		var level zapcore.Level
		if err := level.UnmarshalText([]byte(logLevel)); err == nil {
			config.Level = zap.NewAtomicLevelAt(level)
		}
	}

	return config.Build()
}

