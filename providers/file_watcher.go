package providers

import (
	"os"
	"path/filepath"
	"sync"

	"slices"

	"github.com/fsnotify/fsnotify"
	"github.com/knadh/koanf/providers/file"
)

// FileWatcher wraps the koanf file provider with enhanced watching capabilities
// that monitor the parent directory to handle atomic file operations properly.
type FileWatcher struct {
	filePath string
	provider *file.File
	watcher  *fsnotify.Watcher
	callback func(event any, err error)
	mu       sync.RWMutex
	watching bool
}

// NewFileWatcher creates a new FileWatcher that monitors the parent directory
// of the given file path to handle atomic file operations properly.
func NewFileWatcher(filePath string) (*FileWatcher, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}

	provider := file.Provider(absPath)

	return &FileWatcher{
		filePath: absPath,
		provider: provider,
	}, nil
}

// Read implements the koanf.Provider interface
func (fw *FileWatcher) Read() (map[string]any, error) {
	return fw.provider.Read()
}

// ReadBytes implements the koanf.Provider interface
func (fw *FileWatcher) ReadBytes() ([]byte, error) {
	return fw.provider.ReadBytes()
}

// Watch starts monitoring the parent directory for changes to the target file.
// This approach handles atomic file operations (like those performed by text editors)
// that would otherwise break direct file watching.
func (fw *FileWatcher) Watch(cb func(event any, err error)) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.watching {
		return nil // Already watching
	}

	fw.callback = cb

	// Create fsnotify watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	fw.watcher = watcher

	// Watch the parent directory instead of the file directly
	parentDir := filepath.Dir(fw.filePath)
	err = fw.watcher.Add(parentDir)
	if err != nil {
		fw.watcher.Close()
		return err
	}

	fw.watching = true

	// Start the event processing goroutine
	go fw.processEvents()

	return nil
}

// Unwatch stops monitoring the file for changes
func (fw *FileWatcher) Unwatch() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if !fw.watching {
		return nil // Not watching
	}

	fw.watching = false

	if fw.watcher != nil {
		err := fw.watcher.Close()
		fw.watcher = nil
		return err
	}

	return nil
}

// processEvents handles fsnotify events and filters them for the target file
func (fw *FileWatcher) processEvents() {
	for {
		select {
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return // Watcher closed
			}

			// Filter events to only process our target file
			if fw.isTargetFileEvent(event) {
				// Call the callback for any write, create, or rename operation
				// on our target file
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) {
					fw.mu.RLock()
					cb := fw.callback
					fw.mu.RUnlock()

					if cb != nil {
						cb(nil, nil) // koanf file provider always passes nil event
					}
				}
			}

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return // Watcher closed
			}

			fw.mu.RLock()
			cb := fw.callback
			fw.mu.RUnlock()

			if cb != nil {
				cb(nil, err)
			}
		}
	}
}

// isTargetFileEvent checks if the fsnotify event is for our target file
func (fw *FileWatcher) isTargetFileEvent(event fsnotify.Event) bool {
	// Get the absolute path of the event
	eventPath, err := filepath.Abs(event.Name)
	if err != nil {
		return false
	}

	// Check if it's exactly our target file
	if eventPath == fw.filePath {
		return true
	}

	// Also check if it's a temporary file that might be renamed to our target
	// This handles cases where editors create temp files and rename them
	fileName := filepath.Base(fw.filePath)
	eventFileName := filepath.Base(eventPath)

	// Check for common temporary file patterns that editors use
	tempPatterns := []string{
		"." + fileName + ".swp", // vim swap files
		"." + fileName + ".tmp", // generic temp files
		fileName + ".tmp",       // generic temp files
		fileName + "~",          // backup files
		".#" + fileName,         // emacs temp files
	}

	if slices.Contains(tempPatterns, eventFileName) {
		return false // Ignore temp files
	}

	// For rename events, check if the event is renaming TO our target file
	if event.Has(fsnotify.Rename) || event.Has(fsnotify.Create) {
		// Check if a file was created/renamed to our target path
		if _, err := os.Stat(fw.filePath); err == nil {
			return true
		}
	}

	return false
}

// IsWatching returns true if the file is currently being watched
func (fw *FileWatcher) IsWatching() bool {
	fw.mu.RLock()
	defer fw.mu.RUnlock()
	return fw.watching
}

// GetFilePath returns the absolute path of the file being watched
func (fw *FileWatcher) GetFilePath() string {
	return fw.filePath
}
