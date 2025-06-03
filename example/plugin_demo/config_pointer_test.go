package main

// TestAppConfigWithPointers represents a configuration with pointer types
// This should trigger an error when used with AutoRegisterPlugins
type TestAppConfigWithPointers struct {
	// Using pointer types - this should be detected and cause an error
	KafkaProducer *KafkaConfig `json:"kafka_producer" yaml:"kafka_producer"`
	KafkaConsumer *KafkaConfig `json:"kafka_consumer" yaml:"kafka_consumer"`
	KafkaStream   *KafkaConfig `json:"kafka_stream" yaml:"kafka_stream"`

	// Redis configuration as pointer too
	Redis *RedisConfig `json:"redis" yaml:"redis"`
}
