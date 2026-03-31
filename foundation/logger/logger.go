// Package logger provides a structured logger built on top of log/slog.
package logger

import (
	"log/slog"
	"os"
)

func New() *slog.Logger {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	})

	return slog.New(handler)
}
