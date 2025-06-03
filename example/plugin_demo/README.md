# Plugin Demo - Base Example

This example demonstrates basic plugin loading functionality using the `vcfg.MustInit` method with multiple plugin instances.

## Features

- **Multiple Plugin Instances**: Demonstrates how multiple Kafka configuration blocks create separate plugin instances
- **Auto-Registration**: Uses `ConfigManager.AutoRegisterPlugins()` to automatically discover and register plugins
- **Plugin Lifecycle**: Shows plugin startup, running, and shutdown processes
- **Configuration Watching**: Supports hot reload of configuration changes

## Configuration Structure

The example configuration (`config.yaml`) contains:
- 3 Kafka plugin instances: `kafka_producer`, `kafka_consumer`, `kafka_stream`
- 1 Redis plugin instance: `redis`

Each Kafka configuration block will create a separate plugin instance, demonstrating the multi-instance capability.

## Usage

1. **Build and run the example**:
   ```bash
   cd /home/ruifenglin/app/vcfg/example/plugin_demo/base
   go mod tidy
   go run .
   ```

2. **Expected output**:
   - Configuration loading messages
   - Plugin registration for each configuration block
   - Plugin startup messages for all instances
   - Application running message

3. **Test configuration changes**:
   - Modify `config.yaml` while the application is running
   - Observe automatic plugin reloading

## Key Components

### Configuration (`config.go`)
- `AppConfig`: Main configuration structure
- `KafkaConfig`: Kafka plugin configuration with `plugins.BaseConfig` embedded
- `RedisConfig`: Redis plugin configuration with `plugins.BaseConfig` embedded

### Plugins (`plugins.go`)
- Plugin registration using `plugins.RegisterPluginType()`
- `KafkaPlugin` and `RedisPlugin` implementations
- Plugin lifecycle methods: `Start()`, `Reload()`, `Stop()`

### Main Application (`main.go`)
- Uses `vcfg.MustInit[AppConfig]()` to initialize configuration manager
- Calls `AutoRegisterPlugins()` to discover and register plugin instances
- Calls `StartPlugins()` to start all registered plugins
- Enables configuration watching for hot reload
- Graceful shutdown handling

## Plugin Instance Mapping

Based on the configuration structure:
- `kafka_producer` config → `kafka` plugin instance named `kafkaproducer`
- `kafka_consumer` config → `kafka` plugin instance named `kafkaconsumer`
- `kafka_stream` config → `kafka` plugin instance named `kafkastream`
- `redis` config → `redis` plugin instance named `redis`

This demonstrates how the same plugin type (`kafka`) can have multiple instances with different configurations.