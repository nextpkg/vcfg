package validator

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

var vld = validator.New(validator.WithRequiredStructEnabled())

// Validator defines a method to validate the configuration.
type Validator interface {
	Validate() error
}

// Validate validates the value if it implements the Validator interface.
func Validate(v any) error {
	if v == nil {
		return fmt.Errorf("validation target cannot be nil")
	}

	// basic validate
	err := vld.Struct(v)
	if err != nil {
		return fmt.Errorf("struct validation failed: %w", err)
	}

	if val, ok := v.(Validator); ok {
		return val.Validate()
	}

	return nil
}
