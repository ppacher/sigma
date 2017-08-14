package scheduler

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
func WithID(id string) Option {
	return func(s *scheduler) error {
		s.id = id
		return nil
	}
}
