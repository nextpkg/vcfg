// Package plugins provides a global plugin registry system for managing plugin types
// and their corresponding configuration factories. This file implements the core
// registration mechanism that allows plugins to be discovered and instantiated dynamically.
package plugins

import (
	"fmt"
	"maps"
	"reflect"
	"sync"

	"github.com/nextpkg/vcfg/slogs"
)

var (
	// globalRegistry holds the singleton instance of the plugin type registry
	globalRegistry *globalPluginTypeRegistry
	// globalRegistryOnce ensures the registry is initialized only once
	globalRegistryOnce sync.Once
)

// getGlobalPluginRegistry returns the singleton global plugin registry instance.
// It uses sync.Once to ensure thread-safe initialization of the registry.
// The registry is lazily initialized on first access.
func getGlobalPluginRegistry() *globalPluginTypeRegistry {
	globalRegistryOnce.Do(func() {
		globalRegistry = &globalPluginTypeRegistry{
			pluginTypes: make(map[string]*pluginTypeEntry),
		}
	})
	return globalRegistry
}

// RegisterPluginType registers a plugin type with its corresponding configuration type
// in the global registry. It uses Go generics to ensure type safety between plugins
// and their configurations.
//
// Type parameters:
//   - P: Plugin pointer type that implements PluginPtr[PT]
//   - C: Config pointer type that implements ConfigPtr[CT]
//   - PT: The actual plugin type
//   - CT: The actual config type
//
// Parameters:
//   - pluginType: The string identifier for the plugin type. If empty, it will be
//     automatically derived from the config type name.
//   - p: A pointer to the plugin instance used as a prototype
//   - c: A pointer to the config instance used as a prototype
//   - opts: Optional registration options for customizing the registration behavior
//
// The function panics if a plugin type is already registered to prevent conflicts.
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

	slogs.Info("Plugin type registered", "PluginType", pluginType, "auto_discover", autoDiscover)
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
	slogs.Info("Plugin type unregistered", "type", pluginType)
}

func clonePluginTypes() map[string]*pluginTypeEntry {
	registry := getGlobalPluginRegistry()
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	return maps.Clone(registry.pluginTypes)
}
