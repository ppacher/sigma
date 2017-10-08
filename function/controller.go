package function

import (
	"context"
	"errors"
	"io"
	"reflect"
	"sync"
	"time"

	"github.com/satori/go.uuid"

	"github.com/homebot/core/event"
	"github.com/homebot/core/utils"
	"github.com/homebot/insight/logger"
	sigmaV1 "github.com/homebot/protobuf/pkg/api/sigma/v1"
	"github.com/homebot/sigma"
	"github.com/homebot/sigma/autoscale"
	"github.com/homebot/sigma/metrics"
	"github.com/homebot/sigma/node"
	"github.com/homebot/sigma/trigger"
)

var (
	// ErrNotRunning is returned when the controller registry is assumed to have
	// been started
	ErrNotRunning = errors.New("controller registry not running")

	// ErrRunning is returned  when the controller registry is assumed to be stopped
	ErrRunning = errors.New("controller registry already running")

	// ErrUnknownController is returned when the node controller in question does not
	// exist
	ErrUnknownController = errors.New("unknown node controller")

	// ErrNoSelectableNodes is returned when no nodes could have been selected
	ErrNoSelectableNodes = errors.New("no selectable nodes")

	// ErrHookRegistered is returned when the given control loop hook is already
	// registered
	ErrHookRegistered = errors.New("control loop hook already registered")

	// ErrUnknownHook is returned when the control loop hook in question does not
	// exist on the function controller
	ErrUnknownHook = errors.New("unknown control loop hook")

	// ErrMissingDeployer is returned when a auto-scaler is configured but no node
	// launcher has been set
	ErrMissingDeployer = errors.New("auto-scaling can only be used with a node launcher")
)

// ControlLoopHook is executed during each interation of the function controllers
// control loop
type ControlLoopHook func(c Controller)

// Controller handles all node controller for a given
// function spec.
type Controller interface {
	//urn.Resource
	URN() string

	// Start starts the function controller's control loop
	Start() error

	// Stop stops the function controller control loop
	Stop() error

	// DestroyAll destroys all node controllers
	DestroyAll() error

	// AddNodeController creates a new controller for the given node
	AddNodeController(node.Controller) error

	// DestroyNode destroys the given controller
	DestroyNode(string) error

	// Nodes returns the state for all controllers registered
	Nodes() map[string]node.State

	// Stats returns a map with current node statistics
	Stats() map[string]node.Stats

	// FunctionSpec returns the function specification for node controllers
	// managed by this registry
	FunctionSpec() sigma.FunctionSpec

	// Dispatch dispatches an event to one of the function nodes and returns
	// the ID of the selected node, the result and any error encountered
	Dispatch(event sigma.Event) (string, []byte, error)

	// AttachControlLoopHook attaches a new control loop hook to be executed
	// on each interation of the function controller control loop
	AttachControlLoopHook(hook ControlLoopHook) error

	// DetachControlLoopHook removes a control loop hook from the function
	// controller
	DetachControlLoopHook(hook ControlLoopHook) error
}

type controller struct {
	spec sigma.FunctionSpec

	//ctx urn.ResourceContext

	event          event.Dispatcher
	deployer       node.Deployer
	triggerBuilder trigger.Builder

	triggers map[string]trigger.Trigger

	// registered controllers
	rw          sync.RWMutex
	controllers map[string]node.Controller

	// control loop management
	stop chan struct{}
	wg   sync.WaitGroup

	controlLoopInterval time.Duration
	autoScaler          autoscale.AutoScaler
	metrics             *metrics.Metrics

	l logger.Logger

	hookLock sync.RWMutex
	hooks    []ControlLoopHook
}

func (ctrl *controller) URN() string {
	return ctrl.spec.ID
}

