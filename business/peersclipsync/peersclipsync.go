// Package peersclipsync fetches clipboard history from discovered remote peers.
package peersclipsync

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rhemvi/omaclip/business/clipboard"
	"github.com/rhemvi/omaclip/business/passphrase"
	fmdns "github.com/rhemvi/omaclip/foundation/mdns"
)

// PeerClipboard holds a remote peer's name and its clipboard entries.
type PeerClipboard struct {
	PeerName string                     `json:"peerName"`
	Entries  []clipboard.ClipboardEntry `json:"entries"`
}

// peersProvider is satisfied by any type that can return a list of discovered peers.
type peersProvider interface {
	Peers() []fmdns.Peer
}

type contentKey struct {
	contentType string
	checksum    string
}

type peerState struct {
	committed clipboard.Manifest
	pending   *clipboard.Manifest
	payloads  map[contentKey]string
}

// prune keeps payloads needed by the visible snapshot or the latest pending one.
func (s *peerState) prune() {
	needed := make(map[contentKey]struct{}, len(s.committed.Entries))
	for _, entry := range s.committed.Entries {
		needed[contentKey{entry.ContentType, entry.Checksum}] = struct{}{}
	}
	if s.pending != nil {
		for _, entry := range s.pending.Entries {
			needed[contentKey{entry.ContentType, entry.Checksum}] = struct{}{}
		}
	}
	for key := range s.payloads {
		if _, ok := needed[key]; !ok {
			delete(s.payloads, key)
		}
	}
}

// Fetcher periodically fetches clipboard history from all discovered peers.
type Fetcher struct {
	log             *slog.Logger
	discoverer      peersProvider
	syncInterval    time.Duration
	passphraseStore *passphrase.Store
	client          *http.Client

	mu    sync.RWMutex
	cache map[string]PeerClipboard
	// states is owned by the fetch loop; only published cache snapshots need mu.
	states map[string]*peerState

	OnUpdate func()
}

// New creates a Fetcher. Call Start to begin periodic fetching.
// caPool must contain the shared CA certificate derived from the passphrase so that
// peer leaf certificates can be verified without InsecureSkipVerify.
func New(
	log *slog.Logger,
	discoverer peersProvider,
	syncInterval time.Duration,
	ps *passphrase.Store,
	caPool *x509.CertPool,
) *Fetcher {
	return &Fetcher{
		log:             log,
		discoverer:      discoverer,
		syncInterval:    syncInterval,
		passphraseStore: ps,
		client: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{RootCAs: caPool},
			},
		},
		cache:  make(map[string]PeerClipboard),
		states: make(map[string]*peerState),
	}
}

// Start begins the a fetch loop until ctx is cancelled.
func (f *Fetcher) Start(ctx context.Context) {
	go f.loop(ctx)
}

// GetAll returns a snapshot of all remote peer clipboards.
func (f *Fetcher) GetAll() []PeerClipboard {
	f.mu.RLock()
	defer f.mu.RUnlock()
	out := make([]PeerClipboard, 0, len(f.cache))
	for _, pc := range f.cache {
		out = append(out, pc)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].PeerName < out[j].PeerName
	})
	return out
}

func (f *Fetcher) loop(ctx context.Context) {
	f.fetchAll()
	ticker := time.NewTicker(f.syncInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			f.fetchAll()
		}
	}
}

func (f *Fetcher) fetchAll() {
	peers := f.discoverer.Peers()
	changed := false

	activePeers := make(map[string]struct{}, len(peers))
	for _, p := range peers {
		activePeers[p.Name] = struct{}{}
	}
	for name := range f.states {
		if _, ok := activePeers[name]; !ok {
			delete(f.states, name)
		}
	}
	f.mu.Lock()
	for name := range f.cache {
		if _, ok := activePeers[name]; !ok {
			delete(f.cache, name)
			changed = true
		}
	}
	f.mu.Unlock()

	for _, p := range peers {
		state, ok := f.states[p.Name]
		if !ok {
			state = &peerState{payloads: make(map[contentKey]string)}
			f.states[p.Name] = state
		}
		entries, updated, err := f.fetchPeer(p, state)
		if err != nil {
			f.log.Debug("failed to fetch peer clipboard", "peer", p.Name, "error", err)
			continue
		}
		if !updated {
			continue
		}
		displayName := p.Name
		if !strings.Contains(p.Name, ":") {
			displayName = strings.SplitN(p.Name, ".", 2)[0]
		}
		f.mu.Lock()
		f.cache[p.Name] = PeerClipboard{PeerName: displayName, Entries: entries}
		f.mu.Unlock()
		changed = true
	}
	if changed && f.OnUpdate != nil {
		f.OnUpdate()
	}
}

