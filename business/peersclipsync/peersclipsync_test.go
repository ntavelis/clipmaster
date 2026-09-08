package peersclipsync

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rhemvi/omaclip/business/clipboard"
	"github.com/rhemvi/omaclip/business/passphrase"
	fmdns "github.com/rhemvi/omaclip/foundation/mdns"
)

type mockDiscoverer struct {
	peers []fmdns.Peer
}

func (m *mockDiscoverer) Peers() []fmdns.Peer { return m.peers }

func newTestFetcher(discoverer peersProvider, srvClient *http.Client) *Fetcher {
	ps := &passphrase.Store{}
	ps.Set("testpassword")
	return &Fetcher{
		log:             slog.New(slog.NewTextHandler(io.Discard, nil)),
		discoverer:      discoverer,
		passphraseStore: ps,
		client:          srvClient,
		cache:           make(map[string]PeerClipboard),
		states:          make(map[string]*peerState),
	}
}

func peerFromServer(srv *httptest.Server) fmdns.Peer {
	_, portStr, _ := net.SplitHostPort(srv.Listener.Addr().String())
	port, _ := strconv.Atoi(portStr)
	return fmdns.Peer{Name: "testpeer.local.", Addr: "127.0.0.1", Port: port}
}

type protocolServer struct {
	mu       sync.Mutex
	manifest clipboard.Manifest
	poll     string
	bodies   map[string][]byte
	status   map[string]int
	requests map[string]int
}

func newProtocolServer(t *testing.T) (*protocolServer, *Fetcher, *mockDiscoverer) {
	t.Helper()
	remote := &protocolServer{
		bodies:   make(map[string][]byte),
		status:   make(map[string]int),
		requests: make(map[string]int),
	}
	remote.setManifest()
	srv := httptest.NewTLSServer(http.HandlerFunc(remote.serveHTTP))
	t.Cleanup(srv.Close)
	disc := &mockDiscoverer{peers: []fmdns.Peer{peerFromServer(srv)}}
	return remote, newTestFetcher(disc, srv.Client()), disc
}

func (s *protocolServer) serveHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests[r.URL.Path]++
	ps := &passphrase.Store{}
	ps.Set("testpassword")
	if r.Header.Get("X-Omaclip-Pass") != ps.Hash() {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if status := s.status[r.URL.Path]; status != 0 {
		w.WriteHeader(status)
	}
	switch r.URL.Path {
	case "/api/clipboard/checksum":
		json.NewEncoder(w).Encode(map[string]string{"checksum": s.poll})
	case "/api/clipboard":
		json.NewEncoder(w).Encode(s.manifest)
	default:
		if !strings.HasSuffix(r.URL.Path, "/content") {
			http.NotFound(w, r)
			return
		}
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/clipboard/"), "/content")
		body, ok := s.bodies[id]
		if !ok {
			http.NotFound(w, r)
			return
		}
		// Receiver must take image MIME metadata from the manifest, not this header.
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(body) //nolint:errcheck
	}
}

func (s *protocolServer) setManifest(entries ...clipboard.ManifestEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if entries == nil {
		entries = []clipboard.ManifestEntry{}
	}
	s.manifest = clipboard.Manifest{Checksum: clipboard.ManifestChecksum(entries), Entries: entries}
	s.poll = s.manifest.Checksum
}

func (s *protocolServer) setContent(id string, body []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bodies[id] = body
}

func (s *protocolServer) setStatus(path string, status int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status[path] = status
}

func (s *protocolServer) count(path string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.requests[path]
}

func manifestEntry(id, contentType string, body []byte) clipboard.ManifestEntry {
	sum := sha256.Sum256(body)
	entry := clipboard.ManifestEntry{
		ID: id, Checksum: hex.EncodeToString(sum[:]), ContentType: contentType,
		Timestamp: time.Date(2026, time.September, 7, 12, 0, 0, 0, time.UTC),
	}
	if clipboard.ParseContentType(contentType) == clipboard.ContentTypeImage {
		entry.ImageMimeType = "image/png"
	}
	return entry
}

