package updater

import (
	"aqua-speed-tools/internal/utils"

	"go.uber.org/zap"
)

// InitLogger initializes and returns a zap.Logger instance.
// It falls back to a default logger in case of initialization failure.
func InitLogger() *zap.Logger {
	return utils.GetLogger()
}
