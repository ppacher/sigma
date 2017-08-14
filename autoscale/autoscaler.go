package autoscale

import (
	"errors"
	"sync"

	"github.com/homebot/core/urn"
	"github.com/homebot/sigma/node"
)

var (
	// ErrPolicyAttached is returned if the policy is already attached
	// to the auto scaler
	ErrPolicyAttached = errors.New("policy is already attached")

	// ErrUnknownPolicy is returned if the policy does not exist
	ErrUnknownPolicy = errors.New("policy does not exist")
)

// AutoScaler is responsible for scaling a function controller
type AutoScaler interface {
	// Check checks the current metrics and node controllers and decides if the function controller
	// should be scaled in or out
	Check(metrics map[string]float64, controllers map[urn.URN]node.State) (string, ScaleDirection, int)

	// AttachPolicy attaches a new scaling policy
	AttachPolicy(name string, policy Policy) error

	// DetachPolicy detaches a scaling policy
	DetachPolicy(name string) error
}

type autoScaler struct {
	rw sync.RWMutex

	policies map[string]Policy

	stop chan struct{}
}

// AttachPolicy attaches a new policy to the auto scaler
func (a *autoScaler) AttachPolicy(name string, policy Policy) error {
	a.rw.Lock()
	defer a.rw.Unlock()

	if _, ok := a.policies[name]; ok {
		return ErrPolicyAttached
	}

	a.policies[name] = policy

	return nil
}

// DetachPolicy detaches a policy from the autoscaler
func (a *autoScaler) DetachPolicy(name string) error {
	a.rw.Lock()
	defer a.rw.Unlock()

	if _, ok := a.policies[name]; !ok {
		return ErrUnknownPolicy
	}

	delete(a.policies, name)
	return nil
}

// Check updates the metrics and checks the current state of the controller
// registry
func (a *autoScaler) Check(metrics map[string]float64, states map[urn.URN]node.State) (string, ScaleDirection, int) {
	a.rw.Lock()
	defer a.rw.Unlock()

	running := len(states)

	amount := 0
	direction := ScaleNop
	selected := ""

	if len(a.policies) == 0 {
		active := 0
		for _, state := range states {
			if state == node.StateActive || state == node.StateRunning {
				active++
			}
		}

		if active == 0 {
			direction = ScaleUp
			amount = 1
			selected = "built-in"
		}
	}

	for name, policy := range a.policies {
		d, i, abs := policy.Check(metrics, states)

		if !abs {
			i = (int)((i / 100) * running)
		}

		if d == ScaleUp && d != direction {
			d = ScaleUp
			amount = i
			selected = name
		} else if d == ScaleUp {
			// direction == ScaleUp
			if i > amount {
				amount = i
				selected = name
			}
		} else if d == ScaleDown && direction == ScaleDown {
			// d == direction == ScaleDown
			if i < amount {
				amount = i
				selected = name
			}
		}
	}

	return selected, direction, amount
}

// NewAutoScaler returns a new AutoScaler from the given configuration
func NewAutoScaler(policies map[string]map[string]string) (AutoScaler, error) {
	a := &autoScaler{
		policies: make(map[string]Policy),
	}

	for name, opts := range policies {
		policy, err := Build(name, opts)
		if err != nil {
			return nil, err
		}

		a.policies[name] = policy
	}

	return a, nil
}
