//go:build integration && linux

package integration

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	backup "wsl-backup-cli/src"
)

func TestIntegrationWSLDailyResticFlow(t *testing.T) {
	if os.Getenv("WSL_DISTRO_NAME") == "" {
		t.Skip("WSL-only integration test")
	}
	if _, err := exec.LookPath("restic"); err != nil {
		t.Skip("restic not installed")
	}

	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "repo")
	dataDir := filepath.Join(tempDir, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "hello.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write sample file: %v", err)
	}

	_ = os.Setenv("RESTIC_PASSWORD", "integration-test-password")
	defer func() { _ = os.Unsetenv("RESTIC_PASSWORD") }()

	_, err := backup.ExecuteResticInvocations([]backup.ResticInvocation{{
		Target:     "wsl",
		Executable: "restic",
		Args:       []string{"-r", repoDir, "init"},
	}}, backup.SystemExecutor{})
	if err != nil {
		t.Fatalf("restic init failed: %v", err)
	}

	_, err = backup.ExecuteResticInvocations([]backup.ResticInvocation{{
		Target:     "wsl",
		Executable: "restic",
		Args:       []string{"-r", repoDir, "backup", dataDir},
	}}, backup.SystemExecutor{})
	if err != nil {
		t.Fatalf("restic backup failed: %v", err)
	}

	results, err := backup.ExecuteResticInvocations([]backup.ResticInvocation{{
		Target:     "wsl",
		Executable: "restic",
		Args:       []string{"-r", repoDir, "snapshots"},
	}}, backup.SystemExecutor{})
	if err != nil {
		t.Fatalf("restic snapshots failed: %v", err)
	}
	if len(results) != 1 || results[0].Output == "" {
		t.Fatalf("expected snapshot output, got %#v", results)
	}

	if os.Getenv("BACKUP_ITEST_PAUSE") == "1" {
		fmt.Printf("integration pause enabled\nrepo: %s\ndata: %s\npress Enter to continue...\n", repoDir, dataDir)
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
	}
}