// Start starts the function controllers' control loop
func (ctrl *controller) Start() error {
	ctrl.rw.Lock()
	defer ctrl.rw.Unlock()

	if ctrl.stop != nil {
		return ErrRunning
	}
	ctrl.stop = make(chan struct{})

	if ctrl.triggerBuilder != nil {
		for _, spec := range ctrl.spec.Triggers {
			t, err := ctrl.triggerBuilder.Build(spec.Type, spec.Options)
			if err != nil {
				for _, t := range ctrl.triggers {
					t.Close()
				}
				return err
			}

			ctrl.triggers[spec.Type] = t

			ctrl.wg.Add(1)
			go ctrl.handleTrigger(t, spec, ctrl.spec.Parameteres)
		}

	}

	ctrl.wg.Add(1)
	go ctrl.controlLoop(ctrl.stop)

	return nil
}

// TODO(homebot): add logging
func (ctrl *controller) handleTrigger(t trigger.Trigger, tSpec sigma.TriggerSpec, values utils.ValueMap) {
	defer ctrl.wg.Done()

	for {
		evt, err := t.Next()
		if err != nil && err == io.EOF {
			return
		}

		ok, err := trigger.Evaluate(tSpec.Condition, evt, values)
		if ok && err == nil {
			_, res, err := ctrl.Dispatch(evt)
			if err != nil {
				ctrl.l.Errorf("failed to dispatch trigger event %q: %s", evt.Type(), err)
			} else {
				ctrl.l.Infof("dispatched trigger event %q: %s", evt.Type(), string(res))
			}
		} else if err != nil {
			ctrl.l.Errorf("trigger spec %q: failed to evaluate condition %q: %s", tSpec.Type, tSpec.Condition, err)
		} else {
			ctrl.l.Debugf("trigger spec %s: condition not satisfied for event %q", tSpec.Type, evt.Type())
		}
	}
}

// Stop stops the function controller control loop
func (ctrl *controller) Stop() error {
	ctrl.rw.Lock()
	stop := ctrl.stop
	ctrl.stop = nil
	ctrl.rw.Unlock()

	if stop == nil {
		return ErrNotRunning
	}

	close(stop)

	ctrl.wg.Wait()

	ctrl.rw.Lock()
	defer ctrl.rw.Unlock()

	var first error
	for _, t := range ctrl.triggers {
		if err := t.Close(); err != nil && first == nil {
			first = err
		}
	}

	return first
}

// DestroyAll destroys all controllers
func (ctrl *controller) DestroyAll() error {
	ctrl.rw.Lock()
	defer ctrl.rw.Unlock()

	ctrl.l.Infof("destroying all nodes")

	// TODO(homebot) return a "multi-error"
	var firstErr error
	for key, node := range ctrl.controllers {
		if err := node.Close(); err != nil && firstErr == nil {
			firstErr = err
			ctrl.l.Warnf("failed to destroy node %s: %s", key, err)
		} else if err != nil {
			ctrl.l.Warnf("failed to destroy node %s: %s", key, err)
		}
		delete(ctrl.controllers, key)
	}

	return firstErr
}

// AddNodeController creates a new controller and appends it to the registry
func (ctrl *controller) AddNodeController(n node.Controller) error {
	ctrl.rw.Lock()
	defer ctrl.rw.Unlock()

	ctrl.controllers[n.URN()] = n

	ctrl.l.Infof("node %s attached to controller", n.URN())

	//ctrl.dispatchEvent(urn.SigmaEventNodeCreated, n.URN().Resource(), nil)

	return nil
}

// DestroyNode destroys the controller with `id`
func (ctrl *controller) DestroyNode(u string) error {
	ctrl.rw.Lock()
	defer ctrl.rw.Unlock()

	node, ok := ctrl.controllers[u]
	if !ok {
		return ErrUnknownController
	}

	ctrl.l.Infof("destroying node %s", u)

	delete(ctrl.controllers, u)

	//ctrl.dispatchEvent(urn.SigmaEventNodeDestroyed, u.Resource(), nil)

	return node.Close()
}

// Nodes returns all controllers and their current state
func (ctrl *controller) Nodes() map[string]node.State {
	ctrl.rw.RLock()
	defer ctrl.rw.RUnlock()

	m := make(map[string]node.State)
	for key, node := range ctrl.controllers {
		m[key] = node.State()
	}

	return m
}

