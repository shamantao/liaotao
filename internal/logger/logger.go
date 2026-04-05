// logger.go — Structured logging module
// Dual output: human-readable console + JSON log file with rotation.
// Debug level is automatically forced for pre-release builds (v0.x) and debug mode.

package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"liaotao/internal/config"
)

// Init creates and sets the global slog logger.
// Debug level is forced automatically when appMode is "debug" or appVersion starts with "0.".
// Returns the logger instance for explicit usage.
func Init(cfg *config.LoggerSection, logsDir, appMode, appVersion string) (*slog.Logger, error) {
	level := effectiveLevel(cfg, appMode, appVersion)

	if logsDir == "" {
		logsDir = "logs"
	}

	consoleWriter := io.Writer(os.Stdout)
	jsonWriter := io.Writer(nil)

	// JSON file output
	if cfg.FileJSON {
		if err := os.MkdirAll(logsDir, 0o755); err != nil {
			return nil, err
		}
		logPath := filepath.Join(logsDir, "app.log")
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, err
		}
		jsonWriter = f
	}

	// Console handler (human-readable)
	consoleHandler := slog.NewTextHandler(consoleWriter, &slog.HandlerOptions{Level: level})
	var handler slog.Handler = consoleHandler

	// File handler (JSON) is enabled only when configured.
	if jsonWriter != nil {
		fileHandler := slog.NewJSONHandler(jsonWriter, &slog.HandlerOptions{Level: level})
		handler = multiHandler{handlers: []slog.Handler{consoleHandler, fileHandler}}
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger, nil
}

// multiHandler fans out each record to multiple handlers.
type multiHandler struct {
	handlers []slog.Handler
}

func (m multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m multiHandler) Handle(ctx context.Context, record slog.Record) error {
	for _, h := range m.handlers {
		if !h.Enabled(ctx, record.Level) {
			continue
		}
		if err := h.Handle(ctx, record); err != nil {
			return err
		}
	}
	return nil
}

func (m multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := make([]slog.Handler, 0, len(m.handlers))
	for _, h := range m.handlers {
		next = append(next, h.WithAttrs(attrs))
	}
	return multiHandler{handlers: next}
}

func (m multiHandler) WithGroup(name string) slog.Handler {
	next := make([]slog.Handler, 0, len(m.handlers))
	for _, h := range m.handlers {
		next = append(next, h.WithGroup(name))
	}
	return multiHandler{handlers: next}
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

// effectiveLevel returns the active log level, forcing debug for pre-release builds.
func effectiveLevel(cfg *config.LoggerSection, appMode, appVersion string) slog.Level {
	if appMode == "debug" || strings.HasPrefix(appVersion, "0.") {
		return slog.LevelDebug
	}
	return parseLevel(cfg.Level)
}
