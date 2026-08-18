package copyhook

import "log/slog"

// NewRunner selects a command runner when a hook is configured and a no-op runner otherwise.
func NewRunner(log *slog.Logger, command string) Runner {
	if command == "" {
		return nilRunner{}
	}
	return &commandRunner{log: log, command: command}
}
