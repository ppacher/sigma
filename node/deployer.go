package node

import (
	"time"

	"github.com/homebot/core/urn"
	"github.com/homebot/sigma"
	"github.com/homebot/sigma/launcher"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/net/context"
)

// Deployer deploys a new node and returns the nodes connection
type Deployer interface {

	// Deploy deploys a new node and returns a controller
	// for managing/communication with the node
	Deploy(context.Context, urn.URN, sigma.FunctionSpec) (Controller, error)
}

// DeployFunc implements Deployer
type DeployFunc func(context.Context, urn.URN, sigma.FunctionSpec) (Controller, error)

// Deploy calls `f` and implements Deployer
func (f DeployFunc) Deploy(ctx context.Context, u urn.URN, spec sigma.FunctionSpec) (Controller, error) {
	return f(ctx, u, spec)
}

type deployer struct {
	service          NodeServer
	launcher         launcher.Launcher
	advertiseAddress string
}

// NewDeployer creates a new node deployer. The new deployer will
// setup `svc` to accept the new node and use `launcher` to create
// a new instance. See `Deploy()` for more information
func NewDeployer(svc NodeServer, launcher launcher.Launcher, handlerAddress string) Deployer {
	if svc == nil {
		panic("NewDeployer(): NodeServer parameter is mandatory")
	}

	if launcher == nil {
		panic("NewDeployer(): Launcher parameter is mandatory")
	}

	return &deployer{
		service:          svc,
		launcher:         launcher,
		advertiseAddress: handlerAddress,
	}
}

// Deploy deploys a new node
func (d *deployer) Deploy(ctx context.Context, u urn.URN, spec sigma.FunctionSpec) (Controller, error) {
	// First we need to setup the NodeServer to accept the new node as soon
	// as it is ready
	secret := uuid.NewV4().String()

	conn, err := d.service.Prepare(u, secret, spec)
	if err != nil {
		return nil, err
	}

	// Next, instruct the launcher to deploy a new instance
	instance, err := d.launcher.Create(ctx, spec.Type, launcher.Config{
		URN:     u,
		Secret:  secret,
		Address: d.advertiseAddress,
	})
	if err != nil {
		d.service.Remove(u)
		return nil, err
	}

	// now the instance has been deployed successfully,
	// we now wait until the instance connects
	for {
		// check if the connection has been registered
		if conn.Registered() {
			break
		}

		select {
		case <-ctx.Done():
			go instance.Stop()
			go d.service.Remove(u)
			return nil, ctx.Err()
		case <-time.After(time.Millisecond * 100):
		}
	}

	ctrl := CreateController(u, instance, conn)

	removeController := func(ctrl Controller) { d.service.Remove(ctrl.URN()) }

	ctrl.OnDestroy(removeController)

	return ctrl, nil
}
