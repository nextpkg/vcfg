package xenv

import (
	"fmt"
	"strings"

	"github.com/nextpkg/vcfg/ce"
	"github.com/spf13/viper"
)

// EnvironmentSource 表示环境变量配置源
type EnvironmentSource struct {
	Prefix string
}

// NewEnvironmentSource 创建一个新的环境变量配置源
func NewEnvironmentSource(prefix string) *EnvironmentSource {
	return &EnvironmentSource{
		Prefix: prefix,
	}
}

// Read 实现 Source 接口，从环境变量读取配置
func (e *EnvironmentSource) Read() (*viper.Viper, error) {
	v := viper.New()
	v.SetEnvPrefix(e.Prefix)
	v.AutomaticEnv()
	// 支持嵌套结构
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	return v, nil
}

// Watch 实现 Source 接口，但环境变量不支持监听
func (e *EnvironmentSource) Watch() (func() error, <-chan *viper.Viper, error) {
	return nil, nil, ce.ErrWatchNotSupported
}

// String 实现 Source 接口，返回源的描述
func (e *EnvironmentSource) String() string {
	return fmt.Sprintf("EnvironmentSource(%s)", e.Prefix)
}
