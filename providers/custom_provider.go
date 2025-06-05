package providers

import (
	"errors"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/v2"
)

// CustomJSONProvider is an example of a provider that implements ParserProvider
type CustomJSONProvider struct {
	data []byte
}

// NewCustomJSONProvider creates a new custom JSON provider
func NewCustomJSONProvider(data []byte) *CustomJSONProvider {
	return &CustomJSONProvider{data: data}
}

// ReadBytes implements koanf.Provider interface
func (p *CustomJSONProvider) ReadBytes() ([]byte, error) {
	return p.data, nil
}

// Read implements koanf.Provider interface
func (p *CustomJSONProvider) Read() (map[string]interface{}, error) {
	return nil, errors.New("Read method not implemented, use ReadBytes instead")
}

// RequiredParser implements ParserProvider interface
func (p *CustomJSONProvider) RequiredParser() koanf.Parser {
	return json.Parser()
}

// CustomYAMLProvider is another example that requires YAML parser
type CustomYAMLProvider struct {
	data []byte
}

// NewCustomYAMLProvider creates a new custom YAML provider
func NewCustomYAMLProvider(data []byte) *CustomYAMLProvider {
	return &CustomYAMLProvider{data: data}
}

// ReadBytes implements koanf.Provider interface
func (p *CustomYAMLProvider) ReadBytes() ([]byte, error) {
	return p.data, nil
}

// Read implements koanf.Provider interface
func (p *CustomYAMLProvider) Read() (map[string]interface{}, error) {
	return nil, errors.New("Read method not implemented, use ReadBytes instead")
}

// RequiredParser implements ParserProvider interface
func (p *CustomYAMLProvider) RequiredParser() koanf.Parser {
	return yaml.Parser()
}
