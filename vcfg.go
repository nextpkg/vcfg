package vcfg

import (
	"github.com/nextpkg/vcfg/source"
)

// MustInit initializes a new ConfigManager with the provided sources
// It loads the configuration from sources and optionally sets up configuration watching
// If any error occurs during initialization, it will panic
// Parameters:
//   - enableWatching: if true, enables automatic configuration reloading when source changes
//   - sources: one or more configuration sources to load and WatchStart
//
// Returns:
//   - A fully initialized ConfigManager instance
func MustInit[T any](enableWatching bool, sources ...source.Source) *ConfigManager[T] {
	cm := newManager[T](sources...)

	cfg, err := cm.load()
	if err != nil {
		panic(err)
	}

	cm.cfg.Store(cfg)

	if enableWatching {
		// Register callback to update when configuration changes
		cm.watcher.OnChange(func(t *T) error {
			cm.cfg.Store(t)
			return nil
		})

		// Start monitoring
		cm.startWatch()
	}

	return cm
}

func MustInitFile[T any](path string) *ConfigManager[T] {
	return MustInit[T](true, source.NewFileSource(path))
}
