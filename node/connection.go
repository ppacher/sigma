package node

import (
	"errors"
	"io"
	"sync"

	"golang.org/x/net/context"

	"github.com/homebot/core/urn"
	sigmaV1 "github.com/homebot/protobuf/pkg/api/sigma/v1"
	"github.com/homebot/sigma"
)

// Conn is the connection to a node instance
type Conn interface {
	// Send sends a dispatch event
	Send(*sigmaV1.DispatchEvent) error

	// Receive receives an execution result
	Receive(context.Context) (*sigmaV1.ExecutionResult, error)

	// Connected returns true if the node is currently connected
	Connected() bool

	// Registered returns true if the node as registered itself and
	// has been initialized
	Registered() bool

	// Close closes the connection
	Close() error
}

type nodeChannel struct {
	request  chan *sigmaV1.DispatchEvent
	response chan *sigmaV1.ExecutionResult
}

type nodeConn struct {
	secret string
	URN    urn.URN
	spec   sigma.FunctionSpec

	closed chan struct{}

	rw         sync.Mutex
	channel    *nodeChannel
	registered bool
}

func newNodeConn(urn urn.URN, secret string, spec sigma.FunctionSpec) *nodeConn {
	return &nodeConn{
		secret: secret,
		URN:    urn,
		closed: make(chan struct{}),
		spec:   spec,
	}
}

func (n *nodeConn) Connected() bool {
	n.rw.Lock()
	defer n.rw.Unlock()

	return n.channel != nil
}

func (n *nodeConn) Close() error {
	select {
	case <-n.closed:
		// TODO(homebot) should we return an error here?
		// if the instance died the connection will already be closed
		// which causes controller.DestroyNode() to fail with the error
		// returned from here
		return nil
	default:
	}
	close(n.closed)

	return nil
}

func (n *nodeConn) Send(in *sigmaV1.DispatchEvent) error {
	req, _, err := n.getChannels()
	if err != nil {
		return err
	}

	select {
	case req <- in:
	case <-n.closed:
		return io.EOF
	}
	return nil
}

func (n *nodeConn) Receive(ctx context.Context) (*sigmaV1.ExecutionResult, error) {
	_, res, err := n.getChannels()
	if err != nil {
		return nil, err
	}

	select {
	case out := <-res:
		return out, nil
	case <-n.closed:
		return nil, io.EOF
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (n *nodeConn) Registered() bool {
	n.rw.Lock()
	defer n.rw.Unlock()

	return n.registered
}

func (n *nodeConn) setRegistered(b bool) {
	n.rw.Lock()
	defer n.rw.Unlock()

	n.registered = b
}

func (n *nodeConn) getChannels() (chan *sigmaV1.DispatchEvent, chan *sigmaV1.ExecutionResult, error) {
	n.rw.Lock()
	defer n.rw.Unlock()

	if !n.registered {
		return nil, nil, errors.New("not yet registered")
	}

	if n.channel != nil {
		return n.channel.request, n.channel.response, nil
	}

	return nil, nil, errors.New("not connected")
}

func (n *nodeConn) isClosed() bool {
	select {
	case <-n.closed:
		return true
	default:
		return false
	}
}

func (n *nodeConn) setConnected(channel *nodeChannel) {
	n.rw.Lock()
	defer n.rw.Unlock()

	n.channel = channel
}
