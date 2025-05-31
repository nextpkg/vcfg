# VCFG 示例集合

本目录包含了 VCFG 配置管理库的各种使用示例，帮助您快速了解和使用 VCFG 的功能。

## 📁 示例目录

### 🚀 基础示例

#### 1. `get_config/` - 基础配置加载
**功能**: 演示如何加载配置文件和环境变量
```bash
cd get_config
go run main.go
```

**特性**:
- 支持 YAML 配置文件
- 支持环境变量覆盖（`APP_` 前缀）
- 配置源优先级演示

#### 2. `watch_demo/` - 热加载演示
**功能**: 演示配置文件热加载功能
```bash
cd watch_demo
go run main.go
# 在另一个终端修改 config.yaml 文件观察热加载效果
```

**特性**:
- 自动监听配置文件变化
- 实时重新加载配置
- 优雅的配置变更通知

#### 3. `plugin_demo/` - 插件系统演示
**功能**: 演示内置插件的使用
```bash
cd plugin_demo
go run main.go
```

**特性**:
- 内置指标收集插件
- 插件生命周期管理
- 配置变更时插件自动响应

### 🔌 高级示例

#### 4. `simple_plugins_demo/` - 简化插件框架
**功能**: 演示新的简化插件系统
```bash
cd simple_plugins_demo
go run main.go
```

**特性**:
- 极简插件接口
- 自动热加载
- 多种内置插件（备份、指标、验证）
- 框架级别集成

## 🛠️ 快速开始

### 1. 基础使用
```go
// 创建配置管理器
cm, err := vcfg.NewBuilder[YourConfig]().
    AddFile("config.yaml").
    AddEnv("APP").
    Build()
if err != nil {
    log.Fatal(err)
}
defer cm.Close()

// 获取配置
config := cm.Get()
```

### 2. 启用热加载
```go
cm, err := vcfg.NewBuilder[YourConfig]().
    AddFile("config.yaml").
    WithWatch().  // 启用热加载
    Build()
```

### 3. 使用简化插件
```go
// 注册简化插件
backupPlugin := plugins.NewSimpleBackupPlugin("./backups", 5)
cm.RegisterSimplePlugin(backupPlugin, func(config interface{}) interface{} {
    return config
})

// 启动所有插件
ctx := context.Background()
cm.StartAllSimplePlugins(ctx)
```

## 📋 配置文件格式

所有示例都使用 YAML 格式的配置文件：

```yaml
server:
  host: "localhost"
  port: 8080

database:
  host: "localhost"
  port: 5432
  name: "myapp"
```

## 🌍 环境变量覆盖

使用 `APP_` 前缀的环境变量可以覆盖配置文件中的值：

```bash
export APP_SERVER_PORT=9090
export APP_DATABASE_HOST=prod.example.com
go run main.go
```

## 🔧 插件系统

### 内置插件
- **BackupPlugin**: 自动备份配置变更

### 简化插件框架
新的简化插件框架提供：
- 极简的 4 方法接口
- 自动热加载支持
- 类型安全的配置提取
- 框架级别的生命周期管理

## 📖 更多信息

- [简化插件框架文档](../SIMPLE_PLUGIN_FRAMEWORK.md)
- [项目主页](../README.md)

## 🤝 贡献

欢迎提交 Issue 和 Pull Request 来改进示例和文档！