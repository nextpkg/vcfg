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
	Config any

	// Namer is an optional interface that plugins and configs can implement
	// to provide custom type names instead of using reflection-based naming
	Namer interface {
		// Name returns the unique name of the plugin type
		Name() string
	}

	// BaseConfig provides a default implementation for plugin configuration
	// Users can embed this struct to reduce boilerplate code
	BaseConfig struct {
		name string
	}
)

// NewBaseConfig creates a new BaseConfig with the given name
// This function automatically sets the config name based on the type
func NewBaseConfig(name string) BaseConfig {
	return BaseConfig{name: name}
}

// Name returns the config name
func (bc *BaseConfig) Name() string {
	return bc.name
}

// SetName sets the config name (used internally during registration)
func (bc *BaseConfig) SetName(name string) {
	bc.name = name
}

// GetConfigTypeName extracts the config type name, preferring custom Name() method over reflection
func GetConfigTypeName(config Config) string {
	// Check if config implements Namer interface for custom naming
	if namer, ok := config.(Namer); ok {
		if name := namer.Name(); name != "" {
			return name
		}
	}

	// Check if config is directly a BaseConfig with a name set
	if baseConfig, ok := config.(*BaseConfig); ok && baseConfig.name != "" {
		return baseConfig.name
	}

	// Use reflection to check for embedded BaseConfig
	if name := getNameFromEmbeddedBase(config); name != "" {
		return name
	}

	// Fall back to reflection-based naming
	return getTypeNameByReflection(config)
}

// getNameFromEmbeddedBase tries to extract name from embedded BaseConfig using reflection
func getNameFromEmbeddedBase(config Config) string {
	v := reflect.ValueOf(config)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return ""
	}

	// Look for embedded BaseConfig field
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := v.Type().Field(i)

		// Check if this field is an embedded BaseConfig
		if fieldType.Anonymous && fieldType.Type == reflect.TypeOf(BaseConfig{}) {
			if baseConfig, ok := field.Interface().(BaseConfig); ok {
				// Return the name if it's set, otherwise return empty to continue with fallback
				return baseConfig.name
			}
		}

		// Check if this field is a pointer to BaseConfig
		if fieldType.Anonymous && fieldType.Type == reflect.TypeOf(&BaseConfig{}) {
			if !field.IsNil() {
				if baseConfig, ok := field.Interface().(*BaseConfig); ok {
					// Return the name if it's set, otherwise return empty to continue with fallback
					return baseConfig.name
				}
			}
		}
	}

	return ""
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
