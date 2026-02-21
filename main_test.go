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
		if command.Report != "default" {
			t.Fatalf("expected default report option, got %q", command.Report)
		}
	}
}

func TestParseArgsReportCommandWithNewOption(t *testing.T) {
	t.Parallel()

	command, err := parseArgs([]string{"report", "daily", "new"})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	if command.Name != "report" {
		t.Fatalf("expected command report, got %q", command.Name)
	}
	if command.Cadence != "daily" {
		t.Fatalf("expected cadence daily, got %q", command.Cadence)
	}
	if command.Report != "new" {
		t.Fatalf("expected report option new, got %q", command.Report)
	}
}

func TestParseArgsReportCommandWithExcludedOption(t *testing.T) {
	t.Parallel()

	command, err := parseArgs([]string{"report", "weekly", "excluded"})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	if command.Name != "report" {
		t.Fatalf("expected command report, got %q", command.Name)
	}
	if command.Cadence != "weekly" {
		t.Fatalf("expected cadence weekly, got %q", command.Cadence)
	}
	if command.Report != "excluded" {
		t.Fatalf("expected report option excluded, got %q", command.Report)
	}
}

func TestParseArgsReportCommandWithUnknownOptionFails(t *testing.T) {
	t.Parallel()

	_, err := parseArgs([]string{"report", "daily", "invalid"})
	if err == nil {
		t.Fatal("expected an error for unknown report option")
	}
	if err.Error() != "unknown report option: invalid" {
		t.Fatalf("expected unknown report option error, got %q", err.Error())
	}
}

func TestParseArgsReportCommandWithTooManyOptionsFails(t *testing.T) {
	t.Parallel()

	_, err := parseArgs([]string{"report", "daily", "new", "excluded"})
	if err == nil {
		t.Fatal("expected an error for too many report options")
	}
	if err.Error() != "too many report options" {
		t.Fatalf("expected too many report options error, got %q", err.Error())
	}
}

func TestParseArgsRunCommandWithOptionFails(t *testing.T) {
	t.Parallel()

	_, err := parseArgs([]string{"run", "daily", "new"})
	if err == nil {
		t.Fatal("expected an error when run receives options")
	}
	if err.Error() != "run does not accept options" {
		t.Fatalf("expected run options error, got %q", err.Error())
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

func TestRunCLIExecutesReportNewOption(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runCLI([]string{"report", "daily", "new"}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	if stdout.String() != "daily backup report (new) is not implemented yet.\n" {
		t.Fatalf("unexpected stdout output: %q", stdout.String())
	}
}

func TestRunCLIExecutesReportExcludedOption(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runCLI([]string{"report", "weekly", "excluded"}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	if stdout.String() != "weekly backup report (excluded) is not implemented yet.\n" {
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
