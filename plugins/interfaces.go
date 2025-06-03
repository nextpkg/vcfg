package plugins

import "sync"

// 插件定义
type (
	Plugin interface {
		Start(config any) error
		Reload(config any) error
		Stop() error
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
		InstanceName string // Optional: explicit instance name
		ConfigPath   string // Optional: explicit config path
		AutoDiscover bool   // Whether to enable auto-discovery for this type
	}
)

func (bc *BaseConfig) baseConfigEmbedded() *BaseConfig {
	return bc
}

type (
	// globalPluginTypeRegistry 全局插件注册表
	globalPluginTypeRegistry struct {
		mu          sync.RWMutex
		pluginTypes map[string]*PluginTypeEntry // key: pluginType
	}

	PluginTypeEntry struct {
		PluginFactory PluginFactory
		ConfigFactory ConfigFactory
		PluginType    string
		AutoDiscover  bool
	}

	PluginFactory func() Plugin
	ConfigFactory func() Config
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
