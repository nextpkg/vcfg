// Package koanf provides a wrapper around the koanf configuration library.
package koanf

import (
	"fmt"

	"github.com/knadh/koanf/v2"
	"github.com/nextpkg/vcfg/ce"
)

// Koanf is a wrapper around the koanf.Koanf struct.
// It embeds koanf.Koanf to provide direct access to all its methods.
type Koanf struct {
	*koanf.Koanf
}

// New creates a new instance of the Koanf wrapper.
func New() *Koanf {
	return &Koanf{
		Koanf: koanf.New("."),
	}
}

// Merge merges the settings from another Koanf instance into the current one.
// If the source is nil, it does nothing and returns nil.
// Returns an error if the merge operation fails.
func (target *Koanf) Merge(source *Koanf) error {
	if source == nil {
		return nil
	}

	if err := target.Koanf.Merge(source.Koanf); err != nil {
		return fmt.Errorf("%w: %w", ce.ErrMergeConfigFailed, err)
	}
	return nil
}

// Unmarshal unmarshals the configuration into the provided struct.
// This is a convenience method that calls the embedded Koanf's Unmarshal with empty path.
func (k *Koanf) Unmarshal(v interface{}) error {
	return k.Koanf.Unmarshal("", v)
}
