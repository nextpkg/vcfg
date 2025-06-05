// Package providers implements a factory pattern for creating koanf providers
// with automatic parser detection and configuration management.
// It supports zero-configuration setup for common provider types while
// maintaining flexibility for custom provider implementations.
package providers

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// ParserProvider is an optional interface that providers can implement
// to explicitly specify their required parser. This takes precedence
// over automatic parser detection.
type ParserProvider interface {
	// RequiredParser returns the parser required by this provider.
	// Return nil if the provider handles parsing internally.
	RequiredParser() koanf.Parser
}

// ProviderConfig represents a complete provider configuration
// containing both the data provider and its associated parser.
// Parser can be nil for providers that handle parsing internally.
type ProviderConfig struct {
	// Provider is the koanf data provider
	Provider koanf.Provider
	// Parser is the associated parser, nil if provider handles parsing internally
	Parser koanf.Parser
}

// ProviderFactory is responsible for creating provider configurations
// from various input sources with automatic parser detection.
type ProviderFactory struct{}

// NewProviderFactory creates a new provider factory
func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{}
}

// CreateProviders creates provider configurations from various input sources.
// Supported source types:
//   - string: treated as file path, automatically detects parser from extension
//   - koanf.Provider: uses zero-config auto-detection for parser requirement
//
// Returns a slice of ProviderConfig with appropriate parsers assigned,
// or an error if any source type is unsupported.
func (f *ProviderFactory) CreateProviders(sources ...any) ([]ProviderConfig, error) {
	var configs []ProviderConfig

	for _, source := range sources {
		switch s := source.(type) {
		case string:
			// Create enhanced file watcher that monitors parent directory
			// to handle atomic file operations properly
			fileWatcher, err := NewFileWatcher(s)
			if err != nil {
				return nil, fmt.Errorf("failed to create file watcher for %s: %w", s, err)
			}
			parser := f.getParserForFile(s)
			configs = append(configs, ProviderConfig{
				Provider: fileWatcher,
				Parser:   parser,
			})
		case koanf.Provider:
			// Direct provider instance - use intelligent auto-detection
			// to determine if parser is needed based on provider type
			parser := f.detectParserRequirement(s)

			configs = append(configs, ProviderConfig{
				Provider: s,
				Parser:   parser,
			})
		default:
			return nil, fmt.Errorf("unsupported source type: %T", source)
		}
	}

	return configs, nil
}

// detectParserRequirement intelligently determines the parser requirement
// for a given provider using type assertion. This method implements a
// zero-configuration approach that works with common koanf provider types.
//
// Detection logic:
//  1. Check if provider implements ParserProvider interface (highest priority)
//  2. Use type assertion for known provider types:
//     - env.Env: returns nil (handles parsing internally)
//     - file.File: returns json.Parser() (needs external parsing)
//  3. Default to json.Parser() for unknown types (safe fallback)
//
// This approach is more robust than error message inspection and provides
// better maintainability and forward compatibility.
func (f *ProviderFactory) detectParserRequirement(provider koanf.Provider) koanf.Parser {
	// Priority 1: Check if provider explicitly implements ParserProvider interface
	// This allows custom providers to override automatic detection
	if pp, ok := provider.(ParserProvider); ok {
		return pp.RequiredParser()
	}

	// Priority 2: Use type assertion for known koanf provider types
	switch provider.(type) {
	case *env.Env:
		// Environment provider reads and parses values internally
		// No external parser needed
		return nil
	case *file.File:
		// File provider only reads raw bytes, requires external parser
		// Default to JSON parser for flexibility
		return json.Parser()
	case *FileWatcher:
		// FileWatcher wraps file provider, also needs external parser
		// Default to JSON parser for flexibility
		return json.Parser()
	default:
		// Priority 3: Safe fallback for unknown provider types
		// Assume external parser is needed to avoid runtime errors
		// Custom providers should implement ParserProvider for explicit control
		return json.Parser()
	}
}

// getParserForFile determines the appropriate parser based on file extension.
// Supports common configuration file formats with sensible defaults.
//
// Supported extensions:
//   - .yaml, .yml: returns yaml.Parser()
//   - .json: returns json.Parser()
//   - others: defaults to yaml.Parser() for maximum compatibility
func (f *ProviderFactory) getParserForFile(filePath string) koanf.Parser {
	// Extract and normalize file extension
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".yaml", ".yml":
		return yaml.Parser()
	case ".json":
		return json.Parser()
	default:
		// Default to YAML parser for unknown extensions
		// YAML is more forgiving and human-readable than JSON
		return yaml.Parser()
	}
}
