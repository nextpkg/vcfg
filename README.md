# VCFG - 现代化配置管理库

[![Go Reference](https://pkg.go.dev/badge/github.com/nextpkg/vcfg.svg)](https://pkg.go.dev/github.com/nextpkg/vcfg)
[![Go Report Card](https://goreportcard.com/badge/github.com/nextpkg/vcfg)](https://goreportcard.com/report/github.com/nextpkg/vcfg)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

VCFG 是一个现代化的 Go 配置管理库，提供类型安全、插件化和易于使用的配置解决方案。

## ✨ 核心特性

- **🔒 类型安全**: 使用 Go 泛型提供完全的类型安全
- **🔌 插件系统**: 可扩展的插件架构，支持自定义功能
- **🏗️ 构建器模式**: 流畅的 API 设计，易于配置和使用
- **📁 多源支持**: 支持文件、环境变量等多种配置源
- **🔄 热重载**: 实时监控配置文件变化
- **✅ 数据验证**: 内置验证支持，确保配置正确性
- **🛡️ 错误处理**: 详细的错误类型和上下文信息
- **🚀 高性能**: 优化的并发安全设计

## 📦 安装

```bash
go get github.com/nextpkg/vcfg
```

## 🚀 快速开始

### 基础使用

```go
package main

import (
    "fmt"
    "github.com/nextpkg/vcfg"
)

type AppConfig struct {
    Server struct {
        Host string `json:"host" default:"localhost"`
        Port int    `json:"port" default:"8080"`
    } `json:"server"`
    Database struct {
        Host string `json:"host"`
        Name string `json:"name"`
    } `json:"database"`
}

// 默认值现在通过结构体标签自动设置，无需手动实现 SetDefaults 方法

// 自定义验证
func (c *AppConfig) Validate() error {
    if c.Database.Host == "" {
        return fmt.Errorf("database host cannot be empty")
    }
    return nil
}

func main() {
    // 简单初始化
    cm := vcfg.MustInit[AppConfig]("config.yaml")
    config := cm.Get()
    
    fmt.Printf("Server: %s:%d\n", config.Server.Host, config.Server.Port)
    
    // 记得关闭资源
    defer cm.Close()
}
```

### 构建器模式

```go
func main() {
    cm, err := vcfg.NewBuilder[AppConfig]().
        AddFile("config.yaml").                    // 添加配置文件
        AddEnv("APP_").                           // 添加环境变量
        AddPlugin(plugins.NewBackupPlugin[AppConfig]("./backups", 5)). // 添加备份插件
        AddPlugin(plugins.NewMetricsPlugin[AppConfig]()).              // 添加指标插件
        WithWatch().                              // 启用热重载
        Build()
    
    if err != nil {
        log.Fatal(err)
    }
    defer cm.Close()
    
    config := cm.Get()
    // 使用配置...
}
```

## 🔌 插件系统

### 内置插件

#### 备份插件
自动备份配置变更历史：

```go
backupPlugin := plugins.NewBackupPlugin[AppConfig]("./backups", 10)
cm.RegisterPlugin(backupPlugin)
```

#### 指标插件
收集配置加载和变更统计：

```go
metricsPlugin := plugins.NewMetricsPlugin[AppConfig]()
cm.RegisterPlugin(metricsPlugin)

// 获取统计信息
stats := metricsPlugin.GetStats()
fmt.Printf("配置加载次数: %d\n", stats["load_count"])
```

#### 验证插件
添加自定义验证规则：

```go
validationPlugin := plugins.NewValidationPlugin[AppConfig]()
validationPlugin.AddValidator(func(config *AppConfig) error {
    if config.Server.Port < 1024 {
        return fmt.Errorf("server port should be >= 1024")
    }
    return nil
})
cm.RegisterPlugin(validationPlugin)
```

### 自定义插件

```go
type LoggerPlugin[T any] struct {
    logger *slog.Logger
}

func (p *LoggerPlugin[T]) Name() string {
    return "logger"
}

func (p *LoggerPlugin[T]) Initialize(ctx context.Context, manager *vcfg.ConfigManager[T]) error {
    p.logger.Info("Logger plugin initialized")
    return nil
}

func (p *LoggerPlugin[T]) OnConfigLoaded(ctx context.Context, config *T) error {
    p.logger.Info("Configuration loaded")
    return nil
}

func (p *LoggerPlugin[T]) OnConfigChanged(ctx context.Context, oldConfig, newConfig *T) error {
    p.logger.Info("Configuration changed")
    return nil
}

func (p *LoggerPlugin[T]) Shutdown(ctx context.Context) error {
    p.logger.Info("Logger plugin shutdown")
    return nil
}
```

## 🔧 高级功能

### 环境变量转换

```go
cm := vcfg.NewBuilder[AppConfig]().
    AddEnvWithTransform("APP_", ".", func(key, value string) (string, any) {
        // 转换环境变量键名
        key = strings.ToLower(strings.TrimPrefix(key, "APP_"))
        
        // 转换特定值
        if key == "server.port" {
            if port, err := strconv.Atoi(value); err == nil {
                return key, port
            }
        }
        
        return key, value
    }).
    Build()
```

### 配置热重载

```go
cm := vcfg.NewBuilder[AppConfig]().
    AddFile("config.yaml").
    WithWatch().  // 启用热重载
    Build()

// 配置会自动重载，无需手动干预
for {
    config := cm.Get()
    // 使用最新配置
    time.Sleep(time.Second)
}
```

## 🛡️ 错误处理

VCFG 提供详细的错误类型和上下文信息：

```go
cm, err := vcfg.NewBuilder[AppConfig]().
    AddFile("nonexistent.yaml").
    Build()

if err != nil {
    var configErr *vcfg.ConfigError
    if errors.As(err, &configErr) {
        fmt.Printf("错误类型: %s\n", configErr.Type)
        fmt.Printf("错误源: %s\n", configErr.Source)
        fmt.Printf("错误信息: %s\n", configErr.Message)
        fmt.Printf("底层错误: %v\n", configErr.Unwrap())
    }
}
```

## 📊 性能优化

- **原子操作**: 使用 `atomic.Value` 确保配置读取的高性能
- **读写锁**: 最小化锁竞争，提高并发性能
- **延迟加载**: 只在需要时加载和验证配置
- **资源管理**: 自动清理文件监听器和插件资源

## 🏆 最佳实践

1. **使用构建器模式**: 提供更好的可读性和灵活性
2. **实现 SetDefaults**: 确保配置有合理的默认值
3. **添加自定义验证**: 在 Validate 方法中添加业务逻辑验证
4. **合理使用插件**: 根据需要选择合适的内置插件
5. **及时关闭资源**: 使用 defer cm.Close() 确保资源清理
6. **错误处理**: 使用类型断言获取详细错误信息

## 📝 完整示例

查看 [examples](./example/) 目录获取更多完整示例：

- [基础使用](./example/basic_test/main.go)
- [插件演示](./example/plugin_demo/main.go)
- [配置监听](./example/watch_demo/main.go)
- [获取配置](./example/get_config/main.go)

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。