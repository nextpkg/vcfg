package vcfg

// MustLoad initializes a new ConfigManager with the provided sources.
// It accepts both file paths (strings) and koanf.Provider instances.
// It panics if initialization fails.
func MustLoad[T any](sources ...any) *ConfigManager[T] {
	cm := newManager[T](sources...)

	// Load initial configuration
	cfg, err := cm.load()
	if err != nil {
		panic(err)
	}

	cm.cfg.Store(cfg)
	return cm
}

// MustBuild 创建简单的配置管理器（仅文件源）
func MustBuild[T any](filePaths ...string) *ConfigManager[T] {
	builder := NewBuilder[T]()
	for _, path := range filePaths {
		builder.AddFile(path)
	}
	return builder.MustBuild()
}
