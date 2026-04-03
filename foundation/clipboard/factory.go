package clipboard

import (
	"errors"
	"os/exec"
)

var ErrNoClipAvailable = errors.New("no supported clipboard binary found (tried: wl-paste, xclip, xsel, pbpaste)")

// Reader abstracts clipboard reading across platform backends.
type Reader interface {
	GetText() (string, error)
}

// NewReader returns the first available clipboard reader by probing known binaries in order:
// wl-paste (Wayland) → xclip (X11) → xsel (X11) → pbpaste (macOS).
// Returns an error if none are found.
func NewReader() (Reader, error) {
	switch {
	case available("wl-paste"):
		return WaylandClipboard{}, nil
	case available("xclip"):
		return XclipClipboard{}, nil
	case available("xsel"):
		return XselClipboard{}, nil
	case available("pbpaste"):
		return DarwinClipboard{}, nil
	default:
		return nil, ErrNoClipAvailable
	}
}

func available(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}
