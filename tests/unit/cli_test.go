package unit

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
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

func TestParseArgsRunCommandStrictOverlap(t *testing.T) {
	t.Parallel()

	command, err := backup.ParseArgs([]string{"run", "daily", "--strict-overlap"})
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if !command.StrictOverlap {
		t.Fatalf("expected StrictOverlap=true, got %#v", command)
	}
}

func TestParseArgsRunRejectsOptions(t *testing.T) {
	t.Parallel()

	_, err := backup.ParseArgs([]string{"run", "daily", "wsl"})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "unknown run option: wsl" {
		t.Fatalf("unexpected error: %q", err.Error())
	}
}

func TestParseArgsRunRejectsMultipleOptions(t *testing.T) {
	t.Parallel()

	_, err := backup.ParseArgs([]string{"run", "daily", "--strict-overlap", "extra"})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "too many run options" {
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

func TestParseArgsRestoreCommand(t *testing.T) {
	t.Parallel()

	command, err := backup.ParseArgs([]string{"restore", "/tmp/restore"})
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if command.Name != "restore" || command.Target != "/tmp/restore" {
		t.Fatalf("unexpected command: %#v", command)
	}
}

func TestParseArgsRestoreRejectsOptions(t *testing.T) {
	t.Parallel()

	_, err := backup.ParseArgs([]string{"restore", "/tmp/restore", "extra"})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "restore does not accept options" {
		t.Fatalf("unexpected error: %q", err.Error())
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

func TestRunStrictOverlapFailsOnOverlap(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	content := []byte("profiles:\n  wsl:\n    repository: /repo/wsl\n    include:\n      - /mnt/c/Users/daily\n  windows:\n    repository: C:\\\\repo\\\\windows\n    include:\n      - C:\\\\Users\\\\daily\n")
	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	previousConfig := os.Getenv("BACKUP_CONFIG")
	previousWSL := os.Getenv("WSL_DISTRO_NAME")
	t.Cleanup(func() {
		if previousConfig == "" {
			_ = os.Unsetenv("BACKUP_CONFIG")
		} else {
			_ = os.Setenv("BACKUP_CONFIG", previousConfig)
		}
		if previousWSL == "" {
			_ = os.Unsetenv("WSL_DISTRO_NAME")
		} else {
			_ = os.Setenv("WSL_DISTRO_NAME", previousWSL)
		}
	})

	_ = os.Setenv("BACKUP_CONFIG", configPath)
	_ = os.Setenv("WSL_DISTRO_NAME", "Ubuntu")

	executor := &fakeExecutor{}
	_, err := backup.Run(backup.Command{Name: "run", Cadence: "daily", StrictOverlap: true}, executor)
	if err == nil {
		t.Fatal("expected strict overlap error")
	}
	if !strings.Contains(err.Error(), "platform include overlap detected in strict mode") {
		t.Fatalf("unexpected error: %q", err.Error())
	}
	if len(executor.calls) != 0 {
		t.Fatalf("expected no execution calls, got %#v", executor.calls)
	}
}

func TestRunReportRequiresWSL(t *testing.T) {
	t.Cleanup(func() {
		backup.SetRuntimeDetectorForTests(nil)
	})
	backup.SetRuntimeDetectorForTests(func() backup.Runtime { return backup.RuntimeLinux })

	_, err := backup.Run(backup.Command{Name: "report", Cadence: "daily", Report: "new"}, backup.SystemExecutor{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "must run inside WSL") {
		t.Fatalf("unexpected error: %q", err.Error())
	}
}
