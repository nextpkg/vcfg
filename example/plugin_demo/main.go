package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"slices"

	"github.com/nextpkg/vcfg"
	"github.com/nextpkg/vcfg/plugins"
)

// AppConfig 应用配置结构
type AppConfig struct {
	Server   ServerConfig `json:"server"`
	Database DBConfig     `json:"database"`
	Logger   LogConfig    `json:"logger"`
}

type ServerConfig struct {
	Host string `json:"host" default:"localhost"`
	Port int    `json:"port" validate:"min=1,max=65535" default:"8080"`
}

type DBConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
}

type LogConfig struct {
	Level  string `json:"level" default:"info"`
	Format string `json:"format" default:"json"`
}

// SetDefaults 方法不再需要，默认值通过结构体标签自动设置

// Validate 自定义验证
func (c *AppConfig) Validate() error {
	if c.Database.Host == "" {
		return fmt.Errorf("database host cannot be empty")
	}
	return nil
}

func main() {
	// 创建测试配置文件
	testFile := "app_config.json"
	initialContent := `{
	"server": {
		"host": "0.0.0.0",
		"port": 9090
	},
	"database": {
		"host": "localhost",
		"port": 5432,
		"user": "admin",
		"password": "secret",
		"dbname": "myapp"
	},
	"logger": {
		"level": "debug",
		"format": "text"
	}
}`

	if err := os.WriteFile(testFile, []byte(initialContent), 0644); err != nil {
		log.Fatal("Failed to create test config file:", err)
	}
	defer os.Remove(testFile)

	// 使用新的构建器API创建配置管理器
	cm := vcfg.NewBuilder[AppConfig]().
		AddFile(testFile).
		AddEnvWithTransform("APP_", ".", func(s string, v string) (string, any) {
			return strings.ToLower(strings.TrimPrefix(s, "APP_")), v
		}).
		AddPlugin(plugins.NewBackupPlugin[AppConfig]("./backups", 5)).
		AddPlugin(plugins.NewMetricsPlugin[AppConfig]()).
		AddPlugin(createCustomValidationPlugin()).
		WithWatch().
		MustBuild()

	// 确保在程序退出时清理资源
	defer func() {
		if err := cm.Close(); err != nil {
			log.Printf("Error closing config manager: %v", err)
		}
	}()

	fmt.Println("=== 配置管理器插件演示 ===")
	fmt.Printf("已注册插件: %v\n", cm.ListPlugins())

	// 获取当前配置
	config := cm.Get()
	fmt.Printf("\n当前配置:\n")
	fmt.Printf("服务器: %s:%d\n", config.Server.Host, config.Server.Port)
	fmt.Printf("数据库: %s@%s:%d/%s\n", config.Database.User, config.Database.Host, config.Database.Port, config.Database.DBName)
	fmt.Printf("日志: %s (%s)\n", config.Logger.Level, config.Logger.Format)

	// 演示插件功能
	if metricsPlugin, ok := cm.GetPlugin("metrics"); ok {
		if mp, ok := metricsPlugin.(*plugins.MetricsPlugin[AppConfig]); ok {
			stats := mp.GetStats()
			fmt.Printf("\n指标统计: %+v\n", stats)
		}
	}

	// 模拟配置文件变更
	go func() {
		time.Sleep(3 * time.Second)
		fmt.Println("\n=== 模拟配置文件变更 ===")

		updatedContent := `{
	"server": {
		"host": "0.0.0.0",
		"port": 8080
	},
	"database": {
		"host": "db.example.com",
		"port": 5432,
		"user": "admin",
		"password": "newsecret",
		"dbname": "myapp"
	},
	"logger": {
		"level": "info",
		"format": "json"
	}
}`

		if err := os.WriteFile(testFile, []byte(updatedContent), 0644); err != nil {
			log.Printf("Failed to update config file: %v", err)
			return
		}

		// 等待配置重新加载
		time.Sleep(1 * time.Second)

		// 显示更新后的配置
		newConfig := cm.Get()
		fmt.Printf("\n更新后的配置:\n")
		fmt.Printf("服务器: %s:%d\n", newConfig.Server.Host, newConfig.Server.Port)
		fmt.Printf("数据库: %s@%s:%d/%s\n", newConfig.Database.User, newConfig.Database.Host, newConfig.Database.Port, newConfig.Database.DBName)
		fmt.Printf("日志: %s (%s)\n", newConfig.Logger.Level, newConfig.Logger.Format)

		// 显示更新后的指标
		if metricsPlugin, ok := cm.GetPlugin("metrics"); ok {
			if mp, ok := metricsPlugin.(*plugins.MetricsPlugin[AppConfig]); ok {
				stats := mp.GetStats()
				fmt.Printf("\n更新后的指标统计: %+v\n", stats)
			}
		}
	}()

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("\n程序运行中，按 Ctrl+C 退出...")
	<-sigChan
	fmt.Println("\n正在关闭程序...")
}

// createCustomValidationPlugin 创建自定义验证插件
func createCustomValidationPlugin() *plugins.ValidationPlugin[AppConfig] {
	validationPlugin := plugins.NewValidationPlugin[AppConfig]()

	// 添加自定义验证规则
	validationPlugin.AddValidator(func(config *AppConfig) error {
		if config.Server.Port == config.Database.Port {
			return fmt.Errorf("server port and database port cannot be the same")
		}
		return nil
	})

	validationPlugin.AddValidator(func(config *AppConfig) error {
		validLevels := []string{"debug", "info", "warn", "error"}
		if slices.Contains(validLevels, config.Logger.Level) {
			return nil
		}
		return fmt.Errorf("invalid log level: %s, must be one of %v", config.Logger.Level, validLevels)
	})

	return validationPlugin
}
