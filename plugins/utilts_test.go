package plugins

import (
	"reflect"
	"testing"
)

// TestConfig is a test implementation of Config interface
type TestConfig struct {
	BaseConfig
	Name  string `json:"name"`
	Value int    `json:"value"`
}

// DatabasePlugin is a test config with plugin suffix
type DatabasePlugin struct {
	BaseConfig
	Host string `json:"host"`
	Port int    `json:"port"`
}

// CacheService is a test config with service suffix
type CacheService struct {
	BaseConfig
	URL string `json:"url"`
}

func TestGetPluginKey(t *testing.T) {
	tests := []struct {
		name         string
		pluginType   string
		instanceName string
		expected     string
	}{
		{
			name:         "with instance name",
			pluginType:   "database",
			instanceName: "mysql",
			expected:     "database:mysql",
		},
		{
			name:         "without instance name",
			pluginType:   "cache",
			instanceName: "",
			expected:     "cache",
		},
		{
			name:         "empty plugin type with instance",
			pluginType:   "",
			instanceName: "redis",
			expected:     ":redis",
		},
		{
			name:         "both empty",
			pluginType:   "",
			instanceName: "",
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPluginKey(tt.pluginType, tt.instanceName)
			if result != tt.expected {
				t.Errorf("getPluginKey(%q, %q) = %q, want %q", tt.pluginType, tt.instanceName, result, tt.expected)
			}
		})
	}
}

func TestGetFieldPath(t *testing.T) {
	tests := []struct {
		name        string
		currentPath string
		fieldName   string
		expected    string
	}{
		{
			name:        "with current path",
			currentPath: "config.database",
			fieldName:   "host",
			expected:    "config.database.host",
		},
		{
			name:        "without current path",
			currentPath: "",
			fieldName:   "port",
			expected:    "port",
		},
		{
			name:        "empty field name",
			currentPath: "config",
			fieldName:   "",
			expected:    "",
		},
		{
			name:        "both empty",
			currentPath: "",
			fieldName:   "",
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFieldPath(tt.currentPath, tt.fieldName)
			if result != tt.expected {
				t.Errorf("getFieldPath(%q, %q) = %q, want %q", tt.currentPath, tt.fieldName, result, tt.expected)
			}
		})
	}
}

func TestGetConfigType(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "config with explicit type",
			config: &TestConfig{
				BaseConfig: BaseConfig{Type: "custom"},
				Name:       "test",
			},
			expected: "custom",
		},
		{
			name: "config without type - derive from struct name",
			config: &TestConfig{
				Name: "test",
			},
			expected: "test",
		},
		{
			name: "config with plugin suffix",
			config: &DatabasePlugin{
				Host: "localhost",
			},
			expected: "database",
		},
		{
			name: "config with service suffix",
			config: &CacheService{
				URL: "http://localhost",
			},
			expected: "cache",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getConfigType(tt.config)
			if result != tt.expected {
				t.Errorf("getConfigType() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestToInterface(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected any
		canAddr  bool
	}{
		{
			name:     "addressable value",
			value:    "test",
			expected: "test",
			canAddr:  true,
		},
		{
			name:     "integer value",
			value:    42,
			expected: 42,
			canAddr:  true,
		},
		{
			name:     "struct value",
			value:    TestConfig{Name: "test", Value: 123},
			expected: TestConfig{Name: "test", Value: 123},
			canAddr:  true,
		},
		{
			name:     "non-addressable value",
			value:    "direct",
			expected: "direct",
			canAddr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fieldValue reflect.Value
			if tt.canAddr {
				// Create a struct with the test value as a field to make it addressable
				type wrapper struct {
					Field any
				}
				w := wrapper{Field: tt.value}
				fieldValue = reflect.ValueOf(w).Field(0)
			} else {
				// Create non-addressable value
				fieldValue = reflect.ValueOf(tt.value)
			}

			result := toInterface(fieldValue)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("toInterface() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCopyConfig(t *testing.T) {
	tests := []struct {
		name        string
		src         Config
		dst         Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful copy",
			src: &TestConfig{
				BaseConfig: BaseConfig{Type: "test"},
				Name:       "source",
				Value:      100,
			},
			dst:         &TestConfig{},
			expectError: false,
		},
		{
			name: "copy between different types",
			src: &TestConfig{
				Name:  "test",
				Value: 42,
			},
			dst:         &DatabasePlugin{},
			expectError: true,
			errorMsg:    "config types do not match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := copyConfig(tt.src, tt.dst)

			if tt.expectError {
				if err == nil {
					t.Errorf("copyConfig() expected error but got nil")
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("copyConfig() error = %v, want error containing %q", err, tt.errorMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("copyConfig() unexpected error = %v", err)
				return
			}

			// Verify the copy was successful
			if !reflect.DeepEqual(tt.src, tt.dst) {
				t.Errorf("copyConfig() dst = %+v, want %+v", tt.dst, tt.src)
			}
		})
	}
}

func TestCopyConfigInvalidValues(t *testing.T) {
	tests := []struct {
		name     string
		src      any
		dst      any
		errorMsg string
	}{
		{
			name:     "nil source",
			src:      (*TestConfig)(nil),
			dst:      &TestConfig{},
			errorMsg: "invalid source or destination config",
		},
		{
			name:     "nil destination",
			src:      &TestConfig{},
			dst:      (*TestConfig)(nil),
			errorMsg: "invalid source or destination config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := copyConfig(tt.src.(Config), tt.dst.(Config))
			if err == nil {
				t.Errorf("copyConfig() expected error but got nil")
				return
			}
			if !contains(err.Error(), tt.errorMsg) {
				t.Errorf("copyConfig() error = %v, want error containing %q", err, tt.errorMsg)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}()))
}
