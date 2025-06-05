// Package providers contains custom provider implementations and wrappers
// for the koanf configuration library. This file implements a CLI provider
// wrapper that handles key name mapping and flattening for command-line arguments.
package providers

import (
	"maps"
	"strings"

	"github.com/knadh/koanf/v2"
	"github.com/nextpkg/vcfg/slogs"
)

// flattenMap recursively flattens nested map structures into a flat map
// with dot-separated keys. This is useful for converting hierarchical
// configuration data into a format suitable for command-line processing.
//
// Parameters:
//   - data: The nested map to flatten
//   - prefix: The current key prefix for nested levels
//   - result: The output map to store flattened key-value pairs
func flattenMap(data map[string]any, prefix string, result map[string]any) {
	for key, value := range data {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		if nestedMap, ok := value.(map[string]any); ok {
			// Recursively handle nested maps
			flattenMap(nestedMap, fullKey, result)
		} else {
			// Leaf node, set value directly
			result[fullKey] = value
		}
	}
}

// CliProviderWrapper wraps a CLI provider to handle key name mapping and transformation.
// It processes command-line arguments and converts them into a format compatible
// with the configuration structure, including flattening nested maps and
// applying key transformations.
type CliProviderWrapper struct {
	// original is the underlying CLI provider being wrapped
	original koanf.Provider
	// cmdName is the command name used for logging and identification
	cmdName string
	// delim is the delimiter used for key separation in nested structures
	delim string
}

// NewCliProviderWrapper creates a new CLI provider wrapper with the specified parameters.
//
// Parameters:
//   - original: The underlying koanf.Provider to wrap
//   - cmdName: The command name for logging and identification purposes
//   - delim: The delimiter to use for key separation (typically ".")
//
// Returns a configured CliProviderWrapper ready for use.
func NewCliProviderWrapper(original koanf.Provider, cmdName, delim string) *CliProviderWrapper {
	return &CliProviderWrapper{
		original: original,
		cmdName:  cmdName,
		delim:    delim,
	}
}

// Read implements the koanf.Provider interface by reading data from the wrapped provider,
// processing it through key transformations, and flattening nested structures.
// It handles the conversion of CLI arguments into a standardized configuration format.
func (w *CliProviderWrapper) Read() (map[string]any, error) {
	data, err := w.original.Read()
	if err != nil {
		return nil, err
	}

	// Process key mapping, remove command name prefix
	result := make(map[string]any)
	slogs.Debug("cliProviderWrapper: original data", "data", data)
	slogs.Debug("cliProviderWrapper: cmdName", "cmdName", w.cmdName, "delim", w.delim)

	// If delimiter is empty, special handling needed: flatten nested map structure
	if w.delim == "" {
		slogs.Debug("cliProviderWrapper: empty delimiter, flattening nested structure")

		// Recursively flatten nested map structure
		flattenMap(data, "", result)

		// Remove command name prefix keys, keep only actual config keys
		// Flattened key format: c.l.i.-.d.e.m.o.configName
		cmdPrefixPattern := strings.Join(strings.Split(w.cmdName, ""), ".") + "."

		finalResult := make(map[string]any)
		for key, value := range result {
			if strings.HasPrefix(key, cmdPrefixPattern) {
				// Remove command name prefix to get actual config key name
				actualKey := strings.TrimPrefix(key, cmdPrefixPattern)
				// Remove dot separators to restore original key name
				actualKey = strings.ReplaceAll(actualKey, ".", "")
				slogs.Debug("cliProviderWrapper: mapping prefixed key", "from", key, "to", actualKey)
				finalResult[actualKey] = value
			}
		}

		slogs.Debug("cliProviderWrapper: empty delimiter result", "result", finalResult)
		return finalResult, nil
	}

	// Check if there's a nested map with command name as key
	if cmdData, exists := data[w.cmdName]; exists {
		if cmdMap, ok := cmdData.(map[string]any); ok {
			// Use nested map content directly
			slogs.Debug("cliProviderWrapper: found nested command data", "cmdData", cmdMap)
			maps.Copy(result, cmdMap)
		} else {
			// If not a map, set value directly
			result[w.cmdName] = cmdData
		}
	}

	// Handle other keys (not starting with command name)
	prefix := w.cmdName + w.delim

	// Special handling for empty command name
	if w.cmdName == "" {
		// First pass: handle keys starting with delimiter (higher priority)
		for key, value := range data {
			if strings.HasPrefix(key, w.delim) {
				// Remove leading delimiter
				newKey := strings.TrimPrefix(key, w.delim)
				slogs.Debug("cliProviderWrapper: mapping key with empty cmdName", "from", key, "to", newKey)
				result[newKey] = value
			}
		}
		// Second pass: handle remaining keys (only if not already processed)
		for key, value := range data {
			if !strings.HasPrefix(key, w.delim) {
				// Only add if not already in result
				if _, exists := result[key]; !exists {
					slogs.Debug("cliProviderWrapper: keeping key with empty cmdName", "key", key)
					result[key] = value
				}
			}
		}
	} else {
		// Normal processing for non-empty command name
		for key, value := range data {
			if key == w.cmdName {
				// Skip already processed command key
				continue
			}
			if strings.HasPrefix(key, prefix) {
				// Remove command name prefix
				newKey := strings.TrimPrefix(key, prefix)
				slogs.Debug("cliProviderWrapper: mapping key", "from", key, "to", newKey)
				result[newKey] = value
			} else {
				// Keep original key name
				slogs.Debug("cliProviderWrapper: keeping key", "key", key)
				result[key] = value
			}
		}
	}
	slogs.Debug("cliProviderWrapper: result", "result", result)
	return result, nil
}

// ReadBytes implements the koanf.Provider interface
func (w *CliProviderWrapper) ReadBytes() ([]byte, error) {
	return w.original.ReadBytes()
}
