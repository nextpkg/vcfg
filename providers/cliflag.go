package providers

import (
	"fmt"

	"github.com/knadh/koanf/maps"
	"github.com/urfave/cli/v3"
)

// CliFlagsWrapper wraps CLI flags to only include set flags
type CliFlagsWrapper struct {
	cmd   *cli.Command
	delim string
}

// NewCliFlagsProvider creates a new CLI flags provider that only includes set flags
func NewCliFlagsProvider(cmd *cli.Command, delim string) *CliFlagsWrapper {
	return &CliFlagsWrapper{cmd: cmd, delim: delim}
}

// Read implements koanf.Provider interface
func (w *CliFlagsWrapper) Read() (map[string]any, error) {
	out := make(map[string]any)

	// Only process flags that are actually set
	for _, flag := range w.cmd.Flags {
		name := flag.Names()[0]
		if w.cmd.IsSet(name) {
			var value any
			switch flag.(type) {
			case *cli.StringFlag:
				value = w.cmd.String(name)
			case *cli.IntFlag:
				value = w.cmd.Int(name)
			case *cli.BoolFlag:
				value = w.cmd.Bool(name)
			case *cli.Float64Flag:
				value = w.cmd.Float64(name)
			case *cli.DurationFlag:
				value = w.cmd.Duration(name)
			case *cli.StringSliceFlag:
				value = w.cmd.StringSlice(name)
			case *cli.IntSliceFlag:
				value = w.cmd.IntSlice(name)
			default:
				// For other flag types, try to get the value using the Get() method
				if getter, ok := flag.(interface{ Get() interface{} }); ok {
					value = getter.Get()
				} else {
					// Fallback: use reflection to get the value
					continue
				}
			}

			if value != nil {
				out[name] = value
			}
		}
	}

	// Apply delimiter-based nesting if specified
	if w.delim != "" {
		return maps.Unflatten(out, w.delim), nil
	}

	return out, nil
}

// ReadBytes is not supported by the CLI flags wrapper
func (w *CliFlagsWrapper) ReadBytes() ([]byte, error) {
	return nil, fmt.Errorf("ReadBytes is not supported by CLI flags wrapper")
}
