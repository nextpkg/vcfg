// Package plugins provides utility functions for plugin management and configuration handling.
// This file contains helper functions for plugin key generation, field path handling,
// configuration type detection, and reflection-based operations.
package plugins

import (
	"fmt"
	"reflect"
	"strings"
)

// getPluginKey generates a composite key for plugin registration.
// The key format is "pluginType:instanceName" for named instances,
// or just "pluginType" for default instances.
// This key is used to uniquely identify plugin instances in the registry.
func getPluginKey(pluginType, instanceName string) string {
	if instanceName == "" {
		return pluginType
	}

	return strings.Join([]string{pluginType, instanceName}, ":")
}

// getFieldPath constructs a hierarchical field path by joining the current path
// with the field name using dot notation. This is used to track nested configuration
// structures during plugin discovery and registration.
func getFieldPath(currentPath, fieldName string) string {
	// Build current field path
	if currentPath != "" && fieldName != "" {
		return strings.Join([]string{currentPath, fieldName}, ".")
	}

	return fieldName
}

// getConfigType extracts the plugin type from a configuration object.
// It first checks if the config has an embedded BaseConfig with a Type field.
// If not found, it derives the type from the struct name by removing common suffixes
// like "plugin", "config", "impl", "service", "cfg", "configuration".
// The result is converted to lowercase for consistency.
func getConfigType[T Config](v T) string {
	// Get plugin type from config
	var pluginType string
	if baseConfig := v.baseConfigEmbedded(); baseConfig != nil {
		pluginType = baseConfig.Type
	}

	if pluginType == "" {
		t := reflect.TypeOf(v)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}

		name := strings.ToLower(t.Name())

		// Remove common suffixes to derive clean type names
		suffixes := []string{"plugin", "config", "impl", "service", "cfg", "configuration"}
		for _, suffix := range suffixes {
			if strings.HasSuffix(name, suffix) {
				return strings.TrimSuffix(name, suffix)
			}
		}

		return name
	}

	return pluginType
}

// toInterface safely extracts an interface{} value from a reflect.Value.
// It returns the address of the value if it's addressable, otherwise returns
// the value itself. This is used during reflection-based configuration processing.
func toInterface(fieldValue reflect.Value) any {
	if fieldValue.CanAddr() {
		return fieldValue.Addr().Interface()
	}
	return fieldValue.Interface()
}

// copyConfig performs a deep copy of configuration values from src to dst using reflection.
// Both src and dst must be of the same Config type. The function handles pointer types
// and validates that both values are valid struct types before copying.
// This is used to clone configuration objects during plugin initialization.
func copyConfig[T Config](src, dst T) error {
	srcValue := reflect.ValueOf(src)
	dstValue := reflect.ValueOf(dst)

	// Handle pointers
	if srcValue.Kind() == reflect.Ptr {
		srcValue = srcValue.Elem()
	}
	if dstValue.Kind() == reflect.Ptr {
		dstValue = dstValue.Elem()
	}

	if !srcValue.IsValid() || !dstValue.IsValid() {
		return fmt.Errorf("invalid source or destination config")
	}

	if srcValue.Type() != dstValue.Type() {
		return fmt.Errorf("config types do not match: %v vs %v", srcValue.Type(), dstValue.Type())
	}

	// Copy field values
	for i := range srcValue.NumField() {
		srcField := srcValue.Field(i)
		dstField := dstValue.Field(i)

		if !dstField.CanSet() {
			continue
		}

		dstField.Set(srcField)
	}

	return nil
}

func ToPtr[T any](t T) *T {
	return &t
}
