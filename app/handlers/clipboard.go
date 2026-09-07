// Package handlers contains HTTP handlers for the sync server.
package handlers

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"

	"github.com/rhemvi/omaclip/business/clipboard"
	"github.com/rhemvi/omaclip/business/passphrase"
)

// clipboardMonitor is the subset of clipboard.Monitor used by ClipboardHandler.
type clipboardMonitor interface {
	GetChecksum() string
	GetManifest() clipboard.Manifest
	GetEntry(id string) (clipboard.ClipboardEntry, bool)
}

// ClipboardHandler holds dependencies for all HTTP handlers.
type ClipboardHandler struct {
	Monitor         clipboardMonitor
	PassphraseStore *passphrase.Store
}

// RequirePassphrase returns middleware that validates the X-Omaclip-Pass header.
func RequirePassphrase(store *passphrase.Store, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Omaclip-Pass")), []byte(store.Hash())) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// GetClipboardChecksum returns the precomputed fingerprint of the shared history.
func (h *ClipboardHandler) GetClipboardChecksum(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := struct {
		Checksum string `json:"checksum"`
	}{Checksum: h.Monitor.GetChecksum()}
	json.NewEncoder(w).Encode(response)
}

// GetClipboard returns a payload-free manifest with its own coherent fingerprint.
func (h *ClipboardHandler) GetClipboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.Monitor.GetManifest())
}

// GetClipboardContent returns the exact stored text or raw image bytes for an entry.
func (h *ClipboardHandler) GetClipboardContent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	entry, ok := h.Monitor.GetEntry(id)
	if !ok || (entry.ContentType != "text" && entry.ContentType != "image") {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if entry.ContentType == "text" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		io.WriteString(w, entry.Content) //nolint:errcheck
		return
	}

	imgBytes, err := base64.StdEncoding.DecodeString(entry.ImageData)
	if err != nil {
		http.Error(w, "corrupt image data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", entry.ImageMimeType)
	w.Write(imgBytes) //nolint:errcheck
}
