package updater

import (
	"go.uber.org/zap"
)

// InitLogger initializes and returns a zap.Logger instance.
// It falls back to a default logger in case of initialization failure.
func InitLogger() *zap.Logger {
	logger, err := zap.NewProduction()
	if err != nil {
		// Fallback to a no-op logger to prevent nil pointer dereference
		return zap.NewNop()
	}
	return logger
}
