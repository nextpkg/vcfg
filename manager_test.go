package vcfg

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestConfig struct {
	Name    string `json:"name"`
	Port    int    `json:"port"`
	Enabled bool   `json:"enabled"`
}

func TestConfigManager_Get(t *testing.T) {
	tests := []struct {
		name     string
		config   string
		expected *TestConfig
	}{
		{
			name:   "valid config",
			config: `{"name":"test","port":8080,"enabled":true}`,
			expected: &TestConfig{
				Name:    "test",
				Port:    8080,
				Enabled: true,
			},
		},
		{
			name:   "partial config",
			config: `{"name":"partial"}`,
			expected: &TestConfig{
				Name:    "partial",
				Port:    0,
				Enabled: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := newManager[TestConfig](rawbytes.Provider([]byte(tt.config)))

			cfg, err := cm.load()
			require.NoError(t, err)
			cm.cfg.Store(cfg)

			result := cm.Get()
			require.NotNil(t, result)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigManager_GetNil(t *testing.T) {
	// Test nil manager
	var cm *ConfigManager[TestConfig]
	result := cm.Get()
	assert.Nil(t, result)

	// Test uninitialized config
	cm2 := newManager[TestConfig](rawbytes.Provider([]byte(`{}`)))
	result2 := cm2.Get()
	assert.Nil(t, result2)
}

func TestConfigManager_EnableWatch(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	initialConfig := `{"name":"initial","port":8080,"enabled":true}`

	err := os.WriteFile(configFile, []byte(initialConfig), 0644)
	require.NoError(t, err)

	// Create manager with file provider
	fileProvider := file.Provider(configFile)
	cm := newManager[TestConfig](fileProvider)

	// Load initial config
	cfg, err := cm.load()
	require.NoError(t, err)
	cm.cfg.Store(cfg)

	// Enable watch
	result := cm.EnableWatch()
	assert.Equal(t, cm, result) // Should return self for chaining

	// Verify watchers are set up
	assert.NotEmpty(t, cm.watchers)

	// Test that EnableWatch can be called multiple times safely
	result2 := cm.EnableWatch()
	assert.Equal(t, cm, result2)

	// Clean up
	cm.DisableWatch()
}

func TestConfigManager_DisableWatch(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	initialConfig := `{"name":"initial","port":8080,"enabled":true}`

	err := os.WriteFile(configFile, []byte(initialConfig), 0644)
	require.NoError(t, err)

	// Create manager with file provider
	fileProvider := file.Provider(configFile)
	cm := newManager[TestConfig](fileProvider)

	// Load initial config
	cfg, err := cm.load()
	require.NoError(t, err)
	cm.cfg.Store(cfg)

	// Enable watch first
	cm.EnableWatch()
	assert.NotEmpty(t, cm.watchers)

	// Disable watch
	cm.DisableWatch()
	assert.Empty(t, cm.watchers)

	// Test that DisableWatch can be called multiple times safely
	cm.DisableWatch()
	assert.Empty(t, cm.watchers)
}

func TestConfigManager_DisableWatch_ThreadSafety(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	initialConfig := `{"name":"initial","port":8080,"enabled":true}`

	err := os.WriteFile(configFile, []byte(initialConfig), 0644)
	require.NoError(t, err)

	// Create manager with file provider
	fileProvider := file.Provider(configFile)
	cm := newManager[TestConfig](fileProvider)

	// Load initial config
	cfg, err := cm.load()
	require.NoError(t, err)
	cm.cfg.Store(cfg)

	// Enable watch
	cm.EnableWatch()

	// Test concurrent access to DisableWatch
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cm.DisableWatch()
		}()
	}

	wg.Wait()
	assert.Empty(t, cm.watchers)
}

func TestConfigManager_Close(t *testing.T) {
	cm := newManager[TestConfig](rawbytes.Provider([]byte(`{"name":"test"}`)))

	// Load config
	cfg, err := cm.load()
	require.NoError(t, err)
	cm.cfg.Store(cfg)

	// Test Close method
	err = cm.Close()
	assert.NoError(t, err)
}

func TestConfigManager_CloseWithContext(t *testing.T) {
	cm := newManager[TestConfig](rawbytes.Provider([]byte(`{"name":"test"}`)))

	// Load config
	cfg, err := cm.load()
	require.NoError(t, err)
	cm.cfg.Store(cfg)

	// Test CloseWithContext method
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = cm.CloseWithContext(ctx)
	assert.NoError(t, err)
}

func TestConfigManager_EnableAndStartPlugins(t *testing.T) {
	cm := newManager[TestConfig](rawbytes.Provider([]byte(`{"name":"test","value":42}`)))

	// Load config
	cfg, err := cm.load()
	require.NoError(t, err)
	cm.cfg.Store(cfg)

	// Test EnablePlugins
	err = cm.EnablePlugins()
	assert.NoError(t, err)

	// Test StartPlugins
	err = cm.StartPlugins(context.Background())
	assert.NoError(t, err)
}

func TestConfigManager_MustEnableAndStartPlugins(t *testing.T) {
	cm := newManager[TestConfig](rawbytes.Provider([]byte(`{"name":"test"}`)))

	// Load config
	cfg, err := cm.load()
	require.NoError(t, err)
	cm.cfg.Store(cfg)

	// Test MustEnableAndStartPlugins (should not panic with valid config)
	assert.NotPanics(t, func() {
		cm.MustEnableAndStartPlugins()
	})
}
