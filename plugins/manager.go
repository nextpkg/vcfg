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
	plugins map[string]*PluginEntry // // key: pluginType:instanceName
}

func NewPluginManager[T any]() *PluginManager[T] {
	return &PluginManager[T]{
		plugins: make(map[string]*PluginEntry),
	}
}

// DiscoverAndRegister automatically discovers plugin configurations from the provided config struct
// and registers corresponding plugin instances. It uses reflection to traverse the config structure
// and creates plugin instances for fields that implement the Config interface.
func (pm *PluginManager[T]) DiscoverAndRegister(config *T) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pluginTypes := clonePluginTypes()
	if len(pluginTypes) == 0 {
		slog.Debug("No plugin types registered for auto-discovery")
		return nil
	}

	var discover func(reflect.Value, string) error
	discover = func(configValue reflect.Value, currentPath string) error {
		// Handle pointers
		if configValue.Kind() == reflect.Ptr {
			configValue = configValue.Elem()
		}

		if !configValue.IsValid() || configValue.Kind() != reflect.Struct {
			return fmt.Errorf("invalid config value")
		}

		configType := configValue.Type()
		for i := range configValue.NumField() {
			fieldType := configType.Field(i)
			fieldValue := configValue.Field(i)

			// Skip unexported fields
			if !fieldValue.CanInterface() {
				continue
			}

			// Build current field path
			fieldPath := getFieldPath(currentPath, fieldType.Name)

			// Check for pointer type configs and provide helpful error message
			if fieldValue.Kind() == reflect.Ptr {
				// Check if pointer points to a struct that implements Config interface
				if fieldValue.Type().Elem().Kind() == reflect.Struct {
					// Create a zero value instance to check if it implements Config interface
					zeroValue := reflect.New(fieldValue.Type().Elem()).Interface()
					if _, ok := zeroValue.(Config); ok {
						return fmt.Errorf("配置字段 '%s' 使用了指针类型 '%s'，请改为值类型 '%s'。指针类型配置可能导致意外的共享状态和内存问题",
							fieldPath, fieldValue.Type(), fieldValue.Type().Elem())
					}
				}
			}

			// Check if this field implements Config interface
			if fieldValue.Kind() == reflect.Struct && fieldValue.CanAddr() {
				fieldInterface := fieldValue.Addr().Interface()
				if oldConfig, ok := fieldInterface.(Config); ok {
					pluginType := getConfigType(oldConfig)

					slog.Debug("Found config field",
						"path", fieldPath,
						"type", pluginType,
						"raw_type", oldConfig.baseConfigEmbedded().Type,
					)

					// Check if we have a registered plugin type for this config
					entry, exists := pluginTypes[pluginType]
					if !exists {
						return fmt.Errorf("config field does not have a registered plugin type, type=%s", pluginType)
					}

					// Create newPlugin and config instances
					newPlugin := entry.PluginFactory()
					newConfig := entry.ConfigFactory()

					// Copy configuration values from oldConfig to newConfig
					if err := copyConfig(oldConfig, newConfig); err != nil {
						return fmt.Errorf("failed to copy config for %s: %w", fieldPath, err)
					}

					// Use field path as instance name to support multiple instances
					// This allows the same plugin type to have different instances based on config location
					instanceName := strings.ToLower(fieldPath)

					pluginKey := getPluginKey(pluginType, instanceName)

					// Check if plugin instance already exists
					if _, exists := pm.plugins[pluginKey]; exists {
						return fmt.Errorf("plugin instance %s already registered", pluginKey)
					}

					pm.plugins[pluginKey] = &PluginEntry{
						Plugin:       newPlugin,
						Config:       newConfig,
						PluginType:   pluginType,
						InstanceName: instanceName,
						ConfigPath:   fieldPath,
						started:      false,
					}

					slog.Debug("Plugin registered",
						"type", entry.PluginType,
						"instance", instanceName,
						"key", pluginKey,
						"config_path", fieldPath,
					)

					// Continue to process other fields instead of returning
					continue
				}
			}

			// Recursively process nested structures
			if (fieldValue.Kind() == reflect.Struct) || (fieldValue.Kind() == reflect.Ptr && !fieldValue.IsNil()) {
				if err := discover(fieldValue, fieldPath); err != nil {
					return err
				}
			}
		}
		return nil
	}

	return discover(reflect.ValueOf(config), "")
}

// Startup starts all registered plugins with context
func (pm *PluginManager[T]) Startup(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for pluginKey, entry := range pm.plugins {
		if entry.started {
			continue
		}

		if err := entry.Plugin.Startup(ctx, entry.Config); err != nil {
			return fmt.Errorf("failed to start plugin %s: %w", pluginKey, err)
		}

		entry.started = true
		slog.Info("Plugin started",
			"plugin_type", entry.PluginType,
			"instance", entry.InstanceName,
			"key", pluginKey,
		)
	}

	slog.Debug("All plugins started", "count", len(pm.plugins))

	return nil
}

