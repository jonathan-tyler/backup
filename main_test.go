package main

import (
	"bytes"
	"testing"
)

func TestParseArgsRunCommand(t *testing.T) {
	t.Parallel()

	for _, cadence := range []string{"daily", "weekly", "monthly"} {
		command, err := parseArgs([]string{"run", cadence})
		if err != nil {
			t.Fatalf("parseArgs returned error for cadence %q: %v", cadence, err)
		}
		if command.Name != "run" {
			t.Fatalf("expected command run, got %q", command.Name)
		}
		if command.Cadence != cadence {
			t.Fatalf("expected cadence %q, got %q", cadence, command.Cadence)
		}
	}
}

func TestParseArgsReportCommand(t *testing.T) {
	t.Parallel()

	for _, cadence := range []string{"daily", "weekly", "monthly"} {
		command, err := parseArgs([]string{"report", cadence})
		if err != nil {
			t.Fatalf("parseArgs returned error for cadence %q: %v", cadence, err)
		}
		if command.Name != "report" {
			t.Fatalf("expected command report, got %q", command.Name)
		}
		if command.Cadence != cadence {
			t.Fatalf("expected cadence %q, got %q", cadence, command.Cadence)
		}
	}
}

func TestParseArgsRestoreCommand(t *testing.T) {
	t.Parallel()

	command, err := parseArgs([]string{"restore", "target-dir"})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	if command.Name != "restore" {
		t.Fatalf("expected command restore, got %q", command.Name)
	}
	if command.Target != "target-dir" {
		t.Fatalf("expected target target-dir, got %q", command.Target)
	}
}

func TestRunUnknownCommandReturnsError(t *testing.T) {
	t.Parallel()

	_, err := run(Command{Name: "invalid"})
	if err == nil {
		t.Fatal("expected an error for unknown command")
	}
	if err.Error() != "Unknown command: invalid" {
		t.Fatalf("expected Unknown command error, got %q", err.Error())
	}
}

func TestRunCLIExecutesRunCommand(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runCLI([]string{"run", "daily"}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	if stdout.String() != "daily backup run is not implemented yet.\n" {
		t.Fatalf("unexpected stdout output: %q", stdout.String())
	}
}

func TestRunCLIHelpCommand(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runCLI([]string{"help"}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	if stdout.String() == "" {
		t.Fatal("expected usage output, got empty stdout")
	}
}

func TestRunCLIHelpFlag(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runCLI([]string{"--help"}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	if stdout.String() == "" {
		t.Fatal("expected usage output, got empty stdout")
	}
}

func TestRunCLINoArgsShowsUsage(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runCLI([]string{}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	if stdout.String() == "" {
		t.Fatal("expected usage output, got empty stdout")
	}
}
