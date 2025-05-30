package main

import (
	"fmt"
	"log/slog"
)

// func init() {
//	plugins.RegisterGlobalPlugin(&RedisPlugin{}, &RedisConfig{})
// }
// Note: Global plugin registration is disabled to support multi-instance plugins.
// Plugins are now registered manually in main.go for each configuration instance.

// RedisConfig represents Redis plugin configuration
type RedisConfig struct {
	Host     string `json:"host" yaml:"host"`
	Port     int    `json:"port" yaml:"port"`
	Password string `json:"password" yaml:"password"`
	DB       int    `json:"db" yaml:"db"`
}

// Note: Name() method is now optional. If not implemented, the framework
// will automatically derive the type name from the struct name ("redisconfig" -> "redis")

// RedisPlugin represents a Redis plugin
type RedisPlugin struct {
	config RedisConfig
}

// Note: Name() method is now optional. If not implemented, the framework
// will automatically derive the type name from the struct name ("redisplugin" -> "redis")

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
