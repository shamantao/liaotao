// main.go — Application entry point
// Generated from tao-init v1.0.0 (wails-go adapter)

package main

import (
	"embed"
	"log"

	"liaotao/internal/config"
	"liaotao/internal/logger"
	"liaotao/internal/paths"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed frontend/dist
var assets embed.FS

func main() {
	// 1. Load merged config (default → user → project → env)
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// 2. Initialize logger (console + JSON file)
	appLogger, err := logger.Init(&cfg.Logger)
	if err != nil {
		log.Fatalf("logger: %v", err)
	}

	// 3. Build and validate runtime paths
	runtimePaths, err := paths.Build(cfg)
	if err != nil {
		log.Fatalf("paths: %v", err)
	}

	appLogger.Info("startup",
		"app", cfg.App.Name,
		"version", cfg.App.Version,
		"mode", cfg.App.Mode,
		"logs", runtimePaths.LogsDir,
	)

	// 4. Launch Wails application
	app := application.New(application.Options{
		Name:        cfg.App.Name,
		Description: "liaotao",
		Assets: application.AssetOptions{
			FS: assets,
		},
	})

	app.NewWebviewWindowWithOptions(application.WebviewWindowOptions{
		Title:  cfg.App.Name,
		Width:  1200,
		Height: 800,
	})

	if err := app.Run(); err != nil {
		appLogger.Error("application exited with error", "error", err)
	}
}
