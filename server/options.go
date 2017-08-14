package server

import "github.com/homebot/idam"

// Option is a server option
type Option func(s *Server) error

// WithAuthenticator sets the authenticator to use
func WithAuthenticator(a idam.Authenticator) Option {
	return func(s *Server) error {
		s.authenticator = a
		return nil
	}
}

// WithAuthorizer sets the authorizer to use
func WithAuthorizer(a idam.Authorizer) Option {
	return func(s *Server) error {
		s.authorizer = a
		return nil
	}
}
