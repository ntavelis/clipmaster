//go:build !linux

package clipboard

// newGTKBackend is a no-op on non-Linux platforms.
func newGTKBackend() (Reader, Writer, string, bool) {
	return nil, nil, "", false
}
