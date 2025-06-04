package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/nextpkg/vcfg"
)

type Config struct {
	App struct {
		Name string `yaml:"name"`
	} `yaml:"app"`
}

func main() {
	// Initialize logger
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	slog.Info("Starting context demo application")

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize configuration manager
	configManager := vcfg.MustLoad[Config]("config.yaml")
	defer func() {
		if err := configManager.CloseWithContext(ctx); err != nil {
			slog.Error("Failed to close config manager with context", "error", err)
		}
	}()

	// Test MustEnableAndStartPluginWithContext
	configManager.MustEnableAndStartPluginWithContext(ctx)

	slog.Info("All plugins started with context successfully")

	// Simulate some work
	time.Sleep(2 * time.Second)

	slog.Info("Application shutting down gracefully")
}
