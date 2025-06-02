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

	// PluginPtr defines a constraint for plugin pointer types
	PluginPtr[T any] interface {
		Plugin
		*T
	}

	// ConfigPtr defines a constraint for config pointer types
	// Now requires ConfigWithBase to ensure compile-time type safety for BaseConfig embedding
	ConfigPtr[T any] interface {
		Config
		*T
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
// Generic version with compile-time type safety - requires pointer types
func RegisterPluginType[P PluginPtr[PT], C ConfigPtr[CT], PT any, CT any](opts ...RegisterOptions) {
	registry := getGlobalPluginRegistry()
	registry.mu.Lock()
	defer registry.mu.Unlock()

	// Create simplified type-safe factory functions using generics
	// Since P and C are constrained to be pointer types, we can simplify the logic
	pluginFactory := func() Plugin {
		var p P
		pluginType := reflect.TypeOf(p).Elem() // Get the underlying type
		return reflect.New(pluginType).Interface().(Plugin)
	}

	configFactory := func() Config {
		var c C
		configType := reflect.TypeOf(c).Elem() // Get the underlying type
		return reflect.New(configType).Interface().(Config)
	}

	// Create sample instances to get plugin type
	plugin := pluginFactory()
	config := configFactory()

	// Auto-set names for BaseConfig if embedded
	setBaseConfigName(config, plugin.Name())

	// Determine auto-discovery setting
	autoDiscover := true
	if len(opts) > 0 {
		autoDiscover = opts[0].AutoDiscover
	}

	entry := &PluginTypeEntry{
		PluginFactory: pluginFactory,
		ConfigFactory: configFactory,
		PluginType:    plugin.Name(),
		AutoDiscover:  autoDiscover,
	}

	registry.pluginTypes[plugin.Name()] = entry
	slog.Info("Plugin type registered", "type", plugin.Name(), "auto_discover", autoDiscover)
}

// Register registers a plugin instance with optional configuration
// This provides a simplified API for manual plugin registration
func Register(plugin Plugin, config Config, opts ...RegisterOptions) {
	registry := getGlobalPluginRegistry()
	registry.mu.Lock()
	defer registry.mu.Unlock()

	// Auto-set names for BaseConfig if embedded
	setBaseConfigName(config, plugin.Name())

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
	return discoverAndRegisterPlugins(reflect.ValueOf(config), pluginTypes, "")
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

	for i := range configValue.NumField() {
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

					// Auto-set names for BaseConfig if embedded
					setBaseConfigName(newConfig, pluginType)

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

// setBaseConfigName automatically sets name for config types
// This function works with both embedded BaseConfig and custom implementations
// Uses the Config interface methods for maximum flexibility
func setBaseConfigName(config Config, typeName string) {
	// Check if config already has a name set
	if config.Name() != "" {
		return // User has already set a name, respect their choice
	}

	// Try to set the name using the Config interface
	config.SetName(typeName)

	// Verify that the name was set successfully
	if config.Name() == "" {
		// Log a warning if SetName didn't work as expected
		slog.Warn("Failed to set config name - SetName method may not be properly implemented",
			"type", typeName,
			"config_type", fmt.Sprintf("%T", config))
	}
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
	for i := range v.NumField() {
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
