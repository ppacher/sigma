package autoscale

import (
	"fmt"
	"sync"

	"github.com/homebot/core/urn"
	"github.com/homebot/sigma/node"
)

// ScaleDirection represents the direction for the autoscaler
type ScaleDirection int

// Scaling directions
const (
	ScaleNop = ScaleDirection(iota)
	ScaleUp
	ScaleDown
)

// Policy decided if the given function controller should be
// scaled up or down
type Policy interface {
	// Check checks the given metric values and node states and decides if
	// the node controller should be scaled up or down. It can even suggest
	// how much instances should be created or destroyed in absolute or relative (%)
	// values. In case multiple scaling policies are attached to an auto-scaler,
	// the auto-scaler will always follow the scaling policy that has the most
	// positive impact on the number of instances. That is, if two policies are
	// attached and one returns (Up, 1, true) while the other returns (Up, 20, false) (on a 10 node controller)
	// the autoscaller will create 2 (0.20 * 10) new nodes
	Check(metrics map[string]float64, states map[urn.URN]node.State) (direction ScaleDirection, amount int, abs bool)
}

// PolicyFactory creates a new Policy and
// receives an optional configuration map
type PolicyFactory func(map[string]string) (Policy, error)

type factories struct {
	rw        sync.Mutex
	factories map[string]PolicyFactory
}

// Register registers a new PolicyFactory for the given name
func (f *factories) Register(name string, factory PolicyFactory) {
	f.rw.Lock()
	defer f.rw.Unlock()

	if _, ok := f.factories[name]; ok {
		panic(fmt.Sprintf("scaling policy factory with name %q already registered", name))
	}

	f.factories[name] = factory
}

// Build builds the scaling policy for the given name and options
func (f *factories) Build(name string, opts map[string]string) (Policy, error) {
	f.rw.Lock()
	defer f.rw.Unlock()

	factory, ok := f.factories[name]
	if !ok {
		return nil, fmt.Errorf("unknown scaling policy")
	}

	return factory(opts)
}

var defaultFactories *factories

// Register registers a new auto scaling policy factory
func Register(name string, factory PolicyFactory) {
	defaultFactories.Register(name, factory)
}

// Build build the scaling policy with the given name and options
func Build(name string, opts map[string]string) (Policy, error) {
	return defaultFactories.Build(name, opts)
}

func init() {
	defaultFactories = &factories{
		factories: make(map[string]PolicyFactory),
	}
}
