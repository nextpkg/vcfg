package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/nextpkg/vcfg"
	"github.com/nextpkg/vcfg/plugins"
)

// AppConfig represents the main application configuration
type AppConfig struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Kafka    KafkaConfig    `yaml:"kafka"`
	Redis    RedisConfig    `yaml:"redis"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

func main() {
	// Set up structured logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	slog.Info("Registered global plugins", "plugins", plugins.ListGlobalPlugins())

	// Create configuration manager
	cm := vcfg.New[AppConfig]("config.yaml")

	// Get initial configuration (already loaded by New)
	config := cm.Get()
	if config == nil {
		log.Fatalf("Failed to get configuration")
	}

	// Enable watch for configuration changes
	cm.EnableWatch()

	slog.Info("Initial configuration loaded",
		"kafka_brokers", config.Kafka.Brokers,
		"kafka_topic", config.Kafka.Topic,
		"redis_host", config.Redis.Host,
		"redis_port", config.Redis.Port)

	// Start all plugins
	ctx := context.Background()
	if err := cm.StartAllPlugins(ctx); err != nil {
		log.Fatalf("Failed to start plugins: %v", err)
	}

	slog.Info("All plugins started successfully")
	slog.Info("Smart configuration watching enabled - modify config.yaml to see automatic plugin reloads")
	slog.Info("Only plugins whose configurations actually change will be reloaded")

	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	<-sigChan
	slog.Info("Shutdown signal received, stopping all plugins...")

	// Stop all plugins
	if err := cm.StopAllPlugins(ctx); err != nil {
		slog.Error("Error stopping plugins", "error", err)
	}

	// Close configuration manager
	if err := cm.Close(); err != nil {
		slog.Error("Error closing configuration manager", "error", err)
	}

	slog.Info("Application shutdown complete")
}
