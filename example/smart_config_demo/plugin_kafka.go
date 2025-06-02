package main

import (
	"fmt"
	"log/slog"

	"github.com/nextpkg/vcfg/plugins"
)

// func init() {
//	plugins.RegisterGlobalPlugin(&KafkaPlugin{}, &KafkaConfig{})
// }
// Note: Global plugin registration is disabled to support multi-instance plugins.
// Plugins are now registered manually in main.go for each configuration instance.

// KafkaConfig represents Kafka plugin configuration
type KafkaConfig struct {
	plugins.BaseConfig        // Embed BaseConfig for automatic functionality
	BootstrapServers   string `json:"bootstrap_servers" yaml:"bootstrap_servers"`
	Topic              string `json:"topic" yaml:"topic"`
	GroupID            string `json:"group_id" yaml:"group_id"`
}

// KafkaPlugin represents a Kafka plugin
type KafkaPlugin struct {
	plugins.BasePlugin // Embed BasePlugin for automatic functionality
	config             KafkaConfig
}

// Start implements plugins.Plugin interface
func (p *KafkaPlugin) Start(config any) error {
	slog.Info("Kafka Start Point", "addr", p)

	if kafkaConfig, ok := config.(*KafkaConfig); ok {
		p.config = *kafkaConfig
		slog.Info("Kafka plugin started", "bootstrap_servers", kafkaConfig.BootstrapServers, "topic", kafkaConfig.Topic)
		return nil
	}
	return fmt.Errorf("invalid kafka config type: %T", config)
}

// Reload implements plugins.Plugin interface
func (p *KafkaPlugin) Reload(config any) error {
	if kafkaConfig, ok := config.(*KafkaConfig); ok {
		p.config = *kafkaConfig
		slog.Info("Kafka plugin reloaded", "bootstrap_servers", kafkaConfig.BootstrapServers, "topic", kafkaConfig.Topic)
		return nil
	}
	return fmt.Errorf("invalid kafka config type: %T", config)
}

// Stop implements plugins.Plugin interface
func (p *KafkaPlugin) Stop() error {
	slog.Info("Kafka plugin stopped")
	return nil
}
