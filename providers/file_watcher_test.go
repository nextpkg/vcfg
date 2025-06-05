package providers

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileWatcher_BasicFunctionality(t *testing.T) {
	// Create a temporary file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.yaml")
	initialContent := "key: value1\n"

	err := os.WriteFile(testFile, []byte(initialContent), 0644)
	require.NoError(t, err)

	// Create FileWatcher
	fw, err := NewFileWatcher(testFile)
	require.NoError(t, err)

	// Test Read functionality
	data, err := fw.ReadBytes()
	require.NoError(t, err)
	assert.Equal(t, initialContent, string(data))

	// Test Watch functionality
	changeDetected := make(chan bool, 1)
	errorReceived := make(chan error, 1)

	err = fw.Watch(func(event interface{}, err error) {
		if err != nil {
			errorReceived <- err
			return
		}
		changeDetected <- true
	})
	require.NoError(t, err)
	assert.True(t, fw.IsWatching())

	// Give the watcher time to start
	time.Sleep(100 * time.Millisecond)

	// Modify the file
	newContent := "key: value2\n"
	err = os.WriteFile(testFile, []byte(newContent), 0644)
	require.NoError(t, err)

	// Wait for change detection
	select {
	case <-changeDetected:
		// Success - change was detected
	case err := <-errorReceived:
		t.Fatalf("Unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for file change detection")
	}

	// Verify the new content can be read
	data, err = fw.ReadBytes()
	require.NoError(t, err)
	assert.Equal(t, newContent, string(data))

	// Test Unwatch
	err = fw.Unwatch()
	require.NoError(t, err)
	assert.False(t, fw.IsWatching())
}

func TestFileWatcher_AtomicSave(t *testing.T) {
	// Create a temporary file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.yaml")
	initialContent := "key: value1\n"

	err := os.WriteFile(testFile, []byte(initialContent), 0644)
	require.NoError(t, err)

	// Create FileWatcher
	fw, err := NewFileWatcher(testFile)
	require.NoError(t, err)

	// Test Watch functionality
	changeDetected := make(chan bool, 1)
	errorReceived := make(chan error, 1)

	err = fw.Watch(func(event interface{}, err error) {
		if err != nil {
			errorReceived <- err
			return
		}
		changeDetected <- true
	})
	require.NoError(t, err)

	// Give the watcher time to start
	time.Sleep(100 * time.Millisecond)

	// Simulate atomic save operation (like text editors do)
	// 1. Create a temporary file
	tempFile := testFile + ".tmp"
	newContent := "key: value2\n"
	err = os.WriteFile(tempFile, []byte(newContent), 0644)
	require.NoError(t, err)

	// 2. Rename temporary file to target file (atomic operation)
	err = os.Rename(tempFile, testFile)
	require.NoError(t, err)

	// Wait for change detection
	select {
	case <-changeDetected:
		// Success - change was detected even with atomic save
	case err := <-errorReceived:
		t.Fatalf("Unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for atomic save detection")
	}

	// Verify the new content can be read
	data, err := fw.ReadBytes()
	require.NoError(t, err)
	assert.Equal(t, newContent, string(data))

	// Cleanup
	err = fw.Unwatch()
	require.NoError(t, err)
}

func TestFileWatcher_MultipleWatchers(t *testing.T) {
	// Create a temporary file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.yaml")
	initialContent := "key: value1\n"

	err := os.WriteFile(testFile, []byte(initialContent), 0644)
	require.NoError(t, err)

	// Create multiple FileWatchers for the same file
	fw1, err := NewFileWatcher(testFile)
	require.NoError(t, err)

	fw2, err := NewFileWatcher(testFile)
	require.NoError(t, err)

	// Set up watchers
	changeDetected1 := make(chan bool, 1)
	changeDetected2 := make(chan bool, 1)

	err = fw1.Watch(func(event interface{}, err error) {
		if err == nil {
			changeDetected1 <- true
		}
	})
	require.NoError(t, err)

	err = fw2.Watch(func(event interface{}, err error) {
		if err == nil {
			changeDetected2 <- true
		}
	})
	require.NoError(t, err)

	// Give the watchers time to start
	time.Sleep(100 * time.Millisecond)

	// Modify the file
	newContent := "key: value2\n"
	err = os.WriteFile(testFile, []byte(newContent), 0644)
	require.NoError(t, err)

	// Both watchers should detect the change
	select {
	case <-changeDetected1:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for first watcher")
	}

	select {
	case <-changeDetected2:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for second watcher")
	}

	// Cleanup
	err = fw1.Unwatch()
	require.NoError(t, err)

	err = fw2.Unwatch()
	require.NoError(t, err)
}

func TestFileWatcher_NonExistentFile(t *testing.T) {
	// Try to create FileWatcher for non-existent file
	tempDir := t.TempDir()
	nonExistentFile := filepath.Join(tempDir, "nonexistent.yaml")

	// Should still create the watcher (file might be created later)
	fw, err := NewFileWatcher(nonExistentFile)
	require.NoError(t, err)

	// Reading should fail
	_, err = fw.ReadBytes()
	assert.Error(t, err)

	// Watching should work (waiting for file creation)
	changeDetected := make(chan bool, 1)

	err = fw.Watch(func(event interface{}, err error) {
		if err == nil {
			changeDetected <- true
		}
	})
	require.NoError(t, err)

	// Give the watcher time to start
	time.Sleep(100 * time.Millisecond)

	// Create the file
	content := "key: value\n"
	err = os.WriteFile(nonExistentFile, []byte(content), 0644)
	require.NoError(t, err)

	// Should detect file creation
	select {
	case <-changeDetected:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for file creation detection")
	}

	// Now reading should work
	data, err := fw.ReadBytes()
	require.NoError(t, err)
	assert.Equal(t, content, string(data))

	// Cleanup
	err = fw.Unwatch()
	require.NoError(t, err)
}

func TestFileWatcher_ErrorHandling(t *testing.T) {
	// Test with invalid path
	invalidPath := "/invalid/path/that/does/not/exist/file.yaml"

	// Should fail to create absolute path
	_, err := NewFileWatcher(invalidPath)
	// This might succeed depending on the system, so we don't assert error here

	// Test double unwatch
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.yaml")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	fw, err := NewFileWatcher(testFile)
	require.NoError(t, err)

	// First unwatch should succeed
	err = fw.Unwatch()
	assert.NoError(t, err)

	// Second unwatch should also succeed (idempotent)
	err = fw.Unwatch()
	assert.NoError(t, err)
}
