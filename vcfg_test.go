package vcfg

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/urfave/cli/v3"
)

type TestConfig struct {
	Name    string `json:"name" default:"test-app"`
	Port    int    `json:"port" validate:"min=1,max=65535" default:"8080"`
	Enabled bool   `json:"enabled"`
}

// SetDefaults 方法不再需要，默认值通过结构体标签自动设置

func (c *TestConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	return nil
}

func TestBasicConfigLoading(t *testing.T) {
	// 创建测试配置文件
	testFile := "test_config.json"
	content := `{
	"name": "my-app",
	"port": 9090,
	"enabled": true
}`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}
	defer os.Remove(testFile)

	// 测试基本功能
	t.Log("测试基本配置加载")
	cm := MustInit[TestConfig](testFile)
	defer func() {
		if err := cm.Close(); err != nil {
			t.Errorf("关闭配置管理器时出错: %v", err)
		}
	}()

	config := cm.Get()
	if config == nil {
		t.Fatal("配置为空")
	}

	// 验证配置值
	if config.Name != "my-app" {
		t.Errorf("期望 Name 为 'my-app'，实际为 '%s'", config.Name)
	}
	if config.Port != 9090 {
		t.Errorf("期望 Port 为 9090，实际为 %d", config.Port)
	}
	if !config.Enabled {
		t.Error("期望 Enabled 为 true，实际为 false")
	}

	t.Logf("配置加载成功: %+v", *config)
}

func TestBuilderPattern(t *testing.T) {
	// 创建测试配置文件
	testFile := "test_config_builder.json"
	content := `{
	"name": "builder-app",
	"port": 8080,
	"enabled": false
}`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}
	defer os.Remove(testFile)

	// 测试构建器模式
	t.Log("测试构建器模式")
	cm, err := NewBuilder[TestConfig]().
		AddFile(testFile).
		Build()
	if err != nil {
		t.Fatalf("构建器创建失败: %v", err)
	}
	defer func() {
		if err := cm.Close(); err != nil {
			t.Errorf("关闭配置管理器时出错: %v", err)
		}
	}()

	config := cm.Get()
	if config == nil {
		t.Fatal("构建器配置为空")
	}

	// 验证配置值
	if config.Name != "builder-app" {
		t.Errorf("期望 Name 为 'builder-app'，实际为 '%s'", config.Name)
	}
	if config.Port != 8080 {
		t.Errorf("期望 Port 为 8080，实际为 %d", config.Port)
	}
	if config.Enabled {
		t.Error("期望 Enabled 为 false，实际为 true")
	}

	t.Logf("构建器配置加载成功: %+v", *config)
}

func TestErrorHandling(t *testing.T) {
	// 测试错误处理
	t.Log("测试错误处理")
	_, err := NewBuilder[TestConfig]().
		AddFile("nonexistent.json").
		Build()
	if err == nil {
		t.Error("期望出现错误，但没有错误")
	} else {
		t.Logf("预期的错误: %v", err)
	}
}

func TestConfigDefaults(t *testing.T) {
	// 测试默认值设置
	t.Log("测试配置默认值")
	testFile := "test_config_defaults.json"
	content := `{}`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}
	defer os.Remove(testFile)

	cm := MustInit[TestConfig](testFile)
	defer func() {
		if err := cm.Close(); err != nil {
			t.Errorf("关闭配置管理器时出错: %v", err)
		}
	}()

	config := cm.Get()
	if config == nil {
		t.Fatal("配置为空")
	}

	// 验证默认值
	if config.Name != "test-app" {
		t.Errorf("期望默认 Name 为 'test-app'，实际为 '%s'", config.Name)
	}
	if config.Port != 8080 {
		t.Errorf("期望默认 Port 为 8080，实际为 %d", config.Port)
	}

	t.Logf("默认配置: %+v", *config)
}

func TestMultiProviderMerge(t *testing.T) {
	// 测试多provider配置merge功能
	t.Log("测试多provider配置merge功能")

	// 创建第一个配置文件（基础配置）
	baseFile := "test_base_config.json"
	baseContent := `{
	"name": "base-app",
	"port": 8080,
	"enabled": false
}`
	if err := os.WriteFile(baseFile, []byte(baseContent), 0644); err != nil {
		t.Fatalf("Failed to create base config: %v", err)
	}
	defer os.Remove(baseFile)

	// 创建第二个配置文件（覆盖配置）
	overFile := "test_override_config.json"
	overContent := `{
	"port": 9090,
	"enabled": true
}`
	if err := os.WriteFile(overFile, []byte(overContent), 0644); err != nil {
		t.Fatalf("Failed to create override config: %v", err)
	}
	defer os.Remove(overFile)

	// 使用构建器添加多个配置源
	cm, err := NewBuilder[TestConfig]().
		AddFile(baseFile).
		AddFile(overFile).
		Build()
	if err != nil {
		t.Fatalf("构建多provider配置管理器失败: %v", err)
	}
	defer func() {
		if err := cm.Close(); err != nil {
			t.Errorf("关闭配置管理器时出错: %v", err)
		}
	}()

	config := cm.Get()
	if config == nil {
		t.Fatal("merge后的配置为空")
	}

	// 验证merge结果：name来自base，port和enabled来自override
	if config.Name != "base-app" {
		t.Errorf("期望 Name 为 'base-app'，实际为 '%s'", config.Name)
	}
	if config.Port != 9090 {
		t.Errorf("期望 Port 为 9090（来自override），实际为 %d", config.Port)
	}
	if !config.Enabled {
		t.Error("期望 Enabled 为 true（来自override），实际为 false")
	}

	t.Logf("多provider merge配置: %+v", *config)
}

