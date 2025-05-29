package defaults

import (
	"testing"
	"time"
)

type TestConfig struct {
	Name     string        `default:"test-app"`
	Port     int           `default:"8080"`
	Enabled  bool          `default:"true"`
	Timeout  time.Duration `default:"30s"`
	Rate     float64       `default:"1.5"`
	Tags     []string      `default:"tag1,tag2,tag3"`
	Optional *string       `default:"optional-value"`
}

type NestedConfig struct {
	Database TestConfig `default:""`
	Cache    TestConfig `default:""`
}

func TestSetDefaults(t *testing.T) {
	config := &TestConfig{}
	err := SetDefaults(config)
	if err != nil {
		t.Fatalf("SetDefaults failed: %v", err)
	}

	// Test string default
	if config.Name != "test-app" {
		t.Errorf("Expected Name to be 'test-app', got '%s'", config.Name)
	}

	// Test int default
	if config.Port != 8080 {
		t.Errorf("Expected Port to be 8080, got %d", config.Port)
	}

	// Test bool default
	if !config.Enabled {
		t.Errorf("Expected Enabled to be true, got %v", config.Enabled)
	}

	// Test duration default
	expectedTimeout := 30 * time.Second
	if config.Timeout != expectedTimeout {
		t.Errorf("Expected Timeout to be %v, got %v", expectedTimeout, config.Timeout)
	}

	// Test float default
	if config.Rate != 1.5 {
		t.Errorf("Expected Rate to be 1.5, got %f", config.Rate)
	}

	// Test slice default
	expectedTags := []string{"tag1", "tag2", "tag3"}
	if len(config.Tags) != len(expectedTags) {
		t.Errorf("Expected Tags length to be %d, got %d", len(expectedTags), len(config.Tags))
	} else {
		for i, tag := range expectedTags {
			if config.Tags[i] != tag {
				t.Errorf("Expected Tags[%d] to be '%s', got '%s'", i, tag, config.Tags[i])
			}
		}
	}

	// Test pointer default
	if config.Optional == nil {
		t.Errorf("Expected Optional to be set, got nil")
	} else if *config.Optional != "optional-value" {
		t.Errorf("Expected Optional to be 'optional-value', got '%s'", *config.Optional)
	}
}

func TestSetDefaultsWithExistingValues(t *testing.T) {
	config := &TestConfig{
		Name: "existing-name",
		Port: 9000,
	}

	err := SetDefaults(config)
	if err != nil {
		t.Fatalf("SetDefaults failed: %v", err)
	}

	// Existing values should not be overridden
	if config.Name != "existing-name" {
		t.Errorf("Expected Name to remain 'existing-name', got '%s'", config.Name)
	}

	if config.Port != 9000 {
		t.Errorf("Expected Port to remain 9000, got %d", config.Port)
	}

	// Zero values should get defaults
	if !config.Enabled {
		t.Errorf("Expected Enabled to be set to default true, got %v", config.Enabled)
	}
}

func TestSetDefaultsNested(t *testing.T) {
	nested := &NestedConfig{}
	err := SetDefaults(nested)
	if err != nil {
		t.Fatalf("SetDefaults failed: %v", err)
	}

	// Check nested struct defaults
	if nested.Database.Name != "test-app" {
		t.Errorf("Expected Database.Name to be 'test-app', got '%s'", nested.Database.Name)
	}

	if nested.Cache.Port != 8080 {
		t.Errorf("Expected Cache.Port to be 8080, got %d", nested.Cache.Port)
	}
}

func TestSetDefaultsNilPointer(t *testing.T) {
	err := SetDefaults(nil)
	if err != nil {
		t.Errorf("Expected no error for nil pointer, got %v", err)
	}
}

func TestSetDefaultsNonStruct(t *testing.T) {
	var s string
	err := SetDefaults(&s)
	if err != nil {
		t.Errorf("Expected no error for non-struct, got %v", err)
	}
}
