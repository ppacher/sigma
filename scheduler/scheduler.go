package scheduler

import (
	"errors"
	"sync"
	"time"

	"github.com/golang/glog"
	uuid "github.com/satori/go.uuid"

	"golang.org/x/net/context"

	"github.com/homebot/core/event"
	"github.com/homebot/core/urn"
	"github.com/homebot/sigma"
	"github.com/homebot/sigma/function"
	"github.com/homebot/sigma/node"
	"github.com/homebot/sigma/trigger"
)

// NodeInstance describes a node instance
type NodeInstance struct {
	// URN is the URN of the node
	URN urn.URN

	// State is the current state of the node
	State node.State

	// Stats holds statistics for the node
	Stats node.Stats
}

// FunctionRegistration describes a function registered at the scheduler
type FunctionRegistration struct {
	// URN holds the URN of the function resource
	URN urn.URN

	// Spec holds the function specification this controller is for
	Spec sigma.FunctionSpec

	// Nodes holds a list of nodes baking the function
	Nodes []NodeInstance
}

// Scheduler creates, manages and destroys function controllers
type Scheduler interface {
	urn.Resource

	// Create creates a new function controller for the spec
	Create(context.Context, sigma.FunctionSpec) (urn.URN, error)

	// Destroy destroys the function controller for the URN
	Destroy(context.Context, urn.URN) error

	// Dispatch dispatches an event to a function and returns the result
	Dispatch(context.Context, urn.URN, sigma.Event) (urn.URN, []byte, error)

	// Functions returns a list of functions registered at the scheduler
	Functions(context.Context) ([]FunctionRegistration, error)

	// Inspec inspects a function and returns details and statistics about
	// the function controller
	Inspect(context.Context, urn.URN) (FunctionRegistration, error)
}

type scheduler struct {
	id        string
	namespace string
	deployer  node.Deployer

	mu          sync.Mutex
	controllers map[string]function.Controller
}

func (s *scheduler) URN() urn.URN {
	return urn.SigmaInstanceResource.BuildURN(s.namespace, "", s.id)
}

// NewScheduler creates a new scheduler using the provided deployer
// and namespace
func NewScheduler(d node.Deployer, opts ...Option) (Scheduler, error) {
	s := &scheduler{
		id:          uuid.NewV4().String(),
		deployer:    d,
		controllers: make(map[string]function.Controller),
	}

	for _, fn := range opts {
		if err := fn(s); err != nil {
			return nil, err
		}
	}

	return s, nil
}

// Inspect inspects a function and returns details about the controller
func (s *scheduler) Inspect(ctx context.Context, u urn.URN) (FunctionRegistration, error) {
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
		reg, err := s.inspect(ctx, ctrl.URN())
		if err != nil {
			continue
		}

		res = append(res, reg)
	}

	return res, nil
}

// Create registeres a new function spec at the scheduler
func (s *scheduler) Create(ctx context.Context, spec sigma.FunctionSpec) (urn.URN, error) {
	u := urn.URN("")

	accountID, ok := ctx.Value("accountId").(string)
	if !ok {
		accountID = ""
	}

	opts := []function.ControllerOption{
		function.WithAccountID(accountID),
		function.WithNamespace(s.namespace),
		function.WithScalingPolicies(spec.Policies),
		function.WithEventDispatcher(event.NewNopDispatcher(true)),
		function.WithControlLoopInterval(10 * time.Second),
		function.WithDeployer(s.deployer),
		function.WithTriggerBuilder(trigger.DefaultBuilder),
	}

	ctrl, err := function.NewController(spec, opts...)
	if err != nil {
		return u, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok = s.controllers[ctrl.URN().String()]
	if ok {
		return ctrl.URN(), errors.New("function already created")
	}

	s.controllers[ctrl.URN().String()] = ctrl
	if err := ctrl.Start(); err != nil {
		return u, err
	}

	return ctrl.URN(), nil
}

// Destroy destroys the function controller and all nodes
func (s *scheduler) Destroy(ctx context.Context, u urn.URN) error {
	s.mu.Lock()
	ctrl, ok := s.controllers[u.String()]
	delete(s.controllers, u.String())
	s.mu.Unlock()

	if !ok {
		return errors.New("unknown function")
	}

	if err := ctrl.Stop(); err != nil {
		glog.Error("failed to stop function controller ", err)
	}
	if err := ctrl.DestroyAll(); err != nil {
		glog.Error("failed to destroy function nodes ", err)
		return err
	}

	return nil
}

// Dispatch dispatches an event to the function controller and returns the result
// of the function
func (s *scheduler) Dispatch(ctx context.Context, u urn.URN, event sigma.Event) (urn.URN, []byte, error) {
	s.mu.Lock()
	ctrl, ok := s.controllers[u.String()]
	s.mu.Unlock()

	if !ok {
		return urn.URN(""), nil, errors.New("unknown function")
	}

	return ctrl.Dispatch(event)
}

func (s *scheduler) inspect(ctx context.Context, u urn.URN) (FunctionRegistration, error) {
	reg := FunctionRegistration{
		URN: u,
	}

	ctrl, ok := s.controllers[u.String()]
	if !ok {
		return reg, errors.New("unknown function")
	}

	states := ctrl.Nodes()
	stats := ctrl.Stats()

	for key, value := range states {
		n := NodeInstance{
			URN:   key,
			State: value,
			Stats: stats[key],
		}

		reg.Nodes = append(reg.Nodes, n)
	}

	reg.Spec = ctrl.FunctionSpec()

	return reg, nil
}
