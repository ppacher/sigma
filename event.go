package sigma

// Event is an event that triggers the execution of one or more functions
type Event interface {
	// Type returns the type of the event
	Type() string

	// Payload returns the payload of the event
	Payload() []byte
}

// SimpleEvent is a simple sigma event to be dispatched to
// functions
type SimpleEvent struct {
	typ     string
	payload []byte
}

// Type returns the type of the event and implements sigma.Event
func (s *SimpleEvent) Type() string {
	return s.typ
}

// Payload returns the payload of the event and implements sigma.Event
func (s *SimpleEvent) Payload() []byte {
	return s.payload
}

// NewSimpleEvent returns a new sigma.Event from the given type and
// payload
func NewSimpleEvent(typ string, payload []byte) Event {
	return &SimpleEvent{
		typ:     typ,
		payload: payload,
	}
}
