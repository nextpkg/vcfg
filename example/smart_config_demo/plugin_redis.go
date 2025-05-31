package main

import (
	"fmt"
	"log/slog"

	"github.com/nextpkg/vcfg/plugins"
)

func init() {
	plugins.RegisterGlobalPlugin(&RedisPlugin{}, &RedisConfig{})
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
	if redisConfig, ok := config.(*RedisConfig); ok {
		p.config = *redisConfig
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
