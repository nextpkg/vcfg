package plugins

import (
	"fmt"
	"log/slog"
	"maps"
	"reflect"
	"strings"
	"sync"
)

type (
	// PluginFactory defines a factory function for creating plugin instances
	PluginFactory func() Plugin

	// ConfigFactory defines a factory function for creating config instances
	ConfigFactory func() Config

	// RegisterOptions defines options for plugin registration
	RegisterOptions struct {
		InstanceName string // Optional: explicit instance name
		ConfigPath   string // Optional: explicit config path
		AutoDiscover bool   // Whether to enable auto-discovery for this type
	}

	// PluginTypeEntry represents a registered plugin type with its factories
	PluginTypeEntry struct {
		PluginFactory PluginFactory
		ConfigFactory ConfigFactory
		PluginType    string
		AutoDiscover  bool
	}

	// globalPluginRegistry 全局插件注册表
	globalPluginRegistry struct {
		mu          sync.RWMutex
		plugins     map[string]*PluginEntry     // key: pluginType:instanceName
		pluginTypes map[string]*PluginTypeEntry // key: pluginType
	}

	// PluginEntry 插件注册表条目
	PluginEntry struct {
		Plugin       Plugin
		Config       Config
		PluginType   string // 插件类型名称
		InstanceName string // 实例名称
		ConfigPath   string // 配置路径
		started      bool   // 插件启动状态
	}
)

var (
	globalRegistry     *globalPluginRegistry
	globalRegistryOnce sync.Once
)

// makePluginKey generates a composite key for plugin registration
// Format: pluginType:instanceName
func makePluginKey(pluginType, instanceName string) string {
	if instanceName == "" {
		return pluginType
	}
	return pluginType + ":" + instanceName
}

// getGlobalPluginRegistry 返回单例全局插件注册表实例
func getGlobalPluginRegistry() *globalPluginRegistry {
	globalRegistryOnce.Do(func() {
		globalRegistry = &globalPluginRegistry{
			plugins:     make(map[string]*PluginEntry),
			pluginTypes: make(map[string]*PluginTypeEntry),
		}
	})
	return globalRegistry
}

// RegisterPluginType registers a plugin type with factory functions for auto-discovery
// This is the recommended way to register plugin types for automatic instantiation
// Generic version with compile-time type safety
func RegisterPluginType[P Plugin, C Config](opts ...RegisterOptions) {
	registry := getGlobalPluginRegistry()
	registry.mu.Lock()
	defer registry.mu.Unlock()

	// Create type-safe factory functions using generics
	pluginFactory := func() Plugin {
		var p P
		pluginType := reflect.TypeOf(p)
		if pluginType.Kind() == reflect.Ptr {
			return reflect.New(pluginType.Elem()).Interface().(Plugin)
		}
		return reflect.New(pluginType).Interface().(Plugin)
	}

	configFactory := func() Config {
		var c C
		configType := reflect.TypeOf(c)
		if configType.Kind() == reflect.Ptr {
			return reflect.New(configType.Elem()).Interface().(Config)
		}
		return reflect.New(configType).Interface().(Config)
	}

	// Create sample instances to get plugin type
	plugin := pluginFactory()
	config := configFactory()

	// Derive plugin type name from reflection if not set
	pluginTypeName := getPluginTypeName(plugin, config)

	// Auto-set names for BasePlugin and BaseConfig if they are embedded
	setBaseNames(plugin, config, pluginTypeName)

	// Determine auto-discovery setting
	autoDiscover := true
	if len(opts) > 0 {
		autoDiscover = opts[0].AutoDiscover
	}

	entry := &PluginTypeEntry{
		PluginFactory: pluginFactory,
		ConfigFactory: configFactory,
		PluginType:    pluginTypeName,
		AutoDiscover:  autoDiscover,
	}

	registry.pluginTypes[pluginTypeName] = entry
	slog.Info("Plugin type registered", "type", pluginTypeName, "auto_discover", autoDiscover)
}

