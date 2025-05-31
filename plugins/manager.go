package plugins

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"sync"
)

type PluginManager struct {
	mu      sync.RWMutex
	plugins map[string]*PluginEntry
}

func NewPluginManager() *PluginManager {
	return &PluginManager{
		plugins: make(map[string]*PluginEntry),
	}
}

// Register registers a plugin with the manager
func (pm *PluginManager) Register(plugin Plugin, config Config) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if plugin.Name() != config.Name() {
		return fmt.Errorf("plugin name mismatch: %s != %s", plugin.Name(), config.Name())
	}

	if _, exists := pm.plugins[plugin.Name()]; exists {
		return fmt.Errorf("plugin %s already registered", plugin.Name())
	}

	entry := &PluginEntry{
		Plugin:  plugin,
		Config:  config,
		started: false,
	}

	pm.plugins[plugin.Name()] = entry
	slog.Info("Plugin registered", "name", plugin.Name())
	return nil
}

// StartAll starts all registered plugins
func (pm *PluginManager) StartAll(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for name, entry := range pm.plugins {
		if entry.started {
			continue
		}

		if err := entry.Plugin.Start(entry.Config); err != nil {
			slog.Error("Failed to start plugin", "name", name, "error", err)
			return fmt.Errorf("failed to start plugin %s: %w", name, err)
		}

		entry.started = true
		slog.Info("Plugin started", "name", name)
	}

	return nil
}

// StopAll stops all registered plugins
func (pm *PluginManager) StopAll(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for name, entry := range pm.plugins {
		if !entry.started {
			continue
		}

		if err := entry.Plugin.Stop(); err != nil {
			slog.Error("Failed to stop plugin", "name", name, "error", err)
			return fmt.Errorf("failed to stop plugin %s: %w", name, err)
		}

		entry.started = false
		slog.Info("Plugin stopped", "name", name)
	}

	return nil
}

// HandleConfigChange handles configuration changes and reloads affected plugins
func (pm *PluginManager) HandleConfigChange(ctx context.Context, newConfigs map[string]Config) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for name, entry := range pm.plugins {
		if !entry.started {
			continue
		}

		if newConfig, exists := newConfigs[name]; exists {
			// Check if configuration has changed
			if reflect.DeepEqual(entry.Config, newConfig) {
				continue // No change, skip reload
			}

			go func(pluginName string, e *PluginEntry, cfg Config) {
				// Reload the plugin with new configuration
				if err := e.Plugin.Reload(cfg); err != nil {
					slog.Error("Plugin reload failed", "plugin", pluginName, "error", err)
					return
				}

				e.Config = cfg
				slog.Info("Plugin reloaded", "plugin", pluginName)
			}(name, entry, newConfig)
		}
	}
}

