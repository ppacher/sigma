package node

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/homebot/sigma/launcher"

	"golang.org/x/net/context"

	"github.com/homebot/core/urn"
	sigma "github.com/homebot/protobuf/pkg/api/sigma"
)

// Stats holds node instance statistics
type Stats struct {
	// CreatedAt holds the time the node has been created
	CreatedAt time.Time

	// LastInvocation is the last invocation of this node
	LastInvocation time.Time

	// Invocations holds the total number of invocations of this node
	Invocations int64

	// TotalExecTime holds the total number of seconds the instance
	// has been executed (from dispatch to receiving the result)
	TotalExecTime time.Duration

	// MeanExecTime is the mean execution time of the node
	MeanExecTime time.Duration
}

// ToProtobuf creates the protocol buffer representation of the node state
func (s Stats) ToProtobuf() *sigma.NodeStatistics {
	return &sigma.NodeStatistics{
		CreatedAt:          int64(s.CreatedAt.UnixNano()),
		LastInvocation:     s.LastInvocation.UnixNano(),
		Invocations:        s.Invocations,
		TotalExecutionTime: int64(s.TotalExecTime),
		MeanExecutionTime:  int64(s.MeanExecTime),
	}
}

// StatsFromProtobuf creates a node.Stats from it's protocol buffer representation
func StatsFromProtobuf(s *sigma.NodeStatistics) Stats {
	return Stats{
		LastInvocation: time.Unix(0, s.GetLastInvocation()),
		Invocations:    s.GetInvocations(),
		TotalExecTime:  time.Duration(s.GetTotalExecutionTime()),
		MeanExecTime:   time.Duration(s.GetMeanExecutionTime()),
	}
}

// State describes the current state of a node
type State string

// CanSelect returns true if the current state allows the node to be
// selected for event dispatching
func (s State) CanSelect() bool {
	return s == StateActive
}

// IsHealthy returns true if the node is currently marked as healthy
func (s State) IsHealthy() bool {
	return s != StateUnhealthy
}

// ToProtobuf converts the node state to it's protocol buffer representation
func (s State) ToProtobuf() sigma.Node_State {
	return sigma.Node_State(sigma.Node_State_value[strings.ToUpper(string(s))])
}

// StateFromProtobuf coverts a protocol buffer node state
func StateFromProtobuf(s sigma.Node_State) State {
	switch s {
	case sigma.Node_ACTIVE:
		return StateActive
	case sigma.Node_DISABLED:
		return StateDisabled
	case sigma.Node_RUNNING:
		return StateRunning
	case sigma.Node_UNHEALTHY:
		return StateUnhealthy
	default:
		return State(strings.ToLower(s.String()))
	}
}

const (
	// StateActive is set when the node is healthy and can be used
	StateActive = State("active")

	// StateUnhealthy is set when the node is marked as unhealthy and
	// should not be used for scheduling events
	StateUnhealthy = State("unhealthy")

	// StateDisabled is set when the function should not be used for
	// event dispatching
	StateDisabled = State("disabled")

	// StateRunning is set when the node is currently executing
	StateRunning = State("running")
)

// Controller manages a given function node
type Controller interface {
	urn.Resource

	// State returns the current state of the node
	State() State

	// Stats returns some statistics for this node instance controller
	Stats() Stats

	// Dispatch dispatches an event to the node
	Dispatch(context.Context, *sigma.DispatchEvent) ([]byte, error)

	// OnDestroy registers an on-destroy handler
	OnDestroy(func(Controller))

	// Close closes the connection to the node and stops the instance
	Close() error
}

type controller struct {
	id  string
	urn urn.URN

	router   Router
	instance launcher.Instance

	rw        sync.RWMutex
	state     State
	stats     Stats
	onDestroy []func(Controller)
}

func (ctrl *controller) OnDestroy(f func(Controller)) {
	ctrl.rw.Lock()
	defer ctrl.rw.Unlock()

	ctrl.onDestroy = append(ctrl.onDestroy, f)
}

// State returns the current state of the node
func (ctrl *controller) State() State {
	ctrl.rw.RLock()
	defer ctrl.rw.RUnlock()

	if err := ctrl.instance.Healthy(); err != nil {
		return StateUnhealthy
	}

	return ctrl.state
}

func (ctrl *controller) URN() urn.URN {
	return ctrl.urn
}

// Dispatch dispatches the given event to the node and returns
// the execution result
func (ctrl *controller) Dispatch(ctx context.Context, event *sigma.DispatchEvent) ([]byte, error) {
	start := time.Now()

	ctrl.setState(StateRunning)

	res, err := ctrl.router.Dispatch(ctx, event)
	if err != nil {
		ctrl.setState(StateUnhealthy)
		return nil, err
	}
	ctrl.setState(StateActive)

	execTime := time.Now().Sub(start)

	defer func() {
		ctrl.rw.Lock()
		defer ctrl.rw.Unlock()

		ctrl.stats.LastInvocation = start
		ctrl.stats.Invocations++
		ctrl.stats.TotalExecTime += execTime
		ctrl.stats.MeanExecTime = time.Duration(int64(ctrl.stats.TotalExecTime) / ctrl.stats.Invocations)
	}()

	switch v := res.GetExecutionResult().(type) {
	case *sigma.ExecutionResult_Error:
		return nil, errors.New(v.Error)
	case *sigma.ExecutionResult_Result:
		return v.Result, nil
	default:
		return nil, fmt.Errorf("unexpected result: %#v", v)
	}
}

func (ctrl *controller) Stats() Stats {
	ctrl.rw.RLock()
	defer ctrl.rw.RUnlock()

	return ctrl.stats
}

// Close closes the connection to the node and removes the node instance
func (ctrl *controller) Close() error {
	ctrl.rw.Lock()
	defer ctrl.rw.Unlock()

	for _, fn := range ctrl.onDestroy {
		fn(ctrl)
	}

	ctrl.instance.Stop()
	return ctrl.router.Close()
}

func (ctrl *controller) setState(s State) {
	ctrl.rw.Lock()
	defer ctrl.rw.Unlock()

	ctrl.state = s
}

// CreateController creates a new controller for the given node
func CreateController(u urn.URN, instance launcher.Instance, conn Conn) Controller {
	return &controller{
		urn:      u,
		router:   NewRouter(conn),
		instance: instance,
		state:    StateActive,
	}
}
