// Package copyhook runs a user-configured command after an item is copied from Omaclip.
package copyhook

import (
	"log/slog"
	"os/exec"
)

// Runner triggers a configured post-copy action.
type Runner interface {
	Trigger()
}

type commandRunner struct {
	log     *slog.Logger
	command string
}

// Trigger starts the hook through the system shell and logs failures asynchronously.
func (r *commandRunner) Trigger() {
	cmd := exec.Command("sh", "-c", r.command)
	if err := cmd.Start(); err != nil {
		r.log.Error("failed to start copy hook", "error", err)
		return
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			r.log.Error("copy hook failed", "error", err)
		}
	}()
}
