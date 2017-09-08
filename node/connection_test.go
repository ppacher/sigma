package node

import (
	"runtime"
	"testing"

	"golang.org/x/net/context"

	"github.com/homebot/core/urn"
	sigma_api "github.com/homebot/protobuf/pkg/api/sigma"
	"github.com/homebot/sigma"
	"github.com/stretchr/testify/assert"
)

func TestConnection_getChannels(t *testing.T) {
	assert := assert.New(t)

	node := newNodeConn(urn.URN("urn"), "secret", sigma.FunctionSpec{})
	if !assert.NotNil(node) {
		return
	}

	assert.False(node.Registered())
	assert.False(node.Connected())

	_, _, err := node.getChannels()
	assert.EqualError(err, "not yet registered")

	node.registered = true
	_, _, err = node.getChannels()
	assert.EqualError(err, "not connected")
	assert.True(node.Registered())
	assert.False(node.Connected())

	channel := &nodeChannel{
		request:  make(chan *sigma_api.DispatchEvent),
		response: make(chan *sigma_api.ExecutionResult),
	}

	node.channel = channel
	assert.True(node.Connected())

	req, res, err := node.getChannels()
	assert.NoError(err)
	assert.Equal(channel.request, req)
	assert.Equal(channel.response, res)
}

func TestConnection_Registered(t *testing.T) {
	assert := assert.New(t)
	conn := newNodeConn(urn.URN("urn"), "secret", sigma.FunctionSpec{})

	assert.False(conn.Registered())
	conn.setRegistered(true)
	assert.True(conn.Registered())
	conn.setRegistered(false)
	assert.False(conn.Registered())
}

func TestConnection_Send(t *testing.T) {
	assert := assert.New(t)
	conn := newNodeConn(urn.URN("urn"), "secret", sigma.FunctionSpec{})

	assert.False(conn.Connected())

	channel := &nodeChannel{
		request:  make(chan *sigma_api.DispatchEvent, 10),
		response: make(chan *sigma_api.ExecutionResult, 10),
	}
	conn.setConnected(channel)
	conn.setRegistered(true)
	assert.True(conn.Connected())

	req := &sigma_api.DispatchEvent{
		Id: "foobar",
	}

	err := conn.Send(req)
	if !assert.NoError(err) {
		return
	}

	r := <-channel.request
	assert.Equal("foobar", r.Id)
}

func TestConnection_Send_NotConnected(t *testing.T) {
	assert := assert.New(t)
	conn := newNodeConn(urn.URN("urn"), "secret", sigma.FunctionSpec{})

	assert.False(conn.Connected())
	conn.setRegistered(true)

	req := &sigma_api.DispatchEvent{
		Id: "foobar",
	}

	err := conn.Send(req)
	assert.Error(err)
}

func TestConnection_CloseDuringSend(t *testing.T) {
	assert := assert.New(t)
	conn := newNodeConn(urn.URN("urn"), "secret", sigma.FunctionSpec{})

	assert.False(conn.Connected())

	channel := &nodeChannel{
		request:  make(chan *sigma_api.DispatchEvent),
		response: make(chan *sigma_api.ExecutionResult),
	}
	conn.setConnected(channel)
	conn.setRegistered(true)
	assert.True(conn.Connected())

	req := &sigma_api.DispatchEvent{
		Id: "foobar",
	}

	ch := make(chan struct{})
	go func() {
		err := conn.Send(req)
		assert.Error(err)
		close(ch)
	}()

	runtime.Gosched()
	assert.False(conn.isClosed())
	assert.NoError(conn.Close())
	assert.True(conn.isClosed())

	// Re-closing the connection is a NOP
	assert.NoError(conn.Close())

	<-ch
}

func TestConnection_Receive(t *testing.T) {
	assert := assert.New(t)
	conn := newNodeConn(urn.URN("urn"), "secret", sigma.FunctionSpec{})

	channel := &nodeChannel{
		request:  make(chan *sigma_api.DispatchEvent),
		response: make(chan *sigma_api.ExecutionResult, 1),
	}

	conn.setConnected(channel)
	conn.setRegistered(true)

	msg := &sigma_api.ExecutionResult{
		Id: "foobar",
	}

	channel.response <- msg

	res, err := conn.Receive(context.Background())
	assert.NoError(err)
	if !assert.NotNil(res) {
		return
	}

	assert.Equal("foobar", res.Id)
}

func TestConnection_Receive_Closed(t *testing.T) {
	assert := assert.New(t)
	conn := newNodeConn(urn.URN("urn"), "secret", sigma.FunctionSpec{})

	channel := &nodeChannel{
		request:  make(chan *sigma_api.DispatchEvent),
		response: make(chan *sigma_api.ExecutionResult),
	}

	conn.setConnected(channel)
	conn.setRegistered(true)

	ch := make(chan struct{})

	go func() {
		res, err := conn.Receive(context.Background())
		assert.Error(err)
		assert.Nil(res)

		close(ch)
	}()

	runtime.Gosched()

	assert.False(conn.isClosed())
	assert.NoError(conn.Close())
	assert.True(conn.isClosed())

	// Reclosing the connection is a NOP
	assert.NoError(conn.Close())
	<-ch
}

func TestConnection_Receive_ContextCanceled(t *testing.T) {
	assert := assert.New(t)
	conn := newNodeConn(urn.URN("urn"), "secret", sigma.FunctionSpec{})

	channel := &nodeChannel{
		request:  make(chan *sigma_api.DispatchEvent),
		response: make(chan *sigma_api.ExecutionResult),
	}

	conn.setConnected(channel)
	conn.setRegistered(true)

	ch := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		res, err := conn.Receive(ctx)
		assert.Error(err)
		assert.Nil(res)

		close(ch)
	}()

	runtime.Gosched()

	cancel()

	<-ch
}