func TestFetchAll_NewPeerEmptyClipboard(t *testing.T) {
	_, f, _ := newProtocolServer(t)
	updates := 0
	f.OnUpdate = func() { updates++ }
	f.fetchAll()
	all := f.GetAll()
	if updates != 1 || len(all) != 1 || all[0].Entries == nil || len(all[0].Entries) != 0 {
		t.Fatalf("empty peer was not published: updates=%d clipboards=%+v", updates, all)
	}
	if all[0].PeerName != "testpeer" {
		t.Errorf("display name = %q, want testpeer", all[0].PeerName)
	}
}

func TestFetchAll_UnchangedImagesAvoidTransfer(t *testing.T) {
	remote, f, _ := newProtocolServer(t)
	text := []byte("hello\r\n世界\x00")
	image := []byte{0x89, 'P', 'N', 'G', 0, 0xff}
	textEntry := manifestEntry("text", "text", text)
	imageEntry := manifestEntry("image", "image", image)
	remote.setContent("text", text)
	remote.setContent("image", image)
	remote.setManifest(imageEntry, textEntry)
	updates := 0
	f.OnUpdate = func() { updates++ }
	f.fetchAll()
	f.fetchAll()
	want := []PeerClipboard{{PeerName: "testpeer", Entries: []clipboard.ClipboardEntry{
		{ID: "image", Checksum: imageEntry.Checksum, ContentType: "image", ImageData: base64.StdEncoding.EncodeToString(image), ImageMimeType: "image/png", Timestamp: imageEntry.Timestamp},
		{ID: "text", Checksum: textEntry.Checksum, ContentType: "text", Content: string(text), Timestamp: textEntry.Timestamp},
	}}}
	if got := f.GetAll(); !reflect.DeepEqual(got, want) {
		t.Fatalf("published content = %+v, want %+v", got, want)
	}
	if updates != 1 || remote.count("/api/clipboard/checksum") != 2 || remote.count("/api/clipboard") != 1 || remote.count("/api/clipboard/image/content") != 1 || remote.count("/api/clipboard/text/content") != 1 {
		t.Fatal("unchanged poll must fetch only checksum and must not notify")
	}

	// New IDs and MIME metadata do not require fetching already verified payloads.
	imageEntry.ID = "renamed-image"
	imageEntry.ImageMimeType = "image/webp"
	remote.setManifest(textEntry, imageEntry)
	f.fetchAll()
	got := f.GetAll()[0].Entries
	if updates != 2 || got[0].ID != "text" || got[1].ID != "renamed-image" || got[1].ImageMimeType != "image/webp" || got[1].ImageData != base64.StdEncoding.EncodeToString(image) {
		t.Fatalf("metadata-only update was not published: %+v", got)
	}
	if remote.count("/api/clipboard/renamed-image/content") != 0 || remote.count("/api/clipboard/text/content") != 1 {
		t.Fatal("metadata-only update downloaded cached content")
	}
}

func TestFetchAll_PartialFailureRetriesOnlyMissingContent(t *testing.T) {
	remote, f, _ := newProtocolServer(t)
	old := manifestEntry("old", "text", []byte("old"))
	remote.setContent("old", []byte("old"))
	remote.setManifest(old)
	f.fetchAll()
	visible := f.GetAll()
	updates := 0
	f.OnUpdate = func() { updates++ }
	image := manifestEntry("image", "image", []byte("image"))
	text := manifestEntry("new", "text", []byte("new"))
	remote.setContent("image", []byte("image"))
	remote.setContent("new", []byte("new"))
	remote.setManifest(image, text, old)
	remote.setStatus("/api/clipboard/new/content", http.StatusNotFound)
	f.fetchAll()
	if updates != 0 || !reflect.DeepEqual(f.GetAll(), visible) {
		t.Fatal("partial snapshot replaced the displayed clipboard")
	}
	remote.setStatus("/api/clipboard/new/content", 0)
	f.fetchAll()
	got := f.GetAll()[0].Entries
	if updates != 1 || len(got) != 3 || got[0].ImageData != base64.StdEncoding.EncodeToString([]byte("image")) || got[1].Content != "new" || got[2].Content != "old" {
		t.Fatalf("completed snapshot not published atomically: %+v", got)
	}
	if remote.count("/api/clipboard") != 2 || remote.count("/api/clipboard/image/content") != 1 || remote.count("/api/clipboard/new/content") != 2 || remote.count("/api/clipboard/old/content") != 1 {
		t.Fatal("retry did not reuse the pending manifest and successful payloads")
	}
}

