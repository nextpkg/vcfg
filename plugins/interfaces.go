package plugins

import (
	"context"
	"sync"
)

// 插件定义
type (
	Plugin interface {
		Startup(ctx context.Context, config any) error
		Reload(ctx context.Context, config any) error
		Shutdown(ctx context.Context) error
	}
	Config interface {
		baseConfigEmbedded() *BaseConfig
	}
)

type (
	BaseConfig struct {
		Type string `json:"type,omitempty" yaml:"type,omitempty" koanf:"type"`
	}

	PluginPtr[T any] interface {
		Plugin
		*T
	}
	ConfigPtr[T any] interface {
		Config
		*T
	}

	RegisterOptions struct {
		AutoDiscover bool // Whether to enable auto-discovery for this type
	}
)

func (bc *BaseConfig) baseConfigEmbedded() *BaseConfig {
	return bc
}

type (
	// globalPluginTypeRegistry 全局插件注册表
	globalPluginTypeRegistry struct {
		mu          sync.RWMutex
		pluginTypes map[string]*pluginTypeEntry // key: pluginType
	}

	pluginTypeEntry struct {
		PluginFactory pluginFactory
		ConfigFactory configFactory
		PluginType    string
		AutoDiscover  bool
	}

	pluginFactory func() Plugin
	configFactory func() Config
)

type (
	// PluginEntry 插件注册表条目
	PluginEntry struct {
		Plugin       Plugin
		Config       Config
		PluginType   string // 插件类型名称
		InstanceName string // 实例名称
		ConfigPath   string // 配置路径
		started      bool   // 插件启动状态
	}
)
