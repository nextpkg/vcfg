package main

import (
	"fmt"
	"log/slog"

	"github.com/nextpkg/vcfg/plugins"
)

func init() {
	plugins.RegisterGlobalPlugin(&KafkaPlugin{}, &KafkaConfig{})
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
	if kafkaConfig, ok := config.(*KafkaConfig); ok {
		p.config = *kafkaConfig
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
