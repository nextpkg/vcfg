# vcfg

`vcfg` 是一个简单而强大的 Go 配置库，它基于 Viper 构建但借鉴了 go-micro 的配置设计理念。它提供了一种更简洁的方式来处理应用程序配置，同时保持了灵活性和可扩展性。

## 特性

- 使用 `conf.Load(&Cfg{})` 简洁语法加载配置
- 自动调用 `Validate()` 方法（如果存在）
- 多种配置源支持（文件、环境变量、Consul、内存等）
- 自定义配置源扩展能力
- 结构体标签验证和默认值设置（类似 go-zero）
- 完全基于 Viper，没有额外的复杂依赖

## 安装

```bash
go get github.com/nextpkg/vcfg
```

## 快速开始

### 基本用法

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/nextpkg/vcfg"
)

// ConfigManager 定义配置结构体
type ConfigManager struct {
    Server struct {
        Host string `json:"host"`
        Port int    `json:"port"`
        Mode string `json:",options=dev|prod,default=dev"`
    } `json:"server"`
}

// Validate 实现 Validator 接口（可选）
func (c *ConfigManager) Validate() error {
    if c.Server.Port <= 0 || c.Server.Port > 65535 {
        return fmt.Errorf("invalid server port: %d", c.Server.Port)
    }
    return nil
}

func main() {
    // 创建配置管理器，添加配置源
    cfg := vcfg.NewManager(
        vcfg.NewFileSource("config.yaml"),
        vcfg.NewEnvironmentSource("APP"),
    )
    
    // 加载配置到结构体
    appCfg := &ConfigManager{}
    if err := cfg.Load(appCfg); err != nil {
        log.Fatalf("Failed to doLoad config: %v", err)
    }
    
    // 使用配置
    fmt.Printf("Server running at %s:%d in %s mode\n", 
        appCfg.Server.Host, 
        appCfg.Server.Port,
        appCfg.Server.Mode)
}
```

### 配置源

vcfg 支持多种配置源，它们按添加的顺序加载，后加载的配置会覆盖先加载的：

```go
cfg := vcfg.NewManager(
    // 从配置文件加载基本配置
    vcfg.NewFileSource("config.yaml"),
    
    // 从环境变量加载，可以覆盖某些配置
    vcfg.NewEnvironmentSource("APP"),
    
    // 从 Consul 加载动态配置
    vcfg.NewConsulSource("myapp/config", "localhost:8500"),
    
    // 从内存加载配置（通常用于测试或默认值）
    vcfg.NewMemorySource(map[string]interface{}{
        "server.host": "0.0.0.0",
    }),
)
```

您也可以在后续添加更多配置源：

```go
cfg.AddSource(vcfg.NewMemorySource(map[string]interface{}{
    "feature.experimental": true,
}))
```

### 配置验证

vcfg 支持两种配置验证方式：

1. **结构体标签验证**：使用 JSON 标签指定字段的验证规则和默认值

```go
type ConfigManager struct {
    LogLevel string `json:",options=debug|info|warn|error,default=info"`
    Mode     string `json:",options=dev|test|prod"`
    Timeout  int    `json:",default=30"`
}
```

2. **实现 Validator 接口**：更复杂的验证逻辑可以通过实现 `Validate()` 方法

```go
func (c *ConfigManager) Validate() error {
    if c.Server.Host == "" {
        return errors.New("server host cannot be empty")
    }
    return nil
}
```

### 自定义配置源

您可以通过实现 `Source` 接口创建自己的配置源：

```go
type MyCustomSource struct {
    // 自定义字段
}

func (m *MyCustomSource) Read() (*viper.Viper, error) {
    v := viper.New()
    // 从任何地方读取配置（API、数据库等）
    v.Set("custom.key", "value")
    return v, nil
}

func (m *MyCustomSource) String() string {
    return "MyCustomSource"
}

// 使用自定义源
cfg.AddSource(&MyCustomSource{})
```

## 与 Viper 和 go-micro 的对比

| 特性 | vcfg | Viper | go-micro ConfigManager |
|------|------|-------|----------------|
| 使用简便性 | ★★★★★ | ★★★☆☆ | ★★★★☆ |
| 配置源扩展 | ★★★★★ | ★★★☆☆ | ★★★★★ |
| 配置验证 | ★★★★★ | ★★☆☆☆ | ★★★☆☆ |
| 依赖复杂性 | ★★★★★ | ★★★★★ | ★★☆☆☆ |
| 功能完整性 | ★★★★☆ | ★★★★★ | ★★★★★ |

vcfg 结合了 Viper 的可靠性和 go-micro ConfigManager 的简洁设计，是一个轻量级但功能强大的配置管理解决方案。

## 许可证

MIT