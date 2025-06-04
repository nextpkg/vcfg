package builtins

import "github.com/nextpkg/vcfg/plugins"

// init registers all builtin plugins
// This function automatically registers all builtin plugins when the package is imported
func init() {
	// Register logger plugin
	plugins.RegisterPluginType("vcfg-logger", &LoggerPlugin{}, &LoggerConfig{})
}
