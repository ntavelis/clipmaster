package app

import (
	"github.com/rhemvi/omaclip/app/handlers"
	bsync "github.com/rhemvi/omaclip/business/sync"
)

func registerRoutes(s *bsync.Server, h *handlers.ClipboardHandler) {
	auth := handlers.RequirePassphrase
	s.Handle("GET /api/clipboard", auth(h.PassphraseStore, h.GetClipboard))
	s.Handle("GET /api/clipboard/{id}/image", auth(h.PassphraseStore, h.GetClipboardImage))
}
