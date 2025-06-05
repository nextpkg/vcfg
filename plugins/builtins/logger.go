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
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nextpkg/vcfg/plugins"
)

// LoggerConfig represents the configuration for the logger plugin.
// It defines all configurable aspects of the logging behavior including
// output format, destination, log level, rotation settings, and additional options.
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
	// EnableRotation enables log file rotation
	EnableRotation bool `koanf:"enable_rotation" default:"false"`
	// RotateInterval sets the rotation interval (daily, hourly)
	RotateInterval string `koanf:"rotate_interval" default:"daily"`
	// MaxFileSize sets the maximum file size in bytes before rotation (0 = no size limit)
	MaxFileSize int64 `koanf:"max_file_size" default:"524288000"` // 500MB
	// MaxAge sets the maximum number of days to retain old log files
	MaxAge int `koanf:"max_age" default:"7"`
	// TimeFormat sets the time format for rotated file names
	TimeFormat string `koanf:"time_format" default:"2006-01-02"`
}

// LoggerPlugin implements the logger plugin that provides structured logging
// capabilities with configurable output formats, destinations, and rotation.
type LoggerPlugin struct {
	// mu protects concurrent access to plugin state
	mu sync.RWMutex
	// logger is the configured slog.Logger instance
	logger *slog.Logger
	// file holds the log file handle when file output is enabled
	file *os.File
	// config stores the current plugin configuration
	config *LoggerConfig
	// currentLogDate tracks the current log file date for rotation
	currentLogDate string
	// currentFileSize tracks the current log file size
	currentFileSize int64
	// fileSequence tracks the sequence number for same-day files
	fileSequence int
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
// If rotation is enabled, it handles file rotation logic.
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

	if p.config.EnableRotation {
		return p.createRotatingFileWriter()
	}

	// Open file without rotation
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

// createRotatingFileWriter creates a rotating file writer that handles
// log rotation based on time and file size.
//
// Returns:
//   - io.Writer: The rotating file writer
//   - error: An error if creation fails, nil otherwise
func (p *LoggerPlugin) createRotatingFileWriter() (io.Writer, error) {
	// Get current log file path
	logPath, err := p.getCurrentLogPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get current log path: %w", err)
	}

	// Check if file already exists to determine if it's a new file
	fileExists := true
	if _, err = os.Stat(logPath); os.IsNotExist(err) {
		fileExists = false
	}

	// Open the log file
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Set current file size based on whether it's a new or existing file
	if fileExists {
		// Get current file size for existing file
		stat, err := file.Stat()
		if err != nil {
			return nil, fmt.Errorf("failed to get file stats: %w", err)
		}
		p.currentFileSize = stat.Size()
	} else {
		// New file starts with size 0
		p.currentFileSize = 0
	}

	p.file = file

	// Create rotating writer
	return &rotatingWriter{
		plugin: p,
		file:   file,
	}, nil
}

// rotatingWriter wraps a file and handles rotation logic
type rotatingWriter struct {
	plugin *LoggerPlugin
	file   *os.File
}

// Write implements io.Writer interface with rotation logic
func (rw *rotatingWriter) Write(p []byte) (n int, err error) {
	rw.plugin.mu.Lock()
	defer rw.plugin.mu.Unlock()

	// Check if rotation is needed
	if rw.plugin.needsRotation() {
		if err = rw.plugin.rotateFile(); err != nil {
			return 0, fmt.Errorf("failed to rotate log file: %w", err)
		}
		// Update file reference after rotation
		rw.file = rw.plugin.file
	}

	// Write to current file
	n, err = rw.file.Write(p)
	if err == nil {
		rw.plugin.currentFileSize += int64(n)
	}
	return n, err
}

// needsRotation checks if log rotation is needed based on time or file size
func (p *LoggerPlugin) needsRotation() bool {
	now := time.Now()
	currentDate := now.Format(p.config.TimeFormat)

	// Check time-based rotation
	if p.currentLogDate != currentDate {
		return true
	}

	// Check size-based rotation (if MaxFileSize > 0)
	if p.config.MaxFileSize > 0 && p.currentFileSize >= p.config.MaxFileSize {
		return true
	}

	return false
}

