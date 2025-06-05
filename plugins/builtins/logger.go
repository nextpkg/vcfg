// Package builtins provides built-in plugins for the vcfg configuration system.
// This file implements a comprehensive logger plugin that supports multiple output
// formats, destinations, and log levels with structured logging capabilities.
package builtins

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/nextpkg/vcfg/plugins"
)

// LoggerConfig represents the configuration for the logger plugin.
// It defines all configurable aspects of the logging behavior including
// output format, destination, log level, and additional options.
type LoggerConfig struct {
	// BaseConfig embeds the common plugin configuration
	plugins.BaseConfig `koanf:",squash"`
	// Level sets the minimum log level (debug, info, warn, error)
	Level string `koanf:"level" default:"info"`
	// Format specifies the log output format (json, text)
	Format string `koanf:"format" default:"json"`
	// Output determines where logs are written (stdout, stderr, file, both)
	Output string `koanf:"output" default:"stdout"`
	// FilePath specifies the log file path when output includes file
	FilePath string `koanf:"file_path" default:"./app.log"`
	// AddSource includes source file information in log entries
	AddSource bool `koanf:"add_source" default:"false"`
}

// LoggerPlugin implements the logger plugin that provides structured logging
// capabilities with configurable output formats and destinations.
type LoggerPlugin struct {
	// mu protects concurrent access to plugin state
	mu sync.RWMutex
	// logger is the configured slog.Logger instance
	logger *slog.Logger
	// file holds the log file handle when file output is enabled
	file *os.File
	// config stores the current plugin configuration
	config *LoggerConfig
}

// Global logger state management
var (
	// globalLogger holds the current global logger instance
	globalLogger *slog.Logger
	// globalMu protects concurrent access to the global logger
	globalMu sync.RWMutex
)

// GetLogger returns the current global logger instance for application use.
// If no logger has been configured, it returns the default slog logger.
// This function is thread-safe and can be called from multiple goroutines.
//
// Returns:
//   - *slog.Logger: The current global logger or default logger if none is set
func GetLogger() *slog.Logger {
	globalMu.RLock()
	defer globalMu.RUnlock()
	if globalLogger == nil {
		return slog.Default()
	}
	return globalLogger
}

// setGlobalLogger sets the global logger instance and updates the default slog logger.
// This function is used internally by the logger plugin to make the configured
// logger available globally throughout the application.
//
// Parameters:
//   - logger: The slog.Logger instance to set as global
func setGlobalLogger(logger *slog.Logger) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalLogger = logger
	slog.SetDefault(logger)
}

// Startup implements the plugins.Plugin interface by initializing the logger
// with the provided configuration. It sets up the log level, format, output
// destination, and creates the appropriate handlers.
//
// Parameters:
//   - ctx: Context for the startup operation
//   - config: LoggerConfig instance containing the logger configuration
//
// Returns:
//   - error: An error if initialization fails, nil otherwise
func (p *LoggerPlugin) Startup(ctx context.Context, config any) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	loggerConfig, ok := config.(*LoggerConfig)
	if !ok {
		return fmt.Errorf("invalid logger config type: %T", config)
	}

	p.config = loggerConfig

	// Parse log level
	level, err := parseLogLevel(p.config.Level)
	if err != nil {
		return fmt.Errorf("invalid log level %s: %w", p.config.Level, err)
	}

	// Create writer based on output configuration
	writer, err := p.createWriter()
	if err != nil {
		return fmt.Errorf("failed to create writer: %w", err)
	}

	// Create handler based on format
	var handler slog.Handler
	handlerOpts := &slog.HandlerOptions{
		Level:     level,
		AddSource: p.config.AddSource,
	}

	switch strings.ToLower(p.config.Format) {
	case "json":
		handler = slog.NewJSONHandler(writer, handlerOpts)
	case "text":
		handler = slog.NewTextHandler(writer, handlerOpts)
	default:
		return fmt.Errorf("unsupported log format: %s", p.config.Format)
	}

	// Create logger
	p.logger = slog.New(handler)

	// Set as global logger
	setGlobalLogger(p.logger)

	p.logger.Info("Logger plugin started",
		"level", p.config.Level,
		"format", p.config.Format,
		"output", p.config.Output,
		"add_source", p.config.AddSource,
	)

	return nil
}

// Reload implements the plugins.Plugin interface by reloading the logger
// with new configuration. It gracefully shuts down the current logger
// and reinitializes it with the new settings.
//
// Parameters:
//   - ctx: Context for the reload operation
//   - config: New LoggerConfig instance
//
// Returns:
//   - error: An error if reload fails, nil otherwise
func (p *LoggerPlugin) Reload(ctx context.Context, config any) error {
	p.logger.Info("Reloading logger plugin")

	// Stop current logger first
	if err := p.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to stop logger during reload: %w", err)
	}

	// Start with new config
	return p.Startup(ctx, config)
}

// Shutdown implements the plugins.Plugin interface by gracefully shutting down
// the logger plugin. It closes any open file handles and cleans up resources.
//
// Parameters:
//   - ctx: Context for the shutdown operation
//
// Returns:
//   - error: An error if shutdown fails, nil otherwise
func (p *LoggerPlugin) Shutdown(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.logger != nil {
		p.logger.Info("Logger plugin stopping")
	}

	// Close file if opened
	if p.file != nil {
		if err := p.file.Close(); err != nil {
			return fmt.Errorf("failed to close log file: %w", err)
		}
		p.file = nil
	}

	p.logger = nil
	p.config = nil

	return nil
}

// createWriter creates the appropriate io.Writer based on the output configuration.
// It supports stdout, stderr, file, and both (stdout + file) output modes.
//
// Returns:
//   - io.Writer: The configured writer for log output
//   - error: An error if writer creation fails, nil otherwise
func (p *LoggerPlugin) createWriter() (io.Writer, error) {
	switch strings.ToLower(p.config.Output) {
	case "stdout":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	case "file":
		return p.createFileWriter()
	case "both":
		fileWriter, err := p.createFileWriter()
		if err != nil {
			return nil, err
		}
		return io.MultiWriter(os.Stdout, fileWriter), nil
	default:
		return nil, fmt.Errorf("unsupported output type: %s", p.config.Output)
	}
}

// createFileWriter creates a file writer for log output. It ensures the
// log directory exists and opens the file with appropriate permissions.
//
// Returns:
//   - io.Writer: The file writer for log output
//   - error: An error if file creation fails, nil otherwise
func (p *LoggerPlugin) createFileWriter() (io.Writer, error) {
	// Ensure directory exists
	dir := filepath.Dir(p.config.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open file
	file, err := os.OpenFile(p.config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	p.file = file
	return file, nil
}

// parseLogLevel parses a string log level into the corresponding slog.Level.
// It supports debug, info, warn/warning, and error levels (case-insensitive).
//
// Parameters:
//   - level: String representation of the log level
//
// Returns:
//   - slog.Level: The parsed log level
//   - error: An error if the level is invalid, nil otherwise
func parseLogLevel(level string) (slog.Level, error) {
	if level == "" {
		return slog.LevelInfo, nil
	}
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unknown log level: %s", level)
	}
}
