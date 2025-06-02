package vcfg

import "github.com/nextpkg/vcfg/plugins"

// GetConfig returns the current configuration
func (cm *ConfigManager[T]) GetConfig() *T {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// 如果还没有加载配置，返回nil
	if cm.koanf == nil {
		return nil
	}

	// 尝试获取当前配置
	config, err := cm.loadConfig()
	if err != nil {
		return nil
	}

	return config
}

// MustInit initializes a new ConfigManager with the provided sources.
// It accepts both file paths (strings) and koanf.Provider instances.
// It panics if initialization fails.
func MustInit[T any](sources ...any) *ConfigManager[T] {
	cm := newManager[T](sources...)

	// Load initial configuration
	cfg, err := cm.load()
	if err != nil {
		panic(err)
	}

	cm.cfg.Store(cfg)
	return cm
}

// RegisterGlobalPlugin registers a plugin globally
// The plugin will be automatically registered to any ConfigManager created afterwards
func RegisterGlobalPlugin(plugin plugins.Plugin, config plugins.Config) {
	plugins.Register(plugin, config)
}

// UnregisterGlobalPlugin removes a plugin from global registry
func UnregisterGlobalPlugin(name string) {
	plugins.UnregisterGlobalPlugin(name)
}

// ListGlobalPlugins returns a list of all registered global plugins
func ListGlobalPlugins() []*plugins.PluginEntry {
	return plugins.ListGlobalPlugins()
}

// ClearGlobalRegistry clears all registered global plugins
func ClearGlobalRegistry() {
	plugins.ClearGlobalRegistry()
}
