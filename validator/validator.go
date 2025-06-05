// Package validator provides configuration validation functionality using the
// go-playground/validator library. It supports both struct tag validation
// and custom validation through the Validator interface.
package validator

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

// vld is the global validator instance configured with required struct validation enabled.
// This ensures that struct fields marked as required are properly validated.
var vld = validator.New(validator.WithRequiredStructEnabled())

// Validator defines an interface for custom validation logic.
// Types implementing this interface can provide their own validation
// rules beyond the standard struct tag validation.
type Validator interface {
	// Validate performs custom validation and returns an error if validation fails
	Validate() error
}

// Validate performs comprehensive validation on the provided value.
// It first runs struct tag validation using the go-playground/validator library,
// then checks if the value implements the Validator interface for custom validation.
//
// The validation process:
// 1. Checks for nil values
// 2. Performs struct tag validation (required, format, etc.)
// 3. Calls custom Validate() method if the type implements Validator interface
//
// Returns an error if any validation step fails.
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
