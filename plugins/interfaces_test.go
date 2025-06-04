package plugins

import (
	"testing"
)

func TestBaseConfig_baseConfigEmbedded(t *testing.T) {
	tests := []struct {
		name   string
		config *BaseConfig
	}{
		{
			name: "valid base config",
			config: &BaseConfig{
				Type: "test",
			},
		},
		{
			name:   "empty base config",
			config: &BaseConfig{},
		},
		{
			name:   "nil base config",
			config: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config == nil {
				// Test nil case
				var bc *BaseConfig
				result := bc.baseConfigEmbedded()
				if result != nil {
					t.Errorf("baseConfigEmbedded() on nil = %v, want nil", result)
				}
				return
			}

			result := tt.config.baseConfigEmbedded()
			if result != tt.config {
				t.Errorf("baseConfigEmbedded() = %v, want %v", result, tt.config)
			}
		})
	}
}

// TestConfigInterface tests that our test configs implement Config interface
func TestConfigInterface(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name:   "TestConfig implements Config",
			config: &TestConfig{},
		},
		{
			name:   "DatabasePlugin implements Config",
			config: &DatabasePlugin{},
		},
		{
			name:   "CacheService implements Config",
			config: &CacheService{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseConfig := tt.config.baseConfigEmbedded()
			if baseConfig == nil {
				t.Errorf("baseConfigEmbedded() returned nil for %T", tt.config)
			}
		})
	}
}
