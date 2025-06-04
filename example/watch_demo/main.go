package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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

// ConfigChangeWatcher watches for configuration changes and displays them
type ConfigChangeWatcher struct {
	cm         *vcfg.ConfigManager[AppConfig]
	lastConfig *AppConfig
}

// NewConfigChangeWatcher creates a new configuration change watcher
func NewConfigChangeWatcher(cm *vcfg.ConfigManager[AppConfig]) *ConfigChangeWatcher {
	return &ConfigChangeWatcher{
		cm:         cm,
		lastConfig: cm.Get(),
	}
}

// StartWatching starts watching for configuration changes
func (w *ConfigChangeWatcher) StartWatching(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond) // Check every 500ms
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			currentConfig := w.cm.Get()
			if w.hasConfigChanged(currentConfig) {
				fmt.Println("\n🔄 Configuration change detected!")
				printConfigComparison(w.lastConfig, currentConfig)
				w.lastConfig = currentConfig
				fmt.Println("\n⏳ Waiting for more changes... (Ctrl+C to exit)")
			}
		}
	}
}

// hasConfigChanged checks if the configuration has changed
func (w *ConfigChangeWatcher) hasConfigChanged(newConfig *AppConfig) bool {
	if w.lastConfig == nil {
		return true
	}
	return w.lastConfig.Server.Host != newConfig.Server.Host ||
		w.lastConfig.Server.Port != newConfig.Server.Port ||
		w.lastConfig.Database.Host != newConfig.Database.Host ||
		w.lastConfig.Database.Port != newConfig.Database.Port ||
		w.lastConfig.Database.Name != newConfig.Database.Name
}

func main() {
	fmt.Println("🔄 VCFG Hot Reload Demo")
	fmt.Println("=======================")

	// Create configuration manager with hot reload enabled
	cm, err := vcfg.NewBuilder[AppConfig]().
		AddFile("config.yaml").
		WithWatch(). // Enable hot reload
		Build()
	if err != nil {
		log.Fatalf("❌ Failed to create config manager: %v", err)
	}
	defer cm.CloseWithContext(context.Background())

	fmt.Println("✅ Configuration manager started with hot reload")
	fmt.Println("\n💡 Try modifying config.yaml to see hot reload in action!")
	fmt.Println("   Example commands to test:")
	fmt.Println("   sed -i 's/port: 8080/port: 9090/' config.yaml")
	fmt.Println("   sed -i 's/localhost/127.0.0.1/' config.yaml")
	fmt.Println("   Press Ctrl+C to exit")

	// Display initial configuration
	printConfig("🚀 Initial", cm.Get())

	fmt.Println("\n⏳ Watching for configuration changes...")
	fmt.Println("   (Changes will be detected automatically)")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start configuration change watcher
	watcher := NewConfigChangeWatcher(cm)
	go watcher.StartWatching(ctx)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n🛑 Shutting down...")
	cancel() // Stop the watcher
	cm.DisableWatch()
}

func printConfig(prefix string, config *AppConfig) {
	fmt.Printf("\n%s Configuration [%s]\n", prefix, time.Now().Format("15:04:05"))
	fmt.Printf("├─ Server: %s:%d\n", config.Server.Host, config.Server.Port)
	fmt.Printf("└─ Database: %s:%d/%s\n",
		config.Database.Host,
		config.Database.Port,
		config.Database.Name)
}

// printConfigComparison prints a comparison between old and new configurations
func printConfigComparison(oldConfig, newConfig *AppConfig) {
	fmt.Printf("\n📊 Configuration Changes [%s]\n", time.Now().Format("15:04:05"))

	// Server changes
	if oldConfig.Server.Host != newConfig.Server.Host {
		fmt.Printf("├─ Server Host: %s → %s ✨\n", oldConfig.Server.Host, newConfig.Server.Host)
	} else {
		fmt.Printf("├─ Server Host: %s (unchanged)\n", newConfig.Server.Host)
	}

	if oldConfig.Server.Port != newConfig.Server.Port {
		fmt.Printf("├─ Server Port: %d → %d ✨\n", oldConfig.Server.Port, newConfig.Server.Port)
	} else {
		fmt.Printf("├─ Server Port: %d (unchanged)\n", newConfig.Server.Port)
	}

	// Database changes
	if oldConfig.Database.Host != newConfig.Database.Host {
		fmt.Printf("├─ Database Host: %s → %s ✨\n", oldConfig.Database.Host, newConfig.Database.Host)
	} else {
		fmt.Printf("├─ Database Host: %s (unchanged)\n", newConfig.Database.Host)
	}

	if oldConfig.Database.Port != newConfig.Database.Port {
		fmt.Printf("├─ Database Port: %d → %d ✨\n", oldConfig.Database.Port, newConfig.Database.Port)
	} else {
		fmt.Printf("├─ Database Port: %d (unchanged)\n", newConfig.Database.Port)
	}

	if oldConfig.Database.Name != newConfig.Database.Name {
		fmt.Printf("└─ Database Name: %s → %s ✨\n", oldConfig.Database.Name, newConfig.Database.Name)
	} else {
		fmt.Printf("└─ Database Name: %s (unchanged)\n", newConfig.Database.Name)
	}
}
