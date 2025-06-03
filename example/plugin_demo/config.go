package main

import "github.com/nextpkg/vcfg/plugins"

// AppConfig represents the main application configuration
// It contains multiple Kafka configurations to demonstrate plugin instances
type AppConfig struct {
	// Multiple Kafka configurations - each will create a separate plugin instance
	KafkaProducer KafkaConfig `json:"kafka_producer" yaml:"kafka_producer" koanf:"kafka_producer"`
	KafkaConsumer KafkaConfig `json:"kafka_consumer" yaml:"kafka_consumer" koanf:"kafka_consumer"`
	KafkaStream   KafkaConfig `json:"kafka_stream" yaml:"kafka_stream" koanf:"kafka_stream"`

	// Redis configuration for comparison
	Redis RedisConfig `json:"redis" yaml:"redis" koanf:"redis"`
}

// KafkaConfig represents Kafka plugin configuration
type KafkaConfig struct {
	plugins.BaseConfig
	BootstrapServers string `json:"bootstrap_servers" yaml:"bootstrap_servers" koanf:"bootstrap_servers"`
	Topic            string `json:"topic" yaml:"topic" koanf:"topic"`
	GroupID          string `json:"group_id" yaml:"group_id" koanf:"group_id"`
}

// RedisConfig represents Redis plugin configuration
type RedisConfig struct {
	plugins.BaseConfig
	Host     string `json:"host" yaml:"host" koanf:"host"`
	Port     int    `json:"port" yaml:"port" koanf:"port"`
	Password string `json:"password" yaml:"password" koanf:"password"`
	DB       int    `json:"db" yaml:"db" koanf:"db"`
}
