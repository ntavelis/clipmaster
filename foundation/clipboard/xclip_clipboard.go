package clipboard

import (
	"fmt"
	"os/exec"
	"strings"
)

// XclipClipboard reads the system clipboard via xclip (X11).
type XclipClipboard struct{}

// GetText returns the current clipboard contents using xclip.
func (x XclipClipboard) GetText() (string, error) {
	cmd := exec.Command("xclip", "-selection", "clipboard", "-o")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("xclip: %w", err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}
