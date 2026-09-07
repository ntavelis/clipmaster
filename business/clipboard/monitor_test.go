package clipboard

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log/slog"
	"reflect"
	"testing"
	"time"

	"github.com/rhemvi/omaclip/foundation/imagefilereader"
)

// mockReader implements Reader for testing.
type mockReader struct {
	text    string
	textErr error
	img     []byte
	imgErr  error
}

func (r *mockReader) GetText(_ context.Context) (string, error)  { return r.text, r.textErr }
func (r *mockReader) GetImage(_ context.Context) ([]byte, error) { return r.img, r.imgErr }

// makePNG generates a minimal valid 1x1 PNG image.
func makePNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.White)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// newTestMonitor creates a Monitor wired to a mockReader with sensible defaults.
func newTestMonitor(r *mockReader) *Monitor {
	return NewMonitor(
		slog.Default(),
		r,
		nil, // writer not needed for readClipboard
		50,
		5,
		2,
		0, // pollInterval unused in direct calls
	)
}

func TestReadClipboard_NewText(t *testing.T) {
	r := &mockReader{text: "hello"}
	m := newTestMonitor(r)

	m.readClipboard(context.Background())

	h := m.GetHistory()
	if len(h) != 1 {
		t.Fatalf("got %d entries, want 1", len(h))
	}
	if h[0].ContentType != "text" || h[0].Content != "hello" {
		t.Errorf("got entry %+v, want text=hello", h[0])
	}
	hash := sha256.Sum256([]byte("hello"))
	if h[0].Checksum != hex.EncodeToString(hash[:]) {
		t.Errorf("text checksum = %s, want SHA256 of exact text bytes", h[0].Checksum)
	}
}

func TestReadClipboard_DuplicateTextIgnored(t *testing.T) {
	r := &mockReader{text: "hello"}
	m := newTestMonitor(r)

	m.readClipboard(context.Background())
	m.readClipboard(context.Background())

	if got := len(m.GetHistory()); got != 1 {
		t.Errorf("got %d entries, want 1 (duplicate should be ignored)", got)
	}
}

func TestReadClipboard_EmptyTextIgnored(t *testing.T) {
	r := &mockReader{text: ""}
	m := newTestMonitor(r)

	m.readClipboard(context.Background())

	if got := len(m.GetHistory()); got != 0 {
		t.Errorf("got %d entries, want 0", got)
	}
}

func TestReadClipboard_TextErrorIgnored(t *testing.T) {
	r := &mockReader{textErr: fmt.Errorf("read error"), img: makePNG(t)}
	m := newTestMonitor(r)

	m.readClipboard(context.Background())

	h := m.GetHistory()
	if len(h) != 1 {
		t.Fatalf("got %d entries, want 1", len(h))
	}
	if h[0].ContentType != "image" {
		t.Errorf("got contentType=%s, want image", h[0].ContentType)
	}
}

func TestReadClipboard_NewImage(t *testing.T) {
	pngData := makePNG(t)
	r := &mockReader{img: pngData}
	m := newTestMonitor(r)

	m.readClipboard(context.Background())

	h := m.GetHistory()
	if len(h) != 1 {
		t.Fatalf("got %d entries, want 1", len(h))
	}
	if h[0].ContentType != "image" {
		t.Errorf("got contentType=%s, want image", h[0].ContentType)
	}
	if h[0].ImageMimeType != "image/png" {
		t.Errorf("got mime=%s, want image/png", h[0].ImageMimeType)
	}
	hash := sha256.Sum256(pngData)
	if h[0].Checksum != hex.EncodeToString(hash[:]) {
		t.Errorf("image checksum = %s, want SHA256 of raw image bytes", h[0].Checksum)
	}
}

func TestReadClipboard_DuplicateImageIgnored(t *testing.T) {
	pngData := makePNG(t)
	r := &mockReader{img: pngData}
	m := newTestMonitor(r)

	m.readClipboard(context.Background())
	m.readClipboard(context.Background())

	if got := len(m.GetHistory()); got != 1 {
		t.Errorf("got %d entries, want 1 (duplicate should be ignored)", got)
	}
}

func TestReadClipboard_ImageTooLarge_AddedOnce(t *testing.T) {
	tooLargeErr := fmt.Errorf("%w: photo.jpg is 10.00 MB, limit is 2 MB", imagefilereader.ErrImageTooLarge)
	r := &mockReader{imgErr: tooLargeErr}
	m := newTestMonitor(r)

	m.readClipboard(context.Background())
	m.readClipboard(context.Background())
	m.readClipboard(context.Background())

	h := m.GetHistory()
	if len(h) != 1 {
		t.Fatalf("got %d entries, want 1 (repeated rejection should be deduped)", len(h))
	}
	if h[0].ContentType != "image-rejected" {
		t.Errorf("got contentType=%s, want image-rejected", h[0].ContentType)
	}
}

