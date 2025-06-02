package plugins

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"sync"
)

type PluginManager[T any] struct {
	mu      sync.RWMutex
	plugins map[string]*PluginEntry
}

func NewPluginManager[T any]() *PluginManager[T] {
	return &PluginManager[T]{
		plugins: make(map[string]*PluginEntry),
	}
}

// Register registers a plugin with the manager
// Supports multi-instance plugins by using pluginType:instanceName as the key
func (pm *PluginManager[T]) Register(plugin Plugin, config Config) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pluginType := plugin.Name()
	configType := GetConfigTypeName(config)

	if pluginType != configType {
		return fmt.Errorf("plugin type mismatch: %s != %s", pluginType, configType)
	}

	// Get or generate instance name
	instanceName := GetConfigTypeName(config)
	pluginKey := makePluginKey(pluginType, instanceName)

	// For manual registration, instance name is derived from config type name

	if _, exists := pm.plugins[pluginKey]; exists {
		return fmt.Errorf("plugin instance %s already registered", pluginKey)
	}

	entry := &PluginEntry{
		Plugin:       plugin,
		Config:       config,
		PluginType:   pluginType,
		InstanceName: instanceName,
		started:      false,
	}

	pm.plugins[pluginKey] = entry
	slog.Info("Plugin registered",
		"plugin_type", pluginType,
		"instance", instanceName,
		"key", pluginKey)
	return nil
}

// RegisterWithInstance registers a plugin with a specific instance name
func (pm *PluginManager[T]) RegisterWithInstance(plugin Plugin, config Config, instanceName string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pluginType := plugin.Name()
	configType := GetConfigTypeName(config)

	if pluginType != configType {
		return fmt.Errorf("plugin type mismatch: %s != %s", pluginType, configType)
	}

	pluginKey := makePluginKey(pluginType, instanceName)

	if _, exists := pm.plugins[pluginKey]; exists {
		return fmt.Errorf("plugin instance %s already registered", pluginKey)
	}

	entry := &PluginEntry{
		Plugin:       plugin,
		Config:       config,
		PluginType:   pluginType,
		InstanceName: instanceName,
		started:      false,
	}

	pm.plugins[pluginKey] = entry
	slog.Info("Plugin registered with instance",
		"plugin_type", pluginType,
		"instance", instanceName,
		"key", pluginKey)
	return nil
}

// StartAll starts all registered plugins
func (pm *PluginManager[T]) StartAll(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for pluginKey, entry := range pm.plugins {
		if entry.started {
			continue
		}

		if err := entry.Plugin.Start(entry.Config); err != nil {
			return fmt.Errorf("failed to start plugin %s: %w", pluginKey, err)
		}

		entry.started = true
		slog.Info("Plugin started",
			"plugin_type", entry.PluginType,
			"instance", entry.InstanceName,
			"key", pluginKey)
	}

	return nil
}

// StopAll stops all registered plugins
func (pm *PluginManager[T]) StopAll(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for pluginKey, entry := range pm.plugins {
		if !entry.started {
			continue
		}

		if err := entry.Plugin.Stop(); err != nil {
			return fmt.Errorf("failed to stop plugin %s: %w", pluginKey, err)
		}

		entry.started = false
		slog.Info("Plugin stopped",
			"plugin_type", entry.PluginType,
			"instance", entry.InstanceName,
			"key", pluginKey)
	}

	return nil
}

// HandleSmartConfigChange intelligently handles configuration changes by automatically
// detecting which plugin configurations have changed and reloading only those plugins.
// This method uses reflection to recursively iterate through configuration struct fields
// and automatically reloads plugins when their corresponding configuration implements
// the Config interface and has changed.
func (pm *PluginManager[T]) HandleSmartConfigChange(ctx context.Context, oldConfig, newConfig *T) {
	if oldConfig == nil || newConfig == nil {
		return
	}

	// Get global plugin registry for plugin lookup
	globalPlugins := Clone()

	// Use reflection to recursively iterate through configuration fields
	oldValue := reflect.ValueOf(oldConfig)
	newValue := reflect.ValueOf(newConfig)

	// Start recursive traversal
	pm.handleConfigChangeRecursive(ctx, globalPlugins, oldValue, newValue, "")
}

