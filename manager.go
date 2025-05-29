package vcfg

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"go.uber.org/atomic"

	"github.com/nextpkg/vcfg/validator"
)

type (
	// ProviderConfig holds a koanf provider with its parser
	ProviderConfig struct {
		Provider koanf.Provider
		Parser   koanf.Parser
	}
	// ConfigManager is a configuration manager that handles loading, validation, and watching.
	// It supports generic configuration types through the type parameter T.
	// ConfigManager provides thread-safe access to configuration values.
	ConfigManager[T any] struct {
		providers     []ProviderConfig
		koanf         *koanf.Koanf
		once          sync.Once
		cfg           atomic.Value
		mu            sync.RWMutex
		watchers      []func() // cleanup functions for watchers
		pluginManager *PluginManager[T]
	}

	// Watcher interface for providers that support watching configuration changes
	Watcher interface {
		Watch(cb func(event any, err error)) error
	}

	// Unwatcher interface for providers that support stopping watch
	Unwatcher interface {
		Unwatch()
	}
)

// newManager creates a new configuration manager with the provided sources.
// It accepts file paths and koanf.Provider instances.
// Parameters:
//   - sources: file paths (strings) or koanf.Provider instances
//
// Returns:
//   - A new ConfigManager instance.
func newManager[T any](sources ...any) *ConfigManager[T] {
	var providers []ProviderConfig

	for _, src := range sources {
		switch s := src.(type) {
		case string:
			// File path - create file provider with appropriate parser
			provider := file.Provider(s)
			parser := getParserForFile(s)
			providers = append(providers, ProviderConfig{Provider: provider, Parser: parser})
		case koanf.Provider:
			// Direct koanf.Provider - determine parser based on provider type
			var parser koanf.Parser
			switch s.(type) {
			case *env.Env:
				// Env provider doesn't need a parser
				parser = nil
			default:
				// Use YAML parser as default for other providers
				parser = yaml.Parser()
			}
			providers = append(providers, ProviderConfig{Provider: s, Parser: parser})
		default:
			panic(fmt.Sprintf("unsupported source type: %T, expected string (file path) or koanf.Provider", src))
		}
	}

	cm := &ConfigManager[T]{
		providers: providers,
		koanf:     koanf.New("."),
		watchers:  make([]func(), 0),
	}
	cm.pluginManager = NewPluginManager[T](cm)
	return cm
}

// getParserForFile returns the appropriate parser based on file extension
func getParserForFile(path string) koanf.Parser {
	ext := filepath.Ext(path)
	switch ext {
	case ".yaml", ".yml":
		return yaml.Parser()
	case ".json":
		return json.Parser()
	default:
		return yaml.Parser() // Default to YAML
	}
}

// load loads configuration from sources, validates it, and returns the configuration struct.
// This method is thread-safe through mutex locking.
// Returns:
//   - A pointer to the loaded and validated configuration.
//   - An error if loading or validation fails.
func (cm *ConfigManager[T]) load() (*T, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// load all sources
	err := cm.loadSource()
	if err != nil {
		return nil, err
	}

	return cm.loadConfig()
}

// loadSource loads all configuration providers and merges them into koanf.
// It reads from each provider and combines the configurations.
// Returns:
//   - An error if reading from any provider or merging configurations fails.
func (cm *ConfigManager[T]) loadSource() error {
	for _, providerConfig := range cm.providers {
		if err := cm.koanf.Load(providerConfig.Provider, providerConfig.Parser); err != nil {
			return NewParseError(fmt.Sprintf("%T", providerConfig.Provider), "failed to load from provider", err)
		}
	}

	return nil
}

// loadConfig unmarshals the configuration from koanf into a struct and validates it.
// Returns:
//   - A pointer to the unmarshaled and validated configuration.
//   - An error if unmarshaling or validation fails.
func (cm *ConfigManager[T]) loadConfig() (*T, error) {
	if cm == nil || cm.koanf == nil {
		return nil, NewParseError("manager", "configuration manager not properly initialized", nil)
	}

	var cfg T

	// Set default values if the type implements SetDefaults
	if def, ok := any(&cfg).(interface{ SetDefaults() }); ok {
		def.SetDefaults()
	}

	if err := cm.koanf.Unmarshal("", &cfg); err != nil {
		return nil, NewParseError("koanf", "failed to unmarshal configuration", err)
	}

	if err := validator.Validate(&cfg); err != nil {
		return nil, NewValidationError("validator", "configuration validation failed", err)
	}

	// 通知插件配置已加载
	if cm.pluginManager != nil {
		ctx := context.Background()
		cm.pluginManager.NotifyConfigLoaded(ctx, &cfg)
	}

	return &cfg, nil
}

