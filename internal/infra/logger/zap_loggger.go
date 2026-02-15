package logger

import (
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var globalLogger *zap.Logger
var opensearchCoreInstance *opensearchCore

func Init(core zapcore.Core, options ...zap.Option) {
	globalLogger = zap.New(core, options...)
}

// InitWithConfig инициализирует логгер с конфигурацией для production или development
func InitWithConfig(level, env string) error {
	var config zap.Config

	if env == "production" {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
	}

	// Устанавливаем уровень логирования
	parsedLevel, err := zapcore.ParseLevel(strings.ToLower(level))
	if err != nil {
		parsedLevel = zapcore.InfoLevel
	}
	config.Level = zap.NewAtomicLevelAt(parsedLevel)

	// Для production используем JSON формат, для development - консольный
	if env == "production" {
		config.Encoding = "json"
		config.OutputPaths = []string{"stdout"}
		config.ErrorOutputPaths = []string{"stderr"}
	} else {
		config.Encoding = "console"
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	logger, err := config.Build(
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return err
	}

	globalLogger = logger
	return nil
}

// AddOpenSearchCore adds an OpenSearch core to the global logger via Tee.
// Must be called after InitWithConfig.
func AddOpenSearchCore(level zapcore.Level, endpoint, index string, batchSize, flushIntervalSec int) {
	if globalLogger == nil {
		return
	}

	osCore := NewOpenSearchCore(level, endpoint, index, batchSize, flushIntervalSec).(*opensearchCore)
	opensearchCoreInstance = osCore

	existing := globalLogger.Core()
	tee := zapcore.NewTee(existing, osCore)
	globalLogger = globalLogger.WithOptions(zap.WrapCore(func(zapcore.Core) zapcore.Core {
		return tee
	}))
}

// Sync синхронизирует буферы логгера
func Sync() error {
	if opensearchCoreInstance != nil {
		opensearchCoreInstance.Stop()
	}
	if globalLogger != nil {
		return globalLogger.Sync()
	}
	return nil
}

func Debug(msg string, fields ...zap.Field) {
	globalLogger.Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	globalLogger.Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	globalLogger.Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	globalLogger.Error(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	globalLogger.Fatal(msg, fields...)
}

func WithOptions(opts ...zap.Option) *zap.Logger {
	return globalLogger.WithOptions(opts...)
}