// Shutdown stops all running plugins with context
func (pm *PluginManager[T]) Shutdown(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for pluginKey, entry := range pm.plugins {
		if !entry.started {
			continue
		}

		if err := entry.Plugin.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to stop plugin %s: %w", pluginKey, err)
		}

		entry.started = false
		slog.Info("Plugin stopped",
			"plugin_type", entry.PluginType,
			"instance", entry.InstanceName,
			"key", pluginKey,
		)
	}

	slog.Debug("All plugins stopped", "count", len(pm.plugins))

	return nil
}

// Reload intelligently handles configuration changes by automatically
// detecting which plugins need to be reloaded based on their configuration changes.
// This method uses reflection to recursively iterate through configuration struct fields
// and automatically reloads plugins when their corresponding configuration implements
// the Config interface and has changed.
func (pm *PluginManager[T]) Reload(ctx context.Context, oldConfig, newConfig *T) error {
	pm.mu.RLock()
	if len(pm.plugins) == 0 {
		pm.mu.RUnlock()
		return fmt.Errorf("no plugins registered")
	}
	pm.mu.RUnlock()

	if oldConfig == nil || newConfig == nil {
		return nil
	}

	// Use reflection to recursively iterate through configuration fields
	oldValue := reflect.ValueOf(oldConfig)
	newValue := reflect.ValueOf(newConfig)

	// Start recursive traversal
	return pm.handleConfigChangeRecursive(ctx, oldValue, newValue, "")
}

// handleConfigChangeRecursive recursively traverses configuration structures to detect
// plugin configuration changes at any nesting level with multi-instance support
func (pm *PluginManager[T]) handleConfigChangeRecursive(ctx context.Context, oldValue, newValue reflect.Value, fieldPath string) error {
	// Handle pointers
	if oldValue.Kind() == reflect.Ptr {
		oldValue = oldValue.Elem()
	}
	if newValue.Kind() == reflect.Ptr {
		newValue = newValue.Elem()
	}

	if oldValue.Kind() != reflect.Struct || newValue.Kind() != reflect.Struct {
		return nil
	}

	oldType := oldValue.Type()

	for i := range oldValue.NumField() {
		fieldType := oldType.Field(i)
		vOldField := oldValue.Field(i)
		vNewField := newValue.Field(i)

		// Skip unexported fields
		if !vOldField.CanInterface() || !vNewField.CanInterface() {
			continue
		}

		// Build field path for logging
		fieldPath = getFieldPath(fieldPath, fieldType.Name)

		// Check if the field implements Config interface
		if vOldField.Kind() == reflect.Struct {
			// Try to get config interface from the field
			iOldField := toInterface(vOldField)
			iNewField := toInterface(vNewField)

			if iOldField != nil {
				if config, ok := iOldField.(Config); ok && !reflect.DeepEqual(iOldField, iNewField) {
					return pm.reloadPluginConfig(ctx, config, iNewField, fieldPath)
				} else {
					// If not a plugin config, recursively check nested structures
					return pm.handleConfigChangeRecursive(ctx, vOldField, vNewField, fieldPath)
				}
			}
		}
	}

	return nil
}

// reloadPluginConfig handles the plugin reload logic
func (pm *PluginManager[T]) reloadPluginConfig(ctx context.Context, config Config, newConfig any, fieldPath string) error {
	pluginType := getConfigType(config)

	// Use field path as instance name for consistency with auto-discovery
	instanceName := strings.ToLower(fieldPath)
	pluginKey := getPluginKey(pluginType, instanceName)

	slog.Info("Smart config change detected",
		"field", fieldPath,
		"plugin_type", pluginType,
		"instance", instanceName,
		"key", pluginKey,
	)

	// Try to reload from registered plugins first
	pm.mu.RLock()
	entry, exists := pm.plugins[pluginKey]
	pm.mu.RUnlock()

	if exists && entry.started {
		// Reload registered plugin
		if err := entry.Plugin.Reload(ctx, newConfig); err != nil {
			return fmt.Errorf("smart plugin reload failed, key=%s, err=%w", pluginKey, err)
		}

		// Update config for registered plugins
		if newCfg, ok := newConfig.(Config); ok {
			entry.Config = newCfg
		}
	}

	return nil
}

// Clone returns information about all registered plugins in the global registry
func (pm *PluginManager[T]) Clone() map[string]*PluginEntry {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Create a deep copy of the plugins map
	cloned := make(map[string]*PluginEntry, len(pm.plugins))
	for key, entry := range pm.plugins {
		// Create a copy of the PluginEntry
		cloned[key] = &PluginEntry{
			Plugin:       entry.Plugin,
			Config:       entry.Config,
			PluginType:   entry.PluginType,
			InstanceName: entry.InstanceName,
			ConfigPath:   entry.ConfigPath,
			started:      entry.started,
		}
	}
	return cloned
}
