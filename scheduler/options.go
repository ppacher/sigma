package scheduler

import (
	"github.com/homebot/core/resource"
	"github.com/homebot/insight/logger"
)

// Option is a Scheduler option
type Option func(s *scheduler) error

// WithNamespace configures the namespace for the sigma scheduler
func WithNamespace(n string) Option {
	return func(s *scheduler) error {
		s.namespace = n
		return nil
	}
}

// WithID sets the schedulers instance ID
func WithID(id resource.Name) Option {
	return func(s *scheduler) error {
		s.id = id
		return nil
	}
}

func WithLogger(l logger.Logger) Option {
	return func(s *scheduler) error {
		s.log = l
		return nil
	}
}
