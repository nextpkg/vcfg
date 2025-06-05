// Package builtins provides built-in plugin implementations for the vcfg configuration system.
// This package automatically registers all built-in plugins when imported, making them
// available for use without manual registration.
package builtins

import "github.com/nextpkg/vcfg/plugins"

// init automatically registers all built-in plugins with the global plugin registry.
// This function is called when the package is imported, ensuring that all built-in
// plugins are available for discovery and use by the configuration system.
//
// Currently registered plugins:
//   - LoggerPlugin: Provides logging functionality with configurable levels, formats, and outputs
func init() {
	// Register logger plugin with automatic type detection (empty string for plugin type)
	plugins.RegisterPluginType("", &LoggerPlugin{}, &LoggerConfig{})
}