func TestReadClipboard_ImageTooLarge_DifferentFileNotDeduped(t *testing.T) {
	r := &mockReader{
		imgErr: fmt.Errorf("%w: photo1.jpg is 10.00 MB, limit is 2 MB", imagefilereader.ErrImageTooLarge),
	}
	m := newTestMonitor(r)

	m.readClipboard(context.Background())

	r.imgErr = fmt.Errorf("%w: photo2.jpg is 8.00 MB, limit is 2 MB", imagefilereader.ErrImageTooLarge)
	m.readClipboard(context.Background())

	h := m.GetHistory()
	if len(h) != 2 {
		t.Fatalf("got %d entries, want 2 (different rejections should both appear)", len(h))
	}
}

func TestReadClipboard_ImageExceedsSizeLimit_Rejected(t *testing.T) {
	// Create image data that exceeds the 2 MB non-PNG limit.
	// Use non-PNG bytes so http.DetectContentType won't return image/png.
	bigData := make([]byte, 3*1024*1024)
	bigData[0] = 0xFF // not a PNG header
	r := &mockReader{img: bigData}
	m := newTestMonitor(r)

	m.readClipboard(context.Background())

	h := m.GetHistory()
	if len(h) != 1 {
		t.Fatalf("got %d entries, want 1", len(h))
	}
	if h[0].ContentType != "image-rejected" {
		t.Errorf("got contentType=%s, want image-rejected", h[0].ContentType)
	}
}

func TestReadClipboard_ImageExceedsSizeLimit_DuplicateIgnored(t *testing.T) {
	bigData := make([]byte, 3*1024*1024)
	bigData[0] = 0xFF
	r := &mockReader{img: bigData}
	m := newTestMonitor(r)

	m.readClipboard(context.Background())
	m.readClipboard(context.Background())

	if got := len(m.GetHistory()); got != 1 {
		t.Errorf("got %d entries, want 1 (duplicate should be ignored)", got)
	}
}

func TestReadClipboard_TextAndImage(t *testing.T) {
	pngData := makePNG(t)
	r := &mockReader{text: "hello", img: pngData}
	m := newTestMonitor(r)

	m.readClipboard(context.Background())

	h := m.GetHistory()
	if len(h) != 2 {
		t.Fatalf("got %d entries, want 2", len(h))
	}
	// History is reverse-chronological: image first, then text.
	if h[0].ContentType != "image" {
		t.Errorf("entry[0] got contentType=%s, want image", h[0].ContentType)
	}
	if h[1].ContentType != "text" {
		t.Errorf("entry[1] got contentType=%s, want text", h[1].ContentType)
	}
}

func TestReadClipboard_TextAfterImage(t *testing.T) {
	pngData := makePNG(t)
	r := &mockReader{img: pngData}
	m := newTestMonitor(r)

	m.readClipboard(context.Background())

	r.img = nil
	r.text = "new text"
	m.readClipboard(context.Background())

	h := m.GetHistory()
	if len(h) != 2 {
		t.Fatalf("got %d entries, want 2", len(h))
	}
	if h[0].ContentType != "text" || h[0].Content != "new text" {
		t.Errorf("entry[0] got %+v, want text=new text", h[0])
	}
}

func TestReadClipboard_OnNewEntryCallback(t *testing.T) {
	r := &mockReader{text: "hello"}
	m := newTestMonitor(r)

	var called int
	m.OnNewEntry = func(_ ClipboardEntry) { called++ }

	m.readClipboard(context.Background())

	if called != 1 {
		t.Errorf("OnNewEntry called %d times, want 1", called)
	}
}

func TestReadClipboard_MaxHistoryTrimmed(t *testing.T) {
	r := &mockReader{}
	m := newTestMonitor(r)
	m.maxHistory = 3

	for i := range 5 {
		r.text = fmt.Sprintf("text-%d", i)
		m.readClipboard(context.Background())
	}

	h := m.GetHistory()
	if len(h) != 3 {
		t.Fatalf("got %d entries, want 3", len(h))
	}
	if h[0].Content != "text-4" {
		t.Errorf("newest entry got %s, want text-4", h[0].Content)
	}
	if h[2].Content != "text-2" {
		t.Errorf("oldest entry got %s, want text-2", h[2].Content)
	}
}

