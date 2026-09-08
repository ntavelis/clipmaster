// Package clipboard manages clipboard history and monitors the system clipboard for new entries.
package clipboard

import (
	"encoding/json"
	"time"
)

// ClipboardEntry represents a single item captured from the clipboard.
type ClipboardEntry struct {
	ID            string    `json:"id"`
	Checksum      string    `json:"checksum"`
	Content       string    `json:"content"`
	ContentType   string    `json:"contentType"`
	ImageData     string    `json:"imageData,omitempty"`
	ImageMimeType string    `json:"imageMimeType,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
}

// ManifestEntry describes shared clipboard content without including its payload.
type ManifestEntry struct {
	ID            string    `json:"id"`
	Checksum      string    `json:"checksum"`
	ContentType   string    `json:"contentType"`
	ImageMimeType string    `json:"imageMimeType,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
}

// Manifest is a coherent newest-first snapshot of the shared clipboard history.
type Manifest struct {
	Checksum string          `json:"checksum"`
	Entries  []ManifestEntry `json:"entries"`
}

// ManifestChecksum fingerprints the JSON representation of ordered metadata.
// An empty history is encoded as [] rather than null.
func ManifestChecksum(entries []ManifestEntry) string {
	if entries == nil {
		entries = []ManifestEntry{}
	}
	data, err := json.Marshal(entries)
	if err != nil {
		panic(err)
	}
	return sha256Hex(data)
}

// ContentType identifies a supported clipboard payload type.
type ContentType uint8

const (
	ContentTypeUnknown ContentType = iota
	ContentTypeText
	ContentTypeImage
)

// ParseContentType maps the wire-format content type to its enum value.
func ParseContentType(contentType string) ContentType {
	switch contentType {
	case ContentTypeText.String():
		return ContentTypeText
	case ContentTypeImage.String():
		return ContentTypeImage
	default:
		return ContentTypeUnknown
	}
}

// String returns the wire-format value for a content type.
func (contentType ContentType) String() string {
	switch contentType {
	case ContentTypeText:
		return "text"
	case ContentTypeImage:
		return "image"
	default:
		return ""
	}
}

// IsSupportedContentType reports whether the wire-format content type is supported.
func IsSupportedContentType(contentType string) bool {
	return ParseContentType(contentType) != ContentTypeUnknown
}
