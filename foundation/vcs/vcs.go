// Package vcs returns the application version derived from embedded VCS build info.
// If the working tree was modified at build time, "_dirty" is appended.
package vcs

import (
	"fmt"
	"runtime/debug"
)

// Revision and Modified can be set at build time via ldflags:
//
//	-X clipmaster/foundation/vcs.Revision=$(git rev-parse --short HEAD)
//	-X clipmaster/foundation/vcs.Modified=$(git diff --quiet && echo false || echo true)
var Revision string
var Modified string

func Version() string {
	if Revision != "" {
		if Modified == "true" {
			return fmt.Sprintf("%s_dirty", Revision)
		}
		return Revision
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	var revision string
	var dirty bool

	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}

	if revision == "" {
		return "dev"
	}

	if dirty {
		return fmt.Sprintf("%s_dirty", revision)
	}
	return revision
}