// handleConfigChangeRecursive recursively traverses configuration structures to detect
// plugin configuration changes at any nesting level with multi-instance support
func (pm *PluginManager[T]) handleConfigChangeRecursive(ctx context.Context, globalPlugins map[string]*PluginEntry, oldValue, newValue reflect.Value, fieldPath string) {
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

		// Build field path for logging
		currentFieldPath := fieldType.Name
		if fieldPath != "" {
			currentFieldPath = fieldPath + "." + fieldType.Name
		}

		// Check if the field implements Config interface
		if oldFieldValue.Kind() == reflect.Struct {
			// Try to get the interface from the field
			if oldFieldValue.CanAddr() {
				oldConfigInterface := oldFieldValue.Addr().Interface()
				newConfigInterface := newFieldValue.Addr().Interface()

				// Check if it implements Config interface
				if config, ok := oldConfigInterface.(Config); ok {
					if !reflect.DeepEqual(oldConfigInterface, newConfigInterface) {
						pluginType := GetConfigTypeName(config)
						// Use field path as instance name for consistency with auto-discovery
						instanceName := strings.ToLower(currentFieldPath)
						pluginKey := makePluginKey(pluginType, instanceName)

						slog.Info("Smart config change detected",
							"field", currentFieldPath,
							"plugin_type", pluginType,
							"instance", instanceName,
							"key", pluginKey)

						// Try to reload from registered plugins first
						pm.mu.RLock()
						if entry, exists := pm.plugins[pluginKey]; !exists || !entry.started {
							pm.mu.RUnlock()
							// Try to reload from global registry
							if entry, exists = globalPlugins[pluginKey]; exists {
								go func(key string, plugin Plugin, newCfg any) {
									if err := plugin.Reload(newCfg); err != nil {
										slog.Error("Smart global plugin reload failed", "key", key, "error", err)
										return
									}
									slog.Info("Smart global plugin reloaded", "key", key)
								}(pluginKey, entry.Plugin, newConfigInterface)
							}
						} else {
							pm.mu.RUnlock()
							go func(key string, e *PluginEntry, newCfg any) {
								if err := e.Plugin.Reload(newCfg); err != nil {
									slog.Error("Smart plugin reload failed", "key", key, "error", err)
									return
								}
								// Update last config
								if newConfig, ok := newCfg.(Config); ok {
									e.Config = newConfig
								}
								slog.Info("Smart plugin reloaded", "key", key)
							}(pluginKey, entry, newConfigInterface)
						}
					} else {
						// If not a plugin config, recursively check nested structures
						pm.handleConfigChangeRecursive(ctx, globalPlugins, oldFieldValue, newFieldValue, currentFieldPath)
					}
				}
			} else {
				// Try direct interface check for non-addressable fields
				oldConfigInterface := oldFieldValue.Interface()
				newConfigInterface := newFieldValue.Interface()

				if config, ok := oldConfigInterface.(Config); ok {
					if !reflect.DeepEqual(oldConfigInterface, newConfigInterface) {
						pluginType := GetConfigTypeName(config)
						// Use field path as instance name for consistency with auto-discovery
						instanceName := strings.ToLower(currentFieldPath)
						pluginKey := makePluginKey(pluginType, instanceName)

						slog.Info("Smart config change detected",
							"field", currentFieldPath,
							"plugin_type", pluginType,
							"instance", instanceName,
							"key", pluginKey)

						// Try to reload from registered plugins first
						pm.mu.RLock()
						if entry, exists := pm.plugins[pluginKey]; exists && entry.started {
							pm.mu.RUnlock()
							go func(key string, e *PluginEntry, newCfg interface{}) {
								if err := e.Plugin.Reload(newCfg); err != nil {
									slog.Error("Smart plugin reload failed", "key", key, "error", err)
									return
								}
								// Update last config
								if newConfig, ok := newCfg.(Config); ok {
									e.Config = newConfig
								}
								slog.Info("Smart plugin reloaded", "key", key)
							}(pluginKey, entry, newConfigInterface)
						} else {
							pm.mu.RUnlock()
							// Try to reload from global registry
							if entry, exists := globalPlugins[pluginKey]; exists {
								go func(key string, plugin Plugin, newCfg interface{}) {
									if err := plugin.Reload(newCfg); err != nil {
										slog.Error("Smart global plugin reload failed", "key", key, "error", err)
										return
									}
									slog.Info("Smart global plugin reloaded", "key", key)
								}(pluginKey, entry.Plugin, newConfigInterface)
							}
						}
					} else {
						// If not a plugin config, recursively check nested structures
						pm.handleConfigChangeRecursive(ctx, globalPlugins, oldFieldValue, newFieldValue, currentFieldPath)
					}
				}
			}
		}
	}
}

// Get returns a plugin by key (pluginType:instanceName)
func (pm *PluginManager[T]) Get(key string) (Plugin, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if entry, exists := pm.plugins[key]; exists {
		return entry.Plugin, true
	}
	return nil, false
}

// GetByTypeAndInstance returns a plugin by type and instance name
func (pm *PluginManager[T]) GetByTypeAndInstance(pluginType, instanceName string) (Plugin, bool) {
	key := makePluginKey(pluginType, instanceName)
	return pm.Get(key)
}

// List returns all plugin keys (pluginType:instanceName)
func (pm *PluginManager[T]) List() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	keys := make([]string, 0, len(pm.plugins))
	for key := range pm.plugins {
		keys = append(keys, key)
	}
	return keys
}

// ListByType returns all plugin instances of a specific type
func (pm *PluginManager[T]) ListByType(pluginType string) []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	instances := make([]string, 0)
	for key, entry := range pm.plugins {
		if entry.PluginType == pluginType {
			instances = append(instances, key)
		}
	}
	return instances
}

// Unregister removes a plugin from the manager by key
func (pm *PluginManager[T]) Unregister(key string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	entry, exists := pm.plugins[key]
	if !exists {
		return fmt.Errorf("plugin %s not found", key)
	}

	// Stop the plugin if it's running
	if entry.started {
		if err := entry.Plugin.Stop(); err != nil {
			slog.Error("Failed to stop plugin during unregister", "key", key, "error", err)
		}
	}

	delete(pm.plugins, key)
	slog.Info("Plugin unregistered", "key", key)
	return nil
}

// UnregisterByTypeAndInstance removes a plugin by type and instance name
func (pm *PluginManager[T]) UnregisterByTypeAndInstance(pluginType, instanceName string) error {
	key := makePluginKey(pluginType, instanceName)
	return pm.Unregister(key)
}

// Shutdown stops all plugins and clears the manager
func (pm *PluginManager[T]) Shutdown(ctx context.Context) error {
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
