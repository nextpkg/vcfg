package main

import (
	"fmt"
	"time"

	"github.com/nextpkg/vcfg/plugins"
)

func init() {
	plugins.RegisterGlobalPlugin(&DemoPlugin{}, &DemoConfig{})
}

const (
	PluginName = "demo"
)

type (
	DemoConfig struct {
		MaxSize int           `yaml:"max_size" json:"max_size" default:"1000"`
		TTL     time.Duration `yaml:"ttl" json:"ttl" default:"5m"`
	}
	DemoPlugin struct{}
)

// Name implements plugins.Config.
func (d *DemoConfig) Name() string {
	return PluginName
}

func (p *DemoPlugin) Name() string {
	return PluginName
}

func (p *DemoPlugin) Start(x any) error {
	config := x.(*DemoConfig)

	fmt.Printf("🔌 Plugin 'demo' started with cache config: MaxSize=%d, TTL=%v\n", config.MaxSize, config.TTL)
	return nil
}

func (p *DemoPlugin) Reload(x any) error {
	config := x.(*DemoConfig)

	fmt.Printf("🔄 Plugin 'demo' reloaded with cache config: MaxSize=%d, TTL=%v\n", config.MaxSize, config.TTL)
	return nil
}

func (p *DemoPlugin) Stop() error {
	fmt.Println("🛑 Plugin 'demo' stopped")
	return nil
}
