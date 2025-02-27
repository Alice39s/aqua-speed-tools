package utils

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	IsDebug bool
	logger  *zap.Logger
)

// initLogger initializes the logger with proper configuration
func initLogger() *zap.Logger {
	var config zap.Config
	if IsDebug {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		config.Development = true
		config.Sampling = nil // 禁用采样以显示所有日志
	} else {
		config = zap.NewProductionConfig()
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	l, err := config.Build(
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		// 如果初始化失败，使用基本配置
		return zap.NewExample()
	}
	return l
}

func init() {
	logger = initLogger()
}

// Debug logs a debug message with structured context
func Debug(msg string, fields ...zapcore.Field) {
	if IsDebug {
		logger.Debug(msg, fields...)
	}
}

// DebugRequest logs an HTTP request details
func DebugRequest(method, url string, headers map[string]string) {
	if IsDebug {
		fields := []zapcore.Field{
			zap.String("method", method),
			zap.String("url", url),
		}
		if len(headers) > 0 {
			fields = append(fields, zap.Any("headers", headers))
		}
		logger.Debug("HTTP Request", fields...)
	}
}

// DebugResponse logs an HTTP response details
func DebugResponse(statusCode int, url string, responseBody string) {
	if IsDebug {
		logger.Debug("HTTP Response",
			zap.Int("status", statusCode),
			zap.String("url", url),
			zap.String("body", responseBody),
		)
	}
}

// Info logs an info message with structured context
func Info(msg string, fields ...zapcore.Field) {
	logger.Info(msg, fields...)
}

// Warn logs a warning message with structured context
func Warn(msg string, fields ...zapcore.Field) {
	logger.Warn(msg, fields...)
}

// Error logs an error message with structured context
func Error(msg string, fields ...zapcore.Field) {
	logger.Error(msg, fields...)
}

// Fatal logs a fatal message with structured context and then exits
func Fatal(msg string, fields ...zapcore.Field) {
	logger.Fatal(msg, fields...)
}

// SetLogger allows setting a custom logger
func SetLogger(l *zap.Logger) {
	if l != nil {
		logger = l
	}
}

// GetLogger returns the current logger instance
func GetLogger() *zap.Logger {
	return logger
}

// ResetLogger reinitializes the logger
func ResetLogger() {
	logger = initLogger()
}

// 为了向后兼容，保留旧的格式化函数
func LogDebug(format string, args ...any) {
	if IsDebug {
		logger.Debug(fmt.Sprintf(format, args...))
	}
}

func LogInfo(format string, args ...any) {
	logger.Info(fmt.Sprintf(format, args...))
}

func LogSuccess(format string, args ...any) {
	logger.Info(fmt.Sprintf("[SUCCESS] "+format, args...))
}

func LogWarning(format string, args ...any) {
	logger.Warn(fmt.Sprintf(format, args...))
}

func LogError(format string, args ...any) {
	logger.Error(fmt.Sprintf(format, args...))
}
