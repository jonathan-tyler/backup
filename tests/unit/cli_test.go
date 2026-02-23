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

func TestParseArgsRunRejectsMultipleOptions(t *testing.T) {
	t.Parallel()

	_, err := backup.ParseArgs([]string{"run", "daily", "extra", "more"})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "run does not accept options" {
		t.Fatalf("unexpected error: %q", err.Error())
	}
}

func TestParseArgsTestCommand(t *testing.T) {
	t.Parallel()

	command, err := backup.ParseArgs([]string{"test"})
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if command.Name != "test" {
		t.Fatalf("unexpected command: %#v", command)
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

func TestParseArgsTestRejectsOptions(t *testing.T) {
	t.Parallel()

	_, err := backup.ParseArgs([]string{"test", "--no-pause-between"})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "test does not accept options" {
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
		backup.SetDevContainerDetectorForTests(nil)
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
	backup.SetDevContainerDetectorForTests(func() bool { return false })

	executor := &fakeExecutor{}
	_, err := backup.Run(backup.Command{Name: "run", Cadence: "daily"}, executor)
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
		backup.SetDevContainerDetectorForTests(nil)
	})
	backup.SetRuntimeDetectorForTests(func() backup.Runtime { return backup.RuntimeLinux })
	backup.SetDevContainerDetectorForTests(func() bool { return false })

	_, err := backup.Run(backup.Command{Name: "report", Cadence: "daily", Report: "new"}, backup.SystemExecutor{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "must run inside WSL") {
		t.Fatalf("unexpected error: %q", err.Error())
	}
}

func TestRunReportRejectsDevContainerSession(t *testing.T) {
	t.Cleanup(func() {
		backup.SetRuntimeDetectorForTests(nil)
		backup.SetDevContainerDetectorForTests(nil)
	})

	backup.SetRuntimeDetectorForTests(func() backup.Runtime { return backup.RuntimeWSL })
	backup.SetDevContainerDetectorForTests(func() bool { return true })

	_, err := backup.Run(backup.Command{Name: "report", Cadence: "daily", Report: "new"}, backup.SystemExecutor{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not from a Dev Container") {
		t.Fatalf("unexpected error: %q", err.Error())
	}
}

func TestRunReportRejectsNativeWindows(t *testing.T) {
	t.Cleanup(func() {
		backup.SetRuntimeDetectorForTests(nil)
		backup.SetDevContainerDetectorForTests(nil)
	})
	backup.SetRuntimeDetectorForTests(func() backup.Runtime { return backup.RuntimeWindows })
	backup.SetDevContainerDetectorForTests(func() bool { return false })

	_, err := backup.Run(backup.Command{Name: "report", Cadence: "daily", Report: "new"}, backup.SystemExecutor{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not from native Windows") {
		t.Fatalf("unexpected error: %q", err.Error())
	}
}

func TestRunTestRequiresWSL(t *testing.T) {
	t.Cleanup(func() {
		backup.SetRuntimeDetectorForTests(nil)
		backup.SetManualTestRunnerForTests(nil)
		backup.SetDevContainerDetectorForTests(nil)
	})
	backup.SetRuntimeDetectorForTests(func() backup.Runtime { return backup.RuntimeLinux })
	backup.SetDevContainerDetectorForTests(func() bool { return false })

	runnerCalled := false
	backup.SetManualTestRunnerForTests(func() error {
		runnerCalled = true
		return nil
	})

	_, err := backup.Run(backup.Command{Name: "test"}, backup.SystemExecutor{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "must run inside WSL") {
		t.Fatalf("unexpected error: %q", err.Error())
	}
	if runnerCalled {
		t.Fatal("manual test runner should not be called outside WSL")
	}
}

func TestRunTestInvokesRunner(t *testing.T) {
	t.Cleanup(func() {
		backup.SetRuntimeDetectorForTests(nil)
		backup.SetManualTestRunnerForTests(nil)
		backup.SetDevContainerDetectorForTests(nil)
	})
	backup.SetRuntimeDetectorForTests(func() backup.Runtime { return backup.RuntimeWSL })
	backup.SetDevContainerDetectorForTests(func() bool { return false })

	runnerCalled := false
	backup.SetManualTestRunnerForTests(func() error {
		runnerCalled = true
		return nil
	})

	output, err := backup.Run(backup.Command{Name: "test"}, backup.SystemExecutor{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !runnerCalled {
		t.Fatal("expected manual test runner call")
	}
	if !strings.Contains(output, "linux then windows") {
		t.Fatalf("unexpected output: %q", output)
	}
}
