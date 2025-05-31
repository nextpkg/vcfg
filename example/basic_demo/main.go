package main

import (
	"fmt"
	"log"

	"github.com/nextpkg/vcfg"
)

// AppConfig represents a simple application configuration
type AppConfig struct {
	Server   ServerConfig   `yaml:"server" json:"server"`
	Database DatabaseConfig `yaml:"database" json:"database"`
}

type ServerConfig struct {
	Host string `yaml:"host" json:"host" default:"localhost"`
	Port int    `yaml:"port" json:"port" default:"8080"`
}

type DatabaseConfig struct {
	Host string `yaml:"host" json:"host"`
	Port int    `yaml:"port" json:"port" default:"5432"`
	Name string `yaml:"name" json:"name"`
}

func main() {
	fmt.Println("🚀 VCFG Basic Usage Demo")
	fmt.Println("========================")

	// Create configuration manager with multiple sources
	cm, err := vcfg.NewBuilder[AppConfig]().
		AddFile("config.yaml"). // Primary config file
		AddEnv("APP").          // Environment variables with APP_ prefix
		Build()
	if err != nil {
		log.Fatalf("❌ Failed to create config manager: %v", err)
	}
	defer cm.Close()

	fmt.Println("✅ Configuration loaded successfully")

	// Get current configuration
	config := cm.Get()

	// Display configuration
	fmt.Println("\n📋 Current Configuration:")
	fmt.Printf("Server: %s:%d\n", config.Server.Host, config.Server.Port)
	fmt.Printf("Database: %s:%d/%s\n",
		config.Database.Host,
		config.Database.Port,
		config.Database.Name)

	// Show configuration source priority
	fmt.Println("\n🔄 Configuration Sources (priority order):")
	fmt.Println("1. Environment variables (highest) - prefix: APP_")
	fmt.Println("2. config.yaml (lowest)")

	fmt.Println("\n💡 Try these commands to see environment override:")
	fmt.Println("   export APP_SERVER_PORT=9090")
	fmt.Println("   export APP_DATABASE_HOST=prod.example.com")
	fmt.Println("   go run main.go")
}
