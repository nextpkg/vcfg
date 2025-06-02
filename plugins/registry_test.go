package plugins

import (
	"strings"
	"testing"
)

// TestPlugin implements Plugin interface for testing
type TestPlugin struct {
	config TestConfig
}

// TestConfig implements Config interface for testing
type TestConfig struct {
	BaseConfig
	Enabled bool   `yaml:"enabled"`
	Value   string `yaml:"value"`
}

// Name returns the plugin name for identification
func (p *TestPlugin) Name() string {
	return "test-plugin"
}

// Start implements Plugin interface
func (p *TestPlugin) Start(config any) error {
	if testConfig, ok := config.(*TestConfig); ok {
		p.config = *testConfig
	}
	return nil
}

// Reload implements Plugin interface
func (p *TestPlugin) Reload(config any) error {
	if testConfig, ok := config.(*TestConfig); ok {
		p.config = *testConfig
	}
	return nil
}

// Stop implements Plugin interface
func (p *TestPlugin) Stop() error {
	return nil
}

// TestRegisterPluginType tests the generic RegisterPluginType function
func TestRegisterPluginType(t *testing.T) {
	// Clear registry before test
	registry := getGlobalPluginRegistry()
	registry.mu.Lock()
	registry.pluginTypes = make(map[string]*PluginTypeEntry)
	registry.mu.Unlock()

	// Register plugin type using generics
	RegisterPluginType[*TestPlugin, *TestConfig](RegisterOptions{AutoDiscover: true})

	// Verify registration
	types := ListPluginTypes()
	if len(types) != 1 {
		t.Errorf("Expected 1 plugin type, got %d", len(types))
	}

	if types[0] != "test-plugin" {
		t.Errorf("Expected plugin type 'test-plugin', got '%s'", types[0])
	}
}

// TestRegisterPluginTypeWithOptions tests RegisterPluginType with custom options
func TestRegisterPluginTypeWithOptions(t *testing.T) {
	// Clear registry before test
	registry := getGlobalPluginRegistry()
	registry.mu.Lock()
	registry.pluginTypes = make(map[string]*PluginTypeEntry)
	registry.mu.Unlock()

	// Register plugin type with auto-discovery disabled
	RegisterPluginType[*TestPlugin, *TestConfig](RegisterOptions{AutoDiscover: false})

	// Verify registration
	registry.mu.RLock()
	entry, exists := registry.pluginTypes["test-plugin"]
	registry.mu.RUnlock()

	if !exists {
		t.Error("Plugin type not registered")
	}

	if entry.AutoDiscover {
		t.Error("Expected AutoDiscover to be false")
	}
}

// TestRegisterPluginInstance tests plugin instance registration
func TestRegisterPluginInstance(t *testing.T) {
	// Clear registry before test
	registry := getGlobalPluginRegistry()
	registry.mu.Lock()
	registry.plugins = make(map[string]*PluginEntry)
	registry.mu.Unlock()

	// Register plugin instance
	plugin := &TestPlugin{}
	config := &TestConfig{Enabled: true, Value: "test"}

	Register(plugin, config)

	// Verify registration
	plugins := ListGlobalPlugins()
	if len(plugins) != 1 {
		t.Errorf("Expected 1 plugin instance, got %d", len(plugins))
	}

	if plugins[0].PluginType != "test-plugin" {
		t.Errorf("Expected plugin type 'test-plugin', got '%s'", plugins[0].PluginType)
	}
}

// BenchmarkRegisterPluginType benchmarks plugin type registration
func BenchmarkRegisterPluginType(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// Clear registry
		registry := getGlobalPluginRegistry()
		registry.mu.Lock()
		registry.pluginTypes = make(map[string]*PluginTypeEntry)
		registry.mu.Unlock()

		// Register using generics
		RegisterPluginType[*TestPlugin, *TestConfig]()
	}
}

// BadConfig implements Config interface but does NOT embed BaseConfig for testing panic
type BadConfig struct {
	Enabled bool   `yaml:"enabled"`
	Value   string `yaml:"value"`
}

// Name returns the config type name
func (c *BadConfig) Name() string {
	return "bad-config"
}

// BadPlugin implements Plugin interface for testing panic
type BadPlugin struct {
	config BadConfig
}

// Name returns the plugin name for identification
func (p *BadPlugin) Name() string {
	return "bad-config"
}

// Start implements Plugin interface
func (p *BadPlugin) Start(config any) error {
	if badConfig, ok := config.(*BadConfig); ok {
		p.config = *badConfig
	}
	return nil
}

// Reload implements Plugin interface
func (p *BadPlugin) Reload(config any) error {
	if badConfig, ok := config.(*BadConfig); ok {
		p.config = *badConfig
	}
	return nil
}

// Stop implements Plugin interface
func (p *BadPlugin) Stop() error {
	return nil
}

// TestPanicOnMissingBaseConfig tests that panic occurs when BaseConfig is not embedded
func TestPanicOnMissingBaseConfig(t *testing.T) {
	// Clear registry before test
	registry := getGlobalPluginRegistry()
	registry.mu.Lock()
	registry.pluginTypes = make(map[string]*PluginTypeEntry)
	registry.mu.Unlock()

	// This should panic because BadConfig doesn't embed BaseConfig
	defer func() {
		if r := recover(); r != nil {
			// Expected panic, check the message
			if panicMsg, ok := r.(string); ok {
				if !strings.Contains(panicMsg, "must embed plugins.BaseConfig") {
					t.Errorf("Expected panic message to contain 'must embed plugins.BaseConfig', got: %s", panicMsg)
				}
			} else {
				t.Errorf("Expected string panic message, got: %v", r)
			}
		} else {
			t.Error("Expected panic when BaseConfig is not embedded, but no panic occurred")
		}
	}()

	// This should trigger the panic
	RegisterPluginType[*BadPlugin, *BadConfig]()
}

// BenchmarkRegisterInstance benchmarks plugin instance registration
func BenchmarkRegisterInstance(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// Clear registry
		registry := getGlobalPluginRegistry()
		registry.mu.Lock()
		registry.plugins = make(map[string]*PluginEntry)
		registry.mu.Unlock()

		// Register plugin instance
		plugin := &TestPlugin{}
		config := &TestConfig{Enabled: true, Value: "test"}
		Register(plugin, config)
	}
}
