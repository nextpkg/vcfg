// Package xviper provides an enhanced Viper configuration management.
package viper

import (
	"fmt"

	"github.com/nextpkg/vcfg/ce"
	"github.com/spf13/viper"
)

// Viper is a wrapper around the viper.Viper struct.
type Viper struct {
	*viper.Viper
}

// New creates a new instance of the Viper wrapper.
func New() *Viper {
	return &Viper{
		Viper: viper.New(),
	}
}

// Merge merges the settings from another Viper instance into the current one.
// If the source is nil, it does nothing and returns nil.
// Returns an error if the merge operation fails.
func (target *Viper) Merge(source *Viper) error {
	if source == nil {
		return nil
	}

	err := target.MergeConfigMap(source.AllSettings())
	if err != nil {
		return fmt.Errorf("%w: %w", ce.ErrMergeConfigFailed, err)
	}

	return nil
}
