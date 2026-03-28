package clipboard

import "time"

// ClipboardEntry represents a single item captured from the clipboard.
type ClipboardEntry struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}
