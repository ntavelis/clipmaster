// Package clipboard provides OS-level clipboard access.
package clipboard

import (
	"fmt"
	"os/exec"
)

// WaylandClipboard reads and writes the system clipboard via wl-paste/wl-copy.
type WaylandClipboard struct{}

// GetText returns the current clipboard contents using wl-paste.
func (w WaylandClipboard) GetText() (string, error) {
	cmd := exec.Command("wl-paste", "--no-newline")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("wl-paste: %w", err)
	}
	return string(out), nil
}
