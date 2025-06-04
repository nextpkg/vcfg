package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nextpkg/vcfg/plugins"
)

// Register plugin types in init function
func init() {
	// Register Kafka plugin type
	plugins.RegisterPluginType("kafka", &KafkaPlugin{}, &KafkaConfig{})
	// Register Redis plugin type
	plugins.RegisterPluginType("redis", &RedisPlugin{}, &RedisConfig{})
}

// KafkaPlugin represents a Kafka plugin
type KafkaPlugin struct{}

// Startup implements plugins.Plugin interface
func (p *KafkaPlugin) Startup(ctx context.Context, config any) error {
	if kafkaConfig, ok := config.(*KafkaConfig); ok {
		slog.Info("Kafka plugin started",
			"bootstrap_servers", kafkaConfig.BootstrapServers,
			"topic", kafkaConfig.Topic,
			"group_id", kafkaConfig.GroupID)
		return nil
	}
	return fmt.Errorf("invalid kafka config type: %T", config)
}

// Reload implements plugins.Plugin interface
func (p *KafkaPlugin) Reload(ctx context.Context, config any) error {
	if kafkaConfig, ok := config.(*KafkaConfig); ok {
		slog.Info("Kafka plugin reloaded",
			"bootstrap_servers", kafkaConfig.BootstrapServers,
			"topic", kafkaConfig.Topic,
			"group_id", kafkaConfig.GroupID)
		return nil
	}
	return fmt.Errorf("invalid kafka config type: %T", config)
}

// Shutdown implements plugins.Plugin interface
func (p *KafkaPlugin) Shutdown(ctx context.Context) error {
	slog.Info("Kafka plugin stopped")
	return nil
}

// RedisPlugin represents a Redis plugin
type RedisPlugin struct{}

// Startup implements plugins.Plugin interface
func (p *RedisPlugin) Startup(ctx context.Context, config any) error {
	if redisConfig, ok := config.(*RedisConfig); ok {
		slog.Info("Redis plugin started",
			"host", redisConfig.Host,
			"port", redisConfig.Port,
			"db", redisConfig.DB)
		return nil
	}
	return fmt.Errorf("invalid redis config type: %T", config)
}

// Reload implements plugins.Plugin interface
func (p *RedisPlugin) Reload(ctx context.Context, config any) error {
	if redisConfig, ok := config.(*RedisConfig); ok {
		slog.Info("Redis plugin reloaded",
			"host", redisConfig.Host,
			"port", redisConfig.Port,
			"db", redisConfig.DB)
		return nil
	}
	return fmt.Errorf("invalid redis config type: %T", config)
}

// Shutdown implements plugins.Plugin interface
func (p *RedisPlugin) Shutdown(ctx context.Context) error {
	slog.Info("Redis plugin stopped")
	return nil
}
