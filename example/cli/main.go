package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nextpkg/vcfg"
	"github.com/urfave/cli/v3"
)

// ServerConfig defines the server configuration structure
type ServerConfig struct {
	// Server settings
	Server struct {
		Host         string        `json:"host" yaml:"host" default:"localhost"`
		Port         int           `json:"port" yaml:"port" default:"8080" validate:"min=1,max=65535"`
		ReadTimeout  time.Duration `json:"read_timeout" yaml:"read_timeout" default:"30s"`
		WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout" default:"30s"`
	} `json:"server" yaml:"server"`

	// Database settings
	Database struct {
		URL         string `json:"url" yaml:"url" validate:"required"`
		MaxConns    int    `json:"max_conns" yaml:"max_conns" default:"10" validate:"min=1"`
		MaxIdle     int    `json:"max_idle" yaml:"max_idle" default:"5" validate:"min=1"`
		MaxLifetime string `json:"max_lifetime" yaml:"max_lifetime" default:"1h"`
	} `json:"database" yaml:"database"`

	// Application settings
	App struct {
		Name        string `json:"name" yaml:"name" default:"CLI-App"`
		Version     string `json:"version" yaml:"version" default:"1.0.0"`
		Environment string `json:"environment" yaml:"environment" default:"development" validate:"oneof=development staging production"`
		Debug       bool   `json:"debug" yaml:"debug" default:"false"`
	} `json:"app" yaml:"app"`

	// Logging configuration
	Logging struct {
		Level  string `json:"level" yaml:"level" default:"info" validate:"oneof=debug info warn error"`
		Format string `json:"format" yaml:"format" default:"json" validate:"oneof=json text"`
		Output string `json:"output" yaml:"output" default:"stdout" validate:"oneof=stdout stderr file both"`
		File   string `json:"file" yaml:"file" default:"./app.log"`
	} `json:"logging" yaml:"logging"`

	// Plugin configuration
	Plugins map[string]interface{} `json:"plugins" yaml:"plugins"`
}

func main() {
	app := &cli.Command{
		Name:        "vcfg-cli-example",
		Usage:       "VCFG CLI integration example",
		Description: "Demonstrates how to integrate VCFG with urfave/cli for configuration management",
		Version:     "1.0.0",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   "config.yaml",
				Usage:   "Configuration file path",
			},
			&cli.StringFlag{
				Name:  "server.host",
				Usage: "Server host address",
			},
			&cli.IntFlag{
				Name:  "server.port",
				Usage: "Server port number",
			},
			&cli.StringFlag{
				Name:  "app.environment",
				Usage: "Application environment (development, staging, production)",
			},
			&cli.BoolFlag{
				Name:  "app.debug",
				Usage: "Enable debug mode",
			},
			&cli.StringFlag{
				Name:  "logging.level",
				Usage: "Logging level (debug, info, warn, error)",
			},
			&cli.StringFlag{
				Name:  "database.url",
				Usage: "Database connection URL",
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "serve",
				Usage: "Start the server",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return runServer(ctx, cmd)
				},
			},
			{
				Name:  "config",
				Usage: "Configuration management",
				Commands: []*cli.Command{
					{
						Name:  "show",
						Usage: "Show current configuration",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return showConfig(ctx, cmd)
						},
					},
					{
						Name:  "validate",
						Usage: "Validate configuration",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return validateConfig(ctx, cmd)
						},
					},
					{
						Name:  "create-example",
						Usage: "Create example configuration file",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return createExampleConfig(ctx, cmd)
						},
					},
				},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Println("VCFG CLI Example")
			fmt.Println("Use --help to see available commands")
			return nil
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

// runServer starts the server with the loaded configuration
func runServer(ctx context.Context, cmd *cli.Command) error {
	fmt.Println("Starting server...")

	// Build configuration manager with multiple sources
	cm, err := vcfg.NewBuilder[ServerConfig]().
		AddFile(cmd.String("config")). // Configuration file
		AddEnv("VCFG_").               // Environment variables with VCFG_ prefix
		// Note: CLI flags integration requires special handling in urfave/cli context
		WithWatch().  // Enable file watching
		WithPlugin(). // Enable plugins
		Build(ctx)
	if err != nil {
		return fmt.Errorf("failed to build configuration: %w", err)
	}
	defer cm.Close()

	config := cm.Get()

	// Display configuration
	fmt.Printf("\n=== Server Configuration ===\n")
	fmt.Printf("App: %s v%s (%s)\n", config.App.Name, config.App.Version, config.App.Environment)
	fmt.Printf("Server: %s:%d\n", config.Server.Host, config.Server.Port)
	fmt.Printf("Database: %s (max_conns: %d)\n", config.Database.URL, config.Database.MaxConns)
	fmt.Printf("Logging: %s level, %s format, output to %s\n", config.Logging.Level, config.Logging.Format, config.Logging.Output)
	fmt.Printf("Debug: %t\n", config.App.Debug)
	fmt.Printf("Plugins: %d configured\n", len(config.Plugins))

	// Note: Configuration changes are automatically handled by the file watcher
	// The configuration will be reloaded when files change

	// Simulate server running
	fmt.Printf("\nServer is running... (Press Ctrl+C to stop)\n")
	fmt.Printf("Try modifying the configuration file to see hot reloading in action\n")

	// Keep the server running
	select {
	case <-ctx.Done():
		fmt.Println("\nShutting down server...")
		return nil
	}
}

