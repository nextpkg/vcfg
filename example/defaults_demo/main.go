package main

import (
	"fmt"
	"time"

	"github.com/nextpkg/vcfg/defaults"
)

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Host     string        `default:"localhost"`
	Port     int           `default:"5432"`
	Name     string        `default:"myapp"`
	User     string        `default:"postgres"`
	Password string        `default:""`
	Timeout  time.Duration `default:"30s"`
	SSL      bool          `default:"false"`
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
	Host     string        `default:"localhost"`
	Port     int           `default:"6379"`
	DB       int           `default:"0"`
	Password string        `default:""`
	Timeout  time.Duration `default:"5s"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Host         string        `default:"0.0.0.0"`
	Port         int           `default:"8080"`
	ReadTimeout  time.Duration `default:"30s"`
	WriteTimeout time.Duration `default:"30s"`
	MaxBodySize  int64         `default:"1048576"` // 1MB
}

// LogConfig represents logging configuration
type LogConfig struct {
	Level  string   `default:"info"`
	Format string   `default:"json"`
	Output []string `default:"stdout,file"`
}

// AppConfig represents the complete application configuration
type AppConfig struct {
	AppName     string         `default:"my-awesome-app"`
	Version     string         `default:"1.0.0"`
	Environment string         `default:"development"`
	Debug       bool           `default:"true"`
	Database    DatabaseConfig `default:""`
	Redis       RedisConfig    `default:""`
	Server      ServerConfig   `default:""`
	Log         LogConfig      `default:""`
	Features    *FeatureConfig `default:""`
}

// FeatureConfig represents feature flags
type FeatureConfig struct {
	EnableMetrics   bool `default:"true"`
	EnableTracing   bool `default:"false"`
	EnableRateLimit bool `default:"true"`
}

// SetDefaults sets default values using struct tags
// SetDefaults 方法不再需要，默认值通过结构体标签自动设置

func main() {
	fmt.Println("=== Defaults Demo ===")
	fmt.Println()

	// Example 1: Empty configuration with defaults
	fmt.Println("1. Empty configuration with defaults:")
	config1 := &AppConfig{}
	defaults.SetDefaults(config1)
	printConfig(config1)

	// Example 2: Partial configuration with defaults
	fmt.Println("\n2. Partial configuration with defaults:")
	config2 := &AppConfig{
		AppName:     "custom-app",
		Environment: "production",
		Database: DatabaseConfig{
			Host: "prod-db.example.com",
			User: "prod_user",
		},
	}
	defaults.SetDefaults(config2)
	printConfig(config2)

	// Example 3: Using defaults package directly
	fmt.Println("\n3. Using defaults package directly:")
	config3 := &AppConfig{
		Server: ServerConfig{
			Port: 9000, // Custom port
		},
	}
	if err := defaults.SetDefaults(config3); err != nil {
		fmt.Printf("Error setting defaults: %v\n", err)
		return
	}
	printConfig(config3)

	// Example 4: Demonstrate the old way vs new way
	fmt.Println("\n4. Comparison with old SetDefaults approach:")
	fmt.Println("Old way: Manual if statements for each field")
	fmt.Println("New way: Declarative struct tags + automatic processing")
	fmt.Println("Benefits:")
	fmt.Println("  - No repetitive if statements")
	fmt.Println("  - Self-documenting defaults in struct definition")
	fmt.Println("  - Type-safe default value parsing")
	fmt.Println("  - Support for complex types (duration, slices, pointers)")
	fmt.Println("  - Nested struct support")
	fmt.Println("  - Consistent behavior across all structs")
}

func printConfig(config *AppConfig) {
	fmt.Printf("  AppName: %s\n", config.AppName)
	fmt.Printf("  Version: %s\n", config.Version)
	fmt.Printf("  Environment: %s\n", config.Environment)
	fmt.Printf("  Debug: %v\n", config.Debug)
	fmt.Printf("  Database:")
	fmt.Printf("    Host: %s\n", config.Database.Host)
	fmt.Printf("    Port: %d\n", config.Database.Port)
	fmt.Printf("    Name: %s\n", config.Database.Name)
	fmt.Printf("    User: %s\n", config.Database.User)
	fmt.Printf("    Timeout: %v\n", config.Database.Timeout)
	fmt.Printf("    SSL: %v\n", config.Database.SSL)
	fmt.Printf("  Redis:")
	fmt.Printf("    Host: %s\n", config.Redis.Host)
	fmt.Printf("    Port: %d\n", config.Redis.Port)
	fmt.Printf("    DB: %d\n", config.Redis.DB)
	fmt.Printf("    Timeout: %v\n", config.Redis.Timeout)
	fmt.Printf("  Server:")
	fmt.Printf("    Host: %s\n", config.Server.Host)
	fmt.Printf("    Port: %d\n", config.Server.Port)
	fmt.Printf("    ReadTimeout: %v\n", config.Server.ReadTimeout)
	fmt.Printf("    WriteTimeout: %v\n", config.Server.WriteTimeout)
	fmt.Printf("    MaxBodySize: %d\n", config.Server.MaxBodySize)
	fmt.Printf("  Log:")
	fmt.Printf("    Level: %s\n", config.Log.Level)
	fmt.Printf("    Format: %s\n", config.Log.Format)
	fmt.Printf("    Output: %v\n", config.Log.Output)
	if config.Features != nil {
		fmt.Printf("  Features:")
		fmt.Printf("    EnableMetrics: %v\n", config.Features.EnableMetrics)
		fmt.Printf("    EnableTracing: %v\n", config.Features.EnableTracing)
		fmt.Printf("    EnableRateLimit: %v\n", config.Features.EnableRateLimit)
	} else {
		fmt.Printf("  Features: nil\n")
	}
}
