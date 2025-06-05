package slogs

import (
	"log/slog"
	"os"
	"sync/atomic"
)

var logger atomic.Value

func init() {
	logger.Store(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))
}

func Logger() *slog.Logger {
	return logger.Load().(*slog.Logger)
}

func SetLevel(level slog.Level) {
	slog.NewLogLogger(Logger().Handler(), level)
}

func Error(msg string, args ...any) {
	Logger().Error(msg, args...)
}

func Info(msg string, args ...any) {
	Logger().Info(msg, args...)
}

func Debug(msg string, args ...any) {
	Logger().Debug(msg, args...)
}

func Warn(msg string, args ...any) {
	Logger().Warn(msg, args...)
}
