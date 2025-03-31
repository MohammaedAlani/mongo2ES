package logger

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// InitLogger sets up structured logging with rotation
func InitLogger(logPath string) (*zap.SugaredLogger, error) {
	// Ensure log directory exists
	if logPath != "" {
		// Extract directory part of the path
		lastSlash := strings.LastIndex(logPath, "/")
		if lastSlash > 0 {
			logDir := logPath[:lastSlash]
			err := os.MkdirAll(logDir, 0755)
			if err != nil {
				return nil, fmt.Errorf("failed to create log directory: %w", err)
			}
		}
	}

	// Configure file rotation
	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    100, // MB
		MaxBackups: 10,
		MaxAge:     30, // days
		Compress:   true,
	})

	// Also log to console
	consoleWriter := zapcore.AddSync(os.Stdout)

	// Create encoder configs
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Create multi-writer core
	core := zapcore.NewTee(
		zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			fileWriter,
			zapcore.InfoLevel,
		),
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
			consoleWriter,
			zapcore.InfoLevel,
		),
	)

	// Create logger
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	return logger.Sugar(), nil
}
