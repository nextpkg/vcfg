# Logger Plugin Demo

This demo shows how to use the built-in logger plugin in vcfg.

## Features

The logger plugin provides:

- **Multiple log levels**: debug, info, warn, error
- **Multiple output formats**: JSON, text
- **Multiple output targets**: stdout, stderr, file, both
- **Source file control**: Enable/disable file name and line number in logs
- **Structured logging**: Support for key-value pairs
- **Global logger access**: Easy access to logger instance throughout the application

## Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `type` | string | - | Plugin type, must be "vcfg-logger" |
| `level` | string | "info" | Log level: debug, info, warn, error |
| `format` | string | "json" | Output format: json, text |
| `output` | string | "stdout" | Output target: stdout, stderr, file, both |
| `file_path` | string | "./app.log" | Log file path (when output is file or both) |
| `add_source` | bool | false | Whether to add source file info (file name and line number) |

## Running the Demo

### With Source File Information

```bash
cd /home/ruifenglin/app/vcfg/example/logger
go run main.go
```

This will use `config.yaml` which has `add_source: true`, so logs will include file names and line numbers.

### Modifying Configuration

You can modify the `config.yaml` file to change the logger behavior:

```yaml
# Set add_source to false to disable source file information
add_source: false
```

Then run the example again to see the difference in log output.

## Example Output

### With Source Info (JSON format)

```json
{"time":"2024-01-15T10:30:45.123Z","level":"INFO","source":{"function":"main.main","file":"/path/to/main.go","line":25},"msg":"Application starting","name":"Logger Demo App","version":"1.0.0"}
```

### Without Source Info (Text format)

```
10:30:45 INF Application starting name="Logger Demo App" version=1.0.0
```

## Using the Logger in Your Code

```go
package main

import (
    "github.com/nextpkg/vcfg"
    "github.com/nextpkg/vcfg/plugins/builtins"
)

func main() {
    // Load configuration (this will start the logger plugin)
    cm := vcfg.MustLoad[YourConfig]("config.yaml")
    defer cm.Stop()
    
    // Get the global logger instance
    logger := builtins.GetLogger()
    
    // Use structured logging
    logger.Info("User logged in", 
        "user_id", 12345,
        "ip", "192.168.1.100",
        "success", true,
    )
    
    // Log errors with context
    logger.Error("Database error", 
        "error", err.Error(),
        "query", "SELECT * FROM users",
    )
}

```
## Best Practices

1. **Use structured logging**: Always provide key-value pairs for better log analysis
2. **Choose appropriate log levels**: Use debug for development, info for normal operations
3. **Enable source info during development**: Set `add_source: true` for debugging
4. **Disable source info in production**: Set `add_source: false` for better performance
5. **Configure file rotation**: Set appropriate limits to prevent disk space issues
6. **Use JSON format for production**: Easier to parse and analyze with log aggregation tools
7. **Use text format for development**: More human-readable during development