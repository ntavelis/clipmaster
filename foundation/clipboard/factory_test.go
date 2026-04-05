package clipboard

import (
	"testing"
)

func mockAvailable(bins ...string) func(string) bool {
	set := make(map[string]bool)
	for _, b := range bins {
		set[b] = true
	}
	return func(bin string) bool {
		return set[bin]
	}
}

func TestNewReaderWriter(t *testing.T) {
	tests := []struct {
		name        string
		available   []string
		forceXclip  bool
		gtkAvail    bool
		wantBackend string
		wantErr     bool
	}{
		{
			name:        "gtk preferred over all CLI backends",
			available:   []string{"wl-paste", "xclip"},
			gtkAvail:    true,
			wantBackend: "gtk3 (native)",
		},
		{
			name:        "wayland selected when wl-paste available",
			available:   []string{"wl-paste"},
			wantBackend: "wayland (wl-paste)",
		},
		{
			name:        "wayland preferred over xclip",
			available:   []string{"wl-paste", "xclip"},
			wantBackend: "wayland (wl-paste)",
		},
		{
			name:        "xclip selected when no wl-paste",
			available:   []string{"xclip"},
			wantBackend: "x11 (xclip)",
		},
		{
			name:        "xsel selected when no wl-paste or xclip",
			available:   []string{"xsel"},
			wantBackend: "x11 (xsel)",
		},
		{
			name:        "darwin osascript preferred over pbpaste alone",
			available:   []string{"pbpaste", "osascript"},
			wantBackend: "darwin (osascript)",
		},
		{
			name:        "darwin pbpaste when no osascript",
			available:   []string{"pbpaste"},
			wantBackend: "darwin (pbpaste)",
		},
		{
			name:    "error when nothing available",
			wantErr: true,
		},
		{
			name:        "force xclip overrides wl-paste",
			available:   []string{"wl-paste", "xclip"},
			forceXclip:  true,
			wantBackend: "x11 (xclip, forced)",
		},
		{
			name:       "force xclip errors when xclip not available",
			available:  []string{"wl-paste"},
			forceXclip: true,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalAvail := availableFn
			originalGTK := newGTKBackendFn
			availableFn = mockAvailable(tt.available...)
			if tt.gtkAvail {
				newGTKBackendFn = func() (Reader, Writer, string, bool) {
					g := GTKClipboard{}
					return g, g, "gtk3 (native)", true
				}
			} else {
				newGTKBackendFn = func() (Reader, Writer, string, bool) { return nil, nil, "", false }
			}
			t.Cleanup(func() {
				availableFn = originalAvail
				newGTKBackendFn = originalGTK
			})

			_, _, backend, err := NewReaderWriter(tt.forceXclip)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if backend != tt.wantBackend {
				t.Errorf("backend = %q, want %q", backend, tt.wantBackend)
			}
		})
	}
}
