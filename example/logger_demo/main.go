package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nextpkg/vcfg"
	"github.com/nextpkg/vcfg/plugins/builtins"
)

// AppConfig represents the application configuration
type AppConfig struct {
	// Logger configuration
	Logger builtins.LoggerConfig

	// Application settings
	App struct {
		Name    string `json:"name" yaml:"name" koanf:"name"`
		Version string `json:"version" yaml:"version" koanf:"version"`
		Debug   bool   `json:"debug" yaml:"debug" koanf:"debug"`
	} `json:"app" yaml:"app" koanf:"app"`
}

func main() {
	// Load configuration and start plugins
	cm := vcfg.MustLoad[AppConfig]("config.yaml")

	// Debug: print the loaded configuration
	config := cm.Get()
	fmt.Printf("Logger config: %#v\n", config.Logger)

	if err := cm.EnablePlugins(); err != nil {
		panic(err)
	}
	if err := cm.StartPlugins(context.Background()); err != nil {
		panic(err)
	}

	// Get the logger instance
	logger := builtins.GetLogger()

	// Demonstrate different log levels
	logger.Debug("This is a debug message", "component", "main", "timestamp", time.Now())
	logger.Info("Application starting", "name", cm.Get().App.Name, "version", cm.Get().App.Version)
	logger.Warn("This is a warning message", "reason", "demonstration")

	// Demonstrate structured logging
	logger.Info("User action",
		"user_id", 12345,
		"action", "login",
		"ip", "192.168.1.100",
		"success", true,
		"duration_ms", 150,
	)

	// Demonstrate error logging
	logger.Error("Database connection failed",
		"error", "connection timeout",
		"host", "localhost:5432",
		"retry_count", 3,
	)

	// Demonstrate logging with context
	ctx := slog.String("request_id", "req-123456")
	logger.InfoContext(nil, "Processing request", ctx, "endpoint", "/api/users")

	// Simulate some work
	time.Sleep(2 * time.Second)

	logger.Info("Application shutting down gracefully")

	// Stop the configuration manager (this will stop all plugins)
	cm.CloseWithContext(context.Background())
}
