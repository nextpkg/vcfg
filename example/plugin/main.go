package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/nextpkg/vcfg"
	"github.com/nextpkg/vcfg/plugins"
)

// AppConfig defines the application configuration with plugin support
type AppConfig struct {
	App struct {
		Name    string `json:"name" yaml:"name"`
		Version string `json:"version" yaml:"version"`
	} `json:"app" yaml:"app"`

	// Plugin configuration section - plugins are automatically discovered
	// by the plugin system based on struct fields that implement plugins.Config interface
	Metrics MetricsConfig `json:"metrics" yaml:"metrics,omitempty"`
	Health  HealthConfig  `json:"health" yaml:"health,omitempty"`
}

// ===== Custom Metrics Plugin =====

// MetricsConfig defines the configuration for the metrics plugin
type MetricsConfig struct {
	plugins.BaseConfig `koanf:",squash"`
	Port               int    `koanf:"port" default:"9090" validate:"min=1,max=65535"`
	Path               string `koanf:"path" default:"/metrics"`
	Interval           string `koanf:"interval" default:"30s"`
	Enabled            bool   `koanf:"enabled" default:"true"`
}

// MetricsPlugin implements a simple metrics collection plugin
type MetricsPlugin struct {
	config   *MetricsConfig
	mu       sync.RWMutex
	metrics  map[string]int64
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// Startup initializes the metrics plugin
func (p *MetricsPlugin) Startup(ctx context.Context, config any) error {
	p.config = config.(*MetricsConfig)
	p.metrics = make(map[string]int64)
	p.stopChan = make(chan struct{})

	fmt.Printf("[MetricsPlugin] Starting metrics collection on port %d\n", p.config.Port)
	fmt.Printf("[MetricsPlugin] Metrics endpoint: %s\n", p.config.Path)
	fmt.Printf("[MetricsPlugin] Collection interval: %s\n", p.config.Interval)

	if p.config.Enabled {
		p.startCollection()
	}

	return nil
}

// Reload updates the plugin configuration
func (p *MetricsPlugin) Reload(ctx context.Context, config any) error {
	newConfig := config.(*MetricsConfig)

	fmt.Printf("[MetricsPlugin] Reloading configuration...\n")
	fmt.Printf("[MetricsPlugin] Old port: %d, New port: %d\n", p.config.Port, newConfig.Port)

	// Stop current collection if running
	if p.config.Enabled {
		p.stopCollection()
	}

	// Update configuration
	p.config = newConfig

	// Restart collection if enabled
	if p.config.Enabled {
		p.startCollection()
	}

	fmt.Printf("[MetricsPlugin] Configuration reloaded successfully\n")
	return nil
}

// Shutdown gracefully stops the plugin
func (p *MetricsPlugin) Shutdown(ctx context.Context) error {
	fmt.Printf("[MetricsPlugin] Shutting down...\n")

	if p.config.Enabled {
		p.stopCollection()
	}

	fmt.Printf("[MetricsPlugin] Shutdown complete\n")
	return nil
}

// startCollection begins metrics collection
func (p *MetricsPlugin) startCollection() {
	interval, err := time.ParseDuration(p.config.Interval)
	if err != nil {
		interval = 30 * time.Second
	}

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				p.collectMetrics()
			case <-p.stopChan:
				return
			}
		}
	}()
}

// stopCollection stops metrics collection
func (p *MetricsPlugin) stopCollection() {
	close(p.stopChan)
	p.wg.Wait()
	p.stopChan = make(chan struct{})
}

// collectMetrics simulates metrics collection
func (p *MetricsPlugin) collectMetrics() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Simulate collecting some metrics
	p.metrics["requests_total"]++
	p.metrics["memory_usage"] = time.Now().Unix() % 1000
	p.metrics["cpu_usage"] = time.Now().Unix() % 100

	fmt.Printf("[MetricsPlugin] Collected metrics: requests=%d, memory=%d, cpu=%d\n",
		p.metrics["requests_total"], p.metrics["memory_usage"], p.metrics["cpu_usage"])
}

// GetMetrics returns current metrics (for demonstration)
func (p *MetricsPlugin) GetMetrics() map[string]int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(map[string]int64)
	for k, v := range p.metrics {
		result[k] = v
	}
	return result
}

// ===== Custom Health Check Plugin =====

// HealthConfig defines the configuration for the health check plugin
type HealthConfig struct {
	plugins.BaseConfig `koanf:",squash"`
	Port               int      `koanf:"port" default:"8081" validate:"min=1,max=65535"`
	Path               string   `koanf:"path" default:"/health"`
	Checks             []string `koanf:"checks" default:"[database,redis,external_api]"`
	Timeout            string   `koanf:"timeout" default:"5s"`
	Interval           string   `koanf:"interval" default:"10s"`
}

// HealthPlugin implements a health check plugin
type HealthPlugin struct {
	config     *HealthConfig
	healthData map[string]bool
	mu         sync.RWMutex
	stopChan   chan struct{}
	wg         sync.WaitGroup
}

// Startup initializes the health check plugin
func (p *HealthPlugin) Startup(ctx context.Context, config any) error {
	p.config = config.(*HealthConfig)
	p.healthData = make(map[string]bool)
	p.stopChan = make(chan struct{})

	fmt.Printf("[HealthPlugin] Starting health checks on port %d\n", p.config.Port)
	fmt.Printf("[HealthPlugin] Health endpoint: %s\n", p.config.Path)
	fmt.Printf("[HealthPlugin] Monitoring checks: %v\n", p.config.Checks)

	// Initialize health status
	for _, check := range p.config.Checks {
		p.healthData[check] = true // Assume healthy initially
	}

	p.startHealthChecks()
	return nil
}