func TestManifest_SharedWindowAndRejectedTrimming(t *testing.T) {
	m := newTestMonitor(&mockReader{})
	m.maxHistory = 3
	m.SetRemoteMaxHistory(2)
	for _, id := range []string{"a", "b", "c"} {
		m.addEntry(ClipboardEntry{ID: id, ContentType: "text", Checksum: sha256Hex([]byte(id))})
	}
	before := m.GetManifest()
	if got := []string{before.Entries[0].ID, before.Entries[1].ID}; !reflect.DeepEqual(got, []string{"c", "b"}) {
		t.Fatalf("shared window = %v, want [c b]", got)
	}
	// Removing an entry outside the shared window must not invalidate the checksum.
	m.addEntry(ClipboardEntry{ID: "rejected-1", ContentType: "image-rejected"})
	if got := m.GetChecksum(); got != before.Checksum {
		t.Fatalf("excluded history change altered checksum: %s != %s", got, before.Checksum)
	}
	// A rejected entry can still evict an advertised entry from local history.
	m.addEntry(ClipboardEntry{ID: "rejected-2", ContentType: "image-rejected"})
	after := m.GetManifest()
	if len(after.Entries) != 1 || after.Entries[0].ID != "c" || after.Checksum == before.Checksum {
		t.Fatalf("trimmed manifest = %+v, want only c and a changed checksum", after)
	}
	if before.Entries[1].ID != "b" || before.Checksum != ManifestChecksum(before.Entries) {
		t.Fatal("previously returned manifest changed after history mutation")
	}
}

func TestManifest_PinnedHistoryAndWindowChanges(t *testing.T) {
	m := newTestMonitor(&mockReader{})
	m.maxHistory = 2
	m.SetRemoteMaxHistory(3)
	m.addEntry(ClipboardEntry{ID: "pinned", ContentType: "text"})
	m.SetPinnedIDs([]string{"pinned"})
	m.addEntry(ClipboardEntry{ID: "old", ContentType: "text"})
	m.addEntry(ClipboardEntry{ID: "new", ContentType: "text"})
	before := m.GetManifest()
	if len(before.Entries) != 2 || before.Entries[0].ID != "new" || before.Entries[1].ID != "pinned" {
		t.Fatalf("pinned history manifest = %+v", before)
	}
	m.SetRemoteMaxHistory(1)
	after := m.GetManifest()
	if len(after.Entries) != 1 || after.Entries[0].ID != "new" || after.Checksum == before.Checksum {
		t.Fatalf("reconfigured window manifest = %+v", after)
	}
	after.Entries[0].ID = "caller mutation"
	if got := m.GetManifest(); got.Entries[0].ID != "new" || got.Checksum != ManifestChecksum(got.Entries) {
		t.Fatalf("caller mutated published snapshot: %+v", got)
	}
	m.SetRemoteMaxHistory(0)
	empty := m.GetManifest()
	if empty.Entries == nil || len(empty.Entries) != 0 || empty.Checksum != ManifestChecksum([]ManifestEntry{}) {
		t.Fatalf("empty manifest = %+v", empty)
	}
}

func TestManifestChecksum_MetadataAndOrder(t *testing.T) {
	entries := []ManifestEntry{
		{ID: "a", Checksum: "payload-a", ContentType: "image", ImageMimeType: "image/png", Timestamp: time.Unix(2, 0).UTC()},
		{ID: "b", Checksum: "payload-b", ContentType: "text", Timestamp: time.Unix(1, 0).UTC()},
	}
	data, err := json.Marshal(entries)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(data)
	original := ManifestChecksum(entries)
	if original != hex.EncodeToString(hash[:]) {
		t.Fatalf("manifest checksum does not match SHA256(JSON(entries))")
	}
	changes := map[string]func([]ManifestEntry){
		"id":        func(e []ManifestEntry) { e[0].ID = "replacement" },
		"payload":   func(e []ManifestEntry) { e[0].Checksum = "changed" },
		"type":      func(e []ManifestEntry) { e[0].ContentType = "text" },
		"mime":      func(e []ManifestEntry) { e[0].ImageMimeType = "image/jpeg" },
		"timestamp": func(e []ManifestEntry) { e[0].Timestamp = e[0].Timestamp.Add(time.Second) },
		"order":     func(e []ManifestEntry) { e[0], e[1] = e[1], e[0] },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			changed := append([]ManifestEntry(nil), entries...)
			change(changed)
			if ManifestChecksum(changed) == original {
				t.Fatal("advertised metadata change did not change fingerprint")
			}
		})
	}
}