// Stats returns statistics for each node part of this function controller
func (ctrl *controller) Stats() map[string]node.Stats {
	ctrl.rw.RLock()
	defer ctrl.rw.RUnlock()

	m := make(map[string]node.Stats)
	for key, node := range ctrl.controllers {
		m[key] = node.Stats()
	}

	return m
}

// FunctionSpec returns the function spec of the controller registry
func (ctrl *controller) FunctionSpec() sigma.FunctionSpec {
	return ctrl.spec
}

// Dispatch dispatches an event to a healthy and idle controller
func (ctrl *controller) Dispatch(event sigma.Event) (selectedNode string, result []byte, err error) {
	defer func() {
		if err != nil {
			n := selectedNode
			if n == "" {
				n = ctrl.URN()
			}
			//ctrl.dispatchEvent(urn.SigmaEventFunctionFailed, n.Resource(), []byte(err.Error()))
		} else {
			//ctrl.dispatchEvent(urn.SigmaEventFunctionExecuted, selectedNode.Resource(), result)
		}
	}()

	ctrl.rw.RLock()
	defer ctrl.rw.RUnlock()

	for id, node := range ctrl.controllers {
		if node.State().CanSelect() {
			selectedNode = id
			result, err = node.Dispatch(context.Background(), &sigmaV1.DispatchEvent{
				Urn:     id,
				Payload: event.Payload(),
			})

			if err == nil {
				ctrl.l.Infof("dispatched event to %s", selectedNode)
			} else {
				ctrl.l.Warnf("failed to dispatch event: %s (selected-node %s)", err, selectedNode)
			}

			return
		}
	}

	err = ErrNoSelectableNodes

	return
}

// AttachControlLoopHook attaches a new control loop hook to the function controller
func (ctrl *controller) AttachControlLoopHook(hook ControlLoopHook) error {
	ctrl.hookLock.Lock()
	defer ctrl.hookLock.Unlock()

	ptr := reflect.ValueOf(hook).Pointer()

	for _, fn := range ctrl.hooks {
		if reflect.ValueOf(fn).Pointer() == ptr {
			return ErrHookRegistered
		}
	}

	ctrl.l.Infof("attached control loop hook %q", reflect.TypeOf(hook).Name())

	ctrl.hooks = append(ctrl.hooks, hook)
	return nil
}

// DetachControlLoopHook detaches a previously attached control loop hook from the
// function controller
func (ctrl *controller) DetachControlLoopHook(hook ControlLoopHook) error {
	ctrl.hookLock.Lock()
	defer ctrl.hookLock.Unlock()

	ptr := reflect.ValueOf(hook).Pointer()

	for idx, fn := range ctrl.hooks {
		if reflect.ValueOf(fn).Pointer() == ptr {
			ctrl.hooks[idx] = nil
			ctrl.hooks = append(ctrl.hooks[:idx], ctrl.hooks[idx+1:]...)

			return nil
		}
	}

	return ErrUnknownHook
}

// NewController creates a new node controller registry
func NewController(spec sigma.FunctionSpec, opts ...ControllerOption) (Controller, error) {
	ctrl := &controller{
		spec:        spec,
		metrics:     metrics.GetMetrics(),
		controllers: make(map[string]node.Controller),
		triggers:    make(map[string]trigger.Trigger),
	}

	for _, opt := range opts {
		if err := opt(ctrl); err != nil {
			return nil, err
		}
	}

	// last error checks
	if ctrl.autoScaler != nil && ctrl.deployer == nil {
		return nil, ErrMissingDeployer
	}

	if ctrl.l == nil {
		ctrl.l, _ = logger.NewInsightLogger(logger.WithResource(spec.ID))
	}

	return ctrl, nil
}

