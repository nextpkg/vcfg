package vcfg

// MustInit initializes a new ConfigManager with the provided sources.
// It accepts both file paths (strings) and koanf.Provider instances.
// It panics if initialization fails.
func MustInit[T any](sources ...any) *ConfigManager[T] {
	cm := newManager[T](sources...)

	// Load initial configuration
	cfg, err := cm.load()
	if err != nil {
		panic(err)
	}

	cm.cfg.Store(cfg)
	return cm
}
