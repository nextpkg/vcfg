package plugins

import (
	"fmt"
	"reflect"
	"strings"
)

// key generates a composite key for plugin registration
// Format: pluginType:instanceName
func getPluginKey(pluginType, instanceName string) string {
	if instanceName == "" {
		return pluginType
	}

	return strings.Join([]string{pluginType, instanceName}, ":")
}

// getFieldPath returns the instance ID for a given field path
func getFieldPath(currentPath, fieldName string) string {
	// Build current field path
	if currentPath != "" && fieldName != "" {
		return strings.Join([]string{currentPath, fieldName}, ".")
	}

	return fieldName
}

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

// toInterface extracts config interface from field values
func toInterface(fieldValue reflect.Value) any {
	if fieldValue.CanAddr() {
		return fieldValue.Addr().Interface()
	}
	return fieldValue.Interface()
}

// copyConfig copies configuration values from src to dst using reflection
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