func (ctrl *controller) runHooks() {
	ctrl.hookLock.RLock()
	defer ctrl.hookLock.RUnlock()

	defer func() {
		if x := recover(); x != nil {
			ctrl.l.Errorf("panic in control loop hook: %#v", x)
		}
	}()

	for _, hook := range ctrl.hooks {
		hook(ctrl)
		ctrl.l.Infof("executed hook %q", reflect.ValueOf(hook).Type().Name())
	}
}

func (ctrl *controller) scaleUp(amount int) {
	ch := make(chan error, amount)

	ctrl.wg.Add(amount)
	for i := 0; i < amount; i++ {
		go ctrl.deployNode(ch)
	}

	for i := 0; i < amount; i++ {
		err := <-ch

		if err != nil {
			ctrl.l.Errorf("failed to deploy node: %s", err)
		}
	}

	close(ch)
}

func (ctrl *controller) deployNode(ch chan error) {
	defer ctrl.wg.Done()

	// Deploying the node should not take longer than 10 seconds
	// TODO(ppacher) make this configuratble
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	ctrl.l.Infof("deploying a new node ...")

	newUrn := uuid.NewV4().String() //urn.SigmaInstanceResource.BuildURN(u.Namespace(), u.AccountID(), fmt.Sprintf("%s/%s", u.Resource(), uuid.NewV4().String()))

	controller, err := ctrl.deployer.Deploy(ctx, newUrn, ctrl.spec)
	if err != nil {
		ch <- err
		return
	}

	if err := ctrl.AddNodeController(controller); err != nil {
		ch <- err
		return
	}

	ch <- nil
}

func (ctrl *controller) scaleDown(amount int) {
	removed := 0
	retries := 0

	for removed < amount {
		nodes := ctrl.Nodes()

		for id, state := range nodes {
			switch state {
			case node.StateActive, node.StateDisabled, node.StateUnhealthy:
				removed++
				if err := ctrl.DestroyNode(id); err != nil {
					l, _ := logger.NewInsightLogger()
					l.Warnf("failed to completely destroy %s: %s", id, err.Error())
				}
			default:
			}

			if removed >= amount {
				return
			}
		}

		// TODO(homebot) make maximum number of destroy-tries configurable
		if retries > 10 {
			return
		}

		if removed < amount {
			// Sleep 100ms before trying to find other nodes to kill
			<-time.After(time.Millisecond * 100)
		}
	}
}

func (ctrl *controller) controlLoop(stop chan struct{}) {
	defer ctrl.wg.Done()

	interval := ctrl.controlLoopInterval
	if interval == time.Duration(0) {
		interval = time.Second * 30
	}

	for {
		// first we shutdown all nodes that are marked as unhealthy
		states := ctrl.Nodes()
		for key, state := range states {
			if !state.IsHealthy() {
				if err := ctrl.DestroyNode(key); err != nil {
					ctrl.l.Warnf("failed to destroy unhealthy node %s: %s", key, err)
				}
			}
		}

		// Next, we'll update the current node statistics
		ctrl.rw.Lock()
		metrics := ctrl.metrics.Update(ctrl.controllers)
		ctrl.rw.Unlock()

		// Now, run the auto-scaler (if we have one)
		if ctrl.autoScaler != nil {
			selected, direction, amount := ctrl.autoScaler.Check(metrics, ctrl.Nodes())

			if direction != autoscale.ScaleNop {
				what := "create"
				if direction == autoscale.ScaleDown {
					what = "remove"
				}
				ctrl.l.Infof("policy %q suggests to %s %d nodes", selected, what, amount)
			}

			switch direction {
			case autoscale.ScaleNop:
				// Nothing to do
			case autoscale.ScaleUp:
				ctrl.scaleUp(amount)
			case autoscale.ScaleDown:
				ctrl.scaleDown(amount)
			}
		}

		// Finally, execute registered control loop hooks
		ctrl.runHooks()

		// Sleep until the next iteration
		select {
		case <-stop:
			return

		case <-time.After(interval):
		}

	}
}

func (ctrl *controller) dispatchEvent(typ string, id string, payload []byte) {
}
