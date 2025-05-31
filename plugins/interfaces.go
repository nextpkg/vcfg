package plugins

type (
	// Plugin defines the generic plugin interface
	// This is a type-safe plugin interface that provides hot reload capability with minimal boilerplate
	Plugin interface {
		// Name returns the unique name of the plugin
		Name() string

		// Start initializes the plugin with configuration
		// This method is called when the plugin is first loaded
		Start(config any) error

		// Reload is called when plugin configuration changes
		// The plugin should gracefully update its behavior based on the new configuration
		Reload(config any) error

		// Stop gracefully shuts down the plugin
		Stop() error
	}

	// Config defines the generic plugin configuration interface
	Config interface {
		// Name returns the unique name of the plugin
		Name() string
	}
)