func (f *Fetcher) fetchPeer(p fmdns.Peer, state *peerState) ([]clipboard.ClipboardEntry, bool, error) {
	baseURL := fmt.Sprintf("https://%s:%d", p.Addr, p.Port)
	var version struct {
		Checksum string `json:"checksum"`
	}
	if err := f.fetchJSON(baseURL+"/api/clipboard/checksum", &version); err != nil {
		return nil, false, err
	}
	if version.Checksum == "" {
		return nil, false, fmt.Errorf("missing clipboard checksum")
	}
	if version.Checksum == state.committed.Checksum {
		if state.pending != nil {
			state.pending = nil
			state.prune()
		}
		return nil, false, nil
	}

	if state.pending == nil || version.Checksum != state.pending.Checksum {
		var manifest clipboard.Manifest
		if err := f.fetchJSON(baseURL+"/api/clipboard", &manifest); err != nil {
			state.pending = nil
			state.prune()
			return nil, false, err
		}
		if manifest.Entries == nil || manifest.Checksum != clipboard.ManifestChecksum(manifest.Entries) {
			state.pending = nil
			state.prune()
			return nil, false, fmt.Errorf("invalid clipboard manifest checksum")
		}
		for _, entry := range manifest.Entries {
			if entry.ContentType != "text" && entry.ContentType != "image" {
				state.pending = nil
				state.prune()
				return nil, false, fmt.Errorf("unsupported clipboard content type %q", entry.ContentType)
			}
		}
		// The manifest can be newer than the checksum response.
		if manifest.Checksum == state.committed.Checksum {
			state.pending = nil
			state.prune()
			return nil, false, nil
		}
		state.pending = &manifest
		state.prune()
	}

	var fetchErr error
	for _, entry := range state.pending.Entries {
		key := contentKey{entry.ContentType, entry.Checksum}
		if _, ok := state.payloads[key]; ok {
			continue
		}
		data, err := f.fetchContent(baseURL, entry)
		if err != nil {
			if fetchErr == nil {
				fetchErr = err
			}
			continue
		}
		if entry.ContentType == "image" {
			state.payloads[key] = base64.StdEncoding.EncodeToString(data)
		} else {
			state.payloads[key] = string(data)
		}
	}
	if fetchErr != nil {
		return nil, false, fetchErr
	}

	entries := make([]clipboard.ClipboardEntry, len(state.pending.Entries))
	for i, entry := range state.pending.Entries {
		entries[i] = clipboard.ClipboardEntry{
			ID:            entry.ID,
			Checksum:      entry.Checksum,
			ContentType:   entry.ContentType,
			ImageMimeType: entry.ImageMimeType,
			Timestamp:     entry.Timestamp,
		}
		payload := state.payloads[contentKey{entry.ContentType, entry.Checksum}]
		if entry.ContentType == "image" {
			entries[i].ImageData = payload
		} else {
			entries[i].Content = payload
		}
	}
	state.committed = *state.pending
	state.pending = nil
	state.prune()
	return entries, true, nil
}

func (f *Fetcher) get(endpoint string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Omaclip-Pass", f.passphraseStore.Hash())
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close() //nolint:errcheck
		return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, endpoint)
	}
	return resp, nil
}

func (f *Fetcher) fetchJSON(endpoint string, result any) error {
	resp, err := f.get(endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck
	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}

func (f *Fetcher) fetchContent(baseURL string, entry clipboard.ManifestEntry) ([]byte, error) {
	resp, err := f.get(baseURL + "/api/clipboard/" + url.PathEscape(entry.ID) + "/content")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	checksum := sha256.Sum256(data)
	if hex.EncodeToString(checksum[:]) != entry.Checksum {
		return nil, fmt.Errorf("content checksum mismatch for %s", entry.ID)
	}
	return data, nil
}
