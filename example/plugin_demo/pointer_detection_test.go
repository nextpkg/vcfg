package main

import (
	"strings"
	"testing"

	"github.com/nextpkg/vcfg/plugins"
)

// TestPointerConfigDetection tests that pointer type configs are properly detected and rejected
func TestPointerConfigDetection(t *testing.T) {
	// Create a config with pointer types
	testConfig := TestAppConfigWithPointers{
		KafkaProducer: &KafkaConfig{
			BootstrapServers: "localhost:9092",
			Topic:            "test-topic",
			GroupID:          "test-group",
		},
		KafkaConsumer: &KafkaConfig{
			BootstrapServers: "localhost:9092",
			Topic:            "test-topic-2",
			GroupID:          "test-group-2",
		},
		Redis: &RedisConfig{
			Host: "localhost",
			Port: 6379,
			DB:   0,
		},
	}

	// We need to manually test the plugin discovery logic
	// since we can't easily set config in the manager
	pm := plugins.NewPluginManager[TestAppConfigWithPointers]()
	err := pm.DiscoverAndRegister(&testConfig)

	// Verify that an error occurred
	if err == nil {
		t.Fatal("Expected error when using pointer type configs, but got nil")
	}

	// Verify the error message contains helpful information
	errorMsg := err.Error()
	expectedPhrases := []string{
		"指针类型",
		"值类型",
		"配置字段",
		"意外的共享状态",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(errorMsg, phrase) {
			t.Errorf("Error message should contain '%s', but got: %s", phrase, errorMsg)
		}
	}

	t.Logf("Successfully detected pointer config with error: %s", errorMsg)
}

// TestValueConfigStillWorks tests that value type configs continue to work normally
func TestValueConfigStillWorks(t *testing.T) {
	// Create a config with value types (the normal case)
	testConfig := AppConfig{
		KafkaProducer: KafkaConfig{
			BootstrapServers: "localhost:9092",
			Topic:            "test-topic",
			GroupID:          "test-group",
		},
		KafkaConsumer: KafkaConfig{
			BootstrapServers: "localhost:9092",
			Topic:            "test-topic-2",
			GroupID:          "test-group-2",
		},
		Redis: RedisConfig{
			Host: "localhost",
			Port: 6379,
			DB:   0,
		},
	}

	// Create plugin manager and test directly
	pm := plugins.NewPluginManager[AppConfig]()
	err := pm.DiscoverAndRegister(&testConfig)

	// Verify that no error occurred
	if err != nil {
		t.Fatalf("Expected no error when using value type configs, but got: %s", err)
	}

	t.Log("Successfully registered plugins with value type configs")
}
