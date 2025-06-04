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
	// ConfigManager is a configuration manager that handles loading, validation, and watching.
	// It supports generic configuration types through the type parameter T.
	// ConfigManager provides thread-safe access to configuration values.
	ConfigManager[T any] struct {
		providers     []providers.ProviderConfig
		koanf         *koanf.Koanf
		once          sync.Once
		cfg           atomic.Value
		mu            sync.RWMutex
		watchers      []func() // cleanup functions for watchers
		pluginManager *plugins.PluginManager[T]
		ctx           context.Context
	}

	// Watcher interface for providers that support watching configuration changes
	Watcher interface {
		Watch(cb func(event any, err error)) error
	}

	// Unwatcher interface for providers that support stopping watch
	Unwatcher interface {
		Unwatch()
	}
)

// newManager creates a new configuration manager with the provided sources.
// It accepts file paths and koanf.Provider instances.
// Parameters:
//   - sources: file paths (strings) or koanf.Provider instances
//
// Returns:
//   - A new ConfigManager instance.
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
		ctx:           context.Background(),
	}
}

// load loads configuration from sources, validates it, and returns the configuration struct.
// This method is thread-safe through mutex locking.
// Returns:
//   - A pointer to the loaded and validated configuration.
//   - An error if loading or validation fails.
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

// loadSource loads all configuration providers and merges them into koanf.
// It reads from each provider and combines the configurations.
// Returns:
//   - An error if reading from any provider or merging configurations fails.
func (cm *ConfigManager[T]) loadSource() error {
	for _, providerConfig := range cm.providers {
		if err := cm.koanf.Load(providerConfig.Provider, providerConfig.Parser); err != nil {
			return NewParseError(fmt.Sprintf("%T", providerConfig.Provider), "failed to load from provider", err)
		}
	}

	return nil
}

// loadConfig unmarshals the configuration from koanf into a struct and validates it.
// Returns:
//   - A pointer to the unmarshaled and validated configuration.
//   - An error if unmarshaling or validation fails.
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

					slog.Debug("Configuration change detected", "event", event)

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
						if err := cm.pluginManager.HandleSmartConfigChange(oldConfig, newConfig); err != nil {
							slog.Error("Failed to handle smart plugin reload", "error", err)
							return
						}
					}

					slog.Debug("Configuration reloaded successfully")
				})

				if err != nil {
					slog.Error("Failed to enable watch", "error", err)
					continue
				}

				// Store cleanup function
				if unwatcher, ok := providerConfig.Provider.(Unwatcher); ok {
					cm.watchers = append(cm.watchers, unwatcher.Unwatch)
				}
			}
		}
	})

	return cm
}

// DisableWatch stops monitoring changes of all configuration providers.
func (cm *ConfigManager[T]) DisableWatch() {
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
	return cm.pluginManager.Initialize(config)
}

// StartPlugins starts all registered plugins
// This method should be called after EnablePlugins to start the plugin instances
func (cm *ConfigManager[T]) StartPlugins() error {
	return cm.pluginManager.Startup(cm.ctx)
}

// StopPlugins stops all running plugins
// This method gracefully stops all plugin instances
func (cm *ConfigManager[T]) StopPlugins() error {
	return cm.pluginManager.Shutdown(cm.ctx)
}

// MustEnableAndStartPluginWithContext enables and starts all plugins with context, panics on error
// This is a convenience method that combines EnablePlugins and StartPlugins with context support
func (cm *ConfigManager[T]) MustEnableAndStartPluginWithContext(ctx context.Context) {
	if err := cm.EnablePlugins(); err != nil {
		panic(err)
	}

	if err := cm.StartPlugins(); err != nil {
		panic(err)
	}
}

// CloseWithContext closes the configuration manager with context, including all plugins and watchers
func (cm *ConfigManager[T]) CloseWithContext(ctx context.Context) error {
	if cm == nil {
		return nil
	}

	// 停止所有监听器
	cm.DisableWatch()

	// 关闭所有插件
	return cm.pluginManager.Shutdown(cm.ctx)
}
