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

// LoggerConfig represents the configuration for the logger plugin
type LoggerConfig struct {
	plugins.BaseConfig `koanf:",squash"`
	Level              string `koanf:"level" default:"info"`          // log level: debug, info, warn, error
	Format             string `koanf:"format" default:"json"`         // log format: json, text
	Output             string `koanf:"output" default:"stdout"`       // output: stdout, stderr, file, both
	FilePath           string `koanf:"file_path" default:"./app.log"` // log file path
	AddSource          bool   `koanf:"add_source" default:"false"`    // whether to add source file info
}

// LoggerPlugin implements the logger plugin
type LoggerPlugin struct {
	mu     sync.RWMutex
	logger *slog.Logger
	file   *os.File
	config *LoggerConfig
}

// globalLogger holds the global logger instance
var (
	globalLogger *slog.Logger
	globalMu     sync.RWMutex
)

// GetLogger returns the global logger instance
// GetLogger returns the current global logger instance for application use
func GetLogger() *slog.Logger {
	globalMu.RLock()
	defer globalMu.RUnlock()
	if globalLogger == nil {
		return slog.Default()
	}
	return globalLogger
}

// setGlobalLogger sets the global logger instance
func setGlobalLogger(logger *slog.Logger) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalLogger = logger
	slog.SetDefault(logger)
}

// Startup implements plugins.Plugin interface
// Startup initializes the logger plugin with the provided configuration
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

// Reload implements plugins.Plugin interface
// Reload reloads the logger plugin with new configuration
func (p *LoggerPlugin) Reload(ctx context.Context, config any) error {
	p.logger.Info("Reloading logger plugin")

	// Stop current logger first
	if err := p.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to stop logger during reload: %w", err)
	}

	// Start with new config
	return p.Startup(ctx, config)
}

// Shutdown implements plugins.Plugin interface
// Shutdown gracefully shuts down the logger plugin
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

// createWriter creates the appropriate writer based on output configuration
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

// createFileWriter creates a file writer with rotation support
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

// parseLogLevel parses string log level to slog.Level
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
