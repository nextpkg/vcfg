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

func TestProviderFactory_AutoDetection(t *testing.T) {
	factory := NewProviderFactory()

	// Test auto-detection with different provider types
	envProvider := env.ProviderWithValue("TEST_", ".", func(s string, v string) (string, any) {
		return s, v
	})
	fileProvider := file.Provider("nonexistent.json")
	customProvider := NewCustomJSONProvider([]byte(`{"test": true}`))

	configs, err := factory.CreateProviders(envProvider, fileProvider, customProvider)
	require.NoError(t, err)
	require.Len(t, configs, 3)

	// Env provider should have nil parser (self-parsing)
	assert.Equal(t, envProvider, configs[0].Provider)
	assert.Nil(t, configs[0].Parser)

	// File provider should have JSON parser (needs parsing)
	assert.Equal(t, fileProvider, configs[1].Provider)
	assert.IsType(t, json.Parser(), configs[1].Parser)

	// Custom provider should use its required parser
	assert.Equal(t, customProvider, configs[2].Provider)
	assert.IsType(t, json.Parser(), configs[2].Parser)
}

// TestCustomJSONProvider tests the CustomJSONProvider implementation
func TestCustomJSONProvider(t *testing.T) {
	jsonData := []byte(`{"name": "test", "value": 42}`)
	provider := NewCustomJSONProvider(jsonData)

	// Test ReadBytes
	data, err := provider.ReadBytes()
	require.NoError(t, err)
	assert.Equal(t, jsonData, data)

	// Test Read method (should return error)
	_, err = provider.Read()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Read method not implemented")

	// Test RequiredParser
	parser := provider.RequiredParser()
	assert.IsType(t, json.Parser(), parser)
}

// TestCustomYAMLProvider tests the CustomYAMLProvider implementation
func TestCustomYAMLProvider(t *testing.T) {
	yamlData := []byte(`name: test\nvalue: 42`)
	provider := NewCustomYAMLProvider(yamlData)

	// Test ReadBytes
	data, err := provider.ReadBytes()
	require.NoError(t, err)
	assert.Equal(t, yamlData, data)

	// Test Read method (should return error)
	_, err = provider.Read()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Read method not implemented")

	// Test RequiredParser
	parser := provider.RequiredParser()
	assert.IsType(t, yaml.Parser(), parser)
}

// TestCustomProviders_EmptyData tests custom providers with empty data
func TestCustomProviders_EmptyData(t *testing.T) {
	// Test JSON provider with empty data
	jsonProvider := NewCustomJSONProvider([]byte{})
	data, err := jsonProvider.ReadBytes()
	require.NoError(t, err)
	assert.Empty(t, data)

	// Test YAML provider with empty data
	yamlProvider := NewCustomYAMLProvider([]byte{})
	data, err = yamlProvider.ReadBytes()
	require.NoError(t, err)
	assert.Empty(t, data)
}

// TestCustomProviders_NilData tests custom providers with nil data
func TestCustomProviders_NilData(t *testing.T) {
	// Test JSON provider with nil data
	jsonProvider := NewCustomJSONProvider(nil)
	data, err := jsonProvider.ReadBytes()
	require.NoError(t, err)
	assert.Nil(t, data)

	// Test YAML provider with nil data
	yamlProvider := NewCustomYAMLProvider(nil)
	data, err = yamlProvider.ReadBytes()
	require.NoError(t, err)
	assert.Nil(t, data)
}

// TestGetParserForFile tests the getParserForFile function with various file extensions
func TestGetParserForFile(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected interface{}
	}{
		{"JSON file", "config.json", json.Parser()},
		{"YAML file", "config.yaml", yaml.Parser()},
		{"YML file", "config.yml", yaml.Parser()},
		{"Unknown extension", "config.txt", yaml.Parser()}, // defaults to YAML
		{"No extension", "config", yaml.Parser()},          // defaults to YAML
		{"Empty string", "", yaml.Parser()},                // defaults to YAML
		{"Multiple dots", "config.backup.json", json.Parser()},
		{"Case insensitive", "config.JSON", json.Parser()},
		{"Case insensitive YAML", "config.YAML", yaml.Parser()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewProviderFactory()
			result := factory.getParserForFile(tt.filePath)
			assert.IsType(t, tt.expected, result)
		})
	}
}

// TestProviderFactory_UnsupportedFileExtension tests handling of unsupported file extensions
func TestProviderFactory_UnsupportedFileExtension(t *testing.T) {
	factory := NewProviderFactory()

	// Test with unsupported file extension (should default to YAML)
	configs, err := factory.CreateProviders("config.xml", "config.ini")
	require.NoError(t, err)
	require.Len(t, configs, 2)

	// Both should default to YAML parser
	assert.IsType(t, yaml.Parser(), configs[0].Parser)
	assert.IsType(t, yaml.Parser(), configs[1].Parser)
}

// TestProviderFactory_EmptyProviders tests factory with no providers
func TestProviderFactory_EmptyProviders(t *testing.T) {
	factory := NewProviderFactory()

	configs, err := factory.CreateProviders()
	require.NoError(t, err)
	assert.Empty(t, configs)
}
