package main

import (
	"io"
	"os"
	"testing"
)

func TestRunPrintAgentSkill(t *testing.T) {
	originalArgs := os.Args
	originalStdout := os.Stdout
	t.Cleanup(func() {
		os.Args = originalArgs
		os.Stdout = originalStdout
	})

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	defer reader.Close()

	os.Args = []string{"omaclip", "--print-agent-skill"}
	os.Stdout = writer

	runErr := run()
	if err := writer.Close(); err != nil {
		t.Fatalf("close stdout writer: %v", err)
	}
	os.Stdout = originalStdout
	if runErr != nil {
		t.Fatalf("run: %v", runErr)
	}

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if string(output) != agentSkill {
		t.Fatalf("unexpected skill output:\n%s", output)
	}
}
