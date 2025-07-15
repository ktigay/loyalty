package log

import (
	"bytes"
	"log/slog"
	"os"
)

var (
	buff       bytes.Buffer
	MockLogger = slog.New(slog.NewJSONHandler(&buff, &slog.HandlerOptions{}))
)

// New Конструктор.
func New(lvl string) *slog.Logger {
	opts := slog.HandlerOptions{
		Level: mapLvlFromStr(lvl),
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &opts))
	slog.SetDefault(logger)

	return logger
}

func mapLvlFromStr(lvl string) slog.Level {
	switch lvl {
	case "debug":
		return slog.LevelDebug
	case "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
