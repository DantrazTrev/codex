package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger creates a new zap logger
func NewLogger(verbose bool) (*zap.Logger, error) {
	var config zap.Config

	if verbose {
		config = zap.NewDevelopmentConfig()
		config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	// Customize output format
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return logger, nil
}

// NewSilentLogger creates a no-op logger
func NewSilentLogger() *zap.Logger {
	return zap.NewNop()
}