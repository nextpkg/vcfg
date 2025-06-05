// Package plugins provides the core plugin system for vcfg configuration management.
// It defines interfaces, types, and structures for plugin registration, lifecycle management,
// and configuration handling. The plugin system allows extending vcfg functionality
// through modular components.
package plugins

import (
	"context"
	"sync"
)

// Plugin defines the core interface that all vcfg plugins must implement.
// It provides lifecycle management methods for plugin initialization,
// configuration reloading, and cleanup.
type Plugin interface {
	// Startup initializes the plugin with the provided configuration.
	// It should perform all necessary setup operations and return an error
	// if initialization fails.
	Startup(ctx context.Context, config any) error
	// Reload updates the plugin configuration at runtime.
	// It allows plugins to adapt to configuration changes without restart.
	Reload(ctx context.Context, config any) error
	// Shutdown gracefully terminates the plugin.
	// It should clean up resources and perform necessary cleanup operations.
	Shutdown(ctx context.Context) error
}

// Config defines the interface for plugin configuration structures.
// All plugin configurations must embed BaseConfig and implement this interface.
type Config interface {
	// baseConfigEmbedded returns the embedded BaseConfig for type identification
	baseConfigEmbedded() *BaseConfig
}

// BaseConfig provides the fundamental configuration structure that all plugin
// configurations must embed. It contains the plugin type identifier.
type BaseConfig struct {
	// Type identifies the plugin type for registration and instantiation
	Type string `json:"type,omitempty" yaml:"type,omitempty" koanf:"type"`
}

// PluginPtr is a generic constraint that ensures a type is both a Plugin
// and a pointer type. This is used for type-safe plugin registration.
type PluginPtr[T any] interface {
	Plugin
	*T
}

// ConfigPtr is a generic constraint that ensures a type is both a Config
// and a pointer type. This is used for type-safe configuration registration.
type ConfigPtr[T any] interface {
	Config
	*T
}

// RegisterOptions contains options for plugin type registration.
type RegisterOptions struct {
	// AutoDiscover enables automatic discovery and registration of this plugin type
	AutoDiscover bool
}

// baseConfigEmbedded implements the Config interface by returning the embedded BaseConfig.
// This method allows the plugin system to access type information from any configuration.
func (bc *BaseConfig) baseConfigEmbedded() *BaseConfig {
	return bc
}

// globalPluginTypeRegistry manages the global registry of plugin types.
// It provides thread-safe access to plugin and configuration factories.
type globalPluginTypeRegistry struct {
	// mu protects concurrent access to the plugin registry
	mu sync.RWMutex
	// pluginTypes maps plugin type names to their registration entries
	pluginTypes map[string]*pluginTypeEntry
}

// pluginTypeEntry contains all information needed to create instances
// of a specific plugin type, including factories and metadata.
type pluginTypeEntry struct {
	// PluginFactory creates new instances of the plugin
	PluginFactory pluginFactory
	// ConfigFactory creates new instances of the plugin's configuration
	ConfigFactory configFactory
	// PluginType is the unique identifier for this plugin type
	PluginType string
	// AutoDiscover indicates if this plugin type supports auto-discovery
	AutoDiscover bool
}

// pluginFactory is a function type that creates new plugin instances.
type pluginFactory func() Plugin

// configFactory is a function type that creates new configuration instances.
type configFactory func() Config

// PluginEntry represents a registered plugin instance with its configuration
// and runtime state. It tracks both the plugin and its associated metadata.
type PluginEntry struct {
	// Plugin is the actual plugin instance
	Plugin Plugin
	// Config is the configuration associated with this plugin instance
	Config Config
	// PluginType identifies the type of this plugin
	PluginType string
	// InstanceName is the unique name for this plugin instance
	InstanceName string
	// ConfigPath is the configuration path where this plugin's config is located
	ConfigPath string
	// started tracks whether this plugin instance has been started
	started bool
}
