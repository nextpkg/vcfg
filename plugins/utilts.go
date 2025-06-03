package plugins

import (
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
	if currentPath != "" {
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
