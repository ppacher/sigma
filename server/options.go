package server

import "github.com/homebot/idam/token"

// Option is a server option
type Option func(s *Server) error

// WithIdamKeyProvider sets the JWT key provider function to use
// this enables the IDAM authentication method
func WithIdamKeyProvider(f token.KeyProviderFunc) Option {
	return func(s *Server) error {
		s.keyFn = f
		return nil
	}
}
