package plugins

import (
	"reflect"
	"sort"
	"testing"
)

// MockPlugin is a test implementation of Plugin interface
type MockPlugin struct {
	started bool
	config  any
}

func (mp *MockPlugin) Start(config any) error {
	mp.started = true
	mp.config = config
	return nil
}

func (mp *MockPlugin) Reload(config any) error {
	mp.config = config
	return nil
}

func (mp *MockPlugin) Stop() error {
	mp.started = false
	return nil
}

// MockConfig is a test implementation of Config interface
type MockConfig struct {
	BaseConfig
	Value string `json:"value"`
}

func TestRegisterPluginType(t *testing.T) {
	tests := []struct {
		name        string
		pluginType  string
		plugin      *MockPlugin
		config      *MockConfig
		opts        []RegisterOptions
		expectPanic bool
	}{
		{
			name:        "register new plugin type",
			pluginType:  "mock1",
			plugin:      &MockPlugin{},
			config:      &MockConfig{},
			expectPanic: false,
		},
		{
			name:        "register with empty type (derive from config)",
			pluginType:  "",
			plugin:      &MockPlugin{},
			config:      &MockConfig{},
			expectPanic: false,
		},
		{
			name:        "register with auto-discovery disabled",
			pluginType:  "mock-no-auto",
			plugin:      &MockPlugin{},
			config:      &MockConfig{},
			opts:        []RegisterOptions{{AutoDiscover: false}},
			expectPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up registry before each subtest
			registry := getGlobalPluginRegistry()
			registry.mu.Lock()
			registry.pluginTypes = make(map[string]*pluginTypeEntry)
			registry.mu.Unlock()

			if tt.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("RegisterPluginType() expected panic but didn't panic")
					}
				}()
			}

			RegisterPluginType(tt.pluginType, tt.plugin, tt.config, tt.opts...)

			if !tt.expectPanic {
				// Verify registration
				expectedType := tt.pluginType
				if expectedType == "" {
					expectedType = getConfigType(tt.config)
				}

				registry := getGlobalPluginRegistry()
				registry.mu.RLock()
				entry, exists := registry.pluginTypes[expectedType]
				registry.mu.RUnlock()

				if !exists {
					t.Errorf("Plugin type %s was not registered", expectedType)
					return
				}

				if entry.PluginType != expectedType {
					t.Errorf("PluginType = %s, want %s", entry.PluginType, expectedType)
				}

				// Test factory functions
				plugin := entry.PluginFactory()
				if plugin == nil {
					t.Errorf("PluginFactory() returned nil")
				}
				if reflect.TypeOf(plugin) != reflect.TypeOf(tt.plugin) {
					t.Errorf("PluginFactory() type = %T, want %T", plugin, tt.plugin)
				}

				config := entry.ConfigFactory()
				if config == nil {
					t.Errorf("ConfigFactory() returned nil")
				}
				if reflect.TypeOf(config) != reflect.TypeOf(tt.config) {
					t.Errorf("ConfigFactory() type = %T, want %T", config, tt.config)
				}

				// Check auto-discovery setting
				expectedAutoDiscover := true
				if len(tt.opts) > 0 {
					expectedAutoDiscover = tt.opts[0].AutoDiscover
				}
				if entry.AutoDiscover != expectedAutoDiscover {
					t.Errorf("AutoDiscover = %v, want %v", entry.AutoDiscover, expectedAutoDiscover)
				}
			}
		})
	}
}

func TestRegisterPluginTypePanic(t *testing.T) {
	// Clean up registry before test
	registry := getGlobalPluginRegistry()
	registry.mu.Lock()
	registry.pluginTypes = make(map[string]*pluginTypeEntry)
	registry.mu.Unlock()

	// Register a plugin type first
	RegisterPluginType("duplicate", &MockPlugin{}, &MockConfig{})

	// Try to register the same type again - should panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("RegisterPluginType() expected panic for duplicate registration but didn't panic")
		}
	}()

	RegisterPluginType("duplicate", &MockPlugin{}, &MockConfig{})
}

