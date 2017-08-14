package scaffolding

import (
	"errors"
	"sync"

	"github.com/homebot/sigma/cmd/sigma/config"
)

// LauncherFactory scaffolds creation for a launcher
type LauncherFactory interface {
	Create(c *config.Config, types []string) error
}

var rw sync.Mutex
var factories map[string]LauncherFactory

// Register registers a new launcher factory
func Register(name string, f LauncherFactory) {
	rw.Lock()
	defer rw.Unlock()

	if _, ok := factories[name]; ok {
		panic("launcher factory already registered")
	}

	factories[name] = f
}

// CreateLauncher scaffolds a launcher configuration
func CreateLauncher(name string, config *config.Config, types []string) error {
	rw.Lock()
	defer rw.Unlock()

	f, ok := factories[name]
	if !ok {
		return errors.New("launcher type not supported")
	}

	return f.Create(config, types)
}

func init() {
	factories = make(map[string]LauncherFactory)
}
