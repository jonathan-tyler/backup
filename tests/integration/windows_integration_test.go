//go:build integration

package integration

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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

	backupBinary := os.Getenv("BACKUP_BINARY")
	if backupBinary != "" {
		configPath := filepath.Join(tempDir, "config.yaml")
		configContent := fmt.Sprintf("profiles:\n  windows:\n    repository: %q\n    include:\n      - %q\n    exclude: []\n    use_fs_snapshot: true\n", repoDir, dataDir)
		if writeErr := os.WriteFile(configPath, []byte(configContent), 0o644); writeErr != nil {
			t.Fatalf("write config file: %v", writeErr)
		}

		previousConfig := os.Getenv("BACKUP_CONFIG")
		_ = os.Setenv("BACKUP_CONFIG", configPath)
		defer func() {
			if previousConfig == "" {
				_ = os.Unsetenv("BACKUP_CONFIG")
				return
			}
			_ = os.Setenv("BACKUP_CONFIG", previousConfig)
		}()

		command := exec.Command(backupBinary, "run", "daily")
		commandOutput, commandErr := command.CombinedOutput()
		if commandErr != nil {
			t.Fatalf("backup binary run daily failed: %v (%s)", commandErr, strings.TrimSpace(string(commandOutput)))
		}
	} else {
		_, err = backup.ExecuteResticInvocations([]backup.ResticInvocation{{
			Target:     "windows",
			Executable: resticPath,
			Args:       []string{"-r", repoDir, "backup", dataDir},
		}}, backup.SystemExecutor{})
		if err != nil {
			t.Fatalf("restic backup failed: %v", err)
		}
	}

	if os.Getenv("BACKUP_ITEST_PAUSE") == "1" {
		fmt.Printf("integration pause enabled\nrepo: %s\ndata: %s\npress Enter to continue...\n", repoDir, dataDir)
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
	}
}

func TestIntegrationWindowsLinksFlow(t *testing.T) {
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
	junctionTarget := filepath.Join(dataDir, "junction-target")
	junctionPath := filepath.Join(dataDir, "junction-dir")
	originalPath := filepath.Join(dataDir, "original.txt")
	symlinkPath := filepath.Join(dataDir, "symlink-to-original.txt")
	hardLinkPath := filepath.Join(dataDir, "hardlink-to-original.txt")

	if err := os.MkdirAll(junctionTarget, 0o755); err != nil {
		t.Fatalf("mkdir junction target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(junctionTarget, "inside.txt"), []byte("junction-target"), 0o644); err != nil {
		t.Fatalf("write junction target file: %v", err)
	}
	if err := os.WriteFile(originalPath, []byte("hello-links"), 0o644); err != nil {
		t.Fatalf("write original file: %v", err)
	}
	if err := os.Symlink(originalPath, symlinkPath); err != nil {
		t.Skipf("symlink creation unavailable (Developer Mode/Admin required): %v", err)
	}
	if err := os.Link(originalPath, hardLinkPath); err != nil {
		t.Skipf("hardlink creation unavailable: %v", err)
	}

	junctionCmd := exec.Command("cmd", "/C", "mklink", "/J", junctionPath, junctionTarget)
	if output, err := junctionCmd.CombinedOutput(); err != nil {
		t.Skipf("junction creation unavailable: %v (%s)", err, strings.TrimSpace(string(output)))
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

	lsResults, err := backup.ExecuteResticInvocations([]backup.ResticInvocation{{
		Target:     "windows",
		Executable: resticPath,
		Args:       []string{"-r", repoDir, "ls", "latest"},
	}}, backup.SystemExecutor{})
	if err != nil {
		t.Fatalf("restic ls latest failed: %v", err)
	}

	lsOutput := lsResults[0].Output
	if !strings.Contains(lsOutput, "symlink-to-original.txt") {
		t.Fatalf("snapshot listing missing symlink: %s", lsOutput)
	}
	if !strings.Contains(lsOutput, "hardlink-to-original.txt") {
		t.Fatalf("snapshot listing missing hardlink: %s", lsOutput)
	}
	if !strings.Contains(lsOutput, "junction-dir") {
		t.Fatalf("snapshot listing missing junction: %s", lsOutput)
	}
}
