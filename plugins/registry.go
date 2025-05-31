package plugins

import (
	"fmt"
	"log/slog"
	"maps"
	"sync"
)

type (
	// globalPluginRegistry 全局插件注册表
	globalPluginRegistry struct {
		mu      sync.RWMutex
		plugins map[string]*PluginEntry
	}

	// PluginEntry 插件注册表条目
	PluginEntry struct {
		Plugin  Plugin
		Config  Config
		started bool // 插件启动状态
	}
)

var (
	globalRegistry     *globalPluginRegistry
	globalRegistryOnce sync.Once
)

// getGlobalPluginRegistry 返回单例全局插件注册表实例
func getGlobalPluginRegistry() *globalPluginRegistry {
	globalRegistryOnce.Do(func() {
		globalRegistry = &globalPluginRegistry{
			plugins: make(map[string]*PluginEntry),
		}
	})
	return globalRegistry
}

// RegisterGlobalPlugin 注册全局插件
func RegisterGlobalPlugin(plugin Plugin, config Config) {
	registry := getGlobalPluginRegistry()
	registry.mu.Lock()
	defer registry.mu.Unlock()

	if plugin.Name() != config.Name() {
		panic(fmt.Sprintf("plugin name mismatch: %s != %s", plugin.Name(), plugin.Name()))
	}

	if _, exists := registry.plugins[plugin.Name()]; exists {
		slog.Warn("plugin %s already registered", "name", plugin.Name())
	}

	entry := &PluginEntry{
		Plugin: plugin,
		Config: config,
	}

	registry.plugins[plugin.Name()] = entry

	slog.Info("Plugin registered", "name", plugin.Name())
}

// UnregisterGlobalPlugin 注销全局插件
func UnregisterGlobalPlugin(name string) {
	registry := getGlobalPluginRegistry()
	registry.mu.Lock()
	defer registry.mu.Unlock()

	if _, exists := registry.plugins[name]; !exists {
		slog.Warn("global plugin %s not found", "name", name)
	}

	delete(registry.plugins, name)
}

// ListGlobalPlugins 列出所有全局插件
func ListGlobalPlugins() []*PluginEntry {
	registry := getGlobalPluginRegistry()
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	entries := make([]*PluginEntry, 0, len(registry.plugins))
	for _, entry := range registry.plugins {
		entries = append(entries, entry)
	}

	return entries
}

// Clone 获取所有注册的插件
func Clone() map[string]*PluginEntry {
	registry := getGlobalPluginRegistry()
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	// 返回副本以避免并发修改
	return maps.Clone(registry.plugins)
}

// ClearGlobalRegistry 清空全局注册表
func ClearGlobalRegistry() {
	registry := getGlobalPluginRegistry()
	registry.mu.Lock()
	defer registry.mu.Unlock()

	registry.plugins = make(map[string]*PluginEntry)
}
