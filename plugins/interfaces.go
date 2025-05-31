package plugins

import (
	"reflect"
	"strings"
)

type (
	// Plugin defines the generic plugin interface
	// This is a type-safe plugin interface that provides hot reload capability with minimal boilerplate
	Plugin interface {
		// Start initializes the plugin with configuration
		// This method is called when the plugin is first loaded
		Start(config any) error

		// Reload is called when plugin configuration changes
		// The plugin should gracefully update its behavior based on the new configuration
		Reload(config any) error

		// Stop gracefully shuts down the plugin
		Stop() error
	}

	// Config defines the generic plugin configuration interface
	// Note: Instance names are derived from config field paths, not from this method
	Config interface {
		// Empty interface - plugins can optionally implement Namer for custom names
	}

	// Namer is an optional interface that plugins and configs can implement
	// to provide custom type names instead of using reflection-based naming
	Namer interface {
		// Name returns the unique name of the plugin type
		Name() string
	}
)

// GetPluginTypeName extracts the plugin type name, preferring custom Name() method over reflection
func GetPluginTypeName(plugin Plugin) string {
	// Check if plugin implements Namer interface for custom naming
	if namer, ok := plugin.(Namer); ok {
		return namer.Name()
	}

	// Fall back to reflection-based naming
	return getTypeNameByReflection(plugin)
}

// GetConfigTypeName extracts the config type name, preferring custom Name() method over reflection
func GetConfigTypeName(config Config) string {
	// Check if config implements Namer interface for custom naming
	if namer, ok := config.(Namer); ok {
		return namer.Name()
	}

	// Fall back to reflection-based naming
	return getTypeNameByReflection(config)
}

// getTypeNameByReflection derives type name from struct name using reflection
// This function automatically removes common suffixes and converts to lowercase
func getTypeNameByReflection(v any) string {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	name := strings.ToLower(t.Name())

	// Remove common suffixes to derive clean type names
	suffixes := []string{"plugin", "config", "impl", "service", "cfg", "configuration"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(name, suffix) {
			return strings.TrimSuffix(name, suffix)
		}
	}

	return name
}
