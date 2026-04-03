package peersclipsync

import (
	"crypto/tls"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"io"
	"strconv"
	"testing"
	"time"

	"clipmaster/business/clipboard"
	"clipmaster/business/passphrase"
	fmdns "clipmaster/foundation/mdns"
)

type mockDiscoverer struct {
	peers []fmdns.Peer
}

func (m *mockDiscoverer) Peers() []fmdns.Peer { return m.peers }

func newTestFetcher(discoverer peersProvider) *Fetcher {
	ps := &passphrase.Store{}
	ps.Set("testpassword")
	return &Fetcher{
		log:             slog.New(slog.NewTextHandler(io.Discard, nil)),
		discoverer:      discoverer,
		passphraseStore: ps,
		client: &http.Client{
			Timeout: 2 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
			},
		},
		cache: make(map[string]PeerClipboard),
	}
}

func peerFromServer(srv *httptest.Server) fmdns.Peer {
	_, portStr, _ := net.SplitHostPort(srv.Listener.Addr().String())
	port, _ := strconv.Atoi(portStr)
	return fmdns.Peer{Name: "testpeer.local.", Addr: "127.0.0.1", Port: port}
}

func TestFetchAll_NewPeerEmptyClipboard(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]clipboard.ClipboardEntry{}) //nolint:errcheck
	}))
	defer srv.Close()

	disc := &mockDiscoverer{peers: []fmdns.Peer{peerFromServer(srv)}}
	f := newTestFetcher(disc)

	updateCalled := false
	f.OnUpdate = func() { updateCalled = true }

	f.fetchAll()

	if !updateCalled {
		t.Error("OnUpdate should be called when a new peer is first seen")
	}
	all := f.GetAll()
	if len(all) != 1 {
		t.Fatalf("GetAll() returned %d peers, want 1", len(all))
	}
	if len(all[0].Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(all[0].Entries))
	}
}

func TestFetchAll_NewPeerWithEntries(t *testing.T) {
	entries := []clipboard.ClipboardEntry{
		{ID: "1", Content: "hello"},
		{ID: "2", Content: "world"},
	}
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(entries) //nolint:errcheck
	}))
	defer srv.Close()

	disc := &mockDiscoverer{peers: []fmdns.Peer{peerFromServer(srv)}}
	f := newTestFetcher(disc)

	updateCalled := false
	f.OnUpdate = func() { updateCalled = true }

	f.fetchAll()

	if !updateCalled {
		t.Error("OnUpdate should be called when a new peer is first seen")
	}
	all := f.GetAll()
	if len(all) != 1 {
		t.Fatalf("GetAll() returned %d peers, want 1", len(all))
	}
	if len(all[0].Entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(all[0].Entries))
	}
}

func TestFetchAll_ExistingPeerSameEntries(t *testing.T) {
	entries := []clipboard.ClipboardEntry{{ID: "1", Content: "hello"}}
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(entries) //nolint:errcheck
	}))
	defer srv.Close()

	disc := &mockDiscoverer{peers: []fmdns.Peer{peerFromServer(srv)}}
	f := newTestFetcher(disc)
	f.fetchAll()

	updateCount := 0
	f.OnUpdate = func() { updateCount++ }

	f.fetchAll()

	if updateCount != 0 {
		t.Errorf("OnUpdate should NOT be called when entries are unchanged, got %d calls", updateCount)
	}
}

func TestFetchAll_ExistingPeerUpdatedEntries(t *testing.T) {
	first := []clipboard.ClipboardEntry{{ID: "1", Content: "hello"}}
	second := []clipboard.ClipboardEntry{{ID: "1", Content: "hello"}, {ID: "2", Content: "world"}}

	callCount := 0
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			json.NewEncoder(w).Encode(first) //nolint:errcheck
		} else {
			json.NewEncoder(w).Encode(second) //nolint:errcheck
		}
	}))
	defer srv.Close()

	disc := &mockDiscoverer{peers: []fmdns.Peer{peerFromServer(srv)}}
	f := newTestFetcher(disc)
	f.fetchAll()

	updateCalled := false
	f.OnUpdate = func() { updateCalled = true }

	f.fetchAll()

	if !updateCalled {
		t.Error("OnUpdate should be called when peer entries change")
	}
	all := f.GetAll()
	if len(all[0].Entries) != 2 {
		t.Errorf("expected 2 entries after update, got %d", len(all[0].Entries))
	}
}

func TestFetchAll_PeerDisappears(t *testing.T) {
	entries := []clipboard.ClipboardEntry{{ID: "1", Content: "hello"}}
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(entries) //nolint:errcheck
	}))
	defer srv.Close()

	disc := &mockDiscoverer{peers: []fmdns.Peer{peerFromServer(srv)}}
	f := newTestFetcher(disc)
	f.fetchAll()

	disc.peers = nil
	updateCalled := false
	f.OnUpdate = func() { updateCalled = true }

	f.fetchAll()

	if !updateCalled {
		t.Error("OnUpdate should be called when a peer disappears")
	}
	if len(f.GetAll()) != 0 {
		t.Error("GetAll() should return empty after peer disappears")
	}
}

func TestFetchAll_FetchFails(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	disc := &mockDiscoverer{peers: []fmdns.Peer{peerFromServer(srv)}}
	f := newTestFetcher(disc)

	updateCalled := false
	f.OnUpdate = func() { updateCalled = true }

	f.fetchAll()

	if updateCalled {
		t.Error("OnUpdate should NOT be called when fetch fails")
	}
	if len(f.GetAll()) != 0 {
		t.Error("GetAll() should be empty when fetch fails")
	}
}
