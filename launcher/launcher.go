package launcher

import (
	"context"
	"fmt"
	"os"
)

// Instance is an instance created and managed by a launcher
type Instance interface {
	// Healthy checks if the instance is healthy and returns
	// an error if not
	Healthy() error

	// Stop stops the instance or returns an erro
	Stop() error
}

// Config holds the launch configuration for a new instance
type Config struct {
	Address string
	Secret  string
	URN     string
}

// EnvVars returns the current configuration as a map[string]string
func (c Config) EnvVars() map[string]string {
	return map[string]string{
		"SIGMA_HANDLER_ADDRESS": c.Address,
		"SIGMA_ACCESS_SECRET":   c.Secret,
		"SIGMA_INSTANCE_URN":    c.URN,
	}
}

// Env returns a slice of strings containing environment variables
func (c Config) Env() []string {
	var env []string
	for key, value := range c.EnvVars() {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	return env
}

// ConfigFromEnv returns the configuration from environment variables
func ConfigFromEnv() Config {
	var c Config

	c.Secret = os.Getenv("SIGMA_ACCESS_SECRET")
	c.URN = os.Getenv("SIGMA_INSTANCE_URN")
	c.Address = os.Getenv("SIGMA_HANDLER_ADDRESS")

	return c
}

// Launcher creates and manages the livecycle of an instance
type Launcher interface {
	// Create creates a new instance or returns an error
	Create(context.Context, string, Config) (Instance, error)
}

// CreateFunc creates a new node instance and implements Launcher
type CreateFunc func(context.Context, string, Config) (Instance, error)

// Create calls `f` and implemements Launcher
func (f CreateFunc) Create(ctx context.Context, typ string, config Config) (Instance, error) {
	return f(ctx, typ, config)
}
