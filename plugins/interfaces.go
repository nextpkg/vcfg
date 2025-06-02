package plugins

import (
	"reflect"
	"strings"
)

type (
	// Plugin defines the generic plugin interface
	// This is a type-safe plugin interface that provides hot reload capability with minimal boilerplate
	Plugin interface {
		// Name returns the plugin name for identification
		Name() string

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
		// Name returns the config name
		Name() string
		// SetName sets the config name
		SetName(string)
	}

	// BaseConfig provides a default implementation for plugin configuration
	// Users can embed this struct to reduce boilerplate code
	BaseConfig struct {
		name string
	}
)

// Name returns the config name
func (bc *BaseConfig) Name() string {
	return bc.name
}

// SetName sets the config name (used internally during registration)
func (bc *BaseConfig) SetName(name string) {
	bc.name = name
}

// GetConfigTypeName extracts the config type name, preferring Config.Name() method over reflection
func GetConfigTypeName(config Config) string {
	// First try to use the Config interface Name() method
	if name := config.Name(); name != "" {
		return name
	}

	// Fall back to reflection-based naming
	return getTypeNameByReflection(config)
}

// getTypeNameByReflection derives type name from struct name using reflection
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
