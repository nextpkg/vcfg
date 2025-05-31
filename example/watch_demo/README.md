# VCFG 热加载演示

这个示例演示了 VCFG 配置管理库的热加载功能，当配置文件发生变化时，程序会自动检测并重新加载配置。

## 🚀 功能特性

- **自动检测**: 监听 `config.yaml` 文件的变化
- **实时重载**: 配置变化时自动重新加载
- **变化对比**: 清晰显示配置项的变化情况
- **优雅关闭**: 支持 Ctrl+C 优雅退出

## 📋 运行方式

### 方式一：手动测试

1. 启动程序：
```bash
go run main.go
```

2. 在另一个终端中修改配置文件：
```bash
# 修改服务器端口
sed -i 's/port: 8080/port: 9090/' config.yaml

# 修改主机地址
sed -i 's/localhost/127.0.0.1/g' config.yaml

# 修改数据库名称
sed -i 's/watchdemo/newdemo/' config.yaml
```

3. 观察程序输出的配置变化

### 方式二：自动测试脚本

运行提供的测试脚本：
```bash
./test_watch.sh
```

这个脚本会自动演示完整的热加载流程。

## 📊 输出示例

当配置发生变化时，程序会显示类似以下的输出：

```
🔄 Configuration change detected!

📊 Configuration Changes [12:49:25]
├─ Server Host: localhost → 127.0.0.1 ✨
├─ Server Port: 8080 → 9090 ✨
├─ Database Host: localhost → 127.0.0.1 ✨
├─ Database Port: 5432 (unchanged)
└─ Database Name: watchdemo (unchanged)

⏳ Waiting for more changes... (Ctrl+C to exit)
```

## 🔧 技术实现

### 核心组件

1. **ConfigChangeWatcher**: 配置变化监听器
   - 定期检查配置是否发生变化
   - 比较新旧配置的差异
   - 格式化输出变化信息

2. **热加载机制**: 
   - 使用 `WithWatch()` 启用文件监听
   - 底层基于文件系统事件通知
   - 自动重新加载和验证配置

3. **变化检测**:
   - 逐字段比较配置结构体
   - 识别具体变化的配置项
   - 提供直观的变化展示

### 关键代码片段

```go
// 启用热加载
cm, err := vcfg.NewBuilder[AppConfig]().
    AddFile("config.yaml").
    WithWatch(). // 启用热加载
    Build()

// 配置变化检测
func (w *ConfigChangeWatcher) hasConfigChanged(newConfig *AppConfig) bool {
    return w.lastConfig.Server.Host != newConfig.Server.Host ||
           w.lastConfig.Server.Port != newConfig.Server.Port ||
           // ... 其他字段比较
}
```

## 🎯 使用场景

- **开发环境**: 修改配置无需重启应用
- **生产环境**: 动态调整配置参数
- **微服务**: 实时更新服务配置
- **监控系统**: 动态调整监控参数

## 📝 注意事项

1. **配置验证**: 热加载会自动验证新配置的有效性
2. **原子操作**: 配置更新是原子性的，不会出现部分更新
3. **错误处理**: 如果新配置无效，会保持原有配置不变
4. **性能影响**: 监听机制对性能影响极小

## 🔗 相关示例

- [基础配置加载](../basic_demo/) - 学习基本的配置加载
- [插件系统](../plugin_demo/) - 了解插件与热加载的结合
- [简化插件框架](../simple_plugins_demo/) - 体验业务插件架构