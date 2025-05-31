package main

import (
	"context"
	"fmt"
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

// KafkaConfig represents Kafka configuration and implements plugins.Config
type KafkaConfig struct {
	Brokers []string `yaml:"brokers"`
	Topic   string   `yaml:"topic"`
	GroupID string   `yaml:"group_id"`
}

// Name implements plugins.Config interface
func (k KafkaConfig) Name() string {
	return "kafka"
}

// RedisConfig represents Redis configuration and implements plugins.Config
type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// Name implements plugins.Config interface
func (r RedisConfig) Name() string {
	return "redis"
}

// KafkaPlugin represents a Kafka plugin
type KafkaPlugin struct {
	config KafkaConfig
}

// Name implements plugins.Plugin interface
func (p *KafkaPlugin) Name() string {
	return "kafka"
}

// Start implements plugins.Plugin interface
func (p *KafkaPlugin) Start(config any) error {
	if kafkaConfig, ok := config.(KafkaConfig); ok {
		p.config = kafkaConfig
		slog.Info("Kafka plugin started", "brokers", kafkaConfig.Brokers, "topic", kafkaConfig.Topic)
		return nil
	}
	return fmt.Errorf("invalid kafka config type: %T", config)
}

// Reload implements plugins.Plugin interface
func (p *KafkaPlugin) Reload(config any) error {
	if kafkaConfig, ok := config.(*KafkaConfig); ok {
		p.config = *kafkaConfig
		slog.Info("Kafka plugin reloaded", "brokers", kafkaConfig.Brokers, "topic", kafkaConfig.Topic)
		return nil
	}
	return fmt.Errorf("invalid kafka config type: %T", config)
}

// Stop implements plugins.Plugin interface
func (p *KafkaPlugin) Stop() error {
	slog.Info("Kafka plugin stopped")
	return nil
}

// RedisPlugin represents a Redis plugin
type RedisPlugin struct {
	config RedisConfig
}

// Name implements plugins.Plugin interface
func (p *RedisPlugin) Name() string {
	return "redis"
}

// Start implements plugins.Plugin interface
func (p *RedisPlugin) Start(config any) error {
	if redisConfig, ok := config.(RedisConfig); ok {
		p.config = redisConfig
		slog.Info("Redis plugin started", "host", redisConfig.Host, "port", redisConfig.Port)
		return nil
	}
	return fmt.Errorf("invalid redis config type: %T", config)
}

// Reload implements plugins.Plugin interface
func (p *RedisPlugin) Reload(config any) error {
	if redisConfig, ok := config.(*RedisConfig); ok {
		p.config = *redisConfig
		slog.Info("Redis plugin reloaded", "host", redisConfig.Host, "port", redisConfig.Port)
		return nil
	}
	return fmt.Errorf("invalid redis config type: %T", config)
}

// Stop implements plugins.Plugin interface
func (p *RedisPlugin) Stop() error {
	slog.Info("Redis plugin stopped")
	return nil
}

func main() {
	// Set up structured logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Register global plugins
	kafkaPlugin := &KafkaPlugin{}
	redisPlugin := &RedisPlugin{}

	// Register plugins globally (they implement both Plugin and Config interfaces)
	plugins.RegisterGlobalPlugin(kafkaPlugin, KafkaConfig{})
	plugins.RegisterGlobalPlugin(redisPlugin, RedisConfig{})

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
