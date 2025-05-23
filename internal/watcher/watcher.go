package watcher

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/nextpkg/vcfg/source"
	"go.uber.org/atomic"
)

type (
	// Event defines the callback function type for configuration changes.
	Event[T any] func(T) error

	// Watcher is used to monitor configuration changes from multiple sources.
	Watcher[T any] struct {
		// isWatching controls whether the watcher is currently monitoring changes.
		isWatching atomic.Bool
		// stopChan is used to stop the monitoring process.
		stopChan chan struct{}
		// onChange stores the callback functions to be executed when a change is detected.
		onChange []Event[T]
	}
)

// New creates a new Watcher instance.
func New[T any]() *Watcher[T] {
	return &Watcher[T]{
		stopChan: make(chan struct{}),
		onChange: make([]Event[T], 0),
	}
}

// StopWatch stops monitoring configuration changes from all sources.
func (w *Watcher[T]) StopWatch() {
	ok := w.isWatching.CompareAndSwap(true, false)
	if !ok {
		return
	}

	close(w.stopChan)

	// Create a new stop channel to allow restarting the watcher.
	w.stopChan = make(chan struct{})
}

// OnChange registers a callback function to be executed when a configuration change is detected.
func (w *Watcher[T]) OnChange(callback Event[T]) {
	w.onChange = append(w.onChange, callback)
}

// Watch starts monitoring configuration changes from the specified sources.
// It takes a list of sources and a callback function to handle the change events.
func (w *Watcher[T]) Watch(sources []source.Source, callback func([]Event[T]) error) error {
	ok := w.isWatching.CompareAndSwap(false, true)
	if ok {
		return nil
	}

	wg := sync.WaitGroup{}
	hasError := atomic.NewError(nil)
	for _, src := range sources {
		wg.Add(1)

		// Start a goroutine to monitor changes for each source.
		go func(src source.Source) {
			ch, err := src.Watch()
			if err != nil {
				err = fmt.Errorf("failed to watch source %s: %v", src.String(), err)
				hasError.Store(err)
				wg.Done()
				return
			}
			wg.Done()

			for {
				select {
				case <-ch:
					slog.Info("configuration change detected in source, reloading...", "source", src.String())

					// Execute the callback function to handle the change event.
					err = callback(w.onChange)
					if err != nil {
						slog.Info("failed to reload source %s: %v", src.String(), err)
						continue
					}

					slog.Info("source reloaded successfully", "source", src.String())
				case <-w.stopChan:
					err = src.Stop()
					if err != nil {
						slog.Warn("failed to stop watching source", "source", src.String(), "err", err)
						return
					}

					slog.Info("successfully stopped watching source", "source", src.String())
				}
			}
		}(src)
	}

	wg.Wait()

	return hasError.Load()
}
