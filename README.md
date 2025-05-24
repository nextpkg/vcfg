# VCFG

VCFG 是一个配置管理库，提供了一种灵活且高效的方式来处理配置加载、验证和监控。它支持多种配置源，并提供线程安全的配置值访问。

## 功能

- **通用配置类型**：通过类型参数支持通用配置类型。
- **多种来源**：从文件、环境变量和内存等多种来源加载配置。
- **验证**：使用自定义验证器验证配置。
- **监控**：当检测到源中的更改时，自动重新加载配置。

## 安装

要安装 VCFG，请使用以下命令：

```bash
go get github.com/nextpkg/vcfg
```

## 使用方法

### 初始化配置管理器

要初始化配置管理器，请使用 `MustInit` 函数：

```go
import "github.com/nextpkg/vcfg"

func main() {
    cm := vcfg.MustInit(true, vcfg.NewFileSource("config.yaml"))
    config := cm.Get()
    // 使用配置
}
```

### 添加新来源

您可以添加新的来源，例如环境变量或内存：

```go
import "github.com/nextpkg/vcfg/source/env"

func main() {
    envSource := env.NewEnvironmentSource("MYAPP")
    cm := vcfg.MustInit(true, envSource)
    config := cm.Get()
    // 使用配置
}
```

## 许可证

此项目根据 MIT 许可证授权 - 有关详细信息，请参阅 LICENSE 文件。