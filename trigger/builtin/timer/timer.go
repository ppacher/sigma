package timer

import (
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/homebot/sigma"
	"github.com/homebot/sigma/trigger"
)

var (
	// ErrMissingInterval is returned when the `interval` configuration key
	// is missing during Build()
	ErrMissingInterval = errors.New("missing `interval` configuration key")
)

// Timer is a trigger.Trigger that fires after a given interval
type Timer struct {
	timer  *time.Ticker
	i      int64
	closed chan struct{}
}

// URN returns the URN for the timer
func (t *Timer) URN() string { return "timer" }

// Close closes the timer
func (t *Timer) Close() error {
	select {
	case <-t.closed:
		return errors.New("already closed")
	default:
		close(t.closed)
	}
	t.timer.Stop()

	return nil
}

// Next waits until the timer expires and returns the current
// time as an event
func (t *Timer) Next() (sigma.Event, error) {
	select {
	case cur := <-t.timer.C:
		t.i++
		blob, _ := json.Marshal(map[string]interface{}{
			"time":      cur.Format(time.RFC3339),
			"timestamp": cur.Unix(),
			"tick":      t.i,
		})
		return sigma.NewSimpleEvent("timer", blob), nil
	case <-t.closed:
		return nil, io.EOF
	}
}

// Factory is trigger.Factory for timers
type Factory struct{}

// Build builds a new timer trigger and implements trigger.Factory
func (f Factory) Build(opts map[string]string) (trigger.Trigger, error) {
	is, ok := opts["interval"]
	if !ok {
		return nil, ErrMissingInterval
	}

	duration, err := time.ParseDuration(is)
	if err != nil {
		return nil, err
	}

	return &Timer{
		timer: time.NewTicker(duration),
	}, nil
}

func init() {
	trigger.Register("timer", &Factory{})
}
