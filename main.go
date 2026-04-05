// main.go — Application entry point
// Generated from tao-init v1.0.0 (wails-go adapter)

package main

import (
	"embed"
	"io/fs"
	"log"

	"liaotao/internal/bindings"
	"liaotao/internal/config"
	"liaotao/internal/db"
	"liaotao/internal/logger"
	"liaotao/internal/paths"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed frontend
var assets embed.FS

func main() {
	// 1. Load merged config (default → user → project → env)
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// 2. Build and validate runtime paths
	runtimePaths, err := paths.Build(cfg)
	if err != nil {
		log.Fatalf("paths: %v", err)
	}

	// 3. Initialize logger (console + JSON file, debug forced for v0.x / debug mode)
	appLogger, err := logger.Init(&cfg.Logger, runtimePaths.LogsDir, cfg.App.Mode, cfg.App.Version)
	if err != nil {
		log.Fatalf("logger: %v", err)
	}

	appLogger.Info("startup",
		"app", cfg.App.Name,
		"version", cfg.App.Version,
		"mode", cfg.App.Mode,
		"logs", runtimePaths.LogsDir,
	)

	// 4. Initialize SQLite and run migrations
	database, err := db.OpenAndMigrate(cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer database.Close()

	// 5. Build backend bindings service (chat/providers/settings/conversations)
	bindingService := bindings.NewService(database)
	appLogger.Info("backend services ready", "has_bindings", bindingService != nil)

	frontendFS, err := fs.Sub(assets, "frontend")
	if err != nil {
		log.Fatalf("assets: %v", err)
	}

	// 6. Launch Wails application
	app := application.New(application.Options{
		Name:        cfg.App.Name,
		Description: "liaotao",
		Services: []application.Service{
			application.NewService(bindingService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(frontendFS),
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  cfg.App.Name,
		Width:  1200,
		Height: 800,
	})

	if err := app.Run(); err != nil {
		appLogger.Error("application exited with error", "error", err)
	}
}
