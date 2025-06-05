// Package vcfg provides a flexible configuration management system.
// This file implements the Builder pattern for constructing ConfigManager instances
// with various configuration sources and options.
package vcfg

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/knadh/koanf/providers/cliflagv3"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
	"github.com/urfave/cli/v3"

	"github.com/nextpkg/vcfg/plugins"
	"github.com/nextpkg/vcfg/providers"
)

// Builder provides a fluent interface for constructing ConfigManager instances.
// It allows step-by-step configuration of various sources, plugins, and options
// before building the final ConfigManager.
type Builder[T any] struct {
	// sources holds the configuration sources (file paths, providers, etc.)
	sources []any
	// plugins holds manually added plugin entries
	plugins []plugins.PluginEntry
	// enableWatch determines if configuration file watching should be enabled
	enableWatch bool
	// enablePlugin determines if plugin discovery and initialization should be enabled
	enablePlugin bool
}

// NewBuilder creates a new Builder instance for configuration type T.
// The builder is initialized with empty sources and plugins, ready for configuration.
func NewBuilder[T any]() *Builder[T] {
	return &Builder[T]{
		sources: make([]any, 0),
		plugins: make([]plugins.PluginEntry, 0),
	}
}

// AddFile adds a file path as a configuration source.
// The file format will be automatically detected based on the file extension.
// Supported formats include JSON, YAML, TOML, and others supported by koanf.
func (b *Builder[T]) AddFile(path string) *Builder[T] {
	b.sources = append(b.sources, path)
	return b
}

// AddEnv adds environment variables as a configuration source.
// Environment variables with the specified prefix will be included,
// with the prefix stripped and keys converted using dot notation.
func (b *Builder[T]) AddEnv(prefix string) *Builder[T] {
	envProvider := env.ProviderWithValue(prefix, ".", func(s string, v string) (string, any) {
		// Remove the prefix and convert environment variable names to configuration keys
		// e.g., APP_SERVER_PORT -> server.port
		key := strings.TrimPrefix(s, prefix)
		key = strings.ToLower(strings.ReplaceAll(key, "_", "."))
		return key, v
	})
	b.sources = append(b.sources, envProvider)
	return b
}

// AddProvider adds a custom koanf.Provider as a configuration source.
// This allows integration with any provider that implements the koanf.Provider interface.
func (b *Builder[T]) AddProvider(provider koanf.Provider) *Builder[T] {
	b.sources = append(b.sources, provider)
	return b
}

// AddCliFlags adds CLI flags as a configuration source using the urfave/cli library.
// CLI flags are typically added last to ensure they override other configuration sources.
// The flags are processed through a wrapper that handles key name mapping and flattening.
func (b *Builder[T]) AddCliFlags(cmd *cli.Command, delim string) *Builder[T] {
	// Create a wrapped Provider to handle key name mapping
	cliProvider := providers.NewCliProviderWrapper(cliflagv3.Provider(cmd, delim), cmd.Name, delim)

	slog.Debug("AddCliFlags: created wrapper", "cmd", cmd.Name, "delim", delim)

	b.sources = append(b.sources, cliProvider)
	return b
}

// WithWatch enables configuration file watching for automatic reloading.
// When enabled, the ConfigManager will monitor configuration files for changes
// and automatically reload the configuration when modifications are detected.
func (b *Builder[T]) WithWatch() *Builder[T] {
	b.enableWatch = true
	return b
}

// WithPlugin enables plugin discovery and initialization.
// When enabled, the ConfigManager will automatically discover plugin configurations
// in the loaded config and initialize the corresponding plugin instances.
func (b *Builder[T]) WithPlugin() *Builder[T] {
	b.enablePlugin = true
	return b
}

// Build constructs and returns a ConfigManager instance based on the builder's configuration.
// It loads the initial configuration, initializes plugins if enabled, and sets up
// file watching if enabled.
//
// Parameters:
//   - ctx: Context for plugin initialization and other operations
//
// Returns a fully configured ConfigManager or an error if building fails.
func (b *Builder[T]) Build(ctx context.Context) (*ConfigManager[T], error) {
	if len(b.sources) == 0 {
		return nil, fmt.Errorf("at least one configuration source is required")
	}

	// Create configuration manager
	cm := newManager[T](b.sources...)

	// Load initial configuration
	cfg, err := cm.load()
	if err != nil {
		return nil, fmt.Errorf("failed to load initial configuration: %w", err)
	}
	cm.cfg.Store(cfg)

	// Enable plugins
	if b.enablePlugin {
		err = cm.pluginManager.DiscoverAndRegister(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to register plugins: %w", err)
		}

		err = cm.pluginManager.Startup(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to startup plugins: %w", err)
		}
	}

	// Enable watching
	if b.enableWatch {
		cm.EnableWatch()
	}

	return cm, nil
}

// MustBuild 构建配置管理器，失败时panic
func (b *Builder[T]) MustBuild() *ConfigManager[T] {
	cm, err := b.Build(context.Background())
	if err != nil {
		panic(err)
	}
	return cm
}