// Register registers a plugin instance with optional configuration
// This provides a simplified API for manual plugin registration
func Register(plugin Plugin, config Config, opts ...RegisterOptions) {
	registry := getGlobalPluginRegistry()
	registry.mu.Lock()
	defer registry.mu.Unlock()

	pluginType := plugin.Name()
	configType := GetConfigTypeName(config)

	if pluginType != configType {
		panic(fmt.Sprintf("plugin type mismatch: %s != %s", pluginType, configType))
	}

	// Determine instance name and config path
	var instanceName, configPath string
	if len(opts) > 0 {
		instanceName = opts[0].InstanceName
		configPath = opts[0].ConfigPath
	}

	// Add debug logging to see what's happening
	slog.Debug("Register called", "pluginType", pluginType, "configType", configType, "instanceName", instanceName, "configPath", configPath)

	if instanceName == "" {
		instanceName = GetConfigTypeName(config)
		slog.Debug("Using fallback instanceName", "instanceName", instanceName)
	}

	pluginKey := makePluginKey(pluginType, instanceName)

	if _, exists := registry.plugins[pluginKey]; exists {
		slog.Warn("Plugin instance already registered", "type", pluginType, "instance", instanceName, "key", pluginKey)
		return
	}

	// Instance name is now directly derived from config type name
	entry := &PluginEntry{
		Plugin:       plugin,
		Config:       config,
		PluginType:   pluginType,
		InstanceName: instanceName,
		ConfigPath:   configPath,
		started:      false,
	}

	registry.plugins[pluginKey] = entry
	slog.Info("Plugin registered", "type", pluginType, "instance", instanceName, "key", pluginKey, "config_path", configPath)
}

// AutoRegisterPlugins automatically discovers and registers plugin instances based on configuration structure
// This function uses reflection to traverse the config structure and automatically instantiate plugins
// for any field that matches a registered plugin type
func AutoRegisterPlugins(config any) error {
	registry := getGlobalPluginRegistry()
	registry.mu.RLock()
	pluginTypes := make(map[string]*PluginTypeEntry)
	for k, v := range registry.pluginTypes {
		if v.AutoDiscover {
			pluginTypes[k] = v
		}
	}
	registry.mu.RUnlock()

	if len(pluginTypes) == 0 {
		slog.Debug("No plugin types registered for auto-discovery")
		return nil
	}

	// Start recursive discovery - pass pointer to config so fields can be addressed
	configValue := reflect.ValueOf(config)
	if configValue.Kind() != reflect.Ptr {
		// If config is not a pointer, we need to get its address
		// But since we can't get address of interface{}, we need to handle this differently
		slog.Debug("Config is not a pointer, trying to work with value", "type", configValue.Type().String())
		return discoverAndRegisterPlugins(configValue, pluginTypes, "")
	}
	return discoverAndRegisterPlugins(configValue, pluginTypes, "")
}