func TestMultiProviderWatch(t *testing.T) {
	// 测试多provider配置变更后的watch功能
	t.Log("测试多provider配置变更后的watch功能")

	// 创建第一个配置文件
	file1 := "test_watch_file1.json"
	content1 := `{
	"name": "watch-app",
	"port": 8080,
	"enabled": false
}`
	if err := os.WriteFile(file1, []byte(content1), 0644); err != nil {
		t.Fatalf("Failed to create watch config 1: %v", err)
	}
	defer os.Remove(file1)

	// 创建第二个配置文件
	file2 := "test_watch_file2.json"
	content2 := `{
	"port": 9090
}`
	if err := os.WriteFile(file2, []byte(content2), 0644); err != nil {
		t.Fatalf("Failed to create watch config 2: %v", err)
	}
	defer os.Remove(file2)

	// 创建带watch的配置管理器
	cm, err := NewBuilder[TestConfig]().
		AddFile(file1).
		AddFile(file2).
		WithWatch().
		Build()
	if err != nil {
		t.Fatalf("构建watch配置管理器失败: %v", err)
	}
	defer func() {
		if err := cm.Close(); err != nil {
			t.Errorf("关闭配置管理器时出错: %v", err)
		}
	}()

	// 验证初始配置
	initialConfig := cm.Get()
	if initialConfig == nil {
		t.Fatal("初始配置为空")
	}
	if initialConfig.Name != "watch-app" {
		t.Errorf("期望初始 Name 为 'watch-app'，实际为 '%s'", initialConfig.Name)
	}
	if initialConfig.Port != 9090 {
		t.Errorf("期望初始 Port 为 9090，实际为 %d", initialConfig.Port)
	}
	t.Logf("初始配置: %+v", *initialConfig)

	// 设置配置变更监听
	var wg sync.WaitGroup
	var configChanged bool
	var newConfig *TestConfig

	wg.Add(1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		defer wg.Done()
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				current := cm.Get()
				if current != nil && current.Port != initialConfig.Port {
					configChanged = true
					newConfig = current
					return
				}
			}
		}
	}()

	// 等待一小段时间确保watch已经启动
	time.Sleep(200 * time.Millisecond)

	// 修改第二个配置文件（会影响merge结果）
	newContent2 := `{
	"port": 8888,
	"enabled": true
}`
	if err := os.WriteFile(file2, []byte(newContent2), 0644); err != nil {
		t.Fatalf("Failed to update watch config 2: %v", err)
	}

	// 等待配置变更
	wg.Wait()

	if !configChanged {
		t.Error("配置变更未被检测到")
		return
	}

	if newConfig == nil {
		t.Error("新配置为空")
		return
	}

	// 验证变更后的配置merge结果
	if newConfig.Name != "watch-app" {
		t.Errorf("期望变更后 Name 为 'watch-app'，实际为 '%s'", newConfig.Name)
	}
	if newConfig.Port != 8888 {
		t.Errorf("期望变更后 Port 为 8888，实际为 %d", newConfig.Port)
	}
	if !newConfig.Enabled {
		t.Error("期望变更后 Enabled 为 true，实际为 false")
	}

	t.Logf("配置变更后的merge结果: %+v", *newConfig)
	t.Log("多provider watch功能测试通过")
}

func TestCliFlagsProvider(t *testing.T) {
	// Create a temporary config file
	configFile := "test_config.yaml"
	configContent := `name: file-config
port: 3000
enabled: false`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	defer os.Remove(configFile)

	// Create a CLI command with flags
	cmd := &cli.Command{
		Name: "test",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "name"},
			&cli.IntFlag{Name: "port"},
			&cli.BoolFlag{Name: "enabled"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			// Build configuration manager with file and CLI flags
			manager, err := NewBuilder[TestConfig]().
				AddFile(configFile). // File config (lower priority)
				AddCliFlags(c, "."). // CLI flags (higher priority)
				Build()
			if err != nil {
				t.Errorf("Failed to build config manager: %v", err)
				return err
			}
			defer manager.Close()

			// Get merged configuration
			config := manager.Get()

			// Verify CLI flags override file config
			if config.Name != "cli-override" {
				t.Errorf("Expected name to be 'cli-override', got '%s'", config.Name)
			}
			if config.Port != 9090 {
				t.Errorf("Expected port to be 9090, got %d", config.Port)
			}
			if !config.Enabled {
				t.Errorf("Expected enabled to be true, got %t", config.Enabled)
			}

			t.Log("CLI flags successfully override file config")
			return nil
		},
	}

	// Simulate command line arguments with CLI flags
	args := []string{"test", "--name", "cli-override", "--port", "9090", "--enabled"}
	if err := cmd.Run(context.Background(), args); err != nil {
		t.Fatalf("Failed to run CLI command: %v", err)
	}
}