// EnableWatch enables watching for configuration changes.
// It sets up file watchers for all file-based providers.
// When a configuration file changes, it automatically reloads the configuration.
// Returns:
//   - An error if setting up watchers fails.
func (cm *ConfigManager[T]) EnableWatch() *ConfigManager[T] {
	slog.Info("Setting up watchers for configuration providers")
	for _, providerConfig := range cm.providers {
		slog.Debug("Checking provider for watch support", "provider", fmt.Sprintf("%T", providerConfig.Provider))

		// Use interface assertion to check if provider supports Watch
		if watcher, ok := providerConfig.Provider.(Watcher); ok {
			slog.Debug("Setting up watcher for provider", "provider", fmt.Sprintf("%T", providerConfig.Provider))

			// Create callback function
			callback := func(event any, err error) {
				if err != nil {
					slog.Error("Watch error", "error", err)
					return
				}

				slog.Info("Configuration file changed, reloading...")
				// 获取旧配置
				oldCfg := cm.Get()
				// Reload configuration
				if newCfg, err := cm.load(); err != nil {
					slog.Error("Failed to reload configuration", "error", err)
				} else {
					// Atomically update the configuration
					cm.cfg.Store(newCfg)
					// 通知插件配置已变更
					ctx := context.Background()
					cm.pluginManager.NotifyConfigChanged(ctx, oldCfg, newCfg)
					slog.Info("Configuration reloaded successfully")
				}
			}

			// Call Watch method using interface
			if err := watcher.Watch(callback); err != nil {
				slog.Error("Failed to set up watcher", "error", err)
				continue
			}

			// Check for Unwatch method and store cleanup function
			if unwatcher, ok := providerConfig.Provider.(Unwatcher); ok {
				cm.watchers = append(cm.watchers, func() {
					unwatcher.Unwatch()
				})
			}
		}
	}

	return cm
}

// DisableWatch stops monitoring changes of all configuration providers.
func (cm *ConfigManager[T]) DisableWatch() {
	for _, cleanup := range cm.watchers {
		cleanup()
	}
	cm.watchers = cm.watchers[:0]
	cm.once = sync.Once{}
}

// Get returns the current configuration value.
// It retrieves the stored configuration from atomic.Value and returns it as type T.
// Returns nil if configuration is not initialized.
func (cm *ConfigManager[T]) Get() *T {
	if cm == nil {
		return nil
	}
	cfg := cm.cfg.Load()
	if cfg == nil {
		return nil
	}
	ret, ok := cfg.(*T)
	if !ok {
		return nil
	}
	return ret
}

// RegisterPlugin 注册插件
func (cm *ConfigManager[T]) RegisterPlugin(plugin Plugin[T]) error {
	return cm.pluginManager.Register(plugin)
}

// UnregisterPlugin 注销插件
func (cm *ConfigManager[T]) UnregisterPlugin(name string) error {
	return cm.pluginManager.Unregister(name)
}

// GetPlugin 获取插件
func (cm *ConfigManager[T]) GetPlugin(name string) (Plugin[T], bool) {
	return cm.pluginManager.Get(name)
}

// ListPlugins 列出所有插件
func (cm *ConfigManager[T]) ListPlugins() []string {
	return cm.pluginManager.List()
}

// Close 关闭配置管理器，包括所有插件和监听器
func (cm *ConfigManager[T]) Close() error {
	if cm == nil {
		return nil
	}

	// 先关闭监听器
	cm.DisableWatch()

	// 再关闭插件
	if cm.pluginManager != nil {
		ctx := context.Background()
		return cm.pluginManager.Shutdown(ctx)
	}
	return nil
}
