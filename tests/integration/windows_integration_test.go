//go:build integration

package integration

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	backup "wsl-backup-cli/src"
)

func TestIntegrationWindowsDailyResticFlow(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only integration test")
	}

	resticPath, err := exec.LookPath("restic.exe")
	if err != nil {
		t.Skip("restic.exe not installed")
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

	_, err = backup.ExecuteResticInvocations([]backup.ResticInvocation{{
		Target:     "windows",
		Executable: resticPath,
		Args:       []string{"-r", repoDir, "init"},
	}}, backup.SystemExecutor{})
	if err != nil {
		t.Fatalf("restic init failed: %v", err)
	}

	_, err = backup.ExecuteResticInvocations([]backup.ResticInvocation{{
		Target:     "windows",
		Executable: resticPath,
		Args:       []string{"-r", repoDir, "backup", dataDir},
	}}, backup.SystemExecutor{})
	if err != nil {
		t.Fatalf("restic backup failed: %v", err)
	}

	if os.Getenv("BACKUP_ITEST_PAUSE") == "1" {
		fmt.Printf("integration pause enabled\nrepo: %s\ndata: %s\npress Enter to continue...\n", repoDir, dataDir)
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
	}
}
