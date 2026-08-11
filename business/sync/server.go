// Package sync provides the HTTPS server that exposes clipboard history
// to other Omaclip instances on the local network.
package sync

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
)

// Server is a lightweight HTTPS server bound to all IPv4 interfaces.
type Server struct {
	log      *slog.Logger
	mux      *http.ServeMux
	listener net.Listener
	server   *http.Server
	cert     tls.Certificate
	port     int
}

// New creates a Server. Register routes via Handle, then call Start.
func New(log *slog.Logger, cert tls.Certificate, port int) *Server {
	return &Server{log: log, mux: http.NewServeMux(), cert: cert, port: port}
}

// Handle registers a handler for the given pattern.
func (s *Server) Handle(pattern string, handler http.HandlerFunc) {
	s.mux.HandleFunc(pattern, handler)
}

// Start binds to all IPv4 interfaces and begins serving in a goroutine.
// A port of zero requests an OS-assigned port. Port is valid after Start succeeds.
func (s *Server) Start() error {
	ln, err := net.Listen("tcp4", net.JoinHostPort("0.0.0.0", strconv.Itoa(s.port)))
	if err != nil {
		return fmt.Errorf("sync server: listen: %w", err)
	}

	tlsListener := tls.NewListener(ln, &tls.Config{
		Certificates: []tls.Certificate{s.cert},
	})

	s.listener = tlsListener
	s.port = ln.Addr().(*net.TCPAddr).Port
	s.server = &http.Server{Handler: s.mux}

	go func() {
		if err := s.server.Serve(tlsListener); err != nil && err != http.ErrServerClosed {
			s.log.Error("sync server: serve error", "error", err)
		}
	}()

	s.log.Info("sync server started", "port", s.port)
	return nil
}

// Port returns the bound port. Valid only after Start returns nil.
func (s *Server) Port() int {
	return s.port
}

// Shutdown gracefully stops the HTTPS server.
func (s *Server) Shutdown(ctx context.Context) {
	if s.server != nil {
		s.server.Shutdown(ctx) //nolint:errcheck
	}
}
