package file

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/nextpkg/vcfg/internal/viper"
)

// FileSource represents a file-based configuration source
type FileSource struct {
	Path       string
	Format     string // e.g., "json", "yaml", "toml"
	watcher    *fsnotify.Watcher
	changeChan chan bool
	mutex      sync.Mutex
	isWatching bool
}

// NewFileSource creates a new file configuration source
// It automatically detects the file format based on the file extension
func NewFileSource(path string) *FileSource {
	// Auto-detect format
	format := filepath.Ext(path)
	if format != "" {
		format = format[1:] // Remove the dot
	}

	return &FileSource{
		Path:       path,
		Format:     format,
		changeChan: nil,
		isWatching: false,
	}
}

// Read implements the Source interface, reading configuration from a file
func (f *FileSource) Read() (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigFile(f.Path)

	if f.Format != "" {
		v.SetConfigType(f.Format)
	}

	err := v.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", f.Path, err)
	}

	return v, nil
}

// Watch implements the Source interface, monitoring file changes
// It returns a channel that receives a notification when the file changes
func (f *FileSource) Watch() (<-chan bool, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	// If already watching, return the existing channel
	if f.isWatching && f.changeChan != nil {
		return f.changeChan, nil
	}

	// Create file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	// Get absolute path and directory
	absPath, err := filepath.Abs(f.Path)
	if err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to get absolute path for config file: %w", err)
	}

	dir := filepath.Dir(absPath)

	// Watch the directory instead of the file itself to catch recreations
	if err := watcher.Add(dir); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to add directory %s to watcher: %w", dir, err)
	}

	// Channel for sending configuration updates
	f.changeChan = make(chan bool)
	f.watcher = watcher
	f.isWatching = true

	// Start goroutine to monitor file changes
	go func() {
		filename := filepath.Base(absPath)
		var debounceTimer *time.Timer
		var mu sync.Mutex

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Check if it's the file we're interested in
				if filepath.Base(event.Name) != filename {
					continue
				}

				// Check if it's a write or create event
				if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
					continue
				}

				// Use debounce mechanism to avoid frequent reloads
				mu.Lock()
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(100*time.Millisecond, func() {
					// Send notification
					f.mutex.Lock()
					if f.changeChan != nil {
						f.changeChan <- true
					}
					f.mutex.Unlock()
				})
				mu.Unlock()

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Printf("File watcher error: %v\n", err)
			}
		}
	}()

	return f.changeChan, nil
}

// Stop implements the Source interface, stopping the file watcher
func (f *FileSource) Stop() error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if !f.isWatching {
		return nil
	}

	if f.watcher != nil {
		err := f.watcher.Close()
		f.watcher = nil

		if f.changeChan != nil {
			close(f.changeChan)
			f.changeChan = nil
		}

		f.isWatching = false
		return err
	}

	return nil
}

// String implements the Source interface, returning a description of the source
func (f *FileSource) String() string {
	return fmt.Sprintf("FileSource(%s)", f.Path)
}