func TestFetchAll_SupersededPendingManifest(t *testing.T) {
	remote, f, _ := newProtocolServer(t)
	f.fetchAll()
	shared := manifestEntry("shared", "image", []byte("shared"))
	obsolete := manifestEntry("obsolete", "text", []byte("obsolete"))
	missing := manifestEntry("missing", "text", []byte("missing"))
	remote.setContent("shared", []byte("shared"))
	remote.setContent("obsolete", []byte("obsolete"))
	remote.setManifest(shared, obsolete, missing)
	f.fetchAll()

	shared.ID = "shared-new-id"
	latest := manifestEntry("latest", "text", []byte("latest"))
	remote.setContent("latest", []byte("latest"))
	remote.setManifest(latest, shared)
	f.fetchAll()
	got := f.GetAll()[0].Entries
	if len(got) != 2 || got[0].Content != "latest" || got[1].ID != "shared-new-id" || got[1].ImageData != base64.StdEncoding.EncodeToString([]byte("shared")) {
		t.Fatalf("superseded snapshot published: %+v", got)
	}
	if remote.count("/api/clipboard/missing/content") != 1 || remote.count("/api/clipboard/shared-new-id/content") != 0 {
		t.Fatal("superseded pending work was retried or reusable payload lost")
	}
	// Obsolete pending payloads must not remain cached indefinitely.
	remote.setManifest(obsolete)
	f.fetchAll()
	if remote.count("/api/clipboard/obsolete/content") != 2 {
		t.Fatal("obsolete pending payload was not pruned")
	}
}

func TestFetchAll_ReturnToCommittedClearsPending(t *testing.T) {
	remote, f, _ := newProtocolServer(t)
	f.fetchAll()
	ready := manifestEntry("ready", "text", []byte("ready"))
	missing := manifestEntry("missing", "image", []byte("missing"))
	remote.setContent("ready", []byte("ready"))
	remote.setManifest(ready, missing)
	f.fetchAll()
	updates := 0
	f.OnUpdate = func() { updates++ }
	remote.setManifest()
	f.fetchAll()
	if updates != 0 || remote.count("/api/clipboard") != 2 {
		t.Fatal("return to committed snapshot fetched manifest or notified")
	}
	remote.setManifest(ready, missing)
	remote.setContent("missing", []byte("missing"))
	f.fetchAll()
	if remote.count("/api/clipboard") != 3 || remote.count("/api/clipboard/ready/content") != 2 {
		t.Fatal("stale pending manifest or payload survived return to committed version")
	}
}

func TestFetchAll_PeerDisappearsDropsAllState(t *testing.T) {
	remote, f, disc := newProtocolServer(t)
	old := manifestEntry("old", "text", []byte("old"))
	ready := manifestEntry("ready", "image", []byte("ready"))
	missing := manifestEntry("missing", "text", []byte("missing"))
	remote.setContent("old", []byte("old"))
	remote.setManifest(old)
	f.fetchAll()
	remote.setContent("ready", []byte("ready"))
	remote.setManifest(ready, missing, old)
	f.fetchAll()
	peers := disc.peers
	disc.peers = nil
	updates := 0
	f.OnUpdate = func() { updates++ }
	f.fetchAll()
	if updates != 1 || len(f.GetAll()) != 0 {
		t.Fatal("disappeared peer remains visible or did not notify")
	}
	remote.setContent("missing", []byte("missing"))
	disc.peers = peers
	f.fetchAll()
	if updates != 2 || remote.count("/api/clipboard") != 3 || remote.count("/api/clipboard/old/content") != 2 || remote.count("/api/clipboard/ready/content") != 2 {
		t.Fatal("reappearing peer reused discarded committed or pending state")
	}
}

func TestFetchAll_ManifestVersionWinsChecksumRace(t *testing.T) {
	remote, f, _ := newProtocolServer(t)
	entry := manifestEntry("new", "text", []byte("new"))
	remote.setContent("new", []byte("new"))
	remote.setManifest(entry)
	remote.mu.Lock()
	remote.poll = clipboard.ManifestChecksum([]clipboard.ManifestEntry{})
	remote.mu.Unlock()
	f.fetchAll()
	remote.setManifest(entry)
	f.fetchAll()
	all := f.GetAll()
	if len(all) != 1 || len(all[0].Entries) != 1 || all[0].Entries[0].Content != "new" || remote.count("/api/clipboard") != 1 {
		t.Fatalf("manifest version was not committed: %+v", all)
	}
}

