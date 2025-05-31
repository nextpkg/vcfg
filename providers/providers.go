package providers

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// ProviderConfig holds a provider and its parser
type ProviderConfig struct {
	Provider koanf.Provider
	Parser   koanf.Parser
}

// ProviderFactory creates providers from various sources
type ProviderFactory struct{}

// NewProviderFactory creates a new provider factory
func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{}
}

// CreateProviders creates provider configs from various sources
func (f *ProviderFactory) CreateProviders(sources ...any) ([]ProviderConfig, error) {
	var configs []ProviderConfig

	for _, source := range sources {
		switch s := source.(type) {
		case string:
			// File path
			provider := file.Provider(s)
			parser := f.getParserForFile(s)
			configs = append(configs, ProviderConfig{
				Provider: provider,
				Parser:   parser,
			})
		case koanf.Provider:
			// Direct provider
			configs = append(configs, ProviderConfig{
				Provider: s,
				Parser:   nil, // Provider should handle parsing
			})
		default:
			return nil, fmt.Errorf("unsupported source type: %T", source)
		}
	}

	return configs, nil
}

// getParserForFile returns appropriate parser based on file extension
func (f *ProviderFactory) getParserForFile(filePath string) koanf.Parser {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".yaml", ".yml":
		return yaml.Parser()
	case ".json":
		return json.Parser()
	default:
		// Default to YAML
		return yaml.Parser()
	}
}