// HandleSmartConfigChange intelligently handles configuration changes by automatically
// detecting which plugin configurations have changed and reloading only those plugins.
// This method uses reflection to iterate through configuration struct fields and
// automatically reloads plugins when their corresponding configuration implements
// the Config interface and has changed.
func (pm *PluginManager) HandleSmartConfigChange(ctx context.Context, oldConfig, newConfig any) {
	if oldConfig == nil || newConfig == nil {
		return
	}

	// Get global plugin registry for plugin lookup
	globalPlugins := Clone()

	// Use reflection to iterate through configuration fields
	oldValue := reflect.ValueOf(oldConfig)
	newValue := reflect.ValueOf(newConfig)

	// Handle pointers
	if oldValue.Kind() == reflect.Ptr {
		oldValue = oldValue.Elem()
	}
	if newValue.Kind() == reflect.Ptr {
		newValue = newValue.Elem()
	}

	if oldValue.Kind() != reflect.Struct || newValue.Kind() != reflect.Struct {
		return
	}

	oldType := oldValue.Type()

	for i := range oldValue.NumField() {
		fieldType := oldType.Field(i)
		oldFieldValue := oldValue.Field(i)
		newFieldValue := newValue.Field(i)

		// Skip unexported fields
		if !oldFieldValue.CanInterface() || !newFieldValue.CanInterface() {
			continue
		}

		// Check if the field implements Config interface
		if oldFieldValue.Kind() == reflect.Struct {
			// Try to get the interface from the field
			if oldFieldValue.CanAddr() {
				oldConfigInterface := oldFieldValue.Addr().Interface()
				newConfigInterface := newFieldValue.Addr().Interface()

				// Check if it implements Config interface
				if config, ok := oldConfigInterface.(Config); ok {
					// Check if configuration has changed
					if !reflect.DeepEqual(oldConfigInterface, newConfigInterface) {
						pluginName := config.Name()
						slog.Info("Smart config change detected", "field", fieldType.Name, "plugin", pluginName)

						// Try to reload from registered plugins first
						pm.mu.RLock()
						if entry, exists := pm.plugins[pluginName]; exists && entry.started {
							pm.mu.RUnlock()
							go func(name string, e *PluginEntry, newCfg interface{}) {
								if err := e.Plugin.Reload(newCfg); err != nil {
									slog.Error("Smart plugin reload failed", "plugin", name, "error", err)
									return
								}
								// Update last config
								if newConfig, ok := newCfg.(Config); ok {
									e.Config = newConfig
								}
								slog.Info("Smart plugin reloaded", "plugin", name)
							}(pluginName, entry, newConfigInterface)
						} else {
							pm.mu.RUnlock()
							// Try to reload from global registry
							if entry, exists := globalPlugins[pluginName]; exists {
								go func(name string, plugin Plugin, newCfg interface{}) {
									if err := plugin.Reload(newCfg); err != nil {
										slog.Error("Smart global plugin reload failed", "plugin", name, "error", err)
										return
									}
									slog.Info("Smart global plugin reloaded", "plugin", name)
								}(pluginName, entry.Plugin, newConfigInterface)
							}
						}
					}
				}
			} else {
				// Try direct interface check for non-addressable fields
				oldConfigInterface := oldFieldValue.Interface()
				newConfigInterface := newFieldValue.Interface()

				if config, ok := oldConfigInterface.(Config); ok {
					if !reflect.DeepEqual(oldConfigInterface, newConfigInterface) {
						pluginName := config.Name()
						slog.Info("Smart config change detected", "field", fieldType.Name, "plugin", pluginName)

						// Try to reload from registered plugins first
						pm.mu.RLock()
						if entry, exists := pm.plugins[pluginName]; exists && entry.started {
							pm.mu.RUnlock()
							go func(name string, e *PluginEntry, newCfg interface{}) {
								if err := e.Plugin.Reload(newCfg); err != nil {
									slog.Error("Smart plugin reload failed", "plugin", name, "error", err)
									return
								}
								// Update last config
								if newConfig, ok := newCfg.(Config); ok {
									e.Config = newConfig
								}
								slog.Info("Smart plugin reloaded", "plugin", name)
							}(pluginName, entry, newConfigInterface)
						} else {
							pm.mu.RUnlock()
							// Try to reload from global registry
							if entry, exists := globalPlugins[pluginName]; exists {
								go func(name string, plugin Plugin, newCfg interface{}) {
									if err := plugin.Reload(newCfg); err != nil {
										slog.Error("Smart global plugin reload failed", "plugin", name, "error", err)
										return
									}
									slog.Info("Smart global plugin reloaded", "plugin", name)
								}(pluginName, entry.Plugin, newConfigInterface)
							}
						}
					}
				}
			}
		}
	}
}

// Get returns a plugin by name
func (pm *PluginManager) Get(name string) (Plugin, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if entry, exists := pm.plugins[name]; exists {
		return entry.Plugin, true
	}
	return nil, false
}

// List returns all plugin names
func (pm *PluginManager) List() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	names := make([]string, 0, len(pm.plugins))
	for name := range pm.plugins {
		names = append(names, name)
	}
	return names
}

// Unregister removes a plugin from the manager
func (pm *PluginManager) Unregister(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	entry, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Stop the plugin if it's running
	if entry.started {
		if err := entry.Plugin.Stop(); err != nil {
			slog.Error("Failed to stop plugin during unregister", "name", name, "error", err)
		}
	}

	delete(pm.plugins, name)
	slog.Info("Plugin unregistered", "name", name)
	return nil
}

// Shutdown stops all plugins and clears the manager
func (pm *PluginManager) Shutdown(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for name, entry := range pm.plugins {
		if entry.started {
			if err := entry.Plugin.Stop(); err != nil {
				slog.Error("Failed to stop plugin during shutdown", "name", name, "error", err)
			}
		}
	}

	// Clear all plugins
	pm.plugins = make(map[string]*PluginEntry)
	slog.Info("Plugin manager shutdown completed")
	return nil
}
