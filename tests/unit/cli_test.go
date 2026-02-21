package unit

import (
	"bytes"
	"testing"

	backup "wsl-backup-cli/src"
)

func TestParseArgsRunCommand(t *testing.T) {
	t.Parallel()

	command, err := backup.ParseArgs([]string{"run", "daily"})
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if command.Name != "run" || command.Cadence != "daily" {
		t.Fatalf("unexpected command: %#v", command)
	}
}

func TestParseArgsRunRejectsOptions(t *testing.T) {
	t.Parallel()

	_, err := backup.ParseArgs([]string{"run", "daily", "wsl"})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "run does not accept options" {
		t.Fatalf("unexpected error: %q", err.Error())
	}
}

func TestParseArgsReportModes(t *testing.T) {
	t.Parallel()

	command, err := backup.ParseArgs([]string{"report", "weekly", "excluded"})
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if command.Report != "excluded" {
		t.Fatalf("expected excluded report mode, got %q", command.Report)
	}
}

func TestRunCLIHelpCommand(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := backup.RunCLI([]string{"help"}, &stdout, &stderr, backup.SystemExecutor{})

	if exitCode != 0 || stderr.String() != "" || stdout.String() == "" {
		t.Fatalf("unexpected cli result: code=%d stderr=%q stdout=%q", exitCode, stderr.String(), stdout.String())
	}
}
