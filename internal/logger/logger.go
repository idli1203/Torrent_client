package logger

import (
	"log/slog"
	"os"
)

var defaultLogger *slog.Logger

// Init initializes the logger with the specified output file
// If logFile is empty, logs go to stderr
func Init(logFile string) error {
	var handler slog.Handler

	if logFile != "" {
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}
		handler = slog.NewJSONHandler(file, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	} else {
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}

	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
	return nil
}

// Debug logs at debug level
func Debug(msg string, args ...any) {
	slog.Debug(msg, args...)
}

// Info logs at info level
func Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

// Warn logs at warn level
func Warn(msg string, args ...any) {
	slog.Warn(msg, args...)
}

// Error logs at error level
func Error(msg string, args ...any) {
	slog.Error(msg, args...)
}
