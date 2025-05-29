package vcfg

import (
	"fmt"
	"os"
	"testing"
)

type TestConfig struct {
	Name    string `json:"name"`
	Port    int    `json:"port" validate:"min=1,max=65535"`
	Enabled bool   `json:"enabled"`
}

func (c *TestConfig) SetDefaults() {
	if c.Name == "" {
		c.Name = "test-app"
	}
	if c.Port == 0 {
		c.Port = 8080
	}
}

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
