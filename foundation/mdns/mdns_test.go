package mdns

import (
	"context"
	"io"
	"log/slog"
	"net"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/mdns"
	"github.com/rhemvi/omaclip/business/passphrase"
)

var discardLog = slog.New(slog.NewTextHandler(io.Discard, nil))

func TestFilterIPs(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "keeps LAN, drops Docker",
			in:   []string{"192.168.1.53", "172.17.0.1", "172.18.0.1", "172.19.0.1"},
			want: []string{"192.168.1.53"},
		},
		{
			name: "keeps all when no Docker IPs",
			in:   []string{"192.168.1.10", "10.0.0.5"},
			want: []string{"192.168.1.10", "10.0.0.5"},
		},
		{
			name: "drops all Docker range boundaries",
			in:   []string{"172.16.0.0", "172.31.255.255", "192.168.0.1"},
			want: []string{"192.168.0.1"},
		},
		{
			name: "keeps IPs just outside Docker range",
			in:   []string{"172.15.255.255", "172.32.0.0", "172.20.0.1"},
			want: []string{"172.15.255.255", "172.32.0.0"},
		},
		{
			name: "drops loopback IPs",
			in:   []string{"127.0.0.1", "127.255.255.255", "192.168.1.10"},
			want: []string{"192.168.1.10"},
		},
		{
			name: "drops unspecified IPs",
			in:   []string{"0.0.0.0", "10.0.0.5"},
			want: []string{"10.0.0.5"},
		},
		{
			name: "drops multicast IPs",
			in:   []string{"224.0.0.1", "239.255.255.255", "192.168.1.10"},
			want: []string{"192.168.1.10"},
		},
		{
			name: "drops link-local unicast IPs",
			in:   []string{"169.254.0.1", "169.254.255.255", "10.0.0.5"},
			want: []string{"10.0.0.5"},
		},
		{
			name: "drops non-global-unicast IPs",
			in:   []string{"255.255.255.255", "192.168.1.10"},
			want: []string{"192.168.1.10"},
		},
		{
			name: "drops IPv6 IPs",
			in:   []string{"::1", "2001:db8::1", "192.168.1.10"},
			want: []string{"192.168.1.10"},
		},
		{
			name: "empty input",
			in:   nil,
			want: nil,
		},
		{
			name: "all blocked",
			in:   []string{"172.17.0.1", "172.18.0.1"},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var input []net.IP
			for _, s := range tt.in {
				input = append(input, net.ParseIP(s))
			}

			got := filterIPs(input)

			if len(got) != len(tt.want) {
				t.Fatalf("filterIPs returned %d IPs, want %d\ngot:  %v\nwant: %v", len(got), len(tt.want), got, tt.want)
			}
			for i, ip := range got {
				if ip.String() != tt.want[i] {
					t.Errorf("filterIPs[%d] = %s, want %s", i, ip, tt.want[i])
				}
			}
		})
	}
}

func TestLanIPs_ExcludesDockerSubnet(t *testing.T) {
	host, _ := os.Hostname()
	ips := lanIPs(host)

	_, blockedNet, _ := net.ParseCIDR("172.16.0.0/12")
	for _, ip := range ips {
		if blockedNet.Contains(ip) {
			t.Errorf("lanIPs returned blocked IP %s (172.16.0.0/12)", ip)
		}
	}
}

func TestLanIPs_ExcludesLoopback(t *testing.T) {
	host, _ := os.Hostname()
	ips := lanIPs(host)

	t.Log(ips)
	for _, ip := range ips {
		if ip.IsLoopback() {
			t.Errorf("lanIPs returned loopback IP %s", ip)
		}
	}
}

func TestLanIPs_OnlyIPv4(t *testing.T) {
	host, _ := os.Hostname()
	ips := lanIPs(host)

	for _, ip := range ips {
		if ip.To4() == nil {
			t.Errorf("lanIPs returned non-IPv4 IP %s", ip)
		}
	}
}

func TestNew(t *testing.T) {
	host, _ := os.Hostname()
	d, err := New(discardLog, 5*time.Second, host, &passphrase.Store{}, "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if d.log != discardLog {
		t.Error("logger not set")
	}
	if d.browsePeriod != 5*time.Second {
		t.Errorf("browsePeriod = %v, want 5s", d.browsePeriod)
	}
	if d.peers == nil {
		t.Error("peers map not initialized")
	}
}

func TestNew_InvalidInterface(t *testing.T) {
	host, _ := os.Hostname()
	_, err := New(discardLog, 5*time.Second, host, &passphrase.Store{}, "nonexistent0")
	if err == nil {
		t.Error("expected error for invalid interface name")
	}
}

