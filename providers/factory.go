package providers

import (
	"fmt"
	"path/filepath"
	"reflect"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/cliflagv3"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// ProviderConfig holds a koanf provider with its parser
type ProviderConfig struct {
	Provider koanf.Provider
	Parser   koanf.Parser
}

// ProviderFactory handles the creation of different types of providers
type ProviderFactory struct{}

// NewProviderFactory creates a new provider factory
func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{}
}

// CreateProvider creates a ProviderConfig from various source types
func (f *ProviderFactory) CreateProvider(src any) (ProviderConfig, error) {
	switch s := src.(type) {
	case string:
		// File path - create file provider with appropriate parser
		provider := file.Provider(s)
		parser := f.getParserForFile(s)
		return ProviderConfig{Provider: provider, Parser: parser}, nil
	case koanf.Provider:
		// Direct koanf.Provider - determine parser based on provider type
		parser := f.getParserForProvider(s)
		return ProviderConfig{Provider: s, Parser: parser}, nil
	default:
		return ProviderConfig{}, fmt.Errorf("unsupported source type: %T, expected string (file path) or koanf.Provider", src)
	}
}

// CreateProviders creates multiple ProviderConfig instances from various sources
func (f *ProviderFactory) CreateProviders(sources ...any) ([]ProviderConfig, error) {
	var providers []ProviderConfig

	for _, src := range sources {
		providerConfig, err := f.CreateProvider(src)
		if err != nil {
			return nil, err
		}
		providers = append(providers, providerConfig)
	}

	return providers, nil
}

// getParserForFile returns the appropriate parser based on file extension
func (f *ProviderFactory) getParserForFile(path string) koanf.Parser {
	ext := filepath.Ext(path)
	switch ext {
	case ".yaml", ".yml":
		return yaml.Parser()
	case ".json":
		return json.Parser()
	default:
		return yaml.Parser() // Default to YAML
	}
}

// getParserForProvider determines the appropriate parser for a given provider
func (f *ProviderFactory) getParserForProvider(provider koanf.Provider) koanf.Parser {
	switch provider.(type) {
	case *env.Env:
		// Env provider doesn't need a parser
		return nil
	case *cliflagv3.CliFlag:
		// cliflagv3.CliFlag implements koanf.Provider interface directly
		// and doesn't need a parser
		return nil
	default:
		// Check if it's our custom CliFlagsWrapper by type name
		if reflect.TypeOf(provider).String() == "*providers.CliFlagsWrapper" {
			// CliFlagsWrapper implements koanf.Provider interface directly
			// and doesn't need a parser
			return nil
		}
		// Use YAML parser as default for other providers
		return yaml.Parser()
	}
}
