package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rhemvi/omaclip/business/clipboard"
	"github.com/rhemvi/omaclip/business/passphrase"
)

type fakeMonitor struct {
	history  []clipboard.ClipboardEntry
	manifest clipboard.Manifest
}

func (f *fakeMonitor) GetChecksum() string             { return f.manifest.Checksum }
func (f *fakeMonitor) GetManifest() clipboard.Manifest { return f.manifest }
func (f *fakeMonitor) GetEntry(id string) (clipboard.ClipboardEntry, bool) {
	for _, e := range f.history {
		if e.ID == id {
			return e, true
		}
	}
	return clipboard.ClipboardEntry{}, false
}

func TestRequirePassphrase_Unauthorized(t *testing.T) {
	store := &passphrase.Store{}
	store.Set("correctpassphrase")

	dummy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RequirePassphrase(store, dummy)

	tests := []struct {
		name       string
		headerVal  string
		wantStatus int
	}{
		{"missing header", "", http.StatusUnauthorized},
		{"wrong passphrase", "wronghash", http.StatusUnauthorized},
		{"correct hash", store.Hash(), http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/clipboard", nil)
			if tt.headerVal != "" {
				req.Header.Set("X-Omaclip-Pass", tt.headerVal)
			}
			rec := httptest.NewRecorder()
			handler(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestGetClipboard_PayloadFreeManifest(t *testing.T) {
	entries := []clipboard.ManifestEntry{{
		ID: "text", Checksum: "content-checksum", ContentType: "text", Timestamp: time.Unix(1, 0).UTC(),
	}}
	manifest := clipboard.Manifest{Checksum: clipboard.ManifestChecksum(entries), Entries: entries}
	h := &ClipboardHandler{Monitor: &fakeMonitor{
		manifest: manifest,
		history:  []clipboard.ClipboardEntry{{ID: "text", ContentType: "text", Content: "private payload"}},
	}}
	rec := httptest.NewRecorder()
	h.GetClipboard(rec, httptest.NewRequest(http.MethodGet, "/api/clipboard", nil))
	var got clipboard.Manifest
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Checksum != manifest.Checksum || clipboard.ManifestChecksum(got.Entries) != got.Checksum {
		t.Fatalf("manifest fingerprint does not match response: %+v", got)
	}
	var raw struct {
		Entries []map[string]json.RawMessage `json:"entries"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	if len(raw.Entries) != 1 {
		t.Fatalf("got %d manifest entries, want 1", len(raw.Entries))
	}
	for _, key := range []string{"content", "imageData"} {
		if _, ok := raw.Entries[0][key]; ok {
			t.Errorf("manifest includes payload field %s", key)
		}
	}
}

func TestGetClipboardContent(t *testing.T) {
	text := " \ttext\r\nwith\x00bytes\n"
	image := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0, 0xff}
	h := &ClipboardHandler{Monitor: &fakeMonitor{history: []clipboard.ClipboardEntry{
		{ID: "text", ContentType: "text", Content: text},
		{ID: "image", ContentType: "image", ImageData: base64.StdEncoding.EncodeToString(image), ImageMimeType: "image/png"},
		{ID: "rejected", ContentType: "image-rejected", Content: "not shared"},
		{ID: "corrupt", ContentType: "image", ImageData: "invalid!"},
	}}}
	tests := []struct {
		id          string
		status      int
		contentType string
		body        []byte
	}{
		{"text", http.StatusOK, "text/plain; charset=utf-8", []byte(text)},
		{"image", http.StatusOK, "image/png", image},
		{"rejected", http.StatusNotFound, "", nil},
		{"missing", http.StatusNotFound, "", nil},
		{"corrupt", http.StatusInternalServerError, "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/clipboard/"+tt.id+"/content", nil)
			req.SetPathValue("id", tt.id)
			rec := httptest.NewRecorder()
			h.GetClipboardContent(rec, req)
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.status)
			}
			if tt.status == http.StatusOK {
				if got := rec.Header().Get("Content-Type"); got != tt.contentType {
					t.Errorf("Content-Type = %q, want %q", got, tt.contentType)
				}
				if !bytes.Equal(rec.Body.Bytes(), tt.body) {
					t.Errorf("content = %q, want exact bytes %q", rec.Body.Bytes(), tt.body)
				}
			}
		})
	}
}
