package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nextpkg/vcfg"
)

// AppConfig represents the application configuration
type AppConfig struct {
	Name   string       `yaml:"name" json:"name"`
	Server ServerConfig `yaml:"server" json:"server"`
	Cache  DemoConfig   `yaml:"cache" json:"cache"`
}

type ServerConfig struct {
	Host string `yaml:"host" json:"host" default:"localhost"`
	Port int    `yaml:"port" json:"port" default:"8080"`
}

func main() {
	fmt.Println("🚀 VCFG Global Plugin Demo (Generic Version)")
	fmt.Println("============================================")
	fmt.Println("✨ No cacheExtractor needed - plugin works directly with AppConfig!")
	fmt.Println()

	// Create configuration manager first
	cm, err := vcfg.NewBuilder[AppConfig]().
		AddFile("./config.yaml").
		Build()
	if err != nil {
		log.Fatalf("❌ Failed to create config manager: %v", err)
	}
	defer cm.Close()

	fmt.Printf("✅ Configuration loaded: %s\n", cm.Get().Name)

	// Plugin is automatically registered via init() function in demo_plugin.go
	// No manual registration needed!

	fmt.Println("✅ Plugin registered (no cacheExtractor needed!)")

	// Start all plugins
	ctx := context.Background()
	err = cm.StartAllPlugins(ctx)
	if err != nil {
		log.Fatalf("❌ Failed to start plugins: %v", err)
	}

	fmt.Println("✅ All plugins started")
	fmt.Println()

	// Demonstrate that the plugin has access to the full configuration
	fmt.Println("🎯 Key Benefits:")
	fmt.Println("   • Plugin can access ANY part of the configuration")
	fmt.Println("   • No need for specific extractors like cacheExtractor")
	fmt.Println("   • Simpler, more flexible plugin design")
	fmt.Println("   • Type-safe with generics")

	// Wait a moment to see the plugin in action
	time.Sleep(2 * time.Second)

	fmt.Println("\n🏁 Demo completed successfully")
}
