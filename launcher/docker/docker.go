package docker

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/homebot/sigma/launcher"
	"github.com/moby/moby/client"
)

// NodeConfig configures an image and execution
// context for a given exec-type
type NodeConfig struct {
	// Image holds the name of the image to start
	Image string `json:"image" yaml:"image"`
}

// Config is the configuration for a docker launcher
type Config struct {
	Types map[string]NodeConfig `json:"types" yaml:"types"`
}

// Launcher is a sigma node launcher based on Docker
// It implements the github.com/homebot/sigma/launcher.Launcher
// interface
type Launcher struct {
	cli *client.Client
	cfg Config
}

// New creates a new launcher and docker client from
// environment variables
func New(cfg Config) (*Launcher, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}

	return NewWithClient(cfg, cli)
}

// NewWithClient creates a new docker launcher with
// the given moby client
func NewWithClient(cfg Config, cli *client.Client) (*Launcher, error) {
	return &Launcher{
		cli: cli,
		cfg: cfg,
	}, nil
}

// Create creates a new sigma node by scheduling the registered
// docker image. It implements github.com/homebot/sigma/launcher.Launcher
// interface
func (l *Launcher) Create(ctx context.Context, typ string, config launcher.Config) (launcher.Instance, error) {
	cfg, ok := l.cfg.Types[typ]
	if !ok {
		return nil, errors.New("unknown execution type")
	}

	launcherConfig := &container.Config{
		Image: cfg.Image,
		Env:   config.Env(),
	}

	res, err := l.cli.ContainerCreate(ctx, launcherConfig, nil, nil, "")
	if err != nil {
		return nil, err
	}
	log.Printf("[docker] created container %s\n", res.ID)

	for _, w := range res.Warnings {
		log.Printf("[docker] WARNING: %s\n", w)
	}

	// finally, start up the container
	log.Printf("[docker] starting container %s\n", res.ID)
	if err := l.cli.ContainerStart(ctx, res.ID, types.ContainerStartOptions{}); err != nil {
		log.Printf("[docker] failed to start container: %s\n", err)
		defer func() {
			if err := l.cli.ContainerRemove(context.Background(), res.ID, types.ContainerRemoveOptions{
				Force: true,
			}); err != nil {
				log.Printf("[docker] ERROR: failed to clean up container: %s\n", err)
			}
		}()
		return nil, err
	}
	log.Printf("[docker] container started successfully: %s\n", res.ID)

	return &Instance{
		id:       res.ID,
		launcher: l,
	}, nil
}

// Instance represents a sigma function node instance
// running in a docker container. It implements the
// github.com/homebot/sigma/launcher.Instance interface
type Instance struct {
	id       string
	launcher *Launcher
}

// Healthy returns nil if the container is healthy
func (i *Instance) Healthy() error {
	inspect, err := i.launcher.cli.ContainerInspect(context.Background(), i.id)
	if err != nil {
		return err
	}

	if !inspect.State.Running {
		return errors.New("container not running")
	}

	if inspect.State.Dead || inspect.State.OOMKilled || inspect.State.Restarting {
		return fmt.Errorf("container has bad state: %s", inspect.State.Status)
	}

	return nil
}

// Stop stops the container node and removes it
func (i *Instance) Stop() error {
	err := i.launcher.cli.ContainerRemove(context.Background(), i.id, types.ContainerRemoveOptions{
		Force: true,
	})

	return err
}
