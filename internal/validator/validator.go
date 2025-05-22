package validator

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/nextpkg/vcfg/ce"
)

var vld = validator.New(validator.WithRequiredStructEnabled())

// Validator defines a method to validate the configuration.
type Validator interface {
	Validate() error
}

// Validate validates the value if it implements the Validator interface.
func Validate(v any) error {
	// basic validate
	err := vld.Struct(v)
	if err != nil {
		return fmt.Errorf("%w: %w", ce.ErrLoadProviderFailed, err)
	}

	if val, ok := v.(Validator); ok {
		return val.Validate()
	}

	return nil
}
