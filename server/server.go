package server

import (
	"errors"

	"github.com/golang/protobuf/ptypes/empty"
	uuid "github.com/satori/go.uuid"

	"golang.org/x/net/context"

	"github.com/homebot/idam/policy"
	"github.com/homebot/idam/token"
	sigmaV1 "github.com/homebot/protobuf/pkg/api/sigma/v1"
	"github.com/homebot/sigma"
	"github.com/homebot/sigma/scheduler"
)

// Server is a gRPC Sigma server and implements sigma.SigmaServer
type Server struct {
	scheduler scheduler.Scheduler

	// keyFn is used to resolve the signing certifiact/key
	// for verifying JWTs
	keyFn token.KeyProviderFunc
}

// NewServer creates a new sigma server for the given scheduler
func NewServer(s scheduler.Scheduler, opts ...Option) (*Server, error) {
	srv := &Server{
		scheduler: s,
	}

	for _, fn := range opts {
		if err := fn(srv); err != nil {
			return nil, err
		}
	}

	return srv, nil
}

// Create creates a new function
func (s *Server) Create(ctx context.Context, in *sigmaV1.CreateFunctionRequest) (*sigmaV1.CreateFunctionResponse, error) {
	auth, ok := policy.TokenFromContext(ctx)
	if !ok {
		return nil, errors.New("not authenticated")
	}

	if in == nil {
		return nil, errors.New("invalid request")
	}

	if auth != nil {
		ctx = context.WithValue(ctx, "accountId", auth.Name)
	}

	spec := sigma.SpecFromProto(in.GetSpec())
	if spec.ID == "" || spec.Type == "" {
		return nil, errors.New("invalid function spec")
	}

	u, err := s.scheduler.Create(ctx, spec)
	if err != nil {
		return nil, err
	}

	return &sigmaV1.CreateFunctionResponse{
		Name: u,
	}, nil
}

// Destroy destroys the function and all associated resources identified by URN
func (s *Server) Destroy(ctx context.Context, in *sigmaV1.DestroyRequest) (*empty.Empty, error) {
	if in == nil {
		return nil, errors.New("invalid request")
	}

	u := in.GetName()

	if err := s.scheduler.Destroy(ctx, u); err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}

func (s *Server) VerificationKey(issuer string, alg string) (interface{}, error) {
	return []byte("foobar"), nil
}

func (s *Server) IsResourceOwner(resource, identity string, permissions []string) (bool, error) {
	return false, nil
}

// Dispatch dispatches an event to the given function and returns the result
func (s *Server) Dispatch(ctx context.Context, in *sigmaV1.DispatchRequest) (*sigmaV1.DispatchResult, error) {
	if in == nil || in.Event == nil {
		return nil, errors.New("invalid request")
	}

	// a unique ID for the execution
	in.Event.Id = uuid.NewV4().String()

	u := in.GetTarget()

	if in.GetEvent() == nil || in.GetEvent().GetId() == "" {
		return nil, errors.New("invalid request: event data invalid")
	}

	e := sigma.NewSimpleEvent(in.GetEvent().GetId(), in.GetEvent().GetPayload())

	node, res, err := s.scheduler.Dispatch(ctx, u, e)
	if err != nil {
		return nil, err
	}

	return &sigmaV1.DispatchResult{
		Target: u,
		Node:   node,
		Result: &sigmaV1.DispatchResult_Data{
			Data: res,
		},
	}, nil
}

// Inspect inspects a function and returns details and statistics for the function
func (s *Server) Inspect(ctx context.Context, in *sigmaV1.InspectRequest) (*sigmaV1.Function, error) {
	u := in.GetName()

	f, err := s.scheduler.Inspect(ctx, u)
	if err != nil {
		return nil, err
	}

	var nodes []*sigmaV1.Node

	for _, n := range f.Nodes {
		nodes = append(nodes, &sigmaV1.Node{
			Urn:        n.URN,
			State:      n.State.ToProtobuf(),
			Statistics: n.Stats.ToProtobuf(),
		})
	}

	return &sigmaV1.Function{
		Spec:  f.Spec.ToProtobuf(),
		Urn:   f.URN,
		Nodes: nodes,
	}, nil
}

// List returns a list of functions managed by the scheduler
func (s *Server) List(ctx context.Context, _ *empty.Empty) (*sigmaV1.ListResult, error) {
	functions, err := s.scheduler.Functions(ctx)

	if err != nil {
		return nil, err
	}

	var result []*sigmaV1.Function

	for _, f := range functions {
		var nodes []*sigmaV1.Node

		for _, n := range f.Nodes {
			nodes = append(nodes, &sigmaV1.Node{
				Urn:        n.URN,
				State:      n.State.ToProtobuf(),
				Statistics: n.Stats.ToProtobuf(),
			})
		}

		result = append(result, &sigmaV1.Function{
			Urn:   f.URN,
			Spec:  f.Spec.ToProtobuf(),
			Nodes: nodes,
		})
	}

	return &sigmaV1.ListResult{
		Functions: result,
	}, nil
}

// compile time check
var _ sigmaV1.SigmaServer = &Server{}
