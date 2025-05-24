package source

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/nextpkg/vcfg/viper"
)

// FileSource represents a file-based configuration source that reads from a file.
// It implements the Source interface and provides methods for reading configuration,
// watching for changes, and stopping the watcher.
type FileSource struct {
	path  string
	viper *viper.Viper
	ch    chan struct{}
	once  sync.Once
}

// NewFileSource creates a new FileSource instance for the given file path.
// It automatically detects the file format based on the file extension.
//
// Parameters:
//   - path: string - The path to the configuration file
//
// Returns:
//   - *FileSource - A new FileSource instance configured for the specified file
func NewFileSource(path string) *FileSource {
	v := viper.New()
	v.SetConfigFile(path)

	// Auto-detect format
	format := filepath.Ext(path)
	if format != "" {
		format = format[1:] // Remove the dot
		v.SetConfigType(format)
	}

	return &FileSource{
		path:  path,
		viper: v,
		ch:    make(chan struct{}),
	}
}

// Read implements the Source interface by reading configuration from the file.
// It returns the parsed configuration as a Viper instance.
//
// Returns:
//   - *viper.Viper: The parsed configuration
//   - error: Any error that occurred during reading
func (f *FileSource) Read() (*viper.Viper, error) {
	err := f.viper.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", f.path, err)
	}

	return f.viper, nil
}

// Watch implements the Source interface by monitoring the configuration file for changes.
// It returns a channel that receives notifications (true) when the file changes.
// The watch is initialized only once per FileSource instance.
//
// Returns:
//   - <-chan bool: A channel that receives notifications on file changes
//   - error: Any error that occurred during watch setup
func (f *FileSource) Watch() (<-chan struct{}, error) {
	f.once.Do(func() {
		// Create a new stop channel to allow restarting the watcher.
		f.ch = make(chan struct{})
		f.viper.OnConfigChange(func(e fsnotify.Event) {
			f.ch <- struct{}{}
		})
		f.viper.WatchConfig()
	})

	return f.ch, nil
}

// String implements the Source interface by returning a description of the source.
// The description includes the file path being monitored.
//
// Returns:
//   - string: A descriptive string representation of the FileSource
func (f *FileSource) String() string {
	return fmt.Sprintf("FileSource(%s)", f.path)
}

// Stop implements the Source interface by stopping the file watcher and cleaning up resources.
// It closes the notification channel and resets the sync.Once instance.
//
// Returns:
//   - error: Always returns nil as no errors are expected during cleanup
func (f *FileSource) Stop() error {
	close(f.ch)
	f.once = sync.Once{}
	return nil
}
