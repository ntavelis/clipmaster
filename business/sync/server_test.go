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

func TestServerStartBindsConfiguredAddress(t *testing.T) {
	tests := []struct {
		name string
		ip   string
	}{
		{name: "all IPv4 interfaces", ip: "0.0.0.0"},
		{name: "IPv4 loopback", ip: "127.0.0.1"},
		{name: "IPv6 loopback", ip: "::1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ip == "::1" && !ipv6LoopbackAvailable() {
				t.Skip("IPv6 loopback is unavailable")
			}

			srv := newTestServer(tt.ip, 0)
			if err := srv.Start(); err != nil {
				t.Fatalf("Start: %v", err)
			}
			t.Cleanup(func() { shutdownTestServer(srv) })

			addr := srv.listener.Addr().(*net.TCPAddr)
			if tt.ip == "0.0.0.0" {
				if !addr.IP.IsUnspecified() {
					t.Fatalf("bound IP = %s, want an unspecified address", addr.IP)
				}
			} else if !addr.IP.Equal(net.ParseIP(tt.ip)) {
				t.Fatalf("bound IP = %s, want %s", addr.IP, tt.ip)
			}
			if srv.Port() == 0 {
				t.Fatal("Port returned zero after Start")
			}
		})
	}
}

func TestServerStartUsesConfiguredPort(t *testing.T) {
	port := availablePort(t)
	srv := newTestServer("127.0.0.1", port)
	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { shutdownTestServer(srv) })

	if got := srv.Port(); got != port {
		t.Fatalf("Port = %d, want %d", got, port)
	}
}

func TestServerStartRejectsUnavailablePort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	if err := newTestServer("127.0.0.1", port).Start(); err == nil {
		t.Fatalf("Start succeeded on occupied port %d", port)
	}
}

func TestServerStartRejectsInvalidPort(t *testing.T) {
	if err := newTestServer("127.0.0.1", -1).Start(); err == nil {
		t.Fatal("Start succeeded with an invalid port")
	}
}

func newTestServer(ip string, port int) *Server {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(log, tls.Certificate{}, ip, port)
}

func availablePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("release available port: %v", err)
	}
	return port
}

func ipv6LoopbackAvailable() bool {
	listener, err := net.Listen("tcp", "[::1]:0")
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

func shutdownTestServer(srv *Server) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
