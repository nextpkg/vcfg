# VCFG - Versatile Configuration Manager for Go

[![Go Version](https://img.shields.io/badge/go-1.24.2+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/nextpkg/vcfg)](https://goreportcard.com/report/github.com/nextpkg/vcfg)

VCFG is a powerful and flexible configuration management library for Go applications that supports multiple configuration sources, automatic type conversion, validation, and an extensible plugin system. It provides both simple and advanced configuration loading patterns to meet various application needs.

## Features

### Core Features
- **Multiple Configuration Sources**: Support for JSON, YAML, TOML files, environment variables, CLI flags, and custom providers
- **Type-Safe Configuration**: Generic-based type safety with automatic unmarshaling
- **Configuration Merging**: Intelligent merging of multiple configuration sources with priority handling
- **Hot Reloading**: Automatic configuration reloading when files change
- **Validation**: Built-in validation using `github.com/go-playground/validator/v10`
- **Default Values**: Automatic application of default values using struct tags

### Plugin System
- **Extensible Architecture**: Plugin-based architecture for extending functionality
- **Built-in Plugins**: Logger plugin with advanced features (rotation, multiple outputs, structured logging)
- **Plugin Lifecycle Management**: Automatic plugin discovery, initialization, and cleanup
- **Hot Reload Support**: Automatic plugin reloading when configuration changes are detected
- **Custom Plugins**: Easy creation of custom plugins with well-defined interfaces

### Advanced Features
- **Thread-Safe Operations**: Concurrent access protection with atomic operations
- **Builder Pattern**: Fluent API for configuration setup
- **CLI Integration**: Seamless integration with `urfave/cli/v3`
- **Environment Variable Support**: Flexible environment variable mapping
- **File Watching**: Automatic detection of configuration file changes

## Installation

```bash
go get github.com/nextpkg/vcfg
```

## Quick Start

### Simple Usage

```go
package main

import (
    "fmt"
    "github.com/nextpkg/vcfg"
)

type Config struct {
    Name    string `json:"name" default:"MyApp"`
    Port    int    `json:"port" default:"8080"`
    Debug   bool   `json:"debug" default:"false"`
}

func main() {
    // Load configuration from file
    cm := vcfg.MustLoad[Config]("config.json")
    defer cm.Close()
    
    config := cm.Get()
    fmt.Printf("App: %s, Port: %d, Debug: %t\n", config.Name, config.Port, config.Debug)
}
```

### Advanced Usage with Builder

```go
package main

import (
    "context"
    "log"
    "github.com/nextpkg/vcfg"
    "github.com/urfave/cli/v3"
)

type AppConfig struct {
    Server struct {
        Host string `json:"host" default:"localhost"`
        Port int    `json:"port" default:"8080"`
    } `json:"server"`
    Database struct {
        URL      string `json:"url" validate:"required"`
        MaxConns int    `json:"max_conns" default:"10"`
    } `json:"database"`
    // Plugin configurations are automatically discovered by the plugin system
    // based on struct fields that implement plugins.Config interface
    Logger *LoggerConfig `json:"logger,omitempty"`
}

func main() {
    app := &cli.Command{
        Name: "myapp",
        Flags: []cli.Flag{
            &cli.StringFlag{
                Name:  "config",
                Value: "config.yaml",
                Usage: "Configuration file path",
            },
            &cli.IntFlag{
                Name:  "server.port",
                Usage: "Server port",
            },
        },
        Action: func(ctx context.Context, cmd *cli.Command) error {
            // Build configuration manager with multiple sources
            cm, err := vcfg.NewBuilder[AppConfig]().
                AddFile(cmd.String("config")).        // Configuration file
                AddEnv("MYAPP_").                     // Environment variables
                AddCliFlags(cmd, ".").                // CLI flags
                WithWatch().                          // Enable file watching
                WithPlugin().                         // Enable plugins
                Build(context.Background())
            if err != nil {
                return err
            }
            defer cm.Close()
            
            config := cm.Get()
            log.Printf("Server starting on %s:%d", config.Server.Host, config.Server.Port)
            
            // Your application logic here
            return nil
        },
    }
    
    if err := app.Run(context.Background(), os.Args); err != nil {
        log.Fatal(err)
    }
}
```

## Configuration Sources

### File Sources
Supported formats: JSON, YAML, TOML

```go
// Single file
cm := vcfg.MustLoad[Config]("config.yaml")

// Multiple files (merged in order)
cm := vcfg.MustLoad[Config]("base.yaml", "env.yaml", "local.yaml")
```

### Environment Variables

```go
builder := vcfg.NewBuilder[Config]()
builder.AddEnv("MYAPP_") // Maps MYAPP_SERVER_PORT to server.port
```

### CLI Flags

```go
builder.AddCliFlags(cmd, ".") // Uses dot notation for nested keys
```

### Custom Providers

```go
import "github.com/knadh/koanf/providers/rawbytes"

provider := rawbytes.Provider([]byte(`{"key": "value"}`))
builder.AddProvider(provider)
```

## Plugin System

### Built-in Logger Plugin

```yaml
# config.yaml
plugins:
  logger:
    type: "logger"
    level: "info"
    format: "json"
    output: "both"  # stdout, stderr, file, both
    file_path: "./logs/app.log"
    add_source: true
    enable_rotation: true
    rotate_interval: "daily"
    max_file_size: 524288000  # 500MB
    max_age: 7
```

### Creating Custom Plugins

```go
package myplugin

import (
    "context"
    "github.com/nextpkg/vcfg/plugins"
)

// Plugin configuration
type MyPluginConfig struct {
    plugins.BaseConfig `koanf:",squash"`
    Setting1 string `koanf:"setting1" default:"default_value"`
    Setting2 int    `koanf:"setting2" default:"42"`
}

// Plugin implementation
type MyPlugin struct {
    config *MyPluginConfig
}

func (p *MyPlugin) Startup(ctx context.Context, config any) error {
    p.config = config.(*MyPluginConfig)
    // Initialize plugin
    return nil
}

func (p *MyPlugin) Reload(ctx context.Context, config any) error {
    p.config = config.(*MyPluginConfig)
    // Handle configuration reload
    return nil
}

func (p *MyPlugin) Shutdown(ctx context.Context) error {
    // Cleanup resources
    return nil
}

// Register plugin
func init() {
    plugins.RegisterPluginType("myplugin", &MyPlugin{}, &MyPluginConfig{})
}
```

## Configuration Validation

VCFG uses `github.com/go-playground/validator/v10` for validation:

```go
type Config struct {
    Email    string `json:"email" validate:"required,email"`
    Port     int    `json:"port" validate:"min=1,max=65535"`
    URL      string `json:"url" validate:"required,url"`
    Password string `json:"password" validate:"min=8"`
}
```

## Default Values

Set default values using struct tags:

```go
type Config struct {
    Host     string `json:"host" default:"localhost"`
    Port     int    `json:"port" default:"8080"`
    Debug    bool   `json:"debug" default:"false"`
    Timeout  string `json:"timeout" default:"30s"`
}
```

## File Watching

Enable automatic configuration reloading:

```go
cm := vcfg.NewBuilder[Config]().
    AddFile("config.yaml").
    WithWatch().  // Enable file watching
    MustBuild()
```

## Thread Safety

VCFG is designed to be thread-safe:

```go
// Safe to call from multiple goroutines
config := cm.Get()

// Safe concurrent access
go func() {
    for {
        config := cm.Get()
        // Use config
        time.Sleep(time.Second)
    }
}()
```

## Error Handling

```go
// Using Builder for error handling
cm, err := vcfg.NewBuilder[Config]().
    AddFile("config.yaml").
    Build(context.Background())
if err != nil {
    log.Fatal("Failed to build config:", err)
}

// Using MustLoad for simple cases (panics on error)
cm := vcfg.MustLoad[Config]("config.yaml")
```

## Plugin Hot Reload

VCFG supports automatic plugin reloading when configuration changes are detected:

```go
type AppConfig struct {
    Database DatabaseConfig `json:"database"`
    Cache    CacheConfig   `json:"cache"`
    // Nested plugin configurations are also supported
    Services struct {
        Auth    AuthConfig    `json:"auth"`
        Payment PaymentConfig `json:"payment"`
    } `json:"services"`
}

// When any plugin configuration changes, the corresponding plugin
// will be automatically reloaded without affecting other plugins
cm, err := vcfg.NewBuilder[AppConfig]().
    AddFile("config.yaml").
    WithWatch(). // Enable file watching for hot reload
    Build(context.Background())
```

**Key Features:**
- **Recursive Detection**: Automatically detects changes in nested plugin configurations
- **Selective Reload**: Only reloads plugins whose configurations have actually changed
- **Error Handling**: Continues processing other plugins even if one plugin reload fails
- **Thread-Safe**: All reload operations are thread-safe and non-blocking

## Best Practices

1. **Use struct tags**: Always define `json`, `yaml`, `default`, and `validate` tags
2. **Handle errors**: Use Builder pattern for production code with proper error handling
3. **Close resources**: Always call `cm.Close()` to clean up resources
4. **Validate configuration**: Use validation tags to ensure configuration integrity
5. **Use plugins wisely**: Leverage the plugin system for cross-cutting concerns
6. **Environment-specific configs**: Use multiple configuration files for different environments
7. **Plugin hot reload**: Implement proper `Reload` methods in custom plugins to support hot reloading

## Examples

Check the `example/` directory for more comprehensive examples:

- Basic configuration loading
- Multi-source configuration
- Plugin usage
- CLI integration
- Hot reloading

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

### Development Setup

```bash
# Clone the repository
git clone https://github.com/nextpkg/vcfg.git
cd vcfg

# Install dependencies
go mod download

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [knadh/koanf](https://github.com/knadh/koanf) - The underlying configuration library
- [go-playground/validator](https://github.com/go-playground/validator) - Validation framework
- [urfave/cli](https://github.com/urfave/cli) - CLI framework integration

## Support

If you have any questions or need help, please:

1. Check the documentation and examples
2. Search existing issues
3. Create a new issue with detailed information

---

**VCFG** - Making configuration management in Go simple, powerful, and extensible.