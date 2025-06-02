# VCFG 智能配置变更检测

这个示例展示了 `vcfg` 的智能配置变更检测功能，能够自动识别配置变更并精确重载相关插件。

## Available Commands

### 1. API Test (`api`)
Tests basic plugin registration and discovery functionality:
```bash
go run . api
```

### 2. Base Config Test (`base`)
Demonstrates BaseConfig usage to reduce boilerplate:
```bash
go run . base
```

### 3. Hot Reload Test (`hotreload`)
Demonstrates real-time configuration watching:
```bash
go run . hotreload
```

### 4. Isolation Test (`isolation`)
Simulates configuration changes to show plugin isolation:
```bash
go run . isolation
```

## Key Features Demonstrated

### 1. BaseConfig Benefits
The framework provides embedded BaseConfig structure to reduce boilerplate:
- **Automatic Name Management**: Config names are automatically set during registration
- **Type Safety**: Full compile-time type checking with generics
- **Reduced Code**: Minimal boilerplate for configuration management

### 2. Automatic Plugin Discovery
The framework automatically discovers plugin configurations in your config structure:
- Traverses nested configuration structures
- Identifies fields that implement the `Config` interface
- Automatically instantiates and registers corresponding plugins
- Supports multiple instances of the same plugin type

### 3. Smart Configuration Management
- **Type Safety**: Full compile-time type checking
- **Auto-Discovery**: No manual plugin registration needed
- **Multi-Instance Support**: Multiple instances of the same plugin type
- **Path-Based Naming**: Instance names derived from configuration paths

### 4. Hot Reload Capabilities
- **Real-time Watching**: Monitors configuration file changes
- **Selective Reloading**: Only affected plugins are reloaded
- **Change Detection**: Uses deep comparison to detect actual changes
- **Graceful Handling**: Proper error handling and logging

## Plugin Development Best Practices

### 1. Using BaseConfig (Recommended)
```go
// Plugin implementation - implement all required methods
type KafkaPlugin struct {
    config KafkaConfig
}

// Name must be explicitly implemented
func (k *KafkaPlugin) Name() string {
    return "kafka"
}

// Config implementation with BaseConfig
type KafkaConfig struct {
    plugins.BaseConfig          // Embed for automatic functionality
    BootstrapServers string     `yaml:"bootstrap_servers"`
    Topic           string      `yaml:"topic"`
}

// Implement all plugin methods
func (k *KafkaPlugin) Start(config any) error {
    kafkaConfig := config.(*KafkaConfig)
    k.config = *kafkaConfig
    // Your implementation here
    return nil
}

func (k *KafkaPlugin) Reload(config any) error {
    return k.Start(config)
}

func (k *KafkaPlugin) Stop() error {
    // Your cleanup code here
    return nil
}
```

### 2. Configuration Structure
```go
type Config struct {
    Kafka  KafkaConfig  `yaml:"kafka"`
    Kafka1 KafkaConfig  `yaml:"kafka1"`
    Redis  RedisConfig  `yaml:"redis"`
    Client ClientConfig `yaml:"client"`
}

type ClientConfig struct {
    Kafka KafkaConfig `yaml:"kafka"`
}
```

### 3. Simple Registration
```go
// No manual name management needed!
plugins.RegisterPluginType[*KafkaPlugin, *KafkaConfig]()
plugins.RegisterPluginType[*RedisPlugin, *RedisConfig]()
```

### 4. Legacy Plugin Implementation (Not Recommended)
```go
type OldKafkaPlugin struct {
    name string
}

func (k *OldKafkaPlugin) Name() string { return k.name }
func (k *OldKafkaPlugin) Start(config any) error { /* implementation */ }
func (k *OldKafkaPlugin) Reload(config any) error { /* implementation */ }
func (k *OldKafkaPlugin) Stop() error { /* implementation */ }
```

## 功能特性

### 🎯 智能检测
- **自动识别**：自动检测实现了 `plugins.Config` 接口的配置字段
- **精确重载**：只重载配置发生变更的插件，未变更的插件不受影响
- **零配置**：无需手动编写监听器，基于接口自动工作
- **高性能**：使用反射遍历配置字段，避免轮询开销

### 🔧 工作原理

1. **接口检测**：使用反射遍历配置结构体的所有字段
2. **配置比较**：对实现了 `plugins.Config` 接口的字段进行 `reflect.DeepEqual` 比较
3. **智能重载**：只对配置发生变更的插件调用 `Reload` 方法
4. **多级查找**：优先从注册的插件中查找，然后从全局注册表中查找


