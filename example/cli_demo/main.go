package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/nextpkg/vcfg"
	"github.com/urfave/cli/v3"
)

// Config represents the application configuration
type Config struct {
	Name    string `json:"name" yaml:"name" default:"default-app"`
	Port    int    `json:"port" yaml:"port" default:"8080"`
	Enabled bool   `json:"enabled" yaml:"enabled" default:"false"`
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return nil
}

func main() {
	// Create CLI application
	app := &cli.Command{
		Name:  "cli-demo",
		Usage: "Demo application with CLI flags and file config",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "name",
				Usage: "Application name",
			},
			&cli.IntFlag{
				Name:  "port",
				Usage: "Server port",
			},
			&cli.BoolFlag{
				Name:  "enabled",
				Usage: "Enable the service",
			},
			&cli.StringFlag{
				Name:  "config",
				Usage: "Configuration file path",
				Value: "config.yaml",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			configFile := cmd.String("config")

			// Debug: Check CLI flag status
			fmt.Printf("CLI flag status:\n")
			fmt.Printf("  name IsSet: %v, Value: %s\n", cmd.IsSet("name"), cmd.String("name"))
			fmt.Printf("  port IsSet: %v, Value: %d\n", cmd.IsSet("port"), cmd.Int("port"))
			fmt.Printf("  enabled IsSet: %v, Value: %t\n", cmd.IsSet("enabled"), cmd.Bool("enabled"))
			fmt.Println()

			// Build configuration manager with file and CLI flags
			// Note: CLI flags will override file config values
			manager, err := vcfg.NewBuilder[Config]().
				AddFile(configFile).  // File config (lower priority)
				AddCliFlags(cmd, ""). // CLI flags (higher priority) - empty delimiter to avoid command name prefix
				Build(context.Background())
			if err != nil {
				return fmt.Errorf("failed to build config: %w", err)
			}
			defer manager.CloseWithContext(context.Background())

			// Get merged configuration
			config := manager.Get()
			fmt.Printf("Final configuration:\n")
			fmt.Printf("  Name: %s\n", config.Name)
			fmt.Printf("  Port: %d\n", config.Port)
			fmt.Printf("  Enabled: %t\n", config.Enabled)

			// Show which values came from CLI flags
			fmt.Printf("\nCLI flag values:\n")
			if cmd.IsSet("name") {
				fmt.Printf("  --name: %s\n", cmd.String("name"))
			}
			if cmd.IsSet("port") {
				fmt.Printf("  --port: %d\n", cmd.Int("port"))
			}
			if cmd.IsSet("enabled") {
				fmt.Printf("  --enabled: %t\n", cmd.Bool("enabled"))
			}

			return nil
		},
	}

	// Run the CLI application with actual command line arguments
	// Example usage:
	//   go run main.go --name my-app --port 9000 --enabled
	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
