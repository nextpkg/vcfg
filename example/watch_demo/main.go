package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/knadh/koanf/providers/env"
	"github.com/nextpkg/vcfg"
)

type TestConfig struct {
	Message string `koanf:"message" default:"default message"`
	Value   int    `koanf:"value" default:"42"`
}

func (c *TestConfig) Validate() error {
	return nil
}

// SetDefaults 方法不再需要，默认值通过结构体标签自动设置

func main() {
	// Create a test config file
	testFile := "test_config.json"
	initialContent := `{
	"message": "initial message",
	"value": 100
}`

	// Write initial config
	if err := writeFile(testFile, initialContent); err != nil {
		log.Fatal("Failed to write initial config:", err)
	}

	// Initialize configuration manager with watch enabled
	cm := vcfg.MustInit[TestConfig](
		testFile,
		env.ProviderWithValue("TEST_", ".", func(s string, v string) (string, any) {
			return strings.ToLower(strings.TrimPrefix(s, "TEST_")), v
		}),
	).EnableWatch()

	fmt.Println("Watch enabled successfully")
	fmt.Println("Initial config:")
	printConfig(cm.Get())

	// Wait a bit
	time.Sleep(2 * time.Second)

	// Modify the config file
	updatedContent := `{
	"message": "updated message",
	"value": 200
}`

	fmt.Println("\n=== Updating config file ===")
	if err := writeFile(testFile, updatedContent); err != nil {
		log.Fatal("Failed to update config:", err)
	}

	// Wait for the watch to trigger
	time.Sleep(3 * time.Second)

	fmt.Println("\nConfig after file update:")
	printConfig(cm.Get())

	// Test again with another update
	time.Sleep(2 * time.Second)
	finalContent := `{
	"message": "final message",
	"value": 300
}`

	fmt.Println("\n=== Second update ===")
	if err := writeFile(testFile, finalContent); err != nil {
		log.Fatal("Failed to update config:", err)
	}

	// Wait for the watch to trigger
	time.Sleep(3 * time.Second)

	fmt.Println("\nFinal config:")
	printConfig(cm.Get())

	// Clean up
	// Note: DisableWatch method doesn't exist yet, we'll implement it later
	os.Remove(testFile)
	fmt.Println("\n=== Watch Test Completed ===")
}

func printConfig(cfg *TestConfig) {
	fmt.Printf("  Message: %s\n", cfg.Message)
	fmt.Printf("  Value: %d\n", cfg.Value)
}

func writeFile(filename, content string) error {
	return os.WriteFile(filename, []byte(content), 0644)
}
