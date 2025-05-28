package vcfg

import (
	"fmt"
	"sync"

	"github.com/nextpkg/vcfg/ce"
	"github.com/nextpkg/vcfg/source"
	"github.com/nextpkg/vcfg/validator"
	"github.com/nextpkg/vcfg/viper"
	"github.com/nextpkg/vcfg/watcher"
	"go.uber.org/atomic"
)

// ConfigManager is a configuration manager that handles loading, validation, and watching.
// It supports generic configuration types through the type parameter T.
// ConfigManager provides thread-safe access to configuration values.
type ConfigManager[T any] struct {
	sources []source.Source
	viper   *viper.Viper
	watcher *watcher.Watcher[T]
	once    sync.Once
	cfg     atomic.Value
	mu      sync.RWMutex
}

// newManager creates a new configuration manager with the provided sources.
// It initializes the internal viper and watcher components.
// Parameters:
//   - sources: one or more configuration sources to manage.
//
// Returns:
//   - A new ConfigManager instance.
func newManager[T any](sources ...source.Source) *ConfigManager[T] {
	return &ConfigManager[T]{
		sources: sources,
		viper:   viper.New(),
		watcher: watcher.New[T](),
	}
}

// load loads configuration from sources, validates it, and returns the configuration struct.
// This method is thread-safe through mutex locking.
// Returns:
//   - A pointer to the loaded and validated configuration.
//   - An error if loading or validation fails.
func (cm *ConfigManager[T]) load() (*T, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// load all sources
	err := cm.loadSource()
	if err != nil {
		return nil, err
	}

	return cm.loadConfig()
}

// loadSource loads all configuration sources and merges them into viper.
// It reads from each source and combines the configurations.
// Returns:
//   - An error if reading from any source or merging configurations fails.
func (cm *ConfigManager[T]) loadSource() error {
	for _, src := range cm.sources {
		// Read configuration
		cfg, err := src.Read()
		if err != nil {
			return fmt.Errorf("%w: %w read from source %s", ce.ErrLoadProviderFailed, err, src.String())
		}

		// Merge configuration
		err = cm.viper.Merge(cfg)
		if err != nil {
			return err
		}
	}

	return nil
}

// loadConfig unmarshals the configuration from viper into a struct and validates it.
// Returns:
//   - A pointer to the unmarshaled and validated configuration.
//   - An error if unmarshaling or validation fails.
func (cm *ConfigManager[T]) loadConfig() (*T, error) {
	var cfg T

	// Set default values
	if def, ok := any(&cfg).(interface{ SetDefaults() }); ok {
		def.SetDefaults()
	}

	err := cm.viper.Unmarshal(&cfg)
	if err != nil {
		return nil, fmt.Errorf("unmarshal config failed. %w", err)
	}

	err = validator.Validate(&cfg)
	if err != nil {
		return nil, fmt.Errorf("validate config failed. %w", err)
	}

	return &cfg, nil
}

// EnableWatch starts monitoring changes of all configuration sources.
// It sets up a callback that reloads configurations when changes are detected.
func (cm *ConfigManager[T]) EnableWatch() *ConfigManager[T] {
	cm.once.Do(func() {
		cm.watcher = watcher.New[T]()

		// Register callback to update when configuration changes
		cm.watcher.OnChange(func(t *T) error {
			cm.cfg.Store(t)
			return nil
		})

		callback := func(events []watcher.Event[*T]) error {
			// Reload configuration from all sources
			err := cm.loadSource()
			if err != nil {
				return fmt.Errorf("load source from config failed. %w", err)
			}

			// Create a new configuration object for callback
			for _, fn := range events {
				// Create a new configuration object for each callback
				var newCfg *T
				newCfg, err = cm.loadConfig()
				if err != nil {
					return err
				}

				// Call the callback
				err = fn(newCfg)
				if err != nil {
					return err
				}
			}

			return nil
		}
		cm.watcher.Watch(cm.sources, callback)
	})
	return cm
}

// DisableWatch stops monitoring changes of all configuration sources.
func (cm *ConfigManager[T]) DisableWatch() {
	cm.watcher.Stop()
	cm.once = sync.Once{}
}

// Get returns the current configuration value.
// It retrieves the stored configuration from atomic.Value and returns it as type T.
func (cm *ConfigManager[T]) Get() *T {
	cfg := cm.cfg.Value.Load()
	ret, ok := cfg.(*T)
	if !ok {
		panic("please init config manager at first")
	}

	return ret
}
