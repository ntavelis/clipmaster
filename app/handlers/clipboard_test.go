package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rhemvi/omaclip/business/clipboard"
	"github.com/rhemvi/omaclip/business/passphrase"
)

type fakeMonitor struct {
	history []clipboard.ClipboardEntry
}

func (f *fakeMonitor) GetHistory() []clipboard.ClipboardEntry { return f.history }
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

func TestGetClipboard_SkipsRejectedAndFillsLimit(t *testing.T) {
	// History is returned most-recent-first; mix rejected entries among valid ones.
	history := []clipboard.ClipboardEntry{
		{ID: "1", ContentType: "text", Content: "a"},
		{ID: "2", ContentType: "image-rejected", Content: "rejected"},
		{ID: "3", ContentType: "text", Content: "b"},
		{ID: "4", ContentType: "image-rejected", Content: "rejected"},
		{ID: "5", ContentType: "text", Content: "c"},
		{ID: "6", ContentType: "text", Content: "d"},
		{ID: "7", ContentType: "text", Content: "e"},
	}

	h := &ClipboardHandler{
		Monitor:    &fakeMonitor{history: history},
		MaxHistory: 5,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/clipboard", nil)
	rec := httptest.NewRecorder()
	h.GetClipboard(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var entries []clipboard.ClipboardEntry
	if err := json.NewDecoder(rec.Body).Decode(&entries); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(entries) != 5 {
		t.Errorf("got %d entries, want 5", len(entries))
	}
	for _, e := range entries {
		if e.ContentType == "image-rejected" {
			t.Errorf("entry %s with image-rejected type should not be returned", e.ID)
		}
	}
}
