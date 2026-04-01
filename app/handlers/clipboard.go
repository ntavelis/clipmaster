// Package handlers contains HTTP handlers for the sync server.
package handlers

import (
	"encoding/json"
	"net/http"

	"clipmaster/business/clipboard"
)

// ClipboardHandler holds dependencies for all HTTP handlers.
type ClipboardHandler struct {
	Monitor    *clipboard.Monitor
	MaxHistory int
}

// GetClipboard returns the last 5 clipboard entries as JSON.
func (h *ClipboardHandler) GetClipboard(w http.ResponseWriter, _ *http.Request) {
	all := h.Monitor.GetHistory()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(all[:min(h.MaxHistory, len(all))]) //nolint:errcheck
}
