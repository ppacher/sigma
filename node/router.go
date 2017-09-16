package node

import (
	"errors"
	"sync"

	"github.com/satori/go.uuid"

	sigmaV1 "github.com/homebot/protobuf/pkg/api/sigma/v1"
	"golang.org/x/net/context"
)

// Router wraps Conn and provides a RPC like interface for dispatching
// events and receiving the processing result
type Router interface {
	// Dispatch dispatches an event and returns the result
	Dispatch(context.Context, *sigmaV1.DispatchEvent) (*sigmaV1.ExecutionResult, error)

	// Close closes the router and the underlying NodeConn
	Close() error

	// Connected returns true if the connection is currently established
	Connected() bool

	// Registered returns true if the connection has been registered and the
	// node has been initialized
	Registered() bool
}

type router struct {
	wg     sync.WaitGroup
	mu     sync.Mutex
	routes map[string]chan *sigmaV1.ExecutionResult
	close  chan struct{}

	conn Conn
}

// NewRouter returns a new router for the node connection
func NewRouter(conn Conn) Router {
	router := &router{
		routes: make(map[string]chan *sigmaV1.ExecutionResult),
		close:  make(chan struct{}),
		conn:   conn,
	}

	router.wg.Add(1)
	go router.receive()

	return router
}

// Registered returns true if the connection has been registered and
// the node has been initialized
func (r *router) Registered() bool { return r.conn.Registered() }

// Connected returns true if the node is currently connected
func (r *router) Connected() bool { return r.conn.Connected() }

// Dispatch dispatches an event and returns the result
func (r *router) Dispatch(ctx context.Context, in *sigmaV1.DispatchEvent) (*sigmaV1.ExecutionResult, error) {
	res := make(chan *sigmaV1.ExecutionResult, 1)

	id := uuid.NewV4().String()

	in.Id = id
	r.addRoute(id, res)
	defer r.deleteRoute(id)

	if err := r.conn.Send(in); err != nil {
		return nil, err
	}

	select {
	case response := <-res:
		return response, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-r.close:
		return nil, errors.New("connection closed")
	}
}

// Close closes the router, abort pending calls and closes the connection
// to the node
func (r *router) Close() error {
	select {
	case <-r.close:
		return errors.New("already closed")
	default:
	}

	close(r.close)
	r.wg.Wait()

	return r.conn.Close()
}

func (r *router) deleteRoute(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.routes, id)
}

func (r *router) addRoute(id string, ch chan *sigmaV1.ExecutionResult) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.routes[id] = ch
}

func (r *router) getRoute(id string) (chan *sigmaV1.ExecutionResult, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ch, ok := r.routes[id]
	return ch, ok
}

func (r *router) receive() {
	defer r.wg.Done()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		select {
		case <-r.close:
			cancel()
		case <-ctx.Done():
		}
	}()

	for {
		msg, err := r.conn.Receive(ctx)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				continue
			}
		}

		route, ok := r.getRoute(msg.GetId())
		if ok {
			route <- msg
		}
	}
}
