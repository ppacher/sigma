package trigger

import "github.com/homebot/sigma"
import "github.com/homebot/core/urn"

// Trigger is a function trigger
type Trigger interface {
	urn.Resource

	// Next blocks until the next trigger event occures or an
	// error is encountered
	Next() (sigma.Event, error)

	// Close closes the trigger
	// Any calles blocked in Next() should return an error
	Close() error
}