// discoverAndRegisterPlugins recursively discovers plugin configurations in the config structure
func discoverAndRegisterPlugins(configValue reflect.Value, pluginTypes map[string]*PluginTypeEntry, currentPath string) error {
	// Handle pointers
	if configValue.Kind() == reflect.Ptr {
		configValue = configValue.Elem()
	}

	if !configValue.IsValid() || configValue.Kind() != reflect.Struct {
		return nil
	}

	configType := configValue.Type()

	for i := 0; i < configValue.NumField(); i++ {
		fieldType := configType.Field(i)
		fieldValue := configValue.Field(i)

		// Skip unexported fields
		if !fieldValue.CanInterface() {
			continue
		}

		// Build current field path
		fieldPath := fieldType.Name
		if currentPath != "" {
			fieldPath = currentPath + "." + fieldType.Name
		}

		// Field path for logging purposes
		_ = fieldPath

		// Check if this field implements Config interface
		if fieldValue.Kind() == reflect.Struct && fieldValue.CanAddr() {
			fieldInterface := fieldValue.Addr().Interface()
			if config, ok := fieldInterface.(Config); ok {
				pluginType := GetConfigTypeName(config)
				slog.Debug("Found config field", "path", fieldPath, "type", pluginType)

				// Check if we have a registered plugin type for this config
				if pluginTypeEntry, exists := pluginTypes[pluginType]; exists {
					// Create plugin and config instances
					plugin := pluginTypeEntry.PluginFactory()
					newConfig := pluginTypeEntry.ConfigFactory()

					// Use field path as instance name to support multiple instances
					// This allows the same plugin type to have different instances based on config location
					instanceName := strings.ToLower(fieldPath)

					// Auto-set names for BasePlugin and BaseConfig if they are embedded
					setBaseNames(plugin, newConfig, pluginType)

					// Copy configuration values
					if err := copyConfigValues(fieldInterface, newConfig); err != nil {
						slog.Error("Failed to copy config values", "path", fieldPath, "type", pluginType, "error", err)
						continue
					}

					// Register the plugin instance
					Register(plugin, newConfig, RegisterOptions{
						InstanceName: instanceName,
						ConfigPath:   fieldPath,
					})

					slog.Info("Auto-registered plugin", "type", pluginType, "instance", instanceName, "path", fieldPath)
				}
			}
		}

		// Recursively process nested structures
		if fieldValue.Kind() == reflect.Struct {
			if err := discoverAndRegisterPlugins(fieldValue, pluginTypes, fieldPath); err != nil {
				return err
			}
		} else if fieldValue.Kind() == reflect.Ptr && !fieldValue.IsNil() {
			if err := discoverAndRegisterPlugins(fieldValue, pluginTypes, fieldPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyConfigValues copies configuration values from source to destination using reflection
func copyConfigValues(src, dst interface{}) error {
	srcValue := reflect.ValueOf(src)
	dstValue := reflect.ValueOf(dst)

	// Handle pointers
	if srcValue.Kind() == reflect.Ptr {
		srcValue = srcValue.Elem()
	}
	if dstValue.Kind() == reflect.Ptr {
		dstValue = dstValue.Elem()
	}

	if !dstValue.CanSet() {
		return fmt.Errorf("destination is not settable")
	}

	if srcValue.Type() != dstValue.Type() {
		return fmt.Errorf("type mismatch: %v != %v", srcValue.Type(), dstValue.Type())
	}

	dstValue.Set(srcValue)
	return nil
}

// UnregisterGlobalPlugin 注销全局插件
// 支持 pluginType 或 pluginType:instanceName 格式
func UnregisterGlobalPlugin(key string) {
	registry := getGlobalPluginRegistry()
	registry.mu.Lock()
	defer registry.mu.Unlock()

	if _, exists := registry.plugins[key]; !exists {
		slog.Warn("Global plugin not found", "key", key)
		return
	}

	delete(registry.plugins, key)
	slog.Info("Plugin unregistered", "key", key)
}

// ListGlobalPlugins 列出所有全局插件
func ListGlobalPlugins() []*PluginEntry {
	registry := getGlobalPluginRegistry()
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	entries := make([]*PluginEntry, 0, len(registry.plugins))
	for _, entry := range registry.plugins {
		entries = append(entries, entry)
	}

	return entries
}

// Clone 获取所有注册的插件
func Clone() map[string]*PluginEntry {
	registry := getGlobalPluginRegistry()
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	// 返回副本以避免并发修改
	return maps.Clone(registry.plugins)
}

// GetGlobalPlugin 获取指定的全局插件
func GetGlobalPlugin(pluginType, instanceName string) (*PluginEntry, bool) {
	registry := getGlobalPluginRegistry()
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	key := makePluginKey(pluginType, instanceName)
	entry, exists := registry.plugins[key]
	return entry, exists
}

// GetPluginTypes returns all registered plugin types
func GetPluginTypes() map[string]*PluginTypeEntry {
	registry := getGlobalPluginRegistry()
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	return maps.Clone(registry.pluginTypes)
}

// ListPluginTypes returns a list of all registered plugin type names
func ListPluginTypes() []string {
	registry := getGlobalPluginRegistry()
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	types := make([]string, 0, len(registry.pluginTypes))
	for pluginType := range registry.pluginTypes {
		types = append(types, pluginType)
	}
	return types
}

// GetPluginType returns a specific plugin type entry
func GetPluginType(pluginType string) (*PluginTypeEntry, bool) {
	registry := getGlobalPluginRegistry()
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	entry, exists := registry.pluginTypes[pluginType]
	return entry, exists
}

// UnregisterPluginType removes a plugin type from the registry
func UnregisterPluginType(pluginType string) {
	registry := getGlobalPluginRegistry()
	registry.mu.Lock()
	defer registry.mu.Unlock()

	delete(registry.pluginTypes, pluginType)
	slog.Info("Plugin type unregistered", "type", pluginType)
}

// ClearGlobalRegistry 清空全局注册表
func ClearGlobalRegistry() {
	registry := getGlobalPluginRegistry()
	registry.mu.Lock()
	defer registry.mu.Unlock()

	clear(registry.plugins)
	clear(registry.pluginTypes)
}

// getPluginTypeName derives the plugin type name from plugin and config instances
// It prioritizes plugin.Name() if available, otherwise falls back to reflection
func getPluginTypeName(plugin Plugin, config Config) string {
	// Try to get name from plugin first
	if name := plugin.Name(); name != "" {
		return name
	}

	// Fall back to config type name
	return GetConfigTypeName(config)
}

// setBaseNames automatically sets names for embedded BasePlugin and BaseConfig
// This reduces boilerplate for users who embed these base types
func setBaseNames(plugin Plugin, config Config, typeName string) {
	// Set BasePlugin name if embedded
	if basePlugin := getEmbeddedBasePlugin(plugin); basePlugin != nil {
		if basePlugin.name == "" {
			basePlugin.SetName(typeName)
		}
	}

	// Set BaseConfig name if embedded
	if baseConfig := getEmbeddedBaseConfig(config); baseConfig != nil {
		if baseConfig.name == "" {
			baseConfig.SetName(typeName)
		}
	}
}

// getEmbeddedBasePlugin tries to find and return embedded BasePlugin using reflection
func getEmbeddedBasePlugin(plugin Plugin) *BasePlugin {
	v := reflect.ValueOf(plugin)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	// Look for embedded BasePlugin field
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := v.Type().Field(i)

		// Check if this field is an embedded BasePlugin
		if fieldType.Anonymous && fieldType.Type == reflect.TypeOf(BasePlugin{}) {
			if field.CanAddr() {
				return field.Addr().Interface().(*BasePlugin)
			}
		}

		// Check if this field is a pointer to BasePlugin
		if fieldType.Anonymous && fieldType.Type == reflect.TypeOf(&BasePlugin{}) {
			if !field.IsNil() {
				return field.Interface().(*BasePlugin)
			}
		}
	}

	return nil
}

// getEmbeddedBaseConfig tries to find and return embedded BaseConfig using reflection
func getEmbeddedBaseConfig(config Config) *BaseConfig {
	v := reflect.ValueOf(config)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	// Look for embedded BaseConfig field
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := v.Type().Field(i)

		// Check if this field is an embedded BaseConfig
		if fieldType.Anonymous && fieldType.Type == reflect.TypeOf(BaseConfig{}) {
			if field.CanAddr() {
				return field.Addr().Interface().(*BaseConfig)
			}
		}

		// Check if this field is a pointer to BaseConfig
		if fieldType.Anonymous && fieldType.Type == reflect.TypeOf(&BaseConfig{}) {
			if !field.IsNil() {
				return field.Interface().(*BaseConfig)
			}
		}
	}

	return nil
}

// StartAllPlugins starts all registered plugins in the global registry
func StartAllPlugins() error {
	registry := getGlobalPluginRegistry()
	registry.mu.Lock()
	defer registry.mu.Unlock()

	for pluginKey, entry := range registry.plugins {
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

// StopAllPlugins stops all registered plugins in the global registry
func StopAllPlugins() error {
	registry := getGlobalPluginRegistry()
	registry.mu.Lock()
	defer registry.mu.Unlock()

	for pluginKey, entry := range registry.plugins {
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

// ListAllPlugins returns information about all registered plugins in the global registry
func ListAllPlugins() map[string]*PluginEntry {
	registry := getGlobalPluginRegistry()
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	return maps.Clone(registry.plugins)
}
