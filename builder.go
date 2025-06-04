package vcfg

import (
	"fmt"
	"log/slog"

	"github.com/knadh/koanf/providers/cliflagv3"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
	"github.com/urfave/cli/v3"

	"github.com/nextpkg/vcfg/plugins"
	"github.com/nextpkg/vcfg/providers"
)

// Builder 配置管理器构建器
type Builder[T any] struct {
	sources      []any
	plugins      []plugins.PluginEntry
	enableWatch  bool
	enablePlugin bool
}

// NewBuilder 创建新的构建器
func NewBuilder[T any]() *Builder[T] {
	return &Builder[T]{
		sources: make([]any, 0),
		plugins: make([]plugins.PluginEntry, 0),
	}
}

// AddFile 添加文件配置源
func (b *Builder[T]) AddFile(path string) *Builder[T] {
	b.sources = append(b.sources, path)
	return b
}

// AddEnv 添加环境变量配置源
func (b *Builder[T]) AddEnv(prefix string) *Builder[T] {
	envProvider := env.ProviderWithValue(prefix, ".", func(s string, v string) (string, any) {
		return s, v
	})
	b.sources = append(b.sources, envProvider)
	return b
}

// AddProvider 添加自定义配置源
func (b *Builder[T]) AddProvider(provider koanf.Provider) *Builder[T] {
	b.sources = append(b.sources, provider)
	return b
}

// AddCliFlags 添加 CLI flags 配置源
// CLI flags are typically added last to ensure they override other configuration sources.
func (b *Builder[T]) AddCliFlags(cmd *cli.Command, delim string) *Builder[T] {
	// 创建一个包装的 Provider 来处理键名映射
	cliProvider := providers.NewCliProviderWrapper(cliflagv3.Provider(cmd, delim), cmd.Name, delim)

	slog.Debug("AddCliFlags: created wrapper", "cmd", cmd.Name, "delim", delim)

	b.sources = append(b.sources, cliProvider)
	return b
}

// WithWatch 启用配置监听
func (b *Builder[T]) WithWatch() *Builder[T] {
	b.enableWatch = true
	return b
}

func (b *Builder[T]) WithPlugin() *Builder[T] {
	b.enablePlugin = true
	return b
}

// Build 构建配置管理器
func (b *Builder[T]) Build() (*ConfigManager[T], error) {
	if len(b.sources) == 0 {
		return nil, fmt.Errorf("at least one configuration source is required")
	}

	// 创建配置管理器
	cm := newManager[T](b.sources...)

	// 加载初始配置
	cfg, err := cm.load()
	if err != nil {
		return nil, fmt.Errorf("failed to load initial configuration: %w", err)
	}
	cm.cfg.Store(cfg)

	// 启用插件
	if b.enablePlugin {
		err = cm.pluginManager.Initialize(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize plugins: %w", err)
		}

		err = cm.pluginManager.Startup()
		if err != nil {
			return nil, fmt.Errorf("failed to startup plugins: %w", err)
		}
	}

	// 启用监听
	if b.enableWatch {
		cm.EnableWatch()
	}

	return cm, nil
}

// MustBuild 构建配置管理器，失败时panic
func (b *Builder[T]) MustBuild() *ConfigManager[T] {
	cm, err := b.Build()
	if err != nil {
		panic(err)
	}
	return cm
}