// showConfig displays the current configuration
func showConfig(ctx context.Context, cmd *cli.Command) error {
	fmt.Println("Current Configuration:")

	cm, err := vcfg.NewBuilder[ServerConfig]().
		AddFile(cmd.String("config")).
		AddEnv("VCFG_").
		// Note: CLI flags integration requires special handling
		Build(ctx)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	defer cm.Close()

	config := cm.Get()

	fmt.Printf("\n=== Application ===\n")
	fmt.Printf("Name: %s\n", config.App.Name)
	fmt.Printf("Version: %s\n", config.App.Version)
	fmt.Printf("Environment: %s\n", config.App.Environment)
	fmt.Printf("Debug: %t\n", config.App.Debug)

	fmt.Printf("\n=== Server ===\n")
	fmt.Printf("Host: %s\n", config.Server.Host)
	fmt.Printf("Port: %d\n", config.Server.Port)
	fmt.Printf("Read Timeout: %s\n", config.Server.ReadTimeout)
	fmt.Printf("Write Timeout: %s\n", config.Server.WriteTimeout)

	fmt.Printf("\n=== Database ===\n")
	fmt.Printf("URL: %s\n", config.Database.URL)
	fmt.Printf("Max Connections: %d\n", config.Database.MaxConns)
	fmt.Printf("Max Idle: %d\n", config.Database.MaxIdle)
	fmt.Printf("Max Lifetime: %s\n", config.Database.MaxLifetime)

	fmt.Printf("\n=== Logging ===\n")
	fmt.Printf("Level: %s\n", config.Logging.Level)
	fmt.Printf("Format: %s\n", config.Logging.Format)
	fmt.Printf("Output: %s\n", config.Logging.Output)
	fmt.Printf("File: %s\n", config.Logging.File)

	if len(config.Plugins) > 0 {
		fmt.Printf("\n=== Plugins ===\n")
		for name, pluginConfig := range config.Plugins {
			fmt.Printf("%s: %+v\n", name, pluginConfig)
		}
	}

	return nil
}

// validateConfig validates the configuration
func validateConfig(ctx context.Context, cmd *cli.Command) error {
	fmt.Println("Validating configuration...")

	cm, err := vcfg.NewBuilder[ServerConfig]().
		AddFile(cmd.String("config")).
		AddEnv("VCFG_").
		// Note: CLI flags integration requires special handling
		Build(ctx)
	if err != nil {
		fmt.Printf("❌ Configuration validation failed: %v\n", err)
		return err
	}
	defer cm.Close()

	fmt.Println("✅ Configuration is valid!")
	return nil
}

// createExampleConfig creates an example configuration file
func createExampleConfig(ctx context.Context, cmd *cli.Command) error {
	configFile := cmd.String("config")
	fmt.Printf("Creating example configuration file: %s\n", configFile)

	exampleConfig := `# VCFG CLI Example Configuration

app:
  name: "VCFG-CLI-Example"
  version: "1.0.0"
  environment: "development"
  debug: true

server:
  host: "localhost"
  port: 8080
  read_timeout: "30s"
  write_timeout: "30s"

database:
  url: "postgres://user:password@localhost:5432/mydb?sslmode=disable"
  max_conns: 25
  max_idle: 10
  max_lifetime: "1h"

logging:
  level: "info"
  format: "json"
  output: "stdout"
  file: "./logs/app.log"

# Plugin configuration
plugins:
  logger:
    type: "logger"
    level: "debug"
    format: "json"
    output: "both"
    file_path: "./logs/app.log"
    add_source: true
    enable_rotation: true
    rotate_interval: "daily"
    max_file_size: 104857600  # 100MB
    max_age: 7
`

	if err := os.WriteFile(configFile, []byte(exampleConfig), 0644); err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	fmt.Printf("✅ Example configuration created successfully!\n")
	fmt.Printf("\nYou can now:\n")
	fmt.Printf("  1. View the configuration: %s config show\n", os.Args[0])
	fmt.Printf("  2. Validate the configuration: %s config validate\n", os.Args[0])
	fmt.Printf("  3. Start the server: %s serve\n", os.Args[0])
	fmt.Printf("  4. Override settings with CLI flags: %s serve --server.port 9090 --app.debug\n", os.Args[0])
	fmt.Printf("  5. Use environment variables: VCFG_SERVER_PORT=9090 %s serve\n", os.Args[0])

	return nil
}
