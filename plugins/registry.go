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

// getInstanceName extracts instance name from config
// Now simply uses config type name as the instance name
func getInstanceName(config Config) string {
	return GetConfigTypeName(config)
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

// RegisterType registers a plugin type using generics for type safety and simplicity
// This is the recommended way to register plugin types with compile-time type checking
func RegisterType[P Plugin, C Config](opts ...RegisterOptions) {
	var p P
	var c C
	pluginType := reflect.TypeOf(p)
	configType := reflect.TypeOf(c)

	pluginFactory := func() Plugin {
		return reflect.New(pluginType.Elem()).Interface().(Plugin)
	}
	configFactory := func() Config {
		return reflect.New(configType.Elem()).Interface().(Config)
	}
	RegisterPluginType(pluginFactory, configFactory, opts...)
}

// RegisterPluginType registers a plugin type with factory functions for auto-discovery
// This is the recommended way to register plugin types for automatic instantiation
func RegisterPluginType(pluginFactory PluginFactory, configFactory ConfigFactory, opts ...RegisterOptions) {
	registry := getGlobalPluginRegistry()
	registry.mu.Lock()
	defer registry.mu.Unlock()

	// Create sample instances to get plugin type
	plugin := pluginFactory()
	pluginType := GetPluginTypeName(plugin)

	// Determine auto-discovery setting
	autoDiscover := true
	if len(opts) > 0 {
		autoDiscover = opts[0].AutoDiscover
	}

	entry := &PluginTypeEntry{
		PluginFactory: pluginFactory,
		ConfigFactory: configFactory,
		PluginType:    pluginType,
		AutoDiscover:  autoDiscover,
	}

	registry.pluginTypes[pluginType] = entry
	slog.Info("Plugin type registered", "type", pluginType, "auto_discover", autoDiscover)
}

// Register registers a plugin instance with optional configuration
// This provides a simplified API for manual plugin registration
func Register(plugin Plugin, config Config, opts ...RegisterOptions) {
	registry := getGlobalPluginRegistry()
	registry.mu.Lock()
	defer registry.mu.Unlock()

	pluginType := GetPluginTypeName(plugin)
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

	if instanceName == "" {
		instanceName = getInstanceName(config)
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

// RegisterGlobalPlugin 注册全局插件 (Legacy API for backward compatibility)
// 支持多实例注册，通过 pluginType:instanceName 作为唯一键
func RegisterGlobalPlugin(plugin Plugin, config Config) {
	Register(plugin, config)
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

					// Copy configuration values
					if err := copyConfigValues(fieldInterface, newConfig); err != nil {
						slog.Error("Failed to copy config values", "path", fieldPath, "type", pluginType, "error", err)
						continue
					}

					// Use field path as instance name to support multiple instances
					// This allows the same plugin type to have different instances based on config location
					instanceName := strings.ToLower(fieldPath)

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

// RegisterGlobalPluginWithInstance 注册全局插件并指定实例名称 (Legacy API)
// 这是推荐的多实例注册方式
func RegisterGlobalPluginWithInstance(plugin Plugin, config Config, instanceName string) {
	Register(plugin, config, RegisterOptions{InstanceName: instanceName})
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

// UnregisterGlobalPluginInstance 注销指定实例的全局插件
func UnregisterGlobalPluginInstance(pluginType, instanceName string) {
	key := makePluginKey(pluginType, instanceName)
	UnregisterGlobalPlugin(key)
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

// GetGlobalPluginsByType 获取指定类型的所有插件实例
func GetGlobalPluginsByType(pluginType string) []*PluginEntry {
	registry := getGlobalPluginRegistry()
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	var entries []*PluginEntry
	for _, entry := range registry.plugins {
		if entry.PluginType == pluginType {
			entries = append(entries, entry)
		}
	}
	return entries
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

	registry.plugins = make(map[string]*PluginEntry)
	registry.pluginTypes = make(map[string]*PluginTypeEntry)
}
