package watcher

import (
	"fmt"
	"log/slog"

	"github.com/nextpkg/vcfg/source"
	"go.uber.org/atomic"
)

type (
	// Event defines the callback function type for configuration changes.
	Event func(any) error

	// Watcher is used to monitor configuration changes from multiple sources.
	Watcher struct {
		// isWatching controls whether the watcher is currently monitoring changes.
		isWatching atomic.Bool
		// stopChan is used to stop the monitoring process.
		stopChan chan struct{}
		// onChange stores the callback functions to be executed when a change is detected.
		onChange []Event
	}
)

// New creates a new Watcher instance.
func New() *Watcher {
	return &Watcher{
		stopChan: make(chan struct{}),
		onChange: make([]Event, 0),
	}
}

// StopWatch stops monitoring configuration changes from all sources.
func (w *Watcher) StopWatch() {
	ok := w.isWatching.CompareAndSwap(true, false)
	if !ok {
		return
	}

	close(w.stopChan)

	// Create a new stop channel to allow restarting the watcher.
	w.stopChan = make(chan struct{})
}

// OnChange registers a callback function to be executed when a configuration change is detected.
func (w *Watcher) OnChange(callback Event) {
	w.onChange = append(w.onChange, callback)
}

// Watch starts monitoring configuration changes from the specified sources.
// It takes a list of sources and a callback function to handle the change events.
func (w *Watcher) Watch(sources []source.Source, callback func([]Event) error) error {
	ok := w.isWatching.CompareAndSwap(false, true)
	if ok {
		return nil
	}

	hasError := atomic.NewError(nil)
	for _, src := range sources {
		// Start a goroutine to monitor changes for each source.
		go func(src source.Source) {
			ch, err := src.Watch()
			if err != nil {
				hasError.Store()
				hasError.Store(true)
				fmt.Errorf("failed to watch source %s: %v", src.String(), err)
				return
			}

			for {
				select {
				case <-ch:
					slog.Info("configuration change detected in source %s, reloading...", src.String())

					// Execute the callback function to handle the change event.
					err = callback(w.onChange)
					if err != nil {
						slog.Info("failed to reload source %s: %v", src.String(), err)
						continue
					}

					slog.Info("source %s reloaded successfully", src.String())
				case <-w.stopChan:
					err = src.Stop()
					if err != nil {
						errChan <- fmt.Errorf("failed to stop watching source %s: %v", src.String(), err)
						return
					}

					slog.Info("successfully stopped watching source %s", src.String())
				}
			}
		}(src)
	}

	// Wait for errors from the error channel.
	for err := range errChan {
		slog.Warn("error occurred while watching configuration changes: %v", err)
		w.StopWatch()
	}
}
