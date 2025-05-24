package main

import (
	"fmt"

	"github.com/nextpkg/vcfg"
	"github.com/spf13/viper"
)

// AppConfig 是应用程序配置结构体
type AppConfig struct {
	Server   ServerConfig `json:"server"`
	Database DBConfig     `json:"database"`
	Logger   LogConfig    `json:"logger"`
}

// ServerConfig 是服务器配置
type ServerConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port" validate:"oneof=8080 9090"`
	Protocol string `json:",options=http|https,default=http"`
}

// DBConfig 是数据库配置
type DBConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
}

// LogConfig 是日志配置
type LogConfig struct {
	Level  string `json:",options=debug|info|warn|error,default=info"`
	Format string `json:",options=json|text,default=text"`
	Offset string `json:",options=first|last,default=last"`
}

// Validate 实现 Validator 接口
func (c *AppConfig) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("服务器端口无效: %d", c.Server.Port)
	}

	if c.Database.Host == "" {
		return fmt.Errorf("数据库主机不能为空")
	}

	return nil
}

func main() {
	// 创建配置管理器，添加多个配置源
	cfg := vcfg.MustInitFile[AppConfig]("config.yaml")

	appCfg := cfg.Get()

	// 使用配置
	fmt.Printf("服务器配置: %s:%d (%s)\n",
		appCfg.Server.Host,
		appCfg.Server.Port,
		appCfg.Server.Protocol)

	fmt.Printf("数据库配置: %s:%d/%s\n",
		appCfg.Database.Host,
		appCfg.Database.Port,
		appCfg.Database.DBName)

	fmt.Printf("日志配置: 级别=%s, 格式=%s\n",
		appCfg.Logger.Level,
		appCfg.Logger.Format)
}

// 自定义配置源示例
type MyCustomSource struct {
	// 可以添加自定义字段
}

func NewMyCustomSource() *MyCustomSource {
	return &MyCustomSource{}
}

// Read 实现 Provider 接口
func (m *MyCustomSource) Read() (*viper.Viper, error) {
	// 实现自定义的配置读取逻辑
	v := viper.New()

	// 从任何地方获取配置数据
	// 例如: 远程API, 数据库, 特定文件格式...
	v.Set("custom.key", "custom value")

	return v, nil
}

// String 实现 Provider 接口
func (m *MyCustomSource) String() string {
	return "MyCustomSource"
}
