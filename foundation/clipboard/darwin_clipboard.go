package clipboard

import (
	"fmt"
	"os/exec"
	"strings"
)

// DarwinClipboard reads the system clipboard via pbpaste (macOS).
type DarwinClipboard struct{}

// GetText returns the current clipboard contents using pbpaste.
func (d DarwinClipboard) GetText() (string, error) {
	cmd := exec.Command("pbpaste")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pbpaste: %w", err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}