func TestPeers_EmptyByDefault(t *testing.T) {
	host, _ := os.Hostname()
	d, err := New(discardLog, 5*time.Second, host, &passphrase.Store{}, "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	peers := d.Peers()
	if len(peers) != 0 {
		t.Errorf("expected 0 peers, got %d", len(peers))
	}
}

func TestShutdown_NilServer(t *testing.T) {
	host, _ := os.Hostname()
	d, err := New(discardLog, 5*time.Second, host, &passphrase.Store{}, "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	d.Shutdown()
}

func TestGetInterfaceBindings_NoDiscoverableIPs(t *testing.T) {
	bindings, err := getInterfaceBindings(&net.Interface{Name: "no-addresses"})
	if err != ErrNoDiscoverableIPs {
		t.Errorf("expected ErrNoDiscoverableIPs, got %v", err)
	}
	if bindings != nil {
		t.Errorf("expected no bindings, got %v", bindings)
	}
}

func TestNormalizeInstanceName(t *testing.T) {
	tests := map[string]string{
		"host-19901._omaclip._tcp.local.": "host-19901",
		"host-19901.":                     "host-19901",
		"host-19901":                      "host-19901",
	}

	for input, want := range tests {
		if got := normalizeInstanceName(input); got != want {
			t.Errorf("normalizeInstanceName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestPeerFromServiceEntry(t *testing.T) {
	ps := &passphrase.Store{}
	ps.Set("testpass")
	d, err := New(discardLog, time.Second, "host", ps, "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	d.myName = "host-19901"

	valid := &mdns.ServiceEntry{
		Name:       "peer-19902._omaclip._tcp.local.",
		AddrV4:     net.ParseIP("192.168.1.20"),
		Port:       19902,
		InfoFields: []string{"version=1", "ph=" + ps.ShortHash()},
	}
	peer, ok := d.peerFromServiceEntry(valid)
	if !ok {
		t.Fatal("valid service entry was rejected")
	}
	if peer != (Peer{Name: "peer-19902", Addr: "192.168.1.20", Port: 19902}) {
		t.Fatalf("peer = %+v", peer)
	}

	tests := []struct {
		name  string
		entry *mdns.ServiceEntry
	}{
		{name: "nil entry", entry: nil},
		{name: "missing IPv4", entry: &mdns.ServiceEntry{Name: valid.Name, Port: valid.Port, InfoFields: valid.InfoFields}},
		{name: "missing port", entry: &mdns.ServiceEntry{Name: valid.Name, AddrV4: valid.AddrV4, InfoFields: valid.InfoFields}},
		{name: "self", entry: &mdns.ServiceEntry{Name: "host-19901._omaclip._tcp.local.", AddrV4: valid.AddrV4, Port: valid.Port, InfoFields: valid.InfoFields}},
		{name: "passphrase mismatch", entry: &mdns.ServiceEntry{Name: valid.Name, AddrV4: valid.AddrV4, Port: valid.Port, InfoFields: []string{"ph=wrong"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := d.peerFromServiceEntry(tt.entry); ok {
				t.Error("entry was accepted")
			}
		})
	}
}

func TestReconcilePeers_ExpiresAfterThreeMissedCycles(t *testing.T) {
	d, err := New(discardLog, time.Second, "host", &passphrase.Store{}, "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	peer := Peer{Name: "peer", Addr: "192.168.1.20", Port: 19902}
	d.reconcilePeers(map[string]Peer{peer.Name: peer})
	for cycle := 1; cycle < peerTTLCycles; cycle++ {
		d.reconcilePeers(nil)
		if len(d.Peers()) != 1 {
			t.Fatalf("peer expired after %d missed cycles", cycle)
		}
	}

	d.reconcilePeers(nil)
	if len(d.Peers()) != 0 {
		t.Fatal("peer did not expire after the TTL")
	}
}

func TestDiscovery_TwoPeers(t *testing.T) {
	host, _ := os.Hostname()
	ps := &passphrase.Store{}
	ps.Set("testpass")
	d1, err := New(discardLog, 100*time.Millisecond, host, ps, "")
	if err != nil {
		t.Fatalf("New d1: %v", err)
	}
	d2, err := New(discardLog, 100*time.Millisecond, host, ps, "")
	if err != nil {
		t.Fatalf("New d2: %v", err)
	}

	if err := d1.Register(19901); err != nil {
		t.Fatalf("d1 register: %v", err)
	}
	defer d1.Shutdown()

	if err := d2.Register(19902); err != nil {
		t.Fatalf("d2 register: %v", err)
	}
	defer d2.Shutdown()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d1.Start(ctx)
	d2.Start(ctx)

	deadline := time.After(10 * time.Second)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	var d1Found, d2Found bool
	for !d1Found || !d2Found {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for discovery (d1 found peer: %v, d2 found peer: %v)", d1Found, d2Found)
		case <-ticker.C:
			if len(d1.Peers()) > 0 {
				d1Found = true
			}
			if len(d2.Peers()) > 0 {
				d2Found = true
			}
		}
	}

	found19902 := false
	for _, p := range d1.Peers() {
		if p.Port == 19902 {
			found19902 = true
		}
	}
	if !found19902 {
		t.Errorf("d1 did not discover peer on port 19902")
	}

	found19901 := false
	for _, p := range d2.Peers() {
		if p.Port == 19901 {
			found19901 = true
		}
	}
	if !found19901 {
		t.Errorf("d2 did not discover peer on port 19901")
	}
}

func TestBrowseLoop_StopsOnCancel(t *testing.T) {
	host, _ := os.Hostname()
	d, err := New(discardLog, 50*time.Millisecond, host, &passphrase.Store{}, "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	d.Start(ctx)

	time.Sleep(150 * time.Millisecond)
	cancel()
	time.Sleep(100 * time.Millisecond)
}
