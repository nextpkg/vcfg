// Package providers contains example implementations of custom providers
// that demonstrate how to implement the ParserProvider interface.
// These providers show how to create self-describing providers that
// specify their required parser type.
package providers

import (
	"errors"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/v2"
)

// CustomJSONProvider is an example implementation of a provider that implements
// the ParserProvider interface. It demonstrates how custom providers can specify
// their required parser type (JSON in this case) for automatic parser detection.
type CustomJSONProvider struct {
	// data holds the raw JSON data to be parsed
	data []byte
}

// NewCustomJSONProvider creates a new custom JSON provider with the provided data.
// The provider will automatically use the JSON parser when processed by the
// configuration system.
func NewCustomJSONProvider(data []byte) *CustomJSONProvider {
	return &CustomJSONProvider{data: data}
}

// ReadBytes implements the koanf.Provider interface by returning the raw JSON data.
// This method is used by koanf to obtain the configuration data for parsing.
func (p *CustomJSONProvider) ReadBytes() ([]byte, error) {
	return p.data, nil
}

// Read implements the koanf.Provider interface but is not used in this implementation.
// The provider relies on ReadBytes() and the JSON parser for data processing.
func (p *CustomJSONProvider) Read() (map[string]interface{}, error) {
	return nil, errors.New("Read method not implemented, use ReadBytes instead")
}

// RequiredParser implements the ParserProvider interface by returning the JSON parser.
// This allows the configuration system to automatically detect and use the correct
// parser for this provider without manual specification.
func (p *CustomJSONProvider) RequiredParser() koanf.Parser {
	return json.Parser()
}

// CustomYAMLProvider is another example implementation that demonstrates
// how to create a provider that requires the YAML parser. It shows the
// flexibility of the ParserProvider interface for different data formats.
type CustomYAMLProvider struct {
	// data holds the raw YAML data to be parsed
	data []byte
}

// NewCustomYAMLProvider creates a new custom YAML provider with the provided data.
// The provider will automatically use the YAML parser when processed by the
// configuration system.
func NewCustomYAMLProvider(data []byte) *CustomYAMLProvider {
	return &CustomYAMLProvider{data: data}
}

// ReadBytes implements the koanf.Provider interface by returning the raw YAML data.
// This method is used by koanf to obtain the configuration data for parsing.
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
