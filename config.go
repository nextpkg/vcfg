package vcfg

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

// Config 结构体持有配置管理的核心逻辑
type Config struct {
	providers []Provider
	v         *viper.Viper
	// 监听状态跟踪
	watchers []func() error
	// 配置变更通知通道
	changeChan chan struct{}
	// 锁，用于保护配置更新
	mu sync.RWMutex
}

// NewConfig 创建一个新的配置管理器实例
func NewConfig(providers ...Provider) *Config {
	return &Config{
		providers:  providers,
		v:          viper.New(),
		changeChan: make(chan struct{}, 1),
		watchers:   make([]func() error, 0),
	}
}

// Load load config  加载配置到指定的结构体中
func (c *Config) Load(cfg any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// load all providers
	err := c.loadProviders()
	if err != nil {
		return err
	}

	err = c.v.Unmarshal(cfg)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrLoadProviderFailed, err)
	}

	// basic validate
	err = vali.Struct(cfg)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrLoadProviderFailed, err)
	}

	// custom validate
	return c.validate(cfg)
}

// loadProviders load
func (c *Config) loadProviders() error {
	for _, source := range c.providers {
		v, err := source.Read()
		if err != nil {
			return fmt.Errorf("%w: %w read from source %s",
				ErrLoadProviderFailed, err, source.String())
		}

		// 合并配置
		err = c.mergeConfig(c.v, v)
		if err != nil {
			return fmt.Errorf("%w: merge from source %s", err, source.String())
		}
	}

	return nil
}

// mergeConfig 将一个 viper 实例的配置合并到另一个中
func (c *Config) mergeConfig(target, source *viper.Viper) error {
	if source == nil {
		return nil
	}

	err := target.MergeConfigMap(source.AllSettings())
	if err != nil {
		return fmt.Errorf("%w: %w", ErrMergeConfigFailed, err)
	}

	return nil
}

// validateFields 验证结构体中的字段标签
func validateFields(cfg interface{}) error {
	value := reflect.ValueOf(cfg)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	if value.Kind() != reflect.Struct {
		return nil
	}

	typ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := value.Field(i)

		// 处理嵌套结构体
		if fieldValue.Kind() == reflect.Struct {
			if err := validateFields(fieldValue.Addr().Interface()); err != nil {
				return err
			}
		} else if fieldValue.Kind() == reflect.Ptr && fieldValue.Elem().Kind() == reflect.Struct {
			if err := validateFields(fieldValue.Interface()); err != nil {
				return err
			}
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			continue
		}

		// 解析 JSON 标签
		parts := strings.Split(jsonTag, ",")
		for _, part := range parts[1:] { // 跳过第一部分（字段名）
			if strings.HasPrefix(part, "options=") {
				options := strings.TrimPrefix(part, "options=")
				validOptions := strings.Split(options, "|")

				fieldStr := fmt.Sprintf("%v", fieldValue.Interface())
				valid := false
				for _, opt := range validOptions {
					if fieldStr == opt {
						valid = true
						break
					}
				}

				if !valid && fieldStr != "" { // 如果值不为空且不在选项中，则返回错误
					return fmt.Errorf("字段 %s 的值 %s 不在允许的选项中: %s", field.Name, fieldStr, options)
				}
			} else if strings.HasPrefix(part, "default=") {
				// 如果字段为零值，则设置默认值
				if isZeroValue(fieldValue) {
					defaultValue := strings.TrimPrefix(part, "default=")
					setDefaultValue(fieldValue, defaultValue)
				}
			}
		}
	}

	return nil
}

// isZeroValue 检查一个值是否为零值
func isZeroValue(v reflect.Value) bool {
	return reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface())
}

// setDefaultValue 设置默认值
func setDefaultValue(field reflect.Value, defaultValue string) {
	switch field.Kind() {
	case reflect.String:
		field.SetString(defaultValue)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// 可以添加更多类型的处理
		// ...
	}
}

// Get 返回指定键的值
func (c *Config) Get(key string) interface{} {
	return c.v.Get(key)
}

// Set 设置指定键的值
func (c *Config) Set(key string, value interface{}) {
	c.v.Set(key, value)
}

// GetString 返回指定键的字符串值
func (c *Config) GetString(key string) string {
	return c.v.GetString(key)
}

// GetInt 返回指定键的整数值
func (c *Config) GetInt(key string) int {
	return c.v.GetInt(key)
}

// GetBool 返回指定键的布尔值
func (c *Config) GetBool(key string) bool {
	return c.v.GetBool(key)
}

// AddSource 添加一个配置源
func (c *Config) AddSource(source Provider) {
	c.providers = append(c.providers, source)
}
