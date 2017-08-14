package process

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/homebot/core/log"
	"github.com/homebot/sigma/launcher"
)

// TypeConfig holds type configuration values for a process launcher
type TypeConfig struct {
	// Command holds the command to execute for the exec type
	Command []string `json:"command" yaml:"command"`
}

// Instance is a launcher.Instance node backed by a running process
type Instance struct {
	closed  chan struct{}
	cmd     *exec.Cmd
	exitErr error

	l log.Logger
}

func (i *Instance) watch() {
	err := i.cmd.Wait()
	i.exitErr = err

	i.l.Errorf("process exited")

	close(i.closed)
}

// Healthy returns nil as long as the process instance is healthy
func (i *Instance) Healthy() error {
	select {
	case <-i.closed:
		if i.exitErr != nil {
			return i.exitErr
		}
		return errors.New("exited")
	default:
		return nil
	}
}

// Stop stops the instance and terminates the process
func (i *Instance) Stop() error {
	return i.cmd.Process.Kill()
}

// NewLauncher creates a new process launcher supporting the
// provided types
func NewLauncher(types map[string]TypeConfig) *Launcher {
	return &Launcher{
		nodeTypes: types,
	}
}

// Launcher is a process launcher and implements launcher.Launcher
type Launcher struct {
	nodeTypes map[string]TypeConfig
}

// Create creates a new instance
func (l *Launcher) Create(ctx context.Context, typ string, c launcher.Config) (launcher.Instance, error) {
	log := log.WithURN(c.URN)

	typCfg, ok := l.nodeTypes[typ]
	if !ok {
		return nil, errors.New("unsupported node instance type")
	}

	if len(typCfg.Command) == 0 {
		return nil, errors.New("no command configured for type")
	}

	cmd := exec.Command(typCfg.Command[0], typCfg.Command[1:]...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)

	for key, value := range c.EnvVars() {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	instance := &Instance{
		cmd:    cmd,
		closed: make(chan struct{}),
		l:      log,
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	go instance.watch()

	return instance, nil
}
