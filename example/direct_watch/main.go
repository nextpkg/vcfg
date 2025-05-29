package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Watcher interface {
	Watch(cb func(event any, err error)) error
}

type Unwatcher interface {
	Unwatch()
}

func main() {
	// Create a test config file
	testFile := "direct_test_config.json"
	initialContent := `{
	"message": "initial message",
	"value": 100
}`

	// Write initial config
	if err := os.WriteFile(testFile, []byte(initialContent), 0644); err != nil {
		log.Fatal("Failed to write initial config:", err)
	}
	defer os.Remove(testFile)

	// Create koanf instance and file provider
	k := koanf.New(".")
	f := file.Provider(testFile)

	// Load initial config
	if err := k.Load(f, json.Parser()); err != nil {
		log.Fatalf("error loading config: %v", err)
	}

	fmt.Println("Initial config loaded:")
	fmt.Printf("  Message: %s\n", k.String("message"))
	fmt.Printf("  Value: %d\n", k.Int("value"))

	// Test if file provider supports Watch
	if watcher, ok := any(f).(Watcher); ok {
		fmt.Println("\nFile provider supports Watch interface!")

		// Set up watch
		watcher.Watch(func(event interface{}, err error) {
			if err != nil {
				log.Printf("watch error: %v", err)
				return
			}

			fmt.Println("\n=== File change detected! ===")
			// Reload config
			k = koanf.New(".")
			if err := k.Load(f, json.Parser()); err != nil {
				log.Printf("error reloading config: %v", err)
				return
			}

			fmt.Println("Config reloaded:")
			fmt.Printf("  Message: %s\n", k.String("message"))
			fmt.Printf("  Value: %d\n", k.Int("value"))
		})

		fmt.Println("Watch set up successfully. Waiting for file changes...")

		// Wait a bit
		time.Sleep(2 * time.Second)

		// Modify the config file
		updatedContent := `{
	"message": "updated message",
	"value": 200
}`
		fmt.Println("\nModifying config file...")
		if err := os.WriteFile(testFile, []byte(updatedContent), 0644); err != nil {
			log.Fatal("Failed to update config:", err)
		}

		// Wait for watch to trigger
		time.Sleep(3 * time.Second)

		// Stop watching
		if unwatcher, ok := interface{}(f).(Unwatcher); ok {
			unwatcher.Unwatch()
			fmt.Println("\nWatch stopped.")
		}

	} else {
		fmt.Println("\nFile provider does NOT support Watch interface!")
		fmt.Printf("Provider type: %T\n", f)
	}

	fmt.Println("\nTest completed.")
}
