// Package copyhook runs a user-configured command after an item is copied from Omaclip.
package copyhook

import (
	"log/slog"
	"os/exec"
)

// Runner starts the configured copy hook without blocking the clipboard operation.
type Runner struct {
	log     *slog.Logger
	command string
}

// NewRunner creates a copy hook runner. An empty command disables the hook.
func NewRunner(log *slog.Logger, command string) *Runner {
	return &Runner{log: log, command: command}
}

// Trigger starts the hook through the system shell and logs failures asynchronously.
func (r *Runner) Trigger() {
	if r.command == "" {
		return
	}

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
