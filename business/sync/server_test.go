package sync

import (
	"context"
	"crypto/tls"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"
)

func TestServerStartBindsAllIPv4Interfaces(t *testing.T) {
	srv := newTestServer(0)
	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { shutdownTestServer(srv) })

	addr := srv.listener.Addr().(*net.TCPAddr)
	if !addr.IP.Equal(net.IPv4zero) {
		t.Fatalf("bound IP = %s, want 0.0.0.0", addr.IP)
	}
	if srv.Port() == 0 {
		t.Fatal("Port returned zero after Start")
	}
}

func TestServerStartUsesConfiguredPort(t *testing.T) {
	port := availablePort(t)
	srv := newTestServer(port)
	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { shutdownTestServer(srv) })

	if got := srv.Port(); got != port {
		t.Fatalf("Port = %d, want %d", got, port)
	}
}

func TestServerStartRejectsUnavailablePort(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	if err := newTestServer(port).Start(); err == nil {
		t.Fatalf("Start succeeded on occupied port %d", port)
	}
}

func TestServerStartRejectsInvalidPort(t *testing.T) {
	if err := newTestServer(-1).Start(); err == nil {
		t.Fatal("Start succeeded with an invalid port")
	}
}

func newTestServer(port int) *Server {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(log, tls.Certificate{}, port)
}

func availablePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("release available port: %v", err)
	}
	return port
}

func shutdownTestServer(srv *Server) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
