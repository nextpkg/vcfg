package plugins

import (
	"fmt"
	"log/slog"
	"maps"
	"reflect"
	"sync"
)

var (
	globalRegistry     *globalPluginTypeRegistry
	globalRegistryOnce sync.Once
)

// getGlobalPluginRegistry 返回单例全局插件注册表实例
func getGlobalPluginRegistry() *globalPluginTypeRegistry {
	globalRegistryOnce.Do(func() {
		globalRegistry = &globalPluginTypeRegistry{
			pluginTypes: make(map[string]*pluginTypeEntry),
		}
	})
	return globalRegistry
}

func RegisterPluginType[P PluginPtr[PT], C ConfigPtr[CT], PT any, CT any](pluginType string, p P, c C, opts ...RegisterOptions) {
	registry := getGlobalPluginRegistry()
	registry.mu.Lock()
	defer registry.mu.Unlock()

	if pluginType == "" {
		pluginType = getConfigType(c)
	}

	_, exists := registry.pluginTypes[pluginType]
	if exists {
		panic(fmt.Sprintf("plugin type is registerd, type=%s", pluginType))
	}

	// Create simplified type-safe factory functions using generics
	// Since P and C are constrained to be pointer types, we can simplify the logic
	pluginFactory := func() Plugin {
		return reflect.New(reflect.TypeOf(*p)).Interface().(Plugin)
	}

	configFactory := func() Config {
		return reflect.New(reflect.TypeOf(*c)).Interface().(Config)
	}

	// Determine auto-discovery setting
	autoDiscover := true
	if len(opts) > 0 {
		autoDiscover = opts[0].AutoDiscover
	}

	registry.pluginTypes[pluginType] = &pluginTypeEntry{
		PluginType:    pluginType,
		PluginFactory: pluginFactory,
		ConfigFactory: configFactory,
		AutoDiscover:  autoDiscover,
	}

	slog.Info("Plugin type registered", "PluginType", pluginType, "auto_discover", autoDiscover)
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

// UnregisterPluginType removes a plugin type from the registry
func UnregisterPluginType(pluginType string) {
	registry := getGlobalPluginRegistry()
	registry.mu.Lock()
	defer registry.mu.Unlock()

	delete(registry.pluginTypes, pluginType)
	slog.Info("Plugin type unregistered", "type", pluginType)
}

func clonePluginTypes() map[string]*pluginTypeEntry {
	registry := getGlobalPluginRegistry()
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	return maps.Clone(registry.pluginTypes)
}
