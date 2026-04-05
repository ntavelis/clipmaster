//go:build linux

package clipboard

// newGTKBackend returns the GTK clipboard backend on Linux.
func newGTKBackend() (Reader, Writer, string, bool) {
	g := GTKClipboard{}
	return g, g, "gtk3 (native)", true
}