func TestFetchAll_RejectsCorruptContentAndManifest(t *testing.T) {
	remote, f, _ := newProtocolServer(t)
	f.fetchAll()
	visible := f.GetAll()
	entry := manifestEntry("new", "image", []byte("correct"))
	remote.setContent("new", []byte("corrupt"))
	remote.setManifest(entry)
	remote.mu.Lock()
	remote.manifest.Entries[0].ID = "tampered"
	remote.mu.Unlock()
	f.fetchAll()
	if !reflect.DeepEqual(f.GetAll(), visible) || remote.count("/api/clipboard/tampered/content") != 0 {
		t.Fatal("invalid manifest was used")
	}
	remote.setManifest(entry)
	f.fetchAll()
	if !reflect.DeepEqual(f.GetAll(), visible) {
		t.Fatal("corrupt payload was published")
	}
	remote.setContent("new", []byte("correct"))
	f.fetchAll()
	got := f.GetAll()[0].Entries
	if len(got) != 1 || got[0].ImageData != base64.StdEncoding.EncodeToString([]byte("correct")) || remote.count("/api/clipboard/new/content") != 2 {
		t.Fatal("corrupt payload was cached or valid retry failed")
	}
}

func TestFetchAll_HTTPFailureDoesNotAdvance(t *testing.T) {
	for _, endpoint := range []string{"/api/clipboard/checksum", "/api/clipboard", "/api/clipboard/new/content"} {
		t.Run(endpoint, func(t *testing.T) {
			remote, f, _ := newProtocolServer(t)
			entry := manifestEntry("new", "text", []byte("new"))
			remote.setContent("new", []byte("new"))
			remote.setManifest(entry)
			remote.setStatus(endpoint, http.StatusInternalServerError)
			updates := 0
			f.OnUpdate = func() { updates++ }
			f.fetchAll()
			if updates != 0 || len(f.GetAll()) != 0 {
				t.Fatal("failed fetch published a peer")
			}
			remote.setStatus(endpoint, 0)
			f.fetchAll()
			all := f.GetAll()
			if updates != 1 || len(all) != 1 || len(all[0].Entries) != 1 || all[0].Entries[0].Content != "new" {
				t.Fatal("HTTP failure advanced the version and prevented retry")
			}
		})
	}
}

func TestFetchAll_ContentCacheIncludesType(t *testing.T) {
	remote, f, _ := newProtocolServer(t)
	body := []byte("same bytes")
	text := manifestEntry("text", "text", body)
	image := manifestEntry("image", "image", body)
	remote.setContent("text", body)
	remote.setContent("image", body)
	remote.setManifest(image, text)
	f.fetchAll()
	got := f.GetAll()[0].Entries
	if len(got) != 2 || got[0].ImageData != base64.StdEncoding.EncodeToString(body) || got[1].Content != string(body) || remote.count("/api/clipboard/text/content") != 1 || remote.count("/api/clipboard/image/content") != 1 {
		t.Fatal("content cache conflated image and text with identical checksums")
	}
}

func TestFetchAll_ContentCacheIsPerPeer(t *testing.T) {
	remote, f, disc := newProtocolServer(t)
	body := []byte("shared image")
	remote.setContent("image", body)
	remote.setManifest(manifestEntry("image", "image", body))
	second := disc.peers[0]
	second.Name = "2001:db8::1"
	disc.peers = append(disc.peers, second)
	f.fetchAll()
	got := f.GetAll()
	if len(got) != 2 || got[0].PeerName != second.Name || got[1].PeerName != "testpeer" {
		t.Fatalf("peer display names were not preserved: %+v", got)
	}
	for _, peer := range got {
		if len(peer.Entries) != 1 || peer.Entries[0].ImageData != base64.StdEncoding.EncodeToString(body) {
			t.Fatalf("peer payload missing: %+v", peer)
		}
	}
	if remote.count("/api/clipboard/image/content") != 2 {
		t.Fatal("payload cache was shared across peers")
	}
}
