package vcfg

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"sync"

	"github.com/nextpkg/vcfg/ce"
	"github.com/nextpkg/vcfg/internal/validator"
	"github.com/nextpkg/vcfg/internal/viper"
	"github.com/nextpkg/vcfg/internal/watcher"
	"github.com/nextpkg/vcfg/source"
)

// ConfigManager is a config manager
type ConfigManager[T any] struct {
	sources []source.Source
	viper   *viper.Viper
	watcher *watcher.Watcher[T]
	mu      sync.RWMutex
}

// NewManager create a new config manager
func NewManager[T any](providers ...source.Source) *ConfigManager[T] {
	return &ConfigManager[T]{
		sources: providers,
		viper:   viper.New(),
		watcher: watcher.New[T](),
	}
}

// Load config from sources, validate and return config struct
func (cm *ConfigManager[T]) Load() (*T, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// load all sources
	err := cm.loadSource()
	if err != nil {
		return nil, err
	}

	return cm.loadConfig()
}

// loadSource load all config source, merge config to viper
func (cm *ConfigManager[T]) loadSource() error {
	for _, src := range cm.sources {
		// 读取配置
		cfg, err := src.Read()
		if err != nil {
			return fmt.Errorf("%w: %w read from source %s", ce.ErrLoadProviderFailed, err, src.String())
		}

		// 合并配置
		err = cm.viper.Merge(cfg)
		if err != nil {
			return err
		}
	}

	return nil
}

// loadConfig unmarshal config to struct and validate
func (cm *ConfigManager[T]) loadConfig() (*T, error) {
	var cfg T
	err := cm.viper.Unmarshal(&cfg)
	if err != nil {
		return nil, fmt.Errorf("unmarshal config failed. %w", err)
	}

	err = validator.Validate(&cfg)
	if err != nil {
		return nil, fmt.Errorf("validate config failed. %w", err)
	}

	return &cfg, nil
}

// AddSource add a config source to config manager
func (cm *ConfigManager[T]) AddSource(source source.Source) {
	cm.sources = append(cm.sources, source)
}

// LoadAndWatch 加载配置并开始监听变更
func (cm *ConfigManager[T]) LoadAndWatch(onChange watcher.Event[T]) error {
	cfg, err := cm.Load()
	if err != nil {
		return err
	}

	onChange(cfg)

	// 注册回调以便在配置变更时更新
	cm.watcher.OnChange(func(a any) error {
		onChange(a)
	})

	//// 使用反射将新配置复制到提供的 cfg 中
	//		return updateConfig(cfg, newCfg)

	// 开始监听
	return cm.Watch()
}

// updateConfig 使用反射将源配置复制到目标配置
func updateConfig(target, source any) error {
	targetValue := reflect.ValueOf(target)
	sourceValue := reflect.ValueOf(source)

	// 确保两者都是指针类型
	if targetValue.Kind() != reflect.Ptr || sourceValue.Kind() != reflect.Ptr {
		return errors.New("源和目标必须都是指针")
	}

	// 获取指针指向的元素
	targetElem := targetValue.Elem()
	sourceElem := sourceValue.Elem()

	// 复制值
	targetElem.Set(sourceElem)

	return nil
}

// Watch 开始监听所有配置源的变更
func (cm *ConfigManager[T]) Watch() error {
	callback := func(events []watcher.Event) error {
		// 重新加载所有源的配置
		err := cm.loadSource()
		if err != nil {
			return fmt.Errorf("load source from config failed. %w", err)
		}

		// 创建一个新的配置对象用于回调
		for _, callback := range events {
			// 对每个回调创建一个新的配置对象
			var newCfg T

			cm.loadConfig(newCfg)

			err = cm.Unmarshal(newCfg)
			if err != nil {
				return fmt.Errorf("unmarshal config failed. %w", err)
			}

			// 验证结构体中的字段标签
			if err := validateFields(newCfg); err != nil {
				log.Printf("验证新配置失败: %v", err)
				continue
			}

			// 如果配置实现了 Validator 接口，则调用其 Validate 方法
			if validator, ok := newCfg.(validator.Validator); ok {
				if err := validator.Validate(); err != nil {
					log.Printf("配置验证失败: %v", err)
					continue
				}
			}

			// 调用回调
			if err := callback(newCfg); err != nil {
				log.Printf("配置变更回调失败: %v", err)
			}
		}

		return nil
	}

	return cm.watcher.Watch(cm.sources, callback)
}
