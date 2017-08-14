package trigger

import (
	"errors"
	"sync"
)

// Factory builds a trigger
type Factory interface {
	Build(map[string]string) (Trigger, error)
}

// Builder builds the specifc trigger type
type Builder interface {
	Build(typ string, opts map[string]string) (Trigger, error)
}

var factories map[string]Factory
var rw sync.RWMutex

type defaultBuilder struct{}

func (d defaultBuilder) Build(typ string, opt map[string]string) (Trigger, error) {
	return Build(typ, opt)
}

var DefaultBuilder Builder = defaultBuilder{}

// Register registers a new built-in trigger factory
func Register(name string, f Factory) {
	rw.Lock()
	defer rw.Unlock()

	if _, ok := factories[name]; ok {
		panic("Trigger already registered")
	}

	factories[name] = f
}

// Build builds the trigger with the given name
func Build(name string, opts map[string]string) (Trigger, error) {
	rw.RLock()
	defer rw.RUnlock()

	factory, ok := factories[name]
	if !ok {
		return nil, errors.New("unknown trigger type")
	}

	return factory.Build(opts)
}

func init() {
	factories = make(map[string]Factory)
}
