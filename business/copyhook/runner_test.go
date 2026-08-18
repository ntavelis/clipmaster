package copyhook

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRunnerTrigger(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "triggered")
	runner := NewRunner(slog.New(slog.NewTextHandler(io.Discard, nil)), "printf triggered > "+strconv.Quote(marker))

	runner.Trigger()

	waitFor(t, func() bool {
		data, err := os.ReadFile(marker)
		return err == nil && string(data) == "triggered"
	})
}

func TestNewRunnerSelectsNilRunnerForEmptyCommand(t *testing.T) {
	runner := NewRunner(slog.New(slog.NewTextHandler(io.Discard, nil)), "")

	if _, ok := runner.(nilRunner); !ok {
		t.Fatalf("NewRunner() returned %T, want nilRunner", runner)
	}
	runner.Trigger()
}

func TestNewRunnerSelectsCommandRunnerForCommand(t *testing.T) {
	runner := NewRunner(slog.New(slog.NewTextHandler(io.Discard, nil)), "true")

	if _, ok := runner.(*commandRunner); !ok {
		t.Fatalf("NewRunner() returned %T, want *commandRunner", runner)
	}
}

func TestRunnerLogsCommandFailure(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "hook.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatalf("create log file: %v", err)
	}
	defer logFile.Close()

	runner := NewRunner(slog.New(slog.NewTextHandler(logFile, nil)), "exit 7")
	runner.Trigger()

	waitFor(t, func() bool {
		data, err := os.ReadFile(logPath)
		return err == nil && strings.Contains(string(data), "copy hook failed")
	})
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition was not met before timeout")
}
