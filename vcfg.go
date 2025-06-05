// Package vcfg provides a comprehensive configuration management system with support
// for multiple configuration sources, automatic type conversion, validation, and plugins.
// It offers both simple and advanced configuration loading patterns for Go applications.
package vcfg

// MustLoad is a convenience function that initializes a new ConfigManager with the provided sources
// and loads the initial configuration. It accepts both file paths (strings) and koanf.Provider instances.
//
// Type parameter:
//   - T: The configuration struct type to unmarshal into
//
// Parameters:
//   - sources: Variable number of configuration sources (file paths or koanf.Provider instances)
//
// Returns a fully initialized ConfigManager with the configuration loaded.
// Panics if initialization or loading fails - use Builder for error handling.
func MustLoad[T any](sources ...any) *ConfigManager[T] {
	cm := newManager[T](sources...)

	// Load initial configuration
	cfg, err := cm.load()
	if err != nil {
		panic(err)
	}

	cm.cfg.Store(cfg)
	return cm
}

// MustBuild is a convenience function that creates a simple configuration manager
// using only file sources. It's a shorthand for common use cases where only
// file-based configuration is needed.
//
// Type parameter:
//   - T: The configuration struct type to unmarshal into
//
// Parameters:
//   - filePaths: Variable number of file paths to load configuration from
//
// Returns a ConfigManager configured with the specified files.
// Panics if building fails - use Builder for error handling.
func MustBuild[T any](filePaths ...string) *ConfigManager[T] {
	builder := NewBuilder[T]()
	for _, path := range filePaths {
		builder.AddFile(path)
	}
	return builder.MustBuild()
}
