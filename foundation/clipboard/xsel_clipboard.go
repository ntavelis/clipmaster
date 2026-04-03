package clipboard

import (
	"fmt"
	"os/exec"
	"strings"
)

// XselClipboard reads the system clipboard via xsel (X11).
type XselClipboard struct{}

// GetText returns the current clipboard contents using xsel.
func (x XselClipboard) GetText() (string, error) {
	cmd := exec.Command("xsel", "--clipboard", "--output")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("xsel: %w", err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}
