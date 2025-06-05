# VCFG Examples

This directory contains examples demonstrating various features and usage patterns of the VCFG configuration management library.

## Available Examples

### 1. Basic Usage (`basic/`)

**Features demonstrated:**
- Simple configuration loading from YAML and JSON files
- Environment variable integration with prefix support
- Multiple configuration sources with priority ordering
- Type-safe configuration structures
- Error handling and validation

**Files:**
- `main.go` - Complete basic usage example
- `config.yaml` - Sample YAML configuration
- `config.json` - Sample JSON configuration

**Run the example:**
```bash
cd basic
go run main.go
```

### 2. CLI Integration (`cli/`)

**Features demonstrated:**
- Integration with `urfave/cli/v3` framework
- Command-line flag support
- Configuration file and CLI flag combination
- Validation and error handling

**Files:**
- `main.go` - CLI application with VCFG integration
- `config.yaml` - Default configuration file

**Run the example:**
```bash
cd cli
go run main.go --help
go run main.go --config config.yaml
```

### 3. Plugin System (`plugin/`)

**Features demonstrated:**
- Plugin loading and management
- Plugin lifecycle (start, stop, health checks)
- Multiple plugin instances
- Plugin configuration and validation

**Files:**
- `main.go` - Plugin system demonstration
- `plugin-config.yaml` - Plugin configuration

**Run the example:**
```bash
cd plugin
go run main.go
```

### 4. Logger Integration (`logger/`)

**Features demonstrated:**
- Structured logging with slog integration
- Logger plugin configuration
- Log level and format customization
- Plugin-based logging architecture

**Files:**
- `main.go` - Logger plugin example
- `config.yaml` - Logger configuration
- `README.md` - Detailed logger documentation

**Run the example:**
```bash
cd logger
go run main.go
```

### 5. Hot Reload (`watch/`)

**Features demonstrated:**
- Configuration hot reload/watch functionality
- Real-time configuration updates
- Change detection and notification
- Graceful configuration reloading

**Files:**
- `main.go` - Hot reload demonstration
- `config.yaml` - Sample configuration for testing
- `test_watch.sh` - Script to test hot reload
- `README.md` - Hot reload documentation

**Run the example:**
```bash
cd watch
go run main.go
# In another terminal, modify config.yaml to see hot reload
```

### 6. Default Values (`defaults/`)

**Features demonstrated:**
- Struct tag-based default value setting
- Complex type defaults (duration, slices, pointers)
- Nested struct default support
- Comparison with manual default setting

**Files:**
- `main.go` - Default values demonstration

**Run the example:**
```bash
cd defaults
go run main.go
```

## Configuration Files

Each example creates its own configuration files to demonstrate different scenarios:

- **YAML format** - Human-readable, supports comments
- **JSON format** - Machine-readable, strict syntax
- **Environment variables** - Runtime configuration override

## Key Concepts Demonstrated

### Configuration Source Priority
1. Environment variables (highest priority)
2. Configuration files (lower priority)
3. Default values (lowest priority)

### Type Safety
- Strongly typed configuration structures
- Compile-time type checking
- Runtime validation support

### Plugin System
- Modular architecture
- Plugin lifecycle management
- Configuration-driven plugin loading

### Hot Reload
- File system watching
- Graceful configuration updates
- Change notification callbacks

## Common Patterns

### Simple Loading
```go
config := vcfg.MustLoad[MyConfig]("config.yaml")
```

### Builder Pattern
```go
cm, err := vcfg.NewBuilder[MyConfig]().
    AddFile("config.yaml").
    AddEnv("APP").
    Build(context.Background())
```

### Configuration Changes
```go
cm.OnChange(func(old, new *MyConfig) {
    log.Printf("Config changed: %+v -> %+v", old, new)
})
```

### Plugin Registration
```go
vcfg.RegisterPlugin("my-plugin", &MyPlugin{})
```

## Troubleshooting

### Common Issues

1. **File not found errors**
   - Ensure configuration files exist in the correct directory
   - Check file permissions

2. **Type conversion errors**
   - Verify configuration values match struct field types
   - Check for typos in configuration keys

3. **Environment variable issues**
   - Ensure proper prefix usage (e.g., `APP_SERVER_PORT`)
   - Check environment variable naming conventions

### Debug Tips

1. Enable debug logging:
   ```go
   slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
       Level: slog.LevelDebug,
   })))
   ```

2. Use validation:
   ```go
   func (c *Config) Validate() error {
       // Add validation logic
       return nil
   }
   ```

3. Check configuration sources:
   ```go
   sources := cm.GetSources()
   for _, source := range sources {
       fmt.Printf("Source: %s\n", source)
   }
   ```

## Next Steps

1. **Explore the main library** - Check out the root directory for core functionality
2. **Read the documentation** - Visit the project README for comprehensive guides
3. **Try modifications** - Experiment with different configuration values and structures
4. **Build your own** - Use these examples as templates for your applications

## Contributing

Found an issue or want to add a new example? Please:
1. Check existing issues and examples
2. Create a new example following the established patterns
3. Include proper documentation and error handling
4. Test your example thoroughly

For more information, see the main project documentation.