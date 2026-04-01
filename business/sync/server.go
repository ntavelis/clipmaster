// Package sync provides the HTTP server that exposes clipboard history
// to other Clipmaster instances on the local network.
package sync

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
)

// Server is a lightweight HTTP server bound to a random OS-assigned port.
type Server struct {
	log      *slog.Logger
	mux      *http.ServeMux
	listener net.Listener
	server   *http.Server
	port     int
}

// New creates a Server. Register routes via Handle, then call Start.
func New(log *slog.Logger) *Server {
	return &Server{log: log, mux: http.NewServeMux()}
}

// Handle registers a handler for the given pattern.
func (s *Server) Handle(pattern string, handler http.HandlerFunc) {
	s.mux.HandleFunc(pattern, handler)
}

// Start binds to an OS-assigned port and begins serving in a goroutine.
// Port() is valid after Start returns without error.
func (s *Server) Start(_ context.Context) error {
	ln, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		return fmt.Errorf("sync server: listen: %w", err)
	}

	s.listener = ln
	s.port = ln.Addr().(*net.TCPAddr).Port
	s.server = &http.Server{Handler: s.mux}

	go func() {
		if err := s.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			s.log.Error("sync server: serve error", "error", err)
		}
	}()

	s.log.Info("sync server started", "port", s.port)
	return nil
}

// Port returns the OS-assigned port. Valid only after Start returns nil.
func (s *Server) Port() int {
	return s.port
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) {
	if s.server != nil {
		s.server.Shutdown(ctx) //nolint:errcheck
	}
}
