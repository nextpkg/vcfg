package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/providers/env"
	"github.com/nextpkg/vcfg"
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
	Protocol string
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
	Level  string
	Format string
	Offset string
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

func (c *AppConfig) SetDefaults() {
	if c.Server.Protocol == "" {
		c.Server.Protocol = "https"
	}
}

func main() {
	// Test YAML configuration
	fmt.Println("=== Testing YAML Configuration ===")
	testConfig("config.yaml")

	// Test JSON configuration
	fmt.Println("\n=== Testing JSON Configuration ===")
	testConfig("config.json")
}

func testConfig(configFile string) {
	// Initialize configuration with multiple providers
	// 1. File provider for config file (YAML or JSON)
	// 2. Environment variable provider (with APP_ prefix)
	cm := vcfg.MustInit[AppConfig](
		configFile, // File provider
		env.ProviderWithValue("APP_", ".", func(s string, v string) (string, any) {
			// Convert environment variable names to config keys
			// APP_SERVER_HOST -> server.host			// APP_SERVER_PORT -> server.port
			// APP_DATABASE_HOST -> database.host
			// Remove APP_ prefix first, then convert
			key := strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(s, "APP_"), "_", "."))
			return key, v
		}),
	)

	cfg := cm.Get()

	fmt.Printf("Configuration from %s:\n", configFile)
	fmt.Printf("Server Host: %s\n", cfg.Server.Host)
	fmt.Printf("Server Port: %d\n", cfg.Server.Port)
	fmt.Printf("Server Port: %s\n", cfg.Server.Protocol)
	fmt.Printf("Database Host: %s\n", cfg.Database.Host)
	fmt.Printf("Database Port: %d\n", cfg.Database.Port)
	fmt.Printf("Database Name: %s\n", cfg.Database.DBName)
	fmt.Printf("Logging Level: %s\n", cfg.Logger.Level)
	fmt.Printf("Logging Format: %s\n", cfg.Logger.Format)

	fmt.Println("\nProvider Information:")
	fmt.Printf("1. file.Provider for %s config\n", filepath.Ext(configFile))
	fmt.Println("2. env.Provider for environment variables (APP_ prefix)")
	fmt.Println("Environment variables will override file configuration values.")
}

// 注意：现在 vcfg 直接使用 koanf.Provider 接口
// 如果需要自定义配置源，可以直接实现 koanf.Provider 接口
// 或者使用 koanf 生态系统中现有的 providers
