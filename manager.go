// Package vcfg provides configuration management capabilities.
// This file implements the core ConfigManager that handles configuration loading,
// validation, watching, and plugin management with thread-safe operations.
package vcfg

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/knadh/koanf/v2"
	"go.uber.org/atomic"

	"github.com/nextpkg/vcfg/defaults"
	"github.com/nextpkg/vcfg/plugins"
	"github.com/nextpkg/vcfg/providers"
	"github.com/nextpkg/vcfg/validator"
)

type (
	// ConfigManager is the core configuration manager that handles loading, validation,
	// watching, and plugin management. It supports generic configuration types through
	// the type parameter T and provides thread-safe access to configuration values.
	//
	// The manager coordinates multiple configuration sources, automatically detects
	// file formats, validates configurations, and manages plugin lifecycles.
	ConfigManager[T any] struct {
		// providers holds the configuration sources and their associated parsers
		providers []providers.ProviderConfig
		// koanf is the underlying configuration library instance
		koanf *koanf.Koanf
		// once ensures one-time initialization operations
		once sync.Once
		// cfg stores the current configuration using atomic operations for thread safety
		cfg atomic.Value
		// mu protects concurrent access to manager state
		mu sync.RWMutex
		// watchers holds cleanup functions for active file watchers
		watchers []func()
		// pluginManager manages plugin discovery, initialization, and lifecycle
		pluginManager *plugins.PluginManager[T]
	}

	// Watcher interface defines the contract for providers that support
	// watching configuration changes and notifying about updates.
	Watcher interface {
		// Watch starts watching for configuration changes and calls the callback
		// when changes occur or errors happen
		Watch(cb func(event any, err error)) error
	}

	// Unwatcher interface defines the contract for providers that support
	// stopping the watch operation and cleaning up resources.
	Unwatcher interface {
		// Unwatch stops watching for configuration changes
		Unwatch()
	}
)

// newManager creates a new configuration manager with the provided sources.
// It accepts both file paths (strings) and koanf.Provider instances, automatically
// detecting the appropriate parsers for each source.
//
// Type parameter:
//   - T: The configuration struct type to manage
//
// Parameters:
//   - sources: Variable number of configuration sources (file paths or koanf.Provider instances)
//
// Returns a new ConfigManager instance ready for configuration loading.
// Panics if provider creation fails.
func newManager[T any](sources ...any) *ConfigManager[T] {
	factory := providers.NewProviderFactory()
	providerConfigs, err := factory.CreateProviders(sources...)
	if err != nil {
		panic(err)
	}

	return &ConfigManager[T]{
		providers:     providerConfigs,
		koanf:         koanf.New("."),
		watchers:      make([]func(), 0),
		pluginManager: plugins.NewPluginManager[T](),
	}
}

// load loads configuration from all sources, applies defaults, validates the result,
// and returns the configuration struct. This method is thread-safe through mutex locking.
//
// The loading process:
// 1. Loads and merges all configuration sources
// 2. Unmarshals into the target struct type
// 3. Applies default values
// 4. Validates the configuration
//
// Returns a pointer to the loaded and validated configuration, or an error if any step fails.
func (cm *ConfigManager[T]) load() (*T, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// load all sources
	err := cm.loadSource()
	if err != nil {
		return nil, err
	}

	return cm.loadConfig()
}

// loadSource loads all configuration providers and merges them into the koanf instance.
// Providers are loaded in order, with later providers overriding earlier ones.
// Each provider is loaded with its associated parser for proper data interpretation.
//
// Returns an error if reading from any provider or merging configurations fails.
func (cm *ConfigManager[T]) loadSource() error {
	for _, providerConfig := range cm.providers {
		if err := cm.koanf.Load(providerConfig.Provider, providerConfig.Parser); err != nil {
			return NewParseError(fmt.Sprintf("%T", providerConfig.Provider), "failed to load from provider", err)
		}
	}

	return nil
}

// loadConfig unmarshals the merged configuration from koanf into the target struct type,
// applies default values, and validates the result.
//
// The process includes:
// 1. Unmarshaling the configuration into struct T
// 2. Applying default values to unset fields
// 3. Running validation on the final configuration
//
// Returns a pointer to the processed configuration, or an error if any step fails.
func (cm *ConfigManager[T]) loadConfig() (*T, error) {
	if cm == nil || cm.koanf == nil {
		return nil, NewParseError("manager", "configuration manager not properly initialized", nil)
	}

	var cfg T

	// Set default values using struct tags
	err := defaults.SetDefaults(&cfg)
	if err != nil {
		return nil, NewParseError("defaults", "failed to set default values", err)
	}

	err = cm.koanf.Unmarshal("", &cfg)
	if err != nil {
		return nil, NewParseError("koanf", "failed to unmarshal configuration", err)
	}

	err = validator.Validate(&cfg)
	if err != nil {
		return nil, NewValidationError("validator", "configuration validation failed", err)
	}

	return &cfg, nil
}

