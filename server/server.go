package server

import (
	"errors"

	"github.com/homebot/core/urn"
	"github.com/homebot/idam"
	uuid "github.com/satori/go.uuid"

	"golang.org/x/net/context"

	"github.com/homebot/idam/token"
	"github.com/homebot/protobuf/pkg/api"
	sigma_api "github.com/homebot/protobuf/pkg/api/sigma"
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
func (s *Server) Create(ctx context.Context, in *sigma_api.CreateFunctionRequest) (*sigma_api.CreateFunctionResponse, error) {
	target := urn.SigmaFunctionResource.BuildURN(s.scheduler.URN().Namespace(), s.scheduler.URN().AccountID(), "")

	auth, err := s.isPermitted(ctx, "", target)
	if err != nil {
		return nil, err
	}

	if in == nil {
		return nil, errors.New("invalid request")
	}

	if auth != nil {
		ctx = context.WithValue(ctx, "accountId", auth.URN.AccountID())
	}

	spec := sigma.SpecFromProto(in.GetSpec())
	if spec.ID == "" || spec.Type == "" {
		return nil, errors.New("invalid function spec")
	}

	u, err := s.scheduler.Create(ctx, spec)
	if err != nil {
		return nil, err
	}

	return &sigma_api.CreateFunctionResponse{
		Urn: urn.ToProtobuf(u),
	}, nil
}

// Destroy destroys the function and all associated resources identified by URN
func (s *Server) Destroy(ctx context.Context, in *api.URN) (*api.Empty, error) {
	if in == nil {
		return nil, errors.New("invalid request")
	}

	u := urn.FromProtobuf(in)
	if !u.Valid() {
		return nil, errors.New("invalid URN")
	}

	if _, err := s.isPermitted(ctx, "destroy", u); err != nil {
		return nil, err
	}

	if err := s.scheduler.Destroy(ctx, u); err != nil {
		return nil, err
	}

	return &api.Empty{}, nil
}

// Dispatch dispatches an event to the given function and returns the result
func (s *Server) Dispatch(ctx context.Context, in *sigma_api.DispatchRequest) (*sigma_api.DispatchResult, error) {
	if in == nil || in.Event == nil {
		return nil, errors.New("invalid request")
	}

	// a unique ID for the execution
	in.Event.Id = uuid.NewV4().String()

	u := urn.FromProtobuf(in.GetTarget())
	if !u.Valid() {
		return nil, errors.New("invalid URN")
	}

	if _, err := s.isPermitted(ctx, "dispatch", u); err != nil {
		return nil, err
	}

	if in.GetEvent() == nil || in.GetEvent().GetId() == "" {
		return nil, errors.New("invalid request: event data invalid")
	}

	e := sigma.NewSimpleEvent(in.GetEvent().GetId(), in.GetEvent().GetPayload())

	node, res, err := s.scheduler.Dispatch(ctx, u, e)
	if err != nil {
		return nil, err
	}

	return &sigma_api.DispatchResult{
		Target: urn.ToProtobuf(u),
		Node:   urn.ToProtobuf(node),
		Result: &sigma_api.DispatchResult_Data{
			Data: res,
		},
	}, nil
}

// Inspect inspects a function and returns details and statistics for the function
func (s *Server) Inspect(ctx context.Context, in *api.URN) (*sigma_api.Function, error) {
	u := urn.FromProtobuf(in)
	if !u.Valid() {
		return nil, errors.New("invalid URN")
	}

	if _, err := s.isPermitted(ctx, "inspect", u); err != nil {
		return nil, err
	}

	f, err := s.scheduler.Inspect(ctx, u)
	if err != nil {
		return nil, err
	}

	var nodes []*sigma_api.Node

	for _, n := range f.Nodes {
		nodes = append(nodes, &sigma_api.Node{
			Urn:        urn.ToProtobuf(n.URN),
			State:      n.State.ToProtobuf(),
			Statistics: n.Stats.ToProtobuf(),
		})
	}

	return &sigma_api.Function{
		Spec:  f.Spec.ToProtobuf(),
		Urn:   urn.ToProtobuf(f.URN),
		Nodes: nodes,
	}, nil
}

// List returns a list of functions managed by the scheduler
func (s *Server) List(ctx context.Context, _ *api.Empty) (*sigma_api.ListResult, error) {
	auth, err := s.isPermitted(ctx, "list", "")
	if err != nil {
		return nil, err
	}

	functions, err := s.scheduler.Functions(ctx)

	if err != nil {
		return nil, err
	}

	var result []*sigma_api.Function

	for _, f := range functions {
		if auth != nil && f.URN.AccountID() != auth.URN.AccountID() {
			continue
		}

		var nodes []*sigma_api.Node

		for _, n := range f.Nodes {
			nodes = append(nodes, &sigma_api.Node{
				Urn:        urn.ToProtobuf(n.URN),
				State:      n.State.ToProtobuf(),
				Statistics: n.Stats.ToProtobuf(),
			})
		}

		result = append(result, &sigma_api.Function{
			Urn:   urn.ToProtobuf(f.URN),
			Spec:  f.Spec.ToProtobuf(),
			Nodes: nodes,
		})
	}

	return &sigma_api.ListResult{
		Functions: result,
	}, nil
}

func (s *Server) isPermitted(ctx context.Context, action string, target urn.URN) (*token.Token, error) {
	if s.keyFn != nil {
		t, err := token.FromMetadata(ctx, s.keyFn)
		if err != nil {
			return t, err
		}

		if t.HasGroup(urn.SigmaAdminGroup) {
			return t, nil
		}

		if target != "" {
			if target.AccountID() == t.URN.AccountID() {
				return t, nil
			}
		}

		return t, idam.ErrNotAuthenticated
	}

	return nil, nil
}

// compile time check
var _ sigma_api.SigmaServer = &Server{}
