package node

import (
	"errors"
	"testing"
	"time"

	"golang.org/x/net/context"

	sigma "github.com/homebot/protobuf/pkg/api/sigma"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type nodeConnMock struct {
	mock.Mock
	send chan struct{}
}

func (n *nodeConnMock) Send(in *sigma.DispatchEvent) error {
	return n.Called(in).Error(0)
}

func (n *nodeConnMock) Receive(ctx context.Context) (*sigma.ExecutionResult, error) {
	select {
	case <-n.send:
	case <-ctx.Done():
	}
	args := n.Called()
	return args.Get(0).(*sigma.ExecutionResult), args.Error(1)
}

func (n *nodeConnMock) Connected() bool {
	return n.Called().Bool(0)
}

func (n *nodeConnMock) Registered() bool {
	return n.Called().Bool(0)
}

func (n *nodeConnMock) Close() error {
	return n.Called().Error(0)
}

func TestRouter_Routing(t *testing.T) {
	assert := assert.New(t)

	conn := new(nodeConnMock)
	conn.send = make(chan struct{})

	conn.On("Receive").Return(&sigma.ExecutionResult{}, errors.New("dummy")).Once()
	conn.On("Receive").Return(&sigma.ExecutionResult{
		Id: "foobar",
	}, nil).Once()
	conn.On("Receive").Return(&sigma.ExecutionResult{}, errors.New("closed"))
	conn.On("Close").Return(nil).Once()
	conn.On("Close").Return(errors.New("dummy error"))

	router := NewRouter(conn).(*router)
	if !assert.NotNil(router) {
		return
	}

	res := make(chan *sigma.ExecutionResult, 1)
	router.addRoute("foobar", res)

	conn.send <- struct{}{}
	conn.send <- struct{}{}
	r := <-res
	assert.Equal("foobar", r.Id)

	assert.NoError(router.Close())
	assert.Error(router.Close())

	conn.AssertExpectations(t)
}

func TestRouter_Dispatch(t *testing.T) {
	assert := assert.New(t)
	conn := new(nodeConnMock)
	conn.send = make(chan struct{})

	in := &sigma.DispatchEvent{
		Id: "foobar",
	}
	res := &sigma.ExecutionResult{
		Id: "foobar",
		ExecutionResult: &sigma.ExecutionResult_Result{
			Result: []byte("foobar"),
		},
	}

	conn.On("Receive").Return(res, nil).Once()
	conn.On("Receive").Return(res, errors.New("closed"))
	conn.On("Send", in).Return(nil)
	conn.On("Close").Return(nil)

	router := NewRouter(conn).(*router)
	if !assert.NotNil(router) {
		return
	}

	ch := make(chan struct{})

	go func() {
		res, err := router.Dispatch(context.Background(), in)
		assert.NoError(err)
		assert.NotNil(res)

		close(ch)
	}()

	<-time.After(time.Millisecond)

	if !assert.NotEmpty(router.routes) {
		return
	}

	for key := range router.routes {
		res.Id = key
	}

	conn.send <- struct{}{}

	<-ch
	router.Close()

	conn.AssertExpectations(t)
}

func TestRouter_DispatchCanceled(t *testing.T) {
	assert := assert.New(t)
	conn := new(nodeConnMock)
	conn.send = make(chan struct{})

	in := &sigma.DispatchEvent{
		Id: "foobar",
	}

	ctx, cancel := context.WithCancel(context.Background())
	conn.On("Receive").Return(&sigma.ExecutionResult{}, nil)
	conn.On("Send", in).Return(nil)
	conn.On("Close").Return(nil)

	router := NewRouter(conn)

	ch := make(chan struct{})
	go func() {
		defer close(ch)

		res, err := router.Dispatch(ctx, in)
		assert.Error(err)
		assert.Nil(res)
	}()

	<-time.After(time.Millisecond)
	cancel()

	<-ch
}
