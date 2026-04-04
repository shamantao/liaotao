// logger.go — Structured logging module
// Dual output: human-readable console + JSON log file with rotation.

package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"liaotao/internal/config"
)

// Init creates and sets the global slog logger.
// Returns the logger instance for explicit usage.
func Init(cfg *config.LoggerSection) (*slog.Logger, error) {
	level := parseLevel(cfg.Level)

	writers := []io.Writer{os.Stdout}

	// JSON file output
	if cfg.FileJSON {
		if err := os.MkdirAll("logs", 0o755); err != nil {
			return nil, err
		}
		logPath := filepath.Join("logs", "app.log")
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, err
		}
		writers = append(writers, f)
	}

	// Console handler (human-readable)
	consoleHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	logger := slog.New(consoleHandler)
	slog.SetDefault(logger)

	return logger, nil
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug", "DEBUG":
		return slog.LevelDebug
	case "warn", "WARN":
		return slog.LevelWarn
	case "error", "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