// Reload updates the plugin configuration
func (p *HealthPlugin) Reload(ctx context.Context, config any) error {
	newConfig := config.(*HealthConfig)

	fmt.Printf("[HealthPlugin] Reloading configuration...\n")

	// Stop current health checks
	p.stopHealthChecks()

	// Update configuration
	p.config = newConfig

	// Update health data for new checks
	p.mu.Lock()
	p.healthData = make(map[string]bool)
	for _, check := range p.config.Checks {
		p.healthData[check] = true
	}
	p.mu.Unlock()

	// Restart health checks
	p.startHealthChecks()

	fmt.Printf("[HealthPlugin] Configuration reloaded successfully\n")
	return nil
}

// Shutdown gracefully stops the plugin
func (p *HealthPlugin) Shutdown(ctx context.Context) error {
	fmt.Printf("[HealthPlugin] Shutting down...\n")
	p.stopHealthChecks()
	fmt.Printf("[HealthPlugin] Shutdown complete\n")
	return nil
}

// startHealthChecks begins health monitoring
func (p *HealthPlugin) startHealthChecks() {
	interval, err := time.ParseDuration(p.config.Interval)
	if err != nil {
		interval = 10 * time.Second
	}

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				p.performHealthChecks()
			case <-p.stopChan:
				return
			}
		}
	}()
}

// stopHealthChecks stops health monitoring
func (p *HealthPlugin) stopHealthChecks() {
	close(p.stopChan)
	p.wg.Wait()
	p.stopChan = make(chan struct{})
}

// performHealthChecks simulates health checking
func (p *HealthPlugin) performHealthChecks() {
	p.mu.Lock()
	defer p.mu.Unlock()

	allHealthy := true
	for _, check := range p.config.Checks {
		// Simulate health check (randomly fail sometimes)
		healthy := time.Now().Unix()%10 != 0 // 90% success rate
		p.healthData[check] = healthy
		if !healthy {
			allHealthy = false
		}
	}

	status := "HEALTHY"
	if !allHealthy {
		status = "UNHEALTHY"
	}

	fmt.Printf("[HealthPlugin] Health check completed: %s - %v\n", status, p.healthData)
}

// GetHealth returns current health status (for demonstration)
func (p *HealthPlugin) GetHealth() map[string]bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(map[string]bool)
	for k, v := range p.healthData {
		result[k] = v
	}
	return result
}

// Register plugins
func init() {
	// Register custom plugins with the plugin system
	plugins.RegisterPluginType("metrics", &MetricsPlugin{}, &MetricsConfig{})

	// Register health check plugin
	plugins.RegisterPluginType("health", &HealthPlugin{}, &HealthConfig{})
}

func main() {
	// Set log level to Debug to see configuration change events
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	fmt.Println("VCFG Plugin Development Example")
	fmt.Println("==============================")

	// Create example configuration
	if err := createPluginConfig(); err != nil {
		log.Fatal("Failed to create plugin config:", err)
	}

	fmt.Println("\n1. Loading configuration with custom plugins...")

	// Build configuration manager with plugins enabled
	cm := vcfg.NewBuilder[AppConfig]().
		AddFile("plugin-config.yaml").
		WithPlugin(). // Enable plugin system
		WithWatch().  // Enable file watching for hot reload
		MustBuild()
	defer cm.Close()

	config := cm.Get()
	fmt.Printf("App: %s v%s\n", config.App.Name, config.App.Version)

	// Count active plugins
	pluginCount := 0
	if config.Metrics.Type != "" {
		pluginCount++
		fmt.Printf("Metrics plugin loaded on port %d\n", config.Metrics.Port)
	}
	if config.Health.Type != "" {
		pluginCount++
		fmt.Printf("Health plugin loaded on port %d\n", config.Health.Port)
	}
	fmt.Printf("Total plugins loaded: %d\n", pluginCount)

	// Note: Configuration changes are automatically handled by the file watcher
	// Plugins will be automatically reloaded when configuration changes

	fmt.Println("\n2. Plugins are now running...")
	fmt.Println("   - Metrics plugin is collecting metrics every 5 seconds")
	fmt.Println("   - Health plugin is performing checks every 8 seconds")
	fmt.Println("   - Try modifying plugin-config.yaml to see hot reloading")
	fmt.Println("\n3. Running for 30 seconds... (Press Ctrl+C to stop early)")

	// Run for a while to demonstrate plugin functionality
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Wait for timeout or interruption
	<-ctx.Done()

	fmt.Println("\n4. Shutting down...")
	fmt.Println("Plugins will be automatically shut down by the configuration manager.")
	fmt.Println("\nExample completed successfully!")
}

// createPluginConfig creates an example configuration file with plugin settings
func createPluginConfig() error {
	config := `# VCFG Plugin Development Example Configuration
app:
  name: "Plugin-Development-Example"
  version: "1.0.0"

# Custom metrics plugin
metrics:
  type: "metrics"
  port: 9090
  path: "/metrics"
  interval: "5s"
  enabled: true

# Custom health check plugin
health:
  type: "health"
  port: 8081
  path: "/health"
  checks:
    - "database"
    - "redis"
    - "external_api"
    - "file_system"
  timeout: "3s"
  interval: "8s"
`

	return os.WriteFile("plugin-config.yaml", []byte(config), 0644)
}