// rotateFile performs the actual file rotation
func (p *LoggerPlugin) rotateFile() error {
	// Close current file
	if p.file != nil {
		if err := p.file.Close(); err != nil {
			return fmt.Errorf("failed to close current log file: %w", err)
		}
	}

	// Get new log file path
	newLogPath, err := p.getCurrentLogPath()
	if err != nil {
		return fmt.Errorf("failed to get new log path: %w", err)
	}

	// Open new log file
	file, err := os.OpenFile(newLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open new log file: %w", err)
	}

	// Update plugin state
	p.file = file
	p.currentFileSize = 0

	return nil
}

// getCurrentLogPath generates the current log file path based on rotation settings
func (p *LoggerPlugin) getCurrentLogPath() (string, error) {
	now := time.Now()
	currentDate := now.Format(p.config.TimeFormat)

	// Update current log date
	p.currentLogDate = currentDate

	// Get base path without extension
	basePath := p.config.FilePath
	ext := filepath.Ext(basePath)
	baseWithoutExt := strings.TrimSuffix(basePath, ext)

	// Generate dated filename
	datedPath := fmt.Sprintf("%s-%s%s", baseWithoutExt, currentDate, ext)

	// Check if we need sequence number for size-based rotation
	if p.config.MaxFileSize > 0 {
		// Find the next available sequence number
		dir := filepath.Dir(p.config.FilePath)
		baseName := filepath.Base(baseWithoutExt)
		sequence := p.findNextSequence(dir, baseName, currentDate)
		if sequence > 0 {
			datedPath = fmt.Sprintf("%s-%s-%03d%s", baseWithoutExt, currentDate, sequence, ext)
		}
		p.fileSequence = sequence
	}

	return datedPath, nil
}

// findNextSequence finds the next available sequence number for the current date
func (p *LoggerPlugin) findNextSequence(dir, baseName, currentDate string) int {
	ext := filepath.Ext(p.config.FilePath)
	pattern := fmt.Sprintf("%s-%s*%s", baseName, currentDate, ext)

	files, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return 0 // Start with no sequence if glob fails
	}

	if len(files) == 0 {
		return 0 // Start with no sequence if no files exist
	}

	maxSequence := 0
	sequenceRegex := regexp.MustCompile(fmt.Sprintf(`%s-%s(?:-([0-9]+))?%s$`,
		regexp.QuoteMeta(baseName),
		regexp.QuoteMeta(currentDate),
		regexp.QuoteMeta(ext)))

	// Find the highest sequence number
	for _, file := range files {
		matches := sequenceRegex.FindStringSubmatch(filepath.Base(file))
		if len(matches) > 1 && matches[1] != "" {
			// File with sequence number (e.g., app-2024-01-15-001.log)
			if seq, err := strconv.Atoi(matches[1]); err == nil {
				if seq > maxSequence {
					maxSequence = seq
				}
			}
		}
	}

	// Return the next sequence number
	return maxSequence + 1
}

// cleanupOldLogs removes old log files based on MaxAge setting
func (p *LoggerPlugin) cleanupOldLogs() error {
	if p.config.MaxAge <= 0 {
		return nil // No cleanup needed
	}

	dir := filepath.Dir(p.config.FilePath)
	baseName := filepath.Base(p.config.FilePath)
	ext := filepath.Ext(baseName)
	baseWithoutExt := strings.TrimSuffix(baseName, ext)

	// Find all log files matching the pattern
	pattern := fmt.Sprintf("%s-*%s", baseWithoutExt, ext)
	files, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return fmt.Errorf("failed to glob log files: %w", err)
	}

	// Parse dates and remove old files
	cutoffDate := time.Now().AddDate(0, 0, -p.config.MaxAge)
	prefixLen := len(baseWithoutExt) + 1 // +1 for the dash

	for _, file := range files {
		fileName := filepath.Base(file)

		// Extract date part from filename (e.g., "app-2024-01-15.log" or "app-2024-01-15-001.log")
		if len(fileName) < prefixLen+len(p.config.TimeFormat) {
			continue // Filename too short to contain a valid date
		}

		// Extract the date portion
		datePart := fileName[prefixLen : prefixLen+len(p.config.TimeFormat)]

		// Parse the date
		if fileDate, err := time.Parse(p.config.TimeFormat, datePart); err == nil {
			if fileDate.Before(cutoffDate) {
				if err := os.Remove(file); err != nil {
					// Log error but continue cleanup
					continue
				}
			}
		}
	}

	return nil
}