// EnableWatch enables watching for configuration changes.
// It sets up file watchers for providers that implement the Watcher interface.
// When a configuration change is detected, it reloads the configuration and
// triggers plugin reloads for affected plugins.
// This method is thread-safe and can be called multiple times safely.
func (cm *ConfigManager[T]) EnableWatch() *ConfigManager[T] {
	cm.once.Do(func() {
		for _, providerConfig := range cm.providers {
			if watcher, ok := providerConfig.Provider.(Watcher); ok {
				err := watcher.Watch(func(event any, err error) {
					if err != nil {
						slog.Error("Watch error", "error", err)
						return
					}

					slog.Info("Configuration change detected", "event", event)

					// Get old configuration before reload
					oldConfig := cm.Get()

					// Reload configuration
					newConfig, loadErr := cm.load()
					if loadErr != nil {
						slog.Error("Failed to reload configuration", "error", loadErr)
						return
					}

					// Store new configuration
					cm.cfg.Store(newConfig)

					// Handle plugin configuration changes intelligently
					if oldConfig != nil {
						if err := cm.pluginManager.Reload(context.Background(), oldConfig, newConfig); err != nil {
							slog.Error("Failed to handle smart plugin reload", "error", err)
							return
						}
					}

					slog.Info("Configuration reloaded successfully")
				})

				if err != nil {
					slog.Error("Failed to enable watch", "error", err)
					continue
				}

				// Store cleanup function
				if unwatcher, ok := providerConfig.Provider.(Unwatcher); ok {
					cm.watchers = append(cm.watchers, unwatcher.Unwatch)
				} else {
					// For providers like koanf file provider that have Unwatch() error method
					if fileProvider, ok := providerConfig.Provider.(interface{ Unwatch() error }); ok {
						cm.watchers = append(cm.watchers, func() {
							if err := fileProvider.Unwatch(); err != nil {
								slog.Error("Failed to unwatch", "error", err)
							}
						})
					}
				}
			}
		}
	})

	return cm
}

// DisableWatch stops monitoring changes of all configuration providers.
func (cm *ConfigManager[T]) DisableWatch() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for _, cleanup := range cm.watchers {
		cleanup()
	}
	cm.watchers = cm.watchers[:0]
	cm.once = sync.Once{}
}

// Get returns the current configuration value.
// It retrieves the stored configuration from atomic.Value and returns it as type T.
// Returns nil if configuration is not initialized.
func (cm *ConfigManager[T]) Get() *T {
	if cm == nil {
		return nil
	}
	cfg := cm.cfg.Load()
	if cfg == nil {
		return nil
	}
	ret, ok := cfg.(*T)
	if !ok {
		return nil
	}
	return ret
}

// EnablePlugins automatically discovers and registers plugin instances based on current configuration
// This method uses the global plugin type registry to automatically instantiate and register plugins
// for any configuration field that matches a registered plugin type
func (cm *ConfigManager[T]) EnablePlugins() error {
	config := cm.Get()
	if config == nil {
		return fmt.Errorf("no configuration available for auto-registration")
	}

	// Use auto-registration
	return cm.pluginManager.DiscoverAndRegister(config)
}

// StartPlugins starts all registered plugins
// This method should be called after EnablePlugins to start the plugin instances
func (cm *ConfigManager[T]) StartPlugins(ctx context.Context) error {
	return cm.pluginManager.Startup(ctx)
}

// StopPlugins stops all running plugins
// This method gracefully stops all plugin instances
func (cm *ConfigManager[T]) StopPlugins(ctx context.Context) error {
	return cm.pluginManager.Shutdown(ctx)
}

// MustEnableAndStartPlugins enables and starts all plugins, panics on error
// This is a convenience method that combines EnablePlugins and StartPlugins
func (cm *ConfigManager[T]) MustEnableAndStartPlugins() {
	if err := cm.EnablePlugins(); err != nil {
		panic(err)
	}

	if err := cm.StartPlugins(context.Background()); err != nil {
		panic(err)
	}
}

// Close closes the configuration manager with default context, including all plugins and watchers
func (cm *ConfigManager[T]) Close() error {
	return cm.CloseWithContext(context.Background())
}

// CloseWithContext closes the configuration manager with context, including all plugins and watchers
func (cm *ConfigManager[T]) CloseWithContext(ctx context.Context) error {
	if cm == nil {
		return nil
	}

	// Stop all watchers
	cm.DisableWatch()

	// Shutdown all plugins
	return cm.pluginManager.Shutdown(ctx)
}
