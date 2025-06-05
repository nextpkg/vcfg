package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/nextpkg/vcfg"
)

//go:generate go run config.go

// main demonstrates basic plugin loading functionality using MustInit method
// It supports multiple Kafka plugin instances based on configuration blocks
func main() {
	// Initialize logger
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	slog.Info("Starting plugin demo application")

	// Initialize configuration manager using MustLoad method
	// This will load configuration from config.yaml file
	configManager := vcfg.MustLoad[AppConfig]("config.yaml")
	defer func() {
		if err := configManager.CloseWithContext(context.Background()); err != nil {
			slog.Error("Failed to close config manager", "error", err)
		}
	}()

	// Get current configuration
	config := configManager.Get()
	if config == nil {
		slog.Error("Failed to get configuration")
		return
	}

	slog.Info("Configuration loaded successfully")
	slog.Debug("Kafka Producer Config",
		"bootstrap_servers", config.KafkaProducer.BootstrapServers,
		"topic", config.KafkaProducer.Topic)
	slog.Debug("Kafka Consumer Config",
		"bootstrap_servers", config.KafkaConsumer.BootstrapServers,
		"topic", config.KafkaConsumer.Topic)
	slog.Debug("Kafka Stream Config",
		"bootstrap_servers", config.KafkaStream.BootstrapServers,
		"topic", config.KafkaStream.Topic)

	// Enable plugins based on configuration
	// This will create plugin instances for each Kafka configuration block
	if err := configManager.EnablePlugins(); err != nil {
		slog.Error("Failed to enable plugins", "error", err)
		return
	}

	slog.Info("Plugins enabled successfully")

	// Start all registered plugins
	if err := configManager.StartPlugins(context.Background()); err != nil {
		slog.Error("Failed to start plugins", "error", err)
		return
	}

	slog.Info("All plugins started successfully")

	// Enable configuration watching for hot reload
	configManager.EnableWatch()
	slog.Info("Configuration watching enabled")

	// Keep the application running for a shorter time to allow external config changes
	slog.Info("Application will run for 10 seconds to allow configuration testing...")
	time.Sleep(10 * time.Second)
	slog.Info("Demo completed, shutting down application")
}
