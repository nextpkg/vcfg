package vcfg

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// Plugin 定义插件接口
type Plugin[T any] interface {
	// Name 返回插件名称
	Name() string
	// Initialize 初始化插件
	Initialize(ctx context.Context, manager *ConfigManager[T]) error
	// OnConfigLoaded 配置加载完成时的回调
	OnConfigLoaded(ctx context.Context, config *T) error
	// OnConfigChanged 配置变更时的回调
	OnConfigChanged(ctx context.Context, oldConfig, newConfig *T) error
	// Shutdown 关闭插件
	Shutdown(ctx context.Context) error
}

// PluginManager 管理插件的生命周期
type PluginManager[T any] struct {
	plugins map[string]Plugin[T]
	mu      sync.RWMutex
	manager *ConfigManager[T]
}

// NewPluginManager 创建新的插件管理器
func NewPluginManager[T any](manager *ConfigManager[T]) *PluginManager[T] {
	return &PluginManager[T]{
		plugins: make(map[string]Plugin[T]),
		manager: manager,
	}
}

// Register 注册插件
func (pm *PluginManager[T]) Register(plugin Plugin[T]) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	name := plugin.Name()
	if _, exists := pm.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}

	ctx := context.Background()
	if err := plugin.Initialize(ctx, pm.manager); err != nil {
		return fmt.Errorf("failed to initialize plugin %s: %w", name, err)
	}

	pm.plugins[name] = plugin
	return nil
}

// Unregister 注销插件
func (pm *PluginManager[T]) Unregister(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	ctx := context.Background()
	if err := plugin.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown plugin %s: %w", name, err)
	}

	delete(pm.plugins, name)
	return nil
}

// Get 获取插件
func (pm *PluginManager[T]) Get(name string) (Plugin[T], bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugin, exists := pm.plugins[name]
	return plugin, exists
}

// List 列出所有插件名称
func (pm *PluginManager[T]) List() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	names := make([]string, 0, len(pm.plugins))
	for name := range pm.plugins {
		names = append(names, name)
	}
	return names
}

// NotifyConfigLoaded 通知所有插件配置已加载
func (pm *PluginManager[T]) NotifyConfigLoaded(ctx context.Context, config *T) {
	if pm == nil || config == nil {
		return
	}

	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for name, plugin := range pm.plugins {
		if plugin == nil {
			continue
		}
		if err := plugin.OnConfigLoaded(ctx, config); err != nil {
			// 记录错误但不中断其他插件
			slog.Error("Plugin OnConfigLoaded error", "plugin", name, "error", err)
		}
	}
}

// NotifyConfigChanged 通知所有插件配置已变更
func (pm *PluginManager[T]) NotifyConfigChanged(ctx context.Context, oldConfig, newConfig *T) {
	if pm == nil || newConfig == nil {
		return
	}

	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for name, plugin := range pm.plugins {
		if plugin == nil {
			continue
		}
		if err := plugin.OnConfigChanged(ctx, oldConfig, newConfig); err != nil {
			// 记录错误但不中断其他插件
			slog.Error("Plugin OnConfigChanged error", "plugin", name, "error", err)
		}
	}
}

// Shutdown 关闭所有插件
func (pm *PluginManager[T]) Shutdown(ctx context.Context) error {
	if pm == nil {
		return nil
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	var errors []error
	for name, plugin := range pm.plugins {
		if plugin == nil {
			continue
		}
		if err := plugin.Shutdown(ctx); err != nil {
			errors = append(errors, fmt.Errorf("plugin %s shutdown failed: %w", name, err))
		}
	}

	// 清空插件映射
	pm.plugins = make(map[string]Plugin[T])

	if len(errors) > 0 {
		return fmt.Errorf("multiple plugin shutdown errors: %v", errors)
	}
	return nil
}
