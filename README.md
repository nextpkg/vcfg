# VCFG - 现代化 Go 配置管理库

[![Go Reference](https://pkg.go.dev/badge/github.com/nextpkg/vcfg.svg)](https://pkg.go.dev/github.com/nextpkg/vcfg)
[![Go Report Card](https://goreportcard.com/badge/github.com/nextpkg/vcfg)](https://goreportcard.com/report/github.com/nextpkg/vcfg)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

VCFG 是一个现代化的 Go 配置管理库，专为云原生应用设计，提供类型安全、高性能和极简的配置解决方案。

## ✨ 核心特性

- **🔒 类型安全**: 基于 Go 泛型的完全类型安全配置管理
- **🚀 零配置**: 一行代码即可启动，无需复杂配置
- **🔌 智能插件**: 自动配置变更检测和热重载插件系统
- **📁 多源融合**: 无缝整合文件、环境变量、CLI 参数等配置源
- **⚡ 高性能**: 原子操作和读写锁优化，支持高并发访问
- **🔄 热重载**: 实时监控配置变更，零停机更新
- **✅ 自动验证**: 内置默认值设置和自定义验证支持
- **🛡️ 错误友好**: 详细的错误类型和调试信息

## 📦 安装

```bash
go get github.com/nextpkg/vcfg
```

## 🚀 快速开始

### 30 秒上手

```go
package main

import (
    "fmt"
    "github.com/nextpkg/vcfg"
)

type Config struct {
    Server struct {
        Host string `yaml:"host" default:"localhost"`
        Port int    `yaml:"port" default:"8080"`
    } `yaml:"server"`
    Database struct {
        Host string `yaml:"host" default:"localhost"`
        Name string `yaml:"name" default:"myapp"`
    } `yaml:"database"`
}

func main() {
    // 一行代码启动配置管理
    cm := vcfg.New[Config]("config.yaml")
    defer cm.Close()
    
    config := cm.Get()
    fmt.Printf("Server: %s:%d\n", config.Server.Host, config.Server.Port)
}
```

### 构建器模式 - 高级配置

```go
func main() {
    cm, err := vcfg.NewBuilder[Config]().
        AddFile("config.yaml").              // 配置文件
        AddEnv("APP").                       // 环境变量 (APP_*)
        WithWatch().                         // 启用热重载
        Build()
    if err != nil {
        log.Fatal(err)
    }
    defer cm.Close()
    
    // 配置会自动热重载
    config := cm.Get()
    // 使用配置...
}
```

## 🔌 智能插件系统

### 全局插件注册（推荐）

在应用启动前全局注册插件，实现完全解耦：

```go
// 在 init() 函数中全局注册
func init() {
    // 注册 Kafka 插件
    kafkaPlugin := &KafkaPlugin{}
    vcfg.RegisterGlobalPlugin(kafkaPlugin, &KafkaConfig{})
    
    // 注册 Redis 插件
    redisPlugin := &RedisPlugin{}
    vcfg.RegisterGlobalPlugin(redisPlugin, &RedisConfig{})
}

func main() {
    // 创建配置管理器 - 全局插件自动生效
    cm := vcfg.New[AppConfig]("config.yaml")
    defer cm.Close()
    
    // 插件会自动检测配置变更并重载
    // 无需手动管理插件生命周期
}
```

### 智能配置变更检测

插件系统支持智能配置变更检测，只有相关配置发生变化时才会触发插件重载：

```go
type KafkaPlugin struct {
    client *kafka.Client
}

func (p *KafkaPlugin) Name() string { return "kafka" }

func (p *KafkaPlugin) Start(config any) error {
    kafkaConfig := config.(*KafkaConfig)
    // 初始化 Kafka 客户端
    p.client = kafka.NewClient(kafkaConfig)
    return p.client.Connect()
}

func (p *KafkaPlugin) Reload(config any) error {
    // 只有 Kafka 配置变更时才会调用
    kafkaConfig := config.(*KafkaConfig)
    return p.client.Reconnect(kafkaConfig)
}

func (p *KafkaPlugin) Stop() error {
    return p.client.Close()
}
```

## 🔧 高级功能

### 多源配置融合

```go
cm, err := vcfg.NewBuilder[Config]().
    AddFile("config.yaml").              // 基础配置
    AddFile("config.local.yaml").        // 本地覆盖
    AddEnv("APP").                       // 环境变量
    AddCliFlags(cmd, ".").               // CLI 参数
    Build()
```

### 环境变量映射

```go
// 支持自动环境变量映射
// APP_SERVER_HOST -> config.server.host
// APP_DATABASE_PORT -> config.database.port
cm := vcfg.NewBuilder[Config]().
    AddFile("config.yaml").
    AddEnv("APP").  // 自动映射 APP_ 前缀的环境变量
    Build()
```

### 配置验证

```go
type Config struct {
    Database DatabaseConfig `yaml:"database"`
}

// 实现自定义验证
func (c *Config) Validate() error {
    if c.Database.Host == "" {
        return fmt.Errorf("database host cannot be empty")
    }
    return nil
}
```

### 默认值设置

```go
type ServerConfig struct {
    Host    string `yaml:"host" default:"localhost"`
    Port    int    `yaml:"port" default:"8080"`
    Timeout int    `yaml:"timeout" default:"30"`
}
```

## 📊 性能特性

- **原子操作**: 使用 `atomic.Value` 实现无锁配置读取
- **读写分离**: 读操作零锁竞争，写操作最小锁粒度
- **内存优化**: 智能配置比较，避免不必要的重载
- **并发安全**: 完全的线程安全设计

## 🛡️ 错误处理

VCFG 提供详细的错误信息和类型：

```go
cm, err := vcfg.NewBuilder[Config]().
    AddFile("nonexistent.yaml").
    Build()

if err != nil {
    // 获取详细错误信息
    fmt.Printf("配置加载失败: %v\n", err)
    // 错误包含文件路径、行号等调试信息
}
```

## 📁 项目示例

查看 [examples](./example/) 目录获取完整示例：

- **[基础使用](./example/basic_demo/)** - 快速上手和基本功能
- **[智能配置](./example/plugin_demo/)** - 智能插件和热重载
- **[CLI 集成](./example/cli_demo/)** - 命令行参数集成
- **[插件开发](./example/plugins_demo/)** - 自定义插件开发
- **[配置监听](./example/watch_demo/)** - 文件变更监听

## 🏆 最佳实践

### 1. 使用全局插件注册
```go
// ✅ 推荐：在 init() 中注册
func init() {
    vcfg.RegisterGlobalPlugin(&MyPlugin{}, &MyConfig{})
}
```

### 2. 合理的配置结构
```go
// ✅ 推荐：清晰的配置结构
type Config struct {
    Server   ServerConfig   `yaml:"server"`
    Database DatabaseConfig `yaml:"database"`
    Plugins  PluginsConfig  `yaml:"plugins"`
}
```

### 3. 实现配置验证
```go
// ✅ 推荐：添加业务验证
func (c *Config) Validate() error {
    if c.Server.Port < 1024 {
        return fmt.Errorf("server port must be >= 1024")
    }
    return nil
}
```

### 4. 及时释放资源
```go
// ✅ 推荐：使用 defer 确保资源清理
func main() {
    cm := vcfg.New[Config]("config.yaml")
    defer cm.Close() // 确保资源清理
}
```

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！请查看 [贡献指南](CONTRIBUTING.md) 了解详情。

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

---

**⭐ 如果这个项目对你有帮助，请给我们一个 Star！**