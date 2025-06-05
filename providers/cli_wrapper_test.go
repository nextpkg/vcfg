// Copyright (c) 2024 nextpkg. All rights reserved.
// This file contains unit tests for the CLI provider wrapper.
package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProvider implements koanf.Provider for testing
type mockProvider struct {
	data map[string]any
	err  error
}

func (m *mockProvider) Read() (map[string]any, error) {
	return m.data, m.err
}

func (m *mockProvider) ReadBytes() ([]byte, error) {
	return nil, nil // Not used in CLI wrapper tests
}

// TestNewCliProviderWrapper tests the creation of CLI provider wrapper
func TestNewCliProviderWrapper(t *testing.T) {
	original := &mockProvider{data: map[string]any{"key": "value"}}
	cmdName := "test-cmd"
	delim := "."

	wrapper := NewCliProviderWrapper(original, cmdName, delim)

	assert.NotNil(t, wrapper)
	assert.Equal(t, original, wrapper.original)
	assert.Equal(t, cmdName, wrapper.cmdName)
	assert.Equal(t, delim, wrapper.delim)
}

// TestCliProviderWrapper_Read_WithDelimiter tests reading with delimiter
func TestCliProviderWrapper_Read_WithDelimiter(t *testing.T) {
	tests := []struct {
		name     string
		origData map[string]any
		cmdName  string
		delim    string
		expected map[string]any
	}{
		{
			name: "SimpleMapping",
			origData: map[string]any{
				"test-cmd.config.name": "value1",
				"test-cmd.config.port": "8080",
				"other.key":            "ignored",
			},
			cmdName: "test-cmd",
			delim:   ".",
			expected: map[string]any{
				"config.name": "value1",
				"config.port": "8080",
				"other.key":   "ignored",
			},
		},
		{
			name: "NoMatchingKeys",
			origData: map[string]any{
				"other-cmd.config": "value",
				"random.key":       "ignored",
			},
			cmdName: "test-cmd",
			delim:   ".",
			expected: map[string]any{
				"other-cmd.config": "value",
				"random.key":       "ignored",
			},
		},
		{
			name:     "EmptyData",
			origData: map[string]any{},
			cmdName:  "test-cmd",
			delim:    ".",
			expected: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := &mockProvider{data: tt.origData}
			wrapper := NewCliProviderWrapper(original, tt.cmdName, tt.delim)

			result, err := wrapper.Read()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCliProviderWrapper_Read_EmptyDelimiter tests reading with empty delimiter
func TestCliProviderWrapper_Read_EmptyDelimiter(t *testing.T) {
	tests := []struct {
		name     string
		origData map[string]any
		cmdName  string
		expected map[string]any
	}{
		{
			name: "FlattenNestedStructure",
			origData: map[string]any{
				"c": map[string]any{
					"l": map[string]any{
						"i": map[string]any{
							"-": map[string]any{
								"d": map[string]any{
									"e": map[string]any{
										"m": map[string]any{
											"o": map[string]any{
												"configName": "value1",
											},
										},
									},
								},
							},
						},
					},
				},
				"other": "ignored",
			},
			cmdName: "cli-demo",
			expected: map[string]any{
				"configName": "value1",
			},
		},
		{
			name: "SimpleFlattening",
			origData: map[string]any{
				"a": map[string]any{
					"p": map[string]any{
						"p": map[string]any{
							"name": "test",
						},
					},
				},
			},
			cmdName: "app",
			expected: map[string]any{
				"name": "test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := &mockProvider{data: tt.origData}
			wrapper := NewCliProviderWrapper(original, tt.cmdName, "")

			result, err := wrapper.Read()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCliProviderWrapper_Read_ErrorHandling tests error handling
func TestCliProviderWrapper_Read_ErrorHandling(t *testing.T) {
	originalErr := assert.AnError
	original := &mockProvider{err: originalErr}
	wrapper := NewCliProviderWrapper(original, "test-cmd", ".")

	_, err := wrapper.Read()
	assert.Error(t, err)
	assert.Equal(t, originalErr, err)
}

// TestFlattenMap tests the flattenMap function
func TestFlattenMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		prefix   string
		expected map[string]any
	}{
		{
			name: "SimpleFlattening",
			input: map[string]any{
				"a": map[string]any{
					"b": "value1",
					"c": "value2",
				},
				"d": "value3",
			},
			prefix: "",
			expected: map[string]any{
				"a.b": "value1",
				"a.c": "value2",
				"d":   "value3",
			},
		},
		{
			name: "DeepNesting",
			input: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"level3": "deep_value",
					},
				},
			},
			prefix: "root",
			expected: map[string]any{
				"root.level1.level2.level3": "deep_value",
			},
		},
		{
			name:     "EmptyMap",
			input:    map[string]any{},
			prefix:   "",
			expected: map[string]any{},
		},
		{
			name: "MixedTypes",
			input: map[string]any{
				"string": "value",
				"number": 42,
				"bool":   true,
				"nested": map[string]any{
					"inner": "nested_value",
				},
			},
			prefix: "",
			expected: map[string]any{
				"string":       "value",
				"number":       42,
				"bool":         true,
				"nested.inner": "nested_value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := make(map[string]any)
			flattenMap(tt.input, tt.prefix, result)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCliProviderWrapper_Integration tests integration with mock provider
func TestCliProviderWrapper_Integration(t *testing.T) {
	// Create a mock provider that simulates pflag behavior
	mockProvider := &mockProvider{
		data: map[string]any{
			"test-cmd.config.name": "myapp",
			"test-cmd.config.port": "9090",
			"other.value":          "should_be_ignored",
		},
	}

	// Wrap with CLI provider wrapper
	wrapper := NewCliProviderWrapper(mockProvider, "test-cmd", ".")

	// Read data
	data, err := wrapper.Read()
	require.NoError(t, err)

	// Verify the results
	expected := map[string]any{
		"config.name": "myapp",
		"config.port": "9090",
		"other.value": "should_be_ignored",
	}

	assert.Equal(t, expected, data)
}

// TestCliProviderWrapper_EdgeCases tests various edge cases
func TestCliProviderWrapper_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		origData map[string]any
		cmdName  string
		delim    string
		expected map[string]any
	}{
		{
			name: "ExactCommandNameMatch",
			origData: map[string]any{
				"cmd":     "should_be_ignored",
				"cmd.key": "value",
			},
			cmdName: "cmd",
			delim:   ".",
			expected: map[string]any{
				"cmd": "should_be_ignored",
				"key": "value",
			},
		},
		{
			name: "CommandNameAsSubstring",
			origData: map[string]any{
				"mycmd.config":    "value1",
				"cmd.config":      "value2",
				"cmdother.config": "value3",
			},
			cmdName: "cmd",
			delim:   ".",
			expected: map[string]any{
				"mycmd.config":    "value1",
				"config":          "value2",
				"cmdother.config": "value3",
			},
		},
		{
			name: "EmptyCommandName",
			origData: map[string]any{
				".config": "value",
				"config":  "direct",
			},
			cmdName: "",
			delim:   ".",
			expected: map[string]any{
				"config": "value",
			},
		},
		{
			name: "SpecialCharactersInKeys",
			origData: map[string]any{
				"app.config-name": "value1",
				"app.config_name": "value2",
				"app.config.name": "value3",
			},
			cmdName: "app",
			delim:   ".",
			expected: map[string]any{
				"config-name": "value1",
				"config_name": "value2",
				"config.name": "value3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := &mockProvider{data: tt.origData}
			wrapper := NewCliProviderWrapper(original, tt.cmdName, tt.delim)

			result, err := wrapper.Read()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCliProviderWrapper_ComplexNesting tests complex nested structures with empty delimiter
func TestCliProviderWrapper_ComplexNesting(t *testing.T) {
	origData := map[string]any{
		"m": map[string]any{
			"y": map[string]any{
				"a": map[string]any{
					"p": map[string]any{
						"p": map[string]any{
							"database": map[string]any{
								"host": "localhost",
								"port": 5432,
							},
							"server": map[string]any{
								"port": 8080,
							},
						},
					},
				},
			},
		},
		"other": "ignored",
	}

	original := &mockProvider{data: origData}
	wrapper := NewCliProviderWrapper(original, "myapp", "")

	result, err := wrapper.Read()
	require.NoError(t, err)

	expected := map[string]any{
		"databasehost": "localhost",
		"databaseport": 5432,
		"serverport":   8080,
	}

	assert.Equal(t, expected, result)
}