func TestListPluginTypes(t *testing.T) {
	// Clean up registry before test
	registry := getGlobalPluginRegistry()
	registry.mu.Lock()
	registry.pluginTypes = make(map[string]*pluginTypeEntry)
	registry.mu.Unlock()

	// Test empty registry
	types := ListPluginTypes()
	if len(types) != 0 {
		t.Errorf("ListPluginTypes() on empty registry = %v, want empty slice", types)
	}

	// Register some plugin types
	RegisterPluginType("type1", &MockPlugin{}, &MockConfig{})
	RegisterPluginType("type2", &MockPlugin{}, &MockConfig{})
	RegisterPluginType("type3", &MockPlugin{}, &MockConfig{})

	// Test with registered types
	types = ListPluginTypes()
	expected := []string{"type1", "type2", "type3"}
	sort.Strings(types)
	sort.Strings(expected)

	if len(types) != len(expected) {
		t.Errorf("ListPluginTypes() length = %d, want %d", len(types), len(expected))
		return
	}

	for i, typ := range types {
		if typ != expected[i] {
			t.Errorf("ListPluginTypes()[%d] = %s, want %s", i, typ, expected[i])
		}
	}
}

func TestUnregisterPluginType(t *testing.T) {
	// Clean up registry before test
	registry := getGlobalPluginRegistry()
	registry.mu.Lock()
	registry.pluginTypes = make(map[string]*pluginTypeEntry)
	registry.mu.Unlock()

	// Register a plugin type
	RegisterPluginType("test-unregister", &MockPlugin{}, &MockConfig{})

	// Verify it's registered
	types := ListPluginTypes()
	if len(types) != 1 || types[0] != "test-unregister" {
		t.Errorf("Plugin type not registered properly")
		return
	}

	// Unregister it
	UnregisterPluginType("test-unregister")

	// Verify it's unregistered
	types = ListPluginTypes()
	if len(types) != 0 {
		t.Errorf("UnregisterPluginType() failed, types = %v, want empty", types)
	}

	// Test unregistering non-existent type (should not panic)
	UnregisterPluginType("non-existent")
}

func TestClonePluginTypes(t *testing.T) {
	// Clean up registry before test
	registry := getGlobalPluginRegistry()
	registry.mu.Lock()
	registry.pluginTypes = make(map[string]*pluginTypeEntry)
	registry.mu.Unlock()

	// Test empty registry
	cloned := clonePluginTypes()
	if len(cloned) != 0 {
		t.Errorf("clonePluginTypes() on empty registry = %v, want empty map", cloned)
	}

	// Register some plugin types
	RegisterPluginType("clone-test1", &MockPlugin{}, &MockConfig{})
	RegisterPluginType("clone-test2", &MockPlugin{}, &MockConfig{})

	// Clone and verify
	cloned = clonePluginTypes()
	if len(cloned) != 2 {
		t.Errorf("clonePluginTypes() length = %d, want 2", len(cloned))
		return
	}

	if _, exists := cloned["clone-test1"]; !exists {
		t.Errorf("clonePluginTypes() missing clone-test1")
	}
	if _, exists := cloned["clone-test2"]; !exists {
		t.Errorf("clonePluginTypes() missing clone-test2")
	}

	// Verify it's a clone (modifying clone shouldn't affect original)
	delete(cloned, "clone-test1")
	originalTypes := ListPluginTypes()
	if len(originalTypes) != 2 {
		t.Errorf("Modifying cloned map affected original registry")
	}
}

func TestGetGlobalPluginRegistry(t *testing.T) {
	// Test singleton behavior
	registry1 := getGlobalPluginRegistry()
	registry2 := getGlobalPluginRegistry()

	if registry1 != registry2 {
		t.Errorf("getGlobalPluginRegistry() not returning singleton instance")
	}

	if registry1.pluginTypes == nil {
		t.Errorf("getGlobalPluginRegistry() pluginTypes map is nil")
	}
}
