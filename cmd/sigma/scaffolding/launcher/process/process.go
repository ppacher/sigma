package process

import (
	"fmt"

	"github.com/homebot/sigma/cmd/sigma/config"
	"github.com/homebot/sigma/cmd/sigma/scaffolding"
)

type Scaffolder func(typ string) error

type TypeConfig struct {
	Scaffolder Scaffolder

	config.ProcessTypeConfig
}

var supportedTypes = map[string]TypeConfig{
	"js": TypeConfig{
		ProcessTypeConfig: config.ProcessTypeConfig{
			Command: []string{"node"},
		},
	},
}

// LauncherFactory scaffolds the configuration for a process based launcher
type LauncherFactory struct{}

// Create implements scaffolding.LauncherFactory
func (LauncherFactory) Create(c *config.Config, types []string) error {
	cfg := *c

	cfg.Launchers.Process = &config.ProcessLauncherConfig{
		Types: make(map[string]config.ProcessTypeConfig),
	}
	for _, t := range types {
		f, ok := supportedTypes[t]
		if !ok {
			return fmt.Errorf("node type %q not supported by process launcher", t)
		}
		if f.Scaffolder != nil {
			if err := f.Scaffolder(t); err != nil {
				return err
			}
		}

		cfg.Launchers.Process.Types[t] = f.ProcessTypeConfig
	}

	*c = cfg
	return nil
}

func init() {
	scaffolding.Register("process", &LauncherFactory{})
}
