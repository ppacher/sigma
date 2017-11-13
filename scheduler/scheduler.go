package scheduler

import (
	"errors"
	"sync"
	"time"

	uuid "github.com/satori/go.uuid"

	"golang.org/x/net/context"

	"github.com/homebot/core/event"
	"github.com/homebot/core/resource"
	"github.com/homebot/insight/logger"
	"github.com/homebot/sigma"
	"github.com/homebot/sigma/function"
	"github.com/homebot/sigma/node"
	"github.com/homebot/sigma/trigger"
)

// NodeInstance describes a node instance
type NodeInstance struct {
	// Name is the Name of the node
	Name resource.Name

	// State is the current state of the node
	State node.State

	// Stats holds statistics for the node
	Stats node.Stats
}

// FunctionRegistration describes a function registered at the scheduler
type FunctionRegistration struct {
	// Name holds the name of the function resource
	Name resource.Name

	// Spec holds the function specification this controller is for
	Spec sigma.FunctionSpec

	// Nodes holds a list of nodes baking the function
	Nodes []NodeInstance
}

// Scheduler creates, manages and destroys function controllers
type Scheduler interface {
	resource.Resource

	// Create creates a new function controller for the spec
	Create(context.Context, sigma.FunctionSpec) (string, error)

	// Destroy destroys the function controller for the URN
	Destroy(context.Context, string) error

	// Dispatch dispatches an event to a function and returns the result
	Dispatch(context.Context, string, sigma.Event) (string, []byte, error)

	// Functions returns a list of functions registered at the scheduler
	Functions(context.Context) ([]FunctionRegistration, error)

	// Inspec inspects a function and returns details and statistics about
	// the function controller
	Inspect(context.Context, resource.Name) (FunctionRegistration, error)
}

type scheduler struct {
	id        resource.Name
	namespace string
	deployer  node.Deployer

	log logger.Logger

	mu          sync.Mutex
	controllers map[string]function.Controller
}

func (s *scheduler) Name() resource.Name {
	return s.id
}

// NewScheduler creates a new scheduler using the provided deployer
// and namespace
func NewScheduler(d node.Deployer, opts ...Option) (Scheduler, error) {
	s := &scheduler{
		id:          resource.Name(uuid.NewV4().String()),
		deployer:    d,
		controllers: make(map[string]function.Controller),
	}

	for _, fn := range opts {
		if err := fn(s); err != nil {
			return nil, err
		}
	}

	if s.log == nil {
		s.log = logger.NopLogger{}
	}

	return s, nil
}

// Inspect inspects a function and returns details about the controller
func (s *scheduler) Inspect(ctx context.Context, u resource.Name) (FunctionRegistration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.inspect(ctx, u)
}

// Functions returns a list of function registered at the controller
func (s *scheduler) Functions(ctx context.Context) ([]FunctionRegistration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var res []FunctionRegistration

	for _, ctrl := range s.controllers {
		reg, err := s.inspect(ctx, ctrl.Name())
		if err != nil {
			continue
		}

		res = append(res, reg)
	}

	return res, nil
}

// Create registeres a new function spec at the scheduler
func (s *scheduler) Create(ctx context.Context, spec sigma.FunctionSpec) (string, error) {
	u := ""

	opts := []function.ControllerOption{
		function.WithScalingPolicies(spec.Policies),
		function.WithEventDispatcher(event.NewNopDispatcher(true)),
		function.WithControlLoopInterval(10 * time.Second),
		function.WithDeployer(s.deployer),
		function.WithTriggerBuilder(trigger.DefaultBuilder),
	}

	log := s.log.WithResource(spec.ID)

	ctrl, err := function.NewController(spec, opts...)
	if err != nil {
		log.Errorf("failed to create controller")
		return u, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.controllers[ctrl.Name().String()]
	if ok {
		log.Errorf("function already created")
		return ctrl.Name().String(), errors.New("function already created")
	}

	s.controllers[ctrl.Name().String()] = ctrl
	if err := ctrl.Start(); err != nil {
		log.Errorf("failed to start controller")
		return u, err
	}

	log.Infof("successfully created function")
	return ctrl.Name().String(), nil
}

// Destroy destroys the function controller and all nodes
func (s *scheduler) Destroy(ctx context.Context, u string) error {
	log := s.log.WithResource(u)

	s.mu.Lock()
	ctrl, ok := s.controllers[u]
	delete(s.controllers, u)
	s.mu.Unlock()

	if !ok {
		log.Errorf("unknown function")
		return errors.New("unknown function")
	}

	if err := ctrl.Stop(); err != nil {
		log.Errorf("failed to stop function controller: %s", err)
	}
	if err := ctrl.DestroyAll(); err != nil {
		log.Errorf("failed to destroy function nodes: %s", err)
		return err
	}

	log.Infof("function destroyed")

	return nil
}

// Dispatch dispatches an event to the function controller and returns the result
// of the function
func (s *scheduler) Dispatch(ctx context.Context, u string, event sigma.Event) (string, []byte, error) {
	log := s.log.WithResource(u)

	s.mu.Lock()
	ctrl, ok := s.controllers[u]
	s.mu.Unlock()

	if !ok {
		log.Errorf("unknown function")
		return "", nil, errors.New("unknown function")
	}

	start := time.Now()
	node, res, err := ctrl.Dispatch(event)

	duration := time.Now().Sub(start)

	if err != nil {
		log.Errorf("function execution failed: %s", err)
	} else {
		log.Infof("function executed in %s", duration)
	}

	return node, res, err
}

func (s *scheduler) inspect(ctx context.Context, u resource.Name) (FunctionRegistration, error) {
	reg := FunctionRegistration{
		Name: u,
	}

	ctrl, ok := s.controllers[u.String()]
	if !ok {
		return reg, errors.New("unknown function")
	}

	states := ctrl.Nodes()
	stats := ctrl.Stats()

	for key, value := range states {
		n := NodeInstance{
			Name:  resource.Name(key),
			State: value,
			Stats: stats[key],
		}

		reg.Nodes = append(reg.Nodes, n)
	}

	reg.Spec = ctrl.FunctionSpec()

	return reg, nil
}
