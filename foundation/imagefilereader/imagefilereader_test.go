package imagefilereader

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestIsImage(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/tmp/photo.png", true},
		{"/tmp/photo.PNG", true},
		{"/tmp/photo.jpg", true},
		{"/tmp/photo.JPEG", true},
		{"/tmp/photo.gif", true},
		{"/tmp/photo.bmp", true},
		{"/tmp/photo.tiff", true},
		{"/tmp/photo.tif", true},
		{"/tmp/photo.webp", true},
		{"/tmp/photo.avif", true},
		{"/tmp/photo.heic", true},
		{"/tmp/photo.heif", true},
		{"/tmp/photo.txt", false},
		{"/tmp/photo.pdf", false},
		{"/tmp/noext", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := IsImage(tt.path); got != tt.want {
				t.Errorf("IsImage(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestReadImageFile_Success(t *testing.T) {
	f := createTempFile(t, "test-*.png", 100)
	r := NewReader(1, 1)

	data, err := r.ReadImageFile(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 100 {
		t.Errorf("got %d bytes, want 100", len(data))
	}
}

func TestReadImageFile_TooLarge(t *testing.T) {
	f := createTempFile(t, "test-*.jpg", 3*1024*1024)
	r := NewReader(5, 2)

	_, err := r.ReadImageFile(f)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrImageTooLarge) {
		t.Errorf("expected ErrImageTooLarge, got: %v", err)
	}
}

func TestReadImageFile_PngVsNonPngLimit(t *testing.T) {
	r := NewReader(5, 1)
	size := 3 * 1024 * 1024

	png := createTempFile(t, "test-*.png", size)
	data, err := r.ReadImageFile(png)
	if err != nil {
		t.Fatalf("PNG should be within 5 MB limit, got error: %v", err)
	}
	if len(data) != size {
		t.Errorf("got %d bytes, want %d", len(data), size)
	}

	jpg := createTempFile(t, "test-*.jpg", size)
	_, err = r.ReadImageFile(jpg)
	if !errors.Is(err, ErrImageTooLarge) {
		t.Errorf("JPG 3 MB should exceed 1 MB non-PNG limit, got: %v", err)
	}
}

func TestReadImageFile_NonExistent(t *testing.T) {
	r := NewReader(5, 2)
	_, err := r.ReadImageFile("/tmp/does-not-exist-omaclip-test.png")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func createTempFile(t *testing.T, pattern string, size int) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), pattern)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(make([]byte, size)); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return filepath.Join(f.Name())
}
