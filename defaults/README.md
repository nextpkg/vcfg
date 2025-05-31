# Defaults 包

这个包提供了一种优雅的方式来为 Go 结构体字段设置默认值，使用结构体标签而不是手动的 `if` 语句。

## 🔄 最新更新

## 特性

- **声明式默认值**: 使用 `default` 标签直接在结构体定义中声明默认值
- **类型安全**: 自动解析和转换不同类型的默认值
- **零值检查**: 只为零值字段设置默认值，不覆盖已设置的值
- **嵌套结构体支持**: 递归处理嵌套的结构体
- **指针支持**: 自动创建和设置指针字段的默认值
- **复杂类型支持**: 支持 `time.Duration`、切片、布尔值等

## 支持的类型

- `string`
- `int`, `int8`, `int16`, `int32`, `int64`
- `uint`, `uint8`, `uint16`, `uint32`, `uint64`
- `float32`, `float64`
- `bool`
- `time.Duration`
- `[]string` (逗号分隔的值)
- 嵌套结构体
- 指针类型

## 使用方法

### 基本用法

```go
package main

import (
    "fmt"
    "time"
    "github.com/nextpkg/vcfg/defaults"
)

type Config struct {
    Name     string        `default:"my-app"`
    Port     int           `default:"8080"`
    Debug    bool          `default:"true"`
    Timeout  time.Duration `default:"30s"`
    Tags     []string      `default:"web,api,service"`
    Optional *string       `default:"optional-value"`
}

func main() {
    config := &Config{}
    
    // 设置默认值
    if err := defaults.SetDefaults(config); err != nil {
        panic(err)
    }
    
    fmt.Printf("%+v\n", config)
    // 输出: &{Name:my-app Port:8080 Debug:true Timeout:30s Tags:[web api service] Optional:0xc000010240}
}
```

### 嵌套结构体

```go
type DatabaseConfig struct {
    Host string `default:"localhost"`
    Port int    `default:"5432"`
}

type AppConfig struct {
    Name     string         `default:"my-app"`
    Database DatabaseConfig `default:""`
}

func main() {
    config := &AppConfig{}
    defaults.SetDefaults(config)
    
    // config.Database.Host 将被设置为 "localhost"
    // config.Database.Port 将被设置为 5432
}
```

### 使用方式

```go
type Config struct {
    Name    string `default:"default-app"`
    Port    int    `default:"8080"`
    Enabled bool   `default:"false"`
}
```

## 优势

1. **减少样板代码**: 不需要为每个字段写 `if` 语句
2. **自文档化**: 默认值直接在结构体定义中可见
3. **类型安全**: 自动处理类型转换和验证
4. **一致性**: 所有结构体使用相同的默认值设置机制
5. **可维护性**: 默认值的修改只需要更新标签
6. **可扩展性**: 支持复杂类型和嵌套结构体

## 注意事项

- 只有零值字段会被设置默认值
- 对于指针类型，会自动创建新实例并设置默认值
- 切片类型使用逗号分隔的字符串表示默认值

## 示例

查看 `example/defaults_demo` 目录中的完整示例，了解如何在复杂配置中使用默认值设置。