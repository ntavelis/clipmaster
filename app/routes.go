package app

import (
	"clipmaster/app/handlers"
	bsync "clipmaster/business/sync"
)

func registerRoutes(s *bsync.Server, h *handlers.ClipboardHandler) {
	s.Handle("GET /api/clipboard", h.GetClipboard)
}
