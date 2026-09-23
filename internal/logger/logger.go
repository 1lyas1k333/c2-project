// Package logger предоставляет централизованное логирование с использованием zap.
// Поддерживает два режима: JSON (для продакшена) и консольный (для разработки).
package logger

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

// Init - инициализирует глобальный логгер.
// Режим определяется переменной окружения LOG_MODE:
//   - "production" — JSON-логи (для Docker)
//   - "development" — читаемые логи (для локальной разработки)
func Init() {
	mode := os.Getenv("LOG_MODE")
	if mode == "" {
		mode = "development"
	}

	var config zap.Config
	if strings.ToLower(mode) == "production" {
		config = zap.NewProductionConfig()
		config.Encoding = "json"
	} else {
		config = zap.NewDevelopmentConfig()
		config.Encoding = "console"
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Настраиваем вывод
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	var err error
	log, err = config.Build()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
}

// Get - возвращает глобальный логгер.
func Get() *zap.Logger {
	if log == nil {
		Init()
	}
	return log
}

// Sync - синхронизирует буферы логов (вызывается при завершении).
func Sync() {
	if log != nil {
		_ = log.Sync()
	}
}

// --- Обёртки для удобства ---

// Info - логирует сообщение уровня Info.
func Info(msg string, fields ...zap.Field) {
	Get().Info(msg, fields...)
}

// Debug - логирует сообщение уровня Debug.
func Debug(msg string, fields ...zap.Field) {
	Get().Debug(msg, fields...)
}

// Warn - логирует сообщение уровня Warn.
func Warn(msg string, fields ...zap.Field) {
	Get().Warn(msg, fields...)
}

// Error - логирует сообщение уровня Error.
func Error(msg string, fields ...zap.Field) {
	Get().Error(msg, fields...)
}

// Fatal - логирует сообщение уровня Fatal и завершает программу.
func Fatal(msg string, fields ...zap.Field) {
	Get().Fatal(msg, fields...)
}

// --- Хелперы для часто используемых полей ---

// String - создаёт поле типа string.
func String(key, value string) zap.Field {
	return zap.String(key, value)
}

// Int - создаёт поле типа int.
func Int(key string, value int) zap.Field {
	return zap.Int(key, value)
}

// Err - создаёт поле для ошибки.
func Err(err error) zap.Field {
	return zap.Error(err)
}
