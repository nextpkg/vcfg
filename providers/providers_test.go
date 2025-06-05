package providers

import (
	"testing"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderFactory_CreateProviders_WithCustomProviders(t *testing.T) {
	factory := NewProviderFactory()

	// Test custom JSON provider
	jsonData := []byte(`{"key": "value"}`)
	jsonProvider := NewCustomJSONProvider(jsonData)

	// Test custom YAML provider
	yamlData := []byte(`key: value`)
	yamlProvider := NewCustomYAMLProvider(yamlData)

	// Create providers
	configs, err := factory.CreateProviders(jsonProvider, yamlProvider)
	require.NoError(t, err)
	require.Len(t, configs, 2)

	// Verify JSON provider config
	assert.Equal(t, jsonProvider, configs[0].Provider)
	assert.IsType(t, json.Parser(), configs[0].Parser)

	// Verify YAML provider config
	assert.Equal(t, yamlProvider, configs[1].Provider)
	assert.IsType(t, yaml.Parser(), configs[1].Parser)
}

func TestProviderFactory_CreateProviders_WithFileProvider(t *testing.T) {
	factory := NewProviderFactory()

	// Test with file provider (doesn't implement ParserProvider)
	fileProvider := file.Provider("test.json")

	configs, err := factory.CreateProviders(fileProvider)
	require.NoError(t, err)
	require.Len(t, configs, 1)

	// Should use default JSON parser for providers that don't implement ParserProvider
	assert.Equal(t, fileProvider, configs[0].Provider)
	assert.IsType(t, json.Parser(), configs[0].Parser)
}

func TestProviderFactory_CreateProviders_WithEnvProvider(t *testing.T) {
	factory := NewProviderFactory()

	// Test with env provider (doesn't need parser)
	envProvider := env.ProviderWithValue("TEST_", ".", func(s string, v string) (string, any) {
		return s, v
	})

	configs, err := factory.CreateProviders(envProvider)
	require.NoError(t, err)
	require.Len(t, configs, 1)

	// Env provider should have nil parser
	assert.Equal(t, envProvider, configs[0].Provider)
	assert.Nil(t, configs[0].Parser)
}

func TestProviderFactory_CreateProviders_WithFilePath(t *testing.T) {
	factory := NewProviderFactory()

	// Test with file paths
	configs, err := factory.CreateProviders("config.json", "config.yaml")
	require.NoError(t, err)
	require.Len(t, configs, 2)

	// JSON file should use JSON parser
	assert.IsType(t, json.Parser(), configs[0].Parser)

	// YAML file should use YAML parser
	assert.IsType(t, yaml.Parser(), configs[1].Parser)
}

func TestProviderFactory_CreateProviders_MixedSources(t *testing.T) {
	factory := NewProviderFactory()

	// Test with mixed sources
	customProvider := NewCustomJSONProvider([]byte(`{"custom": true}`))
	fileProvider := file.Provider("test.json")

	configs, err := factory.CreateProviders(
		"config.yaml",  // file path
		customProvider, // custom provider with ParserProvider
		fileProvider,   // koanf provider without ParserProvider
	)
	require.NoError(t, err)
	require.Len(t, configs, 3)

	// File path should use YAML parser
	assert.IsType(t, yaml.Parser(), configs[0].Parser)

	// Custom provider should use its required parser (JSON)
	assert.Equal(t, customProvider, configs[1].Provider)
	assert.IsType(t, json.Parser(), configs[1].Parser)

	// File provider should use default parser (JSON)
	assert.Equal(t, fileProvider, configs[2].Provider)
	assert.IsType(t, json.Parser(), configs[2].Parser)
}