## 配置结构

```go
type AppConfig struct {
    Server   ServerConfig   `yaml:"server"`    // 普通配置，不会触发插件重载
    Database DatabaseConfig `yaml:"database"`  // 普通配置，不会触发插件重载
    Kafka    KafkaConfig    `yaml:"kafka"`     // 实现Config接口，会触发kafka插件重载
    Redis    RedisConfig    `yaml:"redis"`     // 实现Config接口，会触发redis插件重载
}
```

## 关键接口实现

### Config 接口
```go
// KafkaConfig 实现 plugins.Config 接口
type KafkaConfig struct {
    Brokers []string `yaml:"brokers"`
    Topic   string   `yaml:"topic"`
    GroupID string   `yaml:"group_id"`
}

func (k KafkaConfig) Name() string {
    return "kafka"
}
```

### Plugin 接口
```go
type KafkaPlugin struct {
    config KafkaConfig
}

func (p *KafkaPlugin) Name() string { return "kafka" }
func (p *KafkaPlugin) Start(config any) error { /* ... */ }
func (p *KafkaPlugin) Reload(config any) error { /* ... */ }
func (p *KafkaPlugin) Stop() error { /* ... */ }
```

## 运行示例

### 1. 启动程序
```bash
cd plugin_demo
go run main.go
```

### 2. 测试智能重载

#### 场景1：只修改 Kafka 配置
修改 `config.yaml` 中的 kafka 部分：
```yaml
kafka:
  brokers:
    - "localhost:9092"
    - "localhost:9093"  # 添加新的 broker
  topic: "updated-topic"  # 修改 topic
  group_id: "my-group"
```

**预期结果**：只有 Kafka 插件会被重载，Redis 插件不受影响。

#### 场景2：只修改 Redis 配置
修改 `config.yaml` 中的 redis 部分：
```yaml
redis:
  host: "redis.example.com"  # 修改主机
  port: 6380                  # 修改端口
  password: "secret"
  db: 1
```

**预期结果**：只有 Redis 插件会被重载，Kafka 插件不受影响。

#### 场景3：修改普通配置
修改 `config.yaml` 中的 server 或 database 部分：
```yaml
server:
  host: "0.0.0.0"  # 修改服务器配置
  port: 9090
```

**预期结果**：没有插件会被重载，因为 ServerConfig 没有实现 `plugins.Config` 接口。

#### 场景4：同时修改多个插件配置
同时修改 kafka 和 redis 配置：

**预期结果**：Kafka 和 Redis 插件都会被重载。

## 日志输出示例

### 启动时
```
time=2024-01-15T10:00:00.000Z level=INFO msg="Registered global plugins" plugins=[kafka redis]
time=2024-01-15T10:00:00.001Z level=INFO msg="Kafka plugin started" brokers=[localhost:9092] topic=my-topic
time=2024-01-15T10:00:00.002Z level=INFO msg="Redis plugin started" host=localhost port=6379
time=2024-01-15T10:00:00.003Z level=INFO msg="All plugins started successfully"
```

### 智能重载时
```
time=2024-01-15T10:01:00.000Z level=INFO msg="Configuration change detected" event=...
time=2024-01-15T10:01:00.001Z level=INFO msg="Smart config change detected" field=Kafka plugin=kafka
time=2024-01-15T10:01:00.002Z level=INFO msg="Smart global plugin reloaded" plugin=kafka
time=2024-01-15T10:01:00.003Z level=INFO msg="Configuration reloaded successfully"
```

## 实现细节

### HandleSmartConfigChange 方法
- 使用反射遍历配置结构体字段
- 检查字段是否实现 `plugins.Config` 接口
- 使用 `reflect.DeepEqual` 比较配置变更
- 优先从已注册插件中查找，然后从全局注册表查找
- 异步执行插件重载以避免阻塞

### 线程安全
- 使用读写锁保护插件管理器
- 异步执行插件重载操作
- 原子操作更新配置

## 注意事项

1. **接口实现**：配置结构体必须实现 `plugins.Config` 接口
2. **插件注册**：插件必须通过 `RegisterGlobalPlugin` 注册
3. **配置类型**：`Reload` 方法接收的是配置的指针类型
4. **错误处理**：插件重载失败不会影响其他插件

## 扩展使用

要为新的配置添加智能重载支持，只需：

1. 让配置结构体实现 `plugins.Config` 接口
2. 实现对应的插件
3. 注册插件到全局注册表

无需修改任何监听器代码，系统会自动检测和处理配置变更。