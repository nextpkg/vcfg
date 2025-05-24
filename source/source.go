package source

import (
	"github.com/nextpkg/vcfg/viper"
)

// Source interface defines the configuration source
type Source interface {
	Read() (*viper.Viper, error)
	String() string
	// Watch monitors configuration changes
	// and notifies via a callback function when changes occur
	Watch() (<-chan struct{}, error)
	// Stop stops monitoring configuration changes
	Stop() error
}
