package vcfg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

type BuilderTestConfig struct {
	Name    string `json:"name"`
	Port    int    `json:"port"`
	Enabled bool   `json:"enabled"`
}

func TestNewBuilder(t *testing.T) {
	builder := NewBuilder[BuilderTestConfig]()
	assert.NotNil(t, builder)
	assert.Empty(t, builder.sources)
	assert.Empty(t, builder.plugins)
	assert.False(t, builder.enableWatch)
	assert.False(t, builder.enablePlugin)
}

func TestBuilder_AddFile(t *testing.T) {
	builder := NewBuilder[BuilderTestConfig]()
	testFile := "/path/to/config.json"

	result := builder.AddFile(testFile)
	assert.Equal(t, builder, result) // Should return self for chaining
	assert.Len(t, builder.sources, 1)
	assert.Equal(t, testFile, builder.sources[0])
}

func TestBuilder_AddEnv(t *testing.T) {
	builder := NewBuilder[BuilderTestConfig]()
	prefix := "TEST_"

	result := builder.AddEnv(prefix)
	assert.Equal(t, builder, result) // Should return self for chaining
	assert.Len(t, builder.sources, 1)

	// Verify the provider is of correct type
	_, ok := builder.sources[0].(*env.Env)
	assert.True(t, ok)
}

func TestBuilder_AddProvider(t *testing.T) {
	builder := NewBuilder[BuilderTestConfig]()
	provider := rawbytes.Provider([]byte(`{"name":"test"}`))

	result := builder.AddProvider(provider)
	assert.Equal(t, builder, result) // Should return self for chaining
	assert.Len(t, builder.sources, 1)
	assert.Equal(t, provider, builder.sources[0])
}

func TestBuilder_AddCliFlags(t *testing.T) {
	builder := NewBuilder[BuilderTestConfig]()
	cmd := &cli.Command{
		Name: "test",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "name",
				Value: "default",
			},
		},
	}

	result := builder.AddCliFlags(cmd, ".")
	assert.Equal(t, builder, result) // Should return self for chaining
	assert.Len(t, builder.sources, 1)
}

func TestBuilder_WithWatch(t *testing.T) {
	builder := NewBuilder[BuilderTestConfig]()
	assert.False(t, builder.enableWatch)

	result := builder.WithWatch()
	assert.Equal(t, builder, result) // Should return self for chaining
	assert.True(t, builder.enableWatch)
}

func TestBuilder_WithPlugin(t *testing.T) {
	builder := NewBuilder[BuilderTestConfig]()

	result := builder.WithPlugin()
	assert.Equal(t, builder, result) // Should return self for chaining
	assert.True(t, builder.enablePlugin)
}

func TestBuilder_Build(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(*Builder[BuilderTestConfig])
		expectError bool
	}{
		{
			name: "build with valid config",
			setupFunc: func(b *Builder[BuilderTestConfig]) {
				b.AddProvider(rawbytes.Provider([]byte(`{"name":"test","port":8080}`)))
			},
			expectError: false,
		},
		{
			name: "build with file source",
			setupFunc: func(b *Builder[BuilderTestConfig]) {
				// Create a temporary config file
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "config.json")
				err := os.WriteFile(configFile, []byte(`{"name":"file-test","port":9090}`), 0644)
				require.NoError(t, err)
				b.AddFile(configFile)
			},
			expectError: false,
		},
		{
			name: "build with watch enabled",
			setupFunc: func(b *Builder[BuilderTestConfig]) {
				b.AddProvider(rawbytes.Provider([]byte(`{"name":"watch-test"}`)))
				// Note: rawbytes provider will be handled by ProviderFactory
				b.WithWatch()
			},
			expectError: false,
		},
		{
			name: "build with invalid file",
			setupFunc: func(b *Builder[BuilderTestConfig]) {
				b.AddFile("/nonexistent/config.json")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder[BuilderTestConfig]()
			tt.setupFunc(builder)

			cm, err := builder.Build(t.Context())

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, cm)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cm)

				// Clean up
				cm.Close()
			}
		})
	}
}

func TestBuilder_MustBuild(t *testing.T) {
	t.Run("successful build", func(t *testing.T) {
		builder := NewBuilder[BuilderTestConfig]()
		builder.AddProvider(rawbytes.Provider([]byte(`{"name":"must-test"}`)))
		// Note: rawbytes provider will be handled by ProviderFactory

		assert.NotPanics(t, func() {
			cm := builder.MustBuild()
			assert.NotNil(t, cm)
			cm.Close()
		})
	})

	t.Run("failed build should panic", func(t *testing.T) {
		builder := NewBuilder[BuilderTestConfig]()
		builder.AddFile("/nonexistent/config.json")

		assert.Panics(t, func() {
			builder.MustBuild()
		})
	})
}

func TestBuilder_ChainedCalls(t *testing.T) {
	// Test method chaining
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	err := os.WriteFile(configFile, []byte(`{"name":"chain-test","port":8080}`), 0644)
	require.NoError(t, err)

	cm, err := NewBuilder[BuilderTestConfig]().
		AddFile(configFile).
		AddEnv("TEST_").
		WithWatch().
		Build(t.Context())

	assert.NoError(t, err)
	assert.NotNil(t, cm)

	// Verify config was loaded
	config := cm.Get()
	assert.NotNil(t, config)
	assert.Equal(t, "chain-test", config.Name)
	assert.Equal(t, 8080, config.Port)

	// Clean up
	cm.DisableWatch()
	cm.Close()
}
