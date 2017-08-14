package function

import (
	"errors"
	"time"

	"github.com/homebot/sigma/node"
	"github.com/homebot/sigma/trigger"

	"github.com/homebot/core/event"
	"github.com/homebot/core/urn"
	"github.com/homebot/sigma/autoscale"
)

// ControllerOption defines some configuration options for a function
// controller
type ControllerOption func(c *controller) error

// WithEventDispatcher configures the event dispatcher to be used by the
// newly created function controller
func WithEventDispatcher(disp event.Dispatcher) ControllerOption {
	return func(c *controller) error {
		c.event = disp
		return nil
	}
}

// WithNamespace sets the namespace the function ID belongs to
func WithNamespace(ns string) ControllerOption {
	return func(c *controller) error {
		c.ctx.Namespace = ns
		return nil
	}
}

// WithAccountID sets the account ID the function controller belongs to
func WithAccountID(id string) ControllerOption {
	return func(c *controller) error {
		c.ctx.AccountID = id
		return nil
	}
}

// WithResourceContext sets the resource context for the function controller
func WithResourceContext(ctx urn.ResourceContext) ControllerOption {
	return func(c *controller) error {
		c.ctx = ctx
		return nil
	}
}

// WithControlLoopInterval configures the interval for the function controllers
// control loop
func WithControlLoopInterval(duration time.Duration) ControllerOption {
	return func(c *controller) error {
		c.controlLoopInterval = duration
		return nil
	}
}

// WithAutoScaler sets the auto-scaler to use for the function controller
// if another auto-scaler is already attached, an error is returned
func WithAutoScaler(scaler autoscale.AutoScaler) ControllerOption {
	return func(c *controller) error {
		if c.autoScaler != nil {
			return errors.New("auto-scaler already attached")
		}

		c.autoScaler = scaler
		return nil
	}
}

// WithAttachedScalingPolicy attaches a scaling policy to the functions
// auto-scaler
func WithAttachedScalingPolicy(name string, policy autoscale.Policy) ControllerOption {
	return func(c *controller) (err error) {
		if c.autoScaler == nil {
			c.autoScaler, err = autoscale.NewAutoScaler(nil)

			if err != nil {
				return
			}
		}

		return c.autoScaler.AttachPolicy(name, policy)
	}
}

// WithScalingPolicy builds a scaling policy for the given name and options
// and attaches it to the function controller auto scaler
func WithScalingPolicy(name string, opts map[string]string) ControllerOption {
	return func(c *controller) (err error) {
		if c.autoScaler == nil {
			c.autoScaler, err = autoscale.NewAutoScaler(nil)
		}

		policy, err := autoscale.Build(name, opts)
		if err != nil {
			return
		}

		return c.autoScaler.AttachPolicy(name, policy)
	}
}

// WithScalingPolicies builds a scaling policy for each entry in the map (using the value as config options)
// and attaches the policies to the function controllers' auto scaler
func WithScalingPolicies(policies map[string]map[string]string) ControllerOption {
	return func(c *controller) (err error) {
		if c.autoScaler == nil {
			c.autoScaler, err = autoscale.NewAutoScaler(policies)
			return
		}

		for name, opts := range policies {
			if err := WithScalingPolicy(name, opts)(c); err != nil {
				return err
			}
		}

		return nil
	}
}

// WithDeployer sets the instance deployer to use
func WithDeployer(d node.Deployer) ControllerOption {
	return func(c *controller) error {
		c.deployer = d
		return nil
	}
}

// WithTriggerBuilder sets the trigger builder to use
func WithTriggerBuilder(b trigger.Builder) ControllerOption {
	return func(c *controller) error {
		c.triggerBuilder = b
		return nil
	}
}
