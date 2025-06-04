package builtins

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nextpkg/vcfg/defaults"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggerPlugin_Start(t *testing.T) {
	tests := []struct {
		name    string
		config  *LoggerConfig
		wantErr bool
	}{
		{
			name: "valid json stdout config",
			config: &LoggerConfig{
				Level:     "info",
				Format:    "json",
				Output:    "stdout",
				AddSource: true,
			},
			wantErr: false,
		},
		{
			name: "valid text stderr config",
			config: &LoggerConfig{
				Level:     "debug",
				Format:    "text",
				Output:    "stderr",
				AddSource: false,
			},
			wantErr: false,
		},
		{
			name: "invalid log level",
			config: &LoggerConfig{
				Level:  "invalid",
				Format: "json",
				Output: "stdout",
			},
			wantErr: true,
		},
		{
			name: "invalid format",
			config: &LoggerConfig{
				Level:  "info",
				Format: "invalid",
				Output: "stdout",
			},
			wantErr: true,
		},
		{
			name: "invalid output",
			config: &LoggerConfig{
				Level:  "info",
				Format: "json",
				Output: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := &LoggerPlugin{}
			err := plugin.Startup(context.Background(), tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, plugin.logger)
				// Clean up
				_ = plugin.Shutdown(context.Background())
			}
		})
	}
}

func TestLoggerPlugin_FileOutput(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	config := &LoggerConfig{
		Level:    "info",
		Format:   "json",
		Output:   "file",
		FilePath: logFile,
	}

	plugin := &LoggerPlugin{}
	err := plugin.Startup(context.Background(), config)
	require.NoError(t, err)
	defer plugin.Shutdown(context.Background())

	// Write a test log
	logger := GetLogger()
	logger.Info("test message", "key", "value")

	// Give some time for the log to be written
	time.Sleep(100 * time.Millisecond)

	// Check if file exists and contains the log
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "test message")
	assert.Contains(t, string(content), "key")
	assert.Contains(t, string(content), "value")
}

func TestLoggerPlugin_BothOutput(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	config := &LoggerConfig{
		Level:    "info",
		Format:   "json",
		Output:   "both",
		FilePath: logFile,
	}

	plugin := &LoggerPlugin{}
	err := plugin.Startup(context.Background(), config)
	require.NoError(t, err)
	defer plugin.Shutdown(context.Background())

	// Write a test log
	logger := GetLogger()
	logger.Info("test both output", "test", true)

	// Give some time for the log to be written
	time.Sleep(100 * time.Millisecond)

	// Check if file contains the log
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "test both output")
}

func TestLoggerPlugin_AddSource(t *testing.T) {
	tests := []struct {
		name      string
		addSource bool
	}{
		{
			name:      "with source info",
			addSource: true,
		},
		{
			name:      "without source info",
			addSource: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			logFile := filepath.Join(tempDir, "test.log")

			config := &LoggerConfig{
				Level:     "info",
				Format:    "json",
				Output:    "file",
				FilePath:  logFile,
				AddSource: tt.addSource,
			}

			plugin := &LoggerPlugin{}
			err := plugin.Startup(context.Background(), config)
			require.NoError(t, err)
			defer plugin.Shutdown(context.Background())

			// Write a test log
			logger := GetLogger()
			logger.Info("test source info")

			// Give some time for the log to be written
			time.Sleep(100 * time.Millisecond)

			// Check file content
			content, err := os.ReadFile(logFile)
			require.NoError(t, err)

			if tt.addSource {
				// Should contain source information
				assert.Contains(t, string(content), "\"source\"")
				assert.Contains(t, string(content), "logger_test.go")
			} else {
				// Should not contain source information
				assert.NotContains(t, string(content), "\"source\"")
			}
		})
	}
}

func TestLoggerPlugin_Reload(t *testing.T) {
	plugin := &LoggerPlugin{}

	// Start with initial config
	initialConfig := &LoggerConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}
	err := plugin.Startup(context.Background(), initialConfig)
	require.NoError(t, err)

	// Reload with new config
	newConfig := &LoggerConfig{
		Level:     "debug",
		Format:    "text",
		Output:    "stderr",
		AddSource: true,
	}
	err = plugin.Reload(context.Background(), newConfig)
	assert.NoError(t, err)
	assert.NotNil(t, plugin.logger)

	// Clean up
	_ = plugin.Shutdown(context.Background())
}

func TestLoggerPlugin_Stop(t *testing.T) {
	plugin := &LoggerPlugin{}

	config := &LoggerConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}
	err := plugin.Startup(context.Background(), config)
	require.NoError(t, err)

	err = plugin.Shutdown(context.Background())
	assert.NoError(t, err)
	assert.Nil(t, plugin.logger)
	assert.Nil(t, plugin.config)
}

func TestLoggerPlugin_InvalidConfig(t *testing.T) {
	plugin := &LoggerPlugin{}

	// Test with invalid config type
	err := plugin.Startup(context.Background(), "invalid config")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid logger config type")
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected slog.Level
		wantErr  bool
	}{
		{"debug", "debug", slog.LevelDebug, false},
		{"info", "info", slog.LevelInfo, false},
		{"warn", "warn", slog.LevelWarn, false},
		{"warning", "warning", slog.LevelWarn, false},
		{"error", "error", slog.LevelError, false},
		{"uppercase", "INFO", slog.LevelInfo, false},
		{"invalid", "invalid", slog.LevelInfo, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level, err := parseLogLevel(tt.level)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, level)
			}
		})
	}
}

func TestGetLogger(t *testing.T) {
	// Test getting logger when none is set
	logger := GetLogger()
	assert.NotNil(t, logger)

	// Test getting logger after setting one
	plugin := &LoggerPlugin{}
	config := &LoggerConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}
	err := plugin.Startup(context.Background(), config)
	require.NoError(t, err)
	defer plugin.Shutdown(context.Background())

	logger = GetLogger()
	assert.NotNil(t, logger)
}

func TestLoggerPlugin_TextFormat(t *testing.T) {
	// Create a custom writer to capture output
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	config := &LoggerConfig{
		Level:    "info",
		Format:   "text",
		Output:   "file",
		FilePath: logFile,
	}

	plugin := &LoggerPlugin{}
	err := plugin.Startup(context.Background(), config)
	require.NoError(t, err)
	defer plugin.Shutdown(context.Background())

	// Write a test log
	logger := GetLogger()
	logger.Info("test text format", "key", "value")

	// Give some time for the log to be written
	time.Sleep(100 * time.Millisecond)

	// Check file content
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)

	// Text format should not be JSON
	var jsonData map[string]interface{}
	err = json.Unmarshal(content, &jsonData)
	assert.Error(t, err) // Should fail to parse as JSON

	// Should contain the message
	assert.Contains(t, string(content), "test text format")
}

func TestLoggerPlugin_DefaultValues(t *testing.T) {
	plugin := &LoggerPlugin{}

	// Start with minimal config
	config := &LoggerConfig{}

	// Set default values
	err := defaults.SetDefaults(config)
	require.NoError(t, err)

	err = plugin.Startup(context.Background(), config)
	require.NoError(t, err)
	defer plugin.Shutdown(context.Background())

	// Check that defaults were applied
	assert.Equal(t, "info", plugin.config.Level)
	assert.Equal(t, "json", plugin.config.Format)
	assert.Equal(t, "stdout", plugin.config.Output)
	assert.Equal(t, "./app.log", plugin.config.FilePath)
	assert.False(t, plugin.config.AddSource)
}
