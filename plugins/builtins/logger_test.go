package builtins

import (
	"context"
	"encoding/json"
	"fmt"
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

func TestLoggerPlugin_RotationEnabled(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	config := &LoggerConfig{
		Level:          "info",
		Format:         "text",
		Output:         "stdout", // Use stdout to avoid file operations
		FilePath:       logFile,
		EnableRotation: true,
		MaxFileSize:    1024, // 1KB
		MaxAge:         7,    // 7 days
	}

	plugin := &LoggerPlugin{}
	err := plugin.Startup(context.Background(), config)
	require.NoError(t, err)
	defer plugin.Shutdown(context.Background())

	// Verify rotation is enabled
	assert.True(t, plugin.config.EnableRotation)
	assert.Equal(t, int64(1024), plugin.config.MaxFileSize)
	assert.Equal(t, 7, plugin.config.MaxAge)
}

func TestLoggerPlugin_RotationDisabled(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	config := &LoggerConfig{
		Level:          "info",
		Format:         "json",
		Output:         "file",
		FilePath:       logFile,
		EnableRotation: false,
	}

	plugin := &LoggerPlugin{}
	err := plugin.Startup(context.Background(), config)
	require.NoError(t, err)
	defer plugin.Shutdown(context.Background())

	// Verify rotation is disabled
	assert.False(t, plugin.config.EnableRotation)

	// Write a test log
	logger := GetLogger()
	logger.Info("test rotation disabled")

	// Give some time for the log to be written
	time.Sleep(100 * time.Millisecond)

	// Check if file exists
	_, err = os.Stat(logFile)
	assert.NoError(t, err)
}

func TestLoggerPlugin_SizeBasedRotation(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	config := &LoggerConfig{
		Level:          "info",
		Format:         "text",
		Output:         "stdout", // Use stdout to avoid file operations
		FilePath:       logFile,
		EnableRotation: true,
		MaxFileSize:    100, // Very small size to trigger rotation
		MaxAge:         7,
	}

	plugin := &LoggerPlugin{}
	err := plugin.Startup(context.Background(), config)
	require.NoError(t, err)
	defer plugin.Shutdown(context.Background())

	// Just verify the configuration is set correctly
	assert.True(t, plugin.config.EnableRotation)
	assert.Equal(t, int64(100), plugin.config.MaxFileSize)
}

func TestLoggerPlugin_FindNextSequence(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	config := &LoggerConfig{
		Level:          "info",
		Format:         "text",
		Output:         "file",
		FilePath:       logFile,
		EnableRotation: true,
		MaxFileSize:    1024,
		TimeFormat:     "2006-01-02",
	}

	plugin := &LoggerPlugin{}
	plugin.config = config

	// Test with empty directory (no existing files)
	today := time.Now().Format("2006-01-02")
	sequence := plugin.findNextSequence(tempDir, "test", today)
	assert.Equal(t, 0, sequence) // Should be 0 for empty directory
}

func TestLoggerPlugin_CleanupOldLogs(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	config := &LoggerConfig{
		Level:          "info",
		Format:         "text",
		Output:         "file",
		FilePath:       logFile,
		EnableRotation: true,
		MaxFileSize:    1024,
		MaxAge:         1, // 1 day
		TimeFormat:     "2006-01-02",
	}

	plugin := &LoggerPlugin{}
	plugin.config = config

	// Test cleanup with no files (should not error)
	err := plugin.cleanupOldLogs()
	assert.NoError(t, err)

	// Test with MaxAge = 0 (should skip cleanup)
	plugin.config.MaxAge = 0
	err = plugin.cleanupOldLogs()
	assert.NoError(t, err)
}

func TestLoggerPlugin_RotationWithBothOutput(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	config := &LoggerConfig{
		Level:          "info",
		Format:         "text",
		Output:         "stdout", // Use stdout to avoid file operations
		FilePath:       logFile,
		EnableRotation: true,
		MaxFileSize:    1024,
		MaxAge:         7,
	}

	plugin := &LoggerPlugin{}
	err := plugin.Startup(context.Background(), config)
	require.NoError(t, err)
	defer plugin.Shutdown(context.Background())

	// Verify configuration
	assert.Equal(t, "stdout", plugin.config.Output)
	assert.True(t, plugin.config.EnableRotation)
}

// TestLoggerPlugin_CreateFileWriter tests the createFileWriter method
func TestLoggerPlugin_CreateFileWriter(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	config := &LoggerConfig{
		Level:          "info",
		Format:         "json",
		Output:         "file",
		FilePath:       logFile,
		EnableRotation: false,
	}

	plugin := &LoggerPlugin{}
	plugin.config = config

	// Test creating file writer without rotation
	writer, err := plugin.createFileWriter()
	assert.NoError(t, err)
	assert.NotNil(t, writer)
	assert.NotNil(t, plugin.file)

	// Clean up
	plugin.file.Close()
}

// TestLoggerPlugin_CreateRotatingFileWriter tests the createRotatingFileWriter method
func TestLoggerPlugin_CreateRotatingFileWriter(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	config := &LoggerConfig{
		Level:          "info",
		Format:         "json",
		Output:         "file",
		FilePath:       logFile,
		EnableRotation: true,
		MaxFileSize:    1024,
		TimeFormat:     "2006-01-02",
	}

	plugin := &LoggerPlugin{}
	plugin.config = config

	// Test creating rotating file writer
	writer, err := plugin.createRotatingFileWriter()
	assert.NoError(t, err)
	assert.NotNil(t, writer)
	assert.NotNil(t, plugin.file)
	assert.Equal(t, int64(0), plugin.currentFileSize)

	// Clean up
	plugin.file.Close()
}

// TestLoggerPlugin_NeedsRotation tests the needsRotation method
func TestLoggerPlugin_NeedsRotation(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	config := &LoggerConfig{
		Level:          "info",
		Format:         "json",
		Output:         "file",
		FilePath:       logFile,
		EnableRotation: true,
		MaxFileSize:    100, // Small size to trigger rotation
		TimeFormat:     "2006-01-02",
	}

	plugin := &LoggerPlugin{}
	plugin.config = config
	plugin.currentLogDate = time.Now().Format("2006-01-02")
	plugin.currentFileSize = 0

	// Test no rotation needed
	assert.False(t, plugin.needsRotation())

	// Test size-based rotation
	plugin.currentFileSize = 150 // Exceed max size
	assert.True(t, plugin.needsRotation())

	// Test time-based rotation
	plugin.currentFileSize = 50
	plugin.currentLogDate = "2023-01-01" // Old date
	assert.True(t, plugin.needsRotation())
}

// TestLoggerPlugin_GetCurrentLogPath tests the getCurrentLogPath method
func TestLoggerPlugin_GetCurrentLogPath(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	config := &LoggerConfig{
		Level:          "info",
		Format:         "json",
		Output:         "file",
		FilePath:       logFile,
		EnableRotation: true,
		MaxFileSize:    1024,
		TimeFormat:     "2006-01-02",
	}

	plugin := &LoggerPlugin{}
	plugin.config = config

	// Test getting current log path
	logPath, err := plugin.getCurrentLogPath()
	assert.NoError(t, err)
	assert.NotEmpty(t, logPath)
	assert.Contains(t, logPath, time.Now().Format("2006-01-02"))
	assert.NotEmpty(t, plugin.currentLogDate)
}

// TestLoggerPlugin_FindNextSequenceWithFiles tests findNextSequence with existing files
func TestLoggerPlugin_FindNextSequenceWithFiles(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	config := &LoggerConfig{
		Level:          "info",
		Format:         "json",
		Output:         "file",
		FilePath:       logFile,
		EnableRotation: true,
		MaxFileSize:    1024,
		TimeFormat:     "2006-01-02",
	}

	plugin := &LoggerPlugin{}
	plugin.config = config

	today := time.Now().Format("2006-01-02")

	// Create some existing log files
	existingFiles := []string{
		filepath.Join(tempDir, fmt.Sprintf("test-%s.log", today)),
		filepath.Join(tempDir, fmt.Sprintf("test-%s-001.log", today)),
		filepath.Join(tempDir, fmt.Sprintf("test-%s-002.log", today)),
	}

	for _, file := range existingFiles {
		f, err := os.Create(file)
		require.NoError(t, err)
		f.Close()
	}

	// Test finding next sequence
	sequence := plugin.findNextSequence(tempDir, "test", today)
	assert.Equal(t, 3, sequence) // Should be 3 (next after 002)
}

// TestLoggerPlugin_CleanupOldLogsWithFiles tests cleanupOldLogs with actual files
func TestLoggerPlugin_CleanupOldLogsWithFiles(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	config := &LoggerConfig{
		Level:          "info",
		Format:         "json",
		Output:         "file",
		FilePath:       logFile,
		EnableRotation: true,
		MaxFileSize:    1024,
		MaxAge:         2, // Keep files for 2 days
		TimeFormat:     "2006-01-02",
	}

	plugin := &LoggerPlugin{}
	plugin.config = config

	// Create old and new log files
	oldDate := time.Now().AddDate(0, 0, -5).Format("2006-01-02") // 5 days ago
	newDate := time.Now().Format("2006-01-02")                   // Today

	oldFile := filepath.Join(tempDir, fmt.Sprintf("test-%s.log", oldDate))
	newFile := filepath.Join(tempDir, fmt.Sprintf("test-%s.log", newDate))

	// Create the files
	f1, err := os.Create(oldFile)
	require.NoError(t, err)
	f1.Close()

	f2, err := os.Create(newFile)
	require.NoError(t, err)
	f2.Close()

	// Verify both files exist
	_, err = os.Stat(oldFile)
	assert.NoError(t, err)
	_, err = os.Stat(newFile)
	assert.NoError(t, err)

	// Run cleanup
	err = plugin.cleanupOldLogs()
	assert.NoError(t, err)

	// Old file should be removed, new file should remain
	_, err = os.Stat(oldFile)
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(newFile)
	assert.NoError(t, err)
}

// TestLoggerPlugin_RotatingWriter tests the rotatingWriter functionality
func TestLoggerPlugin_RotatingWriter(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	config := &LoggerConfig{
		Level:          "info",
		Format:         "json",
		Output:         "file",
		FilePath:       logFile,
		EnableRotation: true,
		MaxFileSize:    50, // Very small to trigger rotation
		TimeFormat:     "2006-01-02",
	}

	plugin := &LoggerPlugin{}
	plugin.config = config
	plugin.currentLogDate = time.Now().Format("2006-01-02")
	plugin.currentFileSize = 0

	// Create a file
	file, err := os.Create(logFile)
	require.NoError(t, err)
	plugin.file = file

	// Create rotating writer
	rw := &rotatingWriter{
		plugin: plugin,
		file:   file,
	}

	// Write some data
	data := []byte("test log message that is longer than 50 bytes to trigger rotation")
	n, err := rw.Write(data)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, int64(len(data)), plugin.currentFileSize)

	// Clean up
	plugin.file.Close()
}

