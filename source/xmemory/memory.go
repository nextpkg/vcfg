package xmemory

import "github.com/spf13/viper"

// MemorySource 表示内存配置源
type MemorySource struct {
	Data map[string]interface{}
}

func (m *MemorySource) Watch() (func() error, <-chan *viper.Viper, error) {
	//TODO implement me
	panic("implement me")
}

// NewMemorySource 创建一个新的内存配置源
func NewMemorySource(data map[string]interface{}) *MemorySource {
	if data == nil {
		data = make(map[string]interface{})
	}
	return &MemorySource{
		Data: data,
	}
}

// Read 实现 Source 接口，从内存读取配置
func (m *MemorySource) Read() (*viper.Viper, error) {
	v := viper.New()
	for key, value := range m.Data {
		v.Set(key, value)
	}
	return v, nil
}

// String 实现 Source 接口，返回源的描述
func (m *MemorySource) String() string {
	return "MemorySource"
}
