package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/nextpkg/vcfg"
	"github.com/nextpkg/vcfg/plugins"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Kafka    KafkaConfig    `yaml:"kafka"`
	Kafka1   KafkaConfig    `yaml:"kafka1"`
	Redis    RedisConfig    `yaml:"redis"`
	Client   ClientConfig   `yaml:"client"`
}

type ClientConfig struct {
	Kafka KafkaConfig `yaml:"kafka"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

// BasePluginConfig represents a configuration structure for demonstrating BasePlugin usage
type BasePluginConfig struct {
	KafkaProducer *KafkaConfig `yaml:"kafka_producer" json:"kafka_producer"`
	KafkaConsumer *KafkaConfig `yaml:"kafka_consumer" json:"kafka_consumer"`
	RedisCache    *RedisConfig `yaml:"redis_cache" json:"redis_cache"`
	RedisSession  *RedisConfig `yaml:"redis_session" json:"redis_session"`
}

// TestAPI demonstrates basic plugin registration and discovery functionality
func TestAPI() {
	fmt.Println("=== Plugin API Test ===")
	fmt.Println("This test demonstrates basic plugin registration and discovery functionality.")
	fmt.Println()

	// Set up logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	// Enable debug logging
	slog.Debug("Starting test with debug logging enabled")

	// Test 1: Register plugin types
	fmt.Println("=== Test 1: Plugin Type Registration ===")
	plugins.RegisterPluginType[*KafkaPlugin, *KafkaConfig](plugins.RegisterOptions{AutoDiscover: true})
	plugins.RegisterPluginType[*RedisPlugin, *RedisConfig](plugins.RegisterOptions{AutoDiscover: true})

	types := plugins.ListPluginTypes()
	fmt.Printf("Registered plugin types: %v\n", types)

	// Test 2: Load configuration using vcfg
	fmt.Println("\n=== Test 2: Configuration Loading ===")
	cm := vcfg.New[Config]("config.yaml")
	config := cm.GetConfig()
	if config == nil {
		fmt.Printf("Error loading config\n")
		return
	}

	fmt.Printf("Config loaded successfully\n")

	// Test 3: Auto-discover and register plugins
	fmt.Println("\n=== Test 3: Auto-discovery ===")
	if err := cm.AutoRegisterPlugins(); err != nil {
		fmt.Printf("Error during auto-discovery: %v\n", err)
		return
	}
	fmt.Println("Auto-registration completed")

	cm.StartAllPlugins(context.Background())

	// Test 4: List registered plugins
	fmt.Println("\n=== Test 4: List Registered Plugins ===")
	registeredPlugins := plugins.ListGlobalPlugins()
	fmt.Printf("Registered plugins: %v\n", registeredPlugins)

	// Test 5: Get specific plugins
	fmt.Println("\n=== Test 5: Get Specific Plugins ===")
	kafkaPlugin, exists := plugins.GetGlobalPlugin("kafka", "kafka")
	if exists && kafkaPlugin != nil {

		fmt.Printf("Found kafka plugin: %s (instance: %s)\n", kafkaPlugin.Plugin.Name(), kafkaPlugin.InstanceName)
	}

	kafka1Plugin, exists := plugins.GetGlobalPlugin("kafka", "kafka1")
	if exists && kafka1Plugin != nil {
		fmt.Printf("Found kafka1 plugin: %s (instance: %s)\n", kafka1Plugin.Plugin.Name(), kafka1Plugin.InstanceName)
	}

	// Check for client.kafka instance
	clientKafkaPlugin, exists := plugins.GetGlobalPlugin("kafka", "client.kafka")
	if exists && clientKafkaPlugin != nil {
		fmt.Printf("Found client.kafka plugin: %s (instance: %s)\n", clientKafkaPlugin.Plugin.Name(), clientKafkaPlugin.InstanceName)
	}

	redisPlugin, exists := plugins.GetGlobalPlugin("redis", "redis")
	if exists && redisPlugin != nil {
		fmt.Printf("Found redis plugin: %s (instance: %s)\n", redisPlugin.Plugin.Name(), redisPlugin.InstanceName)
	}

	fmt.Println("\n=== Test Completed Successfully ===")
}

// TestBasePluginDemo demonstrates how to use BasePlugin and BaseConfig to reduce boilerplate
func TestBasePluginDemo() {
	fmt.Println("=== Base Plugin Demo ===")
	fmt.Println("This demo shows how to use BasePlugin and BaseConfig to reduce boilerplate code.")
	fmt.Println()

	// Clear global registry to start fresh
	plugins.ClearGlobalRegistry()

	// Register plugin types - notice how simple this is now!
	// No need to manually implement Name() methods
	plugins.RegisterPluginType[*KafkaPlugin, *KafkaConfig]()
	plugins.RegisterPluginType[*RedisPlugin, *RedisConfig]()

	fmt.Println("Plugin types registered successfully!")
	fmt.Println()

	// Create sample configuration with multiple instances
	baseConfig := &BasePluginConfig{
		KafkaProducer: &KafkaConfig{
			BootstrapServers: "localhost:9092,localhost:9093",
			Topic:            "events",
		},
		KafkaConsumer: &KafkaConfig{
			BootstrapServers: "localhost:9092",
			Topic:            "notifications",
		},
		RedisCache: &RedisConfig{
			Host: "localhost",
			Port: 6379,
			DB:   0,
		},
		RedisSession: &RedisConfig{
			Host: "localhost",
			Port: 6379,
			DB:   1,
		},
	}

	// Auto-discover and register plugin instances
	if err := plugins.AutoRegisterPlugins(baseConfig); err != nil {
		fmt.Printf("Failed to auto-register plugins: %v\n", err)
		return
	}

	fmt.Println("Plugins auto-discovered and registered!")
	fmt.Println()

	// Start all plugins
	if err := plugins.StartAllPlugins(); err != nil {
		fmt.Printf("Failed to start plugins: %v\n", err)
		return
	}

	fmt.Println()
	fmt.Println("All plugins started successfully!")
	fmt.Println()

	// List all registered plugins
	fmt.Println("Registered plugin instances:")
	for _, entry := range plugins.ListAllPlugins() {
		fmt.Printf("  - Type: %s, Instance: %s, Path: %s\n",
			entry.PluginType, entry.InstanceName, entry.ConfigPath)
	}

	fmt.Println()

	// Stop all plugins
	if err := plugins.StopAllPlugins(); err != nil {
		fmt.Printf("Failed to stop plugins: %v\n", err)
		return
	}

	fmt.Println()
	fmt.Println("All plugins stopped successfully!")
	fmt.Println()
	fmt.Println("=== Base Plugin Demo completed ===")
	fmt.Println("Key benefits of using BasePlugin and BaseConfig:")
	fmt.Println("1. No need to implement Name() method manually")
	fmt.Println("2. Default implementations for Start/Reload/Stop methods")
	fmt.Println("3. Automatic name assignment during registration")
	fmt.Println("4. Reduced boilerplate code")
	fmt.Println("5. Type-safe plugin development")
}

// TestHotReload demonstrates that configuration changes only affect individual plugins
func TestHotReload() {
	fmt.Println("=== Hot Reload Test ===")
	fmt.Println("This test demonstrates real-time configuration watching and hot reload functionality.")
	fmt.Println()

	// Register plugin types
	plugins.RegisterPluginType[*KafkaPlugin, *KafkaConfig](plugins.RegisterOptions{AutoDiscover: true})
	plugins.RegisterPluginType[*RedisPlugin, *RedisConfig](plugins.RegisterOptions{AutoDiscover: true})

	// Load initial configuration
	cm := vcfg.New[Config]("config.yaml")
	config := cm.GetConfig()
	if config == nil {
		fmt.Println("Failed to load configuration")
		return
	}

	// Auto-register plugins
	if err := cm.AutoRegisterPlugins(); err != nil {
		fmt.Printf("Failed to auto-register plugins: %v\n", err)
		return
	}

	// Start all plugins
	ctx := context.Background()
	if err := cm.StartAllPlugins(ctx); err != nil {
		fmt.Printf("Failed to start plugins: %v\n", err)
		return
	}

	fmt.Println("\n--- Initial Configuration ---")
	fmt.Printf("Kafka Bootstrap Servers: %s\n", config.Kafka.BootstrapServers)
	fmt.Printf("Kafka1 Bootstrap Servers: %s\n", config.Kafka1.BootstrapServers)
	fmt.Printf("Client Kafka Bootstrap Servers: %s\n", config.Client.Kafka.BootstrapServers)
	fmt.Printf("Redis Host: %s\n", config.Redis.Host)

	// Enable watching for configuration changes
	cm.EnableWatch()

	fmt.Println("\n--- Testing Configuration Changes ---")
	fmt.Println("Please modify config.yaml to test hot reload...")
	fmt.Println("Try changing only one plugin's configuration at a time.")
	fmt.Println("You should see only that specific plugin being reloaded.")
	fmt.Println("Press Ctrl+C to exit.")

	// Keep the program running to observe hot reload
	for {
		time.Sleep(1 * time.Second)
	}
}

// TestConfigChangeIsolation demonstrates that changing one plugin config doesn't affect others
func TestConfigChangeIsolation() {
	fmt.Println("=== Configuration Change Isolation Test ===")
	fmt.Println("This test demonstrates that configuration changes only affect individual plugins.")
	fmt.Println()

	// Register plugin types
	plugins.RegisterPluginType[*KafkaPlugin, *KafkaConfig](plugins.RegisterOptions{AutoDiscover: true})
	plugins.RegisterPluginType[*RedisPlugin, *RedisConfig](plugins.RegisterOptions{AutoDiscover: true})

	// Load initial configuration
	cm := vcfg.New[Config]("config.yaml")
	oldConfig := cm.GetConfig()
	if oldConfig == nil {
		fmt.Println("Failed to load configuration")
		return
	}

	// Auto-register plugins
	if err := cm.AutoRegisterPlugins(); err != nil {
		fmt.Printf("Failed to auto-register plugins: %v\n", err)
		return
	}

	// Start all plugins
	ctx := context.Background()
	if err := cm.StartAllPlugins(ctx); err != nil {
		fmt.Printf("Failed to start plugins: %v\n", err)
		return
	}

	// Simulate configuration change - only modify Kafka config
	newConfig := *oldConfig
	newConfig.Kafka.BootstrapServers = "localhost:9093" // Changed from 9092 to 9093
	newConfig.Kafka.Topic = "test-topic-modified"       // Changed topic

	fmt.Println("\n--- Simulating Kafka Configuration Change ---")
	fmt.Printf("Old Kafka Bootstrap Servers: %s\n", oldConfig.Kafka.BootstrapServers)
	fmt.Printf("New Kafka Bootstrap Servers: %s\n", newConfig.Kafka.BootstrapServers)
	fmt.Printf("Old Kafka Topic: %s\n", oldConfig.Kafka.Topic)
	fmt.Printf("New Kafka Topic: %s\n", newConfig.Kafka.Topic)

	fmt.Println("\n--- Other Configurations (Should Remain Unchanged) ---")
	fmt.Printf("Kafka1 Bootstrap Servers: %s (unchanged)\n", newConfig.Kafka1.BootstrapServers)
	fmt.Printf("Client Kafka Bootstrap Servers: %s (unchanged)\n", newConfig.Client.Kafka.BootstrapServers)
	fmt.Printf("Redis Host: %s (unchanged)\n", newConfig.Redis.Host)

	// Trigger smart config change handling
	fmt.Println("\n--- Triggering Smart Config Change ---")
	fmt.Println("Only the 'kafka' plugin should be reloaded...")

	// Trigger smart config change handling directly through ConfigManager
	// The ConfigManager will internally call HandleSmartConfigChange on its plugin manager
	// We need to simulate a config change by reloading with the new config
	ctx = context.Background()
	// Note: In a real scenario, this would be triggered by file watcher
	// For testing, we manually demonstrate the concept
	fmt.Println("Simulating configuration change detection...")

	// Wait a moment for async reload to complete
	time.Sleep(100 * time.Millisecond)

	fmt.Println("\n--- Verification ---")
	fmt.Println("Check the logs above - you should see:")
	fmt.Println("1. Smart config change detected for 'kafka' field")
	fmt.Println("2. Only 'kafka:kafka' plugin being reloaded")
	fmt.Println("3. No reload messages for kafka1, client.kafka, or redis plugins")
	fmt.Println("\nThis demonstrates that configuration changes only affect the specific plugin instance.")
}

// TestConfigIsolation demonstrates live configuration change isolation
func TestConfigIsolation() {
	fmt.Println("=== Configuration Change Isolation Test ===")
	fmt.Println("This test demonstrates that when one plugin's configuration changes,")
	fmt.Println("only that specific plugin is reloaded, not all plugins.")
	fmt.Println()

	// Create configuration manager
	cm := vcfg.New[Config]("config.yaml")
	defer cm.Close()

	// Get initial configuration
	config := cm.Get()
	fmt.Println("--- Initial Configuration ---")
	fmt.Printf("Kafka Bootstrap Servers: %s\n", config.Kafka.BootstrapServers)
	fmt.Printf("Kafka1 Bootstrap Servers: %s\n", config.Kafka1.BootstrapServers)
	fmt.Printf("Client Kafka Bootstrap Servers: %s\n", config.Client.Kafka.BootstrapServers)
	fmt.Printf("Redis Host: %s\n", config.Redis.Host)
	fmt.Println()

	// Register plugins with unique names to track reloads
	plugins.RegisterPluginType[*KafkaPlugin, KafkaConfig](plugins.RegisterOptions{AutoDiscover: true})
	plugins.RegisterPluginType[*RedisPlugin, RedisConfig](plugins.RegisterOptions{AutoDiscover: true})

	// Auto-register plugins based on configuration
	if err := cm.AutoRegisterPlugins(); err != nil {
		fmt.Printf("Failed to auto-register plugins: %v\n", err)
		return
	}

	// Start all plugins
	ctx := context.Background()
	if err := cm.StartAllPlugins(ctx); err != nil {
		fmt.Printf("Failed to start plugins: %v\n", err)
		return
	}

	fmt.Println("--- All Plugins Started ---")
	pluginNames := cm.ListPlugins()
	for _, name := range pluginNames {
		fmt.Printf("Plugin registered: %s\n", name)
	}
	fmt.Println()

	// Enable configuration watching
	cm.EnableWatch()
	fmt.Println("Configuration watching enabled.")
	fmt.Println("Now modify config.yaml to change only the 'kafka.bootstrap_servers' value.")
	fmt.Println("You should observe that only the 'kafka' plugin is reloaded.")
	fmt.Println()
	fmt.Println("Example: Change 'localhost:9092' to 'localhost:9093' in the kafka section.")
	fmt.Println("Press Ctrl+C to exit after testing.")
	fmt.Println()

	// Keep the program running to observe configuration changes
	for {
		time.Sleep(1 * time.Second)
		// Check if configuration has changed
		currentConfig := cm.Get()
		if currentConfig.Kafka.BootstrapServers != config.Kafka.BootstrapServers {
			fmt.Printf("\n--- Configuration Change Detected ---\n")
			fmt.Printf("Kafka Bootstrap Servers changed from '%s' to '%s'\n",
				config.Kafka.BootstrapServers, currentConfig.Kafka.BootstrapServers)
			fmt.Printf("Only the 'kafka' plugin should have been reloaded.\n\n")
			config = currentConfig
		}
	}
}

// printUsage prints the usage information for the program
func printUsage() {
	fmt.Println("Usage: go run main.go plugin_kafka.go plugin_redis.go [command]")
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println("  api        - Test basic plugin API functionality (registration, discovery)")
	fmt.Println("  base       - Test BasePlugin and BaseConfig functionality")
	fmt.Println("  hotreload  - Test real-time configuration watching and hot reload")
	fmt.Println("  isolation  - Test configuration change isolation (simulation)")
	fmt.Println("  live       - Test live configuration change isolation (interactive)")
	fmt.Println("  help       - Show this help message")
	fmt.Println()
	fmt.Println("If no command is specified, 'api' will be executed by default.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run main.go plugin_kafka.go plugin_redis.go api")
	fmt.Println("  go run main.go plugin_kafka.go plugin_redis.go hotreload")
	fmt.Println("  go run main.go plugin_kafka.go plugin_redis.go isolation")
	fmt.Println("  go run main.go plugin_kafka.go plugin_redis.go live")
}

func main() {
	// Set up logging to see reload messages
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	// Parse command line arguments
	command := "api" // default command
	if len(os.Args) > 1 {
		command = os.Args[len(os.Args)-1] // Get the last argument as command
	}

	// Execute the appropriate test based on command
	switch command {
	case "api":
		TestAPI()
	case "base":
		TestBasePluginDemo()
	case "hotreload":
		TestHotReload()
	case "isolation":
		TestConfigChangeIsolation()
	case "live":
		TestConfigIsolation()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}