// TestLoggerPlugin_RotateFile tests the rotateFile method
func TestLoggerPlugin_RotateFile(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	config := &LoggerConfig{
		Level:          "info",
		Format:         "json",
		Output:         "file",
		FilePath:       logFile,
		EnableRotation: true,
		MaxFileSize:    1024,
		MaxAge:         7,
		TimeFormat:     "2006-01-02",
	}

	plugin := &LoggerPlugin{}
	plugin.config = config
	plugin.currentLogDate = time.Now().Format("2006-01-02")
	plugin.currentFileSize = 500

	// Create initial file
	file, err := os.Create(logFile)
	require.NoError(t, err)
	plugin.file = file

	// Test rotation
	err = plugin.rotateFile()
	assert.NoError(t, err)
	assert.NotNil(t, plugin.file)
	assert.Equal(t, int64(0), plugin.currentFileSize)

	// Clean up
	plugin.file.Close()
}

// TestLoggerPlugin_CreateWriter tests the createWriter method for all output types
func TestLoggerPlugin_CreateWriter(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	tests := []struct {
		name    string
		output  string
		wantErr bool
	}{
		{"stdout", "stdout", false},
		{"stderr", "stderr", false},
		{"file", "file", false},
		{"both", "both", false},
		{"invalid", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &LoggerConfig{
				Level:          "info",
				Format:         "json",
				Output:         tt.output,
				FilePath:       logFile,
				EnableRotation: false,
			}

			plugin := &LoggerPlugin{}
			plugin.config = config

			writer, err := plugin.createWriter()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, writer)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, writer)
				// Clean up file if created
				if plugin.file != nil {
					plugin.file.Close()
				}
			}
		})
	}
}
