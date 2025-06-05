// Package defaults provides functionality for setting default values on struct fields
// using struct tags. It supports various data types including primitives, slices,
// nested structs, and pointers with automatic type conversion and recursive processing.
package defaults

import (
	"reflect"
	"strconv"
	"strings"
	"time"
)

// SetDefaults sets default values for struct fields using the "default" struct tag.
// It recursively processes nested structs and handles various data types including
// strings, integers, floats, booleans, slices, and pointers.
//
// The function only sets defaults for fields that have zero values, preserving
// any existing non-zero values.
//
// Supported tag format: `default:"value"`
//
// Examples:
//
//	type Config struct {
//	    Port     int           `default:"8080"`
//	    Host     string        `default:"localhost"`
//	    Timeout  time.Duration `default:"30s"`
//	    Debug    bool          `default:"false"`
//	    Tags     []string      `default:"tag1,tag2,tag3"`
//	}
//
// Parameters:
//   - ptr: A pointer to a struct that should have default values applied
//
// Returns:
//   - error: An error if the operation fails, nil otherwise
func SetDefaults(ptr any) error {
	if ptr == nil {
		return nil
	}

	v := reflect.ValueOf(ptr)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return nil
	}

	v = v.Elem()
	t := v.Type()

	for i := range v.NumField() {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Handle nested structs recursively
		if field.Kind() == reflect.Struct {
			if err := SetDefaults(field.Addr().Interface()); err != nil {
				return err
			}
			continue
		}

		// Get default tag value
		defaultValue, ok := fieldType.Tag.Lookup("default")
		if !ok {
			continue
		}

		// Only set default if field is zero value
		if !field.IsZero() {
			continue
		}

		if err := setFieldValue(field, defaultValue); err != nil {
			return err
		}
	}

	return nil
}

// setFieldValue sets a struct field's value based on its type and the provided string value.
// It handles type conversion for various Go types including primitives, time.Duration,
// slices, nested structs, and pointers.
//
// Parameters:
//   - field: The reflect.Value of the field to set
//   - value: The string representation of the value to set
//
// Returns:
//   - error: An error if type conversion or assignment fails, nil otherwise
func setFieldValue(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			// Handle time.Duration
			duration, err := time.ParseDuration(value)
			if err != nil {
				return err
			}
			field.SetInt(int64(duration))
		} else {
			intVal, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return err
			}
			field.SetInt(intVal)
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(uintVal)

	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(floatVal)

	case reflect.Bool:
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(boolVal)

	case reflect.Slice:
		// Handle string slices (comma-separated values)
		if field.Type().Elem().Kind() == reflect.String {
			if value != "" {
				sliceVal := reflect.MakeSlice(field.Type(), 0, 0)
				for _, item := range splitAndTrim(value, ",") {
					sliceVal = reflect.Append(sliceVal, reflect.ValueOf(item))
				}
				field.Set(sliceVal)
			}
		}

	case reflect.Struct:
		// Recursively handle nested structs
		if field.CanAddr() {
			return SetDefaults(field.Addr().Interface())
		}

	case reflect.Ptr:
		// Handle pointer fields
		if field.IsNil() {
			// Create new instance
			newVal := reflect.New(field.Type().Elem())
			field.Set(newVal)
		}
		// Set the value for the pointed-to element
		if field.Elem().Kind() == reflect.Struct {
			return SetDefaults(field.Interface())
		} else {
			// For non-struct pointers, set the value directly
			return setFieldValue(field.Elem(), value)
		}
	}

	return nil
}

// splitAndTrim splits a string by the specified delimiter and trims whitespace
// from each resulting part. Empty parts after trimming are excluded from the result.
//
// This function is used internally for parsing comma-separated values in default tags
// for slice fields.
//
// Parameters:
//   - s: The string to split
//   - delimiter: The delimiter to split by
//
// Returns:
//   - []string: A slice of trimmed, non-empty parts
func splitAndTrim(s, delimiter string) []string {
	if s == "" {
		return nil
	}
	parts := make([]string, 0)
	for part := range strings.SplitSeq(s, delimiter) {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}
