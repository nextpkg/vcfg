# Global Plugin Demo (Generic Version)

这个示例展示了如何使用泛型来简化插件系统，**消除了对 `cacheExtractor` 的需求**。

## 核心改进

### 原始版本的问题
在原始版本中，插件需要一个特定的配置提取器：

```go
// 原始版本 - 需要 cacheExtractor
type DemoPlugin struct {
    name string
}

func (p *DemoPlugin) Start(config *CacheConfig) error {
    // 只能访问 CacheConfig
}

// 需要额外的提取器函数
cacheExtractor := func(config *AppConfig) *CacheConfig {
    return &config.Cache
}

plugins.RegisterGlobalPlugin(demoPlugin, cacheExtractor)
```

### 泛型版本的优势

使用泛型后，插件直接使用完整的配置类型：

```go
// 泛型版本 - 无需 cacheExtractor
type DemoPlugin struct {
    name string
}

func (p *DemoPlugin) Start(config AppConfig) error {
    // 可以访问完整的 AppConfig
    fmt.Printf("Cache: %+v\n", config.Cache)
    fmt.Printf("Server: %+v\n", config.Server)
    fmt.Printf("Name: %s\n", config.Name)
}

// 简单的身份提取器
identityExtractor := func(config AppConfig) AppConfig {
    return config
}

cm.RegisterPlugin(demoPlugin, identityExtractor)
```

## 主要优势

1. **🎯 更灵活的配置访问**
   - 插件可以访问配置的任何部分
   - 不再局限于特定的配置子集

2. **🔧 简化的代码结构**
   - 消除了复杂的配置提取器
   - 减少了样板代码

3. **🛡️ 类型安全**
   - 编译时类型检查
   - 泛型提供更好的类型推断

4. **📈 更好的可维护性**
   - 更少的抽象层
   - 更直观的插件实现

## 运行示例

```bash
# 从 vcfg 根目录运行
go run ./example/global_plugin_demo/main.go
```

## 输出示例

```
🚀 VCFG Global Plugin Demo (Generic Version)
============================================
✨ No cacheExtractor needed - plugin works directly with AppConfig!

✅ Configuration loaded: Global Plugin Demo
✅ Plugin registered (no cacheExtractor needed!)
🔌 Plugin 'demo-cache-plugin' started with cache config: MaxSize=2000, TTL=10m0s
   Also has access to server config: localhost:9090
   App name: Global Plugin Demo
✅ All plugins started

🎯 Key Benefits:
   • Plugin can access ANY part of the configuration
   • No need for specific extractors like cacheExtractor
   • Simpler, more flexible plugin design
   • Type-safe with generics

🏁 Demo completed successfully
```

## 配置文件

```yaml
name: "Global Plugin Demo"
server:
  host: "localhost"
  port: 9090
cache:
  max_size: 2000
  ttl: "10m"
```

## 总结

通过使用泛型，我们成功地：
- ✅ 消除了 `cacheExtractor` 的需求
- ✅ 简化了插件实现
- ✅ 提高了代码的灵活性和可维护性
- ✅ 保持了类型安全

这种方法展示了 Go 泛型在简化复杂抽象方面的强大能力。